package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// stubVerifier resolves exactly one token string and rejects everything else.
// The auth package's own tests deliberately do not use the real RS256
// verifier: what is under test here is the interceptor's observe-only
// contract, which must hold for *any* TokenVerifier. chain_test.go covers the
// two wired together against real key material.
type stubVerifier struct {
	token     string
	principal auth.Principal
	err       error
	calls     int
}

func (s *stubVerifier) Verify(_ context.Context, raw string) (auth.Principal, error) {
	s.calls++
	if s.err != nil {
		return auth.Principal{}, s.err
	}
	if raw != s.token {
		return auth.Principal{}, auth.ErrTokenSignature
	}
	return s.principal, nil
}

var testPrincipal = auth.Principal{
	Subject:   "auth0|abc123",
	Issuer:    "https://pickleball-test.example.com/",
	Audience:  []string{"https://api.pickleball-test.example.com"},
	Scopes:    []string{"bookings:read"},
	ExpiresAt: time.Date(2026, 8, 14, 12, 10, 0, 0, time.UTC),
}

func TestPrincipalContextRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("absent by default", func(t *testing.T) {
		t.Parallel()

		if p, ok := auth.PrincipalFromContext(context.Background()); ok {
			t.Errorf("PrincipalFromContext(background) = %+v, true; want zero, false", p)
		}
	})

	t.Run("round-trips", func(t *testing.T) {
		t.Parallel()

		ctx := auth.ContextWithPrincipal(context.Background(), testPrincipal)
		got, ok := auth.PrincipalFromContext(ctx)
		if !ok {
			t.Fatal("PrincipalFromContext = _, false; want true")
		}
		if got.Subject != testPrincipal.Subject {
			t.Errorf("Subject = %q, want %q", got.Subject, testPrincipal.Subject)
		}
	})

	// A Principal with no Subject is not a caller identity — it is the zero
	// value wearing a hat. Storing one would let a later ownership check
	// compare "" against an empty owner column and succeed.
	t.Run("a subject-less principal is never stored", func(t *testing.T) {
		t.Parallel()

		ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Issuer: "x"})
		if p, ok := auth.PrincipalFromContext(ctx); ok {
			t.Errorf("PrincipalFromContext = %+v, true; want the subject-less principal to be refused", p)
		}
	})
}

func TestHasScope(t *testing.T) {
	t.Parallel()

	p := auth.Principal{Subject: "s", Scopes: []string{"bookings:read", "payments:write"}}

	if !p.HasScope("payments:write") {
		t.Error("HasScope(payments:write) = false, want true")
	}
	if p.HasScope("payments:read") {
		t.Error("HasScope(payments:read) = true, want false")
	}
	if (auth.Principal{}).HasScope("anything") {
		t.Error("zero Principal HasScope = true, want false")
	}
}

func TestIsTokenRejection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"expired", auth.ErrTokenExpired, true},
		{"bad signature", auth.ErrTokenSignature, true},
		{"wrong issuer", auth.ErrTokenIssuer, true},
		{"wrong audience", auth.ErrTokenAudience, true},
		{"malformed", auth.ErrTokenMalformed, true},
		{"not yet valid", auth.ErrTokenNotYetValid, true},
		{"missing claim", auth.ErrTokenClaimMissing, true},
		{"no token", auth.ErrNoToken, true},
		{"wrapped rejection", errors.Join(errors.New("context"), auth.ErrTokenExpired), true},
		// The one that matters: an unreachable key source is our outage, and
		// must not be reported to the caller as "your token is bad".
		{"key unavailable", auth.ErrKeyUnavailable, false},
		{"unrelated error", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := auth.IsTokenRejection(tt.err); got != tt.want {
				t.Errorf("IsTokenRejection(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestUnaryInterceptorIsObserveOnly is the acceptance criterion of T12.2 in
// test form: for every possible token outcome the handler still runs and the
// caller still gets the handler's own result. The interceptor may add a
// principal; it may never subtract a response.
func TestUnaryInterceptorIsObserveOnly(t *testing.T) {
	t.Parallel()

	const goodToken = "good-token"

	tests := []struct {
		name          string
		md            metadata.MD
		verifierErr   error
		wantPrincipal bool
		wantVerified  bool // did the interceptor bother calling the verifier?
	}{
		{
			name:         "no metadata at all",
			md:           nil,
			wantVerified: false,
		},
		{
			name:         "metadata without an authorization header",
			md:           metadata.Pairs("x-request-id", "abc"),
			wantVerified: false,
		},
		{
			name:          "valid bearer token",
			md:            metadata.Pairs("authorization", "Bearer "+goodToken),
			wantPrincipal: true,
			wantVerified:  true,
		},
		{
			name:          "scheme match is case-insensitive",
			md:            metadata.Pairs("authorization", "bearer "+goodToken),
			wantPrincipal: true,
			wantVerified:  true,
		},
		{
			name:         "non-bearer scheme is ignored",
			md:           metadata.Pairs("authorization", "Basic dXNlcjpwYXNz"),
			wantVerified: false,
		},
		{
			name:         "bearer with no token value",
			md:           metadata.Pairs("authorization", "Bearer"),
			wantVerified: false,
		},
		{
			name:         "bearer with only whitespace",
			md:           metadata.Pairs("authorization", "Bearer    "),
			wantVerified: false,
		},
		{
			name:         "invalid token",
			md:           metadata.Pairs("authorization", "Bearer not-the-good-token"),
			wantVerified: true,
		},
		{
			name:         "verifier reports an infrastructure failure",
			md:           metadata.Pairs("authorization", "Bearer "+goodToken),
			verifierErr:  auth.ErrKeyUnavailable,
			wantVerified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := &stubVerifier{token: goodToken, principal: testPrincipal, err: tt.verifierErr}
			interceptor := auth.UnaryInterceptor(v, nil)

			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			var (
				handlerRan   bool
				gotPrincipal bool
			)
			resp, err := interceptor(ctx, "request",
				&grpc.UnaryServerInfo{FullMethod: "/pickleball.booking.v1.BookingService/CreateBooking"},
				func(ctx context.Context, req any) (any, error) {
					handlerRan = true
					_, gotPrincipal = auth.PrincipalFromContext(ctx)
					return "response", nil
				})

			// Observe-only: no outcome of verification may ever surface as an
			// error or suppress the handler.
			if err != nil {
				t.Fatalf("interceptor returned error %v; observe-only mode must never fail a request", err)
			}
			if !handlerRan {
				t.Fatal("handler was not called; observe-only mode must never short-circuit")
			}
			if resp != "response" {
				t.Errorf("response = %v, want the handler's own response untouched", resp)
			}
			if gotPrincipal != tt.wantPrincipal {
				t.Errorf("principal in handler context = %v, want %v", gotPrincipal, tt.wantPrincipal)
			}
			if verified := v.calls > 0; verified != tt.wantVerified {
				t.Errorf("verifier called = %v, want %v", verified, tt.wantVerified)
			}
		})
	}
}

// TestUnaryInterceptorPreservesHandlerError proves the interceptor is
// transparent in the failure direction too: a handler's own error reaches the
// caller unchanged rather than being masked by anything auth does.
func TestUnaryInterceptorPreservesHandlerError(t *testing.T) {
	t.Parallel()

	want := errors.New("handler said no")
	interceptor := auth.UnaryInterceptor(&stubVerifier{token: "t"}, nil)

	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/svc/M"},
		func(context.Context, any) (any, error) { return nil, want })

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want the handler's own error", err)
	}
}

