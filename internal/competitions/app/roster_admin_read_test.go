// T15.4 — the roster read is Host **or** assigned Competition Admin (closes
// #147, and the consumption half of #168 that #168 itself stays open for —
// see the PR body for why #168 does not close here).
//
// The exact mirror of internal/socialplay/app/roster_admin_read_test.go,
// against Competitions' twin RPC.
//
// # What changed, and why it could not have changed earlier
//
// T13.6 shipped ListEntriesForCompetition as Host-only and said so in its own
// doc comment: an assigned Competition Admin *should* be able to read a
// roster, but the only expression of "assigned Competition Admin" this
// codebase had was a caller-supplied field on Payments' offline-recording
// requests — a repeated string the caller supplied. A rule honouring it would
// have admitted any caller willing to type their own id, which is strictly
// worse than Host-only because it reads as authorization while enforcing
// nothing. T15.3 built the durable store (domain.CompetitionAdmin,
// port.CompetitionAdminRepository, the competition_admins table), so the
// entitled set is now a server fact and the rule can finally widen.
//
// # Every test in this file drives the REAL store
//
// There is no "pretend this user is an admin" seam anywhere below. The only
// way a subject becomes an admin here is Service.AssignCompetitionAdmin — the
// real Host-only write path, through the real domain rule — and the only way
// the read learns about it is Service.ListEntriesForCompetition resolving the
// set through port.CompetitionAdminRepository.ListCompetitionAdmins. A test
// that stubbed the membership answer would prove the widened check calls
// *something*, not that it calls the store; that distinction is the entire
// content of this ticket.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

const (
	rosterHost               = "roster-host-subject"
	rosterAdmin              = "roster-admin-subject"
	rosterEntrant            = "roster-entrant-subject"
	rosterStranger           = "roster-stranger-subject"
	rosterCompetitionID      = "22222222-2222-4222-8222-222222222222"
	rosterOtherCompetitionID = "33333333-3333-4333-8333-333333333333"
)

// rosterFixture seeds a Competition hosted by rosterHost carrying one active
// entry, and returns the Service alongside it. The entry matters: without it
// every "the roster was withheld" assertion would also pass against a check
// that authorized correctly and returned nothing.
func rosterFixture(t *testing.T) (*app.Service, *fakeRepository, *fakeCompetitionAdminRepository, domain.Competition) {
	t.Helper()

	repo := newFakeRepository()
	admins := newFakeCompetitionAdminRepository()
	svc := newAdminTestService(repo, admins)

	competition := domain.Competition{
		ID:       rosterCompetitionID,
		HostID:   rosterHost,
		Name:     "Roster Fixture Open",
		Capacity: 8,
		Status:   domain.StatusScheduled,
	}
	if _, err := repo.Create(context.Background(), competition); err != nil {
		t.Fatalf("seeding the fixture Competition: %v", err)
	}

	if _, err := svc.EnterCompetition(context.Background(), app.EnterCompetitionInput{
		CompetitionID: competition.ID,
		PlayerID:      rosterEntrant,
		Source:        domain.EntrySourceApp,
	}); err != nil {
		t.Fatalf("seeding an entry: %v", err)
	}

	return svc, repo, admins, competition
}

// assignAdmin grants adminUserID Competition-Admin authority over
// competitionID through the real Host-only write path.
func assignAdmin(t *testing.T, svc *app.Service, competitionID, adminUserID string) {
	t.Helper()

	if _, err := svc.AssignCompetitionAdmin(context.Background(), app.AssignCompetitionAdminInput{
		CompetitionID: competitionID,
		ActorUserID:   rosterHost,
		AdminUserID:   adminUserID,
	}); err != nil {
		t.Fatalf("the Host assigning %q as a Competition Admin: %v", adminUserID, err)
	}
}

// TestListEntriesForCompetition_AssignedCompetitionAdminReadsTheRoster is
// this ticket's headline assertion, and the one that fails red against
// T13.6's Host-only check. It is deliberately not satisfied by a bare
// error-is-nil assertion: the roster has to actually arrive, because a
// widened check that authorized and then returned an empty list would break
// the very cash-reconciliation dashboard the widening exists to serve.
func TestListEntriesForCompetition_AssignedCompetitionAdminReadsTheRoster(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, _, competition := rosterFixture(t)
	assignAdmin(t, svc, competition.ID, rosterAdmin)

	got, err := svc.ListEntriesForCompetition(ctx, competition.ID, rosterAdmin)
	if err != nil {
		t.Fatalf("an assigned Competition Admin reading the roster: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the assigned Competition Admin read %d entries, want 1", len(got))
	}
	if got[0].PlayerID != rosterEntrant {
		t.Errorf("CompetitionEntry.PlayerID = %q, want %q", got[0].PlayerID, rosterEntrant)
	}
}

// TestListEntriesForCompetition_HostStillReadsTheRoster is the too-strict
// guard. Widening a check is exactly as capable of breaking the
// previously-entitled actor as of admitting the new one, and the Host's own
// roster read is what this RPC was built for.
func TestListEntriesForCompetition_HostStillReadsTheRoster(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, _, competition := rosterFixture(t)
	assignAdmin(t, svc, competition.ID, rosterAdmin)

	got, err := svc.ListEntriesForCompetition(ctx, competition.ID, rosterHost)
	if err != nil {
		t.Fatalf("the Competition's own Host reading the roster: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the Host read %d entries, want 1", len(got))
	}
}

