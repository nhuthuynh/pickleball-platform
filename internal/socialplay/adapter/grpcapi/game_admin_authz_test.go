// T14.4 — handler-level authorization regression tests for the durable
// Game-Admin store (partial fix for #168).
//
// The assertion the ticket names explicitly (instruction 6) is
// TestAssignGameAdmin_AnAssignedAdminIsDenied: "prove it with a test where an
// assigned admin attempts an assignment and gets PermissionDenied." #168 says
// why in one line — "an admin must not be able to appoint an admin, or the
// Host-only distinction ErrNotGameHost exists to protect is worthless" — and
// the failure mode it guards against is not hypothetical: this codebase
// already has two near-identical checks on the same aggregate
// (EnsureHostOrGameAdmin vs EnsureHost), and domain/errors.go carries a
// DO-NOT-UNIFY warning because merging them would silently widen one.
//
// Handler-level with this package's in-memory port fakes, for the reason
// authz_regression_test.go documents at length: the rule under test lives
// entirely in the domain and app layers, port.GameAdminRepository does not
// influence it, and this environment has no Docker daemon — so the test has to
// actually run where it was written (CLAUDE.md rule 10). The Postgres
// round-trip half is covered separately by
// internal/socialplay/adapter/postgres/game_admin_integration_test.go, which
// proves what a fake cannot: that the real store's uniqueness constraint and
// the real read agree with the domain.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

const (
	gaTestHost     = "ga-host"
	gaTestAdmin    = "ga-admin"
	gaTestSecond   = "ga-second"
	gaTestStranger = "ga-stranger"
)

// assertCode fails unless err carries the expected gRPC code, reporting the
// actual error text so a wrong-code failure is diagnosable rather than just
// wrong.
func assertCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s succeeded, want %v", what, want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", what, err)
	}
	if st.Code() != want {
		t.Fatalf("%s answered %v, want %v (err: %v)", what, st.Code(), want, err)
	}
}

// TestAssignGameAdmin_AnAssignedAdminIsDenied is instruction 6's required
// proof, driven through the real handler -> app -> domain path.
//
// It asserts two things a weaker version would miss: the code is
// PermissionDenied (not Internal, not a silent success), AND the refused
// assignment left no row behind — a check that answered 403 while still
// writing would satisfy a code-only assertion and still be the bug.
func TestAssignGameAdmin_AnAssignedAdminIsDenied(t *testing.T) {
	h, gameRepo, _ := newTestHandler()
	game := seedGameWithHost(t, gameRepo, "ga-admin-cannot-appoint", gaTestHost)

	// The Host assigns a first admin. This is the legitimate path, and it has
	// to work, or the test below would prove nothing.
	if _, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
		GameId: game.ID,
		UserId: gaTestAdmin,
	}); err != nil {
		t.Fatalf("the Host assigning an admin failed: %v", err)
	}

	// That admin now tries to appoint another one.
	_, err := h.AssignGameAdmin(ctxAs(gaTestAdmin), &socialplayv1.AssignGameAdminRequest{
		GameId: game.ID,
		UserId: gaTestSecond,
	})
	assertCode(t, "an assigned Game Admin appointing another admin", err, codes.PermissionDenied)

	// And the refusal wrote nothing: the Host revoking gaTestSecond must find
	// no assignment to remove.
	_, revokeErr := h.RevokeGameAdmin(ctxAs(gaTestHost), &socialplayv1.RevokeGameAdminRequest{
		GameId: game.ID,
		UserId: gaTestSecond,
	})
	assertCode(t, "revoking the assignment a refused appointment would have created", revokeErr, codes.NotFound)
}

// TestGameAdminWrites_RejectEveryNonHostActor widens the above across both
// RPCs and every non-Host actor shape, so a future change that special-cased
// only one of them fails here.
func TestGameAdminWrites_RejectEveryNonHostActor(t *testing.T) {
	actors := []struct {
		name    string
		subject string
	}{
		{"an assigned game admin", gaTestAdmin},
		{"a stranger", gaTestStranger},
	}

	for _, tt := range actors {
		t.Run(tt.name, func(t *testing.T) {
			h, gameRepo, _ := newTestHandler()
			game := seedGameWithHost(t, gameRepo, "ga-nonhost-"+tt.name, gaTestHost)

			if _, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
				GameId: game.ID, UserId: gaTestAdmin,
			}); err != nil {
				t.Fatalf("seeding an admin: %v", err)
			}

			_, assignErr := h.AssignGameAdmin(ctxAs(tt.subject), &socialplayv1.AssignGameAdminRequest{
				GameId: game.ID, UserId: gaTestSecond,
			})
			assertCode(t, tt.name+" assigning", assignErr, codes.PermissionDenied)

			_, revokeErr := h.RevokeGameAdmin(ctxAs(tt.subject), &socialplayv1.RevokeGameAdminRequest{
				GameId: game.ID, UserId: gaTestAdmin,
			})
			assertCode(t, tt.name+" revoking", revokeErr, codes.PermissionDenied)
		})
	}
}

