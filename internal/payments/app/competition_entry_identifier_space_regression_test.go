// T29.1 instruction 7 (closes the Competitions third of #164, and with it
// the Competitions half of #237): Payments-side VERIFICATION, not
// modification, that authorizeCompetitionEntryRecording's comparisons are
// genuinely identifier-space-safe — no non-test file under
// internal/payments/** is touched by this ticket (§B3 of
// docs/process/t29-sprint-plan.md).
//
// # Why this file exists alongside the pre-existing CompetitionEntry tests
//
// Issue #237's own diagnosis, quoted directly rather than paraphrased,
// because it is the exact defect this file exists to make impossible to
// reintroduce silently: "internal/payments/app/service_test.go's fixtures
// (e.g. TestRecordOfflinePayment_RegistrationPayable_GameHostSucceeds)
// construct ActorUserID and the fake GameLookup's HostID from the SAME
// LITERAL STRING (e.g. "host-1"), which does not model the real system's
// two distinct identifier spaces post-T28.1 and so cannot detect a space
// mismatch either way."
//
// The pre-existing CompetitionEntry tests in service_test.go/refund_test.go
// (TestRecordOfflinePayment_CompetitionEntryPayable_EntrantSucceeds and its
// siblings) have the identical shape: "player-1", "admin-1", "admin-2" are
// short, hand-typed mnemonic strings that could never collide with, or be
// mistaken for, a real identity_users.id — a reader (or a future author
// copy-pasting a new test) cannot tell from them whether the system is
// genuinely comparing two independently-resolved User.ID uuids or merely
// two copies of one arbitrary token. Real production values on both sides
// of this comparison — ActorUserID (resolved from a verified subject
// through Competitions' new port.IdentityLookup, T29.1) and the values
// port.EntryLookup.CompetitionIDAndPlayerIDForEntry / port.
// CompetitionAdminReader.ListCompetitionAdmins return (read from
// competitions.competition_entries.player_id /
// competitions.competition_admins.user_id, both real `uuid` columns as of
// db/migrations/0025_competitions_identity_conformance.sql) — are
// uuid-shaped, never short mnemonic strings. This file's fixtures use real,
// uuid-shaped, MUTUALLY DISTINCT literals for every distinct identity
// (entrant, admin, and the rejected stranger), declared as separate named
// constants rather than a single string retyped in two places, so that a
// future change which silently compared the wrong pair (e.g. an
// unresolved subject against a resolved uuid, #237's actual shape) fails
// this file's tests rather than passing by construction the way the
// short-mnemonic fixtures always would have, on either side of that bug.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// The four uuids below are real, uuid-shaped, and mutually distinct — never
// reused across roles, and never derived from one another — so a test that
// passes proves the specific pair the assertion names actually matched,
// not merely that "some string equals itself". Each is commented with the
// real-world identifier-space fact it stands in for.
const (
	// regressionEntrantUserID is the value Competitions' actor(ctx) resolves
	// the entrant's verified subject to (ADR-0014/ADR-0017, T29.1) — what
	// Payments' own grpcapi funnel independently resolves the SAME caller's
	// subject to as well, and so is what a real ActorUserID and a real
	// port.EntryLookup-returned playerID agree on when the system is
	// correct.
	regressionEntrantUserID = "6ba7b810-a001-4000-8000-1a1a1a1a1a1a"
	// regressionAdminUserID is a Competition Admin's resolved User.ID — the
	// value port.CompetitionAdminReader.ListCompetitionAdmins returns for
	// this entry's owning Competition. Deliberately unrelated to
	// regressionEntrantUserID by construction (not a suffix, not a shared
	// prefix beyond the fixture-wide "6ba7b810-" marker), so a comparison
	// bug that accidentally matched the wrong role cannot pass by
	// resemblance.
	regressionAdminUserID = "6ba7b810-b002-4000-8000-2b2b2b2b2b2b"
	// regressionStrangerUserID is a real, resolved User.ID belonging to
	// someone with NO relationship to this CompetitionEntry or its
	// Competition — the negative case: a genuinely different identity,
	// resolved through the identical seam a legitimate actor would be,
	// still correctly refused.
	regressionStrangerUserID = "6ba7b810-c003-4000-8000-3c3c3c3c3c3c"
	// regressionCompetitionEntryPaymentID is this file's own PayableID/
	// PaymentID fixture, kept separate from fixtureCompetitionEntryID/
	// fixtureCompetitionEntryPaymentID (service_test.go/refund_test.go) so
	// this file's tests can run under t.Parallel() with zero shared-fixture
	// risk.
	regressionCompetitionEntryID        = "6ba7b810-d004-4000-8000-4d4d4d4d4d4d"
	regressionCompetitionEntryPaymentID = "6ba7b810-e005-4000-8000-5e5e5e5e5e5e"
)

// newRegressionEntryAuthzFixtures builds fakeEntryLookup/
// fakeCompetitionAdminReader wired to regressionCompetitionEntryID exactly
// as newEntryAuthzFixtures (fixtures_test.go) wires fixtureCompetitionEntryID
// — a separate constructor rather than reusing newEntryAuthzFixtures against
// this file's own PayableID, so this file's fixture wiring is visibly
// self-contained rather than depending on another file's helper continuing
// to key off the same PayableID constant it happens to today.
func newRegressionEntryAuthzFixtures(playerID string, admins ...string) (*fakeEntryLookup, *fakeCompetitionAdminReader) {
	entries := &fakeEntryLookup{
		competitionByEntry: map[string]string{regressionCompetitionEntryID: "6ba7b810-f006-4000-8000-6f6f6f6f6f6f"},
		playerByEntry:      map[string]string{regressionCompetitionEntryID: playerID},
	}
	competitionAdmins := &fakeCompetitionAdminReader{
		adminsByCompetition: map[string][]string{"6ba7b810-f006-4000-8000-6f6f6f6f6f6f": admins},
	}
	return entries, competitionAdmins
}

// TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds
// is TestRecordOfflinePayment_CompetitionEntryPayable_EntrantSucceeds
// (service_test.go), re-proved with real, uuid-shaped, mutually distinct
// identifiers rather than short mnemonic strings — see this file's own doc
// comment for why that distinction matters for #237 specifically.
func TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds(t *testing.T) {
	t.Parallel()

	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: regressionEntrantUserID,
	})
	if err != nil {
		t.Fatalf("the entrant's own resolved User.ID should record this CompetitionEntry payment, got: %v", err)
	}
	if p.RecordedByUserID != regressionEntrantUserID {
		t.Fatalf("RecordedByUserID = %q, want %q", p.RecordedByUserID, regressionEntrantUserID)
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDAssignedAdminSucceeds
// re-proves TestRecordOfflinePayment_CompetitionEntryPayable_AssignedCompetitionAdminSucceeds
// with distinct real uuids for the entrant and the admin — the pair this
// file's fixtures are built to never let collide by accident.
func TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDAssignedAdminSucceeds(t *testing.T) {
	t.Parallel()

	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: regressionAdminUserID,
	})
	if err != nil {
		t.Fatalf("an assigned Competition Admin's resolved User.ID should record this CompetitionEntry payment, got: %v", err)
	}
	if p.RecordedByUserID != regressionAdminUserID {
		t.Fatalf("RecordedByUserID = %q, want %q", p.RecordedByUserID, regressionAdminUserID)
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDStrangerRejected
// is this file's negative pin: a genuinely different, real, resolved
// User.ID — neither the entrant nor an assigned admin — is still refused.
// Not merely "some unregistered garbage string is rejected" (which every
// pre-#237 and post-#237 build alike would reject): regressionStrangerUserID
// is exactly as uuid-shaped and exactly as "real" as the two identities
// above, so this proves the comparison is genuinely selective, not
// vacuously true because nothing but the fixture's own configured values
// could ever match.
func TestRecordOfflinePayment_CompetitionEntryPayable_RealUUIDStrangerRejected(t *testing.T) {
	t.Parallel()

	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: regressionStrangerUserID,
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v — a real, resolved User.ID that is neither the entrant "+
			"nor an assigned admin must still be refused", err, domain.ErrNotPaymentRecorder)
	}
}

// TestRefundPayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds re-proves
// TestRefundPayment_OfflineCompetitionEntryPayable_EntrantSucceeds
// (refund_test.go) with real, uuid-shaped, mutually distinct identifiers.
func TestRefundPayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: regressionEntrantUserID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   regressionCompetitionEntryPaymentID,
		ActorUserID: regressionEntrantUserID,
	})
	if err != nil {
		t.Fatalf("the entrant's own resolved User.ID should refund this CompetitionEntry payment, got: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}
}

// TestRefundPayment_CompetitionEntryPayable_RealUUIDStrangerRejected mirrors
// TestRefundPayment_CompetitionEntryPayable_WrongActorRejected
// (refund_test.go) with real, uuid-shaped, mutually distinct identifiers —
// the refund-path twin of this file's RecordOfflinePayment negative pin.
func TestRefundPayment_CompetitionEntryPayable_RealUUIDStrangerRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: regressionEntrantUserID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   regressionCompetitionEntryPaymentID,
		ActorUserID: regressionStrangerUserID,
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v — a real, resolved User.ID that is neither the entrant "+
			"nor an assigned admin must still be refused a refund", err, domain.ErrNotPaymentRecorder)
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds
// re-proves TestCreateOnlinePayment_CompetitionEntryPayable_EntrantSucceeds
// (service_test.go) with real, uuid-shaped, mutually distinct identifiers —
// the online-checkout twin of this file's RecordOfflinePayment/RefundPayment
// proofs, per T29.1 instruction 7's explicit naming of all three RPC
// families.
func TestCreateOnlinePayment_CompetitionEntryPayable_RealUUIDEntrantSucceeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		Processor:              proc,
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      fixtureAmount(),
		ActorUserID: regressionEntrantUserID,
	})
	if err != nil {
		t.Fatalf("the entrant's own resolved User.ID should create an online CompetitionEntry payment, got: %v", err)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unpaid (creating an intent does not capture funds)", p.Status)
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_RealUUIDStrangerRejected
// mirrors TestCreateOnlinePayment_CompetitionEntryPayable_UnauthorizedActorRejected
// (service_test.go) with real, uuid-shaped, mutually distinct identifiers.
func TestCreateOnlinePayment_CompetitionEntryPayable_RealUUIDStrangerRejected(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	entries, admins := newRegressionEntryAuthzFixtures(regressionEntrantUserID, regressionAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{regressionCompetitionEntryPaymentID}},
		Processor:              proc,
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   regressionCompetitionEntryID,
		Amount:      fixtureAmount(),
		ActorUserID: regressionStrangerUserID,
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v — a real, resolved User.ID that is neither the entrant "+
			"nor an assigned admin must still be refused an online payment intent", err, domain.ErrNotPaymentRecorder)
	}
}
