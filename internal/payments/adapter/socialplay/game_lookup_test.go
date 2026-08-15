package socialplay_test

import (
	"context"
	"testing"

	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	paymentsport "github.com/nhuthuynh/white-label/internal/payments/port"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestGameLookup_ImplementsPort is a compile-time proof that
// *paymentssocialplay.GameLookup satisfies paymentsport.GameLookup.
var _ paymentsport.GameLookup = (*paymentssocialplay.GameLookup)(nil)

const gameLookupHostID = "host-subject-t16-2"

// TestHostIDForGame_Succeeds proves the happy path calls through to the real
// socialplayapp.Service.GetGame (T16.2 — Social Play's first single-Game
// app-layer read, see that method's own doc comment) and returns the Game's
// HostID — driven through the real service and reusing
// game_admin_reader_test.go's adminFakeGames/newAdminTestService (this
// package's already-established "minimal but FUNCTIONAL port.GameRepository"
// fixture), since HostIDForGame needs GetByID to actually resolve a seeded
// Game the same way ListGameAdmins' fixtures already do.
func TestHostIDForGame_Succeeds(t *testing.T) {
	t.Parallel()

	games := newAdminFakeGames(socialplaydomain.Game{ID: gameAdminReaderGameID, HostID: gameLookupHostID})
	svc := newAdminTestService(games, newAdminFakeGameAdmins())
	lookup := paymentssocialplay.NewGameLookup(svc)

	hostID, err := lookup.HostIDForGame(context.Background(), gameAdminReaderGameID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if hostID != gameLookupHostID {
		t.Fatalf("HostIDForGame = %q, want %q", hostID, gameLookupHostID)
	}
}

// TestHostIDForGame_UnknownGame proves an unknown gameID answers ("", nil) —
// mirrors GameAdminReader.ListGameAdmins' identical translation for the
// identical reason (port.GameLookup's own doc comment).
func TestHostIDForGame_UnknownGame(t *testing.T) {
	t.Parallel()

	games := newAdminFakeGames()
	svc := newAdminTestService(games, newAdminFakeGameAdmins())
	lookup := paymentssocialplay.NewGameLookup(svc)

	hostID, err := lookup.HostIDForGame(context.Background(), "00000000-0000-4000-8000-000000009999")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if hostID != "" {
		t.Fatalf("HostIDForGame = %q, want empty", hostID)
	}
}

// TestHostIDForGame_EmptyGameID proves the composition
// RegistrationLookup.GameIDForRegistration's own doc comment promises: an
// empty gameID (RegistrationLookup's safe answer for an unresolved
// Registration) reaches this method as a plain unknown-id lookup, with no
// special-casing required on either side.
func TestHostIDForGame_EmptyGameID(t *testing.T) {
	t.Parallel()

	games := newAdminFakeGames()
	svc := newAdminTestService(games, newAdminFakeGameAdmins())
	lookup := paymentssocialplay.NewGameLookup(svc)

	hostID, err := lookup.HostIDForGame(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if hostID != "" {
		t.Fatalf("HostIDForGame(\"\") = %q, want empty", hostID)
	}
}
