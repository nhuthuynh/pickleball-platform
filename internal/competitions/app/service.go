package app

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// shareTokenShape matches exactly what adapter/sharetoken.Generator ever
// produces: base64.RawURLEncoding output, i.e. the URL-safe alphabet with no
// padding. Anything outside this shape (a NUL byte, invalid UTF-8, stray
// punctuation) cannot possibly be a real token, so GetCompetitionByShareToken
// rejects it before it ever reaches Postgres — not as a distinguishable
// InvalidArgument (that would itself be an oracle: "your guess had the wrong
// shape" is still a signal to an enumerator), but by folding it into the
// exact same ErrCompetitionNotFound every other miss already returns. This
// closes a real gap a review found: a NUL byte is valid UTF-8 and so passed
// this method unmodified before, reached a `text` column, and came back as a
// raw Postgres encoding error at 500 — both a leak of internal error detail
// on a public unauthenticated endpoint and a way to distinguish
// "malformed" from "well-formed but unknown", which this method's own doc
// comment promises never happens.
var shareTokenShape = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// uuidShape matches the canonical 8-4-4-4-12 hex form that
// internal/platform/idgen mints for every Competition and CompetitionEntry ID.
//
// It exists for the same reason shareTokenShape does, one identifier over: a
// caller-supplied ID arrives as an unvalidated HTTP path parameter, and the
// Postgres adapter's mustUUID *panics* on anything pgtype.UUID.Scan can't
// parse. Since grpc installs no recover() of its own, that panic used to take
// the entire server process down — `GET /v1/competitions/not-a-uuid` was an
// unauthenticated total outage. Rejecting the ID here means the panic never
// happens; internal/platform/grpcrecovery is the backstop for the ones nobody
// anticipated, not a substitute for this.
//
// **Deliberately narrower than github.com/google/uuid's Validate, and it must
// stay that way.** uuid.Validate also accepts the braced form
// `{6ba7b810-...}` and the `urn:uuid:6ba7b810-...` form; pgtype.UUID.Scan
// accepts neither, so a guard written with uuid.Validate would have passed both
// straight through to mustUUID and panicked anyway. A validator that is wider
// than the thing it protects is not a validator. The 32-char undashed form that
// pgtype also accepts is likewise rejected — being narrower than the adapter is
// always safe, and idgen never produces it.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Competitions context's application layer: it orchestrates
// the domain and its ports but holds no business rules itself — those live
// in internal/competitions/domain. Mirrors internal/socialplay/app.Service's
// shape, one level deeper because a Competition reserves courts across
// sessions × courts rather than a single time range.
//
// Every dependency is stored on the Service and supplied through
// ServiceOptions. That differs from Social Play, which passes
// CourtReservation and FacilityLookup per-call to ScheduleGame; there is no
// reason for a caller to vary them per request, and a use case whose
// dependency list is visible on the type is easier to wire and to reason
// about than one whose dependencies are split between the constructor and
// individual method signatures.
type Service struct {
	competitions port.Repository
	ids          port.IDGenerator
	reservation  port.CourtReservation
	facilities   port.FacilityLookup
	shareTokens  port.ShareTokenGenerator
}

// ServiceOptions is the dependency bundle for NewService.
//
// An options struct from day one, not positional arguments. HANDOFF.md's
// Cross-cutting section records that socialplayapp.NewService reached four
// positional args in T6.6 and that the next dependency added there should
// switch it to an options struct rather than growing further; this Service
// starts at five, which is past that line before a single call site exists.
// The shape follows paymentsapp.ServiceOptions, which made the same choice
// for the same reason in T6.4.
//
// Unlike Payments' RegistrationUpdater, none of these is optional: every
// field is required by at least one use case, and a Service missing one
// would panic on a nil interface call rather than degrading gracefully.
// Tests supply in-memory fakes for all five (see service_test.go);
// cmd/server supplies the real Postgres repository, internal/platform/idgen,
// and the two adapter packages.
type ServiceOptions struct {
	Competitions port.Repository
	IDs          port.IDGenerator
	Reservation  port.CourtReservation
	Facilities   port.FacilityLookup
	ShareTokens  port.ShareTokenGenerator
}

// NewService constructs a Service from opts.
func NewService(opts ServiceOptions) *Service {
	return &Service{
		competitions: opts.Competitions,
		ids:          opts.IDs,
		reservation:  opts.Reservation,
		facilities:   opts.Facilities,
		shareTokens:  opts.ShareTokens,
	}
}

