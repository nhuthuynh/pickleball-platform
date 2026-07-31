# ADR-0003: Use local `buf` codegen plugins and vendored googleapis protos, not BSR remotes

## Status
Accepted (T4)

## Context
The original `buf.gen.yaml`/`proto/buf.yaml` (T0) used the Buf Schema
Registry (BSR, `buf.build`) for both the `googleapis/googleapis` proto
dependency and the four codegen plugins (`protocolbuffers/go`, `grpc/go`,
`grpc-ecosystem/gateway`, `grpc-ecosystem/openapiv2`). When T4 needed
`make generate` to actually run (to compile the Postgres/gRPC adapters
against real generated code and run the concurrency test), the BSR was
unreachable from this environment: `buf dep update` and `buf generate` both
failed with "the server hosted at that remote is unavailable." `buf` and
`sqlc` themselves installed fine via `go install`, since the Go module
proxy was reachable — only BSR-specific requests failed.

## Decision
- Vendor `google/api/annotations.proto` and `google/api/http.proto` directly
  into `proto/google/api/` (Apache 2.0, sourced from a copy already present
  via the `grpc-ecosystem/grpc-gateway` Go module dependency) instead of
  depending on `buf.build/googleapis/googleapis`.
- Install the four codegen plugins locally via `go install` (all are
  ordinary Go binaries) and reference them in `buf.gen.yaml` as `local:`
  plugins instead of `remote:`.

## Consequences
**Pros (technical):** `make generate` now works fully offline, in any
environment with Go and network access to `proxy.golang.org` — a strictly
weaker requirement than "also needs BSR reachability." This is more
portable, not less, and removes an external dependency on Buf's hosted
service for a solo/small-team project that doesn't need BSR's other
features (schema registry browsing, breaking-change tracking across teams).
**Pros (business/domain):** faster onboarding for whoever picks this project
up next — one less account/network dependency to debug when `make generate`
mysteriously fails.
**Cons:** the two vendored `.proto` files are now a manually-maintained copy
rather than a version-pinned registry dependency; if Google ever revises
`google/api/http.proto`'s wire format (rare, but not impossible), someone
has to notice and re-vendor by hand. Mitigated by these files changing
extremely rarely in practice and by the vendoring being explicit and
commented (`proto/buf.yaml`), not silent.
**Alternative considered and rejected:** keep the BSR dependency and treat
"BSR unreachable" as an environment-specific problem to work around each
time. Rejected because it makes `make generate` unreliable across
environments for no real benefit — this project has no use for BSR's
team-collaboration features.
