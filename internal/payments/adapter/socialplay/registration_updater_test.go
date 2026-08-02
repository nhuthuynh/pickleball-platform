package socialplay_test

import (
	"context"
	"errors"
	"testing"

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
	svc := socialplayapp.NewService(fakeIDs{}, fakeGames{}, regs)
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

	svc := socialplayapp.NewService(fakeIDs{}, fakeGames{}, newFakeRegistrations())
	updater := paymentssocialplay.NewRegistrationUpdater(svc)

	err := updater.UpdatePaymentStatus(context.Background(), "no-such-registration", socialplaydomain.PaymentStatusPaid)
	if !errors.Is(err, socialplaydomain.ErrRegistrationNotFound) {
		t.Fatalf("got err %v, want %v", err, socialplaydomain.ErrRegistrationNotFound)
	}
}