// ScheduleCompetitionInput is the use-case input for scheduling a
// Competition. Every field is passed straight through to
// domain.NewCompetition — see that constructor's doc comment for the
// validation each is subject to.
type ScheduleCompetitionInput struct {
	HostID string
	Name   string
	// VenueFacilityID is a real Facilities-context Facility ID. Optional:
	// an empty value skips the port.FacilityLookup existence check entirely
	// (see ScheduleCompetition's doc comment and
	// domain.Competition.VenueFacilityID).
	VenueFacilityID string
	// Sessions are the Competition's sittings, each a time range plus the
	// courts it reserves for that range. ScheduleCompetition reserves one
	// Booking per (session, court) pair.
	Sessions       []domain.Session
	Capacity       int
	GuestAllowance int
	PaymentMethod  domain.PaymentMethod
	// EntryFee's zero value (domain.Money{}) is valid and means a free
	// competition — a real value, not a stand-in for "unspecified", so
	// unlike PaymentMethod it needs no resolution step in the adapter.
	EntryFee domain.Money
	Format   domain.Format
}

// ScheduleCompetition builds a Competition (domain.NewCompetition, T9.1)
// and reserves one Booking per (session, court) pair via the
// CourtReservation port — this is what makes Competitions inherit the
// Booking context's no-double-booking invariant rather than merely
// modelling a Competition that has no effect on court availability. The
// `competition` Booking source and its cross-source conflict coverage
// already exist in the Booking context, so nothing under internal/booking
// changes to make this work.
//
// The steps run in this order, and the order is load-bearing:
//
//  1. domain.NewCompetition validates everything that needs no
//     infrastructure — capacity, sessions, ranges, the closed enums, and
//     crucially that the Competition's own sessions don't double-book one
//     of its courts (domain.ErrOverlappingSessions). Doing this first means
//     structurally invalid input costs no I/O at all, and that a
//     Competition which conflicts with *itself* is caught up front rather
//     than discovered part-way through the reservation loop and rolled
//     back. This mirrors socialplayapp.ScheduleGame, which likewise
//     constructs before looking anything up.
//  2. A non-empty VenueFacilityID is validated against the real Facilities
//     context via port.FacilityLookup, returning
//     domain.ErrFacilityNotFound (which T9.4 maps to a 404-shaped status).
//     This happens **before any court is reserved**, so a bogus venue can
//     never leave a dangling Booking behind — the requirement T8.3
//     established for Games, which matters more here because a Competition
//     may hold many courts across many dates. An empty VenueFacilityID
//     skips the lookup entirely: the venue is optional by design.
//  3. Every (session, court) pair is reserved, in order.
//  4. The Competition is persisted.
//
// Multi-reservation atomicity: a single ScheduleCompetition can require
// many independent ReserveCourt calls, and there is no transaction spanning
// them. If any one fails — or if persistence fails after all of them
// succeeded — the reservations already made are compensated by
// ReleaseCourt calls in reverse order, so no half-scheduled Competition
// leaves courts held for an event that doesn't exist.
//
// Rollback is best-effort and never masks the original error: if a
// ReleaseCourt call itself fails, ScheduleCompetition still returns the
// reservation conflict that triggered the rollback, because that conflict
// is the actionable information for the caller ("court 3 is taken on the
// 2nd") whereas a rollback failure is an operational problem for the
// platform. The cost is that a failed release can leave a dangling Booking;
// see port.CourtReservation.ReleaseCourt's doc comment, which records that
// as a known gap.
//
// The share token is generated before the Competition is constructed
// (port.ShareTokenGenerator), so every Competition carries one from the
// moment it exists and T9.5's share-link flow needs no second write path
// to back-fill it. A generator failure aborts before any port is touched.
func (s *Service) ScheduleCompetition(ctx context.Context, in ScheduleCompetitionInput) (domain.Competition, error) {
	shareToken, err := s.shareTokens.NewShareToken()
	if err != nil {
		return domain.Competition{}, fmt.Errorf("competitions: generating share token: %w", err)
	}

	competition, err := domain.NewCompetition(
		s.ids.NewID(),
		in.HostID,
		in.Name,
		in.VenueFacilityID,
		in.Sessions,
		in.Capacity,
		in.GuestAllowance,
		in.PaymentMethod,
		in.EntryFee,
		in.Format,
		shareToken,
	)
	if err != nil {
		return domain.Competition{}, err
	}

	// Every court id in every session is shape-checked here — the whole
	// input, before the facility lookup and before the nested reservation
	// loop below (T14.8, closing issue #156). See the twin guard in
	// internal/socialplay/app.ScheduleGame for the full reasoning; the one
	// difference is that this input is a collection of collections, so a
	// malformed entry can hide in a later session behind an earlier one that
	// would have reserved perfectly well. Validating all of them up front is
	// what keeps that case from leaving reserved courts and a rejected
	// Competition behind.
	for _, sess := range competition.Sessions {
		for _, courtID := range sess.CourtIDs {
			if !uuidShape.MatchString(courtID) {
				return domain.Competition{}, domain.ErrMalformedCourtID
			}
		}
	}

	if competition.VenueFacilityID != "" {
		if err := s.facilities.FacilityExists(ctx, competition.VenueFacilityID); err != nil {
			return domain.Competition{}, fmt.Errorf("competitions: validating venue facility %s for competition %s: %w", competition.VenueFacilityID, competition.ID, err)
		}
	}

	var reservedBookingIDs []string
	for _, sess := range competition.Sessions {
		for _, courtID := range sess.CourtIDs {
			bookingID, err := s.reservation.ReserveCourt(ctx, courtID, sess.Range.Start, sess.Range.End, competition.ID)
			if err != nil {
				s.releaseAll(ctx, reservedBookingIDs)
				return domain.Competition{}, fmt.Errorf("competitions: reserving court %s from %s for competition %s: %w", courtID, sess.Range.Start.Format("2006-01-02T15:04:05Z07:00"), competition.ID, err)
			}
			reservedBookingIDs = append(reservedBookingIDs, bookingID)
		}
	}

	persisted, err := s.competitions.Create(ctx, competition)
	if err != nil {
		s.releaseAll(ctx, reservedBookingIDs)
		return domain.Competition{}, fmt.Errorf("competitions: persisting competition %s: %w", competition.ID, err)
	}

	return persisted, nil
}

