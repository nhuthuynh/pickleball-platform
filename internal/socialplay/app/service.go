package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// uuidShape matches the canonical 8-4-4-4-12 hex form internal/platform/idgen
// mints for every Game and Registration ID.
//
// Boundary guard for caller-supplied IDs: the Postgres adapter's mustUUID
// panics on anything pgtype.UUID.Scan can't parse, and grpc installs no
// recover() of its own, so an unvalidated ID off the wire could take the whole
// process down. Deliberately narrower than github.com/google/uuid's Validate,
// which accepts braced and `urn:uuid:` forms that pgtype rejects. The canonical
// write-up lives on internal/competitions/app's copy.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Social Play context's application layer: it orchestrates
// the domain and its ports, but holds no business rules itself — those live
// in internal/socialplay/domain. Mirrors internal/booking/app.Service's
// shape.
//
// games/registrations/waitlist are persistence ports (T5.4/T6.6);
// CourtReservation is deliberately NOT stored here (unlike
// games/registrations/waitlist) because it's passed per-call to
// ScheduleGame — see that method's doc comment for why (T5.3's original
// design, unchanged since).
//
// NewService reached 5 positional constructor args in T10.4 (matches).
// docs/process/t6-sprint-plan.md's kickoff note flagged the threshold at 4
// args ("worth revisiting... if a 4th dependency lands") for Payments' own
// Service and had it switch to an options struct from the start; the T6.6
// doc comment here left Social Play positional but flagged "a candidate for
// the same options-struct treatment if a 5th dependency ever lands", and
// T10.4 deferred it a second time to keep its diff additive while T10.2 was
// landing against the same file. T13.8 is the ticket that finally does it —
// see ServiceOptions below.
type Service struct {
	ids           port.IDGenerator
	games         port.GameRepository
	registrations port.RegistrationRepository
	waitlist      port.WaitlistRepository
	matches       port.MatchRepository
	gameAdmins    port.GameAdminRepository
}

// ServiceOptions is the dependency bundle for NewService (T13.8, closing
// issue #123).
//
// Mirrors internal/booking/app.ServiceOptions, which carries the canonical
// write-up; the shape of both follows paymentsapp.ServiceOptions (T6.4/T6.5),
// the older of the two pre-existing examples and the one that has already
// absorbed a cross-context dependency without touching a call site.
//
// **No field here is optional.** Every one is dereferenced unconditionally by
// at least one use case and no method in this package nil-guards any of them,
// which is why NewService validates rather than documenting which fields a
// caller may skip: there are none.
type ServiceOptions struct {
	// IDs mints every Game, Registration, WaitlistEntry and Match ID.
	IDs port.IDGenerator
	// Games is the Game aggregate's persistence port.
	Games port.GameRepository
	// Registrations backs joining, cancelling and the roster reads.
	Registrations port.RegistrationRepository
	// Waitlist backs JoinWaitlist and the auto-promotion path (T6.6).
	Waitlist port.WaitlistRepository
	// Matches backs the match-recording use cases (T10.4).
	Matches port.MatchRepository
	// GameAdmins backs the durable Game-Admin store (T14.4, partial fix for
	// #168): the assign/revoke use cases and the read T14.5 resolves an
	// entitled admin set from. Required like every other field here — the
	// three methods that use it dereference it unconditionally.
	GameAdmins port.GameAdminRepository
}

// ErrMissingDependency reports that a ServiceOptions left at least one
// required dependency unusable (unset, or holding a nil pointer).
var ErrMissingDependency = errors.New("socialplay/app: ServiceOptions is missing a required dependency")

