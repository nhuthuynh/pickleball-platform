package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestIsRetryableConflict is the regression test for the bug found by
// independently reproducing HANDOFF.md T4's concurrency claim: a real
// concurrent burst against local Postgres produced 40P01 deadlock errors
// that translateErr wrapped as generic errors instead of retrying — see
// docs/LESSONS.md. It does not need internal/gen or a real database, so it
// runs even in this environment.
func TestIsRetryableConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"deadlock detected (40P01) is retryable", &pgconn.PgError{Code: "40P01"}, true},
		{"serialization failure (40001) is retryable", &pgconn.PgError{Code: "40001"}, true},
		{"exclusion violation (23P01) is a real conflict, not retryable", &pgconn.PgError{Code: "23P01"}, false},
		{"unrelated pg error is not retryable", &pgconn.PgError{Code: "42601"}, false},
		{"non-pg error is not retryable", errors.New("boom"), false},
		{"nil error is not retryable", nil, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRetryableConflict(tt.err); got != tt.want {
				t.Errorf("isRetryableConflict(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
