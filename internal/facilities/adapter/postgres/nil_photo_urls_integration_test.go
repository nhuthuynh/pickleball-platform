//go:build integration

package postgres_test

import (
	"context"
	"testing"

	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// T61 — a Facility created with NO photos must persist, and this is a
// PRODUCTION defect rather than a fixture problem.
//
// # The defect
//
// `facilities.photo_urls` is `text[] NOT NULL DEFAULT '{}'`
// (db/migrations/0010_facilities.sql). A column DEFAULT applies only when the
// column is OMITTED from the INSERT — it does nothing for an explicit NULL.
// CreateFacility's query names photo_urls explicitly, so a nil slice is sent
// as NULL and the insert is rejected:
//
//	ERROR: null value in column "photo_urls" of relation "facilities"
//	violates not-null constraint (SQLSTATE 23502)
//
// # Why a real client hits it
//
// Checked rather than assumed: `CreateFacilityRequest.GetPhotoUrls()` on an
// unset `repeated string` returns `[]string(nil)` — verified by probe. A
// client creating a Facility without photos, which is the ordinary case,
// sends nothing; the handler passes that nil straight through
// `domain.NewFacility` (whose `PhotoURLs []string` correctly treats nil as
// "no photos", good Go) and into this adapter.
//
// So every photo-less Facility creation fails against real Postgres with a
// raw constraint error. It was invisible because no gate runnable without a
// Docker daemon executes these tests, and `make vet-integration` only
// COMPILES them — a NOT NULL violation is not a compile error.
//
// # Where the fix belongs
//
// In the adapter, not the domain. nil meaning "no photos" is idiomatic Go and
// the domain is right to allow it; the column's representation of "none" is
// `'{}'`, and translating between a context's own types and its storage
// shape is exactly what CLAUDE.md rule 5 puts in the adapter.
func TestCreateFacility_NilPhotoURLsPersistsAsEmpty(t *testing.T) {
	ctx := context.Background()
	pool := startFacilitiesPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	// nil, exactly as the gRPC handler passes it for an unset photo_urls.
	f, err := domain.NewFacility("00000000-0000-4000-a000-00000000f001",
		"00000000-0000-4000-b000-00000000f001", "No Photos Club", "", "1 Nil Way", nil)
	if err != nil {
		t.Fatalf("NewFacility: %v", err)
	}
	if f.PhotoURLs != nil {
		t.Fatalf("fixture precondition: PhotoURLs = %#v, want nil — this test is about the nil case", f.PhotoURLs)
	}

	created, err := repo.CreateFacility(ctx, f)
	if err != nil {
		t.Fatalf("creating a Facility with no photos failed: %v", err)
	}

	// Round-trips as empty rather than null, so a reader never has to
	// distinguish "no photos" from "unknown".
	if created.PhotoURLs == nil {
		t.Error("CreateFacility returned nil PhotoURLs; want a non-nil empty slice, so the absence is a fact rather than a gap")
	}
	if len(created.PhotoURLs) != 0 {
		t.Errorf("PhotoURLs = %#v, want empty", created.PhotoURLs)
	}

	got, err := repo.GetFacilityByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetFacilityByID: %v", err)
	}
	if len(got.PhotoURLs) != 0 {
		t.Errorf("re-read PhotoURLs = %#v, want empty", got.PhotoURLs)
	}
}