// Validate reports every required dependency opts leaves unusable, or nil if
// the bundle is complete.
//
// The positional constructor this replaced made omitting a dependency a
// compile error; a struct makes it a silent nil. Validate moves that property
// from the compiler to construction time rather than dropping it — see
// internal/booking/app.ServiceOptions.Validate for the full argument, and
// internal/platform/auth.EnsureVerifierConfigured (T13.5) for the precedent.
// It reports the whole missing set at once so a mis-wiring is one fix rather
// than one fix per boot.
func (o ServiceOptions) Validate() error {
	var missing []string
	for _, dep := range []struct {
		name  string
		value any
	}{
		{"IDs", o.IDs},
		{"Games", o.Games},
		{"Registrations", o.Registrations},
		{"Waitlist", o.Waitlist},
		{"Matches", o.Matches},
		{"GameAdmins", o.GameAdmins},
	} {
		if isUnusable(dep.value) {
			missing = append(missing, dep.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrMissingDependency, strings.Join(missing, ", "))
}

// isUnusable reports whether a dependency is unset or holds a nil pointer.
//
// The typed-nil case is the one a plain `== nil` misses: an interface holding
// a nil pointer is itself non-nil, so it walks past the easy check and panics
// on its first method call anyway. Lifted from
// internal/platform/auth.verifierIsNil (T13.5); duplicated per context rather
// than shared, because no app package imports internal/platform — the same
// convention uuidShape above already follows.
func isUnusable(dep any) bool {
	if dep == nil {
		return true
	}
	switch v := reflect.ValueOf(dep); v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		// A non-pointer implementation (internal/platform/idgen.UUID is a
		// value type, as are several of this package's test fakes) cannot be
		// nil.
		return false
	}
}

// NewService wires the Social Play context's application layer.
//
// It panics if opts is incomplete: a missing dependency is a wiring bug in
// cmd/server or in a test, not a runtime condition, and construction is where
// the mistake was made. See internal/booking/app.NewService for the full
// rationale. Callers that want the error rather than the panic can call
// opts.Validate() themselves.
func NewService(opts ServiceOptions) *Service {
	if err := opts.Validate(); err != nil {
		panic(err.Error())
	}
	return &Service{
		ids:           opts.IDs,
		games:         opts.Games,
		registrations: opts.Registrations,
		waitlist:      opts.Waitlist,
		matches:       opts.Matches,
		gameAdmins:    opts.GameAdmins,
	}
}

// ScheduleGameInput is the use-case input for scheduling a Game.
type ScheduleGameInput struct {
	HostID     string
	FacilityID string
	// VenueFacilityID is a real Facilities-context Facility ID (T8.3) —
	// see domain.Game.VenueFacilityID's doc comment. Optional: an empty
	// value skips the port.FacilityLookup existence check entirely (see
	// ScheduleGame's doc comment).
	VenueFacilityID string
	CourtIDs        []string
	Range           domain.TimeRange
	Capacity        int
	// PaymentMethod and GuestAllowance (T8.7) are passed straight through to
	// domain.NewGame — see that constructor's doc comment for their
	// validation. internal/socialplay/adapter/grpcapi is responsible for
	// resolving an unset/zero-value wire PaymentMethod onto
	// domain.PaymentMethodEither before it reaches here (see that package's
	// fromProtoPaymentMethod); this field itself has no implicit default.
	PaymentMethod  domain.PaymentMethod
	GuestAllowance int
	// EntryFee (T9.2) is the price the Host set for one player to join this
	// Game, passed straight through to domain.NewGame — see
	// domain.Money.Validate for its rules. The zero value of this field is
	// domain.Money{} (0 cents, empty currency), which is valid and means a
	// free Game: unlike PaymentMethod above, an unset EntryFee needs no
	// resolution step in the adapter, because "free" is a real value rather
	// than a stand-in for "unspecified".
	EntryFee domain.Money
}

// ScheduleGame builds a Game (domain.NewGame, T5.1) and, once it validates,
// reserves one Booking per court in Game.CourtIDs via the CourtReservation
// port — this is what makes Social Play inherit the Booking context's
// no-double-booking invariant (T5.3's sprint goal) instead of merely
// modelling a Game with no effect on court availability.
//
// Multi-court atomicity: a Game can span more than one court (T5.1), so a
// single ScheduleGame call may need multiple ReserveCourt calls to
// succeed. If a later court conflicts, the courts that already reserved
// successfully must not be left dangling — this method rolls them back via
// best-effort compensating ReleaseCourt calls (see port.CourtReservation's
// doc comment) rather than restricting Games to a single court. That
// alternative (single-court-only) was considered and rejected: T5.1 already
// shipped Game.CourtIDs as a slice specifically to support multi-court
// games, and there is no real distributed-transaction problem here to avoid
// — Booking's CreateBooking calls are independent, idempotent-enough
// single-court operations, so "reserve in order, compensate on failure" is a
// complete, ticket-sized answer, not an improvised saga. See the PR
// description for the full writeup this is meant to be scrutinized against.
//
// Rollback is best-effort: if a ReleaseCourt call itself fails, ScheduleGame
// still returns the original reservation conflict (not a rollback-specific
// error) — that conflict is the actionable information for the caller, and
// masking it behind a secondary failure would make the real problem harder
// to diagnose. A ReleaseCourt failure does mean a dangling Booking can
// remain in the Booking context; that residue is called out as a known gap
// in the PR description (T5.4's real adapter should log/alert on it, since
// this in-memory-only ticket has no persistence to reconcile against).
//
// T5.4 adds persistence: once every court reserves successfully, the Game
// is written via the GameRepository port. If persistence itself fails, the
// already-made reservations are rolled back the same way a mid-loop
// reservation conflict is — a Game that failed to persist must not leave
// dangling Bookings behind, for the same "no half-scheduled Game" reason
// T5.3 rolls back on a reservation conflict.
//
// T8.3 adds a venue Facility existence check: once domain.NewGame's own
// invariants pass, but before any court is reserved or the Game is
// persisted, a non-empty game.VenueFacilityID is validated against the real
// Facilities context via the facilities port.FacilityLookup argument
// (mirrors reservation's per-call, not stored-on-Service, shape — see
// Service's doc comment on why CourtReservation is passed this way). An
// unknown VenueFacilityID returns domain.ErrFacilityNotFound *before*
// touching reservation at all, so — same "no half-scheduled Game"
// requirement as an invalid domain.NewGame input, or a mid-loop reservation
// conflict — a bogus VenueFacilityID can never leave a dangling Booking or
// a persisted Game behind. An empty VenueFacilityID skips this check
// entirely (domain.Game.VenueFacilityID's doc comment): most Games today
// don't set one, and the underlying column is nullable by design (db/
// migrations/0011_socialplay_facility_fk.sql).
func (s *Service) ScheduleGame(ctx context.Context, in ScheduleGameInput, reservation port.CourtReservation, facilities port.FacilityLookup) (domain.Game, error) {
	// domain.NewGame's PaymentMethod/GuestAllowance parameters (T8.6) are
	// wired straight through from ScheduleGameInput (T8.7) — see that
	// struct's doc comment for who's responsible for resolving an
	// unset/wire-zero-value PaymentMethod before it reaches here.
	game, err := domain.NewGame(s.ids.NewID(), in.HostID, in.FacilityID, in.VenueFacilityID, in.CourtIDs, in.Range, in.Capacity, in.PaymentMethod, in.GuestAllowance, in.EntryFee)
	if err != nil {
		return domain.Game{}, err
	}

	// Every court id is shape-checked here — the whole list, before the
	// facility lookup and before the reservation loop below (T14.8, closing
	// issue #156).
	//
	// This is the same T10.7 boundary guard the caller-supplied Game IDs in
	// this file get, applied to the one caller-supplied id *collection* on
	// this context's write path that the T10.7 sweep missed: CourtIDs is
	// consumed by the Booking context across port.CourtReservation, and that
	// sweep only covered ids consumed within this context. Without it,
	// bookingapp.CreateBooking's own uuidShape guard fired instead, and its
	// sentinel is correctly stripped at the adapter boundary (%s, not %w —
	// CLAUDE.md rule 5), so nothing survived for adapter/grpcapi's toStatus
	// to classify: a client typo answered Internal/500 while an empty
	// CourtIDs answered InvalidArgument/400.
	//
	// Placed AFTER domain.NewGame so the domain keeps first say on its own
	// invariants — an empty CourtIDs is still ErrEmptyCourtIDs, which says
	// more than "malformed" would. Placed BEFORE FacilityExists and the
	// reservation loop so a bad id costs no I/O and, more importantly, so a
	// malformed entry anywhere in the list can never leave a reserved court
	// or a persisted Game behind (the whole-input requirement; same
	// reasoning as the VenueFacilityID check immediately below).
	for _, courtID := range game.CourtIDs {
		if !uuidShape.MatchString(courtID) {
			return domain.Game{}, domain.ErrMalformedCourtID
		}
	}

	if game.VenueFacilityID != "" {
		if err := facilities.FacilityExists(ctx, game.VenueFacilityID); err != nil {
			return domain.Game{}, fmt.Errorf("socialplay: validating venue facility %s for game %s: %w", game.VenueFacilityID, game.ID, err)
		}
	}

	reservedBookingIDs := make([]string, 0, len(game.CourtIDs))
	for _, courtID := range game.CourtIDs {
		bookingID, err := reservation.ReserveCourt(ctx, courtID, game.Range.Start, game.Range.End, game.ID)
		if err != nil {
			releaseAll(ctx, reservation, reservedBookingIDs)
			return domain.Game{}, fmt.Errorf("socialplay: reserving court %s for game %s: %w", courtID, game.ID, err)
		}
		reservedBookingIDs = append(reservedBookingIDs, bookingID)
	}

	persisted, err := s.games.Create(ctx, game)
	if err != nil {
		releaseAll(ctx, reservation, reservedBookingIDs)
		return domain.Game{}, fmt.Errorf("socialplay: persisting game %s: %w", game.ID, err)
	}

	return persisted, nil
}

// releaseAll is the shared best-effort rollback helper for ScheduleGame's
// two failure points (a later court conflicting, or persistence itself
// failing after every court already reserved) — see the method's doc
// comment on why rollback is best-effort and doesn't mask the original error.
func releaseAll(ctx context.Context, reservation port.CourtReservation, bookingIDs []string) {
	for _, id := range bookingIDs {
		_ = reservation.ReleaseCourt(ctx, id)
	}
}

// RegisterForGameInput is the use-case input for registering a player into
// a Game.
type RegisterForGameInput struct {
	GameID   string
	PlayerID string
	// GuestCount (T8.7) is passed straight through to domain.Register — see
	// that function's doc comment for the guest-allowance and weighted
	// capacity checks it's validated against. Zero (the type's default) is
	// the pre-T8.6 "no guests" behavior, so a caller that never sets this
	// field is unaffected.
	GuestCount int
}

// RegisterForGame looks up the Game and its current active registrations,
// then applies domain.Register's capacity/double-registration checks
// before persisting the new Registration (HANDOFF.md T5.2/T5.4). The
// Postgres adapter's unique constraint on (game_id, player_id) for active
// registrations remains the authoritative guard under concurrency; this
// pre-check exists to fail fast with a clear domain error, mirroring
// CreateBooking's relationship with EnsureNoConflict/the EXCLUDE constraint.
//
// T6.6 addition: before touching domain.Register, this also checks whether
// the freed slot is currently reserved by another player's unexpired
// waitlist promotion (domain.SlotReservedByPromotion) — a promoted entry
// doesn't consume a Registration row, so Register's own count can't see it.
// A mismatched, non-promoted player during that window gets the same
// ErrGameFull a real capacity conflict would — from their point of view the
// slot genuinely isn't available yet. The promoted player's own call is
// exempted (SlotReservedByPromotion never blocks them), so this is also how
// a promotion gets "confirmed": the promoted player simply calls
// RegisterForGame like anyone else, no separate confirm RPC needed.
func (s *Service) RegisterForGame(ctx context.Context, in RegisterForGameInput) (domain.Registration, error) {
	// A malformed GameID is answered exactly like an unknown one (T10.7,
	// closing issue #97): this method already calls GetByID first and
	// already returns the bare domain.ErrGameNotFound for a miss — found by
	// this ticket's required inspection sweep, since RegisterForGame
	// (unlike ListRegistrationsForGame just above) had never had this guard
	// applied, though it reaches the identical mustUUID panic path.
	if !uuidShape.MatchString(in.GameID) {
		return domain.Registration{}, domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return domain.Registration{}, err
	}

	entries, err := s.waitlist.ListForGame(ctx, in.GameID)
	if err != nil {
		return domain.Registration{}, err
	}
	if domain.SlotReservedByPromotion(entries, in.PlayerID, time.Now()) {
		return domain.Registration{}, domain.ErrGameFull
	}

	existing, err := s.registrations.ListActiveForGame(ctx, in.GameID)
	if err != nil {
		return domain.Registration{}, err
	}

	// domain.Register's guestCount parameter (T8.6) is wired straight
	// through from RegisterForGameInput.GuestCount (T8.7).
	reg, err := domain.Register(game, existing, in.PlayerID, in.GuestCount)
	if err != nil {
		return domain.Registration{}, err
	}
	reg.ID = s.ids.NewID()

	return s.registrations.Create(ctx, reg)
}

// CancelRegistration transitions a registration to cancelled, but only when
// actorPlayerID matches the registration's own player — Registration.Cancel
// enforces the object-level (BOLA-shaped) ownership check (T5.2/P1 #6); this
// method's only job is the repository round trip, mirroring
// Service.CancelBooking's shape. Once cancelled, the slot it held is free —
// domain.Register already ignores cancelled registrations when counting
// capacity, so no separate "free the slot" step is needed here beyond
// persisting the status change itself.
//
// T6.6 auto-promotion: a successful cancellation is exactly the event
// ADR-0006 names as the promotion trigger ("the same event CancelBooking
// (T3) already produces"), so once the cancellation itself is durably
// persisted, this offers the freed slot to the Game's oldest waiting entry
// via promoteNextWaiting. This is a side effect of the cancellation, not a
// precondition for it: the cancellation has already succeeded by the time
// promotion is attempted, and a promotion-side failure is surfaced as an
// error from this call (so it isn't silently swallowed) but does not and
// cannot roll back the cancellation that already committed — mirrored
// return of the now-cancelled Registration either way so a caller who
// checks the error can still see what happened to the thing they asked to
// cancel.
func (s *Service) CancelRegistration(ctx context.Context, registrationID, actorPlayerID string) (domain.Registration, error) {
	// Same T10.7 guard RegisterForGame applies above, for the same reason:
	// this method calls GetByID(registrationID) first, before Cancel's own
	// ownership check even runs, and already returns the bare
	// domain.ErrRegistrationNotFound for an unknown-but-well-formed id.
	if !uuidShape.MatchString(registrationID) {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}

	reg, err := s.registrations.GetByID(ctx, registrationID)
	if err != nil {
		return domain.Registration{}, err
	}

	if err := reg.Cancel(actorPlayerID); err != nil {
		return domain.Registration{}, err
	}

	cancelled, err := s.registrations.Update(ctx, reg)
	if err != nil {
		return domain.Registration{}, err
	}

	if _, err := s.promoteNextWaiting(ctx, cancelled.GameID); err != nil {
		return cancelled, fmt.Errorf("socialplay: promoting next waitlist entry for game %s: %w", cancelled.GameID, err)
	}

	return cancelled, nil
}

// promoteNextWaiting offers the Game's oldest waiting entry a promotion via
// port.WaitlistRepository.PromoteNext — the DB-level race-closing operation
// (see db/migrations/0008_socialplay_waitlist_promotion.sql). An empty
// waitlist (domain.ErrNoWaitingEntries) is an expected, non-error outcome —
// most cancellations happen on Games with no one waiting — so it is
// swallowed here rather than propagated; any other error is a real failure
// and is returned to the caller.
func (s *Service) promoteNextWaiting(ctx context.Context, gameID string) (domain.WaitlistEntry, error) {
	promoted, err := s.waitlist.PromoteNext(ctx, gameID, time.Now())
	if err != nil {
		if errors.Is(err, domain.ErrNoWaitingEntries) {
			return domain.WaitlistEntry{}, nil
		}
		return domain.WaitlistEntry{}, err
	}
	return promoted, nil
}

// JoinWaitlistInput is the use-case input for joining a Game's waitlist.
type JoinWaitlistInput struct {
	GameID   string
	PlayerID string
}

// JoinWaitlist looks up the Game and its current active registrations and
// waitlist entries, then applies domain.JoinWaitlist's checks before
// persisting the new WaitlistEntry — mirrors RegisterForGame's shape
// exactly. The Postgres adapter's unique constraint on (game_id, player_id)
// for active waitlist entries remains the authoritative guard under
// concurrency; this pre-check exists to fail fast with a clear domain
// error, same relationship domain.Register has with its own DB backstop
// (CLAUDE.md rule 4).
func (s *Service) JoinWaitlist(ctx context.Context, in JoinWaitlistInput) (domain.WaitlistEntry, error) {
	// Same T10.7 guard RegisterForGame applies above, on the same field:
	// this method also calls GetByID(GameID) first and already returns the
	// bare domain.ErrGameNotFound for an unknown-but-well-formed id.
	if !uuidShape.MatchString(in.GameID) {
		return domain.WaitlistEntry{}, domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}

	existingRegs, err := s.registrations.ListActiveForGame(ctx, in.GameID)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}

	existingEntries, err := s.waitlist.ListForGame(ctx, in.GameID)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}

	entry, err := domain.JoinWaitlist(game, existingRegs, existingEntries, in.PlayerID)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}
	entry.ID = s.ids.NewID()

	return s.waitlist.Create(ctx, entry)
}

