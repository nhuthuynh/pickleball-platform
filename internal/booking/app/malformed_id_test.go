// Boundary validation for the caller-supplied Court ID on the schedule read.
//
// internal/booking/adapter/postgres.Repository.ListActiveForCourt calls
// mustUUID(courtID), and bookings.court_id is a `uuid` column — so the courtID
// arriving from an HTTP path parameter had to be a UUID or the adapter panicked.
// grpc installs no recover() of its own, so that panic killed the whole server
// process. The guard rejects the malformed ID at the app boundary; the
// interceptor in internal/platform/grpcrecovery is the backstop, not the fix.
package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
)

// courtID mints a deterministic, UUID-shaped Court ID.
//
// Court IDs are opaque references into the Facilities context, but they are
// still real UUIDs in Postgres (bookings.court_id REFERENCES courts (id)).
// Fixtures previously used "court-1", a shape no real Court ever has and the
// adapter cannot store, which is why no test could see this crash.
func courtID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-9000-%012d", n)
}

// TestListCourtBookings_MalformedCourtIDIsEmptyNotAPanic pins the fix. This read
// answers an unknown Court with an empty schedule, so a malformed Court ID must
// answer identically rather than with an error this method never otherwise
// returns.
func TestListCourtBookings_MalformedCourtIDIsEmptyNotAPanic(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"court-1", // the old fixture shape: plausible-looking, still fatal
		"0",
		"'; DROP TABLE bookings;--",
		"../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		// Accepted by github.com/google/uuid's Validate, rejected by
		// pgtype.UUID.Scan — the reason the guard is a canonical-form check
		// rather than a uuid.Validate call.
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	rng := mustTimeRange(t, "2026-08-03T00:00:00Z", "2026-08-04T00:00:00Z")

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, &sequentialIDs{})

			got, err := svc.ListCourtBookings(context.Background(), id, rng)
			if err != nil {
				t.Fatalf("ListCourtBookings(%q) error = %v, want nil", id, err)
			}
			if len(got) != 0 {
				t.Fatalf("ListCourtBookings(%q) = %d bookings, want 0", id, len(got))
			}
		})
	}
}

// TestListCourtBookings_WellFormedCourtIDStillReads is the too-strict guard
// rail: a validator that rejects real Court IDs would silently return an empty
// schedule for every Court, which is quieter and worse than the crash.
func TestListCourtBookings_WellFormedCourtIDStillReads(t *testing.T) {
	t.Parallel()

	svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: courtID(1),
		Source:  "individual",
		Range:   rng,
	}); err != nil {
		t.Fatalf("CreateBooking fixture: %v", err)
	}

	got, err := svc.ListCourtBookings(ctx, courtID(1), rng)
	if err != nil {
		t.Fatalf("ListCourtBookings on a real Court: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d bookings for a real Court, want 1 — the guard is rejecting valid IDs", len(got))
	}
}