// releaseAll is the shared best-effort rollback helper for
// ScheduleCompetition's two failure points (a later (session, court) pair
// conflicting, or persistence failing after every court reserved).
//
// Releases run in reverse order — most recent reservation first. Nothing in
// the Booking context requires that ordering, since each release is
// independent; it is chosen because unwinding in the reverse of the order
// things were done is the convention that makes a partially-failed rollback
// easiest to reason about in a log (the trailing entries are the ones that
// survived), and because it costs nothing to do it the predictable way.
//
// Errors are deliberately discarded rather than collected: the caller
// returns the original failure regardless (see ScheduleCompetition's doc
// comment), so there is no decision a returned rollback error could inform
// here. The real adapter is the right place to log the residue.
func (s *Service) releaseAll(ctx context.Context, bookingIDs []string) {
	for i := len(bookingIDs) - 1; i >= 0; i-- {
		_ = s.reservation.ReleaseCourt(ctx, bookingIDs[i])
	}
}

// EnterCompetitionInput is the use-case input for entering a Competition.
type EnterCompetitionInput struct {
	CompetitionID string
	PlayerID      string
	// GuestCount is how many guests this entrant brings; each occupies a
	// place exactly as the entrant does. Validated against the
	// Competition's GuestAllowance and its weighted capacity by
	// domain.Enter. Zero means no guests.
	GuestCount int
	// Source records how the entrant arrived — in-app or via a shared link
	// (T9.5). Validated against domain.EntrySource's closed enum by
	// domain.Enter: the server checks the caller's claimed source rather
	// than inferring it.
	Source domain.EntrySource
}

