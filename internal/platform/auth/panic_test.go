package auth_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// T13.5 / #135 — what a panicking TokenVerifier does to a request.
//
// The decision this file encodes is **fail-closed, locally**: the auth
// interceptor recovers a panic raised inside a verifier and turns it into a
// denial, rather than letting it propagate to grpcrecovery (which also denies,
// but only because grpcrecovery happens to be registered in front) and rather
// than swallowing it and continuing with no principal (which would be
// fail-open on any RPC not covered by the enforcement list).
//
// The load-bearing assertion in every test here is `handlerRan == false`. A
// panic means the verifier reached no conclusion — the request's auth state is
// *unknown*, which is not the same as "unauthenticated" and emphatically not
// the same as "no auth required". A request whose credentials could not be
// evaluated is not served.
//
// The code is Internal, not Unauthenticated: the caller did nothing wrong, and
// telling a fleet of clients their tokens are bad because of a bug on our side
// sends them into a re-authentication loop during an incident. That is the
// same reasoning ADR-0013 §5 already applies to ErrKeyUnavailable. Internal
// rather than Unavailable because a panic is a deterministic bug, not a
// transient outage, and Unavailable is the code gRPC clients retry on — an
// always-panicking verifier plus a retryable code is a self-inflicted storm.

const panicToken = "make-the-verifier-panic"

// bearerCtx builds an incoming-metadata context carrying token as a bearer
// credential, which is the only way to make the interceptor call the verifier
// at all.
func bearerCtx(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestUnaryInterceptorDeniesWhenVerifierPanics is the fail-closed proof with
// nothing else in the chain: no grpcrecovery, no enforcement interceptor.
// Before this ticket the panic escaped this call and killed the test binary.
func TestUnaryInterceptorDeniesWhenVerifierPanics(t *testing.T) {
	t.Parallel()

	interceptor := auth.UnaryInterceptor(panickingVerifier{trigger: panicToken}, discardLogger())

	var handlerRan bool
	resp, err := interceptor(bearerCtx(panicToken), "request",
		&grpc.UnaryServerInfo{FullMethod: authedMethod},
		func(context.Context, any) (any, error) {
			handlerRan = true
			return "response", nil
		})

	if handlerRan {
		t.Error("handler ran; a request whose verifier panicked must never reach a handler — that is fail-open")
	}
	if resp != nil {
		t.Errorf("response = %v, want nil", resp)
	}
	if err == nil {
		t.Fatal("interceptor returned nil error after a verifier panic; want a denial")
	}
	if got := status.Code(err); got != codes.Internal {
		t.Errorf("status code = %v, want %v", got, codes.Internal)
	}
}

// TestStreamInterceptorDeniesWhenVerifierPanics is the streaming counterpart.
// This repo has no streaming RPCs yet; the interceptor exists so the first one
// anyone adds does not silently behave differently, and that guarantee is
// worth exactly as much as its test.
func TestStreamInterceptorDeniesWhenVerifierPanics(t *testing.T) {
	t.Parallel()

	interceptor := auth.StreamInterceptor(panickingVerifier{trigger: panicToken}, discardLogger())

	var handlerRan bool
	err := interceptor(nil, fakeServerStream{ctx: bearerCtx(panicToken)},
		&grpc.StreamServerInfo{FullMethod: streamMethod},
		func(any, grpc.ServerStream) error {
			handlerRan = true
			return nil
		})

	if handlerRan {
		t.Error("stream handler ran; a stream whose verifier panicked must never reach a handler")
	}
	if err == nil {
		t.Fatal("interceptor returned nil error after a verifier panic; want a denial")
	}
	if got := status.Code(err); got != codes.Internal {
		t.Errorf("status code = %v, want %v", got, codes.Internal)
	}
}

// TestInterceptorSurvivesRepeatedVerifierPanics guards the other half of
// fail-closed: denying is only correct if the process is still there to deny
// the next request too. A recover() that leaked state, or a panic that escaped
// on the second call, would show up here rather than in production.
//
// It also pins that a panic on one caller's token does not affect the next
// caller's: the second request carries no token, so it is resolved normally.
func TestInterceptorSurvivesRepeatedVerifierPanics(t *testing.T) {
	t.Parallel()

	interceptor := auth.UnaryInterceptor(panickingVerifier{trigger: panicToken}, discardLogger())
	info := &grpc.UnaryServerInfo{FullMethod: authedMethod}
	handler := func(context.Context, any) (any, error) { return "response", nil }

	for i := 0; i < 3; i++ {
		if _, err := interceptor(bearerCtx(panicToken), "request", info, handler); err == nil {
			t.Fatalf("call %d: want a denial on every panicking call, got nil", i)
		}
	}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unrelated caller error = %v; one caller's panic must not deny another's request", err)
	}
	if resp != "response" {
		t.Errorf("response = %v, want the handler's own response", resp)
	}
}

