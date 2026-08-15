// Package postgres is the Facilities context's Postgres adapter. It
// implements port.Repository against sqlc-generated queries and translates
// Postgres errors into domain errors — CLAUDE.md rule 5. It only compiles
// after `make generate` has produced internal/gen/facilitiesdb (see
// CLAUDE.md gotchas); that package is gitignored and not committed. Mirrors
// internal/booking/adapter/postgres/repository.go's shape.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
	facilitiesdb "github.com/nhuthuynh/white-label/internal/gen/facilitiesdb"
)

// pgForeignKeyViolation is Postgres' foreign_key_violation SQLSTATE (23503):
// the row being written names a parent row that does not exist. On this
// adapter's writes that means one thing — a facility_id that names no row in
// facilities(id) — because both writes that carry a facility_id
// (AddCourt -> courts.facility_id, AddCameraLink ->
// facility_camera_links.facility_id, both db/migrations/0010_facilities.sql)
// FK to the same parent table. T17.4 (issue #195), mirroring T15.6/#185's
// identical shape for bookings.court_id.
//
// Before this ticket both AddCourt and AddCameraLink were reached only after
// app.Service's own GetFacilityByID guard already confirmed the Facility
// exists (see internal/facilities/app/service.go), so this arm was
// unreachable in the non-racing case — issue #195's own table lists this
// exact pair as "guarded by a read, not a translation". Under a concurrent
// delete of the Facility landing between that guard's read and this INSERT,
// the 23503 fired here, fell through to the wrapped default below, and
// answered codes.Internal (a 500) for what is — at the moment it happens — a
// legitimate, client-visible "no such Facility" condition, the same shape
// #185 fixed for a narrower window on bookings.court_id.
//
// The constraint name is deliberately not part of the match, mirroring
// booking's identical 23503 arm and its own comment on why: Postgres names
// FKs from the table/column by default (courts_facility_id_fkey,
// facility_camera_links_facility_id_fkey — confirmed against
// db/migrations/0010_facilities.sql), but matching on that string would make
// the mapping silently fall back to Internal the day a migration renames
// either constraint.
//
// facility_camera_links also carries a nullable court_id FK
// (0010_facilities.sql) that this same 23503 code could in principle signal,
// but AddCameraLink never populates CourtID from any RPC today (T7.2's
// AddCameraLink only ever appends facility-wide links — see that method's
// own doc comment) — PR #191/T15.6's sibling sweep called this path
// "structurally unreachable" for the same reason. If a future ticket wires a
// per-court camera link, this arm will need to stop assuming every 23503
// here means ErrFacilityNotFound; flagging that here rather than leaving it
// to be rediscovered.
const pgForeignKeyViolation = "23503"

type Repository struct {
	q *facilitiesdb.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: facilitiesdb.New(pool)}
}

func (r *Repository) CreateFacility(ctx context.Context, f domain.Facility) (domain.Facility, error) {
	row, err := r.q.CreateFacility(ctx, facilitiesdb.CreateFacilityParams{
		ID:                    mustUUID(f.ID),
		OwnerID:               mustUUID(f.OwnerID),
		Name:                  f.Name,
		Description:           f.Description,
		Address:               f.Address,
		PhotoUrls:             f.PhotoURLs,
		CameraConsentAttested: f.CameraConsentAttested,
	})
	if err != nil {
		return domain.Facility{}, translateErr(err)
	}
	// A brand new Facility never has camera links yet, so there's no need
	// for the extra ListCameraLinksForFacility round trip GetFacilityByID/
	// ListFacilities make below.
	return fromFacilityFields(row.ID, row.OwnerID, row.Name, row.Description, row.Address, row.PhotoUrls, row.CameraConsentAttested, nil), nil
}

