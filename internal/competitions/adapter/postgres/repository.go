// Package postgres is the Competitions context's Postgres adapter. It
// implements port.Repository against sqlc-generated queries and translates
// Postgres errors into domain errors (CLAUDE.md rule 5) — no infrastructure
// error type is ever allowed to cross out of this package. It only compiles
// after `make generate` has produced internal/gen/competitionsdb (see
// CLAUDE.md gotchas); that package is gitignored and not committed.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
	competitionsdb "github.com/nhuthuynh/white-label/internal/gen/competitionsdb"
)

// Postgres error codes this adapter cares about.
//
// 23505 is the unique violation on competition_entries_active_player_idx
// (db/migrations/0014_competitions.sql) firing — the DB-level mirror of
// domain.ErrAlreadyEntered per CLAUDE.md rule 4.
//
// P0001 is the competition_entries_capacity_guard trigger's RAISE EXCEPTION
// firing — the DB-level mirror of domain.ErrCompetitionFull, and the thing
// that actually closes the race domain.Enter's in-process pre-check cannot
// (two requests both reading a pre-insert snapshot both believe they fit).
// See the migration's doc comment for the full analysis.
const (
	pgUniqueViolation  = "23505"
	pgCapacityExceeded = "P0001"
)

// Repository implements port.Repository. One type covers both Competitions
// and their entries, matching the single port interface — see
// port.Repository's doc comment for why a CompetitionEntry is never read or
// written outside the scope of its owning Competition.
//
// It holds the pool as well as the Queries because Create spans two tables
// and therefore needs a real transaction (see Create).
type Repository struct {
	pool *pgxpool.Pool
	q    *competitionsdb.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: competitionsdb.New(pool)}
}

// Create persists a Competition and its sessions ATOMICALLY, in one
// transaction.
//
// The transaction is not optional here, and this is the one place this
// adapter's shape genuinely differs from Social Play's (whose Game is a
// single row, so CreateGame needs no transaction at all). A Competition
// spans competitions + N competition_sessions rows, and a Competition
// persisted with a partial session list would be silently wrong in the worst
// way: domain.NewCompetition already validated that the FULL session list
// doesn't double-book its own courts, and app.Service.ScheduleCompetition
// already reserved a real Booking for every (session, court) pair. A
// Competition missing sessions would therefore hold courts it no longer
// claims to use, with no error anywhere to say so.
//
// Sessions are written in slice order, with position carrying that order
// into storage so the Competition round-trips unchanged (see the migration's
// doc comment on why starts_at alone would not be a deterministic ordering).
func (r *Repository) Create(ctx context.Context, c domain.Competition) (domain.Competition, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Competition{}, fmt.Errorf("competitions postgres adapter: beginning transaction: %w", err)
	}
	// Rollback is a no-op once the transaction has committed, so this
	// unconditional deferred call is the standard pgx safety net for every
	// early return below, not a second code path to reason about.
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	row, err := qtx.CreateCompetition(ctx, competitionsdb.CreateCompetitionParams{
		ID:              mustUUID(c.ID),
		HostID:          c.HostID,
		Name:            c.Name,
		VenueFacilityID: nullableUUID(c.VenueFacilityID),
		Capacity:        int32(c.Capacity),
		GuestAllowance:  int32(c.GuestAllowance),
		PaymentMethod:   string(c.PaymentMethod),
		Format:          string(c.Format),
		Status:          string(c.Status),
		// EntryFee is always written explicitly, including a 0 for a free
		// Competition — 0 is a real value, not "unset" (domain.Money).
		EntryFeeCents:    c.EntryFee.AmountCents,
		EntryFeeCurrency: c.EntryFee.CurrencyCode,
		ShareToken:       c.ShareToken,
	})
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	sessions := make([]domain.Session, 0, len(c.Sessions))
	for i, s := range c.Sessions {
		sessionRow, err := qtx.CreateCompetitionSession(ctx, competitionsdb.CreateCompetitionSessionParams{
			CompetitionID: mustUUID(c.ID),
			Position:      int32(i),
			StartsAt:      toTimestamptz(s.Range.Start),
			EndsAt:        toTimestamptz(s.Range.End),
			CourtIds:      mustUUIDs(s.CourtIDs),
		})
		if err != nil {
			return domain.Competition{}, translateErr(err)
		}
		sessions = append(sessions, sessionFromFields(sessionRow.StartsAt, sessionRow.EndsAt, sessionRow.CourtIds))
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Competition{}, fmt.Errorf("competitions postgres adapter: committing competition %s: %w", c.ID, err)
	}

	competition := competitionFromFields(row.ID, row.HostID, row.Name, row.VenueFacilityID, row.Capacity, row.GuestAllowance, row.PaymentMethod, row.Format, row.Status, row.EntryFeeCents, row.EntryFeeCurrency, row.ShareToken)
	competition.Sessions = sessions
	return competition, nil
}

