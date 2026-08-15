// T15.3 — app-layer tests for the durable Competition-Admin store (#168,
// partial fix). The mirror of Social Play's T14.4 coverage, one context over.
//
// What these prove that the domain tests cannot: the ORDER the Service runs
// its steps in (boundary guard, existence, current assignments, domain rule),
// that the Host-only rule survives the trip through the app layer, and that
// the read half this ticket is required to ship even though it does not
// consume it (T15.4/T15.5 do) actually works.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// fakeCompetitionAdminRepository is an in-memory
// port.CompetitionAdminRepository.
//
// It reproduces the real Postgres adapter's contracts that the app layer
// depends on, and only those: the composite-PK duplicate rejection
// (domain.ErrAlreadyCompetitionAdmin), the revoke-removed-nothing answer
// (domain.ErrCompetitionAdminNotFound), and the empty-slice-for-an-unknown-
// Competition read. assignOrder keeps the list deterministic, since Go map
// iteration order is unspecified and the real query has an explicit ORDER BY.
type fakeCompetitionAdminRepository struct {
	admins map[string][]domain.CompetitionAdmin

	// listCalls counts ListCompetitionAdmins invocations so a test can prove
	// the malformed-ID and unknown-Competition guards short-circuit BEFORE the
	// store is touched — an ordering fact no return value can demonstrate.
	listCalls int
	// assignCalls counts Assign invocations, so a test can prove an
	// unauthorized caller never reaches the write at all.
	assignCalls int
	// revokeCalls does the same for Revoke.
	revokeCalls int
}

func newFakeCompetitionAdminRepository() *fakeCompetitionAdminRepository {
	return &fakeCompetitionAdminRepository{admins: make(map[string][]domain.CompetitionAdmin)}
}

func (r *fakeCompetitionAdminRepository) Assign(_ context.Context, a domain.CompetitionAdmin) (domain.CompetitionAdmin, error) {
	r.assignCalls++
	// The composite primary key, reproduced: the authoritative guard behind
	// domain.AssignCompetitionAdmin's fail-fast pre-check (CLAUDE.md rule 4).
	if domain.HasCompetitionAdmin(r.admins[a.CompetitionID], a.UserID) {
		return domain.CompetitionAdmin{}, domain.ErrAlreadyCompetitionAdmin
	}
	r.admins[a.CompetitionID] = append(r.admins[a.CompetitionID], a)
	return a, nil
}

func (r *fakeCompetitionAdminRepository) Revoke(_ context.Context, competitionID, userID string) error {
	r.revokeCalls++
	existing := r.admins[competitionID]
	for i, a := range existing {
		if a.UserID == userID {
			r.admins[competitionID] = append(append([]domain.CompetitionAdmin{}, existing[:i]...), existing[i+1:]...)
			return nil
		}
	}
	return domain.ErrCompetitionAdminNotFound
}

func (r *fakeCompetitionAdminRepository) ListCompetitionAdmins(_ context.Context, competitionID string) ([]domain.CompetitionAdmin, error) {
	r.listCalls++
	out := make([]domain.CompetitionAdmin, len(r.admins[competitionID]))
	copy(out, r.admins[competitionID])
	return out, nil
}

// newAdminTestService wires a Service with the Competition-Admin store
// attached. Deliberately its own constructor rather than a sixth parameter on
// newTestService: every existing test in this package builds a Service that
// has no business holding an admin repository, and threading a nil through
// them all would make the dependency look optional when it is not.
func newAdminTestService(repo *fakeRepository, admins *fakeCompetitionAdminRepository) *app.Service {
	return app.NewService(app.ServiceOptions{
		Competitions:      repo,
		IDs:               &sequentialIDs{},
		Reservation:       newFakeReservation(),
		Facilities:        newFakeFacilityLookup(),
		ShareTokens:       &fakeShareTokens{},
		CompetitionAdmins: admins,
	})
}

const (
	appAdminHostID   = "host-subject"
	appAdminUserID   = "admin-subject"
	appAdminStranger = "stranger-subject"
	// appAdminCompetitionID is uuid-shaped on purpose: app.Service's
	// uuidShape guard rejects anything else before the repository is reached,
	// so a fixture with a "comp-1"-style id would make every test below pass
	// for the wrong reason (see malformed_id_test.go's history).
	appAdminCompetitionID = "11111111-1111-4111-8111-111111111111"
)

