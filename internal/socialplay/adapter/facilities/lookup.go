// Package facilities is the one place Social Play code is allowed to import
// internal/facilities/* (CLAUDE.md rule 5, mirrors
// internal/socialplay/adapter/booking's role for the Booking context
// boundary — see that package's own doc comment): it's the adapter boundary
// that implements internal/socialplay/port.FacilityLookup against the real
// Facilities context's app.Service, not domain or app code.
// internal/socialplay/domain and internal/socialplay/app never see a
// facilitiesdomain.Facility or facilitiesapp.Service directly.
package facilities

import (
	"context"
	"errors"
	"fmt"

	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// Lookup implements port.FacilityLookup by calling the Facilities context's
// real app.Service.GetFacility, translating facilitiesdomain.
// ErrFacilityNotFound into Social Play's own, context-local domain.
// ErrFacilityNotFound (CLAUDE.md rule 5) — callers on the Social Play side
// of this boundary must never see a facilitiesdomain error type.
type Lookup struct {
	facilitiesSvc *facilitiesapp.Service
}

func NewLookup(facilitiesSvc *facilitiesapp.Service) *Lookup {
	return &Lookup{facilitiesSvc: facilitiesSvc}
}

// FacilityExists reports whether facilityID refers to a real Facility by
// calling Facilities' own GetFacility and discarding the returned Facility —
// Social Play only needs a yes/no existence answer, never any of the
// Facility's own fields (port.FacilityLookup's doc comment: no
// facilitiesdomain.Facility crosses this boundary).
func (l *Lookup) FacilityExists(ctx context.Context, facilityID string) error {
	_, err := l.facilitiesSvc.GetFacility(ctx, facilityID)
	if err != nil {
		if errors.Is(err, facilitiesdomain.ErrFacilityNotFound) {
			return domain.ErrFacilityNotFound
		}
		// Any other error (e.g. a wrapped infra failure) is stripped of its
		// original type before crossing the boundary — %s, not %w — so a
		// caller can never accidentally errors.Is() against a
		// facilitiesdomain sentinel from the Social Play side (CLAUDE.md
		// rule 5), mirroring internal/socialplay/adapter/booking.Reservation.
		// ReserveCourt's identical non-%w wrapping for its own "any other
		// error" case.
		return fmt.Errorf("socialplay facilities adapter: get facility %s: %s", facilityID, err)
	}
	return nil
}
