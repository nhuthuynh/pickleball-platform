package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestNewFacility_Valid proves the happy path: a valid NewFacility call
// produces a Facility carrying exactly what was passed in, with
// CameraConsentAttested defaulting to false and CameraLinks empty.
func TestNewFacility_Valid(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "Nice courts by the river", "123 Main St", []string{"https://example.com/photo1.jpg"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if f.ID != "f1" {
		t.Fatalf("ID = %q, want f1", f.ID)
	}
	if f.OwnerID != "owner-1" {
		t.Fatalf("OwnerID = %q, want owner-1", f.OwnerID)
	}
	if f.Name != "Riverside Courts" {
		t.Fatalf("Name = %q, want Riverside Courts", f.Name)
	}
	if f.Description != "Nice courts by the river" {
		t.Fatalf("Description = %q, want Nice courts by the river", f.Description)
	}
	if f.Address != "123 Main St" {
		t.Fatalf("Address = %q, want 123 Main St", f.Address)
	}
	if len(f.PhotoURLs) != 1 || f.PhotoURLs[0] != "https://example.com/photo1.jpg" {
		t.Fatalf("PhotoURLs = %v, want [https://example.com/photo1.jpg]", f.PhotoURLs)
	}
	// CameraConsentAttested must default to false: the round-10 design
	// review's finding was specifically that the consent checkbox must
	// default to unchecked, not just be checkable. A domain default of
	// true would silently defeat that finding.
	if f.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = true, want false by default")
	}
	if len(f.CameraLinks) != 0 {
		t.Fatalf("CameraLinks = %v, want empty", f.CameraLinks)
	}
}

// TestNewFacility_Validation is the required boundary coverage: empty
// OwnerID, Name, and Address must each be rejected via the single
// ErrEmptyFacilityField sentinel, with a *domain.FieldError identifying
// which field failed — mirroring payments.ErrInvalidAmount's
// single-sentinel-plus-context pattern rather than three stringly-typed
// sentinels.
func TestNewFacility_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ownerID   string
		fname     string
		address   string
		wantErr   error
		wantField string
	}{
		{
			name:    "valid facility",
			ownerID: "owner-1",
			fname:   "Riverside Courts",
			address: "123 Main St",
			wantErr: nil,
		},
		{
			name:      "empty owner id is rejected",
			ownerID:   "",
			fname:     "Riverside Courts",
			address:   "123 Main St",
			wantErr:   domain.ErrEmptyFacilityField,
			wantField: "OwnerID",
		},
		{
			name:      "empty name is rejected",
			ownerID:   "owner-1",
			fname:     "",
			address:   "123 Main St",
			wantErr:   domain.ErrEmptyFacilityField,
			wantField: "Name",
		},
		{
			name:      "empty address is rejected",
			ownerID:   "owner-1",
			fname:     "Riverside Courts",
			address:   "",
			wantErr:   domain.ErrEmptyFacilityField,
			wantField: "Address",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewFacility("f1", tt.ownerID, tt.fname, "", tt.address, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				var fieldErr *domain.FieldError
				if !errors.As(err, &fieldErr) {
					t.Fatalf("err %v is not a *domain.FieldError", err)
				}
				if fieldErr.Field != tt.wantField {
					t.Fatalf("Field = %q, want %q", fieldErr.Field, tt.wantField)
				}
			}
		})
	}
}

// TestFacility_AddCameraLink_RequiresConsent is the explicitly-required
// test for this ticket's one concrete finding (round-10 review §2b): a
// camera link may only be added once CameraConsentAttested is true. This
// is the test that would fail if that check were subtly removed or
// inverted — it asserts both the rejection *and* that CameraLinks is left
// untouched by the rejected call, then proves the same call succeeds once
// consent is attested.
func TestFacility_AddCameraLink_RequiresConsent(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	if f.CameraConsentAttested {
		t.Fatalf("fixture CameraConsentAttested = true, want false")
	}

	err = f.AddCameraLink("owner-1", "https://example.com/cam1.m3u8")
	if !errors.Is(err, domain.ErrCameraConsentRequired) {
		t.Fatalf("got err %v, want %v", err, domain.ErrCameraConsentRequired)
	}
	if len(f.CameraLinks) != 0 {
		t.Fatalf("CameraLinks = %v, want empty after rejected AddCameraLink", f.CameraLinks)
	}

	f.CameraConsentAttested = true
	if err := f.AddCameraLink("owner-1", "https://example.com/cam1.m3u8"); err != nil {
		t.Fatalf("unexpected err after consent attested: %v", err)
	}
	if len(f.CameraLinks) != 1 || f.CameraLinks[0].URL != "https://example.com/cam1.m3u8" {
		t.Fatalf("CameraLinks = %v, want one link with the added URL", f.CameraLinks)
	}
	if f.CameraLinks[0].CourtID != "" {
		t.Fatalf("CourtID = %q, want empty (facility-wide link)", f.CameraLinks[0].CourtID)
	}
}