// ExpireWaitlistPromotion transitions a promoted WaitlistEntry to expired —
// but only once its PromotionResponseWindow has actually elapsed
// (domain.WaitlistEntry.HasExpired), rejecting a premature call with
// domain.ErrWaitlistPromotionNotExpired rather than silently honouring it —
// and then promotes the Game's next waiting entry in its place, the
// "cascades to the next waiter" requirement (ADR-0006 / T6.6). There is no
// scheduler/cron infrastructure in this codebase yet (HANDOFF.md); this
// method is the unit the future sweep job (or an on-demand check triggered
// by, e.g., a stale promotion being noticed at read time) calls per entry —
// building that scheduler itself is out of scope for this ticket, same as
// T6.3's no-show automation being deferred pending a real trigger.
func (s *Service) ExpireWaitlistPromotion(ctx context.Context, entryID string) (domain.WaitlistEntry, error) {
	entry, err := s.waitlist.GetByID(ctx, entryID)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}

	now := time.Now()
	if !entry.HasExpired(now) {
		return domain.WaitlistEntry{}, domain.ErrWaitlistPromotionNotExpired
	}

	expired, err := s.waitlist.ExpirePromotion(ctx, entryID, now)
	if err != nil {
		return domain.WaitlistEntry{}, err
	}

	if _, err := s.promoteNextWaiting(ctx, expired.GameID); err != nil {
		return expired, fmt.Errorf("socialplay: promoting next waitlist entry for game %s after expiry: %w", expired.GameID, err)
	}

	return expired, nil
}

