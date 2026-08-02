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
type Service struct {
	ids port.IDGenerator
}

func NewService(ids port.IDGenerator) *Service {
	return &Service{ids: ids}
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
func (s *Service) ScheduleGame(ctx context.Context, in ScheduleGameInput, reservation port.CourtReservation) (domain.Game, error) {
	game, err := domain.NewGame(s.ids.NewID(), in.HostID, in.FacilityID, in.CourtIDs, in.Range, in.Capacity)
	if err != nil {
		return domain.Game{}, err
	}

	reservedBookingIDs := make([]string, 0, len(game.CourtIDs))
	for _, courtID := range game.CourtIDs {
		bookingID, err := reservation.ReserveCourt(ctx, courtID, game.Range.Start, game.Range.End, game.ID)
		if err != nil {
			for _, id := range reservedBookingIDs {
				_ = reservation.ReleaseCourt(ctx, id)
			}
			return domain.Game{}, fmt.Errorf("socialplay: reserving court %s for game %s: %w", courtID, game.ID, err)
		}
		reservedBookingIDs = append(reservedBookingIDs, bookingID)
	}

	return game, nil
}
