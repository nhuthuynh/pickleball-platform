# Review — T2: ListCourtBookings

## Phase goal
HANDOFF.md T2: the `ListCourtBookings` proto method and
`port.Repository.ListActiveForCourt` already existed (from T0); this phase
wires the app method + gRPC/REST handler + tests. AC: "REST GET returns
bookings intersecting the range."

## Process note
Unlike T1 (a new domain concept — pricing-band resolution — with real edge
cases), T2 is pure wiring: every piece of logic it touches
(`ListActiveForCourt`'s overlap filtering, `TimeRange.Overlaps`,
cancelled-booking exclusion) was already implemented and tested in T0. Given
the low risk surface, this phase's review was done directly against the
Industry Standards Checklist rather than spawning adversarial subagents —
consistent with the "role-play in one pass" precedent already used for
`docs/spec-design-review.md`, and a deliberate cost/risk tradeoff: subagent
review earns its cost when there's real judgment to stress-test (T1's
band-boundary and midnight-crossing logic), not for a pass-through method
with no new business rule.

## What was done (TDD)
`internal/booking/app/list_bookings_test.go` written and run red (compile
failure — `ListCourtBookings` didn't exist) before
`Service.ListCourtBookings` was added. Two tests:
- `TestListCourtBookings_ReturnsIntersectingBookings` — proves court
  scoping, range intersection (a whole-day query returns both of court-1's
  bookings; a narrower query returns only the one it overlaps), and that a
  different court's booking is excluded even though its time range overlaps.
- `TestListCourtBookings_ExcludesCancelled` — proves a cancelled booking
  (freed by T3's future Cancel, but the `Status` field already exists from
  T0) doesn't show up in listings, matching CLAUDE.md rule 4's "cancelled
  bookings don't hold the invariant" semantics extending to reads too.

`internal/booking/adapter/grpcapi/handler.go` gained `ListCourtBookings`,
translating the proto `from`/`to` fields into a `domain.TimeRange` (reusing
`domain.NewTimeRange`'s validation — an empty/inverted range is rejected the
same way `CreateBooking`'s does) and mapping the result through the existing
`toProto` helper.

## Checklist self-review
- **DDD:** no new domain concept; `Service.ListCourtBookings` is a one-line
  pass-through, correctly keeping the app layer between the API and the
  repository port rather than letting the handler call `port.Repository`
  directly.
- **TDD:** red confirmed before implementation; both tests would fail if
  either the court-scoping or the cancelled-exclusion behavior were removed.
- **Dependency rule:** handler -> app -> domain/port, unchanged shape from
  T1.
- **Nothing to debate:** no new error type, no new invariant, no ADR-worthy
  decision.

## Gate
- `make test-domain` — green (`-race`), including the 2 new tests.
- `go vet` / `golangci-lint run` — clean on every package that compiles
  without generated code.
- `gofmt -l .` — clean.
- `go build ./...` — fails only on the two `internal/gen`-dependent
  packages, as expected.
- HANDOFF.md T2 Definition of Done: AC met (table-driven tests pass;
  `ListCourtBookings` RPC wired; README's smoke test extended). No ADR
  needed.