// GetByID returns a Competition with its sessions loaded. Two queries rather
// than one join: a join between competitions and competition_sessions
// returns the Competition's scalar columns duplicated per session row, which
// this adapter would only have to de-duplicate again.
func (r *Repository) GetByID(ctx context.Context, id string) (domain.Competition, error) {
	row, err := r.q.GetCompetitionByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	sessionRows, err := r.q.ListSessionsForCompetition(ctx, mustUUID(id))
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	competition := competitionFromFields(row.ID, row.HostID, row.Name, row.VenueFacilityID, row.Capacity, row.GuestAllowance, row.PaymentMethod, row.Format, row.Status, row.EntryFeeCents, row.EntryFeeCurrency, row.ShareToken)
	for _, s := range sessionRows {
		competition.Sessions = append(competition.Sessions, sessionFromFields(s.StartsAt, s.EndsAt, s.CourtIds))
	}
	return competition, nil
}

// GetByShareToken returns the Competition whose share_token matches, with
// its sessions loaded — the read behind T9.5's shareable registration link.
//
// It mirrors GetByID's two-query shape exactly, with two differences that
// are both deliberate and both security- rather than style-driven:
//
//   - The token is passed to the query AS-IS, never validated, normalised,
//     trimmed, or run through mustUUID (a share token is opaque base64url
//     text, not a UUID — mustUUID would panic on every real token). A
//     malformed token therefore takes the identical path an unknown one
//     takes: no rows, pgx.ErrNoRows, and the same
//     domain.ErrCompetitionNotFound. Rejecting "obviously wrong" tokens
//     earlier or differently would turn this unauthenticated endpoint into
//     an oracle for which tokens have the right shape.
//   - No status filter, so a cancelled Competition's link still resolves
//     (see the query's own comment and port.Repository.GetByShareToken).
//
// The sessions lookup keys off the row's OWN id rather than re-deriving
// anything from the token, so a Competition read by link carries exactly the
// sessions it would carry when read by ID.
func (r *Repository) GetByShareToken(ctx context.Context, shareToken string) (domain.Competition, error) {
	row, err := r.q.GetCompetitionByShareToken(ctx, shareToken)
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	sessionRows, err := r.q.ListSessionsForCompetition(ctx, row.ID)
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	competition := competitionFromFields(row.ID, row.HostID, row.Name, row.VenueFacilityID, row.Capacity, row.GuestAllowance, row.PaymentMethod, row.Format, row.Status, row.EntryFeeCents, row.EntryFeeCurrency, row.ShareToken)
	for _, s := range sessionRows {
		competition.Sessions = append(competition.Sessions, sessionFromFields(s.StartsAt, s.EndsAt, s.CourtIds))
	}
	return competition, nil
}

// UpdateStatus persists a status transition (app.Service.CancelCompetition)
// via its own single-column query, which cannot touch any other column.
//
// The returned Competition carries its sessions, loaded separately, so a
// caller that renders the result of a cancellation sees the same shape
// GetByID would have given it — an UpdateStatus that silently returned a
// Competition with an empty session list would be a quiet trap for any
// caller that used the response rather than re-fetching.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.Status) (domain.Competition, error) {
	row, err := r.q.UpdateCompetitionStatus(ctx, competitionsdb.UpdateCompetitionStatusParams{
		ID:     mustUUID(id),
		Status: string(status),
	})
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	sessionRows, err := r.q.ListSessionsForCompetition(ctx, mustUUID(id))
	if err != nil {
		return domain.Competition{}, translateErr(err)
	}

	competition := competitionFromFields(row.ID, row.HostID, row.Name, row.VenueFacilityID, row.Capacity, row.GuestAllowance, row.PaymentMethod, row.Format, row.Status, row.EntryFeeCents, row.EntryFeeCurrency, row.ShareToken)
	for _, s := range sessionRows {
		competition.Sessions = append(competition.Sessions, sessionFromFields(s.StartsAt, s.EndsAt, s.CourtIds))
	}
	return competition, nil
}