func (r *Repository) GetFacilityByID(ctx context.Context, id string) (domain.Facility, error) {
	row, err := r.q.GetFacilityByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Facility{}, translateErr(err)
	}
	links, err := r.cameraLinksFor(ctx, id)
	if err != nil {
		return domain.Facility{}, err
	}
	courts, err := r.ListCourtsForFacility(ctx, id)
	if err != nil {
		return domain.Facility{}, err
	}
	f := fromFacilityFields(row.ID, row.OwnerID, row.Name, row.Description, row.Address, row.PhotoUrls, row.CameraConsentAttested, links)
	f.Courts = courts
	return f, nil
}

func (r *Repository) ListFacilities(ctx context.Context, nameFilter string) ([]domain.Facility, error) {
	rows, err := r.q.ListFacilities(ctx, nameFilter)
	if err != nil {
		return nil, translateErr(err)
	}
	out := make([]domain.Facility, 0, len(rows))
	for _, row := range rows {
		links, err := r.cameraLinksFor(ctx, row.ID.String())
		if err != nil {
			return nil, err
		}
		out = append(out, fromFacilityFields(row.ID, row.OwnerID, row.Name, row.Description, row.Address, row.PhotoUrls, row.CameraConsentAttested, links))
	}
	return out, nil
}

// AddCourt inserts into the *existing* courts table (0001_init.sql) with
// facility_id set — not a second courts table (HANDOFF.md T7.3 AC 5) — so
// the Court this returns is immediately a valid bookings.court_id for
// Booking's CreateBooking/ListCourtBookings, unmodified.
func (r *Repository) AddCourt(ctx context.Context, c domain.Court) (domain.Court, error) {
	row, err := r.q.AddCourt(ctx, facilitiesdb.AddCourtParams{
		ID:         mustUUID(c.ID),
		Name:       c.Name,
		FacilityID: mustUUID(c.FacilityID),
	})
	if err != nil {
		return domain.Court{}, translateErr(err)
	}
	return domain.Court{
		ID:         row.ID.String(),
		FacilityID: uuidOrEmpty(row.FacilityID),
		Name:       row.Name,
	}, nil
}

func (r *Repository) AddCameraLink(ctx context.Context, facilityID string, link domain.CameraLink) (domain.CameraLink, error) {
	var courtID pgtype.UUID
	if link.CourtID != "" {
		courtID = mustUUID(link.CourtID)
	}
	row, err := r.q.AddCameraLink(ctx, facilitiesdb.AddCameraLinkParams{
		FacilityID: mustUUID(facilityID),
		CourtID:    courtID,
		Url:        link.URL,
	})
	if err != nil {
		return domain.CameraLink{}, translateErr(err)
	}
	return domain.CameraLink{URL: row.Url, CourtID: uuidOrEmpty(row.CourtID)}, nil
}

// AttestCameraConsent persists CameraConsentAttested = true for facilityID
// (T8.4). Callers must have already run domain.Facility.
// AttestCameraConsent's EnsureOwner check before calling this — see
// port.Repository.AttestCameraConsent's doc comment.
func (r *Repository) AttestCameraConsent(ctx context.Context, facilityID string) error {
	if _, err := r.q.AttestCameraConsent(ctx, mustUUID(facilityID)); err != nil {
		return translateErr(err)
	}
	return nil
}

// GetCourtByID is T11.2's Court-shaped read: given a Court, which Facility
// owns it. A NULL facility_id (a seeded, pre-Facilities court) comes back as
// an empty FacilityID via uuidOrEmpty — found, but belonging to no Facility,
// which is a real answer in this schema and not an error.
func (r *Repository) GetCourtByID(ctx context.Context, courtID string) (domain.Court, error) {
	row, err := r.q.GetCourtByID(ctx, mustUUID(courtID))
	if err != nil {
		return domain.Court{}, translateCourtErr(err)
	}
	return domain.Court{
		ID:         row.ID.String(),
		FacilityID: uuidOrEmpty(row.FacilityID),
		Name:       row.Name,
	}, nil
}

