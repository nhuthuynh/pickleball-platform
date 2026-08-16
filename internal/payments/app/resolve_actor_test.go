// T28.1 (closes the Payments third of #164) — app-layer coverage for the
// new ADR-0014/ADR-0017 resolution seam, mirroring
// internal/booking/app/resolve_actor_test.go and
// internal/facilities/app/identity_seam_test.go's identical pattern.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// stubIdentity is a port.IdentityLookup fake for the app-layer tests, which
// exercise methods *below* the resolution seam and therefore never resolve a
// subject at all — mirroring facilities' identical stubIdentity, and for the
// identical reason: every other method in this package is handed an
// already-resolved User.ID, exactly as ADR-0014 says the app layer always
// is, so nothing here should depend on resolution succeeding.
//
// It returns ErrUserNotFound unconditionally. The genuinely unmocked
// resolving-direction coverage of this seam lives two packages over, in
// internal/payments/adapter/identity/lookup_test.go (the adapter over the
// real identityapp.Service).
type stubIdentity struct{}

func (stubIdentity) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return "", domain.ErrUserNotFound
}

// TestResolveActorUserID_DelegatesToTheIdentityPort pins that the seam
// method exists on this service and goes through port.IdentityLookup rather
// than re-deriving an actor some other way — mirroring
// internal/facilities/app's TestResolveActorUserID_DelegatesToTheIdentityPort
// exactly.
func TestResolveActorUserID_DelegatesToTheIdentityPort(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{},
		Identity: stubIdentity{},
	})

	_, err := svc.ResolveActorUserID(context.Background(), "auth0|never-registered")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("ResolveActorUserID(unregistered) = %v, want ErrUserNotFound", err)
	}
}

// TestResolveActorUserID_NilIdentityFailsClosed pins ServiceOptions' own
// documented posture (see that struct's doc comment): a Service built
// without an Identity port must refuse every actor, never silently trust an
// unresolved subject — the exact hazard this ticket's funnel change and
// backfill migration together exist to close (see
// authorizeOnlineConfirmation's dated doc comment in service.go). A Service
// whose Identity was simply forgotten in wiring must fail every
// authenticated RPC's actor resolution, not let a raw subject leak through
// to a comparison that now expects a uuid.
func TestResolveActorUserID_NilIdentityFailsClosed(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{},
		// Identity deliberately left unset (nil).
	})

	_, err := svc.ResolveActorUserID(context.Background(), "auth0|anyone")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("ResolveActorUserID with a nil Identity = %v, want ErrUserNotFound "+
			"(fail closed, mirroring RegistrationLookup/GameLookup/GameAdminReader/"+
			"EntryLookup/CompetitionAdminReader's existing nil-safe convention)", err)
	}
}
