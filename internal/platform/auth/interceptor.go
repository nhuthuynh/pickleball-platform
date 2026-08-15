package auth

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// metadataKey is the gRPC metadata key carrying the bearer token.
//
// It is also the REST path's key: grpc-gateway's default header matcher
// forwards the HTTP `Authorization` header into incoming gRPC metadata under
// this exact (lowercased) name, so the one interceptor covers both the gRPC
// clients and every REST caller going through the gateway in cmd/server.
// gRPC metadata keys are always lowercase on the wire.
const metadataKey = "authorization"

// bearerScheme is compared case-insensitively: RFC 7235 §2.1 defines the auth
// scheme token as case-insensitive, and real clients send "Bearer", "bearer",
// and occasionally "BEARER".
const bearerScheme = "bearer"

// UnaryInterceptor returns a grpc.UnaryServerInterceptor that resolves the
// caller's bearer token into a Principal and puts it in the request context.
//
// # This interceptor resolves; it does not authorize
//
// It decides nothing about who may call what. If there is no token, or the
// token is expired, forged, malformed, or minted for another audience, the
// request proceeds to the handler with no principal and no error. Only a
// *successful* verification changes anything, and all it changes is that
// PrincipalFromContext starts returning a value. Turning that into a boundary
// is RequireUnaryInterceptor's job (require.go), plus each handler's own
// RequireSubject call.
//
// That separation is an acceptance criterion of the ticket that introduced
// this package, not a temporary shortcut. It means the entire path — client
// metadata, gateway header forwarding, key selection, signature and claim
// checking, context plumbing — was exercisable against a running platform
// before any handler's correctness depended on it.
//
// # The one thing it does decide: a verifier that panics (issue #135)
//
// There is exactly one case in which this interceptor refuses a request
// itself, and it is the case where it cannot honestly say anything else. If a
// TokenVerifier implementation panics, verification reached no conclusion: the
// request's authentication state is *unknown*, which is not the same as
// "unauthenticated" and emphatically not the same as "no authentication
// required". The panic is recovered here, logged at Error with its stack, and
// converted into codes.Internal without the handler ever running.
//
// The rejected alternative was recovering and continuing with no principal.
// That is fail-open: correct while nothing was enforced, and wrong the moment
// something is, because an RPC absent from the MethodSet — a new one nobody
// added, or one deliberately public — would then be served on the strength of
// a verifier that crashed. "We could not evaluate your credentials" must never
// be indistinguishable from "you needed none".
//
// The second rejected alternative was the status quo: let the panic propagate
// to grpcrecovery, which converts it to the same codes.Internal. The caller
// sees an identical result either way — this change is deliberately invisible
// on the wire — but the *guarantee* is not identical. Under the old
// arrangement the fail-closed property held only because recovery happened to
// be registered ahead of auth in cmd/server's chain; the existing ordering
// test proves it, and also proves that reversing the two kills the process.
// A security property that depends on the registration order of a slice
// literal in another package is a property waiting to be lost in a refactor.
// Recovering here makes this interceptor fail closed standing alone, in any
// chain, in any binary, including a test harness that forgot recovery.
// grpcrecovery stays registered outermost regardless: it protects the
// handlers, which is its own job.
//
// The cost, stated plainly: a verifier that panics on every token denies every
// request that presents one, including requests to public RPCs that needed no
// token at all. This project prefers that availability failure to the security
// failure on the other side. A panicking verifier is a bug either way; the
// question is only whether it is a bug that also admits strangers.
//
// # Ordering
//
// Still register this *after* grpcrecovery's interceptor in
// grpc.ChainUnaryInterceptor, and *before* RequireUnaryInterceptor. Recovery
// is outermost so it covers the handlers; this one must run before enforcement
// so there is a principal to enforce against.
//
// A nil verifier is tolerated here and means "resolve nothing", mirroring
// grpcrecovery's tolerance of a nil logger: a wiring mistake should degrade,
// not crash the request path. That tolerance is no longer the last word on the
// subject — a process that enforces any RPC and has no verifier now fails at
// startup, before this code is ever reached. See EnsureVerifierConfigured in
// startup.go, ADR-0013, and issue #136.
//
// A nil logger falls back to slog.Default().
func UnaryInterceptor(verifier TokenVerifier, logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := resolve(ctx, verifier, logger, unaryMethod(info))
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor is the streaming counterpart to UnaryInterceptor.
//
// This repo has no streaming RPCs today. It exists anyway for the reason
// grpcrecovery.StreamInterceptor spells out (recovery.go:93-100):
// grpc.NewServer applies unary and stream interceptors from separate options,
// so registering only the unary form leaves the first streaming method anyone
// adds behaving differently from every other RPC in the process. For recovery
// that gap was an unprotected crash path; here it would be an RPC that can
// never see a principal — and therefore, once the enforcement tickets land, an
// RPC whose authorization silently cannot work. Same trap, same fix: register
// both.
func StreamInterceptor(verifier TokenVerifier, logger *slog.Logger) grpc.StreamServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		fullMethod := "unknown"
		if info != nil {
			fullMethod = info.FullMethod
		}
		ctx, err := resolve(ss.Context(), verifier, logger, fullMethod)
		if err != nil {
			return err
		}
		return handler(srv, &principalStream{ServerStream: ss, ctx: ctx})
	}
}

