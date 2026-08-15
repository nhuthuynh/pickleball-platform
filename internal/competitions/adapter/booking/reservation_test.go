// Package booking_test covers internal/competitions/adapter/booking, the one
// place Competitions is allowed to call into the Booking context.
//
// Before T13.1 this file held a single compile-time assertion and nothing else
// — 25 lines that could not fail for any behavioural reason (T12 retro
// finding 2).
//
// That assertion's original comment argued the behaviour should NOT be
// unit-tested, on the grounds that "a test against a stubbed Booking service
// would prove only that the stub returns what it was told to". The premise is
// correct and the conclusion no longer follows: these tests do not stub
// bookingapp.Service, they drive the REAL one over in-memory repository fakes,
// so the court conflict below is produced by Booking's real
// domain.EnsureNoConflict against a real overlapping Booking. The pattern is
// internal/payments/adapter/socialplay/registration_updater_test.go's. No
// Docker, no testcontainers.
//
// What these tests still do NOT prove, and what the original comment was right
// about: that a conflict arising from the Postgres EXCLUDE constraint under
// real concurrency surfaces the same way. That is the authoritative guard
// (CLAUDE.md rule 4) and remains an integration concern. The
// repo-returned-ErrCourtDoubleBooked case below is the closest Docker-free
// approximation and is labelled as such.
package booking_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	competitionsbooking "github.com/nhuthuynh/white-label/internal/competitions/adapter/booking"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// Compile-time proof that *competitionsbooking.Reservation satisfies
// port.CourtReservation — the whole point of this package. Worth asserting
// explicitly rather than relying on a call site to catch a mismatch, because
// T9.3 had no call site yet: cmd/server didn't wire Competitions until T9.4,
// so without this line a signature drift between the port and the adapter
// would compile cleanly and stay hidden for a whole ticket. Mirrors
// internal/payments/adapter/socialplay's identical assertion.
//
// Kept deliberately (T13.1 instruction 6): cheap and correct, and simply not a
// test. Everything below it is.
var _ port.CourtReservation = (*competitionsbooking.Reservation)(nil)

// Fixture ids are uuid-shaped. This seam carries no actor at all —
// ReserveCourt/ReleaseCourt take court, booking and reference ids only — so
// T13.1's "non-uuid subject where the seam carries one" condition does not
// apply here, and a subject-shaped fixture would exercise a path production
// never takes: bookingapp.Service guards CourtID and bookingID on uuidShape.
// Stated as a checked negative rather than left to inference.
const (
	fixtureCourtID       = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	fixtureCompetitionID = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
	mintedBookingID      = "3f2504e0-4f89-11d3-9a0c-0305e82c3401"
	seededBookingID      = "3f2504e0-4f89-11d3-9a0c-0305e82c3402"
	unknownBookingID     = "3f2504e0-4f89-11d3-9a0c-0305e82c34ff"
)

// errRepoUnavailable stands in for an infrastructure failure with no
// context-local meaning — the "any other error" arm of the adapter.
var errRepoUnavailable = errors.New("booking postgres: connection refused")

func slot(t *testing.T, startHour, endHour int) bookingdomain.TimeRange {
	t.Helper()

	r, err := bookingdomain.NewTimeRange(hour(startHour), hour(endHour))
	if err != nil {
		t.Fatalf("building fixture range: %v", err)
	}
	return r
}

// slotFor is slot() without a *testing.T, for use in table literals. The
// ranges here are all valid by construction, so NewTimeRange's error is
// impossible; the struct is built directly rather than dropping an error.
func slotFor(startHour, endHour int) bookingdomain.TimeRange {
	return bookingdomain.TimeRange{Start: hour(startHour), End: hour(endHour)}
}

func hour(h int) time.Time {
	return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC).Add(time.Duration(h) * time.Hour)
}

// inMemoryRepo is a minimal booking port.Repository fake. It really stores
// bookings, so ListActiveForCourt feeds Booking's real
// domain.EnsureNoConflict and a seeded overlapping Booking produces a genuine
// ErrCourtDoubleBooked rather than a canned one.
//
// createFailsWith / getFailsWith, when set, drive an upstream error at the
// repository layer.
type inMemoryRepo struct {
	bookings        map[string]bookingdomain.Booking
	createFailsWith error
	getFailsWith    error
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{bookings: make(map[string]bookingdomain.Booking)}
}

func (r *inMemoryRepo) Create(_ context.Context, b bookingdomain.Booking) (bookingdomain.Booking, error) {
	if r.createFailsWith != nil {
		return bookingdomain.Booking{}, r.createFailsWith
	}
	r.bookings[b.ID] = b
	return b, nil
}

