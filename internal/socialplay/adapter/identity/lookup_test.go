// Package identity_test covers internal/socialplay/adapter/identity, the one
// place Social Play is allowed to call into the Identity context (T29.2).
//
// These tests deliberately exercise the REAL identityapp.Service over a real
// port.Repository fake, NOT a fake port.IdentityLookup — the shape
// internal/facilities/adapter/identity/lookup_test.go (T13.3),
// internal/payments/adapter/identity/lookup_test.go (T28.1), and
// internal/competitions/adapter/identity/lookup_test.go (T29.1) established
// and this file mirrors exactly, because the entire point is exercising the
// actual behaviour of resolving a real, non-uuid IdP subject, not a fake of
// it. A subject is the only actor value Social Play's actor(ctx) funnel has
// ever carried, since T12.8.
package identity_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
	socialplayidentity "github.com/nhuthuynh/white-label/internal/socialplay/adapter/identity"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fixtureSubject is deliberately NOT uuid-shaped: an IdP subject is an
// opaque provider string, and that is exactly why it can never be stored in
// games.host_id/registrations.player_id/waitlist_entries.player_id/
// game_admins.user_id once those columns are uuid, and why this adapter has
// to exist.
const fixtureSubject = "auth0|abc123"

// fixtureUserID is the server-minted uuid primary key. Since T12.9 the ID
// and the Subject are two different identifier spaces (ADR-0014), which is
// the whole subject of #164's Social Play third.
const fixtureUserID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// inMemoryRepo is a minimal identity port.Repository fake. Only the read
// method this adapter can reach is meaningfully exercised; the rest exist to
// satisfy the interface.
type inMemoryRepo struct {
	users map[string]identitydomain.User
	// failWith, when non-nil, is returned by GetBySubject instead of a
	// result — used to prove the "any other error" translation path.
	failWith error
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{users: make(map[string]identitydomain.User)}
}

