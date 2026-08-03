package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// TestTranslateErr proves the DB-level half of CLAUDE.md rule 4/5: a 23505
// unique_violation on payments_payable_unique_idx (db/migrations/
// 0005_payments.sql) must translate to domain.ErrPaymentAlreadyRecorded, and
// pgx.ErrNoRows must translate to domain.ErrPaymentNotFound — neither may
// leak as a raw pgconn.PgError. Runs without internal/gen or a real
// database, mirroring internal/booking/adapter/postgres's translateErr
// test coverage (see docs/reviews and the T5.5 socialplay translate_test.go
// pattern on the T5 branches).
func TestTranslateErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"unique violation (23505) becomes ErrPaymentAlreadyRecorded", &pgconn.PgError{Code: "23505"}, domain.ErrPaymentAlreadyRecorded},
		{"no rows becomes ErrPaymentNotFound", pgx.ErrNoRows, domain.ErrPaymentNotFound},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrPaymentAlreadyRecorded) || errors.Is(got, domain.ErrPaymentNotFound) {
				t.Fatalf("translateErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestMustUUID_PanicsOnMalformedInput proves mustUUID treats a malformed ID
// as the programmer error it documents, mirroring
// internal/booking/adapter/postgres's identical helper.
func TestMustUUID_PanicsOnMalformedInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected mustUUID to panic on a malformed uuid")
		}
	}()
	mustUUID("not-a-uuid")
}