// ListGames returns scheduled Games matching filter (T8.9's Discover & Join
// Games browse/search read), each paired with a server-computed SpotsLeft.
// All the actual filtering/aggregation lives in the repository — this
// method just delegates, mirroring facilities' app.Service.ListFacilities'
// "no business rules for a read-only filter/list" shape (CLAUDE.md rule 2:
// there is no domain invariant here to enforce, only a query).
func (s *Service) ListGames(ctx context.Context, filter port.GameListingFilter) ([]port.GameListing, error) {
	return s.games.ListGames(ctx, filter)
}

// ListRegistrationsForGame returns the active (non-cancelled) registrations
// for gameID. Added by T8.10 (Payments UI) as a disclosed backend finding,
// not part of that ticket's original "pure frontend" scope: the Host
// pending-cash-payments dashboard needs to enumerate a Game's Registrations
// (to find which ones are PaymentStatus == unpaid), and no RPC exposed
// RegistrationRepository.ListActiveForGame before this — it was previously
// reachable only from inside RegisterForGame's own capacity pre-check (see
// that method's doc comment). This is a pure read with no invariant to
// enforce (CLAUDE.md rule 2: no business rule belongs here, only the
// query), following the same "migration-free-read-path" pattern T8.2
// (ListFacilityCourts) and T8.9 (ListGames) already established this sprint
// for an analogous missing-list-RPC gap — no schema change, no new domain
// type.
//
// # T13.6: Host-only (partial fix for #147)
//
// This method shipped with the same object-level-read openness ListGames has
// — any caller could list any Game's registrations. That was wrong and is now
// fixed: #147 recorded that it returned every registrant's player_id,
// payment_status and guest_count to anyone holding a game_id, and game_id is
// readable straight off the *public* ListGames response, so the leak was
// enumerable. actorUserID must be the Game's Host or the read is refused with
// domain.ErrNotGameHost (PermissionDenied at the handler, never Internal).
//
// **Host-only is narrower than #147's entitled set, deliberately.** An
// assigned Game Admin should be able to read a roster too, but
// assigned_game_admin_user_ids is caller-supplied and persisted nowhere
// (#168; #149 has the Payments-side twin) — honouring a caller-supplied admin
// list here would let any caller name themselves an admin and read the roster
// anyway, i.e. authorization
// theatre strictly worse than the Host check. That is why this method takes
// no AssignedGameAdminUserIDs argument and uses Game.EnsureHost rather than
// EnsureHostOrGameAdmin: the difference is a stated rule, not an accident of
// the caller's arguments. Whether a *registrant* should see the roster of a
// Game they are in is an open product question #147 deliberately does not
// pre-answer; until it is answered, they may not.
//
// **No identifier resolution happens here, on purpose** (ADR-0014 §5a).
// Game.HostID is `text` holding the subject CreateGame minted from the
// verified principal, and actorUserID is the value actor(ctx) returns — the
// same subject space on both sides. ADR-0014 rules explicitly that T13.6
// compares the two unchanged and does NOT add an Identity port to this
// context: doing so would put a uuid on one side of a comparison whose other
// side is a subject, turning a working check into one that silently denies
// everybody.
//
// The authorization needs only facts this Service already holds (s.games),
// so it adds no constructor dependency — a constraint T13.8 is planned on.
func (s *Service) ListRegistrationsForGame(ctx context.Context, gameID, actorUserID string) ([]domain.Registration, error) {
	// A malformed gameID is answered exactly like an unknown one. This read is
	// list-shaped — an unknown Game yields an empty roster rather than an
	// error — so a malformed Game must yield an empty roster too.
	if !uuidShape.MatchString(gameID) {
		return []domain.Registration{}, nil
	}

	// Loading the Game is what makes the check possible: HostID is the fact
	// being compared against. Note the not-found answer stays an empty roster
	// rather than becoming ErrGameNotFound — the "malformed is
	// indistinguishable from unknown" invariant above is unchanged by this
	// ticket, and a Game that does not exist has no roster to withhold.
	game, err := s.games.GetByID(ctx, gameID)
	if err != nil {
		if errors.Is(err, domain.ErrGameNotFound) {
			return []domain.Registration{}, nil
		}
		return nil, err
	}

	if err := game.EnsureHost(actorUserID); err != nil {
		return nil, err
	}

	return s.registrations.ListActiveForGame(ctx, gameID)
}

