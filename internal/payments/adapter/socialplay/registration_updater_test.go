package socialplay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
	socialplayport "github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// fakeIDs is a minimal port.IDGenerator; unused by MarkRegistrationPaymentStatus
// but required by app.NewService's signature.
type fakeIDs struct{}

func (fakeIDs) NewID() string { return "unused" }

// fakeGames is a minimal port.GameRepository stub — MarkRegistrationPaymentStatus
// never touches it, but app.NewService requires one.
type fakeGames struct{}

func (fakeGames) Create(context.Context, socialplaydomain.Game) (socialplaydomain.Game, error) {
	return socialplaydomain.Game{}, nil
}
func (fakeGames) GetByID(context.Context, string) (socialplaydomain.Game, error) {
	return socialplaydomain.Game{}, socialplaydomain.ErrGameNotFound
}

// ListGames is a stub (T8.9): MarkRegistrationPaymentStatus never touches
// it, but app.NewService's port.GameRepository requires it now.
func (fakeGames) ListGames(context.Context, socialplayport.GameListingFilter) ([]socialplayport.GameListing, error) {
	return nil, nil
}

// UpdateStatus is a stub for the same reason as ListGames (T12.4's
// CancelGame added it to port.GameRepository): MarkRegistrationPaymentStatus
// never touches the Game aggregate's own status.
func (fakeGames) UpdateStatus(context.Context, string, socialplaydomain.Status) (socialplaydomain.Game, error) {
	return socialplaydomain.Game{}, socialplaydomain.ErrGameNotFound
}

// fakeGameAdmins is a minimal port.GameAdminRepository stub (T14.4):
// MarkRegistrationPaymentStatus never touches the Game-Admin store, but
// socialplayapp.NewService requires every ServiceOptions field, so this
// package needs one to construct a Service at all. Same "stub, because the
// constructor requires it" role fakeGames above has had since T8.9.
//
// Worth noting rather than dismissing, since this is the Payments adapter:
// the OTHER half of #168 — the locked "per-game Game Admins can record
// offline payments" decision — is the work that will make this stub a real
// dependency. T14.4 built the Social Play half only.
type fakeGameAdmins struct{}

func newFakeGameAdminRepo() fakeGameAdmins { return fakeGameAdmins{} }

func (fakeGameAdmins) Assign(context.Context, socialplaydomain.GameAdmin) (socialplaydomain.GameAdmin, error) {
	return socialplaydomain.GameAdmin{}, socialplaydomain.ErrGameNotFound
}

func (fakeGameAdmins) Revoke(context.Context, string, string) error {
	return socialplaydomain.ErrGameAdminNotFound
}

func (fakeGameAdmins) ListGameAdmins(context.Context, string) ([]socialplaydomain.GameAdmin, error) {
	return nil, nil
}

// fakeRegistrations is a minimal in-memory port.RegistrationRepository —
// just enough for RegistrationUpdater's own tests (this adapter's job is
// the translation at the boundary, not re-testing app.Service's own
// behaviour, which internal/socialplay/app/service_test.go already covers).
type fakeRegistrations struct {
	regs map[string]socialplaydomain.Registration
}

func newFakeRegistrations(seed ...socialplaydomain.Registration) *fakeRegistrations {
	m := make(map[string]socialplaydomain.Registration, len(seed))
	for _, r := range seed {
		m[r.ID] = r
	}
	return &fakeRegistrations{regs: m}
}

