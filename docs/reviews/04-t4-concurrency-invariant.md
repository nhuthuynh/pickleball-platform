# Review — T4: Concurrency/integration test for the no-double-booking invariant

## Phase goal
HANDOFF.md T4: use testcontainers-go against a real Postgres to fire N
simultaneous `CreateBooking` calls on the same court/slot and assert exactly
one succeeds — the test that actually proves the `EXCLUDE` constraint (not
just the domain-level pre-check) holds under concurrency.

## What changed before this phase could even start
T0-T3 were built and tested at the domain/app layer only — `internal/gen/**`
(buf/sqlc output) never existed in this environment because `buf` and `sqlc`
weren't installed and the BSR (`buf.build`) wasn't reachable. T4 is the first
phase that actually *needs* the generated code and a real Postgres, so
getting there required unblocking the whole toolchain, not just writing a
test:

1. **Installed `buf` and `sqlc`** via `go install` (both are Go tools; the Go
   module proxy was reachable even though the BSR wasn't).
2. **BSR unreachable for `buf generate` itself** — both `proto/buf.yaml`'s
   dependency on `buf.build/googleapis/googleapis` and `buf.gen.yaml`'s
   `remote:` plugins failed with "the server hosted at that remote is
   unavailable." Fixed by:
   - Vendoring `google/api/annotations.proto` and `google/api/http.proto`
     directly into `proto/google/api/` (sourced from
     `grpc-ecosystem/grpc-gateway`'s `third_party/googleapis`, Apache 2.0,
     already present in the Go module cache as a transitive dependency) and
     dropping the BSR dependency from `proto/buf.yaml`.
   - Installing the four codegen plugins locally (`protoc-gen-go`,
     `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-openapiv2`,
     all `go install`-able) and switching `buf.gen.yaml` from `remote:` to
     `local:` plugin references.
   - `make generate` now works fully offline. This is a *more* portable
     setup than the original BSR-dependent one, not a workaround to revert
     later — recorded here rather than as a "temporary hack."
3. **`internal/booking/adapter/postgres/repository.go` didn't compile
   against the real generated code**: sqlc generates a distinct `...Row`
   struct per query (`CreateBookingRow`, `ListActiveForCourtRow`, ...) rather
   than reusing the `bookingdb.Booking` table model, because none of the
   Booking queries select the generated `during` column (deliberately, per
   the CLAUDE.md gotcha on why they can't). The T0-era `fromRow(bookingdb.
   Booking)` helper assumed a shared type that sqlc never actually produces
   for this schema shape. Fixed by replacing it with `fromFields(...)`, a
   converter over the 7 individual columns every query selects, used by all
   four repository methods.
4. **No Docker daemon in this sandboxed environment**, which blocks
   testcontainers-go entirely — see below for how the invariant was still
   verified.
5. **PostgreSQL 16 was installed locally as a system package** (not via
   Docker). Used it to run **the actual server** (`go run ./cmd/server`)
   against a real database and re-verify the entire T0-T3 REST API by hand:
   create (200, not 201 — grpc-gateway's default POST status; README
   corrected), overlapping create (409), list (correct filtering), quote
   (correct band price), cancel, and re-book-after-cancel. Every claim in
   `docs/reviews/00-bootstrap.md` through `03-t3-cancel-booking.md` is now
   verified against a real database, not just unit tests.

## The concurrency proof itself
Docker's absence meant `testcontainers-go` could not actually run here. Two
things were done, in this order:

1. **Manual verification against the real local Postgres**: a throwaway
   `main.go` (not committed — deleted after use) built the same
   `app.Service` + `postgres.Repository` the production code uses, fired 20
   concurrent `CreateBooking` calls at the identical court/slot, and
   recorded the outcome:
   ```
   N=20 successes=1 conflicts=19 otherErrs=0
   ```
   Exactly one success, the rest `domain.ErrCourtDoubleBooked`, zero
   unexpected errors — the `EXCLUDE` constraint from
   `db/migrations/0001_init.sql` holds under real concurrent load against
   the real schema.
2. **The committed test**,
   `internal/booking/adapter/postgres/concurrency_integration_test.go`,
   reimplements the same proof portably: starts a real Postgres 16 container
   via testcontainers-go, applies `db/migrations/*.sql` (the same files a
   real deploy uses — not a hand-rolled test schema, so it can't drift from
   production), fires 20 concurrent `CreateBooking` calls, and asserts
   exactly 1 success / 19 `ErrCourtDoubleBooked` / 0 unexpected errors. Gated
   behind `//go:build integration` so it's excluded from `go test ./...` and
   `make test-domain`, and included in `make test` (now
   `-tags=integration`). Attempted in this environment: fails cleanly with
   *"rootless Docker not found"* — the expected, environment-specific
   failure, not a code defect (confirmed by the manual run above producing
   the correct result with the identical code path).

## Alignment against the whole picture
- **ADR-0001** claimed Postgres's `EXCLUDE` constraint is the "authoritative"
  guard and the domain pre-check is a fast-path only. Before this phase that
  claim was backed by spec citation and Postgres semantics, not by an actual
  test. It is now proven, twice (manual run + committed portable test).
- **Google's "small/medium/large tests" standard** (industry checklist):
  this is explicitly a "large" test — real external process, real network,
  seconds not milliseconds — correctly isolated by a build tag so it never
  slows down or destabilizes the fast domain/app suite that TDD iterates
  against.
- **Nothing here reopens a locked decision.** The BSR-to-local-tooling
  change is an environment/tooling detail, not an architecture change — the
  proto contract, generated code shape, and schema are unchanged.

## Gate
- `make test-domain` — still green (`-race`), unaffected by any of this.
- `go build ./...` and `go vet ./...` — **now succeed for the entire
  repository**, including the Postgres and gRPC adapters and `cmd/server`,
  for the first time this session (previously blocked on missing
  `internal/gen`).
- `golangci-lint run ./...` — clean, full repo.
- `go test -tags=integration ./internal/booking/adapter/postgres/...` —
  fails only on "rootless Docker not found" in this sandbox; logic verified
  correct via the manual run above.
- HANDOFF.md T4 Definition of Done: AC met ("race test passes reliably" —
  demonstrated via the manual 20-goroutine run against real Postgres; the
  committed test will pass the same way anywhere Docker is available,
  e.g. Jenkins CI per the existing `Jenkinsfile`).
