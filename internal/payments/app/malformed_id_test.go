// Boundary validation for caller-supplied ids in the Payments context's
// three write handlers (T10.7, closing issue #97): CreateOnlinePayment and
// RecordOfflinePayment on PayableID, and ConfirmOnlinePayment on payment_id
// (via the new app.Service.GetPayment, added by this ticket so the guard
// lives in the same layer as every other context's — see that method's doc
// comment in service.go for why the grpcapi handler used to do this lookup
// itself).
//
// Same class of bug as internal/competitions/app/malformed_id_test.go
// documents in full: internal/payments/adapter/postgres.mustUUID panics on
// anything pgtype.UUID.Scan can't parse, and both payments.id and
// payments.payable_id are `uuid` columns (db/migrations/0005_payments.sql) —
// payable_id is NOT foreign-keyed to another table (it's a polymorphic
// reference, per that migration's own comment), so unlike
// CancelCompetition/AddCourt/CancelBooking there is no existing "does this
// PayableID refer to a real thing" lookup to match the not-found semantics
// of. What *does* already exist is domain.NewPayment's own
// ErrEmptyPayableID/InvalidArgument rejection of an empty PayableID — the
// guard below extends that same sentinel and status code to cover a
// non-empty but malformed-shape PayableID too, rather than inventing a new
// status this field has never returned before (this ticket's own
// instruction: "no new status shape invented per handler").
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// malformedIDs is the shared corpus of things a client can put in an id
// field that are not ids this system ever mints — mirrors the corpus in
// internal/competitions/app/malformed_id_test.go and
// internal/facilities/app/malformed_id_test.go exactly (T10.7 reuses the
// existing helper/pattern rather than re-deriving one, per the ticket's own
// instructions).
var malformedIDs = []string{
	"",
	"not-a-uuid",
	"0",
	"'; DROP TABLE payments;--",
	"../../etc/passwd",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c",
	"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
	"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
	"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	" 6ba7b810-9dad-11d1-80b4-00c04fd430c8 ",
}

// wellFormedUnknownID is a real UUID shape that collides with no fixture
// this file uses, for the "too-strict guard" sanity checks below.
const wellFormedUnknownID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// --- CreateOnlinePayment -----------------------------------------------

func TestCreateOnlinePayment_MalformedPayableIDIsInvalidArgumentAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			svc := app.NewService(app.ServiceOptions{
				Payments:  repo,
				IDs:       &fixedIDs{ids: []string{"pay-1"}},
				Processor: stripestub.NewProcessor(),
			})

			_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
				PayableType: domain.PayableTypeBooking,
				PayableID:   id,
				Amount:      fixtureAmount(),
			})
			if !errors.Is(err, domain.ErrEmptyPayableID) {
				t.Fatalf("CreateOnlinePayment(PayableID=%q) error = %v, want %v", id, err, domain.ErrEmptyPayableID)
			}
			if calls := repo.createCalls.Load(); calls != 0 {
				t.Errorf("malformed PayableID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestCreateOnlinePayment_WellFormedUnknownPayableIDStillReachesRepository is
// the too-strict guard rail: PayableID has no existence check in this
// context (it's a polymorphic, un-FK'd reference — see this file's doc
// comment), so ANY well-formed PayableID, known or not, must still reach the
// repository exactly as it did before this ticket.
func TestCreateOnlinePayment_WellFormedUnknownPayableIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: stripestub.NewProcessor(),
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   wellFormedUnknownID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls := repo.createCalls.Load(); calls != 1 {
		t.Fatalf("well-formed PayableID did not reach the repository (%d calls) — the guard is too strict", calls)
	}
}

// --- RecordOfflinePayment -----------------------------------------------

func TestRecordOfflinePayment_MalformedPayableIDIsInvalidArgumentAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			svc := app.NewService(app.ServiceOptions{
				Payments: repo,
				IDs:      &fixedIDs{ids: []string{"pay-1"}},
			})

			// A legitimate, authorized actor — this proves the PayableID
			// shape guard, not authorizeOfflineRecording's separate check
			// (already covered by TestRecordOfflinePayment_BookingPayable_
			// WrongActorRejected in service_test.go).
			_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
				PayableType:   domain.PayableTypeBooking,
				PayableID:     id,
				Amount:        offlineFixtureAmount(),
				ActorUserID:   "host-1",
				BookingHostID: "host-1",
			})
			if !errors.Is(err, domain.ErrEmptyPayableID) {
				t.Fatalf("RecordOfflinePayment(PayableID=%q) error = %v, want %v", id, err, domain.ErrEmptyPayableID)
			}
			if calls := repo.createCalls.Load(); calls != 0 {
				t.Errorf("malformed PayableID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestRecordOfflinePayment_WellFormedUnknownPayableIDStillReachesRepository
// mirrors the online path's own guard-rail test above.
func TestRecordOfflinePayment_WellFormedUnknownPayableIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     wellFormedUnknownID,
		Amount:        offlineFixtureAmount(),
		ActorUserID:   "host-1",
		BookingHostID: "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls := repo.createCalls.Load(); calls != 1 {
		t.Fatalf("well-formed PayableID did not reach the repository (%d calls) — the guard is too strict", calls)
	}
}

