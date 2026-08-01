# ADR-0001: Enforce the no-double-booking invariant in both Postgres and the domain

## Status
Accepted (T0)

## Context
Spec §6 requires a hard no-double-booking guarantee across all four Booking
sources (D3b). Two enforcement points are possible: the domain layer
(`EnsureNoConflict`, checked against an in-memory/query result before
writing) or the database (a Postgres `EXCLUDE` constraint). Enforcing in only
one place has a failure mode: domain-only enforcement races under concurrent
requests (two goroutines both pass the pre-check before either writes);
DB-only enforcement means every invariant violation surfaces as an opaque
`23P01` error with no domain-level test coverage.

## Decision
Enforce the invariant in **both** places, with Postgres's `EXCLUDE USING gist
(court_id WITH =, during WITH &&) WHERE (status <> 'cancelled')` as the
**authoritative** guard, and `domain.EnsureNoConflict` as a **pre-check**
used by the app service before hitting the database, and directly exercised
by fast, dependency-free unit tests (`internal/booking/domain/booking_test.go`).
The Postgres adapter translates a `23P01` violation into the same
`domain.ErrCourtDoubleBooked` the pre-check returns, so callers see one error
type regardless of which layer caught the conflict.

## Consequences
**Pros (technical):** the invariant has fast, isolated unit tests
(no DB) that TDD can run on every save; production correctness under
concurrency still holds because Postgres is authoritative — HANDOFF.md's T4
concurrency test will prove this against a real database.
**Pros (business/domain):** a double-booking (the platform's worst possible
failure mode — two parties showing up for the same court) gets a fast, clear
error at the API layer 99% of the time (the pre-check catches it before ever
reaching the DB in the non-concurrent case), while T4 proves the rare
concurrent-race case still can't slip through.
**Cons:** two places to keep in sync — if a source/status is added, both
`EnsureNoConflict` and the `EXCLUDE` constraint's `WHERE` clause must be
updated together, or they silently drift. Mitigation: this is called out
explicitly in CLAUDE.md rule 4 and `docs/agent-operating-handbook.md` A3.
**Disagreement considered:** a simpler design would drop the domain
pre-check and trust Postgres alone. Rejected because the QA role brief
(operating handbook B4) requires every invariant to have an explicit
domain-level test, and letting the DB be the *only* test surface would mean
the invariant is untested without a live Postgres — unacceptable for the
TDD-first, dependency-free `make test-domain` gate this project is built
around.