// CreateEntry persists a CompetitionEntry. This is the write the DB-level
// weighted capacity guard fires on: a rejection surfaces here as a P0001,
// which translateEntryErr turns into domain.ErrCompetitionFull — the exact
// same sentinel domain.Enter's own pre-check returns, so callers see one
// error type regardless of which layer caught the conflict.
func (r *Repository) CreateEntry(ctx context.Context, e domain.CompetitionEntry) (domain.CompetitionEntry, error) {
	row, err := r.q.CreateCompetitionEntry(ctx, competitionsdb.CreateCompetitionEntryParams{
		ID:            mustUUID(e.ID),
		CompetitionID: mustUUID(e.CompetitionID),
		PlayerID:      e.PlayerID,
		GuestCount:    int32(e.GuestCount),
		Source:        string(e.Source),
		Status:        string(e.Status),
		PaymentStatus: string(e.PaymentStatus),
	})
	if err != nil {
		return domain.CompetitionEntry{}, translateEntryErr(err)
	}
	return entryFromFields(row.ID, row.CompetitionID, row.PlayerID, row.GuestCount, row.Source, row.Status, row.PaymentStatus), nil
}

// ListActiveEntriesForCompetition returns non-cancelled entries only — the
// read path domain.Enter's weighted capacity rule needs.
func (r *Repository) ListActiveEntriesForCompetition(ctx context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	rows, err := r.q.ListActiveEntriesForCompetition(ctx, mustUUID(competitionID))
	if err != nil {
		return nil, translateEntryErr(err)
	}
	out := make([]domain.CompetitionEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, entryFromFields(row.ID, row.CompetitionID, row.PlayerID, row.GuestCount, row.Source, row.Status, row.PaymentStatus))
	}
	return out, nil
}

// ListEntriesForCompetition returns EVERY entry, cancelled ones included —
// the Host-facing roster read. See port.Repository for why this is a
// separate method from the capacity check's own filtered read.
func (r *Repository) ListEntriesForCompetition(ctx context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	rows, err := r.q.ListEntriesForCompetition(ctx, mustUUID(competitionID))
	if err != nil {
		return nil, translateEntryErr(err)
	}
	out := make([]domain.CompetitionEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, entryFromFields(row.ID, row.CompetitionID, row.PlayerID, row.GuestCount, row.Source, row.Status, row.PaymentStatus))
	}
	return out, nil
}

// GetEntryByID returns a single CompetitionEntry, or
// domain.ErrCompetitionEntryNotFound (T10.6, closes #96).
func (r *Repository) GetEntryByID(ctx context.Context, id string) (domain.CompetitionEntry, error) {
	row, err := r.q.GetCompetitionEntryByID(ctx, mustUUID(id))
	if err != nil {
		return domain.CompetitionEntry{}, translateEntryErr(err)
	}
	return entryFromFields(row.ID, row.CompetitionID, row.PlayerID, row.GuestCount, row.Source, row.Status, row.PaymentStatus), nil
}

// UpdateEntryPaymentStatus persists a PaymentStatus transition
// (app.Service.MarkCompetitionEntryPaymentStatus, T10.6) via its own
// single-column query, which cannot touch any other column.
func (r *Repository) UpdateEntryPaymentStatus(ctx context.Context, id string, status domain.PaymentStatus) (domain.CompetitionEntry, error) {
	row, err := r.q.UpdateCompetitionEntryPaymentStatus(ctx, competitionsdb.UpdateCompetitionEntryPaymentStatusParams{
		ID:            mustUUID(id),
		PaymentStatus: string(status),
	})
	if err != nil {
		return domain.CompetitionEntry{}, translateEntryErr(err)
	}
	return entryFromFields(row.ID, row.CompetitionID, row.PlayerID, row.GuestCount, row.Source, row.Status, row.PaymentStatus), nil
}

// CancelAllActiveForCompetition implements
// port.Repository.CancelAllActiveForCompetition (T16.3, closes the
// mirrored Competitions gap found this ceremony): the bulk cascade
// app.Service.CancelCompetition fires after the Competition's own status
// write persists. One statement
// (db/queries/competitions.sql's CancelAllActiveEntriesForCompetition),
// scoped to status <> 'cancelled', so an entry a player already withdrew
// before the Host cancelled is left untouched. :execrows hands back the
// number of rows actually transitioned, which this method surfaces as an
// int — the count app.Service reports back to its own caller.
func (r *Repository) CancelAllActiveForCompetition(ctx context.Context, competitionID string) (int, error) {
	n, err := r.q.CancelAllActiveEntriesForCompetition(ctx, mustUUID(competitionID))
	if err != nil {
		return 0, translateEntryErr(err)
	}
	return int(n), nil
}