// seedAdminCompetition puts a scheduled Competition hosted by appAdminHostID
// into the fake repository. It writes the aggregate directly rather than going
// through ScheduleCompetition so the fixture's HostID is exactly the subject
// under test and does not depend on the scheduling path.
func seedAdminCompetition(t *testing.T, repo *fakeRepository) domain.Competition {
	t.Helper()

	c := domain.Competition{
		ID:       appAdminCompetitionID,
		HostID:   appAdminHostID,
		Name:     "Autumn Open",
		Capacity: 8,
		Status:   domain.StatusScheduled,
	}
	if _, err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("seeding competition: %v", err)
	}
	return c
}

func TestAssignCompetitionAdmin(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	admins := newFakeCompetitionAdminRepository()
	svc := newAdminTestService(repo, admins)
	seedAdminCompetition(t, repo)

	got, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
		CompetitionID: appAdminCompetitionID,
		ActorUserID:   appAdminHostID,
		AdminUserID:   appAdminUserID,
	})
	if err != nil {
		t.Fatalf("AssignCompetitionAdmin() unexpected error: %v", err)
	}
	if got.CompetitionID != appAdminCompetitionID || got.UserID != appAdminUserID {
		t.Fatalf("assignment = %+v, want competition %q / user %q", got, appAdminCompetitionID, appAdminUserID)
	}
	if got.AssignedBy != appAdminHostID {
		t.Errorf("AssignedBy = %q, want %q — the assigner is the verified actor, never the wire", got.AssignedBy, appAdminHostID)
	}
	if got.AssignedAt.IsZero() {
		t.Error("AssignedAt is zero — the app layer supplies the clock (time.Now()), keeping the domain free of ambient clock reads")
	}

	// The store actually holds it: the write is the point of the ticket, and a
	// method that validated correctly but persisted nothing would satisfy every
	// assertion above.
	stored, err := svc.ListCompetitionAdmins(context.Background(), appAdminCompetitionID)
	if err != nil {
		t.Fatalf("ListCompetitionAdmins() unexpected error: %v", err)
	}
	if len(stored) != 1 || !domain.HasCompetitionAdmin(stored, appAdminUserID) {
		t.Fatalf("stored admins = %+v, want exactly the one just assigned", stored)
	}
}

// TestAssignCompetitionAdmin_Rejections walks every refusal the app layer can
// produce, and asserts on assignCalls for the authorization rows: the point is
// not merely that an unauthorized caller gets an error, but that the write is
// never attempted.
func TestAssignCompetitionAdmin_Rejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		competitionID string
		actorUserID   string
		adminUserID   string
		preAssign     string
		wantErr       error
		why           string
	}{
		{
			name:          "a stranger assigns",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminStranger,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
		},
		{
			name:          "an assigned admin appoints another admin",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminUserID,
			adminUserID:   "second-admin-subject",
			preAssign:     appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
			why:           "#168's headline rule, surviving the trip through the app layer",
		},
		{
			name:          "an unidentified caller assigns",
			competitionID: appAdminCompetitionID,
			actorUserID:   "",
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
		},
		{
			name:          "the host assigns a blank user id",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminHostID,
			adminUserID:   "",
			wantErr:       domain.ErrEmptyCompetitionAdminUserID,
		},
		{
			name:          "the host assigns themselves",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminHostID,
			wantErr:       domain.ErrHostCannotBeCompetitionAdmin,
		},
		{
			name:          "the host re-assigns an existing admin",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminUserID,
			preAssign:     appAdminUserID,
			wantErr:       domain.ErrAlreadyCompetitionAdmin,
		},
		{
			name:          "an unknown competition",
			competitionID: "99999999-9999-4999-8999-999999999999",
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrCompetitionNotFound,
			why:           "a write has no 'nothing to show you' outcome — unlike ListEntriesForCompetition's empty roster",
		},
		{
			name:          "a malformed competition id",
			competitionID: "not-a-uuid",
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrCompetitionNotFound,
			why:           "T10.7's boundary guard: mustUUID panics on non-uuid input, so it must never be reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			admins := newFakeCompetitionAdminRepository()
			svc := newAdminTestService(repo, admins)
			seedAdminCompetition(t, repo)

			if tt.preAssign != "" {
				if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
					CompetitionID: appAdminCompetitionID,
					ActorUserID:   appAdminHostID,
					AdminUserID:   tt.preAssign,
				}); err != nil {
					t.Fatalf("seeding assignment: %v", err)
				}
			}
			before := admins.assignCalls

			_, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
				CompetitionID: tt.competitionID,
				ActorUserID:   tt.actorUserID,
				AdminUserID:   tt.adminUserID,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("AssignCompetitionAdmin() error = %v, want %v — %s", err, tt.wantErr, tt.why)
			}
			if admins.assignCalls != before {
				t.Errorf("Assign was called %d time(s) despite the rejection — a refused request must never reach the store",
					admins.assignCalls-before)
			}
		})
	}
}

