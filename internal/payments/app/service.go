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
//
// T16.2 (closes #168) adds five more fields — RegistrationLookup, GameLookup,
// GameAdminReader, EntryLookup, CompetitionAdminReader — the resolver ports
// authorizeOfflineRecording uses to verify a Registration's Game's
// Host/admins and a CompetitionEntry's Competition's entrant/admins against
// the real Social Play/Competitions stores, replacing the caller-supplied
// GameHostID/AssignedGameAdminUserIDs/EntrantPlayerID/
// AssignedCompetitionAdminUserIDs fields RecordOfflinePaymentInput and
// RefundPaymentInput carried before this ticket (see those structs' own doc
// comments, and port.GameAdminReader's doc comment for the full history of
// why T15.5 built these ports but could not wire them in). Unlike
// RegistrationUpdater/CompetitionEntryUpdater above, these five are not
// "skip the side effect if unset" optional — they are consulted by
// authorizeOfflineRecording itself, a security-critical path, so a nil
// resolver fails CLOSED (see authorizeGameRecording/
// authorizeCompetitionEntryRecording's own nil guards): an operator who
// forgets to wire one gets "nobody can record this payable type", never
// "anybody can".
type Service struct {
	payments                port.Repository
	ids                     port.IDGenerator
	processor               port.PaymentProcessor
	registrationUpdater     socialplayport.RegistrationPaymentUpdater
	competitionEntryUpdater competitionsport.CompetitionEntryPaymentUpdater

	registrationLookup     port.RegistrationLookup
	gameLookup             port.GameLookup
	gameAdminReader        port.GameAdminReader
	entryLookup            port.EntryLookup
	competitionAdminReader port.CompetitionAdminReader
}