// ListCompetitions is the browse/list read path. SpotsLeft is computed by
// the query itself, using the weighted formula (see
// db/queries/competitions.sql's ListCompetitions and domain.SpotsLeft), so
// this adapter has nothing to compute — only columns to map.
//
// Sessions for every returned Competition are fetched in ONE further query
// (ListSessionsForCompetitions, an `= ANY(...)` over the collected IDs) and
// grouped in memory, rather than one query per Competition: a browse list is
// exactly where an N+1 would hurt most. The sessions cannot be folded into
// the main query, because joining a second one-to-many child would fan out
// the rows and corrupt the spots_left aggregate — see that query's doc
// comment for the full reasoning.
func (r *Repository) ListCompetitions(ctx context.Context, filter port.CompetitionListingFilter) ([]port.CompetitionListing, error) {
	rows, err := r.q.ListCompetitions(ctx, competitionsdb.ListCompetitionsParams{
		VenueFacilityID: nullableUUID(filter.VenueFacilityID),
		StartsAfter:     nullableTimestamptz(filter.StartsAfter),
		StartsBefore:    nullableTimestamptz(filter.StartsBefore),
	})
	if err != nil {
		return nil, translateErr(err)
	}
	if len(rows) == 0 {
		return []port.CompetitionListing{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	sessionRows, err := r.q.ListSessionsForCompetitions(ctx, ids)
	if err != nil {
		return nil, translateErr(err)
	}
	// Rows arrive ordered by (competition_id, position), so appending in
	// scan order preserves each Competition's own session ordering.
	sessionsByCompetition := make(map[string][]domain.Session, len(rows))
	for _, s := range sessionRows {
		key := s.CompetitionID.String()
		sessionsByCompetition[key] = append(sessionsByCompetition[key], sessionFromFields(s.StartsAt, s.EndsAt, s.CourtIds))
	}

	out := make([]port.CompetitionListing, 0, len(rows))
	for _, row := range rows {
		competition := competitionFromFields(row.ID, row.HostID, row.Name, row.VenueFacilityID, row.Capacity, row.GuestAllowance, row.PaymentMethod, row.Format, row.Status, row.EntryFeeCents, row.EntryFeeCurrency, row.ShareToken)
		competition.Sessions = sessionsByCompetition[row.ID.String()]
		out = append(out, port.CompetitionListing{
			Competition: competition,
			SpotsLeft:   int(row.SpotsLeft),
		})
	}
	return out, nil
}

// translateErr maps Competition-level infrastructure failures onto domain
// errors (CLAUDE.md rule 5).
func translateErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCompetitionNotFound
	}
	return fmt.Errorf("competitions postgres adapter (competitions): %w", err)
}

// translateEntryErr maps entry-level infrastructure failures onto domain
// errors.
//
// A 23505 on competition_entries_active_player_idx becomes
// domain.ErrAlreadyEntered, and a P0001 from the
// competition_entries_capacity_guard trigger becomes
// domain.ErrCompetitionFull — in both cases the SAME sentinel domain.Enter's
// own pre-check returns, so no caller can tell (or needs to tell) whether
// the domain or the database caught the conflict. That equivalence is the
// entire point of CLAUDE.md rule 5 here: the DB guard exists to close a race
// the pre-check can't, and it would be a poor trade if closing it introduced
// a second error vocabulary for clients to learn.
//
// pgx.ErrNoRows becomes domain.ErrCompetitionEntryNotFound (T10.6, closes
// #96) — before GetEntryByID/UpdateEntryPaymentStatus this branch was
// unreachable dead code (every prior caller of translateEntryErr was an
// insert or a :many query, neither of which can produce pgx.ErrNoRows), so
// there was no wrong-sentinel bug to inherit; this is that branch's first
// real caller and the entry-scoped sentinel it now returns.
func translateEntryErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return domain.ErrAlreadyEntered
		case pgCapacityExceeded:
			return domain.ErrCompetitionFull
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCompetitionEntryNotFound
	}
	return fmt.Errorf("competitions postgres adapter (entries): %w", err)
}

