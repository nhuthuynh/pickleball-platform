// T9.5 — application-layer tests for the shareable registration link's
// token-addressed read path.
//
// SCOPE NOTE, so a reader doesn't go looking for the missing half: T9.5 does
// NOT build the token generator. internal/competitions/adapter/sharetoken
// (T9.4) already implements port.ShareTokenGenerator over crypto/rand with
// 256 bits of entropy, and app.Service.ScheduleCompetition already mints a
// token for every Competition at construction. What this file covers is
// everything DOWNSTREAM of that: resolving a Competition by its token,
// keeping the not-found answer free of any signal about which tokens exist,
// and keeping a cancelled Competition's link honest rather than dead.
package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// TestGetCompetitionByShareToken_ResolvesTheCompetitionTheHostShared is the
// happy path: the token minted at creation resolves back to exactly the
// Competition it was minted for, sessions and all.
//
// It deliberately reads the token off the scheduled Competition rather than
// constructing one, so the test exercises the real produce -> publish ->
// resolve loop instead of a value invented by the test.
func TestGetCompetitionByShareToken_ResolvesTheCompetitionTheHostShared(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 2)

	if c.ShareToken == "" {
		t.Fatal("fixture invalid: ScheduleCompetition must mint a share token (T9.4)")
	}

	got, err := svc.GetCompetitionByShareToken(context.Background(), c.ShareToken)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(got, c) {
		t.Fatalf("token-addressed read returned a different Competition:\n got: %+v\nwant: %+v", got, c)
	}
}

// TestGetCompetitionByShareToken_ResolvesOnlyItsOwnCompetition pins the
// scoping: with two Competitions in the repository, each token resolves to
// its own and never to the other's. Without this, a lookup that ignored the
// token (or matched the first row) would still pass the happy-path test
// above.
func TestGetCompetitionByShareToken_ResolvesOnlyItsOwnCompetition(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := newTestService(repo, newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})
	ctx := context.Background()

	first, err := svc.ScheduleCompetition(ctx, validInput(t))
	if err != nil {
		t.Fatalf("fixture ScheduleCompetition failed: %v", err)
	}
	second, err := svc.ScheduleCompetition(ctx, validInput(t))
	if err != nil {
		t.Fatalf("fixture ScheduleCompetition failed: %v", err)
	}
	if first.ShareToken == second.ShareToken {
		t.Fatal("fixture invalid: two Competitions must not share a token")
	}

	got, err := svc.GetCompetitionByShareToken(ctx, second.ShareToken)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != second.ID {
		t.Fatalf("token for %s resolved to %s", second.ID, got.ID)
	}
}

// TestGetCompetitionByShareToken_NotFoundIsIndistinguishable is the security
// requirement, not a style preference: this endpoint is unauthenticated and
// keyed by a secret, so any difference between "that token isn't real" and
// "that Competition isn't real" turns it into an oracle an enumerator can
// use to confirm hits without ever seeing a Competition.
//
// Every rejected case must therefore produce the SAME error — the bare
// domain.ErrCompetitionNotFound sentinel, unwrapped. Wrapping it with the
// token, the reason, or anything else would leak exactly the distinction
// this test exists to remove, which is why the assertion is on the error
// STRING being byte-identical and not merely on errors.Is.
//
// The ID-addressed GetCompetition's own not-found error is included in the
// comparison set for the same reason: "is this token real" and "is this
// competition real" must be one answer, not two.
func TestGetCompetitionByShareToken_NotFoundIsIndistinguishable(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ctx := context.Background()

	cases := []struct {
		name  string
		token string
	}{
		// Well-formed (same shape as a real 43-char base64url token) but
		// belonging to no Competition — "is this competition real?"
		{"well-formed but unknown", "Zm9vYmFyYmF6cXV4Y29ycmVjdGxlbmd0aHRva2VuMDEy"},
		// Structurally impossible as a token — "is this token real?"
		{"malformed: illegal characters", "not a token!! ***"},
		{"malformed: empty", ""},
		{"malformed: whitespace", "   "},
		// A Competition's ID is NOT its token: passing one must not resolve,
		// and must not be distinguishable from any other miss.
		{"a competition id, not a token", c.ID},
	}

	var want error
	for _, tc := range cases {
		_, err := svc.GetCompetitionByShareToken(ctx, tc.token)
		if err == nil {
			t.Fatalf("%s: resolved, want not-found", tc.name)
		}
		if !errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Fatalf("%s: got err %v, want ErrCompetitionNotFound", tc.name, err)
		}
		if want == nil {
			want = err
			continue
		}
		if err.Error() != want.Error() {
			t.Fatalf("%s: error message %q differs from %q — this endpoint is an oracle for which tokens exist", tc.name, err.Error(), want.Error())
		}
	}

	// And the same answer the ID-addressed read gives for an unknown ID.
	_, err := svc.GetCompetition(ctx, "no-such-competition")
	if err == nil || err.Error() != want.Error() {
		t.Fatalf("GetCompetition's not-found error %v differs from the token-addressed one %v", err, want)
	}
}

