// GetCurrentUser: resolving the verified token subject to this platform's
// User (#344).
//
// The defect this closes: CompleteProfile.vue was the only place the web
// client ever learned its own user id, because the token carries the
// identity provider's `sub` and not a `users.id`. A browser without that id
// stored could not obtain it — `POST /v1/users` answers 409 because the
// account already exists — so clearing site data locked a player out of
// their own profile. Nothing was mis-attributed (the server has always
// resolved the actor from the bearer subject), but the Profile screen
// dead-ended.
//
// These tests share newTestHandler/fakeRepo/principalCtx/seedUser with
// authz_regression_test.go in this package.
package grpcapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	identityv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/identity/v1"
)

func TestGetCurrentUser_ResolvesTheVerifiedSubject(t *testing.T) {
	h, _ := newTestHandler()
	seeded := seedUser(t, h)

	resp, err := h.GetCurrentUser(principalCtx(fixtureSubject), &identityv1.GetCurrentUserRequest{})
	if err != nil {
		t.Fatalf("GetCurrentUser: unexpected error: %v", err)
	}
	if got, want := resp.GetUser().GetId(), seeded.GetId(); got != want {
		t.Errorf("resolved the wrong user: got id %q, want %q", got, want)
	}
	if got, want := resp.GetUser().GetDisplayName(), seeded.GetDisplayName(); got != want {
		t.Errorf("display name: got %q, want %q", got, want)
	}
}

// No principal is Unauthenticated — "I do not know who you are" — never
// PermissionDenied, which would claim we knew the caller and were refusing
// them. Same rule the rest of this handler follows (ADR-0013 §5).
func TestGetCurrentUser_NoPrincipalIsUnauthenticated(t *testing.T) {
	h, _ := newTestHandler()

	_, err := h.GetCurrentUser(context.Background(), &identityv1.GetCurrentUserRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("got %v (%v), want Unauthenticated", status.Code(err), err)
	}
}

// An authenticated caller with no account yet is NotFound, not an empty
// success. That distinction is the whole point for the client: a 404 here
// means "finish signing up", which is a different remedy from 401's "sign
// in again" (#340's banner) — and an empty 200 would make the two
// indistinguishable.
func TestGetCurrentUser_AuthenticatedButUnregisteredIsNotFound(t *testing.T) {
	h, _ := newTestHandler()

	_, err := h.GetCurrentUser(principalCtx("auth0|nobody"), &identityv1.GetCurrentUserRequest{})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("got %v (%v), want NotFound", status.Code(err), err)
	}
}

// A caller cannot reach another player's profile through this RPC, because
// the request has no field in which to name one. This test exists to fail
// if a `user_id` is ever added to GetCurrentUserRequest: it registers a
// second user and asserts the first caller still resolves to themselves.
func TestGetCurrentUser_CannotReachAnotherUser(t *testing.T) {
	h, repo := newTestHandler()
	mine := seedUser(t, h)
	other := registerOther(t, repo, "auth0|someone-else")

	resp, err := h.GetCurrentUser(principalCtx(fixtureSubject), &identityv1.GetCurrentUserRequest{})
	if err != nil {
		t.Fatalf("GetCurrentUser: unexpected error: %v", err)
	}
	if got := resp.GetUser().GetId(); got != mine.GetId() {
		t.Errorf("resolved id %q, want the caller's own %q", got, mine.GetId())
	}
	if got := resp.GetUser().GetId(); got == other.ID {
		t.Errorf("resolved ANOTHER user's id %q", got)
	}
}

// --- the REST route, through the real grpc-gateway mux --------------------

// `/v1/users/me` and `/v1/users/{user_id}` are this repository's first
// literal-vs-wildcard route pair, and the order that makes it work is the
// opposite of the obvious one. grpc-gateway's ServeMux PREPENDS each
// pattern as it is registered:
//
//	// runtime/mux.go, grpc-gateway v2.29.0
//	s.handlers[meth] = append([]handler{{pat: pat, h: h}}, s.handlers[meth]...)
//
// so the LAST pattern registered is the FIRST one matched, and registration
// follows the order of rpcs in identity.proto. GetCurrentUser is therefore
// declared AFTER GetUser. Declared before it — where it started, on the
// assumption that first-declared wins — `GET /v1/users/me` matched
// `/v1/users/{user_id}` with user_id="me" and answered 404, because a
// malformed id is answered exactly like an unknown one (T10.2 item 4).
//
// Mutation-verified: swapping the two rpcs back makes this test fail with
// exactly that 404.
//
// It drives runtime.ServeMux — the same type cmd/server wires — rather than
// asserting anything about the generated code's shape.
func TestRESTRoute_UsersMeReachesGetCurrentUser(t *testing.T) {
	h, _ := newTestHandler()
	seeded := seedUser(t, h)

	mux := runtime.NewServeMux()
	if err := identityv1.RegisterIdentityServiceHandlerServer(context.Background(), mux, h); err != nil {
		t.Fatalf("registering the gateway handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	// The gateway handler reads the principal from the request context,
	// exactly as auth.UnaryInterceptor leaves it in production.
	req = req.WithContext(principalCtx(fixtureSubject))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/users/me: got HTTP %d, want 200. Body: %s\n\n"+
			"A 404 here almost certainly means GetCurrentUser is declared "+
			"BEFORE GetUser in identity.proto. ServeMux.Handle prepends, so "+
			"the last rpc declared is matched first — GetCurrentUser must "+
			"come after GetUser, or this path matches /v1/users/{user_id} "+
			"with user_id=\"me\".", rec.Code, rec.Body.String())
	}

	var got identityv1.GetCurrentUserResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding the response body %q: %v", rec.Body.String(), err)
	}
	if got.GetUser().GetId() != seeded.GetId() {
		t.Errorf("resolved id %q, want %q", got.GetUser().GetId(), seeded.GetId())
	}
}

// The wildcard route still works, so the literal one above has not shadowed
// it. Without this, "fix" the ordering problem by deleting GetUser's route
// and the suite would stay green.
func TestRESTRoute_UsersByIDStillWorks(t *testing.T) {
	h, _ := newTestHandler()
	seeded := seedUser(t, h)

	mux := runtime.NewServeMux()
	if err := identityv1.RegisterIdentityServiceHandlerServer(context.Background(), mux, h); err != nil {
		t.Fatalf("registering the gateway handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/users/"+seeded.GetId(), nil)
	req = req.WithContext(principalCtx(fixtureSubject))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/users/{id}: got HTTP %d, want 200. Body: %s", rec.Code, rec.Body.String())
	}
}
