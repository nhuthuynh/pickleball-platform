// T15.3 — the Competition-Admin RPCs at the wire boundary (#168, partial fix).
// The mirror of internal/socialplay/adapter/grpcapi/game_admin_authz_test.go.
//
// What these prove that the domain and app tests cannot: that the acting
// identity really is the verified principal rather than anything on the
// request (neither request message HAS an actor field, and these tests are
// what make that absence load-bearing rather than incidental), that a missing
// principal is Unauthenticated and not PermissionDenied (ADR-0013 §5), and
// that the Host-only rule survives all the way out to the status code a client
// sees.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// fakeCompetitionAdminRepo is an in-memory port.CompetitionAdminRepository for
// the handler tests, reproducing the real adapter's three observable
// contracts: the composite-PK duplicate rejection, the
// revoke-removed-nothing answer, and the empty read for a Competition with no
// admins.
type fakeCompetitionAdminRepo struct {
	admins map[string][]domain.CompetitionAdmin
}

func newFakeCompetitionAdminRepo() *fakeCompetitionAdminRepo {
	return &fakeCompetitionAdminRepo{admins: make(map[string][]domain.CompetitionAdmin)}
}

func (r *fakeCompetitionAdminRepo) Assign(_ context.Context, a domain.CompetitionAdmin) (domain.CompetitionAdmin, error) {
	if domain.HasCompetitionAdmin(r.admins[a.CompetitionID], a.UserID) {
		return domain.CompetitionAdmin{}, domain.ErrAlreadyCompetitionAdmin
	}
	r.admins[a.CompetitionID] = append(r.admins[a.CompetitionID], a)
	return a, nil
}

func (r *fakeCompetitionAdminRepo) Revoke(_ context.Context, competitionID, userID string) error {
	existing := r.admins[competitionID]
	for i, a := range existing {
		if a.UserID == userID {
			r.admins[competitionID] = append(append([]domain.CompetitionAdmin{}, existing[:i]...), existing[i+1:]...)
			return nil
		}
	}
	return domain.ErrCompetitionAdminNotFound
}

func (r *fakeCompetitionAdminRepo) ListCompetitionAdmins(_ context.Context, competitionID string) ([]domain.CompetitionAdmin, error) {
	out := make([]domain.CompetitionAdmin, len(r.admins[competitionID]))
	copy(out, r.admins[competitionID])
	return out, nil
}

const (
	caHostID   = "ca-host-subject"
	caAdminID  = "ca-admin-subject"
	caStranger = "ca-stranger-subject"
	caSecond   = "ca-second-admin-subject"
)

func TestAssignCompetitionAdmin_HostSucceeds(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler()
	c := seedCompetition(t, h, caHostID, 16, 2)

	resp, err := h.AssignCompetitionAdmin(ctxAs(caHostID), &competitionsv1.AssignCompetitionAdminRequest{
		CompetitionId: c.GetId(),
		UserId:        caAdminID,
	})
	if err != nil {
		t.Fatalf("AssignCompetitionAdmin() unexpected error: %v", err)
	}

	got := resp.GetCompetitionAdmin()
	if got.GetCompetitionId() != c.GetId() || got.GetUserId() != caAdminID {
		t.Fatalf("assignment = %+v, want competition %q / user %q", got, c.GetId(), caAdminID)
	}
	// assigned_by is the verified principal. This is the assertion that would
	// fail if a future change ever let the request name the assigner — the
	// exact caller-asserted admin fact #168 exists to abolish.
	if got.GetAssignedBy() != caHostID {
		t.Errorf("assigned_by = %q, want %q — the assigner is the token's subject, never the wire", got.GetAssignedBy(), caHostID)
	}
	if got.GetAssignedAt() == nil {
		t.Error("assigned_at is nil — the timestamp is a server fact, minted by the app layer")
	}
}

