package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestNewCourt_Valid proves the happy path: a valid NewCourt call produces
// a Court carrying exactly what was passed in.
func TestNewCourt_Valid(t *testing.T) {
	t.Parallel()

	c, err := domain.NewCourt("c1", "f1", "Court 1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.ID != "c1" {
		t.Fatalf("ID = %q, want c1", c.ID)
	}
	if c.FacilityID != "f1" {
		t.Fatalf("FacilityID = %q, want f1", c.FacilityID)
	}
	if c.Name != "Court 1" {
		t.Fatalf("Name = %q, want Court 1", c.Name)
	}
}

// TestNewCourt_Validation is the required boundary coverage: empty
// FacilityID and Name must each be rejected via the single
// ErrEmptyCourtField sentinel, with a *domain.FieldError identifying which
// field failed.
func TestNewCourt_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		facilityID string
		courtName  string
		wantErr    error
		wantField  string
	}{
		{
			name:       "valid court",
			facilityID: "f1",
			courtName:  "Court 1",
			wantErr:    nil,
		},
		{
			name:       "empty facility id is rejected",
			facilityID: "",
			courtName:  "Court 1",
			wantErr:    domain.ErrEmptyCourtField,
			wantField:  "FacilityID",
		},
		{
			name:       "empty name is rejected",
			facilityID: "f1",
			courtName:  "",
			wantErr:    domain.ErrEmptyCourtField,
			wantField:  "Name",
		},
		{
			name:       "both facility id and name empty is rejected on facility id first",
			facilityID: "",
			courtName:  "",
			wantErr:    domain.ErrEmptyCourtField,
			wantField:  "FacilityID",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewCourt("c1", tt.facilityID, tt.courtName)
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
