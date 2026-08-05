package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad fixture time %q: %v", s, err)
	}
	return ts
}

func mustRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	r, err := domain.NewTimeRange(mustTime(t, start), mustTime(t, end))
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}

// TestNewTimeRange_Validity proves competitions' locally-duplicated TimeRange
// enforces the same "end after start" rule as booking's and socialplay's
// domain.TimeRange (CLAUDE.md rule 2: same semantics, independently
// reimplemented, never imported across contexts).
func TestNewTimeRange_Validity(t *testing.T) {
	t.Parallel()

	start := mustTime(t, "2026-09-05T09:00:00Z")

	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr error
	}{
		{"end after start is valid", start, start.Add(time.Hour), nil},
		{"end equal to start is rejected", start, start, domain.ErrInvalidTimeRange},
		{"end before start is rejected", start, start.Add(-time.Hour), domain.ErrInvalidTimeRange},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := domain.NewTimeRange(tt.start, tt.end)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !r.Start.Equal(tt.start) || !r.End.Equal(tt.end) {
				t.Fatalf("range = %+v, want [%v, %v)", r, tt.start, tt.end)
			}
		})
	}
}

// TestTimeRange_Overlaps proves the half-open [Start, End) semantics: two
// sessions where one ends exactly when the other starts do NOT overlap, the
// same rule the Booking invariant uses.
func TestTimeRange_Overlaps(t *testing.T) {
	t.Parallel()

	base := mustRange(t, "2026-09-05T10:00:00Z", "2026-09-05T12:00:00Z")

	tests := []struct {
		name  string
		other domain.TimeRange
		want  bool
	}{
		{"identical ranges overlap", base, true},
		{"partial overlap at the end", mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T13:00:00Z"), true},
		{"partial overlap at the start", mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T10:30:00Z"), true},
		{"fully contained overlaps", mustRange(t, "2026-09-05T10:30:00Z", "2026-09-05T11:00:00Z"), true},
		{"fully containing overlaps", mustRange(t, "2026-09-05T08:00:00Z", "2026-09-05T14:00:00Z"), true},
		{"back-to-back after does not overlap", mustRange(t, "2026-09-05T12:00:00Z", "2026-09-05T13:00:00Z"), false},
		{"back-to-back before does not overlap", mustRange(t, "2026-09-05T08:00:00Z", "2026-09-05T10:00:00Z"), false},
		{"disjoint does not overlap", mustRange(t, "2026-09-06T10:00:00Z", "2026-09-06T12:00:00Z"), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := base.Overlaps(tt.other); got != tt.want {
				t.Fatalf("base.Overlaps(other) = %v, want %v", got, tt.want)
			}
			// Overlap is symmetric — assert both directions so a
			// one-sided implementation can't pass.
			if got := tt.other.Overlaps(base); got != tt.want {
				t.Fatalf("other.Overlaps(base) = %v, want %v", got, tt.want)
			}
		})
	}
}
