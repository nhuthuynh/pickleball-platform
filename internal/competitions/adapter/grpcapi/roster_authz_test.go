// T13.6 — the roster read stopped being public (partial fix for #147).
//
// **Superseded in part by T15.4** (closes #147), which widened the entitled
// set from Host-only to Host-or-assigned-Competition-Admin once T15.3's
// durable store made "assigned Competition Admin" a server fact. The cases
// below are unchanged and still hold — the Host reads, a stranger and an
// entrant are refused, an anonymous caller is Unauthenticated, an unknown or
// malformed id is an empty roster. What is no longer true is this file's
// original claim that the *admin* is refused too; roster_admin_authz_test.go
// carries that half now. The two files are kept separate rather than merged
// so the T13.6 proof stays legible as the thing it was: the fix for the leak
// itself, which is independent of who the entitled set eventually grew to
// include — the exact split Social Play's twin already established at T14.5.
//
// Before this ticket ListEntriesForCompetition had no authorization check of
// any kind: it was in PublicMethods(), took no actor, and returned every
// entrant's player_id and payment status to anyone holding a competition_id —
// a value readable off the *public* ListCompetitions and GetCompetition
// responses, which is what made the leak enumerable rather than merely
// theoretical.
//
// The four cases, in the shape principal_authz_test.go established:
//
//	(a) the Competition's Host                -> succeeds, sees the roster
//	(b) a verified caller who is not the Host -> PermissionDenied
//	(c) no principal at all                   -> Unauthenticated
//	(d) an unknown or malformed id            -> empty roster, unchanged
//
// (b) and (c) stay distinct codes per ADR-0013 §5, and (b) is asserted as
// PermissionDenied specifically, never Internal.
//
// The entrant case in (b) is a deliberate, disclosed narrowness that
// *outlives* this widening: #147 leaves open whether an entrant should be
// able to see the roster of a Competition they entered, and T15.4 does not
// answer that product question — see roster_admin_read_test.go's header for
// why answering it here would smuggle a product decision into an
// authorization change.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// --- (a) the Host reads the roster ---------------------------------------

func TestListEntriesForCompetition_HostPrincipalReadsTheRoster(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	if _, err := h.EnterCompetition(ctxAs(playerSubject), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err != nil {
		t.Fatalf("seeding an entry should succeed: %v", err)
	}

	resp, err := h.ListEntriesForCompetition(ctxAs(hostSubject), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	if err != nil {
		t.Fatalf("ListEntriesForCompetition as the Competition's Host should succeed, got: %v", err)
	}

	// The roster must actually arrive — an authorization check that also
	// emptied the Host's own roster read would pass a bare error-is-nil
	// assertion while breaking the feature.
	entries := resp.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("Host read %d entries, want 1 — the Host's own roster read is broken", len(entries))
	}
	if want := resolvedUserID(playerSubject); entries[0].GetPlayerId() != want {
		t.Errorf("CompetitionEntry.PlayerId = %q, want %q", entries[0].GetPlayerId(), want)
	}
}

// --- (b) and (c) the caller who is not the Host --------------------------

// TestListEntriesForCompetition_NonHostPrincipalIsPermissionDenied is the
// regression for #147. Before T13.6 every row of this table returned OK plus
// the full roster.
func TestListEntriesForCompetition_NonHostPrincipalIsPermissionDenied(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		want codes.Code
	}{
		{
			name: "a verified caller who is not the Host",
			ctx:  func() context.Context { return ctxAs(attackerSubject) },
			want: codes.PermissionDenied,
		},
		{
			// Entered this very Competition and still refused — the disclosed
			// Host-only narrowness, asserted rather than assumed.
			name: "an entrant of this very Competition",
			ctx:  func() context.Context { return ctxAs(playerSubject) },
			want: codes.PermissionDenied,
		},
		{
			name: "no principal at all",
			ctx:  anonymous,
			want: codes.Unauthenticated,
		},
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

			resp, err := h.ListEntriesForCompetition(tc.ctx(), &competitionsv1.ListEntriesForCompetitionRequest{
				CompetitionId: competition.GetId(),
			})
			requireCode(t, "ListEntriesForCompetition", err, tc.want)

			if len(resp.GetEntries()) != 0 {
				t.Errorf("a rejected roster read still returned %d entries", len(resp.GetEntries()))
			}
		})
	}
}

// --- (d) the unknown-Competition answer is unchanged ---------------------

// TestListEntriesForCompetition_UnknownCompetitionIsStillAnEmptyRoster pins
// the behaviour T13.6 deliberately does NOT change — see the twin test in
// Social Play for the full reasoning. In short: malformed_id_test.go's
// invariant is "malformed is indistinguishable from unknown", the Host check
// puts an ErrCompetitionNotFound on the path, and propagating it would be an
// unrelated behaviour change that buys nothing (a Competition that does not
// exist has no roster, and its existence is public via GetCompetition).
func TestListEntriesForCompetition_UnknownCompetitionIsStillAnEmptyRoster(t *testing.T) {
	for _, competitionID := range []string{
		"11111111-2222-3333-4444-555555555555", // well-formed, no such Competition
		"not-a-uuid",                           // malformed
		"",                                     // absent
	} {
		t.Run(competitionID, func(t *testing.T) {
			h, _ := newTestHandler()

			resp, err := h.ListEntriesForCompetition(ctxAs(attackerSubject), &competitionsv1.ListEntriesForCompetitionRequest{
				CompetitionId: competitionID,
			})
			if err != nil {
				t.Fatalf("ListEntriesForCompetition(%q) error = %v, want an empty roster", competitionID, err)
			}
			if len(resp.GetEntries()) != 0 {
				t.Errorf("ListEntriesForCompetition(%q) returned %d entries, want 0", competitionID, len(resp.GetEntries()))
			}
		})
	}
}
