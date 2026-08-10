package app

import (
	"context"
	"regexp"

	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsport "github.com/nhuthuynh/white-label/internal/competitions/port"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/payments/port"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
	socialplayport "github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// uuidShape matches the canonical 8-4-4-4-12 hex form internal/platform/idgen
// mints for every Payment id, and for the Booking/Registration ids that flow
// in as PayableID.
//
// Boundary guard for caller-supplied ids (T10.7, closing issue #97): the
// Postgres adapter's mustUUID panics on anything pgtype.UUID.Scan can't
// parse, and grpc installs no recover() of its own, so an unvalidated id off
// the wire could take the whole process down. Deliberately narrower than
// github.com/google/uuid's Validate, which accepts braced and `urn:uuid:`
// forms that pgtype rejects — a guard wider than the thing it protects is
// not a guard. The canonical write-up lives on internal/competitions/app's
// copy; reused here verbatim rather than re-derived, per this ticket's own
// instructions.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Payments context's application layer: it orchestrates the
// domain and its outbound ports, but holds no business rules itself — those
// live in internal/payments/domain.
//
// T6.2 gave Service just enough to drive the online path against
// port.PaymentProcessor; T6.3 added the offline-recording path. T6.4 added
// port.Repository (persistence) and switched the constructor to an
// options struct rather than growing positional args past three, per the
// sprint plan's kickoff note. T6.5 adds the fourth field the same kickoff
// note named up front, RegistrationUpdater socialplayport.
// RegistrationPaymentUpdater — T6.4 deliberately left it out because
// internal/socialplay didn't exist on that branch yet (see T6.4's PR
// description); T6.5 is the ticket that both merges Social Play into this
// lineage and adds the field, exactly as T6.4 predicted, to the *existing*
// ServiceOptions struct rather than a second constructor path. Payments
// importing internal/socialplay/port (and, transitively,
// internal/socialplay/domain for the PaymentStatus enum) is the expected,
// allowed direction per the context map
// (docs/agent-operating-handbook.md A1): Payments depends on Social Play,
// never the reverse — internal/socialplay/domain and internal/socialplay/app
// must never import anything under internal/payments.
//
// T10.6 (closes #96) adds a fifth field, CompetitionEntryUpdater
// competitionsport.CompetitionEntryPaymentUpdater, to the same
// ServiceOptions struct — exactly the pattern T6.5 established for
// RegistrationUpdater, mirrored for Competitions rather than Social Play.
// Payments importing internal/competitions/port (and, transitively,
// internal/competitions/domain for the PaymentStatus enum) is the expected,
// allowed direction per the context map: Payments depends on Competitions,
// never the reverse — internal/competitions/domain and internal/competitions/app
// must never import anything under internal/payments.
type Service struct {
	payments                port.Repository
	ids                     port.IDGenerator
	processor               port.PaymentProcessor
	registrationUpdater     socialplayport.RegistrationPaymentUpdater
	competitionEntryUpdater competitionsport.CompetitionEntryPaymentUpdater
}

// ServiceOptions is the dependency bundle for NewService (T6.4).
// RegistrationUpdater (T6.5) and CompetitionEntryUpdater (T10.6) are both
// optional: leaving either nil is fine for any caller/test that doesn't
// exercise the corresponding payable path, or doesn't care about that
// context's reconciliation side effect — see
// reconcileRegistrationPaymentStatus's and
// reconcileCompetitionEntryPaymentStatus's nil guards.
type ServiceOptions struct {
	Payments                port.Repository
	IDs                     port.IDGenerator
	Processor               port.PaymentProcessor
	RegistrationUpdater     socialplayport.RegistrationPaymentUpdater
	CompetitionEntryUpdater competitionsport.CompetitionEntryPaymentUpdater
}

// NewService constructs a Service from opts. IDs is required by every use
// case; Processor is required for the online path (CreateOnlinePayment,
// ConfirmOnlinePayment); Payments is required once persistence is wired
// (T6.4's Postgres adapter, cmd/server). Tests that only exercise
// domain/orchestration logic without persistence may leave Payments nil —
// see fixtures_test.go's fixedIDs and stripestub.NewProcessor for the
// deterministic test doubles used in place of Processor/IDs.
func NewService(opts ServiceOptions) *Service {
	return &Service{
		payments:                opts.Payments,
		ids:                     opts.IDs,
		processor:               opts.Processor,
		registrationUpdater:     opts.RegistrationUpdater,
		competitionEntryUpdater: opts.CompetitionEntryUpdater,
	}
}

// Payments exposes the persistence port so a caller that genuinely needs
// direct repository access can get it without cmd/server wiring a second
// handle. Prefer GetPayment below for the id -> Payment lookup
// ConfirmOnlinePayment's handler needs — Payments() is kept for any other
// caller, but the guarded lookup is the one path that used to bypass the app
// layer's boundary validation entirely (T10.7, closing issue #97).
func (s *Service) Payments() port.Repository {
	return s.payments
}

// GetPayment resolves a Payment by id, or domain.ErrPaymentNotFound.
//
// Added by T10.7 (closing issue #97): before this method existed,
// internal/payments/adapter/grpcapi's ConfirmOnlinePayment handler called
// h.svc.Payments().GetByID(ctx, req.GetPaymentId()) directly — the ONE
// caller-supplied-id lookup in this codebase that reached a repository
// straight from the gRPC adapter rather than through app.Service, and so was
// the one place PR #89's Layer 2 pattern (a uuidShape guard living next to
// every other get-shaped read) couldn't be applied without this change. A
// malformed id is answered exactly like an unknown one — the same
// domain.ErrPaymentNotFound port.Repository.GetByID already returns for a
// miss — rather than reaching the Postgres adapter's mustUUID, which panics
// on non-UUID input.
func (s *Service) GetPayment(ctx context.Context, id string) (domain.Payment, error) {
	if !uuidShape.MatchString(id) {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return s.payments.GetByID(ctx, id)
}

// CreateOnlinePaymentInput is the use-case input for starting an online
// Payment.
//
// ActorUserID/EntrantPlayerID/AssignedCompetitionAdminUserIDs (T10.6, closes
// #96) are new, competition_entry-scoped fields — the online-path analogue
// of RecordOfflinePaymentInput's own caller-supplied authorization facts
// (internal/payments has no live join to Competitions' database, mirroring
// the reasoning that section's doc comment already gives for Booking/Social
// Play). They are validated ONLY when PayableType is
// PayableTypeCompetitionEntry (see authorizeOnlineCreation): every other
// payable type accepted by this method today (booking, registration,
// no_show_fee) is unaffected and continues to require no actor at all,
// exactly as before T10.6 — extending that same authorization posture to
// those pre-existing types is explicitly out of this ticket's scope.
type CreateOnlinePaymentInput struct {
	PayableType domain.PayableType
	PayableID   string
	Amount      domain.Money

	ActorUserID string

	EntrantPlayerID                 string
	AssignedCompetitionAdminUserIDs []string
}

// CreateOnlinePayment builds an unpaid, online Payment (domain.NewPayment),
// authorizes it with the payment processor (port.PaymentProcessor.
// CreateIntent), and persists it (port.Repository.Create, T6.4). On
// success, the returned Payment carries the processor's intent reference
// as StripeReference but is still unpaid — creating an intent authorizes
// funds, it does not capture them.
//
// authorizeOnlineCreation runs first (T10.6): for a PayableTypeCompetitionEntry
// payable it requires the actor to be the entrant or an assigned Competition
// Admin, mirroring RecordOfflinePayment's authorization-before-construction
// ordering so an unauthorized actor never learns anything about why the
// input would otherwise be invalid. Every other payable type is unaffected
// (the check is a no-op for them, see authorizeOnlineCreation's doc
// comment).
//
// The Postgres adapter's UNIQUE (payable_type, payable_id) constraint is
// the authoritative guard against recording two Payments for the same
// payable action (CLAUDE.md rule 4); Create translates a 23505 violation
// into domain.ErrPaymentAlreadyRecorded (CLAUDE.md rule 5).
func (s *Service) CreateOnlinePayment(ctx context.Context, in CreateOnlinePaymentInput) (domain.Payment, error) {
	// Authorization runs first (T10.6), before the shape guard below (T10.7)
	// — same ordering RecordOfflinePayment already uses, so an unauthorized
	// actor never learns anything about whether the rest of the input would
	// otherwise be valid, malformed PayableID included.
	if err := authorizeOnlineCreation(in); err != nil {
		return domain.Payment{}, err
	}

	// A malformed PayableID is rejected with the same ErrEmptyPayableID/
	// InvalidArgument domain.NewPayment below already returns for an EMPTY
	// PayableID (T10.7, closing issue #97) — extending that existing
	// sentinel to a non-empty-but-malformed-shape value rather than
	// inventing a new one. PayableID has no existence check in this context
	// (see malformed_id_test.go's doc comment), so unlike a get-shaped
	// handler there is no "unknown but well-formed" NotFound answer to
	// match instead; this is the closest existing precedent on the same
	// field. Checked here, before NewPayment, so the malformed value never
	// reaches port.PaymentProcessor.CreateIntent or port.Repository.Create
	// — both would otherwise see it, and Create's mustUUID panics on it.
	if !uuidShape.MatchString(in.PayableID) {
		return domain.Payment{}, domain.ErrEmptyPayableID
	}

	p, err := domain.NewPayment(s.ids.NewID(), in.PayableType, in.PayableID, in.Amount, domain.MethodOnline, "")
	if err != nil {
		return domain.Payment{}, err
	}

	intentRef, err := s.processor.CreateIntent(ctx, in.Amount, in.PayableID)
	if err != nil {
		return domain.Payment{}, err
	}
	p.StripeReference = intentRef

	return s.payments.Create(ctx, p)
}

// ConfirmOnlinePayment captures funds for p (an online Payment previously
// returned by CreateOnlinePayment, typically reloaded from persistence by
// the caller — see internal/payments/adapter/grpcapi's ConfirmOnlinePayment
// handler, T6.4) via port.PaymentProcessor.CapturePayment, then transitions
// p to paid (domain.Payment.MarkPaid) and persists the transition
// (port.Repository.Update, T6.4) on success.
//
// A declined card (domain.ErrPaymentDeclined) or any other processor
// failure (domain.ErrPaymentProcessorUnavailable) is not an illegal state
// transition — it's a capture that simply didn't happen — so p is
// returned unchanged (still whatever status it was passed in as,
// typically unpaid) alongside the error, and nothing is persisted, rather
// than mutating into or persisting some half-applied state.
func (s *Service) ConfirmOnlinePayment(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	// Any processor-side failure (declined or otherwise unavailable) means
	// no capture happened, so p is returned exactly as it was passed in —
	// there is nothing to roll back because MarkPaid was never called, and
	// nothing is persisted.
	if err := s.processor.CapturePayment(ctx, p.StripeReference); err != nil {
		return p, err
	}

	if err := p.MarkPaid(domain.MethodOnline, p.StripeReference); err != nil {
		return p, err
	}

	updated, err := s.payments.Update(ctx, p)
	if err != nil {
		return updated, err
	}

	if err := s.reconcileRegistrationPaymentStatus(ctx, updated); err != nil {
		return updated, err
	}
	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, updated); err != nil {
		return updated, err
	}
	return updated, nil
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
// database (T6.5 wires the Social Play projection) — so the caller
// supplies the ownership/assignment facts the authorization check needs
// directly, rather than this package querying another context's
// repository:
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
//
// EntrantPlayerID and AssignedCompetitionAdminUserIDs (T10.6, closes #96)
// are the equivalent caller-supplied facts for a PayableTypeCompetitionEntry
// payable — the Competitions analogue of GameHostID/AssignedGameAdminUserIDs
// above, with one deliberate difference: the authorized actor set for a
// competition entry is "the entrant OR an assigned Competition Admin", not
// "the Host or an assigned Admin". A competition entry's payment is most
// often recorded by the entrant paying for their own place (the issue's own
// framing: "as a Player entering a Competition, I want to pay online"), so
// EntrantPlayerID — the CompetitionEntry's own PlayerID — is itself an
// authorized actor, unlike GameHostID/BookingHostID which never name the
// paying Player.
type RecordOfflinePaymentInput struct {
	PayableType domain.PayableType
	PayableID   string
	Amount      domain.Money
	ActorUserID string

	BookingHostID string

	GameHostID               string
	AssignedGameAdminUserIDs []string

	EntrantPlayerID                 string
	AssignedCompetitionAdminUserIDs []string
}

// RecordOfflinePayment builds a Payment (domain.NewPayment, Method:
// offline, RecordedByUserID: in.ActorUserID), immediately marks it paid
// (domain.Payment.MarkPaid) — an offline recording is the payment event,
// there is no separate confirmation step — and persists it
// (port.Repository.Create, T6.4). Authorization is checked first, before
// any domain construction, so an unauthorized actor never learns anything
// about why the input would otherwise be invalid.
//
// The Postgres adapter's UNIQUE (payable_type, payable_id) constraint
// (T6.4) is the authoritative guard against recording a duplicate Payment
// for the same payable action; Create translates a 23505 violation into
// domain.ErrPaymentAlreadyRecorded.
func (s *Service) RecordOfflinePayment(ctx context.Context, in RecordOfflinePaymentInput) (domain.Payment, error) {
	if err := authorizeOfflineRecording(in); err != nil {
		return domain.Payment{}, err
	}

	// Same T10.7 guard CreateOnlinePayment applies to the same field, kept
	// after authorization so an unauthorized actor learns nothing about
	// whether their PayableID was even well-formed (mirrors the existing
	// ordering, where domain.NewPayment's own empty-PayableID check already
	// ran after authorizeOfflineRecording).
	if !uuidShape.MatchString(in.PayableID) {
		return domain.Payment{}, domain.ErrEmptyPayableID
	}

	p, err := domain.NewPayment(s.ids.NewID(), in.PayableType, in.PayableID, in.Amount, domain.MethodOffline, in.ActorUserID)
	if err != nil {
		return domain.Payment{}, err
	}

	if err := p.MarkPaid(domain.MethodOffline, ""); err != nil {
		return domain.Payment{}, err
	}

	created, err := s.payments.Create(ctx, p)
	if err != nil {
		return domain.Payment{}, err
	}

	if err := s.reconcileRegistrationPaymentStatus(ctx, created); err != nil {
		return created, err
	}
	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, created); err != nil {
		return created, err
	}
	return created, nil
}

// reconcileRegistrationPaymentStatus pushes a successfully-paid Payment's
// status through to Social Play (T6.5), when, and only when, it pays for a
// Registration's own seat: PayableTypeBooking never calls this (required
// test), and PayableTypeNoShowFee is deliberately excluded too even though
// it also targets a Registration's Game — a no-show fee is a separate
// charge, not the Registration's own payment status, and the ticket's
// instructions scope the reconciliation to "PayableType == registration"
// specifically (see the PR description's judgment-call note).
//
// registrationUpdater is optional (ServiceOptions doc comment) — a Service
// built without one (e.g. most pre-T6.5 tests, or a deployment that hasn't
// wired Social Play for some reason) simply skips this step rather than
// panicking on a nil interface call.
//
// A failure here is returned to the caller rather than swallowed: the
// Payment itself is already correctly persisted as paid (the source of
// truth is not at risk), but the Social Play projection is now stale until
// the caller retries or a background reconciliation job picks it up — this
// method deliberately does not attempt to undo the already-persisted
// Payment transition, since that failure mode (a successful payment that
// gets un-recorded because an unrelated read-model update failed) would be
// worse than a stale projection.
func (s *Service) reconcileRegistrationPaymentStatus(ctx context.Context, p domain.Payment) error {
	if p.PayableType != domain.PayableTypeRegistration {
		return nil
	}
	if s.registrationUpdater == nil {
		return nil
	}
	return s.registrationUpdater.UpdatePaymentStatus(ctx, p.PayableID, socialplaydomain.PaymentStatusPaid)
}

// reconcileCompetitionEntryPaymentStatus pushes a successfully-paid
// Payment's status through to Competitions (T10.6, closes #96), when, and
// only when, it pays for a CompetitionEntry: called only for
// PayableTypeCompetitionEntry, mirroring reconcileRegistrationPaymentStatus
// exactly, alongside it (both are called, unconditionally, from
// ConfirmOnlinePayment and RecordOfflinePayment — each is a no-op for the
// PayableType the other one owns, so only ever one of the two ever actually
// calls its port for a given Payment).
//
// competitionEntryUpdater is optional (ServiceOptions doc comment) — a
// Service built without one (e.g. most pre-T10.6 tests, or a deployment
// that hasn't wired Competitions for some reason) simply skips this step
// rather than panicking on a nil interface call.
//
// A failure here is returned to the caller rather than swallowed, for the
// identical reason reconcileRegistrationPaymentStatus's doc comment gives:
// the Payment itself is already correctly persisted as paid (the source of
// truth is not at risk), but the Competitions projection is now stale until
// the caller retries or a background reconciliation job picks it up.
func (s *Service) reconcileCompetitionEntryPaymentStatus(ctx context.Context, p domain.Payment) error {
	if p.PayableType != domain.PayableTypeCompetitionEntry {
		return nil
	}
	if s.competitionEntryUpdater == nil {
		return nil
	}
	return s.competitionEntryUpdater.UpdatePaymentStatus(ctx, p.PayableID, competitionsdomain.PaymentStatusPaid)
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

	// PayableTypeCompetitionEntry (T10.6, closes #96): the authorized actor
	// set is "the entrant OR an assigned Competition Admin" — see
	// RecordOfflinePaymentInput's doc comment for why this differs from the
	// Registration/no_show_fee branch below (which never treats the paying
	// Player themself as authorized).
	if in.PayableType == domain.PayableTypeCompetitionEntry {
		if in.EntrantPlayerID != "" && in.ActorUserID == in.EntrantPlayerID {
			return nil
		}
		for _, adminID := range in.AssignedCompetitionAdminUserIDs {
			if adminID != "" && adminID == in.ActorUserID {
				return nil
			}
		}
		return domain.ErrNotPaymentRecorder
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

// authorizeOnlineCreation is CreateOnlinePayment's authorization check
// (T10.6, closes #96) — deliberately scoped to PayableTypeCompetitionEntry
// only. Every other payable type CreateOnlinePayment accepts today
// (booking, registration, no_show_fee) had no online-path authorization
// check before this ticket and keeps none now: extending this check to
// those pre-existing types is real, separate scope this ticket does not
// take on (see CreateOnlinePaymentInput's doc comment). The authorized
// actor set mirrors authorizeOfflineRecording's PayableTypeCompetitionEntry
// branch exactly — the entrant, or an assigned Competition Admin — since
// both are the same underlying rule applied to the two different points a
// Payment can be created at (online-intent-creation vs. offline-recording).
func authorizeOnlineCreation(in CreateOnlinePaymentInput) error {
	if in.PayableType != domain.PayableTypeCompetitionEntry {
		return nil
	}

	if in.ActorUserID == "" {
		return domain.ErrNotPaymentRecorder
	}
	if in.EntrantPlayerID != "" && in.ActorUserID == in.EntrantPlayerID {
		return nil
	}
	for _, adminID := range in.AssignedCompetitionAdminUserIDs {
		if adminID != "" && adminID == in.ActorUserID {
			return nil
		}
	}
	return domain.ErrNotPaymentRecorder
}
