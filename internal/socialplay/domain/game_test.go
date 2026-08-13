package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

func validCourtIDs() []string {
	return []string{"court-1"}
}

// validEntryFee is the "this test isn't about the fee" fixture (T9.2): a
// well-formed, non-free EntryFee, so the tables below keep asserting
// exactly what they asserted before EntryFee existed. Fee-specific
// behaviour lives in money_test.go.
func validEntryFee() domain.Money {
	return domain.Money{Cents: 1500, Currency: "USD"}
}

// TestNewGame_Valid proves the happy path: a well-formed Game is constructed
// with Status starting at "scheduled".
func TestNewGame_Valid(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", "venue-1", validCourtIDs(), r, 4, domain.PaymentMethodOnline, 2, validEntryFee())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.Status != domain.StatusScheduled {
		t.Fatalf("status = %v, want %v", g.Status, domain.StatusScheduled)
	}
	if g.Capacity != 4 {
		t.Fatalf("capacity = %d, want 4", g.Capacity)
	}
	if g.ID != "g1" || g.HostID != "host-1" || g.FacilityID != "facility-1" || g.VenueFacilityID != "venue-1" {
		t.Fatalf("unexpected identity fields: %+v", g)
	}
	if g.PaymentMethod != domain.PaymentMethodOnline {
		t.Fatalf("PaymentMethod = %v, want %v", g.PaymentMethod, domain.PaymentMethodOnline)
	}
	if g.GuestAllowance != 2 {
		t.Fatalf("GuestAllowance = %d, want 2", g.GuestAllowance)
	}
}

