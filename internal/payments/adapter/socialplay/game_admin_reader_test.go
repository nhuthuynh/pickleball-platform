package socialplay_test

import (
	"context"
	"testing"

	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	paymentsport "github.com/nhuthuynh/white-label/internal/payments/port"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
	socialplayport "github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// adminFakeGames is a minimal but FUNCTIONAL port.GameRepository — unlike
// registration_updater_test.go's fakeGames (which always answers
// ErrGameNotFound, because MarkRegistrationPaymentStatus never touches it),
// GetByID here must actually resolve a seeded Game: AssignGameAdmin,
// RevokeGameAdmin and ListGameAdmins all load the Game first.
type adminFakeGames struct {
	games map[string]socialplaydomain.Game
}

func newAdminFakeGames(seed ...socialplaydomain.Game) *adminFakeGames {
	m := make(map[string]socialplaydomain.Game, len(seed))
	for _, g := range seed {
		m[g.ID] = g
	}
	return &adminFakeGames{games: m}
}

func (f *adminFakeGames) Create(_ context.Context, g socialplaydomain.Game) (socialplaydomain.Game, error) {
	f.games[g.ID] = g
	return g, nil
}
func (f *adminFakeGames) GetByID(_ context.Context, id string) (socialplaydomain.Game, error) {
	g, ok := f.games[id]
	if !ok {
		return socialplaydomain.Game{}, socialplaydomain.ErrGameNotFound
	}
	return g, nil
}
func (f *adminFakeGames) ListGames(context.Context, socialplayport.GameListingFilter) ([]socialplayport.GameListing, error) {
	return nil, nil
}
func (f *adminFakeGames) UpdateStatus(context.Context, string, socialplaydomain.Status) (socialplaydomain.Game, error) {
	return socialplaydomain.Game{}, socialplaydomain.ErrGameNotFound
}

// adminFakeGameAdmins is a minimal but FUNCTIONAL port.GameAdminRepository —
// the in-memory store this test actually exercises Assign/Revoke/
// ListGameAdmins against. Mirrors internal/socialplay/app/service_test.go's
// own fakeGameAdminRepository (unreachable from here: that one lives in
// package app_test, unexported), reproduced independently rather than
// imported because this package cannot see it.
//
// Deliberately real, not a stub that "returns whatever it's told" — T14.8's
// cross-context-fake trap this ticket's own instruction 7 names: proving
// GameAdminReader works means proving it against a store that actually
// enforces assignment/revocation, driven through the real
// socialplayapp.Service on top of it (see TestListGameAdmins_AssignThenRevoke
// below), not against a fake that would report "admin" for any input.
type adminFakeGameAdmins struct {
	admins map[string][]socialplaydomain.GameAdmin
}

func newAdminFakeGameAdmins() *adminFakeGameAdmins {
	return &adminFakeGameAdmins{admins: make(map[string][]socialplaydomain.GameAdmin)}
}

func (r *adminFakeGameAdmins) Assign(_ context.Context, a socialplaydomain.GameAdmin) (socialplaydomain.GameAdmin, error) {
	for _, existing := range r.admins[a.GameID] {
		if existing.UserID == a.UserID {
			return socialplaydomain.GameAdmin{}, socialplaydomain.ErrAlreadyGameAdmin
		}
	}
	r.admins[a.GameID] = append(r.admins[a.GameID], a)
	return a, nil
}

func (r *adminFakeGameAdmins) Revoke(_ context.Context, gameID, userID string) error {
	for i, existing := range r.admins[gameID] {
		if existing.UserID == userID {
			r.admins[gameID] = append(r.admins[gameID][:i], r.admins[gameID][i+1:]...)
			return nil
		}
	}
	return socialplaydomain.ErrGameAdminNotFound
}

func (r *adminFakeGameAdmins) ListGameAdmins(_ context.Context, gameID string) ([]socialplaydomain.GameAdmin, error) {
	out := make([]socialplaydomain.GameAdmin, len(r.admins[gameID]))
	copy(out, r.admins[gameID])
	return out, nil
}

func newAdminTestService(games *adminFakeGames, admins *adminFakeGameAdmins) *socialplayapp.Service {
	return socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           fakeIDs{},
		Games:         games,
		Registrations: newFakeRegistrations(),
		Waitlist:      fakeWaitlist{},
		Matches:       fakeMatches{},
		GameAdmins:    admins,
	})
}

