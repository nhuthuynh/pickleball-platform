package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

func mustClockTime(t *testing.T, hour, minute int) domain.ClockTime {
	t.Helper()
	ct, err := domain.NewClockTime(hour, minute)
	if err != nil {
		t.Fatalf("bad fixture clock time: %v", err)
	}
	return ct
}

func TestNewRecurringHireTemplate_Validation(t *testing.T) {
	t.Parallel()

	start := mustClockTime(t, 9, 0)
	end := mustClockTime(t, 10, 0)
	startsAt := mustTime(t, "2026-08-03T00:00:00Z") // a Monday

	tests := []struct {
		name              string
		requestedByUserID string
		courtID           string
		startTime         domain.ClockTime
		endTime           domain.ClockTime
		wantErr           error
	}{
		{"valid template", "user-1", "court-1", start, end, nil},
		{"empty requested-by user id is rejected", "", "court-1", start, end, domain.ErrEmptyRequestedByUserID},
		{"empty court id is rejected", "user-1", "", start, end, domain.ErrEmptyCourtID},
		{"start time equal to end time is rejected", "user-1", "court-1", start, start, domain.ErrInvalidRecurringHireTimeRange},
		{"start time after end time is rejected", "user-1", "court-1", end, start, domain.ErrInvalidRecurringHireTimeRange},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := domain.NewRecurringHireTemplate(
				"tmpl-1", tt.requestedByUserID, tt.courtID, time.Monday,
				tt.startTime, tt.endTime, startsAt, domain.NoRecurringHireEnd(),
			)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tmpl.Status != domain.RecurringHireStatusRequested {
					t.Fatalf("new template status = %q, want %q", tmpl.Status, domain.RecurringHireStatusRequested)
				}
				if tmpl.ID != "tmpl-1" {
					t.Fatalf("template ID = %q, want %q", tmpl.ID, "tmpl-1")
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEndRecurringHireAfterOccurrences_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		n       int
		wantErr error
	}{
		{"positive count is valid", 10, nil},
		{"one is valid", 1, nil},
		{"zero is rejected", 0, domain.ErrInvalidRecurringHireEndAfterOccurrences},
		{"negative is rejected", -1, domain.ErrInvalidRecurringHireEndAfterOccurrences},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cond, err := domain.EndRecurringHireAfterOccurrences(tt.n)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if cond.Kind != domain.RecurringHireEndAfterOccurrences {
					t.Fatalf("Kind = %q, want %q", cond.Kind, domain.RecurringHireEndAfterOccurrences)
				}
				if cond.Occurrences != tt.n {
					t.Fatalf("Occurrences = %d, want %d", cond.Occurrences, tt.n)
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecurringHireTemplate_ApproveReject_Transitions(t *testing.T) {
	t.Parallel()

	newRequested := func(t *testing.T) domain.RecurringHireTemplate {
		t.Helper()
		tmpl, err := domain.NewRecurringHireTemplate(
			"tmpl-1", "user-1", "court-1", time.Monday,
			mustClockTime(t, 9, 0), mustClockTime(t, 10, 0),
			mustTime(t, "2026-08-03T00:00:00Z"), domain.NoRecurringHireEnd(),
		)
		if err != nil {
			t.Fatalf("bad fixture template: %v", err)
		}
		return tmpl
	}

	t.Run("requested -> approved is legal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Approve(); err != nil {
			t.Fatalf("Approve() unexpected error: %v", err)
		}
		if tmpl.Status != domain.RecurringHireStatusApproved {
			t.Fatalf("Status = %q, want %q", tmpl.Status, domain.RecurringHireStatusApproved)
		}
	})

	t.Run("requested -> rejected is legal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Reject(); err != nil {
			t.Fatalf("Reject() unexpected error: %v", err)
		}
		if tmpl.Status != domain.RecurringHireStatusRejected {
			t.Fatalf("Status = %q, want %q", tmpl.Status, domain.RecurringHireStatusRejected)
		}
	})

	t.Run("re-approving an already-approved template is illegal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Approve(); err != nil {
			t.Fatalf("bad fixture: %v", err)
		}
		if err := tmpl.Approve(); !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
			t.Fatalf("err = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
		}
	})

	t.Run("rejecting an already-rejected template is illegal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Reject(); err != nil {
			t.Fatalf("bad fixture: %v", err)
		}
		if err := tmpl.Reject(); !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
			t.Fatalf("err = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
		}
	})

	t.Run("approving an already-rejected template is illegal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Reject(); err != nil {
			t.Fatalf("bad fixture: %v", err)
		}
		if err := tmpl.Approve(); !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
			t.Fatalf("err = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
		}
	})

	t.Run("rejecting an already-approved template is illegal", func(t *testing.T) {
		t.Parallel()
		tmpl := newRequested(t)
		if err := tmpl.Approve(); err != nil {
			t.Fatalf("bad fixture: %v", err)
		}
		if err := tmpl.Reject(); !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
			t.Fatalf("err = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
		}
	})
}

