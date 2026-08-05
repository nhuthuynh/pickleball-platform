// Package grpcrecovery provides the process-wide panic safety net for this
// backend's gRPC server.
//
// # Why this package exists
//
// A panic inside a gRPC handler is not a per-request failure. google.golang.org/
// grpc runs each unary RPC on its own goroutine and installs **no** recover()
// of its own, so an unrecovered panic in any handler unwinds past the server
// and terminates the entire process — taking every other in-flight request,
// every other bounded context, and every other client down with it. This is
// the opposite of net/http, which recovers panics per-connection and keeps
// serving; code review intuition carried over from HTTP handlers is wrong here.
//
// That gap was not theoretical. Before this package existed, cmd/server called
// grpc.NewServer() with no options at all, and every Postgres adapter in the
// repo has a mustUUID helper that deliberately panics on a malformed UUID. Some
// read paths take an ID straight off the wire with no upstream validation, so
// an unauthenticated `GET /v1/competitions/not-a-uuid` reached mustUUID and
// killed the server — a trivially-triggerable total outage.
//
// # What this fixes, and what it does not
//
// This interceptor is the safety net, not the cure. It turns "the platform is
// down for everyone" into "one caller got a 500", which is the difference
// between an outage and a bug. The cure for any specific panic is still to stop
// panicking — validate input at the boundary where it enters the system (see
// the app-layer ID checks in internal/competitions/app and internal/facilities/
// app) so the panic never happens. Both layers are load-bearing: validation
// stops the panics we know about, this stops the ones we don't, including in
// handlers not yet written.
package grpcrecovery

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// clientMessage is what a caller sees when a handler panicked.
//
// It is deliberately constant and deliberately says nothing. The panic value
// routinely contains internal detail — mustUUID's panic string embeds the raw
// input and the adapter's package path, and a nil-map or index-out-of-range
// panic embeds internals no caller should learn about. Several of the endpoints
// this protects are public and unauthenticated (Competitions'
// GetCompetitionByShareToken most obviously), so echoing the panic value back
// would convert a crash bug into an information-disclosure bug. The detail goes
// to the operator via the log, which is who actually needs it.
const clientMessage = "internal error"

// UnaryInterceptor returns a grpc.UnaryServerInterceptor that recovers from any
// panic raised by a unary handler, logs it together with the stack trace of the
// goroutine that panicked, and converts it into a codes.Internal status.
//
// The stack is captured with runtime/debug.Stack() from inside the deferred
// recover, which is what makes it the *panicking* stack rather than the
// already-unwound one — capture it anywhere else and an operator gets a trace
// pointing at the interceptor instead of at the bug. This is a production
// process: a recovered panic that is silently swallowed is arguably worse than
// the crash, because the crash at least announced itself.
//
// Errors that handlers return normally — including domain errors already mapped
// to a status by an adapter's toStatus — pass through untouched. This
// interceptor only ever acts on a panic.
//
// A nil logger is tolerated (falls back to slog.Default()) so that a wiring
// mistake in cmd/server degrades to "we lost the log line" rather than
// "the recovery interceptor itself panicked", which would defeat the purpose.
func UnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Named return values are what let a deferred function
				// substitute a clean error for the panic; without them the RPC
				// would return a nil response and a nil error, which grpc
				// reports to the caller as a bogus success.
				resp = nil
				err = recovered(logger, method(info), r)
			}
		}()
		return handler(ctx, req)
	}
}

// StreamInterceptor is the streaming counterpart to UnaryInterceptor.
//
// This repo has no streaming RPCs today. It is here anyway because grpc.NewServer
// applies unary and stream interceptors from separate options: registering only
// the unary one leaves the first streaming method anyone adds silently
// unprotected, re-opening exactly the hole this package was written to close,
// and the failure mode (whole-process crash) is too severe to leave waiting on
// someone remembering.
func StreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		fullMethod := "unknown"
		if info != nil {
			fullMethod = info.FullMethod
		}
		defer func() {
			if r := recover(); r != nil {
				err = recovered(logger, fullMethod, r)
			}
		}()
		return handler(srv, ss)
	}
}

// recovered performs the logging and status conversion shared by both
// interceptors. Keeping it in one place is deliberate: the guarantee that a
// panic is never silently swallowed and never leaked to the caller should hold
// identically for unary and streaming, not be two copies that can drift.
func recovered(logger *slog.Logger, fullMethod string, r any) error {
	logger.Error("recovered panic in grpc handler",
		"method", fullMethod,
		"panic", r,
		"stack", string(debug.Stack()),
	)
	return status.Error(codes.Internal, clientMessage)
}

// method reports the full RPC method name for logging, tolerating a nil info.
// grpc always supplies one, but this runs on the crash path — the code that
// handles a panic is the last place that should be able to panic itself.
func method(info *grpc.UnaryServerInfo) string {
	if info == nil {
		return "unknown"
	}
	return info.FullMethod
}