// TestGameAdminReader_ImplementsPort is a compile-time proof that
// *paymentssocialplay.GameAdminReader satisfies paymentsport.GameAdminReader
// — mirroring TestRegistrationUpdater_ImplementsPort's role
// (registration_updater_test.go) for the push-out adapter, now for the
// read-in direction (T15.5, §A12 GAP C).
var _ paymentsport.GameAdminReader = (*paymentssocialplay.GameAdminReader)(nil)

const (
	gameAdminReaderGameID = "6ba7b810-0000-4000-8000-0000000000f1"
	gameAdminReaderHostID = "host-subject-t15-5"
	gameAdminReaderAdmin  = "admin-subject-t15-5"
)

func gameAdminReaderFixture() (*socialplayapp.Service, *adminFakeGameAdmins) {
	games := newAdminFakeGames(socialplaydomain.Game{ID: gameAdminReaderGameID, HostID: gameAdminReaderHostID})
	admins := newAdminFakeGameAdmins()
	return newAdminTestService(games, admins), admins
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// TestListGameAdmins_AssignThenRevoke is T15.5 instruction 7's headline
// positive-then-negative control, delivered at the seam this ticket could
// actually wire (see GameAdminReader's own doc comment for why it stops
// short of internal/payments/app.Service): the SAME user appears in
// ListGameAdmins once genuinely assigned through the real Social Play
// app.Service, and disappears again once revoked — driven through the real
// socialplayapp.Service, not a fake that "returns whatever it's told"
// (T14.8's cross-context-fake note, which this ticket's own instruction 7
// names as the trap to avoid).
func TestListGameAdmins_AssignThenRevoke(t *testing.T) {
	t.Parallel()

	svc, _ := gameAdminReaderFixture()
	reader := paymentssocialplay.NewGameAdminReader(svc)
	ctx := context.Background()

	before, err := reader.ListGameAdmins(ctx, gameAdminReaderGameID)
	if err != nil {
		t.Fatalf("ListGameAdmins (before assign) unexpected error: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ListGameAdmins (before assign) = %v, want empty", before)
	}

	if _, err := svc.AssignGameAdmin(ctx, socialplayapp.AssignGameAdminInput{
		GameID:      gameAdminReaderGameID,
		ActorUserID: gameAdminReaderHostID,
		AdminUserID: gameAdminReaderAdmin,
	}); err != nil {
		t.Fatalf("AssignGameAdmin: %v", err)
	}

	assigned, err := reader.ListGameAdmins(ctx, gameAdminReaderGameID)
	if err != nil {
		t.Fatalf("ListGameAdmins (after assign) unexpected error: %v", err)
	}
	if !containsString(assigned, gameAdminReaderAdmin) {
		t.Fatalf("ListGameAdmins (after assign) = %v, want it to contain %s", assigned, gameAdminReaderAdmin)
	}

	if err := svc.RevokeGameAdmin(ctx, socialplayapp.RevokeGameAdminInput{
		GameID:      gameAdminReaderGameID,
		ActorUserID: gameAdminReaderHostID,
		AdminUserID: gameAdminReaderAdmin,
	}); err != nil {
		t.Fatalf("RevokeGameAdmin: %v", err)
	}

	revoked, err := reader.ListGameAdmins(ctx, gameAdminReaderGameID)
	if err != nil {
		t.Fatalf("ListGameAdmins (after revoke) unexpected error: %v", err)
	}
	if containsString(revoked, gameAdminReaderAdmin) {
		t.Fatalf("ListGameAdmins (after revoke) = %v, want it to NOT contain %s", revoked, gameAdminReaderAdmin)
	}
}

// TestListGameAdmins_UnknownGame proves an unknown gameID answers an empty
// slice with no error — GameAdminReader's own doc comment's translation,
// deliberately different from socialplayapp.Service.ListGameAdmins' own
// domain.ErrGameNotFound answer for the same input.
func TestListGameAdmins_UnknownGame(t *testing.T) {
	t.Parallel()

	svc, _ := gameAdminReaderFixture()
	reader := paymentssocialplay.NewGameAdminReader(svc)

	got, err := reader.ListGameAdmins(context.Background(), "00000000-0000-4000-8000-000000009999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}
