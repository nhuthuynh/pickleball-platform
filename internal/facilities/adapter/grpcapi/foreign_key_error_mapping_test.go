// T17.4 (issue #195) — an RPC-level regression test for AddCourt and
// AddCameraLink against an unknown Facility, which no test in this package
// covered before this ticket (grep found none asserting codes.NotFound
// through either RPC), driven through the real Handler -> app.Service ->
// port.Repository path with the fakeRepo authz_regression_test.go already
// defines.
//
// **What this file proves, precisely, after a mutation check corrected the
// first draft's assumption.** Both RPCs answer codes.NotFound, never
// Internal, for an unknown FacilityID — a real, valuable, previously-missing
// guarantee. It does NOT isolate the FK-translation arm this ticket added:
// app.Service.AddCourt/AddCameraLink call GetFacilityByID as their own
// object-level-ownership guard before ever reaching Repository.AddCourt/
// AddCameraLink (app/service.go), so for an unknown id that guard's own
// domain.ErrFacilityNotFound always wins here, exactly as it did before this
// ticket — a mutation check (temporarily make fakeRepo.AddCourt model
// courts.facility_id's FK too, rerun) confirmed the test below stayed green
// either way, because the guard catches the case first regardless of what
// AddCourt itself would have done. fakeRepo.AddCourt is deliberately left
// NOT modeling that FK, with its own comment explaining why: every path to
// it goes through the guard first, so a mutation check proved no test can
// ever demand that check fail, and adding it anyway would be exactly the
// "no [code] without a test that demanded it" gap CLAUDE.md rule 1 names.
// This file cannot, therefore, be the proof that toStatus's mapping is
// exercised specifically "for the FK case" — this ticket's own
// instruction 2's phrase.
//
// tostatus_test.go (in-package, alongside handler.go) is what actually
// supplies that proof: it calls toStatus directly with exactly the value
// adapter/postgres.translateErr's new 23503 arm returns, with no guard able
// to intercept first. adapter/postgres/facility_deleted_race_integration_
// test.go supplies the other half — that a real Postgres genuinely raises
// 23503 here and translateErr genuinely translates it. The three files
// together are the full chain from a real FK violation to a real NotFound
// response; this file's own job, honestly stated, is the RPC-level
// regression guarantee alone.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
)

// unknownFacilityID is uuid-shaped and seeded into no fakeRepo fixture, the
// same convention internal/booking/adapter/postgres/
// unknown_court_integration_test.go's unknownCourtID documents.
const unknownFacilityID = "11111111-1111-1111-1111-1111111111ff"

// TestAddCourt_UnknownFacilityIsNotFoundNotInternal drives AddCourt, through
// the real Handler -> app.Service -> port.Repository path, with a
// well-formed FacilityID naming no Facility, and asserts the mapped status
// is NotFound — never Internal, which is exactly what issue #195 describes
// for the un-translated case (and what #185 already fixed for
// bookings.court_id).
func TestAddCourt_UnknownFacilityIsNotFoundNotInternal(t *testing.T) {
	h, repo := newTestHandler()

	_, err := h.AddCourt(ctxAs(ownerSubject), &facilitiesv1.AddCourtRequest{
		FacilityId: unknownFacilityID,
		Name:       "Court 1",
	})
	if err == nil {
		t.Fatal("AddCourt(unknown facility) succeeded silently — want a NotFound error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("AddCourt(unknown facility) returned a non-gRPC-status error: %v", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("AddCourt(unknown facility) mapped to Internal (500-shaped) — issue #195's defect: "+
			"a legitimate 'no such Facility' condition must not read as a server bug: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("AddCourt(unknown facility) status code = %v, want NotFound", st.Code())
	}

	if len(repo.courts) != 0 {
		t.Errorf("repo.courts has %d entries after a rejected AddCourt, want 0", len(repo.courts))
	}
}

// TestAddCameraLink_UnknownFacilityIsNotFoundNotInternal mirrors the above
// for AddCameraLink. Its fake (fakeRepo.AddCameraLink) does model
// facility_camera_links.facility_id's FK directly, unlike AddCourt's — but
// per this file's own package comment, that distinction is moot for what
// this particular test proves, since the app-level guard answers first for
// both RPCs either way.
func TestAddCameraLink_UnknownFacilityIsNotFoundNotInternal(t *testing.T) {
	h, _ := newTestHandler()

	_, err := h.AddCameraLink(ctxAs(ownerSubject), &facilitiesv1.AddCameraLinkRequest{
		FacilityId: unknownFacilityID,
		Url:        "https://example.com/cam1.m3u8",
	})
	if err == nil {
		t.Fatal("AddCameraLink(unknown facility) succeeded silently — want a NotFound error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("AddCameraLink(unknown facility) returned a non-gRPC-status error: %v", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("AddCameraLink(unknown facility) mapped to Internal (500-shaped) — issue #195's defect: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("AddCameraLink(unknown facility) status code = %v, want NotFound", st.Code())
	}
}
