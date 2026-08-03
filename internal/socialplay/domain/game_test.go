package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

func validCourtIDs() []string {
	return []string{"court-1"}
}

// TestNewGame_Valid proves the happy path: a well-formed Game is constructed
// with Status starting at "scheduled".
func TestNewGame_Valid(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", validCourtIDs(), r, 4)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.Status != domain.StatusScheduled {
		t.Fatalf("status = %v, want %v", g.Status, domain.StatusScheduled)
	}
	if g.Capacity != 4 {
		t.Fatalf("capacity = %d, want 4", g.Capacity)
	}
	if g.ID != "g1" || g.HostID != "host-1" || g.FacilityID != "facility-1" {
		t.Fatalf("unexpected identity fields: %+v", g)
	}
}

// TestNewGame_Validation is the required table-driven boundary coverage:
// Capacity == 0, negative capacity, empty CourtIDs, zero-duration range.
func TestNewGame_Validation(t *testing.T) {
	t.Parallel()

	validRange := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	// Zero-duration range built by bypassing NewTimeRange (which itself
	// rejects this) via the exported struct fields directly — this is how a
	// caller could hand NewGame an already-invalid range, so NewGame must
	// re-validate rather than trust its input, mirroring the "end after
	// start" rule booking.NewTimeRange enforces.
	sameInstant := mustTime(t, "2026-08-03T09:00:00Z")
	zeroDurationRange := domain.TimeRange{Start: sameInstant, End: sameInstant}

	tests := []struct {
		name     string
		hostID   string
		courtIDs []string
		r        domain.TimeRange
		capacity int
		wantErr  error
	}{
		{"capacity zero is rejected", "host-1", validCourtIDs(), validRange, 0, domain.ErrInvalidCapacity},
		{"negative capacity is rejected", "host-1", validCourtIDs(), validRange, -1, domain.ErrInvalidCapacity},
		{"empty court ids is rejected", "host-1", []string{}, validRange, 4, domain.ErrEmptyCourtIDs},
		{"nil court ids is rejected", "host-1", nil, validRange, 4, domain.ErrEmptyCourtIDs},
		{"zero-duration range is rejected", "host-1", validCourtIDs(), zeroDurationRange, 4, domain.ErrInvalidTimeRange},
		{"valid inputs accepted", "host-1", validCourtIDs(), validRange, 1, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewGame("g1", tt.hostID, "facility-1", tt.courtIDs, tt.r, tt.capacity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestGame_Cancel proves the scheduled -> cancelled transition and that a
// double-cancel is rejected rather than silently accepted, mirroring
// booking.Booking.Cancel()'s "illegal transition rejected" pattern. Note:
// cascading this cancellation to the Game's Bookings/Registrations is
// explicitly out of scope for T5.1 (see PR description / HANDOFF follow-up).
func TestGame_Cancel(t *testing.T) {
	t.Parallel()

	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", validCourtIDs(), r, 4)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := g.Cancel(); err != nil {
		t.Fatalf("first cancel should succeed, got %v", err)
	}
	if g.Status != domain.StatusCancelled {
		t.Fatalf("status = %v, want cancelled", g.Status)
	}

	if err := g.Cancel(); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("double-cancel should be rejected, got %v", err)
	}
}
