// T10.2 — object-level authorization regression test for
// UpdateSelfReportedLevel, run through the real gRPC handler (not just the
// domain-level unit tests internal/identity/domain/ensure_self_test.go and
// the app-level test internal/identity/app/service_test.go already have).
// This is the "does the guarantee survive the full stack" test T5.5 (Social
// Play), T6.7 (Payments), and T7.7 (Facilities) already established this
// pattern for.
//
// This is a handler-level test (real grpcapi.Handler + real app.Service +
// real domain, with an in-memory fake standing in for
// internal/identity/adapter/postgres) rather than a `-tags=integration`
// testcontainers-go test, for the identical two reasons T7.7's
// authz_regression_test.go documents: (1) the check under test lives
// entirely in domain.User.EnsureSelf/UpdateSelfReportedLevel, which
// port.Repository doesn't influence, so a real Postgres round trip would
// add infrastructure, not proof; (2) this environment has no Docker daemon
// reachable.
//
// Verified per CLAUDE.md rule 10 / T7.7's own verification pattern:
// temporarily commented out the EnsureSelf call inside
// domain.User.UpdateSelfReportedLevel (internal/identity/domain/user.go),
// confirmed TestUpdateSelfReportedLevel_RejectsMismatchedActor below FAILED
// (the attacker's call succeeded and the level flipped to the attacker's
// requested value), then restored the check and confirmed the test PASSES
// again. Stated in the PR description per the ticket's own instruction.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/identity/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/identity/app"
	"github.com/nhuthuynh/white-label/internal/identity/domain"
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	identityv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/identity/v1"
)

// registerOther puts a second, fully-registered User in the repository for
// its own verified subject, so a test can act as a real authenticated
// caller who simply is not the target User.
//
// It writes straight to the fake rather than going through the handler
// because the handler's stubIDs would mint the fixture id a second time and
// collide. The distinction matters for what the BOLA test proves: an
// authenticated caller who HAS an account but is not the owner must be
// refused for being the wrong person, not merely for being unregistered —
// those are two different code paths and only this one exercises
// domain.User.EnsureSelf.
func registerOther(t *testing.T, repo *fakeRepo, subject string) domain.User {
	t.Helper()
	u, err := domain.NewUser("7ba7b810-9dad-11d1-80b4-00c04fd430c8", subject, "Other Person",
		[]domain.Role{domain.RolePlayer}, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building the other user: %v", err)
	}
	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("registering the other user: %v", err)
	}
	return u
}

// --- in-memory port.Repository fake -----------------------------------
//
// Stands in for internal/identity/adapter/postgres for this test only.
// Implements the exact same port.Repository interface the real Postgres
// adapter does, so app.Service and grpcapi.Handler run unmodified, real
// production code — only the persistence boundary is faked. Mirrors
// internal/facilities/adapter/grpcapi/authz_regression_test.go's fakeRepo.
type fakeRepo struct {
	users map[string]domain.User
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: make(map[string]domain.User)}
}

