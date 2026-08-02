package app

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/payments/port"
)

// Service is the Payments context's application layer: it orchestrates the
// domain and its outbound ports, but holds no business rules itself —
// those live in internal/payments/domain. T6.3 gives Service just enough
// to drive the offline-recording path; persistence (port.Repository) lands
// in T6.4 and will grow this into an options-struct constructor per the
// sprint plan's kickoff note, rather than growing positional args past
// three.
type Service struct {
	ids port.IDGenerator
}

// NewService constructs a Service. ids is a required port — tests inject a
// fixed-sequence IDGenerator so behaviour stays deterministic without a
// real UUID library.
func NewService(ids port.IDGenerator) *Service {
	return &Service{ids: ids}
}

// RecordOfflinePaymentInput is the use-case input for recording an offline
// Payment (cash, bank transfer, or a manually-charged no-show fee) — the
// offline half of HANDOFF.md's T6 AC. There is no separate "intent" step
// the way the online/Stripe path has one (T6.2's CreateOnlinePayment /
// ConfirmOnlinePayment): recording the Payment *is* the payment event.
//
// Authorization is actor-scoped (T6.3), mirroring T5.2/T5.5's
// ErrNotRegistrationOwner pattern exactly, including its known-gap
// caveat: ActorUserID is a request-supplied field, not a verified
// identity — there is no JWT yet (HANDOFF.md's existing Auth cross-cutting
// note, which T5.5 already established and this ticket does not
// contradict). This proves an *object-level* check given a claimed actor,
// not real authentication.
//
// internal/payments has no live join to Social Play's or Booking's
// database in T6 (T6.4 wires persistence; T6.5 wires the Social Play
// projection) — so the caller supplies the ownership/assignment facts the
// authorization check needs directly, rather than this package querying
// another context's repository:
//
//   - BookingHostID is the user id of the Host who owns the Booking's
//     Game/Competition, required for a PayableTypeBooking payable. A
//     Booking with no Host at all (a direct court hire —
//     SourceIndividual/SourceRecurringHire) is explicitly out of scope for
//     offline recording in T6 (narrower-than-spec scope cut, see the PR
//     description): leave BookingHostID empty and RecordOfflinePayment
//     rejects with ErrNotPaymentRecorder rather than silently allowing it.
//   - GameHostID and AssignedGameAdminUserIDs are, for a
//     PayableTypeRegistration or PayableTypeNoShowFee payable, the
//     Registration's Game's Host id and the set of user ids currently
//     assigned as a Game Admin for that Game — the glossary's own
//     definition of Game Admin's scope (agent-operating-handbook.md A2:
//     "may record offline Payments... scoped to the specific
//     Game/Competition they are assigned to"). The actor must be the Host
//     or one of the assigned admins. T5 did not build a persisted
//     Game-Admin-assignment mechanism, so this is the minimal version
//     needed to test this ticket's authorization rule: a caller-supplied
//     list, not a new socialplay feature (see the PR description's
//     judgment-call note).
//
// PayableTypeNoShowFee is authorized identically to PayableTypeRegistration
// (Host-or-assigned-Game-Admin) — deliberately not a third branch, since a
// no-show fee is, per the T6 kickoff note, "structurally just another
// payable action" against the same Registration/Game, and the required
// test (P1 #8) is that recording one takes zero new code paths.
type RecordOfflinePaymentInput struct {
	PayableType domain.PayableType
	PayableID   string
	Amount      domain.Money
	ActorUserID string

	BookingHostID string

	GameHostID               string
	AssignedGameAdminUserIDs []string
}

// RecordOfflinePayment builds a Payment (domain.NewPayment, Method:
// offline, RecordedByUserID: in.ActorUserID) and immediately marks it paid
// (domain.Payment.MarkPaid) — an offline recording is the payment event,
// there is no separate confirmation step. Authorization is checked first,
// before any domain construction, so an unauthorized actor never learns
// anything about why the input would otherwise be invalid.
//
// Persisting the returned Payment is the caller's responsibility (T6.4's
// repository port), the same "this method's job ends at the Payment to
// persist" shape T6.2's CreateOnlinePayment uses.
func (s *Service) RecordOfflinePayment(ctx context.Context, in RecordOfflinePaymentInput) (domain.Payment, error) {
	if err := authorizeOfflineRecording(in); err != nil {
		return domain.Payment{}, err
	}

	p, err := domain.NewPayment(s.ids.NewID(), in.PayableType, in.PayableID, in.Amount, domain.MethodOffline, in.ActorUserID)
	if err != nil {
		return domain.Payment{}, err
	}

	if err := p.MarkPaid(domain.MethodOffline, ""); err != nil {
		return domain.Payment{}, err
	}

	return p, nil
}

// authorizeOfflineRecording is the actor-scoped (BOLA-shaped) check T6.3
// requires, mirroring socialplay.domain.Registration.Cancel's ownership
// check but living in the app layer rather than the domain: unlike
// Registration.Cancel, this check needs facts (Host id, Game Admin
// assignments) that come from outside the Payment aggregate itself, so it
// can't be expressed as a method on domain.Payment the way Cancel is a
// method on Registration.
//
//   - PayableTypeBooking: legal only when ActorUserID matches
//     BookingHostID, and BookingHostID must be non-empty (a Host-less
//     Booking is out of scope for T6.3, see RecordOfflinePaymentInput's
//     doc comment).
//   - Everything else (PayableTypeRegistration, PayableTypeNoShowFee):
//     legal when ActorUserID matches GameHostID, or appears in
//     AssignedGameAdminUserIDs.
//
// A mismatched or missing actor always returns ErrNotPaymentRecorder, the
// same sentinel regardless of which branch rejected it — a caller does not
// get to distinguish "wrong payable type" from "wrong actor" from this
// error alone, matching ErrNotRegistrationOwner's equally flat shape.
func authorizeOfflineRecording(in RecordOfflinePaymentInput) error {
	if in.ActorUserID == "" {
		return domain.ErrNotPaymentRecorder
	}

	if in.PayableType == domain.PayableTypeBooking {
		if in.BookingHostID == "" || in.ActorUserID != in.BookingHostID {
			return domain.ErrNotPaymentRecorder
		}
		return nil
	}

	if in.GameHostID != "" && in.ActorUserID == in.GameHostID {
		return nil
	}
	for _, adminID := range in.AssignedGameAdminUserIDs {
		if adminID != "" && adminID == in.ActorUserID {
			return nil
		}
	}
	return domain.ErrNotPaymentRecorder
}
