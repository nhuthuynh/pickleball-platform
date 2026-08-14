package port

import "context"

// FacilityLookup is Booking's outbound port into the Facilities bounded
// context — the first one Booking has ever had (T11.2; before this ticket
// domain.Booking.CourtID was an opaque string Booking never resolved
// anywhere). It mirrors internal/socialplay/port.FacilityLookup and
// internal/competitions/port.FacilityLookup's shape exactly: a
// primitive-typed interface, implemented once per context rather than
// shared, so internal/booking/domain and internal/booking/app never import
// internal/facilities/domain or internal/facilities/app. Only the adapter
// that implements it (internal/booking/adapter/facilities) sees Facilities'
// real types.
//
// Like those two, this interface deliberately has no method that could
// return a facilitiesdomain.Facility: a port that handed back the whole
// aggregate would invite app-layer code to start reading its fields and
// quietly acquire a dependency on another context's model. Booking needs two
// answers, and asks for exactly those two.
type FacilityLookup interface {
	// EnsureFacilityOwner returns nil when actorUserID owns facilityID.
	//
	// The ownership rule itself is not reimplemented here — the adapter
	// resolves the real Facility and calls facilitiesdomain.Facility.
	// EnsureOwner (T7.7's pattern), so there is one definition of "owner"
	// in the system, living in the context that owns the concept.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrNotFacilityOwner) when the actor does not
	// own the Facility, and errors.Is(err, domain.ErrFacilityNotFound) when
	// facilityID does not resolve at all. Callers must not depend on any
	// other error type crossing this boundary for those two cases.
	EnsureFacilityOwner(ctx context.Context, facilityID, actorUserID string) error

	// FacilityIDForCourt returns the ID of the Facility that owns courtID.
	// GetQuote needs it because DiscountRules are facility-scoped (A9's
	// narrowing) while the quote path keys off CourtID only.
	//
	// Two distinct situations are deliberately collapsed into
	// domain.ErrFacilityNotFound: courtID naming no Court at all, and a
	// Court that exists but belongs to no Facility (courts.facility_id is
	// nullable — the seeded, pre-Facilities courts of 0001_init.sql still
	// have NULL there, per 0010_facilities.sql's own note). From Booking's
	// side both mean the same actionable thing — there is no Facility to
	// scope a discount to — and splitting them would create a second
	// sentinel no caller could act on differently.
	FacilityIDForCourt(ctx context.Context, courtID string) (string, error)

	// CourtIDsForFacility returns the IDs of every Court belonging to
	// facilityID, or domain.ErrFacilityNotFound when facilityID does not
	// resolve. A Facility with no Courts yields an empty slice and no error.
	//
	// Added in T11.5, on this same port rather than a second one (sprint plan
	// A10 is explicit that T11.5 must reuse T11.2's FacilityLookup, not
	// re-derive it). It exists because a RecurringHireTemplate references a
	// Court while its owner-facing read is facility-scoped: resolving the
	// facility's courts here keeps that resolution inside the context that
	// owns Courts, instead of Booking joining the courts table itself or
	// denormalising a facility_id onto recurring_hire_templates.
	//
	// Like the two methods above, it returns IDs rather than
	// facilitiesdomain.Court values — nothing on the Booking side may start
	// reading another context's aggregate.
	CourtIDsForFacility(ctx context.Context, facilityID string) ([]string, error)
}
