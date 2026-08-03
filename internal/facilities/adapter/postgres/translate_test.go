package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestTranslateErr proves the adapter boundary CLAUDE.md rule 5 requires: a
// Postgres pgx.ErrNoRows on a facility lookup must translate to
// domain.ErrFacilityNotFound, and any other error must be wrapped rather
// than silently mismapped to that sentinel. Mirrors
// internal/socialplay/adapter/postgres/translate_test.go's
// TestTranslateGameErr. Runs without internal/gen or a real database.
func TestTranslateErr(t *testing.T) {
	t.Parallel()

	if got := translateErr(pgx.ErrNoRows); !errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateErr(pgx.ErrNoRows) = %v, want ErrFacilityNotFound", got)
	}

	unrelated := errors.New("boom")
	got := translateErr(unrelated)
	if errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateErr(%v) = %v, an unrelated error must not be mismapped to ErrFacilityNotFound", unrelated, got)
	}
	if got == nil {
		t.Fatal("translateErr(unrelated) = nil, want a wrapped error")
	}
}

// TestMustUUID_PanicsOnMalformedInput proves mustUUID treats a malformed ID
// as the programmer error it documents (upstream invariant already
// violated), mirroring internal/booking/adapter/postgres's identical
// helper.
func TestMustUUID_PanicsOnMalformedInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected mustUUID to panic on a malformed uuid")
		}
	}()
	mustUUID("not-a-uuid")
}

// TestUUIDOrEmpty proves the nullable-UUID convention this adapter relies
// on for courts.facility_id (pre-Facilities seeded courts) and
// facility_camera_links.court_id (facility-wide links): an invalid/unset
// pgtype.UUID becomes an empty string, matching domain.Court/
// domain.CameraLink's plain-string-id convention, and a valid one round-trips.
func TestUUIDOrEmpty(t *testing.T) {
	t.Parallel()

	var invalid pgtype.UUID // zero value: Valid=false, e.g. a NULL facility_id/court_id column
	if got := uuidOrEmpty(invalid); got != "" {
		t.Fatalf("uuidOrEmpty(invalid) = %q, want empty string", got)
	}

	const id = "11111111-1111-1111-1111-111111111111"
	if got := uuidOrEmpty(mustUUID(id)); got != id {
		t.Fatalf("uuidOrEmpty(valid) = %q, want %q", got, id)
	}
}
