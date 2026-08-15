// T8.2 — proves the real read path for a Facility's Courts through the
// full grpcapi.Handler (not just the app-level test
// internal/facilities/app/service_test.go already has): GetFacility's
// response carries the Courts that were added via AddCourt, and a Facility
// with no Courts yet gets back an empty (not nil-panicking) list — the
// wiring FacilityDetailPanel.vue depends on to stop always rendering the
// zero-courts empty state.
package grpcapi_test

import (
	"context"
	"testing"

	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
)

func TestGetFacility_ReturnsCourtsAddedViaAddCourt(t *testing.T) {
	// GetFacility stays public (see PublicMethods) but AddCourt does not, so
	// the two writes below authenticate as the owner while the read is made
	// with a bare context — which also keeps this test honest about the
	// browse path remaining callable without a token.
	ctx := context.Background()
	ownerCtx := ctxAs(ownerSubject)
	h, _ := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	addResp1, err := h.AddCourt(ownerCtx, &facilitiesv1.AddCourtRequest{
		FacilityId: facility.GetId(),
		Name:       "Court 1",
	})
	if err != nil {
		t.Fatalf("unexpected err adding Court 1: %v", err)
	}
	addResp2, err := h.AddCourt(ownerCtx, &facilitiesv1.AddCourtRequest{
		FacilityId: facility.GetId(),
		Name:       "Court 2",
	})
	if err != nil {
		t.Fatalf("unexpected err adding Court 2: %v", err)
	}

	getResp, err := h.GetFacility(ctx, &facilitiesv1.GetFacilityRequest{FacilityId: facility.GetId()})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	courts := getResp.GetCourts()
	if len(courts) != 2 {
		t.Fatalf("Courts = %v, want 2 entries", courts)
	}
	if courts[0].GetId() != addResp1.GetCourt().GetId() || courts[0].GetName() != "Court 1" {
		t.Errorf("Courts[0] = %v, want Court 1 (%s)", courts[0], addResp1.GetCourt().GetId())
	}
	if courts[1].GetId() != addResp2.GetCourt().GetId() || courts[1].GetName() != "Court 2" {
		t.Errorf("Courts[1] = %v, want Court 2 (%s)", courts[1], addResp2.GetCourt().GetId())
	}
	for _, c := range courts {
		if c.GetFacilityId() != facility.GetId() {
			t.Errorf("Court %v FacilityId = %q, want %q", c, c.GetFacilityId(), facility.GetId())
		}
	}
}

func TestGetFacility_ReturnsEmptyCourtsForFacilityWithNone(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	getResp, err := h.GetFacility(ctx, &facilitiesv1.GetFacilityRequest{FacilityId: facility.GetId()})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if got := getResp.GetCourts(); len(got) != 0 {
		t.Fatalf("Courts = %v, want empty for a facility with no courts added", got)
	}
}
