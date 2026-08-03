package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// Repository is the Facilities context's persistence boundary. The domain
// and app layers only ever see this interface; adapter/postgres implements
// it against the real database, and tests implement it in-memory. Mirrors
// internal/booking/port.Repository's shape. Implementors must translate
// infrastructure errors into domain errors (CLAUDE.md rule 5) — e.g. a
// Postgres pgx.ErrNoRows on a facility lookup becomes
// domain.ErrFacilityNotFound.
type Repository interface {
	// CreateFacility persists a new Facility.
	CreateFacility(ctx context.Context, f domain.Facility) (domain.Facility, error)

	// GetFacilityByID returns a single Facility (with its CameraLinks), or
	// domain.ErrFacilityNotFound.
	GetFacilityByID(ctx context.Context, id string) (domain.Facility, error)

	// ListFacilities returns Facilities whose Name matches nameFilter
	// (case-insensitive substring match), or all Facilities when
	// nameFilter is empty.
	ListFacilities(ctx context.Context, nameFilter string) ([]domain.Facility, error)

	// AddCourt persists a new Court into the existing courts table (the
	// same table internal/booking/adapter/postgres reads court_id from),
	// with FacilityID set. It must not create a second courts table or
	// duplicate rows — see HANDOFF.md T7.3.
	AddCourt(ctx context.Context, c domain.Court) (domain.Court, error)

	// AddCameraLink persists a new CameraLink for facilityID. Callers must
	// have already checked domain.Facility.AddCameraLink's
	// CameraConsentAttested invariant before calling this — this method
	// only persists, it does not re-check consent.
	AddCameraLink(ctx context.Context, facilityID string, link domain.CameraLink) (domain.CameraLink, error)

	// ListCourtsForFacility returns every Court belonging to facilityID, in
	// creation order (T8.2 — the read path AddCourt (T7.3) never had). An
	// unknown facilityID is not itself an error here — it simply has no
	// courts; GetFacilityByID is what returns ErrFacilityNotFound for an
	// unknown Facility.
	ListCourtsForFacility(ctx context.Context, facilityID string) ([]domain.Court, error)
}
