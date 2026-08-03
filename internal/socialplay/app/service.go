package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

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
// NewService is now at 4 positional constructor args (T6.6 adds waitlist).
// docs/process/t6-sprint-plan.md's kickoff note flags this exact threshold
// ("worth revisiting... if a 4th dependency lands") for Payments' own
// Service and has it switch to an options struct from the start; Social
// Play's constructor is left positional here rather than folded into that
// refactor, since this ticket is not the one that raised it and doing so
// would be an unrelated, ticket-widening change to every existing call
// site. Flagged in the PR description as a candidate for the same
// options-struct treatment if a 5th dependency ever lands.
type Service struct {
	ids           port.IDGenerator
	games         port.GameRepository
	registrations port.RegistrationRepository
	waitlist      port.WaitlistRepository
}

func NewService(ids port.IDGenerator, games port.GameRepository, registrations port.RegistrationRepository, waitlist port.WaitlistRepository) *Service {
	return &Service{ids: ids, games: games, registrations: registrations, waitlist: waitlist}
}

// ScheduleGameInput is the use-case input for scheduling a Game.
type ScheduleGameInput struct {
	HostID     string
	FacilityID string
	CourtIDs   []string
	Range      domain.TimeRange
	Capacity   int
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
func (s *Service) ScheduleGame(ctx context.Context, in ScheduleGameInput, reservation port.CourtReservation) (domain.Game, error) {
	game, err := domain.NewGame(s.ids.NewID(), in.HostID, in.FacilityID, in.CourtIDs, in.Range, in.Capacity)
	if err != nil {
		return domain.Game{}, err
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

	reg, err := domain.Register(game, existing, in.PlayerID)
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
