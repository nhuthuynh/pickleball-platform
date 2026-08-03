package domain

// Court is a single playable surface within a Facility — the same Court
// booking.domain.Booking reserves via its CourtID, and pricing_rules
// (T1) prices via its court_id. Deliberately minimal in this ticket: no
// pricing fields, since pricing already lives entirely in the existing
// pricing_rules table keyed by court_id and is untouched here (see
// docs/process/t7-sprint-plan.md T7.2/T7.3 for the architecture note on how
// this maps onto the existing courts table without touching Booking's
// schema).
type Court struct {
	ID         string
	FacilityID string
	Name       string
}

// NewCourt constructs a Court, validating that FacilityID and Name are both
// non-empty.
//
// id is supplied by the caller (the app layer, via its IDGenerator port),
// the same pattern domain.NewBooking/domain.NewFacility use — Court does
// not generate its own identity.
func NewCourt(id, facilityID, name string) (Court, error) {
	if facilityID == "" {
		return Court{}, newFieldError(ErrEmptyCourtField, "FacilityID")
	}
	if name == "" {
		return Court{}, newFieldError(ErrEmptyCourtField, "Name")
	}
	return Court{ID: id, FacilityID: facilityID, Name: name}, nil
}
