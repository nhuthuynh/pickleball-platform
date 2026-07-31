# Review — T0: Bootstrap

## What this covers
Reconstructing the repository from the uploaded planning docs (spec, design
review, technology options, CLAUDE.md, HANDOFF.md) into a runnable Go
codebase: docs in place, the Booking bounded context (domain + app + ports +
Postgres/gRPC adapters), the proto contract, DB schema/queries, and
Docker/Jenkins/Makefile scaffolding — matching the state HANDOFF.md describes
as "done and runnable now."

This was not a backlog phase debated by the six roles in real time — it is a
reconstruction of already-locked decisions (D1-D4, D3a, D3b, D4a in the spec;
the golden rules in CLAUDE.md). Where T0 *did* require judgment calls not
already pinned down by the docs, they're recorded below and in
`docs/adr/0001-dual-invariant-enforcement.md` and
`docs/adr/0002-pricing-rule-ambiguity-is-an-error.md`.

## Decisions made and reasoning

1. **Removed the pre-existing unrelated TypeScript project** rather than
   leaving it alongside the Go code. *Domain/business view:* irrelevant to
   the pickleball platform — keeping it would confuse anyone reading the repo
   about what's real. *Technical view:* a clean root matching CLAUDE.md's
   documented architecture avoids ambiguity about which `go.mod`/tooling
   applies where. Confirmed with the user before deleting (a destructive,
   pre-existing-content action) rather than assuming.
2. **Half-open time ranges `[start, end)`** for both `domain.TimeRange` and
   the Postgres `during tstzrange`. *Business view:* this is what "back to
   back bookings don't conflict" has to mean in practice — a 9-10am booking
   and a 10-11am booking on the same court must both be allowed, which the
   spec implies (§6) but doesn't spell out arithmetically. *Technical view:*
   half-open ranges are the standard, composable choice for interval overlap
   (`a.Start < b.End && b.Start < a.End`) and match Postgres's own `&&`
   operator on `tstzrange` with a `[)` bound — one semantics, no translation
   layer.
3. **Dual enforcement of the no-double-booking invariant** (domain pre-check
   + authoritative Postgres `EXCLUDE`) — see ADR-0001.
4. **Ambiguous pricing-rule match is an error**, not resolved by an implicit
   priority — see ADR-0002.
5. **`source`/`status` as `TEXT` + `CHECK` rather than Postgres `ENUM`
   types.** *Technical view:* adding a new source (unlikely, but the spec's
   four sources aren't marked closed-forever) is a plain migration with
   `TEXT`/`CHECK`, versus `ALTER TYPE ... ADD VALUE` friction (can't run
   inside the same transaction as other DDL in older Postgres) with a real
   enum. Cost: slightly less type safety at the SQL level; sqlc still gives
   Go-level type safety via `domain.Source`/`domain.Status` at the boundary.
6. **`internal/gen/**` adapters (Postgres, gRPC) were written against a
   package that doesn't exist yet** (`make generate` hasn't been run in this
   environment — no `buf`/`sqlc` installed). This exactly matches the
   authoring gotcha already documented in `CLAUDE.md`. Verified: every
   package that *doesn't* depend on generated code (`domain`, `app`,
   `internal/platform/pg`, `internal/platform/idgen`) builds, vets, and lints
   cleanly; the only `go build ./...` failures are the two adapter packages
   and `cmd/server`, exactly as expected — not a regression.

## Gate
- `make test-domain` — **green** (`-race`, all Booking domain + app tests).
- `go vet` / `golangci-lint run` — clean on every package that compiles
  without generated code.
- `gofmt -l .` — clean.
- Full `make generate && make test` was **not** run in this environment
  (no `buf`/`sqlc`/Docker daemon verified available) — this is the next
  developer's/session's first action per HANDOFF.md T0 steps 2-4, not a gap
  introduced here.

## Open items carried forward
- Pricing is modelled and tested but not wired to a use case/API (T1).
- `ListCourtBookings`/`CancelBooking` handlers not implemented (T2, T3).
- No concurrency/integration test yet proving the `EXCLUDE` constraint under
  real concurrent load (T4) — ADR-0001's "authoritative" claim is backed by
  Postgres semantics and cited spec §6, but not yet proven by an automated
  test in this repo.
