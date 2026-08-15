package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
	"github.com/nhuthuynh/white-label/internal/platform/grpcrecovery"
)

// This file exercises the real wiring cmd/server installs: a grpc.Server whose
// chain is [recovery, auth], serving over an in-memory bufconn listener, with
// a real RS256 verifier over a locally-minted keypair. Nothing here touches
// the network or Docker.

const (
	chainIssuer   = "https://pickleball-chain-test.example.com/"
	chainAudience = "https://api.pickleball-chain-test.example.com"
	chainKeyID    = "chain-key-1"

	unaryMethod  = "/pickleball.authtest.v1.EchoService/WhoAmI"
	streamMethod = "/pickleball.authtest.v1.EchoService/WhoAmIStream"
)

// echoServer reports the subject of whatever principal the auth interceptor
// managed to put in its context — "anonymous" when there is none, which is
// what every already-shipped handler effectively sees today.
type echoServer struct{}

func subjectOf(ctx context.Context) string {
	if p, ok := auth.PrincipalFromContext(ctx); ok {
		return p.Subject
	}
	return "anonymous"
}

var echoServiceDesc = grpc.ServiceDesc{
	ServiceName: "pickleball.authtest.v1.EchoService",
	// A bare `any` handler type makes every implementation satisfy grpc's
	// registration-time type check, which is all this fake service needs.
	HandlerType: (*any)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "WhoAmI",
		Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			in := new(emptypb.Empty)
			if err := dec(in); err != nil {
				return nil, err
			}
			handler := func(ctx context.Context, _ any) (any, error) {
				return wrapperspb.String(subjectOf(ctx)), nil
			}
			if interceptor == nil {
				return handler(ctx, in)
			}
			return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: unaryMethod}, handler)
		},
	}},
	Streams: []grpc.StreamDesc{{
		StreamName: "WhoAmIStream",
		Handler: func(_ any, stream grpc.ServerStream) error {
			if err := stream.RecvMsg(new(emptypb.Empty)); err != nil {
				return err
			}
			return stream.SendMsg(wrapperspb.String(subjectOf(stream.Context())))
		},
		ServerStreams: true,
	}},
	Metadata: "internal/platform/auth/chain_test.go",
}

type chainFixture struct {
	conn    *grpc.ClientConn
	signKey *rsa.PrivateKey
}

// newChainFixture stands up the server with exactly the chain order
// cmd/server uses: recovery first (outermost, so it covers everything after
// it), auth second.
func newChainFixture(t *testing.T, verifier auth.TokenVerifier, signKey *rsa.PrivateKey) chainFixture {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcrecovery.UnaryInterceptor(logger),
			auth.UnaryInterceptor(verifier, logger),
		),
		grpc.ChainStreamInterceptor(
			grpcrecovery.StreamInterceptor(logger),
			auth.StreamInterceptor(verifier, logger),
		),
	)
	return serveFixture(t, srv, signKey)
}

// serveFixture is the bufconn plumbing shared by every fixture in this
// package's tests: register the echo service on srv, serve it in memory, and
// hand back a client. Split out of newChainFixture so panic_test.go can stand
// up a server with a *different* interceptor chain — specifically, one with no
// grpcrecovery in front — without duplicating any of this.
func serveFixture(t *testing.T, srv *grpc.Server, signKey *rsa.PrivateKey) chainFixture {
	t.Helper()

	srv.RegisterService(&echoServiceDesc, &echoServer{})

	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Logf("bufconn server stopped: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	})

	return chainFixture{conn: conn, signKey: signKey}
}

func (f chainFixture) whoAmI(t *testing.T, token string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}

	out := new(wrapperspb.StringValue)
	if err := f.conn.Invoke(ctx, unaryMethod, new(emptypb.Empty), out); err != nil {
		return "", err
	}
	return out.GetValue(), nil
}

func (f chainFixture) whoAmIStream(t *testing.T, token string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}

	stream, err := f.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, streamMethod)
	if err != nil {
		return "", err
	}
	if err := stream.SendMsg(new(emptypb.Empty)); err != nil {
		return "", err
	}
	if err := stream.CloseSend(); err != nil {
		return "", err
	}
	out := new(wrapperspb.StringValue)
	if err := stream.RecvMsg(out); err != nil {
		return "", err
	}
	return out.GetValue(), nil
}