// resolve is the shared body of both interceptors: extract, verify, and — only
// on success — enrich the context. Keeping it in one place is deliberate, for
// the same reason grpcrecovery shares its recovered(): the observe-only
// guarantee must hold identically for unary and streaming rather than being
// two copies that can drift apart.
//
// It returns the original context unchanged, and a nil error, on every failure
// path *except* a panicking verifier — the one case in which the caller must
// be refused rather than passed through. See the UnaryInterceptor doc comment
// for why that single exception exists and what it costs.
func resolve(ctx context.Context, verifier TokenVerifier, logger *slog.Logger, fullMethod string) (context.Context, error) {
	if verifier == nil {
		return ctx, nil
	}

	token, ok := bearerToken(ctx)
	if !ok {
		// Not logged. The overwhelming majority of requests to this platform's
		// public RPCs have no token, and logging each one would produce a line
		// per request for a condition that is entirely normal.
		return ctx, nil
	}

	principal, err := verifyRecovering(ctx, verifier, token, logger, fullMethod)
	if err != nil {
		if errors.Is(err, ErrVerifierPanicked) {
			// Already logged at Error, with the stack, by verifyRecovering.
			// Fail closed: the handler does not run.
			return ctx, errVerifierFailed
		}
		// Logged at warn, without the token: a rejected token is a signal an
		// operator needs, since the caller is told only "unauthenticated" and
		// enforcement tells RPC-absent-from-the-set callers nothing at all.
		// The raw token is a live credential and never goes to a log.
		logger.Warn("rejected bearer token (request proceeds with no principal)",
			"method", fullMethod,
			"error", err,
			"caller_fault", IsTokenRejection(err),
		)
		return ctx, nil
	}

	return ContextWithPrincipal(ctx, principal), nil
}

// verifyRecovering calls verifier.Verify, converting a panic into
// ErrVerifierPanicked so the request path has an ordinary error to handle
// instead of an unwinding goroutine.
//
// The logging happens here rather than in resolve for the reason
// grpcrecovery's UnaryInterceptor spells out: runtime/debug.Stack() is only
// the *panicking* stack while called from inside the deferred recover. Capture
// it one frame later and the operator gets a trace pointing at this package
// instead of at the broken verifier, which is the one thing they need.
//
// The named return values are what let the deferred function substitute a
// clean error for the panic; without them Verify's zero values would be
// returned as a bogus success.
func verifyRecovering(
	ctx context.Context,
	verifier TokenVerifier,
	token string,
	logger *slog.Logger,
	fullMethod string,
) (principal Principal, err error) {
	defer func() {
		if r := recover(); r != nil {
			principal = Principal{}
			err = ErrVerifierPanicked
			logger.Error("token verifier panicked; denying the request",
				"method", fullMethod,
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()
	return verifier.Verify(ctx, token)
}

// errVerifierFailed is what a caller sees when the verifier could not reach a
// conclusion about their token.
//
// The message is grpcrecovery's, verbatim and deliberately: propagating the
// panic used to produce exactly this status and exactly this string, so
// recovering it here changes nothing a client can observe. It also says
// nothing, for grpcrecovery's reason — a panic value routinely embeds internal
// detail, and this one is raised while handling a credential.
//
// codes.Internal rather than codes.Unauthenticated because the caller did
// nothing wrong, and reporting our bug as "your token is bad" sends a fleet of
// clients into a pointless re-authentication loop during an incident — the
// same reasoning ADR-0013 §5 applies to ErrKeyUnavailable. Internal rather
// than the codes.Unavailable that ADR gives ErrKeyUnavailable because a panic
// is a deterministic bug rather than a transient outage, and Unavailable is
// the code gRPC clients retry on: an always-panicking verifier plus a
// retryable status is a self-inflicted request storm.
var errVerifierFailed = status.Error(codes.Internal, "internal error")

// bearerToken pulls the bearer credential out of incoming gRPC metadata.
//
// It reports false — rather than an error — for every "there is nothing to
// verify here" case, so that the absence of a token and the presence of a
// different auth scheme both cost nothing and produce no log noise.
func bearerToken(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(metadataKey)
	if len(values) == 0 {
		return "", false
	}

	// Only the first value is considered. A caller sending two authorization
	// headers is either confused or probing; picking one deterministically
	// beats concatenating them or scanning for the one that happens to work.
	scheme, credentials, found := strings.Cut(strings.TrimSpace(values[0]), " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}

	credentials = strings.TrimSpace(credentials)
	if credentials == "" {
		return "", false
	}
	return credentials, true
}

// principalStream substitutes the principal-carrying context for the stream's
// own. grpc.ServerStream has no setter for its context, so wrapping is the
// only way a stream handler can observe anything an interceptor added.
type principalStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *principalStream) Context() context.Context { return s.ctx }

// unaryMethod reports the full RPC method name for logging, tolerating a nil
// info the way grpcrecovery's method() does.
func unaryMethod(info *grpc.UnaryServerInfo) string {
	if info == nil {
		return "unknown"
	}
	return info.FullMethod
}
