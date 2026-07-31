# Review — T3: CancelBooking

## Phase goal
HANDOFF.md T3: add the status → cancelled transition and wire it end to end.
AC: "test that cancelling then re-booking the same slot succeeds; REST
endpoint added."

## Process note
Same reasoning as T2 (docs/reviews/02-t2-list-court-bookings.md): the core
rule (`Booking.Cancel`, confirmed→cancelled only, illegal-transition
rejected) was already implemented and tested at the domain level in T0.
This phase is wiring the app/adapter layers around it, plus one genuinely
new thing worth a dedicated test: proving the *interaction* between
`CancelBooking` and `CreateBooking` — that cancelling actually frees the
slot for `EnsureNoConflict` to allow a re-booking, not just that the status
field flips. Reviewed directly against the checklist; no subagent debate
needed for the same reason as T2.

## What was done (TDD)
`internal/booking/app/cancel_booking_test.go` written and run red (compile
failure) before `Service.CancelBooking` existed. Three tests:
- `TestCancelBooking_FreesTheSlotForRebooking` — the literal AC: create a
  booking, prove the slot is genuinely taken (a second booking on the same
  court/time is rejected with `ErrCourtDoubleBooked`), cancel the first,
  then prove the same slot can now be booked by a *different* source (Game)
  — reinforcing D3b/F1 (the invariant and its release apply uniformly across
  sources, not just within one).
- `TestCancelBooking_UnknownBookingReturnsNotFound` — `GetByID` on a missing
  ID surfaces `domain.ErrBookingNotFound` through the use case unchanged.
- `TestCancelBooking_AlreadyCancelledIsRejected` — the T0-level illegal
  transition test extended through the app layer, so a double-cancel via
  the API (not just via a bare `domain.Booking`) is rejected.

`internal/booking/adapter/grpcapi/handler.go` gained `CancelBooking`; no new
error-code mapping was needed since `toStatus` already handles
`ErrBookingNotFound` (404) and `ErrIllegalStatusTransition` (400) from T0.

## Checklist self-review
- **DDD/invariant:** confirms, not just assumes, that "cancelled bookings
  don't hold the invariant" (CLAUDE.md rule 4) is true through the full
  create→cancel→re-create path, not only inside `domain.EnsureNoConflict`'s
  own unit tests.
- **TDD:** red confirmed before implementation.
- **Dependency rule:** unchanged shape; `Service.CancelBooking` orchestrates
  `repo.GetByID` -> `domain.Booking.Cancel` -> `repo.Update`, no new
  dependencies.
- **Nothing to debate:** no new error type, no new invariant, no ADR-worthy
  decision.

## Gate
- `make test-domain` — green (`-race`), including the 3 new tests.
- `go vet` / `golangci-lint run` — clean on every package that compiles
  without generated code.
- `gofmt -l .` — clean.
- `go build ./...` — fails only on the two `internal/gen`-dependent
  packages, as expected.
- HANDOFF.md T3 Definition of Done: AC met; README's smoke test extended
  with a cancel-then-rebook example. No ADR needed.
