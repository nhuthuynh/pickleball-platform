package postgres

import (
	"strings"
	"testing"
)

// T13.3 / issue #154 — the mechanism, pinned Docker-free.
//
// This file exists because ticket instruction 1 asks for the panic to be
// *reproduced* rather than reasoned about. It is deliberately an in-package
// test (`package postgres`, like translate_test.go beside it): mustUUID is
// unexported, and the whole point is to hit it directly rather than through a
// Postgres round trip that this environment has no Docker daemon for.
//
// **What this test is and is not.** It is not the regression test for #154 —
// it passes both before and after the fix, because mustUUID's behaviour is
// correct and does not change. It is the *evidence* for why the fix has to be
// a translation at the grpcapi boundary (ADR-0014 §1) and not something
// gentler further down: the value CreateFacility used to hand this adapter was
// a raw IdP subject, and a raw IdP subject reaching CreateFacility's
// `OwnerID: mustUUID(f.OwnerID)` does not return an error a client could act
// on. It takes the process down.
//
// The regression test — the one that fails before the fix and passes after —
// is TestCreateFacility_SubjectShapedOwnerResolvesEndToEnd in
// internal/facilities/adapter/grpcapi/subject_owner_seam_test.go. The two are
// a pair: that one proves the subject never gets here, this one proves what
// would happen if it did.
//
// It is also deliberately narrower than translate_test.go's
// TestMustUUID_PanicsOnMalformedInput beside it, rather than a duplicate of
// it. That test asserts the general contract with "not-a-uuid" and a bare
// recover(); this one asserts the *specific production values* #154 is about
// — the three subject shapes 0019 names — and checks the panic message, so a
// panic arriving from somewhere else entirely cannot satisfy it. The positive
// direction (a real uuid converts cleanly) is already pinned one file over by
// TestUUIDOrEmpty, which round-trips a valid id through mustUUID.
func TestMustUUID_PanicsOnAnIdPSubject(t *testing.T) {
	// The three shapes db/migrations/0019_identity_subject.sql names as real
	// subjects. None is a uuid; none ever will be. `sub` is the identity
	// provider's namespace, not ours.
	for _, subject := range []string{
		"auth0|abc123",
		"google-oauth2|10769150350006150715",
		"opaque-pairwise-identifier",
	} {
		t.Run(subject, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("mustUUID(%q) returned normally — this test encodes the #154 "+
						"mechanism, so if mustUUID has stopped panicking on a non-uuid, "+
						"ADR-0014's reasoning about this seam needs re-reading, not this "+
						"test deleting", subject)
				}
				// Assert on the message, not merely that *something* panicked:
				// a panic from an unrelated nil deref would otherwise satisfy
				// a bare recover() != nil and prove nothing.
				if msg, ok := r.(string); !ok || !strings.Contains(msg, "invalid uuid") {
					t.Fatalf("mustUUID(%q) panicked with %#v, want the adapter's own "+
						"\"invalid uuid\" message", subject, r)
				}
			}()

			mustUUID(subject)
		})
	}
}
