// T15.4 — the roster read admits an assigned Competition Admin (closes
// #147), proved through the real handler -> app -> domain path.
//
// roster_authz_test.go (T13.6) is the four-case proof that this RPC stopped
// being public. This file is the fifth case that ticket could not write: the
// Competition Admin who *should* have been entitled all along and was
// refused because "assigned Competition Admin" was, at the time, whatever
// the caller typed. T15.3 made it a stored fact; these tests are what that
// fact bought. Exact twin of Social Play's roster_admin_authz_test.go
// (T14.5).
//
// Everything here goes through AssignCompetitionAdmin, the real Host-only
// write RPC. There is no seam for declaring someone an admin, which is the
// property under test as much as the read itself — a test that reached into
// the fake repository to plant a row would still pass if the handler had
// been wired to some other, forgeable source of admin identity.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// assignRosterAdmin has the Competition's Host grant adminSubject
// Competition-Admin authority over competitionID, through the real RPC.
func assignRosterAdmin(t *testing.T, h *grpcapi.Handler, competitionID, adminSubject string) {
	t.Helper()

	if _, err := h.AssignCompetitionAdmin(ctxAs(hostSubject), &competitionsv1.AssignCompetitionAdminRequest{
		CompetitionId: competitionID,
		UserId:        adminSubject,
	}); err != nil {
		t.Fatalf("the Host assigning %q as a Competition Admin: %v", adminSubject, err)
	}
}

// rosterAdminSubject is the Competition Admin these tests assign.
// Deliberately distinct from playerSubject and attackerSubject
// (principal_authz_test.go) so no row can pass by accidentally sharing an
// identity with an already-entitled actor.
const rosterAdminSubject = "auth0|competition-admin-1"

// TestListEntriesForCompetition_AssignedAdminPrincipalReadsTheRoster is the
// headline case. It asserts the roster actually arrives rather than merely
// that no error came back: an authorization change that admitted the admin
// and then returned nothing would break the pending-cash-entries
// reconciliation this widening exists to enable, while passing a bare
// error-is-nil check.
func TestListEntriesForCompetition_AssignedAdminPrincipalReadsTheRoster(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	if _, err := h.EnterCompetition(ctxAs(playerSubject), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err != nil {
		t.Fatalf("seeding an entry should succeed: %v", err)
	}

	assignRosterAdmin(t, h, competition.GetId(), rosterAdminSubject)

	resp, err := h.ListEntriesForCompetition(ctxAs(rosterAdminSubject), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	if err != nil {
		t.Fatalf("ListEntriesForCompetition as an assigned Competition Admin should succeed, got: %v", err)
	}

	entries := resp.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("the assigned Competition Admin read %d entries, want 1", len(entries))
	}
	if want := resolvedUserID(playerSubject); entries[0].GetPlayerId() != want {
		t.Errorf("CompetitionEntry.PlayerId = %q, want %q", entries[0].GetPlayerId(), want)
	}
}

// TestListEntriesForCompetition_UnentitledPrincipalsStillDenied is the
// negative half at the handler boundary: widening the entitled set must not
// have widened it past the store.
//
// Every row asserts PermissionDenied specifically, never Internal — a
// 500-shaped answer to an authorization question means the error escaped
// toStatus, which is its own bug class this package's tests have called out
// since T5.5.
func TestListEntriesForCompetition_UnentitledPrincipalsStillDenied(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		// revokeFirst assigns the actor and then revokes them, so the row
		// proves the read resolves authority at call time.
		revokeFirst bool
	}{
		{name: "a stranger with no assignment", actor: attackerSubject},
		{name: "an entrant of this very Competition", actor: playerSubject},
		{name: "an admin whose assignment was revoked", actor: "auth0|revoked-competition-admin-1", revokeFirst: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestHandler()
			competition := seedCompetition(t, h, hostSubject, 8, 0)

			// An entry must exist, so a passing test proves the roster was
			// *withheld* rather than merely empty.
			if _, err := h.EnterCompetition(ctxAs(playerSubject), &competitionsv1.EnterCompetitionRequest{
				CompetitionId: competition.GetId(),
			}); err != nil {
				t.Fatalf("seeding an entry should succeed: %v", err)
			}

			// A real admin exists throughout, so these rows prove the caller
			// was refused rather than that the check simply denies everyone.
			assignRosterAdmin(t, h, competition.GetId(), rosterAdminSubject)

			if tc.revokeFirst {
				assignRosterAdmin(t, h, competition.GetId(), tc.actor)
				if _, err := h.RevokeCompetitionAdmin(ctxAs(hostSubject), &competitionsv1.RevokeCompetitionAdminRequest{
					CompetitionId: competition.GetId(),
					UserId:        tc.actor,
				}); err != nil {
					t.Fatalf("the Host revoking %q: %v", tc.actor, err)
				}
			}

			resp, err := h.ListEntriesForCompetition(ctxAs(tc.actor), &competitionsv1.ListEntriesForCompetitionRequest{
				CompetitionId: competition.GetId(),
			})
			requireCode(t, "ListEntriesForCompetition", err, codes.PermissionDenied)

			if len(resp.GetEntries()) != 0 {
				t.Errorf("a rejected roster read still returned %d entries", len(resp.GetEntries()))
			}
		})
	}
}

