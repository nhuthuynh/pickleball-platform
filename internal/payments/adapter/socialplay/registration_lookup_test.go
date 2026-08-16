package socialplay_test

import (
	"context"
	"testing"

	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	paymentsport "github.com/nhuthuynh/white-label/internal/payments/port"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestRegistrationLookup_ImplementsPort is a compile-time proof that
// *paymentssocialplay.RegistrationLookup satisfies
// paymentsport.RegistrationLookup — mirroring
// TestGameAdminReader_ImplementsPort's (game_admin_reader_test.go) role for
// this new T16.2 port.
var _ paymentsport.RegistrationLookup = (*paymentssocialplay.RegistrationLookup)(nil)

const (
	registrationLookupGameID = "6ba7b810-0000-4000-8000-0000000000f2"
	registrationLookupRegID  = "6ba7b810-0000-4000-8000-0000000000f3"
)

func newRegistrationLookupTestService(regs *fakeRegistrations) *socialplayapp.Service {
	return socialplayapp.NewService(socialplayapp.ServiceOptions{
		Identity:      fakeSocialplayIdentityLookup{},
		IDs:           fakeIDs{},
		Games:         fakeGames{},
		Registrations: regs,
		Waitlist:      fakeWaitlist{},
		Matches:       fakeMatches{},
		GameAdmins:    newFakeGameAdminRepo(),
	})
}

// TestGameIDForRegistration_Succeeds proves the happy path calls through to
// the real socialplayapp.Service.GetRegistrationByID (T16.2) and returns
// that Registration's own GameID — driven through the real service, not a
// fake app.Service, per T14.8/T15.5's cross-context-fake warning
// (game_admin_reader_test.go's own doc comment names this ticket's own
// instruction 9 as the place that warning is repeated).
func TestGameIDForRegistration_Succeeds(t *testing.T) {
	t.Parallel()

	regs := newFakeRegistrations(socialplaydomain.Registration{
		ID:            registrationLookupRegID,
		GameID:        registrationLookupGameID,
		PlayerID:      "player-1",
		Status:        socialplaydomain.RegistrationStatusRegistered,
		PaymentStatus: socialplaydomain.PaymentStatusUnpaid,
	})
	svc := newRegistrationLookupTestService(regs)
	lookup := paymentssocialplay.NewRegistrationLookup(svc)

	gameID, err := lookup.GameIDForRegistration(context.Background(), registrationLookupRegID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gameID != registrationLookupGameID {
		t.Fatalf("GameIDForRegistration = %q, want %q", gameID, registrationLookupGameID)
	}
}

// TestGameIDForRegistration_UnknownRegistration proves an unknown
// registrationID answers ("", nil) — port.RegistrationLookup's own
// documented translation of socialplaydomain.ErrRegistrationNotFound,
// deliberately different from socialplayapp.Service.GetRegistrationByID's
// own domain.ErrRegistrationNotFound answer for the same input.
func TestGameIDForRegistration_UnknownRegistration(t *testing.T) {
	t.Parallel()

	svc := newRegistrationLookupTestService(newFakeRegistrations())
	lookup := paymentssocialplay.NewRegistrationLookup(svc)

	gameID, err := lookup.GameIDForRegistration(context.Background(), "00000000-0000-4000-8000-000000009999")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gameID != "" {
		t.Fatalf("GameIDForRegistration = %q, want empty", gameID)
	}
}
