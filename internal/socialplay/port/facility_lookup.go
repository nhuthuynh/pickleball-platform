package port

import "context"

// FacilityLookup is Social Play's outbound port for checking whether a
// Game's VenueFacilityID (T8.3) refers to a real Facility in the Facilities
// bounded context. It is expressed entirely in primitive types (a facility
// ID string) so that internal/socialplay/domain and internal/socialplay/app
// never import internal/facilities/domain or internal/facilities/app — the
// context map (agent-operating-handbook.md A1) requires that dependency to
// run through a port, not a direct import, exactly the same reasoning
// CourtReservation's doc comment gives for the Booking context boundary
// (court_reservation.go), just applied to Facilities instead. Only the
// adapter that implements this interface (internal/socialplay/adapter/
// facilities, built in T8.3) is allowed to see Facilities' real types
// (facilitiesdomain.Facility, facilitiesapp.Service) — a
// facilitiesdomain.Facility must never cross this boundary, so this
// interface deliberately has no method that could return one.
type FacilityLookup interface {
	// FacilityExists reports whether facilityID refers to an existing
	// Facility in the Facilities context. It returns nil when the Facility
	// exists. If it does not, implementations return an error satisfying
	// errors.Is(err, domain.ErrFacilityNotFound); callers must not depend
	// on any other error type crossing this boundary for that case.
	// Callers (app.Service.ScheduleGame) only invoke this for a non-empty
	// VenueFacilityID — an empty one is legal and skips the lookup
	// entirely (see domain.Game.VenueFacilityID's doc comment), so
	// implementations are never asked to resolve an empty facilityID.
	FacilityExists(ctx context.Context, facilityID string) error
}