func (f *fakeRegistrations) Create(_ context.Context, r socialplaydomain.Registration) (socialplaydomain.Registration, error) {
	f.regs[r.ID] = r
	return r, nil
}
func (f *fakeRegistrations) GetByID(_ context.Context, id string) (socialplaydomain.Registration, error) {
	r, ok := f.regs[id]
	if !ok {
		return socialplaydomain.Registration{}, socialplaydomain.ErrRegistrationNotFound
	}
	return r, nil
}
func (f *fakeRegistrations) ListActiveForGame(_ context.Context, gameID string) ([]socialplaydomain.Registration, error) {
	var out []socialplaydomain.Registration
	for _, r := range f.regs {
		if r.GameID == gameID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRegistrations) Update(_ context.Context, r socialplaydomain.Registration) (socialplaydomain.Registration, error) {
	if _, ok := f.regs[r.ID]; !ok {
		return socialplaydomain.Registration{}, socialplaydomain.ErrRegistrationNotFound
	}
	f.regs[r.ID] = r
	return r, nil
}
func (f *fakeRegistrations) UpdatePaymentStatus(_ context.Context, id string, status socialplaydomain.PaymentStatus) (socialplaydomain.Registration, error) {
	r, ok := f.regs[id]
	if !ok {
		return socialplaydomain.Registration{}, socialplaydomain.ErrRegistrationNotFound
	}
	r.PaymentStatus = status
	f.regs[id] = r
	return r, nil
}

// CancelAllActiveForGame (T16.3) is unexercised by this test — the
// Payments-context updater this file tests never cancels a Game — so,
// like the stub methods below, it returns a zero value.
func (f *fakeRegistrations) CancelAllActiveForGame(context.Context, string) (int, error) {
	return 0, nil
}

// fakeWaitlist is a minimal, empty-always port.WaitlistRepository stub,
// mirroring internal/socialplay/adapter/grpcapi/authz_regression_test.go's
// fakeWaitlistRepo: MarkRegistrationPaymentStatus (this package's own
// surface, exercised via socialplayapp.Service.UpdateRegistrationPaymentStatus)
// never touches the waitlist, but socialplayapp.NewService has required a
// port.WaitlistRepository argument since T6.6 — required only to satisfy
// that constructor signature, not exercised by any test in this file.
type fakeWaitlist struct{}

func (fakeWaitlist) Create(context.Context, socialplaydomain.WaitlistEntry) (socialplaydomain.WaitlistEntry, error) {
	return socialplaydomain.WaitlistEntry{}, nil
}
func (fakeWaitlist) GetByID(context.Context, string) (socialplaydomain.WaitlistEntry, error) {
	return socialplaydomain.WaitlistEntry{}, socialplaydomain.ErrWaitlistEntryNotFound
}
func (fakeWaitlist) ListForGame(context.Context, string) ([]socialplaydomain.WaitlistEntry, error) {
	return nil, nil
}
func (fakeWaitlist) PromoteNext(context.Context, string, time.Time) (socialplaydomain.WaitlistEntry, error) {
	return socialplaydomain.WaitlistEntry{}, socialplaydomain.ErrNoWaitingEntries
}
func (fakeWaitlist) ExpirePromotion(context.Context, string, time.Time) (socialplaydomain.WaitlistEntry, error) {
	return socialplaydomain.WaitlistEntry{}, socialplaydomain.ErrWaitlistEntryNotFound
}

// fakeMatches is a minimal, empty-always port.MatchRepository stub (T10.4),
// mirroring fakeWaitlist's identical role: MarkRegistrationPaymentStatus
// never touches matches, but socialplayapp.NewService has required a
// port.MatchRepository argument since T10.4 — required only to satisfy
// that constructor signature, not exercised by any test in this file.
type fakeMatches struct{}

func (fakeMatches) Create(context.Context, socialplaydomain.Match) (socialplaydomain.Match, error) {
	return socialplaydomain.Match{}, nil
}
func (fakeMatches) ListForGame(context.Context, string) ([]socialplaydomain.Match, error) {
	return nil, nil
}

// TestRegistrationUpdater_ImplementsPort is a compile-time proof that
// *paymentssocialplay.RegistrationUpdater satisfies
// socialplayport.RegistrationPaymentUpdater — the whole point of this
// package (mirror image of internal/socialplay/adapter/booking implementing
// internal/socialplay/port.CourtReservation against Booking's real
// app.Service).
var _ socialplayport.RegistrationPaymentUpdater = (*paymentssocialplay.RegistrationUpdater)(nil)

// TestUpdatePaymentStatus_Succeeds proves the happy path calls through to
// the real Social Play app.Service and the change is observable via a
// fresh GetByID — not just that no error was returned.
func TestUpdatePaymentStatus_Succeeds(t *testing.T) {
	t.Parallel()

	regs := newFakeRegistrations(socialplaydomain.Registration{
		ID: "reg-1", GameID: "game-1", PlayerID: "player-1",
		Status:        socialplaydomain.RegistrationStatusRegistered,
		PaymentStatus: socialplaydomain.PaymentStatusUnpaid,
	})
	svc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           fakeIDs{},
		Games:         fakeGames{},
		Registrations: regs,
		Waitlist:      fakeWaitlist{},
		Matches:       fakeMatches{},
		GameAdmins:    newFakeGameAdminRepo(),
	})
	updater := paymentssocialplay.NewRegistrationUpdater(svc)

	if err := updater.UpdatePaymentStatus(context.Background(), "reg-1", socialplaydomain.PaymentStatusPaid); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	stored, err := regs.GetByID(context.Background(), "reg-1")
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	if stored.PaymentStatus != socialplaydomain.PaymentStatusPaid {
		t.Fatalf("PaymentStatus = %v, want paid", stored.PaymentStatus)
	}
}

// TestUpdatePaymentStatus_NotFound proves a bogus registration id surfaces
// socialplaydomain.ErrRegistrationNotFound unchanged — the exact sentinel
// port.RegistrationPaymentUpdater's doc comment promises, not some
// Payments-side wrapper error.
func TestUpdatePaymentStatus_NotFound(t *testing.T) {
	t.Parallel()

	svc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           fakeIDs{},
		Games:         fakeGames{},
		Registrations: newFakeRegistrations(),
		Waitlist:      fakeWaitlist{},
		Matches:       fakeMatches{},
		GameAdmins:    newFakeGameAdminRepo(),
	})
	updater := paymentssocialplay.NewRegistrationUpdater(svc)

	err := updater.UpdatePaymentStatus(context.Background(), "no-such-registration", socialplaydomain.PaymentStatusPaid)
	if !errors.Is(err, socialplaydomain.ErrRegistrationNotFound) {
		t.Fatalf("got err %v, want %v", err, socialplaydomain.ErrRegistrationNotFound)
	}
}