// TestAssignCompetitionAdmin_MalformedIDNeverReachesTheStore pins the ordering
// fact the table above can only imply: the uuidShape guard runs before ANY
// port call. The real Postgres adapter's mustUUID panics on a non-uuid, and
// grpc installs no recover() of its own, so "the guard runs first" is a
// crash-avoidance property, not a tidiness one (T10.7, #97).
func TestAssignCompetitionAdmin_MalformedIDNeverReachesTheStore(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	admins := newFakeCompetitionAdminRepository()
	svc := newAdminTestService(repo, admins)
	seedAdminCompetition(t, repo)

	if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
		CompetitionID: "not-a-uuid",
		ActorUserID:   appAdminHostID,
		AdminUserID:   appAdminUserID,
	}); !errors.Is(err, domain.ErrCompetitionNotFound) {
		t.Fatalf("error = %v, want ErrCompetitionNotFound", err)
	}
	if admins.listCalls != 0 || admins.assignCalls != 0 {
		t.Fatalf("store was touched (list=%d assign=%d) for a malformed id, want 0/0",
			admins.listCalls, admins.assignCalls)
	}
}

func TestRevokeCompetitionAdmin(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	admins := newFakeCompetitionAdminRepository()
	svc := newAdminTestService(repo, admins)
	seedAdminCompetition(t, repo)

	if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
		CompetitionID: appAdminCompetitionID,
		ActorUserID:   appAdminHostID,
		AdminUserID:   appAdminUserID,
	}); err != nil {
		t.Fatalf("seeding assignment: %v", err)
	}

	if err := svc.RevokeCompetitionAdmin(context.Background(), app.RevokeCompetitionAdminInput{
		CompetitionID: appAdminCompetitionID,
		ActorUserID:   appAdminHostID,
		AdminUserID:   appAdminUserID,
	}); err != nil {
		t.Fatalf("RevokeCompetitionAdmin() unexpected error: %v", err)
	}

	// Revoking must actually remove the authority, not merely report success —
	// the T3 lesson (prove the slot is freed, not that a field flipped).
	remaining, err := svc.ListCompetitionAdmins(context.Background(), appAdminCompetitionID)
	if err != nil {
		t.Fatalf("ListCompetitionAdmins() unexpected error: %v", err)
	}
	if domain.HasCompetitionAdmin(remaining, appAdminUserID) {
		t.Fatalf("still an admin after revoke: %+v", remaining)
	}
}

