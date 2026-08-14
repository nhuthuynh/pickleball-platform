package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MethodSet is the set of fully-qualified gRPC methods that require a verified
// caller — the composed authentication policy for this process.
//
// It is deliberately a *set of method names* and not a per-context flag or a
// handler-side registry. Each bounded context exports its own
// AuthenticatedMethods() from its grpcapi package (A11 Ruling 2: the knowledge
// of which of a context's RPCs are public belongs next to that context's
// handlers), and cmd/server composes them here. A method absent from this set
// is public — which is the safe default in exactly one direction and the
// dangerous one in the other, so a new RPC that should be authenticated must
// be added to its context's list. That is why the migration tickets state the
// authenticated/public split explicitly in their PRs rather than inferring it.
type MethodSet struct {
	methods map[string]struct{}
}

// NewMethodSet composes each context's AuthenticatedMethods() into one policy.
//
// It takes several lists rather than one so the cmd/server composition is one
// line per context, which is what keeps the shared-file append in main.go to a
// single line each for T12.7/T12.8/T12.9 (T12 sprint plan A12).
func NewMethodSet(lists ...[]string) MethodSet {
	set := MethodSet{methods: make(map[string]struct{})}
	for _, list := range lists {
		for _, m := range list {
			set.methods[m] = struct{}{}
		}
	}
	return set
}

// Len is how many distinct methods the composed policy covers.
//
// It exists for cmd/server's no-verifier startup warning, which reports the
// number of RPCs that will reject every caller. That was previously a
// hand-written sum of each context's len(AuthenticatedMethods()) at the call
// site — a list that has to be edited every time a context is migrated, and
// that silently under-reports when it isn't. Asking the set itself removes the
// second place the composition has to be kept in step (T12.8).
func (s MethodSet) Len() int { return len(s.methods) }

// Requires reports whether fullMethod may only be called by a verified caller.
func (s MethodSet) Requires(fullMethod string) bool {
	_, ok := s.methods[fullMethod]
	return ok
}

// RequireUnaryInterceptor rejects calls to methods in set that carry no
// verified Principal.
//
// # This is defense in depth, not the only check
//
// The handlers of every migrated RPC resolve their actor through the principal
// themselves and return the same codes.Unauthenticated when there is none —
// that check is the authoritative one, it is what the per-context
// authz_regression_test.go files exercise, and it is what protects a handler
// invoked outside this interceptor chain. This interceptor exists because a
// declarative, process-wide list is auditable in a way that nine handlers'
// individual habits are not: if a handler is ever added or edited without its
// principal check, the policy list still stops the call at the boundary.
//
// # Ordering
//
// Register after both grpcrecovery's interceptor and this package's
// observe-only UnaryInterceptor. Recovery must stay outermost (see
// interceptor.go), and this one can only decide correctly once the
// observe-only interceptor has had its chance to resolve a token into the
// context — reversing those two would make every authenticated call fail.
//
// A method this interceptor cannot identify (a nil grpc.UnaryServerInfo) fails
// closed with Unauthenticated. The observe-only interceptor tolerates nil info
// because it only logs; an enforcement decision that cannot name its method
// must not be an approval.
func RequireUnaryInterceptor(set MethodSet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info == nil {
			return nil, errUnauthenticated
		}
		if set.Requires(info.FullMethod) {
			if _, ok := PrincipalFromContext(ctx); !ok {
				return nil, errUnauthenticated
			}
		}
		return handler(ctx, req)
	}
}

// RequireStreamInterceptor is the streaming counterpart, registered for the
// reason interceptor.go's StreamInterceptor spells out: grpc applies unary and
// stream interceptors from separate options, so registering only the unary
// form would leave the first streaming RPC anyone adds enforcing nothing.
func RequireStreamInterceptor(set MethodSet) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if info == nil {
			return errUnauthenticated
		}
		if set.Requires(info.FullMethod) {
			if _, ok := PrincipalFromContext(ss.Context()); !ok {
				return errUnauthenticated
			}
		}
		return handler(srv, ss)
	}
}

// errUnauthenticated is the one wording every rejection uses.
//
// It says nothing about *why* the token was unacceptable — missing, expired,
// wrong audience, bad signature. That is deliberate: the caller's remedy is
// identical in every case (obtain a fresh token), while the difference is
// useful mainly to someone probing the endpoint. The operator-facing detail is
// already logged by the observe-only interceptor, which is where it belongs.
var errUnauthenticated = status.Error(codes.Unauthenticated, "this endpoint requires an authenticated caller")

// ErrUnauthenticated returns the canonical Unauthenticated status this package
// rejects with, so handlers report a missing principal identically to the
// interceptor rather than each inventing their own wording.
func ErrUnauthenticated() error { return errUnauthenticated }

// RequireSubject is the handler-boundary translation every migrated RPC uses:
// it yields the verified caller's Subject, or Unauthenticated when there is no
// verified caller.
//
// This is the whole of A11 Ruling 3 in one function. Handlers call it and pass
// the resulting string into the app/domain signatures that already exist,
// which is why no domain or app package imports this one and why every
// existing domain-level authorization test stays valid unchanged.
//
// Critically, it takes only a context. It has no access to the request
// message, so it *cannot* fall back to a caller-supplied actor_user_id field
// even by accident — a handler that trusted the wire field when no principal
// was present would have changed nothing, and making that fallback
// unexpressible here is stronger than remembering not to write it.
func RequireSubject(ctx context.Context) (string, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return "", errUnauthenticated
	}
	return principal.Subject, nil
}
