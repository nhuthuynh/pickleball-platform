package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/nhuthuynh/white-label/internal/identity/app"
	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

// inMemoryRepo is a minimal port.Repository fake, mirroring
// internal/facilities/app's inMemoryRepo — no persistence, no concurrency
// guarantees. It exists purely to let the app-layer orchestration be tested
// without a database.
type inMemoryRepo struct {
	users map[string]domain.User

	// getByIDCalls counts real invocations of GetByID, so a test can prove a
	// malformed-shape id never reaches the repository at all (T10.2's
	// GetUser/UpdateSelfReportedLevel guards) — the in-memory map here can't
	// reproduce Postgres rejecting a non-UUID against a `uuid` column, so
	// the app-layer shape-check short-circuit can only be proven by
	// observing the call itself never happens, not by the return value
	// alone. atomic.Int64 mirrors internal/facilities/app's inMemoryRepo.
	getByIDCalls atomic.Int64
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{users: make(map[string]domain.User)}
}

// Create emulates BOTH uniqueness rules the real table enforces: the
// primary key on id, and — since T12.9 — the UNIQUE constraint on subject
// (db/migrations/0019_identity_subject.sql). The subject rule is the one
// that matters now: id is server-minted, so a PK collision is unreachable,
// while a repeat registration for an already-registered subject is a real,
// reachable case. A fake that only checked the id would let
// TestCreateUser_DuplicateSubjectRejected pass vacuously.
func (r *inMemoryRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
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

func (r *inMemoryRepo) GetBySubject(_ context.Context, subject string) (domain.User, error) {
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *inMemoryRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	r.getByIDCalls.Add(1)
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *inMemoryRepo) UpdateSelfReportedLevel(_ context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

func validRoles() []domain.Role {
	return []domain.Role{domain.RolePlayer}
}

// fixtureUserID is what stubIDs mints. It is a uuid because User.ID still
// is one; what changed in T12.9 is only WHO chooses it — the server, not
// the caller.
const fixtureUserID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// fixtureSubject is deliberately NOT uuid-shaped. An IdP subject is an
// arbitrary provider-specific string (`auth0|abc123`), which is exactly why
// it cannot be identity_users.id and gets its own column — a fixture that
// used a uuid here would hide any code that wrongly validated a subject as
// one.
const fixtureSubject = "auth0|ada"

// stubIDs is a deterministic port.IDGenerator, mirroring the fake
// internal/facilities/app's tests use. Identity gained this port in T12.9;
// before that its User IDs came off the wire, which was the bug.
type stubIDs struct{ id string }

func (s stubIDs) NewID() string {
	if s.id == "" {
		return fixtureUserID
	}
	return s.id
}

// countingIDs mints a DISTINCT id per call. Tests about subject uniqueness
// use it so that a rejection can only be attributed to the subject rule —
// with stubIDs' fixed id, a second Create would collide on the primary key
// and the test would pass for the wrong reason.
type countingIDs struct{ n int }

func (c *countingIDs) NewID() string {
	c.n++
	return fmt.Sprintf("6ba7b810-9dad-11d1-80b4-00c04fd430%02d", c.n)
}

func TestCreateUser_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	u, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.ID != fixtureUserID {
		t.Fatalf("ID = %q, want the server-minted %q", u.ID, fixtureUserID)
	}
	if u.Subject != fixtureSubject {
		t.Fatalf("Subject = %q, want the verified %q", u.Subject, fixtureSubject)
	}
	if u.DisplayName != "Ada Lovelace" {
		t.Fatalf("DisplayName = %q, want Ada Lovelace", u.DisplayName)
	}
}

func TestCreateUser_ValidationRejectedBeforeTouchingRepo(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:     fixtureSubject,
		DisplayName: "",
		Roles:       validRoles(),
	})
	if !errors.Is(err, domain.ErrEmptyDisplayName) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyDisplayName)
	}
	if len(repo.users) != 0 {
		t.Fatalf("invalid user must not be persisted, repo has %d entries", len(repo.users))
	}
}

// TestCreateUser_EmptySubjectRejected replaces T10.2's
// TestCreateUser_EmptyActorUserIDRejected, which tested a field that no
// longer exists. The ID cannot be empty any more — it is minted here — so
// the remaining "who is this?" failure is an absent verified subject. In
// production the handler rejects that with Unauthenticated long before this
// point; the domain check is kept as defence in depth for any future caller
// that reaches the app layer directly.
func TestCreateUser_EmptySubjectRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:     "",
		DisplayName: "Ada Lovelace",
		Roles:       validRoles(),
	})
	if !errors.Is(err, domain.ErrEmptySubject) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptySubject)
	}
	if len(repo.users) != 0 {
		t.Fatalf("a subject-less user must not be persisted, repo has %d entries", len(repo.users))
	}
}

// TestCreateUser_DuplicateSubjectRejected pins T12.9's
// idempotent-or-rejected decision at the app layer: REJECTED. See
// domain.ErrUserAlreadyExists's doc comment for the reasoning (the second
// call carries its own display name and level, so replaying the first would
// answer a different request than the one that was made).
func TestCreateUser_DuplicateSubjectRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	// Distinct IDs per call, so the rejection can only come from the
	// subject rule and never from an id collision.
	svc := app.NewService(repo, &countingIDs{})
	ctx := context.Background()

	first := app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	}
	if _, err := svc.CreateUser(ctx, first); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	second := first
	second.DisplayName = "Someone Else"
	second.SelfReportedStartingLevel = domain.SelfReportedStartingLevel(5)
	if _, err := svc.CreateUser(ctx, second); !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("got err %v, want %v", err, domain.ErrUserAlreadyExists)
	}
	if len(repo.users) != 1 {
		t.Fatalf("repo has %d users, want exactly 1", len(repo.users))
	}
}