// TestInterceptorsTolerateNilVerifier mirrors grpcrecovery's nil-logger
// tolerance: a wiring mistake in cmd/server must degrade to "no principal",
// which is exactly today's behavior, rather than panicking on the request
// path. Note the ADR: once enforcement lands this must instead be a startup
// failure, because a nil verifier under enforcement is fail-open.
func TestInterceptorsTolerateNilVerifier(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer anything"))

	t.Run("unary", func(t *testing.T) {
		t.Parallel()

		ran := false
		_, err := auth.UnaryInterceptor(nil, nil)(ctx, "req", &grpc.UnaryServerInfo{FullMethod: "/svc/M"},
			func(ctx context.Context, req any) (any, error) {
				ran = true
				if _, ok := auth.PrincipalFromContext(ctx); ok {
					t.Error("a nil verifier produced a principal")
				}
				return "ok", nil
			})
		if err != nil || !ran {
			t.Fatalf("err = %v, handlerRan = %v; want nil, true", err, ran)
		}
	})

	t.Run("stream", func(t *testing.T) {
		t.Parallel()

		ran := false
		err := auth.StreamInterceptor(nil, nil)(nil, &fakeStream{ctx: ctx},
			&grpc.StreamServerInfo{FullMethod: "/svc/S"},
			func(srv any, ss grpc.ServerStream) error {
				ran = true
				if _, ok := auth.PrincipalFromContext(ss.Context()); ok {
					t.Error("a nil verifier produced a principal")
				}
				return nil
			})
		if err != nil || !ran {
			t.Fatalf("err = %v, handlerRan = %v; want nil, true", err, ran)
		}
	})
}

// TestStreamInterceptorIsObserveOnly is the streaming half. It exists for the
// same reason grpcrecovery.StreamInterceptor does (recovery.go:93-100):
// grpc.NewServer takes unary and stream interceptors as separate options, so
// registering only the unary form leaves the first streaming RPC anyone adds
// with no principal at all — and under later enforcement, that is an
// unauthenticated hole rather than a missing convenience.
func TestStreamInterceptorIsObserveOnly(t *testing.T) {
	t.Parallel()

	const goodToken = "good-token"

	tests := []struct {
		name          string
		md            metadata.MD
		wantPrincipal bool
	}{
		{name: "no token", md: nil},
		{name: "valid token", md: metadata.Pairs("authorization", "Bearer "+goodToken), wantPrincipal: true},
		{name: "invalid token", md: metadata.Pairs("authorization", "Bearer nope")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			interceptor := auth.StreamInterceptor(&stubVerifier{token: goodToken, principal: testPrincipal}, nil)

			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			var handlerRan, gotPrincipal bool
			err := interceptor(nil, &fakeStream{ctx: ctx},
				&grpc.StreamServerInfo{FullMethod: "/pickleball.booking.v1.BookingService/Watch"},
				func(srv any, ss grpc.ServerStream) error {
					handlerRan = true
					_, gotPrincipal = auth.PrincipalFromContext(ss.Context())
					return nil
				})

			if err != nil {
				t.Fatalf("interceptor returned error %v; observe-only mode must never fail a stream", err)
			}
			if !handlerRan {
				t.Fatal("stream handler was not called")
			}
			if gotPrincipal != tt.wantPrincipal {
				t.Errorf("principal in stream context = %v, want %v", gotPrincipal, tt.wantPrincipal)
			}
		})
	}
}

// fakeStream is the minimum grpc.ServerStream needed to exercise the stream
// interceptor's context substitution.
type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }
