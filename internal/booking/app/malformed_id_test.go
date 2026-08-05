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
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
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

// TestListCourtBookings_MalformedCourtIDNeverReachesRepository is the real
// regression proof, and TestListCourtBookings_MalformedCourtIDIsEmptyNotAPanic
// above is not: this project's in-memory inMemoryRepo can't reproduce
// Postgres rejecting a non-UUID against bookings.court_id (a `uuid` column)
// — a Go map/slice scan just finds no match for any unrecognized key,
// malformed or well-formed-but-unknown alike. Reverting the uuidShape guard
// in internal/booking/app/service.go and re-running the test above still
// passes, which means it was never actually proving the guard exists. The
// only way to prove the short-circuit is doing its job is to observe that a
// malformed courtID never reaches the repository call at all — this is a
// deliberate exception to "don't write change-detector tests against a call
// log" (this project's QA dossier §6): here, "never calls the repository"
// IS the security property, not an incidental implementation detail.
func TestListCourtBookings_MalformedCourtIDNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"court-1",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
	rng := mustTimeRange(t, "2026-08-03T00:00:00Z", "2026-08-04T00:00:00Z")

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})

			if _, err := svc.ListCourtBookings(context.Background(), id, rng); err != nil {
				t.Fatalf("ListCourtBookings(%q) error = %v, want nil", id, err)
			}
			if calls := repo.listActiveForCourtCalls.Load(); calls != 0 {
				t.Fatalf("ListCourtBookings(%q): repository.ListActiveForCourt was called for a malformed-shape courtID — against real Postgres this reaches a `uuid` column and panics instead of returning the empty schedule this endpoint promises", id)
			}
		})
	}

	// Sanity check the negative: a well-formed-but-unknown Court SHOULD
	// reach the repository — this test only claims malformed shapes are
	// short-circuited, not that the repository is never called at all.
	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	if _, err := svc.ListCourtBookings(context.Background(), courtID(99), rng); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls := repo.listActiveForCourtCalls.Load(); calls != 1 {
		t.Fatalf("well-formed-but-unknown courtID: repository call count = %d, want 1 — the shape check must not also swallow legitimate lookups", calls)
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

// TestGetQuote_MalformedCourtIDIsNoPricingRuleNotAPanic closes a gap PR #89's
// own review found: GetQuote is a second unauthenticated public read that
// reached internal/booking/adapter/postgres.mustUUID via
// PricingRuleRepository.ListForCourt, undisclosed in that PR's Layer 2
// coverage table. Unlike ListCourtBookings (list-shaped, empty is the
// no-match answer), GetQuote is get-shaped and already returns
// domain.ErrNoPricingRule for a well-formed Court with no configured rules —
// so a malformed Court must produce that same sentinel, not a distinct error
// that would let an unauthenticated caller tell "no such Court" apart from
// "not even a real Court ID".
func TestGetQuote_MalformedCourtIDIsNoPricingRuleNotAPanic(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"court-1",
		"0",
		"'; DROP TABLE pricing_rules;--",
		"../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, &sequentialIDs{})

			_, err := svc.GetQuote(context.Background(), id, rng)
			if !errors.Is(err, domain.ErrNoPricingRule) {
				t.Fatalf("GetQuote(%q) error = %v, want domain.ErrNoPricingRule", id, err)
			}
		})
	}
}

// TestGetQuote_MalformedCourtIDNeverReachesRepository is the real regression
// proof, for the same reason
// TestListCourtBookings_MalformedCourtIDNeverReachesRepository is needed
// above: fakePricingRepo.ListForCourt returns (nil, nil) for any
// unrecognized key, so domain.ResolvePrice's empty-matches branch already
// yields ErrNoPricingRule whether or not the app-layer guard runs —
// reverting the guard still passes
// TestGetQuote_MalformedCourtIDIsNoPricingRuleNotAPanic. The only way to
// prove the shape check is doing its job is to observe that a malformed
// courtID never reaches the repository call at all.
func TestGetQuote_MalformedCourtIDNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"court-1",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			pricingRepo := &fakePricingRepo{}
			svc := app.NewService(newInMemoryRepo(), pricingRepo, &sequentialIDs{})

			if _, err := svc.GetQuote(context.Background(), id, rng); !errors.Is(err, domain.ErrNoPricingRule) {
				t.Fatalf("GetQuote(%q) error = %v, want domain.ErrNoPricingRule", id, err)
			}
			if calls := pricingRepo.listForCourtCalls.Load(); calls != 0 {
				t.Fatalf("GetQuote(%q): repository.ListForCourt was called for a malformed-shape courtID — against real Postgres this reaches a `uuid` column and panics instead of returning the ErrNoPricingRule this endpoint promises", id)
			}
		})
	}

	// Sanity check the negative: a well-formed-but-unknown Court SHOULD
	// still reach the repository.
	pricingRepo := &fakePricingRepo{}
	svc := app.NewService(newInMemoryRepo(), pricingRepo, &sequentialIDs{})
	if _, err := svc.GetQuote(context.Background(), courtID(99), rng); !errors.Is(err, domain.ErrNoPricingRule) {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls := pricingRepo.listForCourtCalls.Load(); calls != 1 {
		t.Fatalf("well-formed-but-unknown courtID: repository call count = %d, want 1 — the shape check must not also swallow legitimate lookups", calls)
	}
}

// TestGetQuote_WellFormedCourtIDStillReads is GetQuote's version of the
// too-strict guard rail above: a validator that rejects real Court IDs would
// silently return "no pricing rule" for every real Court, which is quieter
// and worse than the crash it replaces.
func TestGetQuote_WellFormedCourtIDStillReads(t *testing.T) {
	t.Parallel()

	pricingRepo := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(1): weekdayPricingRules(),
	}}
	svc := app.NewService(newInMemoryRepo(), pricingRepo, &sequentialIDs{})

	quote, err := svc.GetQuote(context.Background(), courtID(1), mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"))
	if err != nil {
		t.Fatalf("GetQuote on a real Court: %v", err)
	}
	if quote.PriceCents != 2000 {
		t.Fatalf("price = %d, want 2000 — the guard is rejecting a valid Court ID", quote.PriceCents)
	}
}