// TestCreateUser_DistinctSubjectsBothRegister is the symmetric positive
// case: without it, the rejection test above could not tell "duplicate
// subjects are rejected" apart from "the second CreateUser always fails".
func TestCreateUser_DistinctSubjectsBothRegister(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &countingIDs{})
	ctx := context.Background()

	for _, subject := range []string{"auth0|ada", "google-oauth2|10769150350006150715"} {
		if _, err := svc.CreateUser(ctx, app.CreateUserInput{
			Subject:                   subject,
			DisplayName:               "Someone",
			Roles:                     validRoles(),
			SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
		}); err != nil {
			t.Fatalf("CreateUser(subject=%q): %v", subject, err)
		}
	}
	if len(repo.users) != 2 {
		t.Fatalf("repo has %d users, want 2", len(repo.users))
	}
}

// TestUserBySubject resolves a verified subject back to its User — the
// lookup the grpcapi boundary uses to turn an auth.Principal into the actor
// ID the domain understands, without app or domain importing
// internal/platform/auth (A11 Ruling 3).
func TestUserBySubject(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := svc.UserBySubject(ctx, fixtureSubject)
	if err != nil {
		t.Fatalf("UserBySubject(%q): %v", fixtureSubject, err)
	}
	if got.ID != created.ID {
		t.Fatalf("UserBySubject returned id %q, want %q", got.ID, created.ID)
	}

	for _, unknown := range []string{"auth0|nobody", ""} {
		if _, err := svc.UserBySubject(ctx, unknown); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("UserBySubject(%q) err = %v, want %v", unknown, err, domain.ErrUserNotFound)
		}
	}
}

// TestCreateUser_NonPlayerRoleRejected is the PR #106 review fix: the
// public, unauthenticated CreateUser path must not let a caller
// self-assign an elevated role (e.g. ROLE_PLATFORM_ADMIN) — it only ever
// accepts RolePlayer. This is a distinct failure mode from
// TestCreateUser_InvalidRoleRejected below (an unrecognized enum value):
// ROLE_HOST_ORGANISER here is a perfectly valid, recognized Role — it's
// just not self-assignable via this RPC.
func TestCreateUser_NonPlayerRoleRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     []domain.Role{domain.RolePlatformAdmin},
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if !errors.Is(err, domain.ErrRoleNotSelfAssignable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrRoleNotSelfAssignable)
	}
	if len(repo.users) != 0 {
		t.Fatalf("a rejected non-player role must not be persisted, repo has %d entries", len(repo.users))
	}
}

// TestCreateUser_PlayerRoleAmongOthersRejected proves the check applies to
// every element, not just a single-role request: RolePlayer alongside
// RoleHostOrganiser is still rejected, not silently downgraded to just
// RolePlayer.
func TestCreateUser_PlayerRoleAmongOthersRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     []domain.Role{domain.RolePlayer, domain.RoleHostOrganiser},
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if !errors.Is(err, domain.ErrRoleNotSelfAssignable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrRoleNotSelfAssignable)
	}
}

func TestCreateUser_InvalidRoleRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     []domain.Role{domain.Role("coach")},
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidRole)
	}
}

func TestCreateUser_InvalidLevelRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(9),
	})
	if !errors.Is(err, domain.ErrInvalidSelfReportedStartingLevel) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidSelfReportedStartingLevel)
	}
}

func TestGetUser_UnknownReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.GetUser(ctx, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrUserNotFound)
	}
}

func TestGetUser_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got, err := svc.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("ID = %q, want %q", got.ID, created.ID)
	}
}

// TestUpdateSelfReportedLevel_Valid proves the app-level orchestration:
// fetch, domain check + mutation, persist, return the updated User.
func TestUpdateSelfReportedLevel_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	updated, err := svc.UpdateSelfReportedLevel(ctx, created.ID, created.ID, domain.SelfReportedStartingLevel(5))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if updated.SelfReportedStartingLevel != 5 {
		t.Fatalf("SelfReportedStartingLevel = %d, want 5", updated.SelfReportedStartingLevel)
	}

	stored, err := svc.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if stored.SelfReportedStartingLevel != 5 {
		t.Fatalf("persisted SelfReportedStartingLevel = %d, want 5", stored.SelfReportedStartingLevel)
	}
}

// TestUpdateSelfReportedLevel_RejectsMismatchedActor is T10.2's app-level
// proof that a non-matching actor is rejected before any persistence write
// — mirrors internal/facilities/app's TestAddCourt_RejectsNonOwner.
func TestUpdateSelfReportedLevel_RejectsMismatchedActor(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		Subject:                   fixtureSubject,
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, err = svc.UpdateSelfReportedLevel(ctx, created.ID, "attacker", domain.SelfReportedStartingLevel(5))
	if !errors.Is(err, domain.ErrNotSelf) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotSelf)
	}

	stored, err := svc.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if stored.SelfReportedStartingLevel != 3 {
		t.Fatalf("SelfReportedStartingLevel = %d after a rejected update, want unchanged 3", stored.SelfReportedStartingLevel)
	}
}

func TestUpdateSelfReportedLevel_UnknownUserReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIDs{})
	ctx := context.Background()

	_, err := svc.UpdateSelfReportedLevel(ctx, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", domain.SelfReportedStartingLevel(5))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrUserNotFound)
	}
}
