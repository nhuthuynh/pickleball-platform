//go:build integration

// T9.5 — a "large" test for the share-link read path against a REAL
// Postgres: it proves the GetCompetitionByShareToken query
// (db/queries/competitions.sql) really resolves a Competition by its token,
// really does NOT filter cancelled Competitions, and really produces the same
// domain.ErrCompetitionNotFound for a miss that an unknown ID produces.
//
// WHY THIS NEEDS A DATABASE, when sharelink_test.go in app/ and grpcapi/
// already covers the behaviour against fakes: the two properties that matter
// most here are properties of the SQL, and a fake cannot be wrong about them
// in the same way the SQL can. `WHERE share_token = $1` with an accidental
// `AND status = 'scheduled'` (copied from ListCompetitions, which correctly
// has one) would pass every fake-backed test in this ticket while turning
// every cancelled Competition's published link into a 404. Likewise, the
// no-rows -> ErrCompetitionNotFound translation is pgx behaviour, not fake
// behaviour.
//
// ---------------------------------------------------------------------------
// HOW THIS FILE'S EVIDENCE STANDS — read before citing it
// ---------------------------------------------------------------------------
// NOT EXECUTED BY ITS AUTHOR. No Docker daemon was available in this ticket's
// sandbox (`docker ps` fails with "dial unix /var/run/docker.sock: connect:
// no such file or directory") — the same gap HANDOFF.md records for
// T4/T5.4/T6.4/T6.5/T7/T8.3/T8.7/T9.2/T9.4. It was type-checked under its
// build tag (`go vet -tags=integration ./internal/competitions/adapter/
// postgres/`) and no further. It is committed anyway for the same reason
// capacity_concurrency_integration_test.go was: it is the portable,
// CI-runnable guard against exactly the SQL regression described above, and
// T6.4's review finding was that an uncommitted verification leaves nothing
// protecting the invariant.
//
// Do not describe this file as having passed until a run with a Docker
// daemon says so.
package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// TestGetByShareToken_ResolvesIncludingCancelled covers the whole
// token-addressed read against real SQL in one fixture: a hit, a
// cancelled-but-still-resolving hit, and the misses that must all answer
// identically.
func TestGetByShareToken_ResolvesIncludingCancelled(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	_, repo := newTestService(pool)

	const (
		scheduledToken = "share-token-scheduled-t95"
		cancelledToken = "share-token-cancelled-t95"
	)

	scheduled := seedCompetition(t, ctx, repo, "44444444-4444-4444-4444-200000000001", 8, 2, scheduledToken)
	cancelled := seedCompetition(t, ctx, repo, "44444444-4444-4444-4444-200000000002", 8, 2, cancelledToken)

	// --- a scheduled Competition resolves by its token, sessions and all ---
	got, err := repo.GetByShareToken(ctx, scheduledToken)
	if err != nil {
		t.Fatalf("GetByShareToken: %v", err)
	}
	if got.ID != scheduled.ID {
		t.Errorf("resolved competition %s, want %s", got.ID, scheduled.ID)
	}
	if len(got.Sessions) != len(scheduled.Sessions) {
		t.Errorf("resolved %d session(s), want %d — a link must show the same Competition an ID read shows", len(got.Sessions), len(scheduled.Sessions))
	}
	if got.ShareToken != scheduledToken {
		t.Errorf("round-tripped token = %q, want %q", got.ShareToken, scheduledToken)
	}

	// --- a CANCELLED Competition's token STILL resolves -------------------
	// The T9.5 requirement, and the one a stray `AND status = 'scheduled'`
	// in the query would break while every fake-backed test kept passing.
	if _, err := repo.UpdateStatus(ctx, cancelled.ID, domain.StatusCancelled); err != nil {
		t.Fatalf("fixture cancel failed: %v", err)
	}
	afterCancel, err := repo.GetByShareToken(ctx, cancelledToken)
	if err != nil {
		t.Fatalf("a cancelled Competition's share link must still resolve, got err: %v", err)
	}
	if afterCancel.Status != domain.StatusCancelled {
		t.Errorf("status = %q, want %q — the link must report an honest cancelled state, not a 404", afterCancel.Status, domain.StatusCancelled)
	}

	// --- every miss answers identically -----------------------------------
	// Unknown, malformed, empty, and "an ID used as a token" must all be the
	// same unwrapped sentinel: this read is unauthenticated and keyed by a
	// secret, so a distinguishable answer is an enumeration oracle.
	misses := []struct {
		name  string
		token string
	}{
		{"well-formed but unknown", "Zm9vYmFyYmF6cXV4Y29ycmVjdGxlbmd0aHRva2VuMDEy"},
		{"malformed", "not a token!! ***"},
		{"empty", ""},
		{"a competition id used as a token", scheduled.ID},
	}
	for _, m := range misses {
		_, err := repo.GetByShareToken(ctx, m.token)
		if !errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Errorf("%s: got err %v, want ErrCompetitionNotFound", m.name, err)
			continue
		}
		if err.Error() != domain.ErrCompetitionNotFound.Error() {
			t.Errorf("%s: error message %q is not the bare sentinel %q — a wrapped message would leak which tokens exist (and log the token itself)",
				m.name, err.Error(), domain.ErrCompetitionNotFound.Error())
		}
	}

	// And the ID-addressed read's own miss is the same answer.
	if _, err := repo.GetByID(ctx, "44444444-4444-4444-4444-2000000000ff"); !errors.Is(err, domain.ErrCompetitionNotFound) {
		t.Errorf("GetByID miss: got err %v, want ErrCompetitionNotFound", err)
	}
}
