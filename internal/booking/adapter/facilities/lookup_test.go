package facilities_test

import (
	bookingfacilities "github.com/nhuthuynh/white-label/internal/booking/adapter/facilities"
	"github.com/nhuthuynh/white-label/internal/booking/port"
)

// Compile-time proof that *bookingfacilities.Lookup satisfies
// port.FacilityLookup — asserted here rather than left for the cmd/server
// wiring to discover, mirroring the identical assertion in
// internal/competitions/adapter/facilities' own lookup_test.go.
var _ port.FacilityLookup = (*bookingfacilities.Lookup)(nil)
