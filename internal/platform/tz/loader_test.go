package tz_test

import (
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/platform/tz"
)

// There is deliberately no `var _ bookingport.LocationLoader = tz.Loader{}`
// assertion here, unlike the one each context's own adapter test carries for
// its ports: internal/platform must not import a bounded context, not even
// from a test file, and the dependency rule points the other way (a context
// depends on platform). The compile-time proof lives where the assignment
// does — cmd/server passes tz.Loader{} as bookingapp.ServiceOptions.Zones,
// whose type is bookingport.LocationLoader, so a signature drift fails that
// build (and `make test-cmd`).

func TestLoader_LoadLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		zone    string
		wantErr bool
		// wantOffsetSeconds is checked at a fixed instant, so a zone with DST
		// has one right answer rather than two.
		wantOffsetSeconds int
	}{
		{
			// ADR-0004's own example zone, and #356's.
			name: "a real IANA name loads", zone: "Asia/Ho_Chi_Minh",
			wantOffsetSeconds: 7 * 3600,
		},
		{
			name: "a DST zone loads and reports the offset in force at the instant",
			zone: "America/Los_Angeles",
			// 2026-08-03 is inside PDT (UTC-7).
			wantOffsetSeconds: -7 * 3600,
		},
		{
			name: "UTC loads", zone: "UTC", wantOffsetSeconds: 0,
		},
		{
			// The whole reason this type is not a bare time.LoadLocation:
			// the standard library answers UTC here, with no error, which is
			// how an unrecorded facility zone becomes a wrong price instead
			// of a failed request.
			name: "an empty name is an error, not UTC", zone: "", wantErr: true,
		},
		{
			// Resolves to the server process's TZ, which is never a fact
			// about a facility.
			name: "Local is an error", zone: "Local", wantErr: true,
		},
		{
			name: "an unknown name is an error", zone: "Mars/Olympus_Mons", wantErr: true,
		},
	}

	at := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			loc, err := tz.Loader{}.LoadLocation(tt.zone)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("LoadLocation(%q) = %v, want an error", tt.zone, loc)
				}
				if loc != nil {
					t.Errorf("LoadLocation(%q) returned a location alongside its error: %v", tt.zone, loc)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadLocation(%q): %v", tt.zone, err)
			}
			if loc == nil {
				t.Fatalf("LoadLocation(%q) returned a nil location and no error", tt.zone)
			}
			if _, off := at.In(loc).Zone(); off != tt.wantOffsetSeconds {
				t.Errorf("offset at %s = %d, want %d", at, off, tt.wantOffsetSeconds)
			}
		})
	}
}

// TestLoader_EmptyNameDiffersFromTheStandardLibrary states the behaviour
// difference as a fact about the standard library rather than a claim in a
// comment. If a future Go release made time.LoadLocation("") an error, this
// test would say so — and tz.Loader's extra guard could then be reconsidered
// rather than kept on faith.
func TestLoader_EmptyNameDiffersFromTheStandardLibrary(t *testing.T) {
	t.Parallel()

	stdlib, err := time.LoadLocation("")
	if err != nil {
		t.Skipf("time.LoadLocation(\"\") now errors (%v) — tz.Loader's empty-name guard may be redundant", err)
	}
	if stdlib != time.UTC {
		t.Fatalf("time.LoadLocation(\"\") = %v, want UTC — this test's premise has changed", stdlib)
	}
	if _, err := (tz.Loader{}).LoadLocation(""); err == nil {
		t.Fatal("tz.Loader accepted an empty name, matching the stdlib's silent UTC")
	}
}
