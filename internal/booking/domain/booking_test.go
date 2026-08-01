package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

func mustRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	r, err := domain.NewTimeRange(mustTime(t, start), mustTime(t, end))
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}

func TestNewBooking_SourceValidation(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	tests := []struct {
		name    string
		source  domain.Source
		wantErr error
	}{
		{"recurring hire is valid", domain.SourceRecurringHire, nil},
		{"individual is valid", domain.SourceIndividual, nil},
		{"game is valid", domain.SourceGame, nil},
		{"competition is valid", domain.SourceCompetition, nil},
		{"unknown source is rejected", domain.Source("subscription"), domain.ErrInvalidSource},
		{"empty source is rejected", domain.Source(""), domain.ErrInvalidSource},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewBooking("b1", "court-1", tt.source, r, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewBooking_RequiresCourtID(t *testing.T) {
	t.Parallel()
	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	_, err := domain.NewBooking("b1", "", domain.SourceIndividual, r, "")
	if !errors.Is(err, domain.ErrEmptyCourtID) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyCourtID)
	}
}

func TestBooking_Cancel(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	b, err := domain.NewBooking("b1", "court-1", domain.SourceIndividual, r, "")
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := b.Cancel(); err != nil {
		t.Fatalf("first cancel should succeed, got %v", err)
	}
	if b.Status != domain.StatusCancelled {
		t.Fatalf("status = %v, want cancelled", b.Status)
	}

	if err := b.Cancel(); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("double-cancel should be rejected, got %v", err)
	}
}

// TestEnsureNoConflict_CrossSource proves the no-double-booking invariant
// covers every source uniformly (D3b / spec §6 / design-review F1) — a Game
// booking must be blocked by an existing Competition booking on the same
// court/slot, and vice versa. This is the domain-level mirror of the test
// QA required in docs/spec-design-review.md Topic 2.
func TestEnsureNoConflict_CrossSource(t *testing.T) {
	t.Parallel()

	overlapping := mustRange(t, "2026-08-03T09:30:00Z", "2026-08-03T10:30:00Z")

	tests := []struct {
		name           string
		existingRange  domain.TimeRange
		existingSrc    domain.Source
		candidateSrc   domain.Source
		existingCourt  string
		candidateCourt string
		existingStatus domain.Status
		wantConflict   bool
	}{
		{
			name:          "game vs competition same court overlapping conflicts",
			existingRange: mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			existingSrc:   domain.SourceCompetition, candidateSrc: domain.SourceGame,
			existingCourt: "court-1", candidateCourt: "court-1", existingStatus: domain.StatusConfirmed,
			wantConflict: true,
		},
		{
			name:          "individual vs recurring hire same court overlapping conflicts",
			existingRange: mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			existingSrc:   domain.SourceRecurringHire, candidateSrc: domain.SourceIndividual,
			existingCourt: "court-1", candidateCourt: "court-1", existingStatus: domain.StatusConfirmed,
			wantConflict: true,
		},
		{
			name:          "same source same court overlapping still conflicts",
			existingRange: mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			existingSrc:   domain.SourceGame, candidateSrc: domain.SourceGame,
			existingCourt: "court-1", candidateCourt: "court-1", existingStatus: domain.StatusConfirmed,
			wantConflict: true,
		},
		{
			name:          "different court never conflicts",
			existingRange: mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			existingSrc:   domain.SourceGame, candidateSrc: domain.SourceCompetition,
			existingCourt: "court-1", candidateCourt: "court-2", existingStatus: domain.StatusConfirmed,
			wantConflict: false,
		},
		{
			name:          "cancelled existing booking never conflicts",
			existingRange: mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			existingSrc:   domain.SourceGame, candidateSrc: domain.SourceCompetition,
			existingCourt: "court-1", candidateCourt: "court-1", existingStatus: domain.StatusCancelled,
			wantConflict: false,
		},
		{
			name:          "back-to-back booking does not conflict",
			existingRange: mustRange(t, "2026-08-03T08:00:00Z", "2026-08-03T09:30:00Z"),
			existingSrc:   domain.SourceIndividual, candidateSrc: domain.SourceGame,
			existingCourt: "court-1", candidateCourt: "court-1", existingStatus: domain.StatusConfirmed,
			wantConflict: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			existing, err := domain.NewBooking("existing", tt.existingCourt, tt.existingSrc, tt.existingRange, "")
			if err != nil {
				t.Fatalf("unexpected err building fixture: %v", err)
			}
			existing.Status = tt.existingStatus

			candidate, err := domain.NewBooking("candidate", tt.candidateCourt, tt.candidateSrc, overlapping, "")
			if err != nil {
				t.Fatalf("unexpected err building fixture: %v", err)
			}

			err = domain.EnsureNoConflict(candidate, []domain.Booking{existing})
			if tt.wantConflict {
				if !errors.Is(err, domain.ErrCourtDoubleBooked) {
					t.Fatalf("got err %v, want %v", err, domain.ErrCourtDoubleBooked)
				}
			} else if err != nil {
				t.Fatalf("expected no conflict, got %v", err)
			}
		})
	}
}

func TestEnsureNoConflict_IgnoresSelf(t *testing.T) {
	t.Parallel()
	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	b, err := domain.NewBooking("b1", "court-1", domain.SourceIndividual, r, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Re-validating a booking against a slice that includes itself (e.g. a
	// no-op update) must not conflict with itself.
	if err := domain.EnsureNoConflict(b, []domain.Booking{b}); err != nil {
		t.Fatalf("booking should not conflict with itself, got %v", err)
	}
}
