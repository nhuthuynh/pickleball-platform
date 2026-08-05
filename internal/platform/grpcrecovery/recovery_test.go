package grpcrecovery_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/nhuthuynh/white-label/internal/platform/grpcrecovery"
)

// These tests run against a real grpc.Server over a real (in-memory) connection
// rather than calling the interceptor function directly. That is the whole
// point: the bug being fixed is "an unrecovered panic unwinds past grpc and
// kills the process", and only an actual server actually serving can show that
// it no longer does. A unit test that invokes the returned closure and asserts
// on its error proves the error mapping and nothing about survival.
const (
	testService = "pickleball.platform.grpcrecovery.testv1.RecoveryProbe"
	probeMethod = "/" + testService + "/Probe"
	okMethod    = "/" + testService + "/Ok"
)

// probe is the test service implementation. behave is what the Probe method
// does — panic, fail, or succeed — and is swapped between subtests so one
// long-lived server can be driven through every case. If any case actually
// crashed the server, every later case on the same server would fail to
// connect, which is precisely the regression signal we want.
type probe struct {
	mu     sync.Mutex
	behave func() (any, error)
}

func (p *probe) set(fn func() (any, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.behave = fn
}

func (p *probe) call() (any, error) {
	p.mu.Lock()
	fn := p.behave
	p.mu.Unlock()
	return fn()
}

type recoveryProbeServer any

func probeHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(*probe).call()
	}
	if interceptor == nil {
		return handler(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: probeMethod}, handler)
}

// okHandler is a method that never panics. It exists so a test can ask the
// server "are you still there?" without depending on the state of the method
// that just panicked.
func okHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return &emptypb.Empty{}, nil
	}
	if interceptor == nil {
		return handler(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: okMethod}, handler)
}

var probeDesc = grpc.ServiceDesc{
	ServiceName: testService,
	HandlerType: (*recoveryProbeServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Probe", Handler: probeHandler},
		{MethodName: "Ok", Handler: okHandler},
	},
	Metadata: "grpcrecovery_test",
}

// newProbeServer starts a grpc.Server wired with the recovery interceptors and
// returns a connected client, the mutable probe, and the buffer capturing the
// server's log output.
func newProbeServer(t *testing.T) (*grpc.ClientConn, *probe, *bytes.Buffer) {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(syncWriter{buf: logs}, nil))

	p := &probe{behave: func() (any, error) { return &emptypb.Empty{}, nil }}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcrecovery.UnaryInterceptor(logger)),
		grpc.ChainStreamInterceptor(grpcrecovery.StreamInterceptor(logger)),
	)
	srv.RegisterService(&probeDesc, p)

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		srv.Stop()
	})

	return conn, p, logs
}

// syncWriter guards the log buffer: slog writes happen on the server's handler
// goroutine while assertions read from the test goroutine, which -race would
// otherwise (correctly) flag.
type syncWriter struct {
	buf *bytes.Buffer
}

var logMu sync.Mutex

func (w syncWriter) Write(p []byte) (int, error) {
	logMu.Lock()
	defer logMu.Unlock()
	return w.buf.Write(p)
}

func readLogs(buf *bytes.Buffer) string {
	logMu.Lock()
	defer logMu.Unlock()
	return buf.String()
}

func invoke(t *testing.T, conn *grpc.ClientConn, method string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return conn.Invoke(ctx, method, &emptypb.Empty{}, &emptypb.Empty{})
}

// TestUnaryInterceptorRecoversPanicAndServerSurvives is the test that actually
// proves the fix. For each kind of panic it asserts the caller gets a clean
// Internal status, and then — the part that matters — issues a *second,
// ordinary* RPC and requires it to succeed. Before this interceptor existed the
// first panic would have taken the whole process down and there would be no
// second call to make.
func TestUnaryInterceptorRecoversPanicAndServerSurvives(t *testing.T) {
	conn, p, _ := newProbeServer(t)

	tests := []struct {
		name  string
		panic func()
	}{
		{
			name: "panic with a string",
			// The shape of the real bug: every Postgres adapter's mustUUID
			// panics with a formatted string on a malformed UUID.
			panic: func() {
				panic(`competitions postgres adapter: invalid uuid "not-a-uuid": cannot parse UUID not-a-uuid`)
			},
		},
		{
			name:  "panic with an error value",
			panic: func() { panic(errors.New("boom")) },
		},
		{
			name:  "nil map write",
			panic: func() { var m map[string]string; m["k"] = "v" },
		},
		{
			name:  "index out of range",
			panic: func() { s := []int{}; _ = s[3] },
		},
		{
			name:  "nil pointer dereference",
			panic: func() { var q *probe; _ = q.behave },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p.set(func() (any, error) { tc.panic(); return nil, nil })

			err := invoke(t, conn, probeMethod)
			if err == nil {
				t.Fatal("want an error from a panicking handler, got nil (a recovered panic must never look like success)")
			}
			if got := status.Code(err); got != codes.Internal {
				t.Fatalf("status code = %v, want %v", got, codes.Internal)
			}

			// The survival assertion. A crashed process cannot answer this.
			p.set(func() (any, error) { return &emptypb.Empty{}, nil })
			if err := invoke(t, conn, okMethod); err != nil {
				t.Fatalf("server did not survive the panic: follow-up call failed: %v", err)
			}
		})
	}

	// And still alive after every case in the table, not just after one.
	if err := invoke(t, conn, okMethod); err != nil {
		t.Fatalf("server did not survive the full panic sequence: %v", err)
	}
}

