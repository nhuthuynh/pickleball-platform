package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// inMemoryRepo is a minimal port.Repository fake — no persistence, no
// concurrency guarantees (that's the Postgres EXCLUDE constraint's job, see
// T4). It exists purely to let the app-layer orchestration be tested without
// a database.
//
// Every existing test constructs its own inMemoryRepo (never shared across
// t.Parallel() subtests), so the unsynchronized bookings map is safe today.
// listActiveForCourtCalls is atomic regardless — quote_test.go's
// fakePricingRepo had the identical shape and its equivalent plain-int
// counter raced under -race the moment a shared-fixture parallel test
// existed, so this field is atomic from the start rather than waiting for
// the same class of bug to recur here.
type inMemoryRepo struct {
	bookings map[string]domain.Booking

	// listActiveForCourtCalls counts real invocations of ListActiveForCourt,
	// so a test can prove a malformed-shape courtID never reaches the
	// repository at all — the in-memory map here can't reproduce Postgres
	// rejecting a non-UUID against a `uuid` column (it just returns no
	// matches for any unrecognized key), so the app-layer shape-check
	// short-circuit can only be proven by observing the call itself never
	// happens, not by its return value. See
	// TestListCourtBookings_MalformedCourtIDNeverReachesRepository in
	// malformed_id_test.go.
	listActiveForCourtCalls atomic.Int64

	// getByIDCalls counts real invocations of GetByID, so a test can prove
	// a malformed-shape bookingID never reaches the repository at all
	// (T10.7, closing issue #97, CancelBooking's own guard) — same
	// reasoning as listActiveForCourtCalls above, applied to the other
	// caller-supplied id this package's write paths take.
	getByIDCalls atomic.Int64
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{bookings: make(map[string]domain.Booking)}
}

func (r *inMemoryRepo) Create(_ context.Context, b domain.Booking) (domain.Booking, error) {
	r.bookings[b.ID] = b
	return b, nil
}

func (r *inMemoryRepo) ListActiveForCourt(_ context.Context, courtID string, rng domain.TimeRange) ([]domain.Booking, error) {
	r.listActiveForCourtCalls.Add(1)
	var out []domain.Booking
	for _, b := range r.bookings {
		if b.CourtID != courtID || b.Status == domain.StatusCancelled {
			continue
		}
		if b.Range.Overlaps(rng) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *inMemoryRepo) GetByID(_ context.Context, id string) (domain.Booking, error) {
	r.getByIDCalls.Add(1)
	b, ok := r.bookings[id]
	if !ok {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	return b, nil
}

func (r *inMemoryRepo) Update(_ context.Context, b domain.Booking) (domain.Booking, error) {
	if _, ok := r.bookings[b.ID]; !ok {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	r.bookings[b.ID] = b
	return b, nil
}

// sequentialIDs is a deterministic port.IDGenerator fake so test assertions
// don't depend on real UUID generation.
type sequentialIDs struct{ n int }

// NewID mints UUID-shaped ids (T10.7, closing issue #97). It used to return
// "booking-1", "booking-2", ... — a shape the real port.IDGenerator
// (internal/platform/idgen.UUID) never produces and the Postgres adapter's
// mustUUID panics on, which is why no existing test could see
// CancelBooking's malformed-id crash before this ticket added its guard:
// every Booking these tests ever cancelled had a non-UUID id already, the
// same "fixture infidelity" LESSONS.md's T9 entry documents for
// internal/socialplay and internal/facilities' equivalent generators.
func (g *sequentialIDs) NewID() string {
	g.n++
	// The "a000" group (vs. courtID's "9000" in malformed_id_test.go) keeps
	// a Booking's own id visibly distinct from a Court id in test output,
	// even though the two are stored in unrelated keyspaces and a value
	// collision between them would not actually be a bug.
	return fmt.Sprintf("00000000-0000-4000-a000-%012d", g.n)
}

func mustTimeRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatalf("bad fixture time: %v", err)
	}
	e, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatalf("bad fixture time: %v", err)
	}
	r, err := domain.NewTimeRange(s, e)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}

// TestCreateBooking_RejectsCrossSourceOverlap is the app-level proof
// (HANDOFF.md "Done and runnable now") that the no-double-booking invariant
// is enforced across all four Booking sources, not just within one — a
// Competition already holding a court blocks a Game trying to book the same
// court/time, exactly the scenario docs/spec-design-review.md Topic 2 (F1)
// required a test for.
func TestCreateBooking_RejectsCrossSourceOverlap(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	competitionRange := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T11:00:00Z")
	_, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.SourceCompetition, Range: competitionRange, ReferenceID: "competition-1",
	})
	if err != nil {
		t.Fatalf("competition booking should succeed, got %v", err)
	}

	gameRange := mustTimeRange(t, "2026-08-03T10:00:00Z", "2026-08-03T12:00:00Z")
	_, err = svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.SourceGame, Range: gameRange, ReferenceID: "game-1",
	})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("overlapping game booking should be rejected, got %v", err)
	}
}

func TestCreateBooking_AllowsDifferentCourts(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.SourceIndividual, Range: rng,
	}); err != nil {
		t.Fatalf("first booking should succeed, got %v", err)
	}

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-2", Source: domain.SourceIndividual, Range: rng,
	}); err != nil {
		t.Fatalf("booking on a different court at the same time should succeed, got %v", err)
	}
}

func TestCreateBooking_AllowsBackToBack(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	first := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	second := mustTimeRange(t, "2026-08-03T10:00:00Z", "2026-08-03T11:00:00Z")

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.SourceIndividual, Range: first,
	}); err != nil {
		t.Fatalf("first booking should succeed, got %v", err)
	}

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.SourceGame, Range: second, ReferenceID: "game-1",
	}); err != nil {
		t.Fatalf("back-to-back booking should succeed, got %v", err)
	}
}

func TestCreateBooking_InvalidSourceRejectedBeforeTouchingRepo(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: "court-1", Source: domain.Source("bogus"), Range: rng,
	})
	if !errors.Is(err, domain.ErrInvalidSource) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidSource)
	}
	if len(repo.bookings) != 0 {
		t.Fatalf("invalid booking must not be persisted, repo has %d entries", len(repo.bookings))
	}
}