// competitionFromFields builds a domain.Competition from the 12 scalar
// columns every competitions query selects.
//
// The per-query "fromFields" pattern, not a shared row struct: sqlc emits a
// DISTINCT ...Row type per query (CreateCompetitionRow,
// GetCompetitionByIDRow, ListCompetitionsRow, ...) whenever the column list
// doesn't exactly match SELECT * — which is the case for all of them here,
// since ListCompetitions adds a computed spots_left and none of them select
// created_at. Converting from the shared COLUMNS rather than a shared STRUCT
// is what lets one function serve every query (CLAUDE.md gotcha; the pattern
// originates in internal/booking/adapter/postgres/repository.go's fromFields
// and is mirrored by socialplay's gameFromFields).
//
// Sessions are deliberately NOT a parameter: they come from a different
// query and every caller assigns them explicitly afterwards, so a Competition
// can never silently acquire an empty session list from a caller that forgot.
func competitionFromFields(id pgtype.UUID, hostID, name string, venueFacilityID pgtype.UUID, capacity, guestAllowance int32, paymentMethod, format, status string, entryFeeCents int64, entryFeeCurrency, shareToken string) domain.Competition {
	return domain.Competition{
		ID:              id.String(),
		HostID:          hostID,
		Name:            name,
		VenueFacilityID: fromNullableUUID(venueFacilityID),
		Capacity:        int(capacity),
		GuestAllowance:  int(guestAllowance),
		PaymentMethod:   domain.PaymentMethod(paymentMethod),
		Format:          domain.Format(format),
		Status:          domain.Status(status),
		EntryFee: domain.Money{
			AmountCents:  entryFeeCents,
			CurrencyCode: entryFeeCurrency,
		},
		ShareToken: shareToken,
	}
}

// sessionFromFields builds a domain.Session from the three columns that
// carry one. The storage id and position are intentionally absent: a Session
// is a value type with no domain identity, and position is a storage
// mechanism for preserving order, not a domain field (see the migration).
func sessionFromFields(startsAt, endsAt pgtype.Timestamptz, courtIDs []pgtype.UUID) domain.Session {
	ids := make([]string, 0, len(courtIDs))
	for _, c := range courtIDs {
		ids = append(ids, c.String())
	}
	return domain.Session{
		Range: domain.TimeRange{
			Start: startsAt.Time,
			End:   endsAt.Time,
		},
		CourtIDs: ids,
	}
}

// entryFromFields builds a domain.CompetitionEntry from the 7 columns every
// competition_entries query selects — see competitionFromFields's doc
// comment for why this pattern exists.
func entryFromFields(id, competitionID pgtype.UUID, playerID string, guestCount int32, source, status, paymentStatus string) domain.CompetitionEntry {
	return domain.CompetitionEntry{
		ID:            id.String(),
		CompetitionID: competitionID.String(),
		PlayerID:      playerID,
		GuestCount:    int(guestCount),
		Source:        domain.EntrySource(source),
		PaymentStatus: domain.PaymentStatus(paymentStatus),
		Status:        domain.EntryStatus(status),
	}
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		// Competition/entry IDs are generated by port.IDGenerator
		// (internal/platform/idgen) as UUIDs; a malformed one here means an
		// upstream invariant was already violated, which is a programmer
		// error rather than a runtime one — mirrors the mustUUID in
		// internal/booking and internal/socialplay's adapters.
		panic(fmt.Sprintf("competitions postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}

func mustUUIDs(ss []string) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ss))
	for _, s := range ss {
		out = append(out, mustUUID(s))
	}
	return out
}

// nullableUUID converts an optional domain string ("" meaning unset — a
// Competition with no venue Facility, or an unfiltered ListCompetitions
// call) into the explicitly-invalid (Valid: false) pgtype.UUID that a
// nullable column and a sqlc.narg parameter both expect. Never routes ""
// through mustUUID, which would panic.
func nullableUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	return mustUUID(s)
}

// fromNullableUUID is nullableUUID's inverse: reads a possibly-NULL
// venue_facility_id back into domain.Competition.VenueFacilityID, mapping
// SQL NULL to Go's "" rather than a formatted all-zeroes uuid string.
func fromNullableUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// nullableTimestamptz converts an optional filter bound (a zero time.Time
// meaning "no bound on this side") into the explicitly-invalid
// pgtype.Timestamptz sqlc's narg params expect.
func nullableTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return toTimestamptz(t)
}