func newChainVerifier(t *testing.T) (auth.TokenVerifier, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	v, err := rs256.NewVerifier(rs256.Config{
		Issuer:   chainIssuer,
		Audience: chainAudience,
		Keys:     rs256.NewStaticKeys(map[string]*rsa.PublicKey{chainKeyID: &priv.PublicKey}),
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v, priv
}

func mintChainToken(t *testing.T, key *rsa.PrivateKey, subject string, expiresIn time.Duration) string {
	t.Helper()

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    chainIssuer,
		Audience:  jwt.ClaimStrings{chainAudience},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
	})
	tok.Header["kid"] = chainKeyID

	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// TestChainPopulatesPrincipalOverRealGRPC proves the whole path — client
// metadata, gateway-compatible `authorization` header, interceptor, verifier,
// context — works against a running server, and that every non-authenticated
// case still gets a normal successful response (observe-only).
func TestChainPopulatesPrincipalOverRealGRPC(t *testing.T) {
	t.Parallel()

	verifier, key := newChainVerifier(t)
	f := newChainFixture(t, verifier, key)

	tests := []struct {
		name  string
		token func() string
		want  string
	}{
		{
			name:  "valid token",
			token: func() string { return mintChainToken(t, key, "auth0|host-42", 10*time.Minute) },
			want:  "auth0|host-42",
		},
		{
			name:  "no token",
			token: func() string { return "" },
			want:  "anonymous",
		},
		{
			name:  "expired token",
			token: func() string { return mintChainToken(t, key, "auth0|host-42", -10*time.Minute) },
			want:  "anonymous",
		},
		{
			name:  "garbage token",
			token: func() string { return "not.a.jwt" },
			want:  "anonymous",
		},
	}

	for _, tt := range tests {
		t.Run("unary/"+tt.name, func(t *testing.T) {
			got, err := f.whoAmI(t, tt.token())
			if err != nil {
				t.Fatalf("WhoAmI error = %v; observe-only mode must not fail any request", err)
			}
			if got != tt.want {
				t.Errorf("subject = %q, want %q", got, tt.want)
			}
		})

		t.Run("stream/"+tt.name, func(t *testing.T) {
			got, err := f.whoAmIStream(t, tt.token())
			if err != nil {
				t.Fatalf("WhoAmIStream error = %v; observe-only mode must not fail any stream", err)
			}
			if got != tt.want {
				t.Errorf("stream subject = %q, want %q", got, tt.want)
			}
		})
	}
}

// panickingVerifier panics on one specific token and behaves normally
// otherwise, so the test can prove both halves of the claim: the panic is
// contained, *and* the server is still serving afterwards.
type panickingVerifier struct{ trigger string }

func (p panickingVerifier) Verify(_ context.Context, raw string) (auth.Principal, error) {
	if raw == p.trigger {
		panic("verifier exploded")
	}
	return auth.Principal{}, auth.ErrTokenMalformed
}

// TestVerifierPanicIsContainedByRecovery pins the end-to-end result of a
// panicking verifier through the chain cmd/server installs: codes.Internal,
// and a server that keeps serving.
//
// T13.5 note on what this test does and no longer does. It was written as the
// *ordering* proof — recovery registered first, therefore outermost, therefore
// catching a panic the auth interceptor deliberately let through; reversing
// the two used to crash this binary. T13.5 (#135) moved that containment into
// the auth interceptor itself, so the assertions below now pass because auth
// recovers the panic, and they would keep passing with grpcrecovery removed
// entirely — see TestVerifierPanicDeniedWithNoRecoveryInterceptor in
// panic_test.go, which is the test that now carries that weight.
//
// Kept, unchanged in substance, because the property it states is still the
// one that matters to a caller and is worth pinning at the real wire boundary:
// this is the assertion that would fail if some future change made a verifier
// panic observable as anything other than Internal.
func TestVerifierPanicIsContainedByRecovery(t *testing.T) {
	t.Parallel()

	const boomToken = "make-the-verifier-panic"
	f := newChainFixture(t, panickingVerifier{trigger: boomToken}, nil)

	t.Run("unary panic becomes Internal, not a dead process", func(t *testing.T) {
		_, err := f.whoAmI(t, boomToken)
		if err == nil {
			t.Fatal("WhoAmI succeeded, want an error from the recovered panic")
		}
		if got := status.Code(err); got != codes.Internal {
			t.Fatalf("status code = %v, want %v", got, codes.Internal)
		}
	})

	t.Run("stream panic becomes Internal, not a dead process", func(t *testing.T) {
		_, err := f.whoAmIStream(t, boomToken)
		if err == nil {
			t.Fatal("WhoAmIStream succeeded, want an error from the recovered panic")
		}
		if got := status.Code(err); got != codes.Internal {
			t.Fatalf("status code = %v, want %v", got, codes.Internal)
		}
	})

	// The load-bearing assertion: the server survived the panic and still
	// serves other callers. If the panic had escaped, this test binary would
	// already be gone rather than reaching this line.
	t.Run("server still serves other callers afterwards", func(t *testing.T) {
		got, err := f.whoAmI(t, "")
		if err != nil {
			t.Fatalf("post-panic WhoAmI error = %v; the server did not survive", err)
		}
		if got != "anonymous" {
			t.Errorf("subject = %q, want %q", got, "anonymous")
		}
	})
}