// EnterCompetition loads the Competition and its current active entries,
// applies domain.Enter's rules (cancelled-competition rejection, guest
// allowance, double entry, and the **weighted** capacity check where an
// entry and each of its guests occupy one place), assigns the new entry an
// ID, and persists it.
//
// The weighted capacity check here is a fail-fast pre-check, not the
// authoritative guard: T9.4's DB-level trigger — which locks the owning
// competitions row and sums 1 + guest_count across non-cancelled entries —
// is what actually prevents two simultaneous entries from overfilling a
// Competition. This is the same relationship domain.EnsureNoConflict has
// with Booking's EXCLUDE constraint (CLAUDE.md rule 4): the domain
// expresses the rule and gives callers a clear error, the database enforces
// it.
//
// **No ownership check, deliberately.** Any Player may enter a published
// Competition; entering is not an act *on* the Competition, it is the
// invitation being accepted, exactly as socialplayapp.RegisterForGame has
// no ownership check. Ownership gates only acts on the Competition itself —
// see CancelCompetition, which calls domain.Competition.EnsureHost. That
// asymmetry is intentional rather than a missed check: a Competition an
// entrant may not enter is one that is cancelled or full, and both of those
// are domain rules domain.Enter already enforces.
func (s *Service) EnterCompetition(ctx context.Context, in EnterCompetitionInput) (domain.CompetitionEntry, error) {
	// A malformed CompetitionID is answered exactly like an unknown one
	// (T10.7, closing issue #97): this method already calls GetByID first
	// and returns the bare domain.ErrCompetitionNotFound for a miss, so a
	// malformed id must produce the same sentinel rather than reaching the
	// adapter's mustUUID, which panics on non-UUID input — the same guard
	// GetCompetition already applies to the same field, applied here because
	// this is a write path PR #89's original Layer 2 pass didn't cover.
	if !uuidShape.MatchString(in.CompetitionID) {
		return domain.CompetitionEntry{}, domain.ErrCompetitionNotFound
	}

	competition, err := s.competitions.GetByID(ctx, in.CompetitionID)
	if err != nil {
		return domain.CompetitionEntry{}, err
	}

	existing, err := s.competitions.ListActiveEntriesForCompetition(ctx, in.CompetitionID)
	if err != nil {
		return domain.CompetitionEntry{}, fmt.Errorf("competitions: listing entries for competition %s: %w", in.CompetitionID, err)
	}

	entry, err := domain.Enter(competition, existing, in.PlayerID, in.GuestCount, in.Source)
	if err != nil {
		return domain.CompetitionEntry{}, err
	}

	// domain.Enter leaves the ID empty on purpose — assigning a durable
	// identifier is this layer's job, not a pure domain constructor's.
	entry.ID = s.ids.NewID()

	return s.competitions.CreateEntry(ctx, entry)
}