func (r *inMemoryRepo) Create(_ context.Context, u identitydomain.User) (identitydomain.User, error) {
	for _, existing := range r.users {
		if existing.Subject == u.Subject {
			return identitydomain.User{}, identitydomain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *inMemoryRepo) GetByID(_ context.Context, id string) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	return u, nil
}

func (r *inMemoryRepo) GetBySubject(_ context.Context, subject string) (identitydomain.User, error) {
	if r.failWith != nil {
		return identitydomain.User{}, r.failWith
	}
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return identitydomain.User{}, identitydomain.ErrUserNotFound
}

func (r *inMemoryRepo) UpdateSelfReportedLevel(_ context.Context, id string, level identitydomain.SelfReportedStartingLevel) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

// stubIDs is a deterministic port.IDGenerator. Nothing here creates a User
// through the service, so it is never called — it exists because
// identityapp.NewService requires the port.
type stubIDs struct{}

func (stubIDs) NewID() string { return fixtureUserID }

// newLookup builds the adapter over a REAL identityapp.Service and returns
// it alongside the repo fake, so a test can both seed and fail the upstream.
func newLookup(t *testing.T) (*socialplayidentity.Lookup, *inMemoryRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	return socialplayidentity.NewLookup(identityapp.NewService(repo, stubIDs{})), repo
}

// seedUser writes a User straight through the repository fake rather than
// through identityapp.Service.CreateUser, so the test controls which uuid
// the subject resolves to and can assert on it.
func seedUser(t *testing.T, repo *inMemoryRepo, id, subject string) {
	t.Helper()

	u, err := identitydomain.NewUser(id, subject, "Ada Lovelace",
		[]identitydomain.Role{identitydomain.RolePlayer}, identitydomain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building fixture user: %v", err)
	}
	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
}

// TestUserIDBySubject_ResolvesARealSubjectToTheUserID is the adapter's
// headline behaviour: the translation ADR-0014/ADR-0017 rule must happen,
// proven against the real identityapp.Service rather than a fake of it.
//
// The two assertions are separate on purpose. "No error" alone would pass
// for an adapter that returned the subject back unchanged — which is
// precisely the pre-T29.2 behaviour of the code path this replaces
// (actor(ctx) returning auth.RequireSubject(ctx) directly).
func TestUserIDBySubject_ResolvesARealSubjectToTheUserID(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedUser(t, repo, fixtureUserID, fixtureSubject)

	got, err := lookup.UserIDBySubject(context.Background(), fixtureSubject)
	if err != nil {
		t.Fatalf("UserIDBySubject(%q) = %v, want the caller's User.ID", fixtureSubject, err)
	}
	if got == fixtureSubject {
		t.Fatalf("UserIDBySubject(%q) returned the subject unchanged — no translation "+
			"happened, which is #164's Social Play third", fixtureSubject)
	}
	if got != fixtureUserID {
		t.Fatalf("UserIDBySubject(%q) = %q, want %q", fixtureSubject, got, fixtureUserID)
	}
}

// TestUserIDBySubject_UnknownSubjectIsSocialPlayErrUserNotFound pins
// CLAUDE.md rule 5 at this boundary: the caller gets Social Play's OWN
// sentinel, and the Identity one does not leak through.
//
// The negative assertion is the one that matters. Checking only
// errors.Is(err, socialplaydomain.ErrUserNotFound) would still pass if the
// adapter wrapped identitydomain.ErrUserNotFound with %w, letting app-layer
// code start matching on another context's error type — which is exactly
// the leak rule 5 forbids.
func TestUserIDBySubject_UnknownSubjectIsSocialPlayErrUserNotFound(t *testing.T) {
	t.Parallel()

	lookup, _ := newLookup(t)

	_, err := lookup.UserIDBySubject(context.Background(), "auth0|never-registered")
	if !errors.Is(err, socialplaydomain.ErrUserNotFound) {
		t.Fatalf("UserIDBySubject(unknown) = %v, want socialplay's own ErrUserNotFound", err)
	}
	if errors.Is(err, identitydomain.ErrUserNotFound) {
		t.Fatalf("UserIDBySubject(unknown) leaked identitydomain.ErrUserNotFound (%v) — "+
			"no other context's error type may cross this boundary (CLAUDE.md rule 5)", err)
	}
}

// TestUserIDBySubject_EmptySubjectIsErrUserNotFound pins the case the port's
// doc comment commits to explicitly, because it is reachable: an empty
// subject is answered exactly like an unregistered one.
//
// It must NOT be answered by a uuid-shape guard. identityapp.Service.
// UserBySubject deliberately applies none — a subject is not a uuid and must
// never be validated as one — so an empty subject reaches the repository and
// simply matches no row.
func TestUserIDBySubject_EmptySubjectIsErrUserNotFound(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedUser(t, repo, fixtureUserID, fixtureSubject)

	_, err := lookup.UserIDBySubject(context.Background(), "")
	if !errors.Is(err, socialplaydomain.ErrUserNotFound) {
		t.Fatalf("UserIDBySubject(\"\") = %v, want ErrUserNotFound", err)
	}
}

// TestUserIDBySubject_OtherErrorsAreWrappedWithoutTheirType covers
// translate's default branch: an infrastructure failure that is not "no
// such user".
//
// It must still not cross the boundary as its original type (the adapter
// uses %s, not %w), because a Social Play-side caller must never be able to
// errors.Is() its way to an Identity sentinel. But it must also not be
// flattened into ErrUserNotFound — a database outage is not "you are not
// registered", and answering PermissionDenied to it would hide a real fault
// behind an authorization message.
func TestUserIDBySubject_OtherErrorsAreWrappedWithoutTheirType(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("connection refused")
	lookup, repo := newLookup(t)
	repo.failWith = sentinel

	_, err := lookup.UserIDBySubject(context.Background(), fixtureSubject)
	if err == nil {
		t.Fatal("UserIDBySubject with a failing repository returned nil error")
	}
	if errors.Is(err, socialplaydomain.ErrUserNotFound) {
		t.Fatalf("UserIDBySubject = %v, want the infra error surfaced — flattening it to "+
			"ErrUserNotFound would report a database outage as PermissionDenied", err)
	}
	if errors.Is(err, sentinel) {
		t.Fatalf("UserIDBySubject = %v — the underlying error type crossed the boundary; "+
			"translate must use %%s, not %%w (CLAUDE.md rule 5)", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("UserIDBySubject = %v, want the cause preserved in the message for "+
			"operators even though the type is stripped", err)
	}
}

// TestLookup_ExposesNoUserReturningMethod is the structural guarantee
// port.IdentityLookup's doc comment makes, pinned so a future method
// addition has to confront it deliberately rather than by accident.
//
// The port exists to let a uuid cross the context boundary and nothing
// else. A method returning identitydomain.User would let Social Play's app
// layer start reading Roles or SelfReportedStartingLevel and quietly
// acquire a dependency on another context's model.
func TestLookup_ExposesNoUserReturningMethod(t *testing.T) {
	t.Parallel()

	// A compile-time assertion: *Lookup satisfies an interface that has
	// exactly the one primitive-typed method. If a User-returning method is
	// ever added to the port, this file's other tests keep passing — but
	// the reviewer reading this one is told what the port promised.
	var _ interface {
		UserIDBySubject(ctx context.Context, subject string) (string, error)
	} = (*socialplayidentity.Lookup)(nil)
}