// TestUnaryInterceptorDoesNotLeakPanicDetail guards the security half of the
// mapping. mustUUID's panic string embeds the caller's raw input and the
// adapter's internal package path, and the endpoints most likely to panic on
// bad input are the public unauthenticated ones — so the panic value must not
// travel back to the client.
func TestUnaryInterceptorDoesNotLeakPanicDetail(t *testing.T) {
	conn, p, _ := newProbeServer(t)

	const secret = "competitions postgres adapter: invalid uuid at /internal/competitions/adapter/postgres"
	p.set(func() (any, error) { panic(secret) })

	err := invoke(t, conn, probeMethod)
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if msg := status.Convert(err).Message(); strings.Contains(msg, "adapter") || strings.Contains(msg, "uuid") {
		t.Fatalf("panic detail leaked to the client in %q", msg)
	}
}

// TestUnaryInterceptorLogsPanicWithStack proves the operator half: a recovered
// panic that nobody is told about is a silently-swallowed bug.
func TestUnaryInterceptorLogsPanicWithStack(t *testing.T) {
	conn, p, logs := newProbeServer(t)

	p.set(func() (any, error) { panic("kaboom") })
	if err := invoke(t, conn, probeMethod); err == nil {
		t.Fatal("want an error, got nil")
	}

	out := readLogs(logs)
	for _, want := range []string{
		"recovered panic in grpc handler", // the operator-facing message
		"kaboom",                          // the panic value itself
		probeMethod,                       // which RPC blew up
		"runtime/debug.Stack",             // a real stack trace, not a placeholder
		"grpcrecovery",                    // the frames are the panicking goroutine's
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// TestUnaryInterceptorPassesThroughNonPanics makes sure the safety net is only
// a safety net. A handler that returns an ordinary error — the NotFound that
// Layer 2's ID validation produces, most importantly — must reach the client
// exactly as written, not be flattened into Internal.
func TestUnaryInterceptorPassesThroughNonPanics(t *testing.T) {
	conn, p, _ := newProbeServer(t)

	tests := []struct {
		name     string
		behave   func() (any, error)
		wantCode codes.Code
	}{
		{
			name:     "success is untouched",
			behave:   func() (any, error) { return &emptypb.Empty{}, nil },
			wantCode: codes.OK,
		},
		{
			name:     "NotFound is preserved, not turned into Internal",
			behave:   func() (any, error) { return nil, status.Error(codes.NotFound, "competition not found") },
			wantCode: codes.NotFound,
		},
		{
			name:     "InvalidArgument is preserved",
			behave:   func() (any, error) { return nil, status.Error(codes.InvalidArgument, "bad") },
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "PermissionDenied is preserved",
			behave:   func() (any, error) { return nil, status.Error(codes.PermissionDenied, "nope") },
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "a handler's own Internal is preserved",
			behave:   func() (any, error) { return nil, status.Error(codes.Internal, "db down") },
			wantCode: codes.Internal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p.set(tc.behave)
			err := invoke(t, conn, probeMethod)
			if got := status.Code(err); got != tc.wantCode {
				t.Fatalf("status code = %v, want %v (err=%v)", got, tc.wantCode, err)
			}
		})
	}
}

// TestUnaryInterceptorNilLoggerDoesNotPanic covers the degenerate wiring case:
// the recovery path is the last code that should be able to crash the process.
func TestUnaryInterceptorNilLoggerDoesNotPanic(t *testing.T) {
	interceptor := grpcrecovery.UnaryInterceptor(nil)
	_, err := interceptor(context.Background(), &emptypb.Empty{},
		&grpc.UnaryServerInfo{FullMethod: probeMethod},
		func(context.Context, any) (any, error) { panic("boom") },
	)
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("status code = %v, want %v", got, codes.Internal)
	}
}

// TestStreamInterceptorRecoversPanic covers the streaming counterpart directly.
// There are no streaming RPCs in the repo yet, so there is no server harness to
// hang this off — the interceptor contract is what is being pinned.
func TestStreamInterceptorRecoversPanic(t *testing.T) {
	logs := &bytes.Buffer{}
	interceptor := grpcrecovery.StreamInterceptor(slog.New(slog.NewJSONHandler(syncWriter{buf: logs}, nil)))

	err := interceptor(nil, nil, &grpc.StreamServerInfo{FullMethod: probeMethod},
		func(any, grpc.ServerStream) error { panic("stream boom") },
	)
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("status code = %v, want %v", got, codes.Internal)
	}
	if out := readLogs(logs); !strings.Contains(out, "stream boom") {
		t.Errorf("stream panic not logged: %s", out)
	}
}