// TestFacility_AddCameraLink_MultipleLinksAppend proves repeated calls
// append rather than overwrite, once consent is attested.
func TestFacility_AddCameraLink_MultipleLinksAppend(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	f.CameraConsentAttested = true

	if err := f.AddCameraLink("owner-1", "https://example.com/cam1.m3u8"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := f.AddCameraLink("owner-1", "https://example.com/cam2.m3u8"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(f.CameraLinks) != 2 {
		t.Fatalf("CameraLinks len = %d, want 2", len(f.CameraLinks))
	}
}

// --- T7.7: object-level (BOLA) ownership checks ----------------------------

// TestFacility_EnsureOwner proves the ownership predicate itself: the
// Facility's own OwnerID is accepted, any other actor (including an empty
// one) is rejected with ErrNotFacilityOwner.
func TestFacility_EnsureOwner(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := f.EnsureOwner("owner-1"); err != nil {
		t.Fatalf("EnsureOwner(owner) = %v, want nil", err)
	}
	if err := f.EnsureOwner("someone-else"); !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("EnsureOwner(non-owner) = %v, want %v", err, domain.ErrNotFacilityOwner)
	}
	if err := f.EnsureOwner(""); !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("EnsureOwner(empty actor) = %v, want %v", err, domain.ErrNotFacilityOwner)
	}
}

// --- T8.4: AttestCameraConsent --------------------------------------------

// TestFacility_AttestCameraConsent_Owner proves the happy path: the
// Facility's own owner attesting consent sets CameraConsentAttested to
// true, and that a camera link can then be added — closing the T7-flagged
// gap that nothing ever set this field server-side.
func TestFacility_AttestCameraConsent_Owner(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	if f.CameraConsentAttested {
		t.Fatalf("fixture CameraConsentAttested = true, want false")
	}

	if err := f.AttestCameraConsent("owner-1"); err != nil {
		t.Fatalf("AttestCameraConsent(owner) = %v, want nil", err)
	}
	if !f.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = false after AttestCameraConsent, want true")
	}

	// Now that consent is attested, AddCameraLink should actually succeed —
	// the end-to-end proof this ticket exists for.
	if err := f.AddCameraLink("owner-1", "https://example.com/cam1.m3u8"); err != nil {
		t.Fatalf("AddCameraLink after AttestCameraConsent should succeed, got: %v", err)
	}
}

// TestFacility_AttestCameraConsent_Idempotent proves attesting twice is not
// an error — re-attesting an already-attested Facility just confirms the
// same state, per the ticket's explicit requirement.
func TestFacility_AttestCameraConsent_Idempotent(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := f.AttestCameraConsent("owner-1"); err != nil {
		t.Fatalf("first AttestCameraConsent = %v, want nil", err)
	}
	if err := f.AttestCameraConsent("owner-1"); err != nil {
		t.Fatalf("second AttestCameraConsent = %v, want nil (idempotent)", err)
	}
	if !f.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = false after idempotent re-attest, want true")
	}
}

// TestFacility_AttestCameraConsent_RejectsNonOwner proves the ownership
// check runs before consent is ever touched: a non-owner is rejected with
// ErrNotFacilityOwner (mapped to 403 by grpcapi) and CameraConsentAttested
// is left untouched — the non-owner never even learns the Facility's
// current consent state, mirroring AddCameraLink's/AddCourt's check
// ordering (T7.7).
func TestFacility_AttestCameraConsent_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	err = f.AttestCameraConsent("attacker")
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotFacilityOwner)
	}
	if f.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = true after rejected non-owner attest, want false (untouched)")
	}
}

// TestFacility_AddCameraLink_RejectsNonOwner is the domain-level proof that
// AddCameraLink checks ownership before it checks camera consent: a
// non-owner is rejected with ErrNotFacilityOwner even when
// CameraConsentAttested is already true (so the two checks can't be
// confused with each other), and CameraLinks is left untouched.
func TestFacility_AddCameraLink_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	f, err := domain.NewFacility("f1", "owner-1", "Riverside Courts", "", "123 Main St", nil)
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	f.CameraConsentAttested = true

	err = f.AddCameraLink("attacker", "https://example.com/cam1.m3u8")
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotFacilityOwner)
	}
	if len(f.CameraLinks) != 0 {
		t.Fatalf("CameraLinks = %v, want empty after rejected AddCameraLink", f.CameraLinks)
	}
}