func (r *inMemoryRepo) ListActiveForCourt(_ context.Context, courtID string, rng bookingdomain.TimeRange) ([]bookingdomain.Booking, error) {
	var out []bookingdomain.Booking
	for _, b := range r.bookings {
		if b.CourtID != courtID || b.Status == bookingdomain.StatusCancelled {
			continue
		}
		if b.Range.Overlaps(rng) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *inMemoryRepo) GetByID(_ context.Context, id string) (bookingdomain.Booking, error) {
	if r.getFailsWith != nil {
		return bookingdomain.Booking{}, r.getFailsWith
	}
	b, ok := r.bookings[id]
	if !ok {
		return bookingdomain.Booking{}, bookingdomain.ErrBookingNotFound
	}
	return b, nil
}

func (r *inMemoryRepo) Update(_ context.Context, b bookingdomain.Booking) (bookingdomain.Booking, error) {
	if _, ok := r.bookings[b.ID]; !ok {
		return bookingdomain.Booking{}, bookingdomain.ErrBookingNotFound
	}
	r.bookings[b.ID] = b
	return b, nil
}

// The remaining bookingapp.NewService dependencies are stubs: ReserveCourt
// and ReleaseCourt reach only CreateBooking and CancelBooking, neither of
// which touches pricing, discounts, recurring hires, Facilities or Identity.
// They exist to satisfy the seven-parameter constructor.

type stubPricing struct{}

func (stubPricing) ListForCourt(context.Context, string) ([]bookingdomain.PricingRule, error) {
	return nil, nil
}

type stubDiscounts struct{}

func (stubDiscounts) Create(_ context.Context, r bookingdomain.DiscountRule) (bookingdomain.DiscountRule, error) {
	return r, nil
}
func (stubDiscounts) ListForFacility(context.Context, string) ([]bookingdomain.DiscountRule, error) {
	return nil, nil
}

type stubRecurring struct{}

func (stubRecurring) Create(_ context.Context, tmpl bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	return tmpl, nil
}
func (stubRecurring) GetByID(context.Context, string) (bookingdomain.RecurringHireTemplate, error) {
	return bookingdomain.RecurringHireTemplate{}, bookingdomain.ErrRecurringHireTemplateNotFound
}
func (stubRecurring) UpdateStatus(_ context.Context, tmpl bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	return tmpl, nil
}
func (stubRecurring) ListForCourts(context.Context, []string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}
func (stubRecurring) ListForRequester(context.Context, string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}

type stubFacilities struct{}

func (stubFacilities) EnsureFacilityOwner(context.Context, string, string) error { return nil }
func (stubFacilities) FacilityIDForCourt(context.Context, string) (string, error) {
	return "", bookingdomain.ErrFacilityNotFound
}
func (stubFacilities) CourtIDsForFacility(context.Context, string) ([]string, error) {
	return nil, nil
}

type stubIdentity struct{}

func (stubIdentity) EnsureClubRole(context.Context, string) error { return nil }

// UserIDBySubject joined booking's port.IdentityLookup in T13.2 (ADR-0014).
// Competitions reserves courts through app.Service.CreateBooking, which takes
// no actor at all, so no test here reaches it — hence the fail-closed
// sentinel rather than a plausible-looking id, which would let a future
// wiring mistake pass unnoticed.
func (stubIdentity) UserIDBySubject(context.Context, string) (string, error) {
	return "", bookingdomain.ErrUserNotFound
}

// fixedIDs mints the uuid-shaped id every Booking created here receives.
type fixedIDs struct{}

func (fixedIDs) NewID() string { return mintedBookingID }

// newReservation builds the adapter over a REAL bookingapp.Service and returns
// the repo fake alongside it, so a test can seed conflicting bookings, fail the
// upstream, and inspect what was actually written.
func newReservation(t *testing.T) (*competitionsbooking.Reservation, *inMemoryRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	svc := bookingapp.NewService(repo, stubPricing{}, stubDiscounts{}, stubRecurring{}, stubFacilities{}, stubIdentity{}, fixedIDs{})
	return competitionsbooking.NewReservation(svc), repo
}

// seedBooking writes a confirmed Booking straight through the repo fake.
func seedBooking(repo *inMemoryRepo, id string, rng bookingdomain.TimeRange) {
	repo.bookings[id] = bookingdomain.Booking{
		ID:      id,
		CourtID: fixtureCourtID,
		Source:  bookingdomain.SourceIndividual,
		Status:  bookingdomain.StatusConfirmed,
		Range:   rng,
	}
}

// TestReserveCourt_WritesACompetitionSourcedBooking proves the happy path
// reaches the real Booking service and what it wrote is observable — not
// merely that no error came back.
//
// The Source assertion is the point, not decoration: port.CourtReservation
// fixes the source here rather than accepting it as a parameter precisely so a
// caller cannot write a Booking that misreports what is holding the court.
// Nothing else in the repository tests that.
func TestReserveCourt_WritesACompetitionSourcedBooking(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	id, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureCompetitionID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != mintedBookingID {
		t.Fatalf("ReserveCourt returned id %q, want the server-minted %q", id, mintedBookingID)
	}

	stored, ok := repo.bookings[id]
	if !ok {
		t.Fatalf("no Booking was persisted for id %q", id)
	}
	if stored.Source != bookingdomain.SourceCompetition {
		t.Fatalf("stored Source = %q, want %q", stored.Source, bookingdomain.SourceCompetition)
	}
	if stored.ReferenceID != fixtureCompetitionID {
		t.Fatalf("stored ReferenceID = %q, want %q", stored.ReferenceID, fixtureCompetitionID)
	}
	if stored.CourtID != fixtureCourtID {
		t.Fatalf("stored CourtID = %q, want %q", stored.CourtID, fixtureCourtID)
	}
	if !stored.Range.Start.Equal(rng.Start) || !stored.Range.End.Equal(rng.End) {
		t.Fatalf("stored Range = %v, want %v", stored.Range, rng)
	}
}

// TestReserveCourt_ConflictIsTranslated is the central case. The conflict is
// real: a genuine overlapping Booking sits in the repository and Booking's own
// domain.EnsureNoConflict rejects the candidate. Both sides of the translation
// are asserted, per T13.1 instruction 3.
//
// The conflicting fixture is a SourceIndividual Booking on purpose: Booking's
// invariant is source-agnostic (D3b/F1), so a Competition must be blocked by
// an individual booking just as by another competition.
func TestReserveCourt_ConflictIsTranslated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing bookingdomain.TimeRange
		want     bookingdomain.TimeRange
		conflict bool
	}{
		{name: "exact same slot", existing: slotFor(10, 11), want: slotFor(10, 11), conflict: true},
		{name: "partial overlap at the start", existing: slotFor(10, 12), want: slotFor(11, 13), conflict: true},
		{name: "adjacent slot does not conflict", existing: slotFor(9, 10), want: slotFor(10, 11), conflict: false},
		{name: "disjoint slot does not conflict", existing: slotFor(8, 9), want: slotFor(14, 15), conflict: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, repo := newReservation(t)
			seedBooking(repo, seededBookingID, tt.existing)

			_, err := res.ReserveCourt(context.Background(), fixtureCourtID, tt.want.Start, tt.want.End, fixtureCompetitionID)

			if !tt.conflict {
				if err != nil {
					t.Fatalf("ReserveCourt = %v, want nil (ranges are half-open, [a,b) and [b,c) do not overlap)", err)
				}
				return
			}
			if !errors.Is(err, domain.ErrCourtUnavailable) {
				t.Fatalf("ReserveCourt = %v, want competitions domain.ErrCourtUnavailable", err)
			}
			if errors.Is(err, bookingdomain.ErrCourtDoubleBooked) {
				t.Fatalf("ReserveCourt leaked the bookingdomain sentinel across the boundary: %v", err)
			}
		})
	}
}

