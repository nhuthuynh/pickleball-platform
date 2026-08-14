package auth

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
// # This interceptor is observe-only, on purpose
//
// It enforces nothing. If there is no token, or the token is expired, forged,
// malformed, or minted for another audience, the request proceeds to the
// handler exactly as it does today, with no principal and no error. Only a
// *successful* verification changes anything, and all it changes is that
// PrincipalFromContext starts returning a value.
//
// That is an acceptance criterion of the ticket that introduced this package,
// not a temporary shortcut. It means the entire path — client metadata,
// gateway header forwarding, key selection, signature and claim checking,
// context plumbing — can be exercised against a running platform before any
// handler's correctness depends on it, and it means installing this
// interceptor is structurally incapable of breaking an already-shipped flow.
// The migration tickets that follow turn enforcement on per-RPC, in the
// contexts that own those RPCs.
//
// # Ordering
//
// This must be registered *after* grpcrecovery's interceptor in
// grpc.ChainUnaryInterceptor. Recovery is outermost so that it covers
// everything after it, including this one: a panic in a TokenVerifier
// implementation is a panic on the request path like any other, and grpc
// installs no recover() of its own. This interceptor deliberately does not
// recover panics itself — masking a broken verifier locally would hide the
// failure at exactly the moment enforcement makes it security-relevant.
//
// A nil verifier is tolerated and means "resolve nothing", which is
// byte-for-byte today's behavior; a nil logger falls back to slog.Default().
// Both mirror grpcrecovery's tolerance of a nil logger: a wiring mistake in
// cmd/server should degrade, not crash the request path. Note that once
// enforcement is on, a nil verifier is fail-open and must instead be a
// startup failure — see ADR-0013.
func UnaryInterceptor(verifier TokenVerifier, logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(resolve(ctx, verifier, logger, unaryMethod(info)), req)
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
		ctx := resolve(ss.Context(), verifier, logger, fullMethod)
		return handler(srv, &principalStream{ServerStream: ss, ctx: ctx})
	}
}

// resolve is the shared body of both interceptors: extract, verify, and — only
// on success — enrich the context. Keeping it in one place is deliberate, for
// the same reason grpcrecovery shares its recovered(): the observe-only
// guarantee must hold identically for unary and streaming rather than being
// two copies that can drift apart.
//
// It returns the original context unchanged on every failure path.
func resolve(ctx context.Context, verifier TokenVerifier, logger *slog.Logger, fullMethod string) context.Context {
	if verifier == nil {
		return ctx
	}

	token, ok := bearerToken(ctx)
	if !ok {
		// Not logged. In observe-only mode the overwhelming majority of
		// requests have no token, and logging each one would produce a line
		// per request for a condition that is currently normal.
		return ctx
	}

	principal, err := verifier.Verify(ctx, token)
	if err != nil {
		// Logged at warn, without the token: a rejected token is the signal
		// an operator needs while the platform is still observe-only, since
		// nothing else will report it — no caller is told, by design. The
		// raw token is a live credential and never goes to a log.
		logger.Warn("rejected bearer token (observe-only: request proceeds unauthenticated)",
			"method", fullMethod,
			"error", err,
			"caller_fault", IsTokenRejection(err),
		)
		return ctx
	}

	return ContextWithPrincipal(ctx, principal)
}

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
