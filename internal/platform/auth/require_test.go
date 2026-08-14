package auth_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// T12.7 — the enforcement half of this package. T12.2 shipped the
// observe-only interceptors; these tests pin the behavior that turns the
// resolved Principal into an actual boundary, per ADR-0013 §4 ("Enforcement is
// turned on per-RPC by T12.7/T12.8/T12.9, each context exporting its own
// AuthenticatedMethods()").
//
// The two gRPC codes must stay distinct (ADR-0013 §5), so every assertion here
// names the exact code rather than merely "an error".

const (
	publicMethod = "/pickleball.booking.v1.BookingService/GetQuote"
	authedMethod = "/pickleball.booking.v1.BookingService/CreateDiscountRule"
)

// okHandler records whether the request reached the handler at all. That is
// the property under test: an interceptor that returns the right code but
// still runs the handler has enforced nothing.
func okHandler(reached *bool) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		*reached = true
		return "ok", nil
	}
}

func TestMethodSet_RequiresOnlyListedMethods(t *testing.T) {
	set := auth.NewMethodSet([]string{authedMethod}, []string{publicMethod + "X"})

	if !set.Requires(authedMethod) {
		t.Errorf("Requires(%q) = false, want true — a listed method must require a principal", authedMethod)
	}
	if set.Requires(publicMethod) {
		t.Errorf("Requires(%q) = true, want false — an unlisted method must stay public", publicMethod)
	}
}

// TestNewMethodSet_ComposesEveryContext is the cmd/server composition contract
// (A11 Ruling 2): each context contributes its own list and every one of them
// must survive composition. A composition that silently dropped a context's
// list would disable that context's enforcement with no failing test anywhere.
func TestNewMethodSet_ComposesEveryContext(t *testing.T) {
	booking := []string{"/pickleball.booking.v1.BookingService/CreateDiscountRule"}
	facilities := []string{"/pickleball.facilities.v1.FacilitiesService/AddCourt"}
	identity := []string{"/pickleball.identity.v1.IdentityService/CreateUser"}

	set := auth.NewMethodSet(booking, facilities, identity)

	for _, m := range append(append([]string{}, booking...), append(facilities, identity...)...) {
		if !set.Requires(m) {
			t.Errorf("Requires(%q) = false after composition, want true", m)
		}
	}
}

func TestRequireUnaryInterceptor_UnauthenticatedWithoutPrincipal(t *testing.T) {
	set := auth.NewMethodSet([]string{authedMethod})
	interceptor := auth.RequireUnaryInterceptor(set)

	reached := false
	_, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: authedMethod}, okHandler(&reached))

	if err == nil {
		t.Fatal("an authenticated method with no principal succeeded — enforcement is off")
	}
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated (%q means 'I do not know who you are', ADR-0013 §5)", got, codes.Unauthenticated)
	}
	if reached {
		t.Error("the handler ran anyway — the interceptor rejected but did not block")
	}
}

func TestRequireUnaryInterceptor_AllowsAuthenticatedCaller(t *testing.T) {
	set := auth.NewMethodSet([]string{authedMethod})
	interceptor := auth.RequireUnaryInterceptor(set)

	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "auth0|owner-1"})
	reached := false
	_, err := interceptor(ctx, nil,
		&grpc.UnaryServerInfo{FullMethod: authedMethod}, okHandler(&reached))

	if err != nil {
		t.Fatalf("a verified caller was rejected: %v", err)
	}
	if !reached {
		t.Error("the handler did not run for a verified caller")
	}
}

// TestRequireUnaryInterceptor_LeavesPublicMethodsAlone is the regression guard
// for the failure the T12.7 ticket names explicitly: silently authenticating a
// currently-public browse path would break a shipped flow.
func TestRequireUnaryInterceptor_LeavesPublicMethodsAlone(t *testing.T) {
	set := auth.NewMethodSet([]string{authedMethod})
	interceptor := auth.RequireUnaryInterceptor(set)

	reached := false
	_, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: publicMethod}, okHandler(&reached))

	if err != nil {
		t.Fatalf("public method %q was rejected for an anonymous caller: %v", publicMethod, err)
	}
	if !reached {
		t.Errorf("public method %q did not reach its handler", publicMethod)
	}
}

// TestRequireUnaryInterceptor_NilInfoIsRejected covers the defensive path: a
// nil UnaryServerInfo means the interceptor cannot tell which method is being
// called, and "cannot tell" must fail closed rather than wave the call
// through. The observe-only interceptor tolerates a nil info because it only
// logs; this one decides.
func TestRequireUnaryInterceptor_NilInfoIsRejected(t *testing.T) {
	interceptor := auth.RequireUnaryInterceptor(auth.NewMethodSet([]string{authedMethod}))

	reached := false
	_, err := interceptor(context.Background(), nil, nil, okHandler(&reached))

	if err == nil || status.Code(err) != codes.Unauthenticated {
		t.Errorf("nil info: err = %v (code %v), want Unauthenticated — an undeterminable method must fail closed", err, status.Code(err))
	}
	if reached {
		t.Error("the handler ran for an undeterminable method")
	}
}
