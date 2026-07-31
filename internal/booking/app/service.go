package app

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/booking/port"
)

// Service is the Booking context's application layer: it orchestrates the
// domain and the repository port, but holds no business rules itself — those
// live in internal/booking/domain.
type Service struct {
	repo port.Repository
	ids  port.IDGenerator
}

func NewService(repo port.Repository, ids port.IDGenerator) *Service {
	return &Service{repo: repo, ids: ids}
}

// CreateBookingInput is the use-case input for creating any of the four
// Booking sources (D3b) — the same use case serves a direct court booking, a
// Game reserving its courts, a Competition reserving its courts, and a Club
// recurring-hire occurrence, differing only in Source/ReferenceID.
type CreateBookingInput struct {
	CourtID     string
	Source      domain.Source
	Range       domain.TimeRange
	ReferenceID string
}

// CreateBooking validates the candidate booking, pre-checks it against the
// court's other active bookings (domain.EnsureNoConflict — regardless of
// their source, per D3b/F1), and persists it. The Postgres adapter's EXCLUDE
// constraint remains the authoritative guard under concurrent requests (see
// HANDOFF.md T4); this pre-check exists to fail fast and give a clear domain
// error without waiting on a round trip that's doomed to be rejected.
func (s *Service) CreateBooking(ctx context.Context, in CreateBookingInput) (domain.Booking, error) {
	candidate, err := domain.NewBooking(s.ids.NewID(), in.CourtID, in.Source, in.Range, in.ReferenceID)
	if err != nil {
		return domain.Booking{}, err
	}

	existing, err := s.repo.ListActiveForCourt(ctx, in.CourtID, in.Range)
	if err != nil {
		return domain.Booking{}, err
	}

	if err := domain.EnsureNoConflict(candidate, existing); err != nil {
		return domain.Booking{}, err
	}

	return s.repo.Create(ctx, candidate)
}
