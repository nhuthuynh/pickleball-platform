package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequireAuthentication returns a grpc.UnaryServerInterceptor that rejects a
// request to any of the named methods unless UnaryInterceptor already
// resolved a Principal for it.
//
// # Why this exists, and what it is not
//
// T12.2 shipped this package observe-only: it resolves tokens and enforces
// nothing (see the package doc and interceptor.go). ADR-0013 §4 states that
// enforcement is turned on per-RPC by the migration tickets, each context
// exporting its own AuthenticatedMethods() (A11 Ruling 2), with cmd/server
// composing them. This is the piece that consumes that composition — the
// platform-side primitive the per-context lists are lists *for*. Without it
// AuthenticatedMethods() would be a declaration nothing reads.
//
// It is a BACKSTOP, not the primary check. Handlers that need a principal
// resolve it themselves with PrincipalFromContext, because a handler
// usually needs the principal's value and not merely its presence, and
// because a handler-level guard is what handler-level regression tests can
// exercise. What this interceptor adds is that a future RPC added to an
// already-migrated service cannot be reached anonymously just because
// someone forgot a guard: the failure mode becomes "this RPC is
// unreachable until you list it", which is discovered immediately, rather
// than "this RPC is silently public", which is discovered by an attacker.
// Defence in depth in the direction that fails safe.
//
// # Ordering
//
// Register this AFTER UnaryInterceptor in grpc.ChainUnaryInterceptor —
// interceptors run in registration order, so the resolving interceptor must
// have run before this one can observe what it produced. Registering them
// the other way round rejects every request, including well-authenticated
// ones. The full intended chain is grpcrecovery, then UnaryInterceptor,
// then this.
//
// # Fail-closed, deliberately
//
// A method named here with no principal is refused. In particular, when
// cmd/server has no verifier configured (no AUTH_ISSUER/AUTH_AUDIENCE/
// AUTH_JWKS_FILE), no request can ever carry a principal, so every listed
// method returns Unauthenticated. That is the correct direction to fail and
// it is the visible half of known gap #136: a nil verifier is fail-OPEN
// under an enforcement model that only consults handler guards, and
// fail-CLOSED under this one. The deployment consequence — a server with
// enforcement compiled in but no verifier configured cannot serve its
// authenticated RPCs — is stated in cmd/server's wiring rather than papered
// over here, because the alternative (skipping enforcement when no verifier
// is configured) is precisely the silent downgrade ADR-0013 §3 refuses.
//
// An empty or nil methods slice yields an interceptor that enforces
// nothing, which is the correct behavior for a server whose contexts have
// all not yet migrated.
func RequireAuthentication(methods []string) grpc.UnaryServerInterceptor {
	required := methodSet(methods)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info != nil && required[info.FullMethod] {
			if _, ok := PrincipalFromContext(ctx); !ok {
				return nil, errUnauthenticated(info.FullMethod)
			}
		}
		return handler(ctx, req)
	}
}

// RequireAuthenticationStream is the streaming counterpart, registered for
// the same reason grpcrecovery and UnaryInterceptor both ship stream forms
// (interceptor.go's StreamInterceptor doc comment): grpc.NewServer takes
// unary and stream interceptors from separate options, so a unary-only
// registration would leave the first streaming RPC anyone adds enforcing
// nothing. This repo has no streaming RPCs today; the gap it would leave is
// the point.
func RequireAuthenticationStream(methods []string) grpc.StreamServerInterceptor {
	required := methodSet(methods)

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if info != nil && required[info.FullMethod] {
			if _, ok := PrincipalFromContext(ss.Context()); !ok {
				return errUnauthenticated(info.FullMethod)
			}
		}
		return handler(srv, ss)
	}
}

// methodSet builds the lookup once, at construction, rather than scanning a
// slice per request.
func methodSet(methods []string) map[string]bool {
	set := make(map[string]bool, len(methods))
	for _, m := range methods {
		set[m] = true
	}
	return set
}

// errUnauthenticated is codes.Unauthenticated — "I do not know who you
// are" — and never codes.PermissionDenied, which would assert we had
// identified the caller and were refusing them. Conflating the two is the
// classic authorization bug ADR-0013 §5 is pedantic about.
//
// The message names no detail about why verification failed (expired vs.
// forged vs. absent): the interceptor cannot distinguish those cases
// anyway, since the resolving interceptor logs the reason server-side and
// passes nothing to the caller by design.
func errUnauthenticated(fullMethod string) error {
	return status.Errorf(codes.Unauthenticated, "auth: %s requires an authenticated caller", fullMethod)
}