func TestRevokeCompetitionAdmin_Rejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		competitionID string
		actorUserID   string
		adminUserID   string
		wantErr       error
		wantRevoke    bool
		why           string
	}{
		{
			name:          "a stranger revokes",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminStranger,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
		},
		{
			name:          "the assigned admin revokes themselves",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminUserID,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
			why:           "granting and withdrawing are the same authority; an admin is refused from both ends",
		},
		{
			name:          "an unidentified caller revokes",
			competitionID: appAdminCompetitionID,
			actorUserID:   "",
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrNotCompetitionHost,
		},
		{
			name:          "the host revokes someone holding no assignment",
			competitionID: appAdminCompetitionID,
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminStranger,
			wantErr:       domain.ErrCompetitionAdminNotFound,
			wantRevoke:    true,
			why:           "not a silent success — a Host asserting who holds authority must not be confirmed by a no-op",
		},
		{
			name:          "an unknown competition",
			competitionID: "99999999-9999-4999-8999-999999999999",
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrCompetitionNotFound,
		},
		{
			name:          "a malformed competition id",
			competitionID: "not-a-uuid",
			actorUserID:   appAdminHostID,
			adminUserID:   appAdminUserID,
			wantErr:       domain.ErrCompetitionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			admins := newFakeCompetitionAdminRepository()
			svc := newAdminTestService(repo, admins)
			seedAdminCompetition(t, repo)

			if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
				CompetitionID: appAdminCompetitionID,
				ActorUserID:   appAdminHostID,
				AdminUserID:   appAdminUserID,
			}); err != nil {
				t.Fatalf("seeding assignment: %v", err)
			}
			admins.revokeCalls = 0

			err := svc.RevokeCompetitionAdmin(context.Background(), app.RevokeCompetitionAdminInput{
				CompetitionID: tt.competitionID,
				ActorUserID:   tt.actorUserID,
				AdminUserID:   tt.adminUserID,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RevokeCompetitionAdmin() error = %v, want %v — %s", err, tt.wantErr, tt.why)
			}
			// An unauthorized caller must not reach the store: otherwise the
			// difference between "not assigned" and "revoked" would be an
			// enumeration oracle for a Competition's admin list.
			if reached := admins.revokeCalls > 0; reached != tt.wantRevoke {
				t.Errorf("store reached = %v, want %v — the authorization check runs before the store is consulted", reached, tt.wantRevoke)
			}
		})
	}
}

// TestListCompetitionAdmins is the read half T15.3 must ship even though it
// consumes none of it: T15.4 resolves the roster read's entitled set from it,
// and T15.5 calls it from the Payments context. Shipping it here is what stops
// a later ticket having to widen a merged interface (sprint plan §A12 GAP A).
func TestListCompetitionAdmins(t *testing.T) {
	t.Parallel()

	t.Run("returns every assignment", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepository()
		admins := newFakeCompetitionAdminRepository()
		svc := newAdminTestService(repo, admins)
		seedAdminCompetition(t, repo)

		for _, u := range []string{appAdminUserID, "second-admin-subject"} {
			if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
				CompetitionID: appAdminCompetitionID,
				ActorUserID:   appAdminHostID,
				AdminUserID:   u,
			}); err != nil {
				t.Fatalf("seeding %q: %v", u, err)
			}
		}

		got, err := svc.ListCompetitionAdmins(context.Background(), appAdminCompetitionID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d admins, want 2: %+v", len(got), got)
		}
		for _, u := range []string{appAdminUserID, "second-admin-subject"} {
			if !domain.HasCompetitionAdmin(got, u) {
				t.Errorf("%q missing from the resolved set %+v", u, got)
			}
		}
	})

	t.Run("a competition with no admins is an empty set, not an error", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepository()
		svc := newAdminTestService(repo, newFakeCompetitionAdminRepository())
		seedAdminCompetition(t, repo)

		got, err := svc.ListCompetitionAdmins(context.Background(), appAdminCompetitionID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %+v, want an empty set — most Competitions have no admins", got)
		}
	})

	// An unknown or malformed Competition is ErrCompetitionNotFound rather
	// than an empty set: an empty admin list is a real and common state, so
	// returning it for a Competition that does not exist would make the two
	// indistinguishable to the one kind of caller — an authorization decision
	// — that must not confuse them.
	t.Run("an unknown competition is not found", func(t *testing.T) {
		t.Parallel()

		svc := newAdminTestService(newFakeRepository(), newFakeCompetitionAdminRepository())
		if _, err := svc.ListCompetitionAdmins(context.Background(), "99999999-9999-4999-8999-999999999999"); !errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Fatalf("error = %v, want ErrCompetitionNotFound", err)
		}
	})

	t.Run("a malformed competition id is not found and never reaches the store", func(t *testing.T) {
		t.Parallel()

		admins := newFakeCompetitionAdminRepository()
		svc := newAdminTestService(newFakeRepository(), admins)

		if _, err := svc.ListCompetitionAdmins(context.Background(), "not-a-uuid"); !errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Fatalf("error = %v, want ErrCompetitionNotFound", err)
		}
		if admins.listCalls != 0 {
			t.Fatalf("store was read %d time(s) for a malformed id, want 0", admins.listCalls)
		}
	})
}
