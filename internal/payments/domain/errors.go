package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures (e.g. a
// Postgres 23505 unique-violation on (payable_type, payable_id), T6.4)
// into these — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package or with internal/socialplay/domain (each
// bounded context owns its own errors — CLAUDE.md rule 3).
var (
	ErrEmptyPayableID     = errors.New("payments: payable id is required")
	ErrInvalidPayableType = errors.New("payments: invalid payable type")
	ErrInvalidAmount      = errors.New("payments: amount must be greater than zero")
	ErrInvalidCurrency    = errors.New("payments: currency must be a 3-letter ISO 4217 code")

	ErrIllegalStatusTransition = errors.New("payments: illegal status transition")

	// ErrPaymentAlreadyRecorded is returned both by the domain-side
	// EnsureOnePaymentPerPayable pre-check and, in T6.4, by the Postgres
	// adapter translating a 23505 unique-violation on
	// (payable_type, payable_id) — one sentinel for the same invariant at
	// both layers, mirroring how booking.ErrCourtDoubleBooked is returned
	// by both EnsureNoConflict and the EXCLUDE-constraint translation.
	ErrPaymentAlreadyRecorded = errors.New("payments: a payment already exists for this payable action")

	// ErrNotPaymentRecorder is T6.3's actor-scoped authorization sentinel:
	// returned by app.Service.RecordOfflinePayment when the caller-supplied
	// ActorUserID does not match the Host/Game-Admin ownership facts the
	// caller supplied for the payable being recorded against. A distinct,
	// stable sentinel (not folded into ErrIllegalStatusTransition) so a
	// mismatched actor is distinguishable by type from a legal-but-wrong-
	// state transition, mirroring socialplay.domain.ErrNotRegistrationOwner
	// (T5.2/T5.5) exactly.
	ErrNotPaymentRecorder = errors.New("payments: actor is not authorized to record an offline payment for this payable action")
)