// GetCompetition returns a single Competition by ID, or
// domain.ErrCompetitionNotFound (which T9.4's handler maps to a 404-shaped
// status).
//
// A deliberate thin pass-through to the repository with no orchestration of
// its own: there is no rule to apply on reading a Competition, and adding a
// use-case method that only forwards is still worth it so the gRPC handler
// depends on app.Service alone rather than reaching past it into
// port.Repository — the dependency direction CLAUDE.md rule 3 requires, and
// the same shape socialplay's read paths use.
//
// **No ownership check, deliberately**, for the same reason
// EnterCompetition has none: reading a Competition is not an act *on* it.
// Anyone who can see the listing can read the Competition it names.
func (s *Service) GetCompetition(ctx context.Context, competitionID string) (domain.Competition, error) {
	// A malformed ID is answered exactly like an unknown one, for the reason
	// spelled out on uuidShape and on GetCompetitionByShareToken below: this is
	// a public read, so "that isn't even a valid ID" would be a distinguishable
	// reply and therefore an oracle. It is also what stops the adapter's
	// mustUUID from panicking the process on an unauthenticated request.
	if !uuidShape.MatchString(competitionID) {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	return s.competitions.GetByID(ctx, competitionID)
}

// GetCompetitionByShareToken resolves a Competition from the token behind
// its shareable registration link (T9.5) — the public, unauthenticated read
// a Player hits after following a link the Host posted outside the app.
//
// It returns the SAME projection GetCompetition returns: the same
// domain.Competition, which the gRPC adapter renders through the same
// toProtoCompetition into the same wire message. Nothing extra is disclosed
// because a caller arrived by link rather than by ID — a share link must
// never be a wider window onto a Competition than the app itself already
// gives (the inverse of the BA dossier §5 scope-narrowing concern, and the
// likelier bug of the two).
//
// **The error is deliberately NOT wrapped**, unlike almost every other
// failure path in this Service. An unknown token, a malformed token, and a
// token for a Competition that doesn't exist must all produce one
// byte-identical domain.ErrCompetitionNotFound, because this endpoint is
// unauthenticated and keyed by a secret: any difference in the answer is an
// oracle that tells an enumerator "that guess had the right shape" or "that
// token exists but the Competition is gone". A fmt.Errorf that helpfully
// named the token or the reason would reintroduce exactly that signal (and,
// worse, write the token itself into every log line), so the plain sentinel
// is returned and the empty token short-circuits into the same sentinel
// rather than an InvalidArgument that would single itself out.
//
// There is deliberately no ownership or authentication check, for the same
// reason GetCompetition has none: the token IS the capability. Possession of
// the link is what grants the read, which is why the token's entropy carries
// the whole weight of the control and why generating it was settled before
// this read path existed (internal/competitions/adapter/sharetoken, T9.4 —
// crypto/rand, 256 bits; T9.5 neither rebuilds nor second-guesses it).
//
// A CANCELLED Competition resolves here exactly as a scheduled one does, and
// is returned with Status cancelled — no status filter, deliberately unlike
// ListCompetitions. See port.Repository.GetByShareToken for why, and
// domain.Enter for the separate rejection that still blocks entering it.
//
// KNOWN GAP — REVOCATION, named rather than silently omitted: there is no
// rotate-token or revoke-token path in T9. A Host who posts a link to a
// public channel cannot un-publish it; the token stays valid for the
// Competition's lifetime. This is not an oversight and not a "we'll harden
// it later" placeholder — it is blocked on a specific prerequisite. Rotation
// is an authenticated act on a Competition ("only the Host may rotate this
// Competition's token"), and this codebase has no authenticated identity
// yet: actor_user_id is a caller-supplied claim (HANDOFF.md's open
// cross-cutting Auth item). Shipping rotation now would gate it on a claim
// anyone can make, meaning any caller could invalidate any Host's published
// link — strictly worse than no rotation at all. TRIGGER: build it alongside
// real auth, as an authenticated CompetitionsService method that reuses
// domain.Competition.EnsureHost, plus an UPDATE of the share_token column.
func (s *Service) GetCompetitionByShareToken(ctx context.Context, shareToken string) (domain.Competition, error) {
	if shareToken == "" || !shareTokenShape.MatchString(shareToken) {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	return s.competitions.GetByShareToken(ctx, shareToken)
}

// ListCompetitions is the browse/list read path (T9.4), returning each
// matching Competition with its server-computed, **weighted** SpotsLeft —
// see port.CompetitionListing for why the weighting is load-bearing rather
// than cosmetic, and port.CompetitionListingFilter for what each filter
// field means when unset.
func (s *Service) ListCompetitions(ctx context.Context, filter port.CompetitionListingFilter) ([]port.CompetitionListing, error) {
	return s.competitions.ListCompetitions(ctx, filter)
}

// ListEntriesForCompetition returns a Competition's full roster, cancelled
// entries included — see port.Repository.ListEntriesForCompetition for why
// this is a different method from the capacity check's own
// ListActiveEntriesForCompetition rather than the same one with a flag.
//
// # T13.6: Host-only (partial fix for #147)
//
// This method shipped with no authorization check at all — anyone holding a
// competition_id read every entrant's player_id and payment status, and
// competition_id is readable off the *public* ListCompetitions and
// GetCompetition responses, which made the leak enumerable (#147).
// actorUserID must be the Competition's Host or the read is refused with
// domain.ErrNotCompetitionHost (PermissionDenied at the handler, never
// Internal).
//
// **Host-only is narrower than #147's entitled set, deliberately** — the
// admin list that would widen it is caller-supplied and persisted nowhere
// (#168), so honouring it would let any caller name themselves an admin.
// Whether an entrant should see the roster of a Competition they are in is an
// open product question #147 does not pre-answer; until it is answered, they
// may not. Social Play's ListRegistrationsForGame carries the full reasoning
// and this is its exact twin.
//
// **No identifier resolution happens here, on purpose** (ADR-0014 §5a).
// Competition.HostID is `text` holding the subject CreateCompetition minted
// from the verified principal, and actorUserID is what actor(ctx) returns —
// the same subject space on both sides, compared unchanged. ADR-0014 rules
// explicitly that T13.6 adds no Identity port to this context.
//
// The check uses only facts this Service already holds (s.competitions), so
// it adds no constructor dependency — a constraint T13.8 is planned on.
func (s *Service) ListEntriesForCompetition(ctx context.Context, competitionID, actorUserID string) ([]domain.CompetitionEntry, error) {
	// Same boundary guard as GetCompetition, but this read is list-shaped: an
	// unknown Competition yields an empty roster rather than an error, so a
	// malformed one must yield an empty roster too. Matching *this* method's
	// own not-found answer is the point — the invariant is "malformed is
	// indistinguishable from unknown", not "malformed is always NotFound".
	if !uuidShape.MatchString(competitionID) {
		return []domain.CompetitionEntry{}, nil
	}

	// Loading the Competition is what makes the check possible: HostID is the
	// fact being compared against. The not-found answer stays an empty roster
	// rather than becoming ErrCompetitionNotFound — that invariant is
	// unchanged by this ticket, and a Competition that does not exist has no
	// roster to withhold.
	competition, err := s.competitions.GetByID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			return []domain.CompetitionEntry{}, nil
		}
		return nil, err
	}

	if err := competition.EnsureHost(actorUserID); err != nil {
		return nil, err
	}

	return s.competitions.ListEntriesForCompetition(ctx, competitionID)
}

