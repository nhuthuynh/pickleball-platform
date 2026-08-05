package facilities_test

import (
	competitionsfacilities "github.com/nhuthuynh/white-label/internal/competitions/adapter/facilities"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// Compile-time proof that *competitionsfacilities.Lookup satisfies
// port.FacilityLookup — see the identical assertion (and the reasoning for
// asserting it rather than waiting for a call site) in
// internal/competitions/adapter/booking's reservation_test.go.
var _ port.FacilityLookup = (*competitionsfacilities.Lookup)(nil)
