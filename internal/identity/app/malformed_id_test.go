// Boundary validation for caller-supplied User IDs.
//
// Same class of bug as internal/competitions/app/malformed_id_test.go and
// internal/facilities/app/malformed_id_test.go document in full: a Postgres
// adapter's mustUUID panics on a non-UUID, and grpc installs no recover() of
// its own, so an unvalidated caller-supplied ID reaching it can take the
// whole server process down. The guard makes a malformed ID resolve to the
// same domain.ErrUserNotFound an unknown ID already produces — a 404, never
// a 500 — on GetUser (T10.2's required scope, item 4) and, per this
// ticket's own instruction not to create a second instance of issue #97 on
// a write path, on UpdateSelfReportedLevel's id parameter too.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/identity/app"
	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

// malformedUserIDs mirrors the facilities/competitions corpus exactly. The
// braced and urn: entries are the ones that matter: github.com/google/uuid's
// Validate accepts both, pgtype.UUID.Scan rejects both, so a guard built on
// uuid.Validate would still have panicked on them.
var malformedUserIDs = []string{
	"",
	"not-a-uuid",
	"0",
	"'; DROP TABLE identity_users;--",
	"../../etc/passwd",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c",
	"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
	"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
	"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	" 6ba7b810-9dad-11d1-80b4-00c04fd430c8 ",
}

// --- CreateUser ----------------------------------------------------------
//
// CreateUser's id is caller-claimed via ActorUserID (app.CreateUserInput's
// doc comment), not server-generated the way every other aggregate's id in
// this codebase is — so, unlike Facility/Court/Payment/Competition, a
// malformed-but-nonempty CreateUser ActorUserID reaches the Postgres
// adapter's mustUUID unless guarded here too. This is the same bug class
// PR #89 fixed for read paths, reachable on this ticket's own new write
// path — not a second instance of issue #97 by omission.
func TestCreateUser_MalformedActorUserIDIsRejectedAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedUserIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo)

			_, err := svc.CreateUser(context.Background(), app.CreateUserInput{
				ActorUserID:               id,
				DisplayName:               "Ada Lovelace",
				Roles:                     validRoles(),
				SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
			})
			if !errors.Is(err, domain.ErrEmptyID) {
				t.Fatalf("CreateUser(actor_user_id=%q) error = %v, want %v", id, err, domain.ErrEmptyID)
			}
			if len(repo.users) != 0 {
				t.Errorf("malformed actor_user_id %q reached the repository (repo has %d users); it must be rejected at the boundary", id, len(repo.users))
			}
		})
	}
}

// --- GetUser -----------------------------------------------------------

func TestGetUser_MalformedIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedUserIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo)

			_, err := svc.GetUser(context.Background(), id)
			if !errors.Is(err, domain.ErrUserNotFound) {
				t.Fatalf("GetUser(%q) error = %v, want %v", id, err, domain.ErrUserNotFound)
			}
			if calls := repo.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed id %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestGetUser_WellFormedUnknownIDStillReachesRepository is the too-strict
// guard rail: a validator that rejects real IDs turns a loud crash into a
// silent 404 on every User, which is worse.
func TestGetUser_WellFormedUnknownIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	_, err := svc.GetUser(context.Background(), unknown)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("GetUser(%q) error = %v, want %v", unknown, err, domain.ErrUserNotFound)
	}
	if calls := repo.getByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown id did not reach the repository (%d calls)", calls)
	}
}

// TestGetUser_WellFormedRealIDStillResolves proves the guard doesn't reject
// a real, freshly-created User's own ID.
func TestGetUser_WellFormedRealIDStillResolves(t *testing.T) {
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
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := svc.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser(%q) on a real, freshly-created User: %v", created.ID, err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetUser returned %q, want %q", got.ID, created.ID)
	}
}

// --- UpdateSelfReportedLevel ---------------------------------------------
//
// UpdateSelfReportedLevel calls Repository.GetByID(id) first, before
// EnsureSelf, with the identical unguarded-caller-supplied-ID shape GetUser
// had — a malformed id must answer identically rather than reaching the
// Postgres adapter's mustUUID, which panics on non-UUID input.

func TestUpdateSelfReportedLevel_MalformedIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedUserIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo)

			_, err := svc.UpdateSelfReportedLevel(context.Background(), id, "attacker", domain.SelfReportedStartingLevel(4))
			if !errors.Is(err, domain.ErrUserNotFound) {
				t.Fatalf("UpdateSelfReportedLevel(%q) error = %v, want %v", id, err, domain.ErrUserNotFound)
			}
			if calls := repo.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed id %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestUpdateSelfReportedLevel_WellFormedUnknownIDStillReachesRepository is
// the too-strict guard rail for the write path, mirroring
// TestGetUser_WellFormedUnknownIDStillReachesRepository.
func TestUpdateSelfReportedLevel_WellFormedUnknownIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo)

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	_, err := svc.UpdateSelfReportedLevel(context.Background(), unknown, unknown, domain.SelfReportedStartingLevel(4))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("UpdateSelfReportedLevel(%q) error = %v, want %v", unknown, err, domain.ErrUserNotFound)
	}
	if calls := repo.getByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown id did not reach the repository (%d calls)", calls)
	}
}
