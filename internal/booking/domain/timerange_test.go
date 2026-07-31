package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad fixture time %q: %v", s, err)
	}
	return ts
}

func TestNewTimeRange_Validity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		start   string
		end     string
		wantErr error
	}{
		{"valid one hour", "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z", nil},
		{"end equals start is invalid", "2026-08-03T09:00:00Z", "2026-08-03T09:00:00Z", domain.ErrInvalidTimeRange},
		{"end before start is invalid", "2026-08-03T10:00:00Z", "2026-08-03T09:00:00Z", domain.ErrInvalidTimeRange},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewTimeRange(mustTime(t, tt.start), mustTime(t, tt.end))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimeRange_Overlaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		aStart      string
		aEnd        string
		bStart      string
		bEnd        string
		wantOverlap bool
	}{
		{
			name: "identical ranges overlap", wantOverlap: true,
			aStart: "2026-08-03T09:00:00Z", aEnd: "2026-08-03T10:00:00Z",
			bStart: "2026-08-03T09:00:00Z", bEnd: "2026-08-03T10:00:00Z",
		},
		{
			name: "b starts mid-a overlaps", wantOverlap: true,
			aStart: "2026-08-03T09:00:00Z", aEnd: "2026-08-03T10:00:00Z",
			bStart: "2026-08-03T09:30:00Z", bEnd: "2026-08-03T10:30:00Z",
		},
		{
			name: "b fully inside a overlaps", wantOverlap: true,
			aStart: "2026-08-03T09:00:00Z", aEnd: "2026-08-03T11:00:00Z",
			bStart: "2026-08-03T09:30:00Z", bEnd: "2026-08-03T10:00:00Z",
		},
		{
			name: "back-to-back: a ends exactly when b starts does NOT overlap", wantOverlap: false,
			aStart: "2026-08-03T09:00:00Z", aEnd: "2026-08-03T10:00:00Z",
			bStart: "2026-08-03T10:00:00Z", bEnd: "2026-08-03T11:00:00Z",
		},
		{
			name: "back-to-back reversed does NOT overlap", wantOverlap: false,
			aStart: "2026-08-03T10:00:00Z", aEnd: "2026-08-03T11:00:00Z",
			bStart: "2026-08-03T09:00:00Z", bEnd: "2026-08-03T10:00:00Z",
		},
		{
			name: "disjoint, gap between does NOT overlap", wantOverlap: false,
			aStart: "2026-08-03T09:00:00Z", aEnd: "2026-08-03T10:00:00Z",
			bStart: "2026-08-03T11:00:00Z", bEnd: "2026-08-03T12:00:00Z",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, err := domain.NewTimeRange(mustTime(t, tt.aStart), mustTime(t, tt.aEnd))
			if err != nil {
				t.Fatalf("bad fixture range a: %v", err)
			}
			b, err := domain.NewTimeRange(mustTime(t, tt.bStart), mustTime(t, tt.bEnd))
			if err != nil {
				t.Fatalf("bad fixture range b: %v", err)
			}

			if got := a.Overlaps(b); got != tt.wantOverlap {
				t.Errorf("a.Overlaps(b) = %v, want %v", got, tt.wantOverlap)
			}
			if got := b.Overlaps(a); got != tt.wantOverlap {
				t.Errorf("Overlaps must be symmetric: b.Overlaps(a) = %v, want %v", got, tt.wantOverlap)
			}
		})
	}
}