// TestGetCompetitionByShareToken_CancelledCompetitionStillResolves is the
// ticket's Given/When/Then, at the layer where the decision lives.
//
// GIVEN a Host cancelled a Competition after posting its link,
// WHEN a Player follows that (still-valid) link,
// THEN they see the Competition with status cancelled — an honest "this
// competition was cancelled" state — rather than a 404 that is
// indistinguishable from a broken or mistyped link (NN/g heuristic #9: a
// dead link and a cancelled event are different facts, and only one of them
// is the Player's problem to fix).
//
// The second half asserts the other side of that contract: reading is
// allowed, ENTERING is still rejected. That rejection is T9.3/T9.4's
// existing domain.Enter behaviour — verified here, not reimplemented.
func TestGetCompetitionByShareToken_CancelledCompetitionStillResolves(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ctx := context.Background()

	// GIVEN: the Host posts the link, then cancels.
	sharedLinkToken := c.ShareToken
	if _, err := svc.CancelCompetition(ctx, c.ID, c.HostID); err != nil {
		t.Fatalf("fixture cancel failed: %v", err)
	}

	// WHEN: a Player follows the link that is already out in the world.
	got, err := svc.GetCompetitionByShareToken(ctx, sharedLinkToken)
	if err != nil {
		t.Fatalf("a cancelled Competition's link must still resolve, got err: %v", err)
	}

	// THEN: an honest cancelled state, not a not-found.
	if got.Status != domain.StatusCancelled {
		t.Fatalf("status = %q, want %q", got.Status, domain.StatusCancelled)
	}
	if got.ID != c.ID || got.Name != c.Name {
		t.Fatalf("resolved the wrong Competition: %+v", got)
	}

	// AND: entering it is still rejected (existing T9.3/T9.4 behaviour).
	if _, err := svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: got.ID, PlayerID: "player-1", Source: domain.EntrySourceSocial,
	}); !errors.Is(err, domain.ErrCompetitionCancelled) {
		t.Fatalf("entering a cancelled Competition: got err %v, want ErrCompetitionCancelled", err)
	}
}

// TestGetCompetitionByShareToken_DoesNotInferEntrySource is the "the server
// validates, it does not infer" rule stated as a test.
//
// A Player who arrived through a share link and then enters declaring
// source=app is recorded as `app`. The share token is not a signal the
// server is allowed to reason from: the client declares the channel
// explicitly on EnterCompetitionRequest, and anything else would make the
// attribution a guess that looks like a fact on a Host's roster.
func TestGetCompetitionByShareToken_DoesNotInferEntrySource(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ctx := context.Background()

	resolved, err := svc.GetCompetitionByShareToken(ctx, c.ShareToken)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	entry, err := svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: resolved.ID,
		PlayerID:      "player-1",
		// The entrant reached this Competition through the share link, but
		// declares `app`. The declaration wins.
		Source: domain.EntrySourceApp,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry.Source != domain.EntrySourceApp {
		t.Fatalf("source = %q, want %q — the server must not infer `social` from the share-token lookup", entry.Source, domain.EntrySourceApp)
	}
}