// ServiceOptions is the dependency bundle for NewService (T6.4).
// RegistrationUpdater (T6.5) and CompetitionEntryUpdater (T10.6) are both
// optional: leaving either nil is fine for any caller/test that doesn't
// exercise the corresponding payable path, or doesn't care about that
// context's reconciliation side effect — see
// reconcileRegistrationPaymentStatus's and
// reconcileCompetitionEntryPaymentStatus's nil guards.
//
// RegistrationLookup/GameLookup/GameAdminReader/EntryLookup/
// CompetitionAdminReader (T16.2, closes #168) are, by contrast, load-bearing
// for authorization correctness on the payable types they cover — see
// Service's own doc comment for the fail-closed behaviour when one is left
// nil. A test or deployment that only exercises PayableTypeBooking (the one
// payable type these five ports do not touch, per instruction 6's scope cut)
// may still leave all five nil, exactly as booking-only tests already leave
// RegistrationUpdater/CompetitionEntryUpdater nil today.
type ServiceOptions struct {
	Payments                port.Repository
	IDs                     port.IDGenerator
	Processor               port.PaymentProcessor
	RegistrationUpdater     socialplayport.RegistrationPaymentUpdater
	CompetitionEntryUpdater competitionsport.CompetitionEntryPaymentUpdater

	RegistrationLookup     port.RegistrationLookup
	GameLookup             port.GameLookup
	GameAdminReader        port.GameAdminReader
	EntryLookup            port.EntryLookup
	CompetitionAdminReader port.CompetitionAdminReader
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

		registrationLookup:     opts.RegistrationLookup,
		gameLookup:             opts.GameLookup,
		gameAdminReader:        opts.GameAdminReader,
		entryLookup:            opts.EntryLookup,
		competitionAdminReader: opts.CompetitionAdminReader,
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

	// in.ActorUserID becomes the Payment's RecordedByUserID (T13.7, closes
	// issue #148). It used to be "" here, on the reasoning that nobody
	// "records" an online payment — but that left ConfirmOnlinePayment with no
	// ownership fact to check, so anyone holding the payment_id could capture
	// the intent. The value is safe to store precisely because
	// CreateOnlinePayment is in grpcapi.AuthenticatedMethods(): the actor
	// reaching here is a principal the auth interceptor verified, for every
	// payable type, not a claim off the wire.
	p, err := domain.NewPayment(s.ids.NewID(), in.PayableType, in.PayableID, in.Amount, domain.MethodOnline, in.ActorUserID)
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
//
// actorUserID is the verified caller attempting the capture (T13.7, closes
// issue #148). authorizeOnlineConfirmation runs before anything else — before
// the processor is touched, so a refused caller cannot move money and learns
// nothing about the Payment's state from the answer they get. It is a separate
// parameter rather than a field read off p because p carries the *stored*
// ownership fact and the actor must come from somewhere the caller does not
// control; the grpcapi handler loads p through Service.GetPayment and takes the
// actor from the request's verified principal, never from its body.
func (s *Service) ConfirmOnlinePayment(ctx context.Context, p domain.Payment, actorUserID string) (domain.Payment, error) {
	if err := authorizeOnlineConfirmation(p, actorUserID); err != nil {
		return domain.Payment{}, err
	}

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

	if err := s.reconcileRegistrationPaymentStatus(ctx, updated, socialplaydomain.PaymentStatusPaid); err != nil {
		return updated, err
	}
	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, updated, competitionsdomain.PaymentStatusPaid); err != nil {
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
// internal/payments has no live join to Booking's database (ADR-0015's
// still-open D1 blocks resolving a Booking's Host the same way this ticket
// now resolves a Game's/Competition's — see BookingHostID below) — so
// BookingHostID alone remains a caller-supplied fact. Every other
// ownership/assignment fact this check needs is resolved from the real
// Social Play/Competitions stores as of T16.2 (closes #168):
//
//   - BookingHostID is the user id of the Host who owns the Booking's
//     Game/Competition, required for a PayableTypeBooking payable. A
//     Booking with no Host at all (a direct court hire —
//     SourceIndividual/SourceRecurringHire) is explicitly out of scope for
//     offline recording in T6 (narrower-than-spec scope cut, see the PR
//     description): leave BookingHostID empty and RecordOfflinePayment
//     rejects with ErrNotPaymentRecorder rather than silently allowing it.
//     STILL CALLER-SUPPLIED, STILL FORGEABLE: unlike every other fact this
//     struct used to carry, BookingHostID is not resolved by this ticket —
//     see Service's doc comment on why, and issue #149, which stays open
//     for this one remaining fact.
//   - For a PayableTypeRegistration or PayableTypeNoShowFee payable, the
//     Registration's Game's Host id and its current Game-Admin set are now
//     RESOLVED, not accepted from the caller: authorizeGameRecording
//     (called from authorizeOfflineRecording) looks up PayableID's owning
//     GameID via port.RegistrationLookup, then that Game's HostID via
//     port.GameLookup and its admin set via port.GameAdminReader — the
//     glossary's own definition of Game Admin's scope
//     (agent-operating-handbook.md A2: "may record offline Payments...
//     scoped to the specific Game/Competition they are assigned to"). The
//     now-deleted GameHostID/AssignedGameAdminUserIDs fields that used to
//     live on this struct (T6.3/T14.4-era caller-supplied facts) are gone,
//     not merely unread — a future re-plumb of the wire list fails to
//     *compile* rather than silently restoring a forgeable check (T14.5/
//     T15.5's established pattern, per this ticket's own instructions).
//
// PayableTypeNoShowFee is authorized identically to PayableTypeRegistration
// (Host-or-assigned-Game-Admin) — deliberately not a third branch, since a
// no-show fee is, per the T6 kickoff note, "structurally just another
// payable action" against the same Registration/Game, and the required
// test (P1 #8) is that recording one takes zero new code paths.
//
//   - For a PayableTypeCompetitionEntry payable (T10.6, closes #96), the
//     CompetitionEntry's own CompetitionID and PlayerID are likewise now
//     RESOLVED: authorizeCompetitionEntryRecording looks up both in one
//     call via port.EntryLookup, then the Competition's admin set via
//     port.CompetitionAdminReader. The now-deleted EntrantPlayerID/
//     AssignedCompetitionAdminUserIDs fields are gone for the identical
//     compile-time-forcing reason GameHostID/AssignedGameAdminUserIDs are.
//     The authorized actor set keeps its one deliberate difference from the
//     Game branch: "the entrant OR an assigned Competition Admin", not "the
//     Host or an assigned Admin" — a competition entry's payment is most
//     often recorded by the entrant paying for their own place (the
//     issue's own framing: "as a Player entering a Competition, I want to
//     pay online"), so the resolved PlayerID is itself an authorized actor,
//     unlike GameHostID/BookingHostID which never name the paying Player.
type RecordOfflinePaymentInput struct {
	PayableType domain.PayableType
	PayableID   string
	Amount      domain.Money
	ActorUserID string

	BookingHostID string
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
	if err := s.authorizeOfflineRecording(ctx, in); err != nil {
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

	if err := s.reconcileRegistrationPaymentStatus(ctx, created, socialplaydomain.PaymentStatusPaid); err != nil {
		return created, err
	}
	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, created, competitionsdomain.PaymentStatusPaid); err != nil {
		return created, err
	}
	return created, nil
}

// RefundPaymentInput is the use-case input for refunding a Payment (T12.3,
// closing the gap open since T6.5).
//
// PaymentID identifies an existing Payment — unlike RecordOfflinePayment,
// which creates one, a refund always acts on a Payment that is already
// recorded, so this input carries no amount, method or payable type. The
// payable type is read from the *stored* Payment rather than accepted from
// the caller, deliberately: taking it from the request would let a caller
// pick which branch of authorizeOfflineRecording judges them (claiming
// `booking` on a registration Payment to be checked against a
// BookingHostID they supply themselves), which is the whole authorization
// check defeated by its own input.
//
// BookingHostID is the same caller-supplied, still-forgeable fact
// RecordOfflinePaymentInput's BookingHostID documents, for the same reason:
// internal/payments has no live join to Booking's database (ADR-0015's
// still-open D1). The same known-gap caveat applies unchanged — ActorUserID
// is a claimed identity, not a verified one, until T12.8 wires real auth.
// This proves an object-level check given a claimed actor.
//
// T16.2 (closes #168): the GameHostID/AssignedGameAdminUserIDs fields this
// struct used to carry are deleted, not merely unread — RefundPayment now
// reuses authorizeOfflineRecording unchanged (see that method's doc comment
// below), which resolves a registration Payment's Game/Host/admin set
// itself via the same port.RegistrationLookup/GameLookup/GameAdminReader
// RecordOfflinePayment uses, rather than trusting a caller-supplied claim.
// See RecordOfflinePaymentInput's doc comment for the full reasoning; a
// future re-plumb of the wire list fails to *compile* rather than silently
// restoring a forgeable check.
//
// PCI: no card-shaped field appears here, or on the proto message that
// feeds it (CLAUDE.md rule 11) — a refund needs an identifier and an actor,
// never card data. The processor reference used to actually move the money
// is the one already stored on the Payment.
type RefundPaymentInput struct {
	PaymentID   string
	ActorUserID string

	BookingHostID string
}

// RefundPayment refunds an already-paid Payment: it resolves the Payment,
// checks the actor may act on it, drives domain.Payment.Refund()'s existing
// state machine, refunds the processor side for an online Payment, persists
// the transition, and pushes `refunded` through to Social Play for a
// registration payable.
//
// Scope: `booking`, `registration`, and — as of T16.4, closing the
// corrected #125 — `competition_entry` payables. `no_show_fee`, which no
// version of this ticket's scope sentence has ever named, is still left out
// rather than silently included (issue #130, which unlike #125 carries a
// genuinely open product question — see RefundPayment's own instruction 3
// history in docs/process/t16-sprint-plan.md). It is rejected with the
// existing ErrInvalidPayableType rather than a new sentinel invented for a
// scope boundary.
//
// `competition_entry` was out of scope from T12.3 through T16.3: Payments
// had no live join to Competitions' Competition-Admin/entrant facts to
// authorize a refund against (see authorizeCompetitionEntryRecording's own
// history). T16.2 built and wired that resolution for the *recording* path;
// T16.4 is the ticket that widens the gate below to admit the same payable
// type for the symmetric *refund* path, reusing authorizeOfflineRecording's
// existing PayableTypeCompetitionEntry branch unchanged rather than adding a
// second authorization rule — exactly the precedent T12.3 set for
// `registration` reusing the same method for its own refund path.
//
// Ordering, and why it differs from RecordOfflinePayment's:
//
//  1. Resolve the Payment first. Authorization here needs the Payment's own
//     payable type to pick a branch, and that fact only exists in the
//     store — so unlike RecordOfflinePayment (where every authorization
//     input arrives on the request and the check can run before anything
//     else), the lookup genuinely has to come first. GetPayment applies the
//     uuidShape boundary guard, so a malformed id is answered exactly like
//     an unknown one and never reaches the Postgres adapter's mustUUID.
//  2. Scope gate, before authorization. An out-of-scope payable type is a
//     static contract fact, not a per-object secret, and this package's own
//     gRPC adapter already rejects an unrecognised payable type with the
//     same InvalidArgument-shaped answer before app.Service (and therefore
//     before any authorization) is reached — see fromProtoPayableType.
//  3. Authorization, reusing authorizeOfflineRecording unchanged (T12.3
//     instruction 1): the question "who may act on this Payment" was
//     already answered in T6.3, and a refund is the same Host/Game-Admin
//     action as recording one. A second authorization concept for refunds
//     is exactly what this ticket forbids.
//  4. The domain transition, evaluated on a copy *before* the processor
//     call. Two things fall out of that ordering, both deliberate: an
//     already-refunded or never-paid Payment is rejected by
//     domain.Payment.Refund() itself (T12.3 instruction 2 — no duplicate
//     app-level status check) and never reaches the processor, so the
//     platform can't ask Stripe to refund the same charge twice; and
//     because the transition is applied to a copy, a processor failure
//     leaves both the returned Payment and the stored one untouched.
//  5. Persist, then project. A persist failure is returned to the caller
//     rather than swallowed (T12.3 instruction 7): the money really has
//     moved at the processor by that point, so silently reporting success
//     would leave the record permanently disagreeing with the processor
//     with nobody aware of it. The projection step pushes `refunded`
//     through whichever of RegistrationPaymentUpdater/CompetitionEntryUpdater
//     owns this Payment's payable type (T16.4 adds the second one) — each
//     helper is a no-op for the payable type it doesn't own, so exactly one
//     ever does real work for a given Payment, mirroring
//     ConfirmOnlinePayment/RecordOfflinePayment calling both unconditionally
//     on the `paid` direction.
func (s *Service) RefundPayment(ctx context.Context, in RefundPaymentInput) (domain.Payment, error) {
	p, err := s.GetPayment(ctx, in.PaymentID)
	if err != nil {
		return domain.Payment{}, err
	}

	if p.PayableType != domain.PayableTypeBooking &&
		p.PayableType != domain.PayableTypeRegistration &&
		p.PayableType != domain.PayableTypeCompetitionEntry {
		return domain.Payment{}, domain.ErrInvalidPayableType
	}

	// PayableID is now load-bearing for authorization (T16.2): the
	// PayableType==Registration branch resolves the Game/Host/admin set
	// from it via RegistrationLookup/GameLookup/GameAdminReader, so — unlike
	// before this ticket, when the check only ever compared caller-supplied
	// fields and never touched PayableID at all — it must be threaded
	// through from the stored Payment, the same one GetPayment above just
	// loaded, not the request (RefundPaymentInput carries no PayableID of
	// its own; see that struct's doc comment for why the payable type and,
	// as of this ticket, the payable id itself are both read from the store
	// rather than accepted from the caller).
	if err := s.authorizeOfflineRecording(ctx, RecordOfflinePaymentInput{
		PayableType:   p.PayableType,
		PayableID:     p.PayableID,
		ActorUserID:   in.ActorUserID,
		BookingHostID: in.BookingHostID,
	}); err != nil {
		return domain.Payment{}, err
	}

	// domain.Payment is a plain value type, so this is a real copy: Refund's
	// pointer receiver mutates `refunded` alone, leaving p — the value
	// returned on every failure path below — exactly as it was loaded.
	refunded := p
	if err := refunded.Refund(); err != nil {
		return p, err
	}

	// Offline Payments have no processor side to refund: the money moved as
	// cash, and recording the refund *is* the refund — the mirror image of
	// RecordOfflinePayment having no separate confirmation step.
	if p.Method == domain.MethodOnline {
		if err := s.processor.RefundPayment(ctx, p.StripeReference); err != nil {
			return p, err
		}
	}

	updated, err := s.payments.Update(ctx, refunded)
	if err != nil {
		return domain.Payment{}, err
	}

	if err := s.reconcileRegistrationPaymentStatus(ctx, updated, socialplaydomain.PaymentStatusRefunded); err != nil {
		return updated, err
	}
	// T16.4 (closes the corrected #125): the CompetitionEntry mirror of the
	// registration push immediately above, through the same
	// CompetitionEntryUpdater port ConfirmOnlinePayment/RecordOfflinePayment
	// already push `paid` through (T10.6) — not a new updater or a second
	// pattern, per this ticket's own instruction 2.
	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, updated, competitionsdomain.PaymentStatusRefunded); err != nil {
		return updated, err
	}
	return updated, nil
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
// status is a parameter rather than a hardcoded PaymentStatusPaid as of
// T12.3: RefundPayment pushes socialplaydomain.PaymentStatusRefunded
// through this same helper, the same port, and the same call site shape.
// Generalizing the existing reconciliation is deliberate — the ticket's
// instruction 3 is explicit that `refunded` must go out through the
// existing RegistrationPaymentUpdater rather than a second updater path,
// and a second near-identical helper differing only in a constant would
// have been exactly that.
func (s *Service) reconcileRegistrationPaymentStatus(ctx context.Context, p domain.Payment, status socialplaydomain.PaymentStatus) error {
	if p.PayableType != domain.PayableTypeRegistration {
		return nil
	}
	if s.registrationUpdater == nil {
		return nil
	}
	return s.registrationUpdater.UpdatePaymentStatus(ctx, p.PayableID, status)
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
// status is a parameter rather than a hardcoded PaymentStatusPaid as of
// T16.4: RefundPayment pushes competitionsdomain.PaymentStatusRefunded
// through this same helper, the same port, and the same call site shape —
// mirroring exactly how T12.3 generalized reconcileRegistrationPaymentStatus
// above for the identical reason. See that method's own doc comment for why
// a second, near-identical helper differing only in a constant would have
// been the wrong shape.
func (s *Service) reconcileCompetitionEntryPaymentStatus(ctx context.Context, p domain.Payment, status competitionsdomain.PaymentStatus) error {
	if p.PayableType != domain.PayableTypeCompetitionEntry {
		return nil
	}
	if s.competitionEntryUpdater == nil {
		return nil
	}
	return s.competitionEntryUpdater.UpdatePaymentStatus(ctx, p.PayableID, status)
}

// authorizeOfflineRecording is the actor-scoped (BOLA-shaped) check T6.3
// requires, mirroring socialplay.domain.Registration.Cancel's ownership
// check but living in the app layer rather than the domain: unlike
// Registration.Cancel, this check needs facts (Host id, Game/Competition
// Admin assignments) that come from outside the Payment aggregate itself,
// so it can't be expressed as a method on domain.Payment the way Cancel is
// a method on Registration.
//
// T16.2 (closes #168): for PayableTypeRegistration/PayableTypeNoShowFee and
// PayableTypeCompetitionEntry, this now RESOLVES those facts from the real
// Social Play/Competitions stores (authorizeGameRecording/
// authorizeCompetitionEntryRecording below) instead of comparing against
// caller-supplied AssignedGameAdminUserIDs/AssignedCompetitionAdminUserIDs
// fields — see RecordOfflinePaymentInput's doc comment for the full history
// of why T15.5 built port.GameAdminReader/CompetitionAdminReader but could
// not wire them in, and what changed. PayableTypeBooking is UNCHANGED: it
// still compares against caller-supplied BookingHostID, because Payments has
// no live join to Booking's database — ADR-0015's still-open D1 blocks the
// identical resolution this ticket just built for Social Play/Competitions.
// See ctx's use below: this method now performs I/O (the two Booking-adjacent
// helpers below do not need ctx passed further than the resolver calls they
// make), which is why it takes one — every other branch of this codebase's
// authorization checks is still a pure function of its input, and this is
// the one exception, load-bearing rather than incidental.
//
//   - PayableTypeBooking: legal only when ActorUserID matches
//     BookingHostID, and BookingHostID must be non-empty (a Host-less
//     Booking is out of scope for T6.3, see RecordOfflinePaymentInput's
//     doc comment).
//   - PayableTypeRegistration/PayableTypeNoShowFee: delegated to
//     authorizeGameRecording.
//   - PayableTypeCompetitionEntry: delegated to
//     authorizeCompetitionEntryRecording.
//
// A mismatched or missing actor always returns ErrNotPaymentRecorder, the
// same sentinel regardless of which branch rejected it — a caller does not
// get to distinguish "wrong payable type" from "wrong actor" from this
// error alone, matching ErrNotRegistrationOwner's equally flat shape. A
// resolver I/O failure (as opposed to a resolved-but-mismatched actor) is
// returned as-is, not folded into ErrNotPaymentRecorder — see
// authorizeGameRecording/authorizeCompetitionEntryRecording's own doc
// comments for why that distinction matters.
func (s *Service) authorizeOfflineRecording(ctx context.Context, in RecordOfflinePaymentInput) error {
	if in.ActorUserID == "" {
		return domain.ErrNotPaymentRecorder
	}

	if in.PayableType == domain.PayableTypeBooking {
		if in.BookingHostID == "" || in.ActorUserID != in.BookingHostID {
			return domain.ErrNotPaymentRecorder
		}
		return nil
	}

	if in.PayableType == domain.PayableTypeCompetitionEntry {
		return s.authorizeCompetitionEntryRecording(ctx, in)
	}

	return s.authorizeGameRecording(ctx, in)
}

// authorizeGameRecording is authorizeOfflineRecording's
// PayableTypeRegistration/PayableTypeNoShowFee branch (T16.2, closes #168):
// legal when ActorUserID matches the Registration's Game's HostID, or
// appears in that Game's current admin set — resolved from the real Social
// Play store via RegistrationLookup -> GameLookup/GameAdminReader, rather
// than compared against a caller-supplied claim.
//
// Fails CLOSED, not open, when a resolver dependency is missing (nil): see
// Service's own doc comment for why these three ports are not optional the
// way RegistrationUpdater is. A resolver returning ("", nil)/(nil, nil) for
// an unresolvable Registration/Game (RegistrationLookup/GameLookup/
// GameAdminReader's own documented convention for "no such id") flows
// through to the same ErrNotPaymentRecorder a real mismatch produces,
// without a distinct code path — an id that doesn't resolve is exactly as
// unauthorized as one that resolves to someone else's Game, and the two
// must not be distinguishable from the response alone (mirrors
// authorizeOnlineConfirmation's "wrong payable type" vs "wrong actor"
// non-distinction, one level up the same principle).
//
// A genuine I/O error from either resolver (not a not-found) is returned to
// the caller as-is rather than folded into ErrNotPaymentRecorder: the
// caller may be perfectly legitimate and the check simply couldn't
// complete, which is a different, retriable condition from "you are not
// authorized" and toStatus's default Internal mapping is the correct answer
// for it, not a misleading PermissionDenied.
func (s *Service) authorizeGameRecording(ctx context.Context, in RecordOfflinePaymentInput) error {
	if s.registrationLookup == nil || s.gameLookup == nil || s.gameAdminReader == nil {
		return domain.ErrNotPaymentRecorder
	}

	gameID, err := s.registrationLookup.GameIDForRegistration(ctx, in.PayableID)
	if err != nil {
		return err
	}

	hostID, err := s.gameLookup.HostIDForGame(ctx, gameID)
	if err != nil {
		return err
	}
	if hostID != "" && in.ActorUserID == hostID {
		return nil
	}

	admins, err := s.gameAdminReader.ListGameAdmins(ctx, gameID)
	if err != nil {
		return err
	}
	for _, adminID := range admins {
		if adminID != "" && adminID == in.ActorUserID {
			return nil
		}
	}
	return domain.ErrNotPaymentRecorder
}

// authorizeCompetitionEntryRecording is authorizeOfflineRecording's
// PayableTypeCompetitionEntry branch (T16.2, closes #96/#168): legal when
// ActorUserID matches the CompetitionEntry's own PlayerID (the entrant), or
// appears in the owning Competition's current admin set — both resolved
// from the real Competitions store via EntryLookup/CompetitionAdminReader,
// rather than compared against a caller-supplied claim. Mirrors
// authorizeGameRecording exactly, with the one deliberate difference
// RecordOfflinePaymentInput's doc comment records: the resolved PlayerID
// (the entrant) is itself an authorized actor here, unlike
// authorizeGameRecording's HostID/admin-only set.
//
// Fails CLOSED when a resolver dependency is missing, and treats an
// unresolvable CompetitionEntry/Competition the same "safely not
// authorized, not a distinct error" way authorizeGameRecording does — see
// that method's doc comment for the full reasoning, which applies here
// unchanged.
func (s *Service) authorizeCompetitionEntryRecording(ctx context.Context, in RecordOfflinePaymentInput) error {
	if s.entryLookup == nil || s.competitionAdminReader == nil {
		return domain.ErrNotPaymentRecorder
	}

	competitionID, playerID, err := s.entryLookup.CompetitionIDAndPlayerIDForEntry(ctx, in.PayableID)
	if err != nil {
		return err
	}
	if playerID != "" && in.ActorUserID == playerID {
		return nil
	}

	admins, err := s.competitionAdminReader.ListCompetitionAdmins(ctx, competitionID)
	if err != nil {
		return err
	}
	for _, adminID := range admins {
		if adminID != "" && adminID == in.ActorUserID {
			return nil
		}
	}
	return domain.ErrNotPaymentRecorder
}

// authorizeOnlineConfirmation is ConfirmOnlinePayment's object-level check
// (T13.7, closes issue #148): only the actor a Payment records as its own may
// capture it.
//
// This is the one authorization check in this package that consults no
// caller-supplied fact whatsoever, and — as of T16.2 — it has company:
// authorizeGameRecording/authorizeCompetitionEntryRecording (called from
// authorizeOfflineRecording) now resolve their facts from the real Social
// Play/Competitions stores instead of accepting them on the request.
// authorizeOfflineRecording's PayableTypeBooking branch, and
// authorizeOnlineCreation below, are the two REMAINING exceptions: Payments
// still has no read path into Booking's database (BookingHostID — issue
// #149, blocked on ADR-0015's still-open D1) or a wired resolution for
// CreateOnlinePayment's competition_entry branch specifically (EntrantPlayerID/
// AssignedCompetitionAdminUserIDs — issue #198, opened by this ticket's own
// sibling sweep; mechanically answerable by the identical
// EntryLookup/CompetitionAdminReader ports this ticket built, just not wired
// into this RPC). This check needs none of that: p.RecordedByUserID is a fact
// this context recorded itself, from a verified principal, at
// CreateOnlinePayment time. That is exactly why #148 was closable inside the
// Payments context and #149/#198 are not, by this ticket.
//
// Both sides of the comparison are subjects — actorUserID is whatever
// grpcapi's actor(ctx) returned, and p.RecordedByUserID is whatever an earlier
// actor(ctx) wrote — so they are compared unchanged, with no resolution step,
// per ADR-0014 §5a's explicit ruling for this ticket.
//
// An empty p.RecordedByUserID means nobody may confirm, never that anybody may:
//
//   - Every online Payment written before T13.7 has one, and grandfathering
//     them would leave issue #148's "anyone holding a payment_id can capture
//     the intent" true for exactly the rows it was reported against. The
//     alternative the issue itself names — "reject confirmation on ownerless
//     legacy rows, or grandfather them" — is answered here, deliberately, in
//     the safe direction. Recovering such a row means recording it through the
//     offline path or re-creating the intent, not weakening this check.
//   - An empty actorUserID can never satisfy it either, so a caller who
//     somehow reaches this code with no principal is refused rather than
//     matched against an ownerless Payment. (The handler rejects that caller
//     earlier, with Unauthenticated; this is the belt to that braces.)
func authorizeOnlineConfirmation(p domain.Payment, actorUserID string) error {
	if actorUserID == "" || p.RecordedByUserID == "" || actorUserID != p.RecordedByUserID {
		return domain.ErrNotPaymentOwner
	}
	return nil
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
//
// T15.5 sibling-sweep finding (§A15), RE-VERIFIED AND STILL TRUE by T16.2's
// own required sibling sweep (instruction 10) — the T16 sprint plan's own
// inspection missed it, so this was checked against the code directly rather
// than trusted: EntrantPlayerID/AssignedCompetitionAdminUserIDs below are the
// same caller-supplied-entitlement pattern authorizeOfflineRecording's
// PayableTypeCompetitionEntry branch had before T16.2 resolved it via
// port.EntryLookup/CompetitionAdminReader. Those exact two ports would answer
// this branch identically (CreateOnlinePaymentInput.PayableID is the same
// CompetitionEntry id RecordOfflinePaymentInput.PayableID is) — this is NOT
// blocked on a missing read the way #149's BookingHostID is. Not fixed here
// because T16.2's own instructions scope the wiring to authorizeOfflineRecording
// only; tracked as its own issue, #198, rather than folded into this ticket
// silently.
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
