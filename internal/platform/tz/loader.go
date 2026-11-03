// Package tz provides a shared port.LocationLoader implementation for every
// bounded context, so contexts don't each reach for time.LoadLocation
// directly from their app layer (T68.1, ADR-0004, closes #356). Same role
// internal/platform/idgen plays for port.IDGenerator.
//
// # Where the zone database comes from
//
// time.LoadLocation reads $ZONEINFO, then the host's zoneinfo directories,
// then the copy `import _ "time/tzdata"` embeds in the binary. cmd/server
// carries that import — see the comment on it there, which records the probe
// of this project's actual runtime image (gcr.io/distroless/static-debian12
// does ship /usr/share/zoneinfo, so the embed is a belt-and-braces guarantee
// rather than the only source). The import deliberately lives in main rather
// than here: Go's own guidance is that a library must not decide to add 450 KB
// to every program that links it.
package tz

import (
	"fmt"
	"time"
)

// Loader implements every context's port.LocationLoader using the standard
// library's zone database.
type Loader struct{}

// LoadLocation resolves an IANA identifier to a *time.Location.
//
// It is time.LoadLocation with one deliberate difference: **an empty name is
// an error here, where the standard library returns UTC.** That single
// behaviour is the reason this type exists rather than a function value
// wrapping time.LoadLocation directly. A Facility whose timezone was never
// recorded reaches this layer as "", and `time.LoadLocation("")` would hand
// back UTC with no error — silently pricing an 18:00 Ho Chi Minh City slot
// against the wrong band, which is precisely the defect #356 reported. The
// app layer rejects "" before calling (app.Service.courtLocation), so this is
// the second of two guards on the same mistake; it is duplicated on purpose,
// because the cost of the two disagreeing is a wrong price nobody notices.
//
// "Local" is likewise refused: it resolves to whatever TZ the *server
// process* happens to run with, which is never a fact about a facility.
//
// # Why it returns a nil Location on every error, explicitly
//
// A second reason, found by the review of PR #370 and measured rather than
// reasoned about. On an UNKNOWN name the standard library returns
// `(nil, error)` — and a nil *time.Location is not harmless:
//
//	LoadLocation("Not/AZone"): err=unknown time zone Not/AZone
//	                           loc==nil: true | loc==time.UTC: false
//	                           fmt.Sprint(loc)="UTC"
//	time.Time.In(nil) PANICS: time: missing Location in call to Time.In
//
// So a caller that logged the value would see the word **UTC** while holding
// something that panics the moment it is used. Returning nil here is
// therefore not a formality: it means the one value that can reach the domain
// is either usable or nil, and domain.ResolvePrice's nil check turns the nil
// into ErrFacilityTimezoneUnset instead of a panicked request. The empty-name
// case above is the opposite shape and the reason this type exists — a real,
// non-nil time.UTC and no error at all.
func (Loader) LoadLocation(name string) (*time.Location, error) {
	switch name {
	case "":
		return nil, fmt.Errorf("tz: empty timezone name is not a zone (it would silently resolve to UTC)")
	case "Local":
		return nil, fmt.Errorf("tz: %q is the server's own zone, not a facility's", name)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("tz: load %q: %w", name, err)
	}
	if loc == nil {
		// Unreachable through the standard library, which never returns
		// (nil, nil) — asserted because every caller dereferences the result
		// and a nil here would panic a quote instead of failing it.
		return nil, fmt.Errorf("tz: load %q: nil location", name)
	}
	return loc, nil
}
