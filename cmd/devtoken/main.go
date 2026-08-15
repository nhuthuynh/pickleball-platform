// Command devtoken mints a local-development bearer token, or regenerates the
// development key fixture it signs with.
//
// It exists because until now nothing in this repository could call an
// authenticated RPC. Enforcement has been on since T12.7–T12.9 and the tokens
// it demands are minted by an identity provider this project cannot provision,
// so "does auth actually work end to end?" was unanswerable locally — and
// since T13.5 made a missing verifier a startup failure, `make up` did not
// start at all. See issue #160 and tools/devtoken's package doc.
//
// Usage:
//
//	go run ./cmd/devtoken                      # a token for the default dev subject
//	go run ./cmd/devtoken -subject 'dev|alice' # someone else
//	go run ./cmd/devtoken -ttl 5m -scope 'bookings:read'
//	go run ./cmd/devtoken -regenerate          # rotate dev/auth/*
//
// The token goes to stdout and nothing else does, so it composes:
//
//	TOKEN=$(go run ./cmd/devtoken)
//	curl -H "Authorization: Bearer $TOKEN" localhost:8080/v1/recurring-hire-templates
//
// Everything explanatory — including the banner making it unmistakable that
// this credential is worthless outside local development — goes to stderr.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nhuthuynh/white-label/tools/devtoken"
)

// keyBits is the size of a regenerated fixture key. 2048 is the verifier's
// floor (RFC 8725 §3.5) and the right size for a key whose only job is to
// make local development work: a larger one would be slower to generate and
// no less public.
const keyBits = 2048

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "devtoken: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	flags := flag.NewFlagSet("devtoken", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var (
		keyPath  = flags.String("key", devtoken.PrivateJWKFile, "path to the dev private JWK fixture")
		jwksPath = flags.String("jwks", devtoken.JWKSFile, "path to write the derived JWKS to (with -regenerate)")
		issuer   = flags.String("issuer", devtoken.Issuer, "`iss` claim; must match the server's AUTH_ISSUER")
		audience = flags.String("audience", devtoken.Audience, "`aud` claim; must match the server's AUTH_AUDIENCE")
		subject  = flags.String("subject", devtoken.Subject, "`sub` claim — the caller this token authenticates as")
		scope    = flags.String("scope", "", "space-separated `scope` claim (optional)")
		ttl      = flags.Duration("ttl", devtoken.DefaultTTL, "how long the token is valid for")

		// Rotation is a flag rather than a separate command because it is the
		// same key material and the same two files; splitting it would invite
		// one to be regenerated without the other.
		regenerate = flags.Bool("regenerate", false,
			"generate a NEW dev keypair and rewrite both fixture files, then exit")
	)

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *regenerate {
		return regenerateFixture(*keyPath, *jwksPath, stderr)
	}

	document, err := os.ReadFile(*keyPath)
	if err != nil {
		return fmt.Errorf("reading the dev key fixture: %w "+
			"(run this from the repository root, or pass -key)", err)
	}
	key, err := devtoken.LoadPrivateJWK(document)
	if err != nil {
		return err
	}

	token, err := key.Mint(devtoken.MintParams{
		Issuer:   *issuer,
		Audience: *audience,
		Subject:  *subject,
		Scope:    *scope,
		TTL:      *ttl,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stderr,
		"devtoken: DEV-ONLY token — signed with the PUBLIC, COMMITTED fixture key %q.\n"+
			"devtoken: It authenticates as %q against issuer %s, and is worthless anywhere real.\n"+
			"devtoken: Valid for %s (until %s).\n",
		key.KeyID(), *subject, *issuer, *ttl, time.Now().Add(*ttl).UTC().Format(time.RFC3339))

	fmt.Fprintln(stdout, token)
	return nil
}

// regenerateFixture rotates the committed fixture. Both files are written
// from one freshly-generated key, so they cannot drift apart — the JWKS is
// derived, never hand-edited.
func regenerateFixture(keyPath, jwksPath string, stderr *os.File) error {
	key, err := devtoken.GenerateKeyPair(keyBits)
	if err != nil {
		return err
	}

	privateDoc, err := key.MarshalPrivateJWK()
	if err != nil {
		return err
	}
	jwksDoc, err := key.MarshalJWKS()
	if err != nil {
		return err
	}

	for _, f := range []struct {
		path string
		body []byte
	}{{keyPath, privateDoc}, {jwksPath, jwksDoc}} {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(f.path), err)
		}
		// 0o644, not 0o600: treating this file as a secret would be a lie
		// that costs a reader the one thing they most need to know about it.
		if err := os.WriteFile(f.path, f.body, 0o644); err != nil { //nolint:gosec // deliberately world-readable dev fixture
			return fmt.Errorf("writing %s: %w", f.path, err)
		}
		fmt.Fprintf(stderr, "devtoken: wrote %s\n", f.path)
	}

	fmt.Fprintf(stderr,
		"devtoken: rotated the DEV-ONLY fixture (kid %q). Commit BOTH files together;\n"+
			"devtoken: any token minted from the previous key stops verifying immediately.\n", key.KeyID())
	return nil
}