// MarkCompetitionEntryPaymentStatus updates a CompetitionEntry's
// PaymentStatus (T10.6, closes #96). This is the sole app-layer entry point
// for changing that field outside of EnterCompetition's initial "unpaid"
// default — internal/payments/adapter/competitions.EntryUpdater is the only
// caller, invoked through port.CompetitionEntryPaymentUpdater after a
// Payment for that CompetitionEntry transitions in the Payments context.
// Competitions itself never decides when a payment is made; it only records
// what Payments (the source of truth) reports, which is why this method does
// no authorization check of its own — the caller crossing the context
// boundary is Payments' own app.Service, not an end user request. Mirrors
// internal/socialplay/app.Service.MarkRegistrationPaymentStatus exactly.
func (s *Service) MarkCompetitionEntryPaymentStatus(ctx context.Context, entryID string, status domain.PaymentStatus) error {
	entry, err := s.competitions.GetEntryByID(ctx, entryID)
	if err != nil {
		return err
	}

	if err := entry.MarkPaymentStatus(status); err != nil {
		return err
	}

	_, err = s.competitions.UpdateEntryPaymentStatus(ctx, entry.ID, entry.PaymentStatus)
	return err
}

// CancelCompetition transitions a Competition to cancelled on the Host's
// behalf.
//
// domain.Competition.EnsureHost runs first: a non-Host actor is rejected
// with domain.ErrNotCompetitionHost (which T9.4 maps to PermissionDenied,
// never Internal) before any state changes. This is Competitions'
// object-level (BOLA) authorization check, and — as everywhere else in this
// codebase — actorUserID is a caller-supplied claim, not a verified
// identity: real authentication remains HANDOFF.md's open Auth item and
// this check must not be read as closing it.
//
// KNOWN GAP, inherited from domain.Competition.Cancel and restated here
// because this is the layer that could close it: cancelling does NOT
// release the Competition's court reservations or cancel its entries. The
// Bookings its sessions hold stay held. Releasing them is mechanically
// straightforward through the CourtReservation port already wired to this
// Service, but it needs the Competition's booking IDs, which nothing
// currently persists — port.Repository.Create stores the Competition and
// its sessions, not the Booking IDs the reservation loop received back.
// Wiring that is a schema question (T9.4 owns the tables) plus a decision
// about what a cancelled Competition's paid entries mean, so it is left
// visibly undone rather than half-implemented here. Entering a cancelled
// Competition is already blocked (domain.Enter), so the gap is "courts stay
// reserved and existing entries keep their status", not "a cancelled
// Competition still takes entries".
func (s *Service) CancelCompetition(ctx context.Context, competitionID, actorUserID string) (domain.Competition, error) {
	// Same T10.7 guard as EnterCompetition above, for the same reason: this
	// method calls GetByID(competitionID) first, before EnsureHost even
	// runs, and already returns the bare domain.ErrCompetitionNotFound for
	// an unknown-but-well-formed id — a malformed one must match that
	// exactly rather than reaching mustUUID and panicking.
	if !uuidShape.MatchString(competitionID) {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}

	competition, err := s.competitions.GetByID(ctx, competitionID)
	if err != nil {
		return domain.Competition{}, err
	}

	if err := competition.EnsureHost(actorUserID); err != nil {
		return domain.Competition{}, err
	}

	if err := competition.Cancel(); err != nil {
		return domain.Competition{}, err
	}

	return s.competitions.UpdateStatus(ctx, competition.ID, competition.Status)
}