// TestListEntriesForCompetition_AdminAuthorityIsPerCompetition pins the
// per-Competition scoping mirroring CLAUDE.md's locked "per-game Game
// Admins" decision.
//
// A Competition Admin of one Competition is a legitimate admin holding a
// legitimate stored row — the strongest available shape of the wrong
// answer. If the resolution query ever lost its competition scoping, every
// other test in this file would still pass and this one would fail, which is
// exactly why it is separate.
func TestListEntriesForCompetition_AdminAuthorityIsPerCompetition(t *testing.T) {
	h, _ := newTestHandler()

	first := seedCompetition(t, h, hostSubject, 8, 0)
	second := seedCompetition(t, h, hostSubject, 8, 0)
	if first.GetId() == second.GetId() {
		t.Fatalf("fixture precondition: the two Competitions must be distinct, both are %q", first.GetId())
	}

	if _, err := h.EnterCompetition(ctxAs(playerSubject), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: first.GetId(),
	}); err != nil {
		t.Fatalf("seeding an entry on the first Competition: %v", err)
	}

	// The admin is assigned to the SECOND Competition only.
	assignRosterAdmin(t, h, second.GetId(), rosterAdminSubject)

	resp, err := h.ListEntriesForCompetition(ctxAs(rosterAdminSubject), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: first.GetId(),
	})
	requireCode(t, "an admin of another Competition reading this Competition's roster", err, codes.PermissionDenied)
	if len(resp.GetEntries()) != 0 {
		t.Errorf("a rejected cross-Competition roster read still returned %d entries", len(resp.GetEntries()))
	}

	// And the same principal reading the Competition they actually
	// administer succeeds — without which the row above would be satisfied
	// by an admin who is simply broken everywhere.
	if _, err := h.ListEntriesForCompetition(ctxAs(rosterAdminSubject), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: second.GetId(),
	}); err != nil {
		t.Fatalf("the admin reading the Competition they administer should succeed, got: %v", err)
	}
}

// TestListEntriesForCompetition_AnonymousIsStillUnauthenticated re-pins
// ADR-0013 §5 at this RPC after the widening. "I do not know who you are"
// must stay a different code from "I know who you are and you may not": the
// fix for the former is to authenticate, and PermissionDenied does not say
// so.
//
// roster_authz_test.go already covers this for the Host-only rule; it is
// repeated here because the check it guards is a different one now, and a
// widened check that resolved admins *before* demanding a principal could
// plausibly have started answering PermissionDenied to anonymous callers.
func TestListEntriesForCompetition_AnonymousIsStillUnauthenticated(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)
	assignRosterAdmin(t, h, competition.GetId(), rosterAdminSubject)

	_, err := h.ListEntriesForCompetition(anonymous(), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	requireCode(t, "an anonymous roster read", err, codes.Unauthenticated)
}