// TestReserveCourt_ConflictFromTheRepositoryIsTranslated covers the same
// translation arriving from the repository rather than from the domain
// pre-check — the shape a Postgres EXCLUDE-constraint violation takes once
// adapter/postgres has translated 23P01 (CLAUDE.md rule 5). It is the closest
// Docker-free approximation of the authoritative guard; the real concurrent
// case stays an integration test.
func TestReserveCourt_ConflictFromTheRepositoryIsTranslated(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	repo.createFailsWith = bookingdomain.ErrCourtDoubleBooked
	rng := slot(t, 10, 11)

	_, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureCompetitionID)

	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("ReserveCourt = %v, want competitions domain.ErrCourtUnavailable", err)
	}
	if errors.Is(err, bookingdomain.ErrCourtDoubleBooked) {
		t.Fatalf("ReserveCourt leaked the bookingdomain sentinel across the boundary: %v", err)
	}
}

// TestReserveCourt_NeverLeaksBookingSentinels holds the other half of
// CLAUDE.md rule 5: everything that is NOT the one translated sentinel is
// wrapped with %s, not %w, so a Competitions caller cannot errors.Is() across
// the context boundary at all.
func TestReserveCourt_NeverLeaksBookingSentinels(t *testing.T) {
	t.Parallel()

	upstreams := []struct {
		name string
		err  error
	}{
		{name: "untranslated booking sentinel", err: bookingdomain.ErrBookingNotFound},
		{name: "infrastructure failure", err: errRepoUnavailable},
	}

	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			t.Parallel()

			res, repo := newReservation(t)
			repo.createFailsWith = up.err
			rng := slot(t, 10, 11)

			_, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureCompetitionID)
			if err == nil {
				t.Fatal("want an error when the upstream fails")
			}
			if errors.Is(err, up.err) {
				t.Fatalf("leaked the upstream error across the boundary: %v", err)
			}
			if errors.Is(err, domain.ErrCourtUnavailable) {
				t.Fatalf("misclassified an unrelated upstream failure as a court conflict: %v", err)
			}
			if !strings.Contains(err.Error(), up.err.Error()) {
				t.Fatalf("dropped the upstream message: %v", err)
			}
		})
	}
}

