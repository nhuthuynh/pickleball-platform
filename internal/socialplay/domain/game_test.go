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
// coverage for the object-level (BOLA) authorization check RecordMatchResult
// (T10.4) and the roster read (T14.5) share: the Host is always allowed, an
// assigned Game Admin is allowed, and every other actor -- including an empty
// one, even against a Game with an empty HostID -- is rejected with the single,
// flat ErrNotGameHostOrAdmin sentinel (mirrors
// competitions.Competition.EnsureHost's table shape).
//
// T14.5 changed the second parameter from []string to []GameAdmin. The rows
// below therefore hold *resolved store rows* rather than a caller's assertion,
// and the cross-Game row is new: a GameAdmin carries the GameID its authority
// is scoped to, which a bare string never could.
func TestGame_EnsureHostOrGameAdmin(t *testing.T) {
	t.Parallel()

	const thisGame = "game-under-test"
	admin := func(userID string) domain.GameAdmin {
		return domain.GameAdmin{GameID: thisGame, UserID: userID, AssignedBy: "host-1"}
	}

	tests := []struct {
		name        string
		hostID      string
		actorUserID string
		assigned    []domain.GameAdmin
		wantErr     error
	}{
		{"host is allowed", "host-1", "host-1", nil, nil},
		{"assigned game admin is allowed", "host-1", "admin-2", []domain.GameAdmin{admin("admin-1"), admin("admin-2")}, nil},
		{"host is allowed even with no admins assigned", "host-1", "host-1", []domain.GameAdmin{}, nil},
		{"mismatched actor is rejected", "host-1", "random-player", []domain.GameAdmin{admin("admin-1")}, domain.ErrNotGameHostOrAdmin},
		{"empty actor is rejected", "host-1", "", []domain.GameAdmin{admin("admin-1")}, domain.ErrNotGameHostOrAdmin},
		{"empty actor against an empty host is still rejected", "", "", nil, domain.ErrNotGameHostOrAdmin},
		// A blank stored UserID must not match a blank actor. AssignGameAdmin
		// refuses to create such a row, but a rule that relies on the other end
		// having been enforced breaks the first time a row arrives from
		// somewhere else (a backfill, a manual insert).
		{"a blank stored admin row never matches", "host-1", "", []domain.GameAdmin{admin("")}, domain.ErrNotGameHostOrAdmin},
		// Authority is per-Game. The caller of this method is responsible for
		// passing the set scoped to *this* Game (app.Service does, by querying
		// with the Game's own id); this row states what the method itself does
		// with a row whose UserID matches — it matches. That is deliberate: the
		// scoping lives in the query, and duplicating it here as a GameID
		// filter would create two places for it to be wrong. See
		// TestListRegistrationsForGame_UnentitledActorsAreStillRefused's
		// "an admin of a different Game" row for the end-to-end proof that the
		// scoping actually holds.
		{"a row for another Game still matches on user id alone", "host-1", "admin-9",
			[]domain.GameAdmin{{GameID: "some-other-game", UserID: "admin-9"}}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := domain.Game{ID: thisGame, HostID: tt.hostID}
			err := g.EnsureHostOrGameAdmin(tt.actorUserID, tt.assigned)
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

// --- T12.4: EnsureHost -----------------------------------------------------

// TestGame_EnsureHost is the required table-driven boundary coverage for
// CancelGame's object-level (BOLA) authorization check. Deliberately
// separate from TestGame_EnsureHostOrGameAdmin above, and asserting a
// *different* sentinel: cancelling a Game is Host-only, so a Game Admin who
// would be allowed to record a match result is rejected here. That
// difference is the whole point of ErrNotGameHost existing alongside
// ErrNotGameHostOrAdmin — see ErrNotGameHost's doc comment in errors.go.
func TestGame_EnsureHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		hostID        string
		actorPlayerID string
		wantErr       error
	}{
		{"host is allowed", "host-1", "host-1", nil},
		{"mismatched actor is rejected", "host-1", "random-player", domain.ErrNotGameHost},
		{"empty actor is rejected", "host-1", "", domain.ErrNotGameHost},
		{"empty actor against an empty host is still rejected", "", "", domain.ErrNotGameHost},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := domain.Game{HostID: tt.hostID}
			err := g.EnsureHost(tt.actorPlayerID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestGame_EnsureHost_GameAdminIsRejected pins the distinction that makes
// ErrNotGameHost a separate sentinel rather than a reuse of
// ErrNotGameHostOrAdmin: the same actor that EnsureHostOrGameAdmin admits as an
// assigned Game Admin (and who may therefore record a match result and read the
// roster) is NOT permitted to cancel the Game, nor to assign or revoke another
// admin. A future "simplification" that unified the two checks would fail here.
//
// T14.5 makes this test more load-bearing, not less: now that the admin set is
// a durable, trustworthy fact rather than a caller's assertion, the temptation
// to route the remaining Host-only rules through it is real, and this is what
// stands in the way.
func TestGame_EnsureHost_GameAdminIsRejected(t *testing.T) {
	t.Parallel()

	g := domain.Game{HostID: "host-1"}
	const gameAdmin = "admin-1"

	if err := g.EnsureHostOrGameAdmin(gameAdmin, []domain.GameAdmin{{GameID: g.ID, UserID: gameAdmin}}); err != nil {
		t.Fatalf("fixture precondition: an assigned game admin must pass EnsureHostOrGameAdmin, got %v", err)
	}
	if err := g.EnsureHost(gameAdmin); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("a game admin must NOT be able to cancel the game: got err %v, want %v", err, domain.ErrNotGameHost)
	}
}
