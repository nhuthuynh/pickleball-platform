package domain

// PaymentMethod is a Competition-level field: the payment method(s) a Host
// accepts for that Competition's entries. Same type, same three values, and
// the same deliberate distinction from a per-entry PaymentStatus as
// internal/socialplay/domain.PaymentMethod (T8.6) — see that file for the
// full reasoning, which applies here unchanged:
//
//   - PaymentMethod (this type) lives on Competition, is set once by the
//     Host at scheduling time (via NewCompetition), and describes what the
//     Host is willing to accept — a policy statement about the Competition.
//   - PaymentStatus (entry.go) lives on CompetitionEntry and describes
//     whether one specific entrant's place has actually been paid for — a
//     fact about that entry.
//
// As in Social Play, a Competition's PaymentMethod does not by itself
// enforce which Payments RPC the backend will accept; it is read-only
// guidance for clients until a real enforcement mechanism is designed. That
// deferral is inherited knowingly, not rediscovered — see
// docs/process/t8-sprint-plan.md's T8.6 kickoff note.
type PaymentMethod string

const (
	PaymentMethodOnline PaymentMethod = "online"
	PaymentMethodCash   PaymentMethod = "cash"
	PaymentMethodEither PaymentMethod = "either"
)

// IsValid reports whether m is one of PaymentMethod's closed enum values.
// NewCompetition uses this to keep the field a closed enum.
func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentMethodOnline, PaymentMethodCash, PaymentMethodEither:
		return true
	default:
		return false
	}
}