// Create emulates both uniqueness rules identity_users enforces: the
// primary key on id, and — since T12.9 — the UNIQUE constraint on subject
// (db/migrations/0019_identity_subject.sql). The subject rule is the
// reachable one now that ids are server-minted.
func (r *fakeRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if _, exists := r.users[u.ID]; exists {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	for _, existing := range r.users {
		if existing.Subject == u.Subject {
			return domain.User{}, domain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *fakeRepo) GetBySubject(_ context.Context, subject string) (domain.User, error) {
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeRepo) UpdateSelfReportedLevel(_ context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

// mintedID is what stubIDs hands out — the server-minted User id T12.9
// introduced. Tests assert against it precisely because it is NOT anything
// the caller sent.
const mintedID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// T10.2's `fixtureUserID` const is gone. It existed so a test could name
// the id it was about to claim, which is precisely the capability T12.9
// removed — the id is minted by the server now, so tests read it back from
// the response (or assert against mintedID) instead of choosing it.

// fixtureSubject is the verified IdP subject of the seeded fixture user.
// Deliberately not uuid-shaped — see db/migrations/0019_identity_subject.sql.
const fixtureSubject = "auth0|ada"

// stubIDs is a deterministic port.IDGenerator. Identity gained this port in
// T12.9; before that a User's id came off the wire, which was the
// identity-squatting bug.
type stubIDs struct{ id string }

func (s stubIDs) NewID() string {
	if s.id == "" {
		return mintedID
	}
	return s.id
}

// sequentialIDs mints a distinct id per call, for tests where a rejection
// must be attributable to the subject rule rather than an id collision.
type sequentialIDs struct{ n int }

func (s *sequentialIDs) NewID() string {
	s.n++
	return fmt.Sprintf("6ba7b810-9dad-11d1-80b4-00c04fd430%02d", s.n)
}

// principalCtx builds a request context carrying a verified caller, the way
// auth.UnaryInterceptor does in production after a token verifies. Handler
// tests use it rather than standing up a gRPC server and minting a real
// token — which is exactly what auth.ContextWithPrincipal is exported for.
func principalCtx(subject string) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: subject})
}

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against the in-memory
// fakeRepo above.
func newTestHandler() (*grpcapi.Handler, *fakeRepo) {
	repo := newFakeRepo()
	svc := app.NewService(repo, stubIDs{})
	return grpcapi.NewHandler(svc), repo
}

// seedUser registers the fixture user as the verified owner of
// fixtureSubject. Since T12.9 the id is server-minted, so this no longer
// takes one — it returns the User the server created. The subject is what
// the caller now controls, and only by holding a token for it.
func seedUser(t *testing.T, h *grpcapi.Handler) *identityv1.User {
	t.Helper()
	resp, err := h.CreateUser(principalCtx(fixtureSubject), &identityv1.CreateUserRequest{
		DisplayName:               "Ada Lovelace",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	})
	if err != nil {
		t.Fatalf("failed to seed fixture user: %v", err)
	}
	return resp.GetUser()
}

// --- UpdateSelfReportedLevel: object-level (BOLA) regression --------------

// TestUpdateSelfReportedLevel_RejectsMismatchedActor is the ticket's
// required test: create a User, then attempt UpdateSelfReportedLevel as a
// different actor, through the real handler -> app -> domain path, and
// assert the request is rejected with the correctly mapped status — not a
// 500, not a silent success.
//
// T12.9 strengthened it rather than replacing it: the "different actor" is
// now a genuinely authenticated caller with a registered account of their
// own, and the request additionally LIES on the wire by naming the victim
// in actor_user_id. Under the pre-T12.9 handler that lie was the whole
// authorization input, so this call would have succeeded.
func TestUpdateSelfReportedLevel_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandler()

	user := seedUser(t, h)

	// The BOLA attempt. T12.9 changed HOW the attacker is identified but
	// not what is being proven: the attacker is now a caller holding a
	// valid token for their OWN subject (and their own registered User),
	// trying to update somebody else's level. The wire actor_user_id is
	// set to the victim's id as well, so this doubles as proof that the
	// handler resolves the actor from the principal and not from the
	// claim — if it read the wire field, this call would SUCCEED.
	registerOther(t, repo, "auth0|attacker")
	_, err := h.UpdateSelfReportedLevel(principalCtx("auth0|attacker"), &identityv1.UpdateSelfReportedLevelRequest{
		UserId:                    user.GetId(),
		ActorUserId:               user.GetId(),
		SelfReportedStartingLevel: 5,
	})
	if err == nil {
		t.Fatal("UpdateSelfReportedLevel(attacker) succeeded silently — a non-self actor was able to update another User's self-reported level (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("UpdateSelfReportedLevel(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("UpdateSelfReportedLevel(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("UpdateSelfReportedLevel(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove the
	// level was never changed, not just that an error came back on the
	// wire.
	if repo.users[user.GetId()].SelfReportedStartingLevel != 3 {
		t.Errorf("SelfReportedStartingLevel = %d after a rejected UpdateSelfReportedLevel, want unchanged 3 (the attacker's rejected attempt must not have any side effect)", repo.users[user.GetId()].SelfReportedStartingLevel)
	}
}

// TestUpdateSelfReportedLevel_AllowsOwningActor is the symmetric
// positive-path case: without it, TestUpdateSelfReportedLevel_
// RejectsMismatchedActor alone couldn't tell "the self check correctly
// rejects a mismatched actor" apart from "UpdateSelfReportedLevel is broken
// and rejects everyone" — this pins down that the real owning User's own
// call still succeeds through the same handler path.
func TestUpdateSelfReportedLevel_AllowsOwningActor(t *testing.T) {
	h, repo := newTestHandler()

	user := seedUser(t, h)

	resp, err := h.UpdateSelfReportedLevel(principalCtx(fixtureSubject), &identityv1.UpdateSelfReportedLevelRequest{
		UserId: user.GetId(),
		// Wire actor deliberately left EMPTY: the handler must not need
		// it, because it resolves the actor from the principal.
		SelfReportedStartingLevel: 5,
	})
	if err != nil {
		t.Fatalf("UpdateSelfReportedLevel(self) (the owning actor) should succeed, got: %v", err)
	}
	if resp.GetUser().GetSelfReportedStartingLevel() != 5 {
		t.Errorf("SelfReportedStartingLevel = %d, want 5", resp.GetUser().GetSelfReportedStartingLevel())
	}
	if repo.users[user.GetId()].SelfReportedStartingLevel != 5 {
		t.Errorf("repo SelfReportedStartingLevel = %d, want 5 after the owning actor's successful update", repo.users[user.GetId()].SelfReportedStartingLevel)
	}
}

// --- GetUser: malformed-ID boundary guard, through the real handler ------

// TestGetUser_MalformedIDIsMappedNotFound is the handler-level companion to
// internal/identity/app/malformed_id_test.go's app-level proof: through the
// real handler, a malformed id maps to the NotFound gRPC status (-> HTTP
// 404 via grpc-gateway), not Internal (-> HTTP 500).
func TestGetUser_MalformedIDIsMappedNotFound(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	_, err := h.GetUser(ctx, &identityv1.GetUserRequest{UserId: "not-a-uuid"})
	if err == nil {
		t.Fatal("GetUser(not-a-uuid) succeeded, want a NotFound error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("GetUser(not-a-uuid) returned a non-gRPC-status error: %v", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("GetUser(not-a-uuid) mapped to Internal (500-shaped) — want NotFound (404-shaped): %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("GetUser(not-a-uuid) status code = %v, want NotFound", st.Code())
	}
}

// --- CreateUser: self-elevation regression, through the real handler ------
//
// PR #106 review finding 2: an unchecked Roles field would let a caller
// mint themselves a brand-new, permanently-persisted ROLE_PLATFORM_ADMIN
// (or any other privileged role) out of nothing.
//
// T12.9 made CreateUser authenticated, and this test was KEPT and given a
// valid principal rather than retired, because authentication does not
// answer the question it asks. Holding a valid token proves who you are; it
// says nothing about whether you may appoint yourself an administrator. The
// caller below is fully authenticated and must still be refused. Verified non-vacuous per CLAUDE.md rule
// 10: temporarily removed the role-restriction loop in
// app.Service.CreateUser, confirmed
// TestCreateUser_RejectsSelfElevationToPlatformAdmin below FAILED (the
// request succeeded and a ROLE_PLATFORM_ADMIN User was persisted), then
// restored it and confirmed green again.

// TestCreateUser_RejectsSelfElevationToPlatformAdmin proves the real
// handler -> app -> domain path rejects a public CreateUser request
// claiming an elevated role, mapped to InvalidArgument (not Internal), and
// that no User is persisted.
func TestCreateUser_RejectsSelfElevationToPlatformAdmin(t *testing.T) {
	h, repo := newTestHandler()

	_, err := h.CreateUser(principalCtx("auth0|attacker"), &identityv1.CreateUserRequest{
		DisplayName:               "Attacker",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLATFORM_ADMIN},
		SelfReportedStartingLevel: 3,
	})
	if err == nil {
		t.Fatal("CreateUser(roles=[ROLE_PLATFORM_ADMIN]) succeeded silently — an authenticated caller was able to self-elevate to platform admin")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CreateUser(roles=[ROLE_PLATFORM_ADMIN]) returned a non-gRPC-status error: %v", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CreateUser(roles=[ROLE_PLATFORM_ADMIN]) mapped to Internal (500-shaped) — want InvalidArgument (400-shaped): %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("CreateUser(roles=[ROLE_PLATFORM_ADMIN]) status code = %v, want InvalidArgument", st.Code())
	}

	if len(repo.users) != 0 {
		t.Error("a User was persisted for a rejected self-elevation CreateUser call, want none")
	}
}

// TestCreateUser_AllowsPlayerRole is the symmetric positive-path case:
// without it, TestCreateUser_RejectsSelfElevationToPlatformAdmin alone
// couldn't tell "the role restriction correctly rejects an elevated role"
// apart from "CreateUser is broken and rejects everyone."
func TestCreateUser_AllowsPlayerRole(t *testing.T) {
	h, repo := newTestHandler()

	resp, err := h.CreateUser(principalCtx(fixtureSubject), &identityv1.CreateUserRequest{
		DisplayName:               "Ada Lovelace",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	})
	if err != nil {
		t.Fatalf("CreateUser(roles=[ROLE_PLAYER]) (the only self-assignable role) should succeed, got: %v", err)
	}
	if len(resp.GetUser().GetRoles()) != 1 || resp.GetUser().GetRoles()[0] != identityv1.Role_ROLE_PLAYER {
		t.Errorf("Roles = %v, want [ROLE_PLAYER]", resp.GetUser().GetRoles())
	}
	if _, exists := repo.users[mintedID]; !exists {
		t.Error("User was not persisted after a successful CreateUser")
	}
}
