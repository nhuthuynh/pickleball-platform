// T12.9 — tests for the enforcement interceptor that consumes each
// context's AuthenticatedMethods() list (A11 Ruling 2).
//
// The properties worth pinning here are the ones a mistake would make
// silent: that an unlisted method stays reachable exactly as before (a
// regression here breaks every public read on the platform), that a listed
// one is refused with Unauthenticated rather than PermissionDenied, and
// that the handler is not invoked at all on refusal.
package auth_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

const (
	protectedMethod = "/pickleball.identity.v1.IdentityService/CreateUser"
	publicMethod    = "/pickleball.identity.v1.IdentityService/GetUser"
)

// countingHandler records whether the RPC body ever ran. Asserting on the
// error alone would not distinguish "refused before the handler" from
// "handler ran, did the work, and then something returned an error" — and
// for an enforcement check only the former is acceptable.
type countingHandler struct{ calls int }

func (h *countingHandler) unary(context.Context, any) (any, error) {
	h.calls++
	return "ok", nil
}

func TestRequireAuthentication(t *testing.T) {
	t.Parallel()

	authenticated := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "auth0|ada"})

	tests := []struct {
		name        string
		methods     []string
		fullMethod  string
		ctx         context.Context
		wantCode    codes.Code
		wantHandler int
	}{
		{
			name:        "listed method with no principal is refused before the handler",
			methods:     []string{protectedMethod},
			fullMethod:  protectedMethod,
			ctx:         context.Background(),
			wantCode:    codes.Unauthenticated,
			wantHandler: 0,
		},
		{
			name:        "listed method with a principal proceeds",
			methods:     []string{protectedMethod},
			fullMethod:  protectedMethod,
			ctx:         authenticated,
			wantCode:    codes.OK,
			wantHandler: 1,
		},
		{
			// The regression that would matter most: enforcing on
			// something nobody listed would break every public read on
			// the platform at once.
			name:        "unlisted method is untouched with no principal",
			methods:     []string{protectedMethod},
			fullMethod:  publicMethod,
			ctx:         context.Background(),
			wantCode:    codes.OK,
			wantHandler: 1,
		},
		{
			name:        "empty policy enforces nothing",
			methods:     nil,
			fullMethod:  protectedMethod,
			ctx:         context.Background(),
			wantCode:    codes.OK,
			wantHandler: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := &countingHandler{}
			interceptor := auth.RequireAuthentication(tc.methods)

			_, err := interceptor(tc.ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.fullMethod}, handler.unary)

			if got := status.Code(err); got != tc.wantCode {
				t.Fatalf("status code = %v, want %v (err: %v)", got, tc.wantCode, err)
			}
			if handler.calls != tc.wantHandler {
				t.Fatalf("handler ran %d times, want %d", handler.calls, tc.wantHandler)
			}
		})
	}
}

// TestRequireAuthentication_NeverPermissionDenied guards the distinction
// ADR-0013 §5 is deliberately pedantic about. An absent principal means "I
// do not know who you are", and answering PermissionDenied instead would
// tell a caller they were identified and refused — the classic auth bug,
// and one that also misleads a client into not retrying with credentials.
func TestRequireAuthentication_NeverPermissionDenied(t *testing.T) {
	t.Parallel()

	handler := &countingHandler{}
	interceptor := auth.RequireAuthentication([]string{protectedMethod})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: protectedMethod}, handler.unary)
	if err == nil {
		t.Fatal("expected an error for a principal-less call to a protected method")
	}
	if status.Code(err) == codes.PermissionDenied {
		t.Fatal("got PermissionDenied for an unidentified caller, want Unauthenticated")
	}
}

// TestRequireAuthentication_ZeroSubjectPrincipalDoesNotSatisfy is the
// belt-and-braces companion to auth.ContextWithPrincipal's own refusal to
// store a zero-Subject principal. If that guard were ever relaxed, an empty
// principal must still not count as a verified caller here — an
// authenticated-looking request whose subject is "" is exactly what would
// later match an empty owner column and succeed.
func TestRequireAuthentication_ZeroSubjectPrincipalDoesNotSatisfy(t *testing.T) {
	t.Parallel()

	// ContextWithPrincipal drops this, so the context carries no principal.
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{})
	if _, ok := auth.PrincipalFromContext(ctx); ok {
		t.Fatal("a zero-Subject principal was stored in the context")
	}

	handler := &countingHandler{}
	interceptor := auth.RequireAuthentication([]string{protectedMethod})

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: protectedMethod}, handler.unary)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
	if handler.calls != 0 {
		t.Fatalf("handler ran %d times, want 0", handler.calls)
	}
}

// TestRequireAuthenticationStream mirrors the unary cases for the streaming
// form. This repo has no streaming RPCs yet; the interceptor exists so the
// first one added does not silently enforce nothing, and this test is what
// keeps the two forms from drifting apart.
func TestRequireAuthenticationStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fullMethod string
		ctx        context.Context
		wantCode   codes.Code
		wantCalls  int
	}{
		{"listed method with no principal is refused", protectedMethod, context.Background(), codes.Unauthenticated, 0},
		{"listed method with a principal proceeds", protectedMethod, auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "auth0|ada"}), codes.OK, 1},
		{"unlisted method is untouched", publicMethod, context.Background(), codes.OK, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			handler := func(any, grpc.ServerStream) error {
				calls++
				return nil
			}

			interceptor := auth.RequireAuthenticationStream([]string{protectedMethod})
			err := interceptor(nil, stubStream{ctx: tc.ctx}, &grpc.StreamServerInfo{FullMethod: tc.fullMethod}, handler)

			if got := status.Code(err); got != tc.wantCode {
				t.Fatalf("status code = %v, want %v (err: %v)", got, tc.wantCode, err)
			}
			if calls != tc.wantCalls {
				t.Fatalf("handler ran %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

// TestRequireAuthentication_NilInfoIsTolerated mirrors the nil-tolerance
// UnaryInterceptor and grpcrecovery already have: a wiring mistake should
// degrade, not panic on the request path. With no method name there is
// nothing to match against the policy, so the request proceeds.
func TestRequireAuthentication_NilInfoIsTolerated(t *testing.T) {
	t.Parallel()

	handler := &countingHandler{}
	interceptor := auth.RequireAuthentication([]string{protectedMethod})

	if _, err := interceptor(context.Background(), nil, nil, handler.unary); err != nil {
		t.Fatalf("unexpected error with nil info: %v", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler ran %d times, want 1", handler.calls)
	}

	streamCalls := 0
	streamInterceptor := auth.RequireAuthenticationStream([]string{protectedMethod})
	err := streamInterceptor(nil, stubStream{ctx: context.Background()}, nil, func(any, grpc.ServerStream) error {
		streamCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected stream error with nil info: %v", err)
	}
	if streamCalls != 1 {
		t.Fatalf("stream handler ran %d times, want 1", streamCalls)
	}
}

// TestRequireAuthentication_ErrorIsAGRPCStatus makes sure a client can map
// the refusal cleanly rather than receiving an opaque error that surfaces
// as Unknown/500.
func TestRequireAuthentication_ErrorIsAGRPCStatus(t *testing.T) {
	t.Parallel()

	handler := &countingHandler{}
	interceptor := auth.RequireAuthentication([]string{protectedMethod})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: protectedMethod}, handler.unary)
	if _, ok := status.FromError(err); !ok {
		t.Fatalf("error is not a gRPC status: %v", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("unexpected error identity")
	}
}

// stubStream is a grpc.ServerStream that carries only a context — all this
// interceptor reads.
type stubStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s stubStream) Context() context.Context { return s.ctx }