// MarkRegistrationPaymentStatus updates a Registration's PaymentStatus
// (T6.5). This is the sole app-layer entry point for changing that field
// outside of RegisterForGame's initial "unpaid" default —
// internal/payments/adapter/socialplay.RegistrationUpdater is the only
// caller, invoked through port.RegistrationPaymentUpdater after a Payment
// for that Registration transitions in the Payments context. Social Play
// itself never decides when a payment is made; it only records what
// Payments (the source of truth) reports, which is why this method does no
// authorization check of its own — the caller crossing the context
// boundary is Payments' own app.Service, not an end user request.
func (s *Service) MarkRegistrationPaymentStatus(ctx context.Context, registrationID string, status domain.PaymentStatus) error {
	reg, err := s.registrations.GetByID(ctx, registrationID)
	if err != nil {
		return err
	}

	if err := reg.MarkPaymentStatus(status); err != nil {
		return err
	}

	_, err = s.registrations.UpdatePaymentStatus(ctx, reg.ID, reg.PaymentStatus)
	return err
}

// RecordMatchResultInput is the use-case input for recording a Match result
// against a Game (T10.4).
type RecordMatchResultInput struct {
	GameID  string
	Players []string
	Score   map[string]int

	// ActorUserID is the claimed identity attempting to record this result
	// — same known-gap caveat as every other actor-scoped check in this
	// codebase (RegistrationRequest.actor_player_id, RecordOfflinePayment's
	// ActorUserID, ...): a request-supplied field, not a verified identity.
	ActorUserID string

	// AssignedGameAdminUserIDs is the caller-supplied set of user ids
	// currently assigned as a Game Admin for GameID's Game — see
	// domain.Game.EnsureHostOrGameAdmin's doc comment for why this is a
	// caller-supplied fact rather than a persisted one (this codebase has
	// never built a durable Game-Admin-assignment mechanism; T6.3's
	// RecordOfflinePaymentInput.AssignedGameAdminUserIDs is the identical
	// precedent).
	AssignedGameAdminUserIDs []string
}