// TestGameAdminWrites_RequireAPrincipal pins ADR-0013 §5's distinction at
// these two new RPCs: "I do not know who you are" is Unauthenticated, not
// PermissionDenied. Without it, a caller with no token would be told they lack
// permission — which is both wrong and unactionable (a client that reads 403
// does not go get a token).
//
// authenticated_test.go separately proves both RPCs are in
// AuthenticatedMethods(), which is what makes the interceptor reject them
// before this handler runs in production. This test covers the handler's own
// behaviour when reached without a principal anyway.
func TestGameAdminWrites_RequireAPrincipal(t *testing.T) {
	h, gameRepo, _ := newTestHandler()
	game := seedGameWithHost(t, gameRepo, "ga-anonymous", gaTestHost)

	_, assignErr := h.AssignGameAdmin(anonymous(), &socialplayv1.AssignGameAdminRequest{
		GameId: game.ID, UserId: gaTestAdmin,
	})
	assertCode(t, "an anonymous caller assigning", assignErr, codes.Unauthenticated)

	_, revokeErr := h.RevokeGameAdmin(anonymous(), &socialplayv1.RevokeGameAdminRequest{
		GameId: game.ID, UserId: gaTestAdmin,
	})
	assertCode(t, "an anonymous caller revoking", revokeErr, codes.Unauthenticated)
}

// TestAssignGameAdmin_HostHappyPathAndItsAnswers covers the Host's own path
// end to end: the assignment succeeds, the response carries the server-minted
// facts (assigner and timestamp) rather than echoing the request, and the
// three input rejections answer the codes the proto promises.
func TestAssignGameAdmin_HostHappyPathAndItsAnswers(t *testing.T) {
	h, gameRepo, _ := newTestHandler()
	game := seedGameWithHost(t, gameRepo, "ga-happy-path", gaTestHost)

	resp, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
		GameId: game.ID, UserId: gaTestAdmin,
	})
	if err != nil {
		t.Fatalf("the Host assigning an admin: %v", err)
	}

	got := resp.GetGameAdmin()
	if got.GetUserId() != gaTestAdmin {
		t.Errorf("response user_id = %q, want %q", got.GetUserId(), gaTestAdmin)
	}
	if got.GetGameId() != game.ID {
		t.Errorf("response game_id = %q, want %q", got.GetGameId(), game.ID)
	}
	if got.GetAssignedBy() != gaTestHost {
		t.Errorf("response assigned_by = %q, want the verified Host %q — the assigner is a server fact, not a request field",
			got.GetAssignedBy(), gaTestHost)
	}
	if got.GetAssignedAt() == nil {
		t.Error("response assigned_at is unset; the server supplies the clock")
	}

	t.Run("assigning the same user twice", func(t *testing.T) {
		_, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
			GameId: game.ID, UserId: gaTestAdmin,
		})
		assertCode(t, "a duplicate assignment", err, codes.AlreadyExists)
	})

	t.Run("assigning the host themselves", func(t *testing.T) {
		_, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
			GameId: game.ID, UserId: gaTestHost,
		})
		assertCode(t, "assigning the Host as their own admin", err, codes.InvalidArgument)
	})

	t.Run("assigning a blank user id", func(t *testing.T) {
		_, err := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
			GameId: game.ID, UserId: "",
		})
		assertCode(t, "assigning a blank user id", err, codes.InvalidArgument)
	})

	t.Run("the host revokes", func(t *testing.T) {
		if _, err := h.RevokeGameAdmin(ctxAs(gaTestHost), &socialplayv1.RevokeGameAdminRequest{
			GameId: game.ID, UserId: gaTestAdmin,
		}); err != nil {
			t.Fatalf("the Host revoking: %v", err)
		}

		// Revoking again finds nothing — which is also the proof that the
		// first revoke actually removed the row rather than reporting success
		// over a no-op.
		_, err := h.RevokeGameAdmin(ctxAs(gaTestHost), &socialplayv1.RevokeGameAdminRequest{
			GameId: game.ID, UserId: gaTestAdmin,
		})
		assertCode(t, "revoking an already-revoked assignment", err, codes.NotFound)
	})
}

// TestGameAdminWrites_MalformedAndUnknownGameIDs pins the T10.7-shaped
// boundary guard on both new RPCs. A malformed game_id must answer exactly
// what an unknown-but-well-formed one answers, and must never reach the
// Postgres adapter's mustUUID — which panics, and grpc installs no recover()
// of its own (the T9 incident recorded in docs/LESSONS.md).
func TestGameAdminWrites_MalformedAndUnknownGameIDs(t *testing.T) {
	ids := []struct {
		name   string
		gameID string
	}{
		{"malformed", "not-a-uuid"},
		{"empty", ""},
		{"unknown but well formed", mapUnknownID},
	}

	for _, tt := range ids {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _ := newTestHandler()

			_, assignErr := h.AssignGameAdmin(ctxAs(gaTestHost), &socialplayv1.AssignGameAdminRequest{
				GameId: tt.gameID, UserId: gaTestAdmin,
			})
			assertCode(t, "assigning against a "+tt.name+" game id", assignErr, codes.NotFound)

			_, revokeErr := h.RevokeGameAdmin(ctxAs(gaTestHost), &socialplayv1.RevokeGameAdminRequest{
				GameId: tt.gameID, UserId: gaTestAdmin,
			})
			assertCode(t, "revoking against a "+tt.name+" game id", revokeErr, codes.NotFound)
		})
	}
}