// TestListEntriesForCompetition_UnentitledActorsAreStillRefused is the
// negative half, and the reason the widening is not a regression of #147
// itself.
//
// The rows are chosen so that each one fails a *different* way to be an
// admin. "Named themselves" is the row that matters most: it is the
// caller-supplied list this codebase used to have no alternative to,
// re-expressed as the closest thing the new API still permits — an actor id
// the caller controls, with no store row behind it.
func TestListEntriesForCompetition_UnentitledActorsAreStillRefused(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		actor string
		// setup runs against the fixture before the read, so each row can put
		// the store into the state its case is about.
		setup func(t *testing.T, svc *app.Service, repo *fakeRepository, competition domain.Competition)
	}{
		{
			// #147's enumerating stranger: a competition_id off the public
			// ListCompetitions response and nothing else.
			name:  "a stranger with no assignment",
			actor: rosterStranger,
		},
		{
			// The disclosed product limitation T13.6 recorded, unchanged by
			// this ticket: #147 leaves "may an entrant see the roster of a
			// Competition they are in" open, so until it is answered they may
			// not.
			name:  "an entrant of this very Competition",
			actor: rosterEntrant,
		},
		{
			// The caller-supplied-list attack, in the only form the widened
			// API leaves available. There is no wire field to forge any
			// more, so this row asserts the property that replaced it: an
			// actor id with no row in the store buys nothing at all.
			name:  "a caller who named themselves an admin but has no stored assignment",
			actor: "roster-self-appointed-subject",
		},
		{
			// Authority is per-Competition by construction (mirroring
			// CLAUDE.md's locked "per-game Game Admins"). An admin of some
			// *other* Competition is a real admin holding a real row — just
			// not one scoped here.
			name:  "an admin of a different Competition",
			actor: "roster-other-competition-admin-subject",
			setup: func(t *testing.T, svc *app.Service, repo *fakeRepository, _ domain.Competition) {
				t.Helper()
				other := domain.Competition{
					ID:       rosterOtherCompetitionID,
					HostID:   rosterHost,
					Name:     "Roster Fixture Open (second)",
					Capacity: 8,
					Status:   domain.StatusScheduled,
				}
				if _, err := repo.Create(context.Background(), other); err != nil {
					t.Fatalf("seeding the second Competition: %v", err)
				}
				assignAdmin(t, svc, rosterOtherCompetitionID, "roster-other-competition-admin-subject")
			},
		},
		{
			// A revoked admin is refused, which is what makes the read
			// resolve the set at call time rather than caching an answer.
			// Without this row a Host could withdraw authority and the
			// reader would keep it.
			name:  "an admin whose assignment was revoked",
			actor: "roster-revoked-admin-subject",
			setup: func(t *testing.T, svc *app.Service, _ *fakeRepository, competition domain.Competition) {
				t.Helper()
				assignAdmin(t, svc, competition.ID, "roster-revoked-admin-subject")
				if err := svc.RevokeCompetitionAdmin(context.Background(), app.RevokeCompetitionAdminInput{
					CompetitionID: competition.ID,
					ActorUserID:   rosterHost,
					AdminUserID:   "roster-revoked-admin-subject",
				}); err != nil {
					t.Fatalf("the Host revoking: %v", err)
				}
			},
		},
		{
			// An unidentified caller is never the Host and never an admin,
			// even against a store that somehow held a blank row —
			// domain.HasCompetitionAdmin's blank guard, reached from here.
			name:  "no actor at all",
			actor: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, repo, _, competition := rosterFixture(t)
			assignAdmin(t, svc, competition.ID, rosterAdmin)
			if tc.setup != nil {
				tc.setup(t, svc, repo, competition)
			}

			got, err := svc.ListEntriesForCompetition(ctx, competition.ID, tc.actor)
			if !errors.Is(err, domain.ErrNotCompetitionHostOrAdmin) {
				t.Fatalf("ListEntriesForCompetition as %s: err = %v, want %v", tc.name, err, domain.ErrNotCompetitionHostOrAdmin)
			}
			if len(got) != 0 {
				t.Errorf("a refused roster read still returned %d entries", len(got))
			}
		})
	}
}

// TestListEntriesForCompetition_UnknownCompetitionIsStillAnEmptyRoster pins
// the one behaviour this ticket must NOT change while adding a second
// repository read to the path.
//
// "Malformed is indistinguishable from unknown" is malformed_id_test.go's
// invariant, and the widened check now reaches
// port.CompetitionAdminRepository as well as port.Repository — a naive
// implementation that consulted the admin store before establishing the
// Competition exists could easily turn either case into an error. A
// Competition that does not exist has no roster to withhold.
func TestListEntriesForCompetition_UnknownCompetitionIsStillAnEmptyRoster(t *testing.T) {
	t.Parallel()

	for _, competitionID := range []string{
		"11111111-2222-3333-4444-555555555555", // well-formed, no such Competition
		"not-a-uuid",                           // malformed
		"",                                     // absent
	} {
		t.Run(competitionID, func(t *testing.T) {
			t.Parallel()

			svc, _, _, _ := rosterFixture(t)

			got, err := svc.ListEntriesForCompetition(context.Background(), competitionID, rosterStranger)
			if err != nil {
				t.Fatalf("ListEntriesForCompetition(%q) = %v, want an empty roster", competitionID, err)
			}
			if len(got) != 0 {
				t.Errorf("ListEntriesForCompetition(%q) returned %d entries, want 0", competitionID, len(got))
			}
		})
	}
}