// TestReserveCourt_MalformedCourtIDDoesNotLeak covers the guard a Competitions
// caller can actually trip: bookingapp.CreateBooking answers a non-uuid
// CourtID with ErrInvalidCourtReference before either repository call, and
// that sentinel must not cross the boundary typed either.
func TestReserveCourt_MalformedCourtIDDoesNotLeak(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	_, err := res.ReserveCourt(context.Background(), "not-a-uuid", rng.Start, rng.End, fixtureCompetitionID)
	if err == nil {
		t.Fatal("want an error for a malformed court id")
	}
	if errors.Is(err, bookingdomain.ErrInvalidCourtReference) {
		t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
	}
	if len(repo.bookings) != 0 {
		t.Fatalf("a malformed court id reached the repository: %v", repo.bookings)
	}
}

// TestReserveCourt_InvalidRangeDoesNotLeak covers the adapter's own
// NewTimeRange guard — the branch its doc comment calls unreachable in
// practice, which is exactly the kind of branch that goes untested and then
// leaks a foreign error type the first time it does fire.
func TestReserveCourt_InvalidRangeDoesNotLeak(t *testing.T) {
	t.Parallel()

	res, _ := newReservation(t)
	rng := slot(t, 10, 11)

	// end before start.
	_, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.End, rng.Start, fixtureCompetitionID)
	if err == nil {
		t.Fatal("want an error for an inverted range")
	}
	if errors.Is(err, bookingdomain.ErrInvalidTimeRange) {
		t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
	}
}

// TestReleaseCourt_CancelsTheBooking proves the compensating action really
// cancels — asserted on the stored Booking's status, not on a nil error.
func TestReleaseCourt_CancelsTheBooking(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)
	seedBooking(repo, seededBookingID, rng)

	if err := res.ReleaseCourt(context.Background(), seededBookingID); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if got := repo.bookings[seededBookingID].Status; got != bookingdomain.StatusCancelled {
		t.Fatalf("stored Status = %q, want %q", got, bookingdomain.StatusCancelled)
	}

	// And the released slot is genuinely free again — the T3 standard for this
	// assertion: prove the slot can be re-booked, not just that a field flipped.
	if _, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureCompetitionID); err != nil {
		t.Fatalf("re-reserving the released slot = %v, want nil", err)
	}
}

// TestReleaseCourt_NeverLeaksBookingSentinels holds rule 5 on the rollback
// path, which has no translated sentinel at all — every failure it can have is
// %s-wrapped.
func TestReleaseCourt_NeverLeaksBookingSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		bookingID string
		failWith  error
		want      error
	}{
		{name: "unknown booking", bookingID: unknownBookingID, want: bookingdomain.ErrBookingNotFound},
		{name: "malformed booking id", bookingID: "not-a-uuid", want: bookingdomain.ErrBookingNotFound},
		{name: "infrastructure failure", bookingID: seededBookingID, failWith: errRepoUnavailable, want: errRepoUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, repo := newReservation(t)
			seedBooking(repo, seededBookingID, slot(t, 10, 11))
			repo.getFailsWith = tt.failWith

			err := res.ReleaseCourt(context.Background(), tt.bookingID)
			if err == nil {
				t.Fatal("want an error")
			}
			if errors.Is(err, tt.want) {
				t.Fatalf("leaked the upstream error across the boundary: %v", err)
			}
			if !strings.Contains(err.Error(), tt.want.Error()) {
				t.Fatalf("dropped the upstream message: %v", err)
			}
		})
	}
}