// ListCourtsForFacility is T8.2's read path — AddCourt (T7.3) had no way to
// list Courts back. It reads the *existing* courts table (0001_init.sql)
// filtered by facility_id, the same table AddCourt inserts into and Booking
// reads court_id from — no second courts table, no Booking schema change.
func (r *Repository) ListCourtsForFacility(ctx context.Context, facilityID string) ([]domain.Court, error) {
	rows, err := r.q.ListCourtsForFacility(ctx, mustUUID(facilityID))
	if err != nil {
		return nil, translateErr(err)
	}
	out := make([]domain.Court, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Court{
			ID:         row.ID.String(),
			FacilityID: uuidOrEmpty(row.FacilityID),
			Name:       row.Name,
		})
	}
	return out, nil
}

func (r *Repository) cameraLinksFor(ctx context.Context, facilityID string) ([]domain.CameraLink, error) {
	rows, err := r.q.ListCameraLinksForFacility(ctx, mustUUID(facilityID))
	if err != nil {
		return nil, translateErr(err)
	}
	out := make([]domain.CameraLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CameraLink{URL: row.Url, CourtID: uuidOrEmpty(row.CourtID)})
	}
	return out, nil
}

// translateErr maps infrastructure failures onto domain errors — the only
// errors allowed to cross out of this package (CLAUDE.md rule 5).
func translateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		// T17.4 (issue #195): AddCourt/AddCameraLink's facility_id FK firing
		// on a since-deleted Facility. See pgForeignKeyViolation's doc
		// comment above for why this is safe to map generically. Reuses
		// ErrFacilityNotFound rather than a new sentinel — the caller-visible
		// fact in this race case ("no such Facility") is identical to
		// GetFacilityByID's own non-racing miss, and toStatus already maps
		// this sentinel to codes.NotFound.
		return domain.ErrFacilityNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrFacilityNotFound
	}
	return fmt.Errorf("facilities postgres adapter: %w", err)
}

// translateCourtErr is deliberately separate from translateErr, for the same
// reason internal/booking/adapter/postgres keeps translatePricingErr separate
// from translateErr (see that function's comment): translateErr maps
// pgx.ErrNoRows to ErrFacilityNotFound, which is the wrong 404 for a Court
// lookup. A miss on GetCourtByID means the *Court* doesn't exist, and
// Booking's FacilityLookup adapter distinguishes the two.
func translateCourtErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCourtNotFound
	}
	return fmt.Errorf("facilities postgres adapter: %w", err)
}

// fromFacilityFields builds a domain.Facility from the 7 columns every
// facilities query selects, plus its camera links (fetched separately —
// see cameraLinksFor). sqlc generates a distinct Row struct per query
// (CreateFacilityRow, GetFacilityByIDRow, ...) rather than reusing a
// shared table model (CLAUDE.md gotcha); a shared field-level converter
// avoids duplicating this mapping across every query, mirroring
// internal/booking/adapter/postgres/repository.go's fromFields.
func fromFacilityFields(id, ownerID pgtype.UUID, name, description, address string, photoURLs []string, cameraConsentAttested bool, cameraLinks []domain.CameraLink) domain.Facility {
	return domain.Facility{
		ID:                    id.String(),
		OwnerID:               ownerID.String(),
		Name:                  name,
		Description:           description,
		Address:               address,
		PhotoURLs:             photoURLs,
		CameraConsentAttested: cameraConsentAttested,
		CameraLinks:           cameraLinks,
	}
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		// Facility/Court/CameraLink IDs are generated by port.IDGenerator
		// (platform/idgen) as UUIDs; a malformed ID here means an upstream
		// invariant was already violated, which is a programmer error, not
		// a runtime one — mirrors booking's adapter.
		panic(fmt.Sprintf("facilities postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}

// uuidOrEmpty converts a possibly-NULL pgtype.UUID (e.g. courts.facility_id
// for a pre-Facilities seeded court, or facility_camera_links.court_id for
// a facility-wide link) into an empty string, matching domain.Court/
// domain.CameraLink's plain-string-id convention.
func uuidOrEmpty(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}
