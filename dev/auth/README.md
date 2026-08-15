# Local-development auth fixture — PUBLIC KEY MATERIAL, NOT A SECRET

**Everything in this directory is committed on purpose and is worthless.**
Anyone who can read this repository holds the private key, so a token signed
with it proves nothing. It exists so that `make up` starts and so a developer
can call an authenticated endpoint locally and watch it succeed.

**Never point a deployment at these files.** A real environment sets
`AUTH_ISSUER` / `AUTH_AUDIENCE` / `AUTH_JWKS_FILE` at a real identity
provider's issuer, audience and JWKS document, and never mounts this
directory. The Docker image carries no copy of these files — `docker-compose`
bind-mounts them at run time (see `docker-compose.yml`'s `app` service), so the
image a deployment runs is the same image `make up` builds, minus the mount.

## The files

| File | What it is |
|---|---|
| `dev-only-insecure.private-jwk.json` | the signing key, used only by `cmd/devtoken` |
| `dev-only-insecure.jwks.json` | the public half, in the JWK Set shape `internal/platform/auth/rs256` parses — this is what `AUTH_JWKS_FILE` names |

The public file is **derived** from the private one, never hand-edited.
`tools/devtoken`'s tests regenerate it and compare bytes, so the two cannot
drift apart into a JWKS nobody can mint against.

### Why a JWK and not a PEM

The private key is stored as an RFC 7517 JWK rather than
`-----BEGIN RSA PRIVATE KEY-----`. Two reasons, in order of weight:

1. **It stops a deliberately-public file from tripping every secret scanner
   pointed at this repository** — GitHub push protection and this repo's own
   security gate (ADR-0011) both pattern-match PEM private-key headers. Issue
   #160 raised exactly this as the thing to decide before choosing a layout. A
   permanent known-false alert is worse than no alert, because it teaches
   people to click through the real ones.
2. The public half beside it is already a JWK Set — that is the format
   `cmd/server` parses — so one format covers both halves and one reader
   understands both.

The key material is identical either way; only the encoding differs.

## Minting a token

```bash
# From the repository root:
make dev-token                                    # prints a token to stdout
TOKEN=$(make -s dev-token)                        # capture it
go run ./cmd/devtoken -subject 'dev|alice' -ttl 5m # someone else, briefly
```

Then call an authenticated endpoint:

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  'http://localhost:8080/v1/recurring-hire-templates'
```

Without the header the same call returns `401 Unauthenticated`. That contrast
is the point of the fixture: the enforcement is real, and now it is
observable. See `README.md`'s "Authenticated endpoints" section.

## Identity of the fixture

| | |
|---|---|
| issuer (`iss`) | `https://dev-auth.pickleball.invalid/` |
| audience (`aud`) | `https://api.pickleball.invalid/dev` |
| key id (`kid`) | `dev-only-insecure-do-not-trust` |
| default subject (`sub`) | `dev\|local-user-1` |

`.invalid` is reserved by RFC 2606 and can never resolve, so neither URL can
collide with a real provider's. The subject is shaped like a real provider's
(`<connection>|<id>`, as Auth0 renders them) so a handler that mishandles that
shape fails locally rather than in a deployment.

`cmd/devtoken` refuses to sign with any key whose `kid` does not start with
`dev-only-`, so it cannot be repurposed into a general-purpose token forger
pointed at a real signing key that happens to be on the machine.

## Rotating it

```bash
go run ./cmd/devtoken -regenerate   # rewrites BOTH files from one new key
```

Commit both files together. Every previously-minted token stops verifying
immediately, which is the intended behaviour and costs nothing — mint another.

## Background

- Issue #160 (this fixture), issue #136 / #135 (the fail-closed startup check
  that made it necessary), T14.9 in `docs/process/t14-sprint-plan.md`.
- `docs/adr/0013-verified-caller-identity-as-a-platform-capability.md`, and its
  T13.5 amendment.
- `tools/devtoken` — the minting logic and its tests.
