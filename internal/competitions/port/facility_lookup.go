package port

import "context"

// FacilityLookup is Competitions' outbound port for checking whether a
// Competition's VenueFacilityID refers to a real Facility in the Facilities
// bounded context. It is expressed entirely in primitive types (a facility
// ID string) so that internal/competitions/domain and
// internal/competitions/app never import internal/facilities/domain or
// internal/facilities/app — the same reasoning CourtReservation's doc
// comment gives for the Booking context boundary (court_reservation.go),
// applied to Facilities instead. Only the adapter that implements this
// interface (internal/competitions/adapter/facilities) is allowed to see
// Facilities' real types (facilitiesdomain.Facility,
// facilitiesapp.Service).
//
// A facilitiesdomain.Facility must never cross this boundary, so this
// interface deliberately has no method that could return one: Competitions
// only needs a yes/no existence answer, and a port that handed back the
// whole Facility would invite app-layer code to start reading its fields
// and quietly acquire a dependency on another context's model.
type FacilityLookup interface {
	// FacilityExists reports whether facilityID refers to an existing
	// Facility in the Facilities context. It returns nil when the Facility
	// exists. If it does not, implementations return an error satisfying
	// errors.Is(err, domain.ErrFacilityNotFound); callers must not depend
	// on any other error type crossing this boundary for that case.
	//
	// Callers (app.Service.ScheduleCompetition) only invoke this for a
	// non-empty VenueFacilityID — an empty one is legal and skips the
	// lookup entirely (see domain.Competition.VenueFacilityID's doc
	// comment), so implementations are never asked to resolve an empty
	// facilityID.
	FacilityExists(ctx context.Context, facilityID string) error
}
