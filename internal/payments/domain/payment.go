package domain

// PayableType identifies the kind of payable action a Payment records
// money against. String-backed and extensible by design (t6-sprint-plan.md
// T6.1 kickoff note): booking, registration, and no_show_fee were the
// values T6 produced; T10.6 (closes #96) adds competition_entry, in the
// same PR that first produces its caller (internal/payments/adapter/
// competitions), per this type's own extension policy stated below.
// recurring_hire and subscription are named as future Payment targets by
// the glossary's own Payment definition (docs/agent-operating-handbook.md
// A2) but are deliberately not added as enum values here — nothing
// produces them yet, and speculative unused enum values are exactly what
// the PE dossier warns against over-engineering. Add them in the same PR
// that first produces one.
type PayableType string

const (
	// PayableTypeBooking is a Payment against a Booking (a direct court
	// hire, recurring hire, or a Booking created on behalf of a Game or
	// Competition).
	PayableTypeBooking PayableType = "booking"
	// PayableTypeRegistration is a Payment against a Social Play
	// Registration (a Player's spot in a Game).
	PayableTypeRegistration PayableType = "registration"
	// PayableTypeNoShowFee is a Payment charging a fee for a no-show.
	// T6 ships this as a payable type a Game Admin can record manually
	// through the same offline-recording path as any other payment
	// (T6.3) — there is no automatic no-show detection or trigger yet;
	// see the T6 kickoff note's P1 #8 resolution.
	PayableTypeNoShowFee PayableType = "no_show_fee"
	// PayableTypeCompetitionEntry is a Payment against a Competitions
	// CompetitionEntry (T10.6, closes #96). Before this value existed,
	// PayableTypeRegistration was the only non-Booking option, and
	// routing a Competition entry through it would have written the
	// entry's ID into Social Play's Registration table via
	// reconcileRegistrationPaymentStatus — a real, money-adjacent
	// data-corruption bug T9.6/T9.7 independently found and this value
	// exists specifically to prevent. See
	// internal/payments/adapter/competitions (mirrors
	// internal/payments/adapter/socialplay exactly) for the port this
	// value routes to.
	PayableTypeCompetitionEntry PayableType = "competition_entry"
)

// IsValid reports whether t is one of the payable types this codebase
// recognises.
func (t PayableType) IsValid() bool {
	switch t {
	case PayableTypeBooking, PayableTypeRegistration, PayableTypeNoShowFee, PayableTypeCompetitionEntry:
		return true
	default:
		return false
	}
}

// Method identifies how a Payment was, or will be, paid.
type Method string

const (
	MethodOnline  Method = "online"
	MethodOffline Method = "offline"
)

// Status is a Payment's lifecycle state.
type Status string

const (
	StatusUnpaid   Status = "unpaid"
	StatusPaid     Status = "paid"
	StatusRefunded Status = "refunded"
)

// Payment is the single aggregate recording money for any payable action
// (docs/agent-operating-handbook.md A2's Payment definition). StripeReference
// is empty for an offline Payment (there is no Stripe involvement) and is
// populated once an online Payment has an intent/charge reference.
// RecordedByUserID is the verified actor a Payment belongs to: the acting
// Host/Game Admin who recorded it for an offline Payment (T6.3), and — as of
// T13.7, closing issue #148 — the actor who created the intent for an online
// one. It used to be deliberately empty on the online path ("nobody recorded
// it — Stripe's confirmation did"), which left ConfirmOnlinePayment with no
// ownership fact to check and made holding a payment_id the same thing as
// holding the money.
//
// UPDATED BY T28.1 (closes the Payments third of #164, dated deliberately —
// see app.authorizeOnlineConfirmation's own dated comment for why): it was a
// subject, not a User uuid, on both paths from T13.7 through T28's start,
// per ADR-0014 §5a's explicit deferral of this context. As of T28.1 it is a
// **User.ID uuid** on both paths, per ADR-0017's ruling — the column behind
// it, `payments.recorded_by_user_id`, is now `uuid REFERENCES identity_users
// (id)` (nullable), backfilled by
// db/migrations/0024_payments_recorded_by_user_id_uuid.sql. The Go field
// stays a plain `string` (a uuid string, not a Go UUID type), matching every
// other actor field in this codebase — this migration changed storage and
// identifier space, not Go representation.
//
// It can still be empty on a Payment read back from storage, for two
// distinct reasons that both resolve to the same safe answer: every online
// row written before T13.7 has no owner at all, and — as of T28.1 — a row
// whose pre-migration subject matched no identity_users row (an orphan,
// ADR-0017 Decision 3) reads back NULL/empty rather than a resolved uuid.
// app.authorizeOnlineConfirmation treats both identically: "nobody may
// confirm this", never "anybody may".
type Payment struct {
	ID               string
	PayableType      PayableType
	PayableID        string
	Amount           Money
	Method           Method
	Status           Status
	StripeReference  string
	RecordedByUserID string
}

