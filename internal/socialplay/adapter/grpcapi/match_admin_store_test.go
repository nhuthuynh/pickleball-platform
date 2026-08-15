// T14.5 — RecordMatchResult's admin branch stops trusting the caller.
//
// `assigned_game_admin_user_ids` stays on the wire (deprecated, retained for
// compatibility exactly as T12.8's `actor_*` fields were) and must be IGNORED.
// This file is that proof, in the shape T12.8 used for its own deprecated
// fields: a request naming the caller as an admin, with no corresponding store
// row, must be denied.
//
// # Why a wire field being ignored needs its own test
//
// It is the one property that cannot be observed from the change that
// introduced it. Deleting the line that read the field, and never writing a
// test, produces a diff that looks complete and a system whose behaviour
// nobody checked; re-adding the line later produces a diff that looks like a
// bug fix. T12.8's PR body made the same argument about the naive migration
// that would have passed its (a), (b) and (c) cases while remaining exactly as
// exploitable — the case that catches it is the one where the wire says what
// the attacker wants and the server has to not care.
//
// The mutation check CLAUDE.md rule 10 asks for is recorded in the PR body:
// with the field re-plumbed into app.RecordMatchResultInput, the "names
// themselves" test below goes green, which is the pre-T14.5 behaviour.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// TestRecordMatchResult_WireAdminListIsIgnored is instruction 4's required
// mutation proof.
//
// Each row sends a request whose `assigned_game_admin_user_ids` would, under
// the pre-T14.5 rule, have authorized the caller. None of them has a row in the
// store. All must be refused.
func TestRecordMatchResult_WireAdminListIsIgnored(t *testing.T) {
	tests := []struct {
		name      string
		actor     string
		wireAdmin []string
	}{
		{
			// The exact attack #168 describes: name yourself, pass the check.
			name:      "the caller names themselves",
			actor:     "wire-self-appointed",
			wireAdmin: []string{"wire-self-appointed"},
		},
		{
			// Buried in a longer list, in case anything ever short-circuited on
			// the first element.
			name:      "the caller names themselves among others",
			actor:     "wire-self-appointed",
			wireAdmin: []string{"someone-else", "another", "wire-self-appointed"},
		},
		{
			// The Host's own id on the wire buys nothing either: the actor is
			// the principal, and the list is not consulted at all.
			name:      "the caller names the Host",
			actor:     "wire-impostor",
			wireAdmin: []string{"host-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, gameRepo, matchRepo := newTestHandlerWithMatches()
			game := seedGame(t, gameRepo, "wire-ignored-"+tc.name, 4)

			_, err := h.RecordMatchResult(ctxAs(tc.actor), &socialplayv1.RecordMatchResultRequest{
				GameId:                   game.ID,
				Players:                  []string{"player-1", "player-2"},
				Score:                    map[string]int32{"player-1": 11, "player-2": 7},
				AssignedGameAdminUserIds: tc.wireAdmin,
			})
			assertCode(t, tc.name, err, codes.PermissionDenied)

			// Belt and braces: a refusal that still wrote would satisfy a
			// code-only assertion and remain the bug.
			if len(matchRepo.matches) != 0 {
				t.Errorf("a refused RecordMatchResult persisted %d Matches, want 0", len(matchRepo.matches))
			}
		})
	}
}

// TestRecordMatchResult_StoreGrantsWhatTheWireCannot is the positive control
// for the test above, and the reason its refusals mean something.
//
// The same subject, sending the same request, is refused before the Host
// assigns them and succeeds afterwards — with the wire list empty in both
// calls. Without this, every row above would also pass against a server that
// had simply broken RecordMatchResult for admins entirely.
func TestRecordMatchResult_StoreGrantsWhatTheWireCannot(t *testing.T) {
	h, gameRepo, matchRepo := newTestHandlerWithMatches()
	game := seedGame(t, gameRepo, "wire-vs-store", 4)

	const subject = "wire-vs-store-admin"
	req := func() *socialplayv1.RecordMatchResultRequest {
		return &socialplayv1.RecordMatchResultRequest{
			GameId:  game.ID,
			Players: []string{"player-1", "player-2"},
			Score:   map[string]int32{"player-1": 11, "player-2": 7},
			// Note: the caller asserts admin authority over themselves here,
			// and it changes nothing in either direction.
			AssignedGameAdminUserIds: []string{subject},
		}
	}

	_, before := h.RecordMatchResult(ctxAs(subject), req())
	assertCode(t, "recording before the assignment exists", before, codes.PermissionDenied)

	if _, err := h.AssignGameAdmin(ctxAs("host-1"), &socialplayv1.AssignGameAdminRequest{
		GameId: game.ID,
		UserId: subject,
	}); err != nil {
		t.Fatalf("the Host assigning %q: %v", subject, err)
	}

	if _, err := h.RecordMatchResult(ctxAs(subject), req()); err != nil {
		t.Fatalf("recording after the Host assigned %q should succeed, got: %v", subject, err)
	}
	if len(matchRepo.matches) != 1 {
		t.Fatalf("match repo holds %d Matches, want exactly 1 (the authorized call)", len(matchRepo.matches))
	}

	// And revoking takes it away again on the next call, which is what makes
	// the entitled set a live fact rather than a one-time grant.
	if _, err := h.RevokeGameAdmin(ctxAs("host-1"), &socialplayv1.RevokeGameAdminRequest{
		GameId: game.ID,
		UserId: subject,
	}); err != nil {
		t.Fatalf("the Host revoking %q: %v", subject, err)
	}

	_, after := h.RecordMatchResult(ctxAs(subject), req())
	assertCode(t, "recording after the assignment was revoked", after, codes.PermissionDenied)
}