// TestVerifierPanicDoesNotAdmitAnEnforcedCall is the same property observed
// where it actually matters — through the real interceptor chain cmd/server
// installs, enforcement included.
//
// It exists because the two failure modes are indistinguishable from inside
// the auth interceptor alone: "denied by auth" and "admitted by auth, then
// denied by enforcement" both look like an error at this level. Only a chain
// test can show that the request never became a *served* one, and that the
// answer does not depend on which other interceptors happen to be registered.
func TestVerifierPanicDoesNotAdmitAnEnforcedCall(t *testing.T) {
	t.Parallel()

	set := auth.NewMethodSet([]string{authedMethod})
	observe := auth.UnaryInterceptor(panickingVerifier{trigger: panicToken}, discardLogger())
	enforce := auth.RequireUnaryInterceptor(set)

	var handlerRan bool
	handler := func(context.Context, any) (any, error) {
		handlerRan = true
		return "response", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: authedMethod}

	// observe wraps enforce wraps handler — the order cmd/server registers.
	_, err := observe(bearerCtx(panicToken), "request", info,
		func(ctx context.Context, req any) (any, error) {
			return enforce(ctx, req, info, handler)
		})

	if handlerRan {
		t.Error("handler ran for an enforced method after a verifier panic; the call was admitted")
	}
	if err == nil {
		t.Fatal("enforced call succeeded after a verifier panic; want a denial")
	}
}

// TestVerifierPanicDeniedWithNoRecoveryInterceptor is the structural claim, at
// the real wire boundary: fail-closed no longer depends on what else is in the
// chain.
//
// The server here is built with the auth interceptors and *nothing else* — no
// grpcrecovery. Before this ticket that configuration was fatal: the panic
// unwound past grpc, which installs no recover() of its own, and took the
// process with it, so the same test crashed this binary rather than failing.
// The old arrangement's safety was real but borrowed, resting on a chain order
// written in cmd/server and defended only by a comment.
//
// This does not argue for dropping grpcrecovery, which protects the handlers.
// It argues that the auth interceptor should not need it to be safe.
func TestVerifierPanicDeniedWithNoRecoveryInterceptor(t *testing.T) {
	t.Parallel()

	logger := discardLogger()
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(auth.UnaryInterceptor(panickingVerifier{trigger: panicToken}, logger)),
		grpc.ChainStreamInterceptor(auth.StreamInterceptor(panickingVerifier{trigger: panicToken}, logger)),
	)
	f := serveFixture(t, srv, nil)

	t.Run("unary", func(t *testing.T) {
		if _, err := f.whoAmI(t, panicToken); err == nil {
			t.Fatal("WhoAmI succeeded; a verifier panic must deny the request")
		} else if got := status.Code(err); got != codes.Internal {
			t.Errorf("status code = %v, want %v", got, codes.Internal)
		}
	})

	t.Run("stream", func(t *testing.T) {
		if _, err := f.whoAmIStream(t, panicToken); err == nil {
			t.Fatal("WhoAmIStream succeeded; a verifier panic must deny the request")
		} else if got := status.Code(err); got != codes.Internal {
			t.Errorf("status code = %v, want %v", got, codes.Internal)
		}
	})

	// Reaching this line at all is the load-bearing part: with no recovery
	// interceptor registered, an escaping panic would have ended the process.
	t.Run("server survives and still serves other callers", func(t *testing.T) {
		got, err := f.whoAmI(t, "")
		if err != nil {
			t.Fatalf("post-panic WhoAmI error = %v; the server did not survive", err)
		}
		if got != "anonymous" {
			t.Errorf("subject = %q, want %q", got, "anonymous")
		}
	})
}

// TestVerifierPanicIsNotACallerFault pins ErrVerifierPanicked's place in the
// error vocabulary. IsTokenRejection is what every enforcement path asks to
// decide between "your credentials are bad" and "we are broken", and a
// verifier crash is emphatically the second — the same call ADR-0013 §5 makes
// for ErrKeyUnavailable. Get this wrong and a crashing verifier tells an
// entire fleet of clients to go re-authenticate during an incident.
func TestVerifierPanicIsNotACallerFault(t *testing.T) {
	t.Parallel()

	if auth.IsTokenRejection(auth.ErrVerifierPanicked) {
		t.Error("IsTokenRejection(ErrVerifierPanicked) = true, want false — a crash on our side is not the caller's token being bad")
	}
}

// fakeServerStream is the minimum grpc.ServerStream the stream interceptor
// touches: it reads Context() and wraps the stream. Every other method panics
// if called, which would be a real finding rather than test noise.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s fakeServerStream) Context() context.Context { return s.ctx }