// NewPayment constructs an unpaid Payment, validating the invariants that
// don't require knowledge of other payments (distinctness against other
// payments is EnsureOnePaymentPerPayable's job, mirroring how
// domain.NewBooking/EnsureNoConflict split those responsibilities).
//
// id is supplied by the caller (the app layer, via its IDGenerator port),
// the same pattern domain.NewBooking uses — Payment does not generate its
// own identity.
func NewPayment(id string, payableType PayableType, payableID string, amount Money, method Method, recordedByUserID string) (Payment, error) {
	if payableID == "" {
		return Payment{}, ErrEmptyPayableID
	}
	if !payableType.IsValid() {
		return Payment{}, ErrInvalidPayableType
	}
	if amount.Cents <= 0 {
		return Payment{}, ErrInvalidAmount
	}
	if !isValidCurrencyCode(amount.Currency) {
		return Payment{}, ErrInvalidCurrency
	}

	return Payment{
		ID:               id,
		PayableType:      payableType,
		PayableID:        payableID,
		Amount:           amount,
		Method:           method,
		Status:           StatusUnpaid,
		RecordedByUserID: recordedByUserID,
	}, nil
}

// MarkPaid transitions a Payment to paid, recording the method it was paid
// by and a reference (a Stripe payment-intent id for the online path, or
// left as whatever the caller passes — empty is fine — for the offline
// path where there is no processor reference). Legal only from unpaid;
// marking an already-paid or refunded Payment paid again is rejected
// rather than silently accepted, mirroring Booking.Cancel()'s reasoning
// for rejecting a double-cancel.
func (p *Payment) MarkPaid(method Method, ref string) error {
	if p.Status != StatusUnpaid {
		return ErrIllegalStatusTransition
	}
	p.Method = method
	p.StripeReference = ref
	p.Status = StatusPaid
	return nil
}

// Refund transitions a paid Payment to refunded. Legal only from paid —
// refunding an unpaid Payment is explicitly illegal (nothing was paid to
// refund yet). refunded is terminal: neither MarkPaid nor Refund ever
// transitions out of it again.
func (p *Payment) Refund() error {
	if p.Status != StatusPaid {
		return ErrIllegalStatusTransition
	}
	p.Status = StatusRefunded
	return nil
}

// EnsureOnePaymentPerPayable is the domain-side fast pre-check mirroring
// EnsureNoConflict's role for Booking (CLAUDE.md rule 4): a
// (PayableType, PayableID) pair may have at most one Payment row. This is
// a *distinctness* invariant, not a *capacity* one — it is closed
// authoritatively by a Postgres UNIQUE(payable_type, payable_id)
// constraint in T6.4, not a locking trigger (contrast with Social Play's
// capacity-guard trigger, which this is not analogous to: distinctness
// races are closed by a unique index, counting/ordering races need a row
// lock).
func EnsureOnePaymentPerPayable(candidate Payment, existing []Payment) error {
	for _, other := range existing {
		if other.ID == candidate.ID {
			continue
		}
		if other.PayableType == candidate.PayableType && other.PayableID == candidate.PayableID {
			return ErrPaymentAlreadyRecorded
		}
	}
	return nil
}
