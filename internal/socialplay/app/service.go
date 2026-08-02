package app

import (
	"context"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// Service is the Social Play context's application layer: it orchestrates
// the domain and its ports, but holds no business rules itself — those live
// in internal/socialplay/domain. Mirrors internal/booking/app.Service's
// shape.
//
// games/registrations are persistence ports (T5.4); CourtReservation is
// deliberately NOT stored here (unlike games/registrations) because it's
// passed per-call to ScheduleGame — see that method's doc comment for why
// (T5.3's original design, unchanged by T5.4's persistence wiring).
type Service struct {
	ids           port.IDGenerator
	games         port.GameRepository
	registrations port.RegistrationRepository
}

func NewService(ids port.IDGenerator, games port.GameRepository, registrations port.RegistrationRepository) *Service {
	return &Service{ids: ids, games: games, registrations: registrations}
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
func (s *Service) RegisterForGame(ctx context.Context, in RegisterForGameInput) (domain.Registration, error) {
	game, err := s.games.GetByID(ctx, in.GameID)
	if err != nil {
		return domain.Registration{}, err
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
func (s *Service) CancelRegistration(ctx context.Context, registrationID, actorPlayerID string) (domain.Registration, error) {
	reg, err := s.registrations.GetByID(ctx, registrationID)
	if err != nil {
		return domain.Registration{}, err
	}

	if err := reg.Cancel(actorPlayerID); err != nil {
		return domain.Registration{}, err
	}

	return s.registrations.Update(ctx, reg)
}