// --- ConfirmOnlinePayment (via the new app.Service.GetPayment) ----------
//
// ConfirmOnlinePayment itself takes a domain.Payment, not an id (see
// service.go's doc comment) — the id -> Payment lookup that used to happen
// directly in internal/payments/adapter/grpcapi's handler (bypassing the
// app layer entirely, unlike every other context's equivalent guard) now
// goes through app.Service.GetPayment, added by this ticket. GetPayment
// already returns the bare domain.ErrPaymentNotFound for an
// unknown-but-well-formed id (mirrors GetByID's own contract), so a
// malformed id must answer identically.

func TestGetPayment_MalformedIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			svc := app.NewService(app.ServiceOptions{Payments: repo, IDs: &fixedIDs{}})

			_, err := svc.GetPayment(context.Background(), id)
			if !errors.Is(err, domain.ErrPaymentNotFound) {
				t.Fatalf("GetPayment(%q) error = %v, want %v", id, err, domain.ErrPaymentNotFound)
			}
			if calls := repo.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed payment id %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

func TestGetPayment_WellFormedUnknownIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{Payments: repo, IDs: &fixedIDs{}})

	_, err := svc.GetPayment(context.Background(), wellFormedUnknownID)
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("GetPayment(%q) error = %v, want %v", wellFormedUnknownID, err, domain.ErrPaymentNotFound)
	}
	if calls := repo.getByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown payment id did not reach the repository (%d calls) — the guard is too strict", calls)
	}
}

// TestGetPayment_WellFormedKnownIDStillResolves proves the guard doesn't
// break the real ConfirmOnlinePayment flow: a Payment created via
// CreateOnlinePayment must still be resolvable by GetPayment afterward.
func TestGetPayment_WellFormedKnownIDStillResolves(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		// UUID-shaped, unlike this file's other fixedIDs fixtures (which
		// use the non-UUID "pay-1"/"pay-2" convention pre-dating T10.7):
		// this test's whole point is proving GetPayment resolves a REAL
		// id, and a non-UUID-shaped one would be rejected by GetPayment's
		// own guard before ever reaching the repository — silently turning
		// this into a copy of TestGetPayment_MalformedIDIsNotFoundAnd...
		// rather than the positive case it's meant to be.
		IDs:       &fixedIDs{ids: []string{"6ba7b810-0000-4000-8000-00000000000f"}},
		Processor: stripestub.NewProcessor(),
	})

	created, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("fixture CreateOnlinePayment: %v", err)
	}

	got, err := svc.GetPayment(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetPayment(%q) on a real Payment: %v", created.ID, err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetPayment returned %q, want %q", got.ID, created.ID)
	}
}
