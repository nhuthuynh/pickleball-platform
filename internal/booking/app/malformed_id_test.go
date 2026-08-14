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

			svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

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
			svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

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
	svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})
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

	svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})
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

			svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

			_, err := svc.GetQuote(context.Background(), app.GetQuoteInput{CourtID: id, Source: domain.SourceIndividual, Range: rng})
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
			svc := app.NewService(newInMemoryRepo(), pricingRepo, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

			if _, err := svc.GetQuote(context.Background(), app.GetQuoteInput{CourtID: id, Source: domain.SourceIndividual, Range: rng}); !errors.Is(err, domain.ErrNoPricingRule) {
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
	svc := app.NewService(newInMemoryRepo(), pricingRepo, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})
	if _, err := svc.GetQuote(context.Background(), app.GetQuoteInput{CourtID: courtID(99), Source: domain.SourceIndividual, Range: rng}); !errors.Is(err, domain.ErrNoPricingRule) {
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
	svc := app.NewService(newInMemoryRepo(), pricingRepo, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

	quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{CourtID: courtID(1), Source: domain.SourceIndividual, Range: mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")})
	if err != nil {
		t.Fatalf("GetQuote on a real Court: %v", err)
	}
	if quote.PriceCents != 2000 {
		t.Fatalf("price = %d, want 2000 — the guard is rejecting a valid Court ID", quote.PriceCents)
	}
}

// --- T10.7 (closing issue #97): CancelBooking's own malformed-id guard ----
//
// CancelBooking was found by this ticket's required inspection sweep
// (grepping cmd/server/main.go's route registrations for any other public
// write handler taking a caller-supplied id — issue #97's own instruction
// not to assume its named handler list was exhaustive): it calls
// Repository.GetByID(bookingID) first, exactly the same shape
// ListCourtBookings/GetQuote's already-guarded reads have, and already
// returns the bare domain.ErrBookingNotFound for an unknown-but-well-formed
// id (TestCancelBooking_UnknownBookingReturnsNotFound in
// cancel_booking_test.go pins that today) — but, unlike those two reads, it
// had no uuidShape guard at all, so a malformed bookingID reached
// mustUUID unguarded.

func TestCancelBooking_MalformedBookingIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"booking-1",
		"0",
		"'; DROP TABLE bookings;--",
		"../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

			_, err := svc.CancelBooking(context.Background(), id)
			if !errors.Is(err, domain.ErrBookingNotFound) {
				t.Fatalf("CancelBooking(%q) error = %v, want %v", id, err, domain.ErrBookingNotFound)
			}
			if calls := repo.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed bookingID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestCancelBooking_WellFormedUnknownBookingIDStillReachesRepository is the
// too-strict guard rail: a well-formed but unknown bookingID must still
// reach the repository and get its own ErrBookingNotFound, or every real
// cancel-by-id would silently 404.
func TestCancelBooking_WellFormedUnknownBookingIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	_, err := svc.CancelBooking(context.Background(), unknown)
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("CancelBooking(%q) error = %v, want %v", unknown, err, domain.ErrBookingNotFound)
	}
	if calls := repo.getByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown bookingID did not reach the repository (%d calls)", calls)
	}
}

// --- T10.7 follow-up (PR #101 review finding): CreateBooking's own
// malformed-CourtID guard ---------------------------------------------
//
// CreateBooking calls Repository.ListActiveForCourt(in.CourtID, ...) BEFORE
// Repository.Create — the same mustUUID-backed adapter method
// ListCourtBookings' own already-guarded read calls. A malformed CourtID
// reached it unguarded (CreateBooking had no uuidShape check at all, unlike
// every other handler this ticket touches), panicking there rather than
// ever reaching Create's FK-violation path. This is a genuinely different
// case from this file's package-level "CreateBooking's CourtID" scope note
// in the PR description, which is about a *well-formed-but-unknown* court's
// untranslated FK-violation Internal — that path is real and still not
// this ticket's to fix (see domain.ErrInvalidCourtReference's doc comment),
// but a malformed *shape* never even reaches it: it panics one step
// earlier, inside ListActiveForCourt, before Create is ever called.

func TestCreateBooking_MalformedCourtIDNeverReachesRepository(t *testing.T) {
	t.Parallel()

	// Non-empty malformed shapes: rejected by the new uuidShape guard with
	// domain.ErrInvalidCourtReference. Empty is handled separately below —
	// domain.NewBooking's own, pre-existing ErrEmptyCourtID check runs
	// first and still fires for that specific case, unchanged by this
	// guard (see the guard's own comment in service.go).
	malformed := []string{
		"not-a-uuid",
		"court-1", // the old fixture shape: plausible-looking, still fatal
		"0",
		"'; DROP TABLE bookings;--",
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

			repo := newInMemoryRepo()
			svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

			_, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
				CourtID: id, Source: domain.SourceIndividual, Range: rng,
			})
			if !errors.Is(err, domain.ErrInvalidCourtReference) {
				t.Fatalf("CreateBooking(CourtID=%q) error = %v, want %v", id, err, domain.ErrInvalidCourtReference)
			}
			if calls := repo.listActiveForCourtCalls.Load(); calls != 0 {
				t.Errorf("malformed CourtID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}

	t.Run("empty (pre-existing ErrEmptyCourtID, unchanged)", func(t *testing.T) {
		t.Parallel()

		repo := newInMemoryRepo()
		svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})

		_, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
			CourtID: "", Source: domain.SourceIndividual, Range: rng,
		})
		if !errors.Is(err, domain.ErrEmptyCourtID) {
			t.Fatalf("CreateBooking(CourtID=\"\") error = %v, want %v", err, domain.ErrEmptyCourtID)
		}
		if calls := repo.listActiveForCourtCalls.Load(); calls != 0 {
			t.Errorf("empty CourtID reached the repository (%d calls); domain.NewBooking must reject it first", calls)
		}
	})
}

// TestCreateBooking_WellFormedCourtIDStillReachesRepository is the
// too-strict guard rail: a well-formed CourtID (known or not — this
// in-memory fake has no FK check, unlike the real Postgres adapter) must
// still reach ListActiveForCourt, or every real booking attempt would be
// rejected.
func TestCreateBooking_WellFormedCourtIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, newFakeDiscountRepo(), &fakeFacilityLookup{}, &sequentialIDs{})
	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	if _, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID: courtID(1), Source: domain.SourceIndividual, Range: rng,
	}); err != nil {
		t.Fatalf("CreateBooking on a well-formed CourtID: %v", err)
	}
	if calls := repo.listActiveForCourtCalls.Load(); calls != 1 {
		t.Fatalf("well-formed CourtID did not reach the repository (%d calls) — the guard is too strict", calls)
	}
}
