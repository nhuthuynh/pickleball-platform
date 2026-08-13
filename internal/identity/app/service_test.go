package app_test

import (
	"context"
	"errors"
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

func (r *inMemoryRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if _, exists := r.users[u.ID]; exists {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	r.users[u.ID] = u
	return u, nil
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

func TestCreateUser_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)
	ctx := context.Background()

	u, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		DisplayName:               "Ada Lovelace",
		Roles:                     validRoles(),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.ID != "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		t.Fatalf("ID = %q, want the claimed actor_user_id", u.ID)
	}
	if u.DisplayName != "Ada Lovelace" {
		t.Fatalf("DisplayName = %q, want Ada Lovelace", u.DisplayName)
	}
}

func TestCreateUser_ValidationRejectedBeforeTouchingRepo(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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

func TestCreateUser_EmptyActorUserIDRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID: "",
		DisplayName: "Ada Lovelace",
		Roles:       validRoles(),
	})
	if !errors.Is(err, domain.ErrEmptyID) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyID)
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.GetUser(ctx, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrUserNotFound)
	}
}

func TestGetUser_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
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
	svc := app.NewService(repo)
	ctx := context.Background()

	_, err := svc.UpdateSelfReportedLevel(ctx, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", domain.SelfReportedStartingLevel(5))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrUserNotFound)
	}
}
