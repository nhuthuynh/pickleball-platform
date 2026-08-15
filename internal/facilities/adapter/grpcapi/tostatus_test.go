package grpcapi

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestToStatus_ForeignKeyTranslationPathMapsToNotFound is T17.4's precise
// answer to its own instruction 2 ("confirm the mapping is genuinely
// exercised for the FK case too rather than assuming the existing entry
// already covers it").
//
// **Why this file exists, in-package, alongside foreign_key_error_mapping_
// test.go rather than instead of it.** foreign_key_error_mapping_test.go
// drives AddCourt/AddCameraLink through the real Handler -> app.Service with
// an unknown FacilityID, and a mutation check run for this ticket (remove
// fakeRepo.AddCourt's newly-added FK modeling, rerun) proved something worth
// recording plainly rather than glossing over: that test still passed with
// the mutation in place. The reason is structural, not a test bug —
// app.Service.AddCourt/AddCameraLink both call GetFacilityByID as their own
// object-level-ownership guard BEFORE ever reaching Repository.AddCourt/
// AddCameraLink (see app/service.go), so for an unknown FacilityID the guard
// always answers ErrFacilityNotFound first. No fake-level test driven
// through app.Service can therefore isolate "the sentinel that comes
// specifically from a translated 23503" from "the sentinel that comes from
// the pre-existing, already-correct guard" — both are the identical
// domain.ErrFacilityNotFound value, indistinguishable once returned. That
// is exactly this ticket's own instruction 2 warning, discovered by doing
// the mutation check it asks for rather than skipping it because the first
// attempt looked plausible (CLAUDE.md rule 10).
//
// This file is what actually closes that gap: it calls the package's own
// toStatus directly with domain.ErrFacilityNotFound — the exact value
// adapter/postgres.translateErr's new 23503 arm returns (pinned Docker-free
// by adapter/postgres/foreign_key_test.go) — with no guard in front of it to
// confound the result. Combined with
// adapter/postgres/facility_deleted_race_integration_test.go (proves a real
// Postgres genuinely raises 23503 here and that translateErr genuinely
// translates it), the chain from a real FK violation to a real NotFound
// response is now proved end-to-end without any single test having to fake
// the whole stack. foreign_key_error_mapping_test.go stays: it is still a
// genuine, previously-missing RPC-level regression test for "an unknown
// Facility answers NotFound, never Internal" through AddCourt/AddCameraLink,
// its doc comment now says so accurately.
func TestToStatus_ForeignKeyTranslationPathMapsToNotFound(t *testing.T) {
	t.Parallel()

	err := toStatus(domain.ErrFacilityNotFound)

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("toStatus(ErrFacilityNotFound) = %v, not a gRPC status error", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("toStatus(ErrFacilityNotFound) = Internal (500-shaped) — issue #195's exact defect: " +
			"a legitimate 'no such Facility' condition, whether from the app-level guard or from a " +
			"translated 23503 race, must never read as a server bug")
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("toStatus(ErrFacilityNotFound) code = %v, want NotFound", st.Code())
	}
}