// RecordMatchResult validates and persists a Match result against an
// existing Game (T10.4). Order of checks, and why:
//
//  1. Malformed GameID (T10.7-shaped boundary guard, uuidShape) is rejected
//     with the bare domain.ErrGameNotFound BEFORE touching the repository at
//     all — mirrors RegisterForGame/JoinWaitlist's identical guard, for the
//     identical reason (the Postgres adapter's mustUUID panics on a
//     non-UUID, and grpc installs no recover() of its own).
//  2. s.games.GetByID: an unknown-but-well-formed GameID gets the
//     repository's own domain.ErrGameNotFound, the same answer a malformed
//     one already gets from step 1 (T10.7's own "no worse than an
//     unknown-but-well-formed id" requirement).
//  3. game.EnsureHostOrGameAdmin(ActorUserID, AssignedGameAdminUserIDs): the
//     object-level (BOLA) authorization check, run before either the
//     cancelled-game precondition or domain.RecordMatch's own field
//     validation — mirrors authorizeOfflineRecording's ordering
//     (internal/payments/app/service.go: "checked first... so an
//     unauthorized actor never learns anything about why the input would
//     otherwise be invalid"), extended here to also cover the Game's own
//     state: an unauthorized caller learns nothing about whether the Game
//     happens to be cancelled either.
//  4. game.EnsureNotCancelled: FailedPrecondition per this ticket's
//     error-handling table — a cancelled Game can no longer have results
//     recorded against it.
//  5. domain.RecordMatch: GameID/Players/Score field validation (T10.3,
//     T10.4's ErrEmptyScore addition) — InvalidArgument.
//
// The returned Match's ID is assigned here (s.ids.NewID()), mirroring
// RegisterForGame/JoinWaitlist's identical "domain constructor doesn't take
// an id, app layer assigns one at persistence time" pattern.
func (s *Service) RecordMatchResult(ctx context.Context, in RecordMatchResultInput) (domain.Match, error) {
	if !uuidShape.MatchString(in.GameID) {
		return domain.Match{}, domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return domain.Match{}, err
	}

	if err := game.EnsureHostOrGameAdmin(in.ActorUserID, in.AssignedGameAdminUserIDs); err != nil {
		return domain.Match{}, err
	}

	if err := game.EnsureNotCancelled(); err != nil {
		return domain.Match{}, err
	}

	m, err := domain.RecordMatch(in.GameID, in.Players, in.Score, time.Now())
	if err != nil {
		return domain.Match{}, err
	}
	m.ID = s.ids.NewID()

	return s.matches.Create(ctx, m)
}

