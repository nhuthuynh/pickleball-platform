// Package facilities is the one place Booking code is allowed to import
// internal/facilities/* (CLAUDE.md rule 5). It mirrors the role
// internal/socialplay/adapter/facilities and
// internal/competitions/adapter/facilities already play for their own
// contexts — same shape, separate implementation, deliberately not shared
// code across contexts: it's the adapter that implements
// internal/booking/port.FacilityLookup against the real Facilities context's
// app.Service. internal/booking/domain and internal/booking/app never see a
// facilitiesdomain.Facility or facilitiesapp.Service directly.
//
// T11.2 is the first time the Booking context calls into Facilities at all —
// before it, domain.Booking.CourtID was an opaque string Booking never
// resolved anywhere (HANDOFF.md's own note, confirmed by the sprint plan's A8
// dependency check).
package facilities

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// Lookup implements port.FacilityLookup against Facilities' real app.Service,
// translating facilitiesdomain sentinels into Booking's own context-local
// ones (CLAUDE.md rule 5) — callers on the Booking side of this boundary must
// never see a facilitiesdomain error type.
type Lookup struct {
	facilitiesSvc *facilitiesapp.Service
}

func NewLookup(facilitiesSvc *facilitiesapp.Service) *Lookup {
	return &Lookup{facilitiesSvc: facilitiesSvc}
}

// EnsureFacilityOwner resolves the real Facility and delegates the ownership
// decision to facilitiesdomain.Facility.EnsureOwner (T7.7's pattern) rather
// than comparing OwnerID here. That matters beyond tidiness: "who owns a
// Facility" then has exactly one definition, in the context that owns the
// concept, and a future change to it (an owners list, a delegated manager)
// cannot leave a second, stale copy of the rule behind in Booking.
//
// The Facility itself is discarded — port.FacilityLookup deliberately has no
// method that could return one, so nothing on the Booking side can start
// reading another context's aggregate.
func (l *Lookup) EnsureFacilityOwner(ctx context.Context, facilityID, actorUserID string) error {
	f, err := l.facilitiesSvc.GetFacility(ctx, facilityID)
	if err != nil {
		return translate(facilityID, err)
	}
	if err := f.EnsureOwner(actorUserID); err != nil {
		if errors.Is(err, facilitiesdomain.ErrNotFacilityOwner) {
			return domain.ErrNotFacilityOwner
		}
		return translate(facilityID, err)
	}
	return nil
}

// FacilityIDForCourt resolves the Facility that owns courtID, via Facilities'
// own GetCourt — Booking never reads the courts table itself, even though
// bookings.court_id references it.
//
// Both "no such Court" and "a Court belonging to no Facility" (the nullable
// courts.facility_id case, 0010_facilities.sql) come back as
// domain.ErrFacilityNotFound, per the port's documented contract: from
// Booking's side both mean there is no Facility to scope a discount to.
func (l *Lookup) FacilityIDForCourt(ctx context.Context, courtID string) (string, error) {
	c, err := l.facilitiesSvc.GetCourt(ctx, courtID)
	if err != nil {
		if errors.Is(err, facilitiesdomain.ErrCourtNotFound) {
			return "", domain.ErrFacilityNotFound
		}
		return "", translate(courtID, err)
	}
	if c.FacilityID == "" {
		return "", domain.ErrFacilityNotFound
	}
	return c.FacilityID, nil
}

// CourtIDsForFacility resolves the Facility and returns the IDs of its Courts
// (T11.5). Facilities' own GetFacility already carries them on the aggregate,
// so this needs no second round trip and, more importantly, no query against
// the courts table from Booking's side — the court->facility fact stays owned
// by the context that owns Courts.
//
// Only the IDs cross the boundary; the facilitiesdomain.Court values are
// discarded, per port.FacilityLookup's rule that nothing on the Booking side
// may read another context's aggregate.
func (l *Lookup) CourtIDsForFacility(ctx context.Context, facilityID string) ([]string, error) {
	f, err := l.facilitiesSvc.GetFacility(ctx, facilityID)
	if err != nil {
		return nil, translate(facilityID, err)
	}
	out := make([]string, 0, len(f.Courts))
	for _, c := range f.Courts {
		out = append(out, c.ID)
	}
	return out, nil
}

// translate maps the Facilities sentinels Booking has its own names for, and
// strips the original type off anything else — %s, not %w — so a caller can
// never accidentally errors.Is() against a facilitiesdomain sentinel from the
// Booking side (CLAUDE.md rule 5), mirroring
// internal/socialplay/adapter/facilities.Lookup.FacilityExists' identical
// non-%w wrapping for its own "any other error" case.
func translate(id string, err error) error {
	if errors.Is(err, facilitiesdomain.ErrFacilityNotFound) {
		return domain.ErrFacilityNotFound
	}
	return fmt.Errorf("booking facilities adapter: %s: %s", id, err)
}