// TestNewGame_Validation is the required table-driven boundary coverage:
// Capacity == 0, negative capacity, empty CourtIDs, zero-duration range.
func TestNewGame_Validation(t *testing.T) {
	t.Parallel()

	validRange := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	// Zero-duration range built by bypassing NewTimeRange (which itself
	// rejects this) via the exported struct fields directly — this is how a
	// caller could hand NewGame an already-invalid range, so NewGame must
	// re-validate rather than trust its input, mirroring the "end after
	// start" rule booking.NewTimeRange enforces.
	sameInstant := mustTime(t, "2026-08-03T09:00:00Z")
	zeroDurationRange := domain.TimeRange{Start: sameInstant, End: sameInstant}

	tests := []struct {
		name           string
		hostID         string
		courtIDs       []string
		r              domain.TimeRange
		capacity       int
		paymentMethod  domain.PaymentMethod
		guestAllowance int
		wantErr        error
	}{
		{"capacity zero is rejected", "host-1", validCourtIDs(), validRange, 0, domain.PaymentMethodEither, 0, domain.ErrInvalidCapacity},
		{"negative capacity is rejected", "host-1", validCourtIDs(), validRange, -1, domain.PaymentMethodEither, 0, domain.ErrInvalidCapacity},
		{"empty court ids is rejected", "host-1", []string{}, validRange, 4, domain.PaymentMethodEither, 0, domain.ErrEmptyCourtIDs},
		{"nil court ids is rejected", "host-1", nil, validRange, 4, domain.PaymentMethodEither, 0, domain.ErrEmptyCourtIDs},
		{"zero-duration range is rejected", "host-1", validCourtIDs(), zeroDurationRange, 4, domain.PaymentMethodEither, 0, domain.ErrInvalidTimeRange},
		{"valid inputs accepted", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethodEither, 0, nil},
		{"payment method online accepted", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethodOnline, 0, nil},
		{"payment method cash accepted", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethodCash, 0, nil},
		{"invalid payment method string is rejected", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethod("crypto"), 0, domain.ErrInvalidPaymentMethod},
		{"empty payment method string is rejected", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethod(""), 0, domain.ErrInvalidPaymentMethod},
		{"negative guest allowance is rejected", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethodEither, -1, domain.ErrInvalidGuestAllowance},
		{"positive guest allowance accepted", "host-1", validCourtIDs(), validRange, 1, domain.PaymentMethodEither, 3, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewGame("g1", tt.hostID, "facility-1", "venue-1", tt.courtIDs, tt.r, tt.capacity, tt.paymentMethod, tt.guestAllowance, validEntryFee())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestGame_Cancel proves the scheduled -> cancelled transition and that a
// double-cancel is rejected rather than silently accepted, mirroring
// booking.Booking.Cancel()'s "illegal transition rejected" pattern. Note:
// cascading this cancellation to the Game's Bookings/Registrations is
// explicitly out of scope for T5.1 (see PR description / HANDOFF follow-up).
func TestGame_Cancel(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", "venue-1", validCourtIDs(), r, 4, domain.PaymentMethodEither, 0, validEntryFee())
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := g.Cancel(); err != nil {
		t.Fatalf("first cancel should succeed, got %v", err)
	}
	if g.Status != domain.StatusCancelled {
		t.Fatalf("status = %v, want cancelled", g.Status)
	}

	if err := g.Cancel(); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("double-cancel should be rejected, got %v", err)
	}
}

// TestNewGame_EmptyVenueFacilityIDAccepted proves VenueFacilityID (T8.3) is
// optional at the domain layer: NewGame stores whatever it's given (empty
// string included) without validating its existence — that check requires
// the Facilities context, which is app.Service.ScheduleGame's job via
// port.FacilityLookup, not this pure constructor's (see VenueFacilityID's
// doc comment on game.go).
func TestNewGame_EmptyVenueFacilityIDAccepted(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", "", validCourtIDs(), r, 4, domain.PaymentMethodEither, 0, validEntryFee())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.VenueFacilityID != "" {
		t.Fatalf("VenueFacilityID = %q, want empty", g.VenueFacilityID)
	}
}

// --- T10.4: EnsureHostOrGameAdmin / EnsureNotCancelled ---------------------

// TestGame_EnsureHostOrGameAdmin is the required table-driven boundary
// coverage for RecordMatchResult's object-level (BOLA) authorization check:
// the Host is always allowed, an assigned Game Admin is allowed, and every
// other actor -- including an empty one, even against a Game with an empty
// HostID -- is rejected with the single, flat ErrNotGameHostOrAdmin
// sentinel (mirrors competitions.Competition.EnsureHost's table shape).
func TestGame_EnsureHostOrGameAdmin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		hostID         string
		actorUserID    string
		assignedAdmins []string
		wantErr        error
	}{
		{"host is allowed", "host-1", "host-1", nil, nil},
		{"assigned game admin is allowed", "host-1", "admin-2", []string{"admin-1", "admin-2"}, nil},
		{"mismatched actor is rejected", "host-1", "random-player", []string{"admin-1"}, domain.ErrNotGameHostOrAdmin},
		{"empty actor is rejected", "host-1", "", []string{"admin-1"}, domain.ErrNotGameHostOrAdmin},
		{"empty actor against an empty host is still rejected", "", "", nil, domain.ErrNotGameHostOrAdmin},
		{"a blank entry in the admin list never matches", "host-1", "", []string{""}, domain.ErrNotGameHostOrAdmin},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := domain.Game{HostID: tt.hostID}
			err := g.EnsureHostOrGameAdmin(tt.actorUserID, tt.assignedAdmins)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestGame_EnsureNotCancelled proves the precondition RecordMatchResult
// needs: a scheduled Game passes, a cancelled Game is rejected with
// ErrGameCancelled.
func TestGame_EnsureNotCancelled(t *testing.T) {
	t.Parallel()

	scheduled := domain.Game{Status: domain.StatusScheduled}
	if err := scheduled.EnsureNotCancelled(); err != nil {
		t.Fatalf("scheduled game: got err %v, want nil", err)
	}

	cancelled := domain.Game{Status: domain.StatusCancelled}
	if err := cancelled.EnsureNotCancelled(); !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("cancelled game: got err %v, want %v", err, domain.ErrGameCancelled)
	}
}