// ListMatchesForGame returns every Match recorded against gameID (T10.4).
// Unlike ListRegistrationsForGame's "unknown Game yields an empty roster"
// convention, this ticket's own error-handling instructions require
// NotFound for an unknown GameID, so this method does its own existence
// check via s.games.GetByID before delegating to the repository — a real
// read, not a pure pass-through, despite having no business rule of its own
// to enforce (CLAUDE.md rule 2: the existence check is the one thing this
// method does beyond delegating).
func (s *Service) ListMatchesForGame(ctx context.Context, gameID string) ([]domain.Match, error) {
	// Same T10.7-shaped boundary guard RecordMatchResult applies above, for
	// the identical reason: this method calls GetByID(gameID) next, and a
	// malformed id must answer no worse than an unknown-but-well-formed one
	// already does — both are domain.ErrGameNotFound here (NotFound), unlike
	// ListRegistrationsForGame's list-shaped "empty roster" answer, because
	// this RPC's own error-handling table requires NotFound specifically.
	if !uuidShape.MatchString(gameID) {
		return nil, domain.ErrGameNotFound
	}

	if _, err := s.games.GetByID(ctx, gameID); err != nil {
		return nil, err
	}

	return s.matches.ListForGame(ctx, gameID)
}

// CancelGame transitions a Game to cancelled on its Host's behalf (T12.4) —
// the RPC that domain.Game.Cancel has had no caller for since T5.1.
//
// domain.Game.EnsureHost runs first, before any state changes and before
// any status precondition: a non-Host actor is rejected with
// domain.ErrNotGameHost (which the handler maps to PermissionDenied, never
// Internal). This is Social Play's Game-level object-level (BOLA)
// authorization check, and it is deliberately **Host-only** — an assigned
// Game Admin, who may record a match result via RecordMatchResult, may not
// cancel the Game. That is why this method takes no
// AssignedGameAdminUserIDs argument and why ErrNotGameHost is a separate
// sentinel from ErrNotGameHostOrAdmin (see domain/errors.go).
//
// As everywhere else in this codebase, actorPlayerID is a caller-supplied
// claim, not a verified identity: this is an object-level check given a
// claimed actor, not authentication. Real authentication remains
// HANDOFF.md's open Auth item — T12.8 migrates this context to a verified
// principal, and only then does this check become trustworthy end to end.
//
// The already-cancelled case deliberately goes through
// Game.EnsureNotCancelled (-> domain.ErrGameCancelled ->
// codes.FailedPrecondition) rather than relying on Game.Cancel's own
// ErrIllegalStatusTransition. T12.4's error table requires
// FailedPrecondition for a second cancel, but socialplay's toStatus maps
// ErrIllegalStatusTransition to InvalidArgument — and it is shared with the
// three waitlist transition paths (domain/waitlist.go), so re-mapping the
// sentinel globally would silently change JoinWaitlist/promotion/expiry
// response codes for a rule this ticket has no business touching. Cancel()
// is still called afterwards and remains the domain's authoritative
// transition guard; with Status's current two-value enum the two checks
// coincide, but the guard is kept so a future third status is still
// rejected by the domain rather than only by this method.
//
// KNOWN GAP, inherited from domain.Game.Cancel and restated here because
// this is the layer that could close it: cancelling a Game does NOT cascade
// to the court Bookings it reserved or to its Registrations. Tracked in
// GitHub issue #124; it needs a product decision (are the reserved courts
// freed, are players refunded) before it can be built, so it is left
// visibly undone rather than half-implemented here.
func (s *Service) CancelGame(ctx context.Context, gameID, actorPlayerID string) (domain.Game, error) {
	// Same T10.7-shaped boundary guard the methods above apply, for the
	// identical reason: this method calls GetByID(gameID) next, and a
	// malformed id must answer exactly what an unknown-but-well-formed one
	// answers (domain.ErrGameNotFound) rather than reaching the Postgres
	// adapter's mustUUID and panicking.
	if !uuidShape.MatchString(gameID) {
		return domain.Game{}, domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, gameID)
	if err != nil {
		return domain.Game{}, err
	}

	if err := game.EnsureHost(actorPlayerID); err != nil {
		return domain.Game{}, err
	}

	if err := game.EnsureNotCancelled(); err != nil {
		return domain.Game{}, err
	}

	if err := game.Cancel(); err != nil {
		return domain.Game{}, err
	}

	return s.games.UpdateStatus(ctx, game.ID, game.Status)
}

// AssignGameAdminInput is the use-case input for assigning a Game Admin
// (T14.4, partial fix for #168).
//
// There is deliberately no field here for a caller-supplied admin list. That
// is the shape this ticket exists to retire: the whole point of the store is
// that "who is an admin of this Game" becomes a fact the server holds, not one
// the request asserts.
type AssignGameAdminInput struct {
	// GameID is the Game the assignment is scoped to.
	GameID string
	// ActorUserID is the acting identity, resolved by the grpcapi adapter
	// from the verified auth.Principal (T12.8/ADR-0014), never read off the
	// wire. Only the Game's Host passes domain.AssignGameAdmin's check.
	ActorUserID string
	// AdminUserID is the subject being granted Game-Admin authority.
	AdminUserID string
}

// RevokeGameAdminInput is the use-case input for withdrawing a Game-Admin
// assignment (T14.4). Same actor semantics as AssignGameAdminInput.
type RevokeGameAdminInput struct {
	GameID      string
	ActorUserID string
	AdminUserID string
}