// TestAssignCompetitionAdmin_AnAdminCannotAppointAnAdmin is #168's headline
// rule proven at the RPC boundary, not merely in the domain: an assigned
// Competition Admin gets PermissionDenied, exactly as a stranger does.
//
// The two rows share one expectation on purpose. If a future change ever
// widened the gate to "host or admin", the stranger row would still pass and
// only the admin row would fail — which is precisely why the admin row is
// written out separately rather than folded into a generic "non-host" case.
func TestAssignCompetitionAdmin_AnAdminCannotAppointAnAdmin(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler()
	c := seedCompetition(t, h, caHostID, 16, 2)

	if _, err := h.AssignCompetitionAdmin(ctxAs(caHostID), &competitionsv1.AssignCompetitionAdminRequest{
		CompetitionId: c.GetId(),
		UserId:        caAdminID,
	}); err != nil {
		t.Fatalf("seed AssignCompetitionAdmin: %v", err)
	}

	tests := []struct {
		name  string
		actor string
	}{
		{name: "an assigned competition admin", actor: caAdminID},
		{name: "a stranger", actor: caStranger},
	}

	for _, tt := range tests {
		t.Run(tt.name+" cannot assign", func(t *testing.T) {
			t.Parallel()

			_, err := h.AssignCompetitionAdmin(ctxAs(tt.actor), &competitionsv1.AssignCompetitionAdminRequest{
				CompetitionId: c.GetId(),
				UserId:        caSecond,
			})
			if got := status.Code(err); got != codes.PermissionDenied {
				t.Fatalf("code = %v (err %v), want PermissionDenied — an admin who can appoint admins makes the Host-only rule unenforceable (#168)", got, err)
			}
		})

		t.Run(tt.name+" cannot revoke", func(t *testing.T) {
			t.Parallel()

			_, err := h.RevokeCompetitionAdmin(ctxAs(tt.actor), &competitionsv1.RevokeCompetitionAdminRequest{
				CompetitionId: c.GetId(),
				UserId:        caAdminID,
			})
			if got := status.Code(err); got != codes.PermissionDenied {
				t.Fatalf("code = %v (err %v), want PermissionDenied — appointing and un-appointing are the same authority", got, err)
			}
		})
	}
}

// TestCompetitionAdminRPCs_RequireAPrincipal pins ADR-0013 §5: an unauthenticated
// caller is Unauthenticated ("I do not know who you are"), never
// PermissionDenied ("I know who you are and you may not"). Conflating the two
// tells an anonymous caller that a valid token would not have helped, which is
// both wrong and an information leak.
func TestCompetitionAdminRPCs_RequireAPrincipal(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler()
	c := seedCompetition(t, h, caHostID, 16, 2)

	t.Run("assign", func(t *testing.T) {
		t.Parallel()

		_, err := h.AssignCompetitionAdmin(context.Background(), &competitionsv1.AssignCompetitionAdminRequest{
			CompetitionId: c.GetId(),
			UserId:        caAdminID,
		})
		if got := status.Code(err); got != codes.Unauthenticated {
			t.Fatalf("code = %v (err %v), want Unauthenticated", got, err)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		t.Parallel()

		_, err := h.RevokeCompetitionAdmin(context.Background(), &competitionsv1.RevokeCompetitionAdminRequest{
			CompetitionId: c.GetId(),
			UserId:        caAdminID,
		})
		if got := status.Code(err); got != codes.Unauthenticated {
			t.Fatalf("code = %v (err %v), want Unauthenticated", got, err)
		}
	})
}

// TestRevokeCompetitionAdmin_HostRemovesTheAuthority proves the revoke
// actually removes the assignment rather than merely reporting success — the
// T3 lesson (prove the slot is freed, not that a field flipped), applied here
// by re-assigning the same user afterwards: that only succeeds if the row is
// genuinely gone, since the composite primary key would otherwise reject it.
func TestRevokeCompetitionAdmin_HostRemovesTheAuthority(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandler()
	c := seedCompetition(t, h, caHostID, 16, 2)
	req := &competitionsv1.AssignCompetitionAdminRequest{CompetitionId: c.GetId(), UserId: caAdminID}

	if _, err := h.AssignCompetitionAdmin(ctxAs(caHostID), req); err != nil {
		t.Fatalf("seed AssignCompetitionAdmin: %v", err)
	}
	if _, err := h.RevokeCompetitionAdmin(ctxAs(caHostID), &competitionsv1.RevokeCompetitionAdminRequest{
		CompetitionId: c.GetId(),
		UserId:        caAdminID,
	}); err != nil {
		t.Fatalf("RevokeCompetitionAdmin() unexpected error: %v", err)
	}
	if _, err := h.AssignCompetitionAdmin(ctxAs(caHostID), req); err != nil {
		t.Fatalf("re-assigning after revoke failed with %v — the revoke reported success without removing the row", err)
	}
}
