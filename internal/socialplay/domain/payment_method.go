package domain

// PaymentMethod is a Game-level field: the payment method(s) a Host accepts
// for that Game's registrations. Deliberately a distinct type from
// Registration.PaymentStatus (registration.go) — the two must not be
// conflated:
//
//   - PaymentMethod (this type) lives on Game, is set once by the Host at
//     scheduling time (via NewGame), and describes what the Host is willing
//     to accept (online payment, cash/offline, or either) — a policy
//     statement about the Game itself.
//   - PaymentStatus lives on Registration, is set per-registration
//     (Register's initial "unpaid" default, then only ever changed via
//     Registration.MarkPaymentStatus), and describes whether one specific
//     Player's spot has actually been paid for — a fact about that
//     Registration, reported by the Payments context (T6.5).
//
// A Game's PaymentMethod does not, by itself, enforce which of Payments'
// CreateOnlinePayment/RecordOfflinePayment RPCs the backend will accept
// (docs/process/t8-sprint-plan.md T8.6 kickoff note) — it is read-only
// UI guidance (T8.9) until a real port.GamePaymentPolicy-shaped enforcement
// mechanism is designed, which is explicitly deferred, not built here.
type PaymentMethod string

const (
	PaymentMethodOnline PaymentMethod = "online"
	PaymentMethodCash   PaymentMethod = "cash"
	PaymentMethodEither PaymentMethod = "either"
)

// IsValid reports whether m is one of PaymentMethod's closed enum values.
// NewGame uses this to keep the field a closed enum, mirroring
// PaymentStatus.IsValid's role for the Registration-level field.
func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentMethodOnline, PaymentMethodCash, PaymentMethodEither:
		return true
	default:
		return false
	}
}