// AssignGameAdmin records that a Game's Host granted Game-Admin authority over
// that Game to another user (T14.4, partial fix for #168).
//
// Order of checks, and why:
//
//  1. A malformed GameID is answered with the bare domain.ErrGameNotFound
//     before the repository is touched at all — the T10.7-shaped boundary
//     guard every caller-supplied Game id in this file already carries, for
//     the identical reason (the Postgres adapter's mustUUID panics on a
//     non-uuid, and grpc installs no recover() of its own).
//  2. s.games.GetByID: the Game must exist, and its HostID is the fact the
//     authorization check compares against — loading it is what makes the
//     check possible, exactly as in ListRegistrationsForGame (T13.6).
//  3. s.gameAdmins.ListGameAdmins: the Game's current assignments, handed to
//     the domain as the already-assigned pre-check's input.
//  4. domain.AssignGameAdmin: the Host-only authorization rule (**including
//     "an admin cannot appoint an admin"**), the blank- and self-assignment
//     input rules, and the duplicate pre-check. All of them live there, not
//     here — this method holds no business rule of its own (CLAUDE.md rule 2).
//
// An unknown Game answers domain.ErrGameNotFound rather than the empty answer
// ListRegistrationsForGame gives, because this is a write: there is no
// "nothing to show you" outcome, and a Host whose assignment silently landed
// nowhere is worse served than one told the Game does not exist.
//
// Under concurrency, game_admins' composite primary key is the authoritative
// duplicate guard; step 3/4's pre-check exists to fail fast with a clear
// domain error, the same relationship domain.Register has with the
// registrations unique index (CLAUDE.md rule 4). The Postgres adapter
// translates that constraint's violation into the same
// domain.ErrAlreadyGameAdmin this method's pre-check returns, so a caller
// cannot tell which of the two answered — and has no reason to.
func (s *Service) AssignGameAdmin(ctx context.Context, in AssignGameAdminInput) (domain.GameAdmin, error) {
	if !uuidShape.MatchString(in.GameID) {
		return domain.GameAdmin{}, domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return domain.GameAdmin{}, err
	}

	existing, err := s.gameAdmins.ListGameAdmins(ctx, in.GameID)
	if err != nil {
		return domain.GameAdmin{}, err
	}

	admin, err := domain.AssignGameAdmin(game, existing, in.ActorUserID, in.AdminUserID, time.Now())
	if err != nil {
		return domain.GameAdmin{}, err
	}

	return s.gameAdmins.Assign(ctx, admin)
}

// RevokeGameAdmin withdraws a Game-Admin assignment on the Game Host's behalf
// (T14.4). Host-only, via domain.EnsureMayRevokeGameAdmin: granting and
// withdrawing a delegated authority are the same authority, so an assigned
// Game Admin is refused here exactly as they are on the assign path.
//
// Revoking a user who holds no assignment returns domain.ErrGameAdminNotFound
// rather than succeeding silently — see port.GameAdminRepository.Revoke's doc
// comment for why a no-op success is the wrong answer to a Host asserting a
// belief about who holds authority. The authorization check still runs first,
// so an unauthorized caller cannot use the distinction between "not assigned"
// and "revoked" to enumerate a Game's admins.
func (s *Service) RevokeGameAdmin(ctx context.Context, in RevokeGameAdminInput) error {
	if !uuidShape.MatchString(in.GameID) {
		return domain.ErrGameNotFound
	}

	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return err
	}

	if err := domain.EnsureMayRevokeGameAdmin(game, in.ActorUserID); err != nil {
		return err
	}

	return s.gameAdmins.Revoke(ctx, in.GameID, in.AdminUserID)
}

// ListGameAdmins returns every Game-Admin assignment recorded against gameID
// (T14.4) — the read half of the sprint plan's §A13 GAP A, and the method
// T14.5 resolves its entitled set from via domain.HasGameAdmin.
//
// An unknown or malformed gameID answers domain.ErrGameNotFound rather than an
// empty slice, following ListMatchesForGame's convention rather than
// ListRegistrationsForGame's: an empty admin list is a real and common state
// (most Games have no admins), so returning it for a Game that does not exist
// would make the two indistinguishable to the one kind of caller — an
// authorization decision — that must not confuse them.
//
// **This method performs no authorization check of its own, and is
// deliberately not exposed as an RPC.** It is an in-process read for callers
// that have already authorized their own operation: AssignGameAdmin's
// pre-check above, and T14.5's roster and match paths, which run their own
// Host-or-admin check against the set this returns. That is the shape
// MarkRegistrationPaymentStatus already has (no check of its own, because its
// only caller crosses a context boundary rather than arriving from an end
// user). Putting a Game's admin roster on the wire needs its own decision
// about who may see it — plausibly the Host and the admins themselves,
// plausibly every registrant — and no ticket has made that decision; inventing
// an answer here would be a product decision smuggled in as an implementation
// detail. Named as out of scope so it is a decision rather than an omission.
func (s *Service) ListGameAdmins(ctx context.Context, gameID string) ([]domain.GameAdmin, error) {
	if !uuidShape.MatchString(gameID) {
		return nil, domain.ErrGameNotFound
	}

	if _, err := s.games.GetByID(ctx, gameID); err != nil {
		return nil, err
	}

	return s.gameAdmins.ListGameAdmins(ctx, gameID)
}
