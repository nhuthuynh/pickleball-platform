// Boundary validation for caller-supplied User IDs on the READ/UPDATE
// paths. (The create path no longer takes one — see the CreateUser section
// below.)
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
// T10.2's TestCreateUser_MalformedActorUserIDIsRejectedAndNeverReaches-
// Repository lived here. It is DELETED rather than adapted, because the
// thing it guarded no longer exists: CreateUser had a caller-claimed id
// (ActorUserID) that reached the Postgres adapter's mustUUID unless the app
// layer screened it, and T12.9 removed that field entirely — the id is
// server-minted now, exactly like Facility/Court/Payment/Competition, so
// there is no caller input on this path to malform. Keeping the old test
// would have meant keeping the field it tested.
//
// What replaces it is the inverse assertion: that the minted id really is
// well-formed, and that a subject — which is NOT a uuid and must never be
// screened as one — does not get caught by the uuid guards this file is
// about.

// TestCreateUser_MintsAWellFormedIDRegardlessOfSubjectShape proves the
// adapter's mustUUID cannot be reached with a bad id from this path, and
// that a realistic non-uuid IdP subject flows through untouched. If a
// future change ever validated the subject with uuidShape, every case here
// would fail — which is the point, since real subjects look like these.
func TestCreateUser_MintsAWellFormedIDRegardlessOfSubjectShape(t *testing.T) {
	t.Parallel()

	subjects := []string{
		"auth0|abc123",
		"google-oauth2|10769150350006150715",
		"CiRlM2E5ZDI0Yi0wMDAwLTQwMDAtOGYwMC1hYmNkZWYwMTIzNDUSBWxvY2Fs",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8", // a subject that happens to look like a uuid is still just a string
	}

	for _, subject := range subjects {
		t.Run(subject, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, stubIDs{})

			created, err := svc.CreateUser(context.Background(), app.CreateUserInput{
				Subject:                   subject,
				DisplayName:               "Ada Lovelace",
				Roles:                     validRoles(),
				SelfReportedStartingLevel: domain.SelfReportedStartingLevel(3),
			})
			if err != nil {
				t.Fatalf("CreateUser(subject=%q): %v", subject, err)
			}
			if created.ID != fixtureUserID {
				t.Fatalf("ID = %q, want the server-minted %q", created.ID, fixtureUserID)
			}
			if created.Subject != subject {
				t.Fatalf("Subject = %q, want %q unchanged", created.Subject, subject)
			}

			// The minted id must survive the same guard the read paths
			// apply, or GetUser could never find what CreateUser wrote.
			if _, err := svc.GetUser(context.Background(), created.ID); err != nil {
				t.Fatalf("GetUser on the freshly-minted id %q: %v", created.ID, err)
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
			svc := app.NewService(repo, stubIDs{})

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
	svc := app.NewService(repo, stubIDs{})

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
			svc := app.NewService(repo, stubIDs{})

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
	svc := app.NewService(repo, stubIDs{})

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	_, err := svc.UpdateSelfReportedLevel(context.Background(), unknown, unknown, domain.SelfReportedStartingLevel(4))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("UpdateSelfReportedLevel(%q) error = %v, want %v", unknown, err, domain.ErrUserNotFound)
	}
	if calls := repo.getByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown id did not reach the repository (%d calls)", calls)
	}
}