func TestGenerateOccurrences(t *testing.T) {
	t.Parallel()

	newTemplate := func(t *testing.T, startsAt string, endCondition domain.RecurringHireEndCondition) domain.RecurringHireTemplate {
		t.Helper()
		tmpl, err := domain.NewRecurringHireTemplate(
			"tmpl-1", "user-1", "court-1", time.Monday,
			mustClockTime(t, 9, 0), mustClockTime(t, 10, 0),
			mustTime(t, startsAt), endCondition,
		)
		if err != nil {
			t.Fatalf("bad fixture template: %v", err)
		}
		return tmpl
	}

	t.Run("weekly occurrences starting from StartsAt, bounded by upTo", func(t *testing.T) {
		t.Parallel()
		// 2026-08-03 is a Monday.
		tmpl := newTemplate(t, "2026-08-03T00:00:00Z", domain.NoRecurringHireEnd())
		upTo := mustTime(t, "2026-08-24T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		want := []domain.TimeRange{
			mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			mustRange(t, "2026-08-10T09:00:00Z", "2026-08-10T10:00:00Z"),
			mustRange(t, "2026-08-17T09:00:00Z", "2026-08-17T10:00:00Z"),
		}
		assertRangesEqual(t, got, want)
	})

	t.Run("StartsAt not on Weekday advances to the first matching date", func(t *testing.T) {
		t.Parallel()
		// 2026-08-04 is a Tuesday; the template's Weekday is Monday, so the
		// first occurrence should be the following Monday, 2026-08-10.
		tmpl := newTemplate(t, "2026-08-04T00:00:00Z", domain.NoRecurringHireEnd())
		upTo := mustTime(t, "2026-08-11T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		want := []domain.TimeRange{
			mustRange(t, "2026-08-10T09:00:00Z", "2026-08-10T10:00:00Z"),
		}
		assertRangesEqual(t, got, want)
	})

	t.Run("NoEnd is bounded only by upTo, never unbounded", func(t *testing.T) {
		t.Parallel()
		tmpl := newTemplate(t, "2026-08-03T00:00:00Z", domain.NoRecurringHireEnd())
		// A far-future cap: this must still return a finite, bounded slice.
		upTo := mustTime(t, "2027-08-03T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		if len(got) == 0 {
			t.Fatalf("expected at least one occurrence")
		}
		for _, r := range got {
			if !r.Start.Before(upTo) {
				t.Fatalf("occurrence %v starts at/after upTo %v", r, upTo)
			}
		}
		// 52 or 53 weeks in a year.
		if len(got) < 52 || len(got) > 53 {
			t.Fatalf("len(got) = %d, want ~52-53", len(got))
		}
	})

	t.Run("EndAfterOccurrences stops after n occurrences even if upTo allows more", func(t *testing.T) {
		t.Parallel()
		cond, err := domain.EndRecurringHireAfterOccurrences(2)
		if err != nil {
			t.Fatalf("bad fixture end condition: %v", err)
		}
		tmpl := newTemplate(t, "2026-08-03T00:00:00Z", cond)
		upTo := mustTime(t, "2027-08-03T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		want := []domain.TimeRange{
			mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			mustRange(t, "2026-08-10T09:00:00Z", "2026-08-10T10:00:00Z"),
		}
		assertRangesEqual(t, got, want)
	})

	t.Run("EndAfterDate stops generating occurrences starting after that date, inclusive of the cutoff day", func(t *testing.T) {
		t.Parallel()
		endDate := mustTime(t, "2026-08-10T00:00:00Z")
		tmpl := newTemplate(t, "2026-08-03T00:00:00Z", domain.EndRecurringHireAfterDate(endDate))
		upTo := mustTime(t, "2027-08-03T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		want := []domain.TimeRange{
			mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			mustRange(t, "2026-08-10T09:00:00Z", "2026-08-10T10:00:00Z"),
		}
		assertRangesEqual(t, got, want)
	})

	t.Run("upTo before StartsAt yields no occurrences", func(t *testing.T) {
		t.Parallel()
		tmpl := newTemplate(t, "2026-08-03T00:00:00Z", domain.NoRecurringHireEnd())
		upTo := mustTime(t, "2026-08-01T00:00:00Z")

		got := domain.GenerateOccurrences(tmpl, upTo)

		if len(got) != 0 {
			t.Fatalf("len(got) = %d, want 0", len(got))
		}
	})
}

func assertRangesEqual(t *testing.T, got, want []domain.TimeRange) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if !got[i].Start.Equal(want[i].Start) || !got[i].End.Equal(want[i].End) {
			t.Fatalf("occurrence %d = %v, want %v", i, got[i], want[i])
		}
	}
}
