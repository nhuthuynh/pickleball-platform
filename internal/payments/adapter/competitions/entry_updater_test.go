package competitions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsport "github.com/nhuthuynh/white-label/internal/competitions/port"
)

// --- minimal in-memory port.Repository fake -----------------------------
//
// This adapter's own job is the translation at the boundary (see
// entry_updater.go), not re-testing app.Service's own behaviour, which
// internal/competitions/app/service_test.go already covers — so this fake
// implements only what MarkCompetitionEntryPaymentStatus actually touches
// (GetEntryByID/UpdateEntryPaymentStatus), plus the rest of port.Repository
// as never-called stubs required to satisfy the interface, mirroring
// internal/payments/adapter/socialplay/registration_updater_test.go's
// fakeRegistrations shape.
type fakeRepository struct {
	entries map[string]competitionsdomain.CompetitionEntry
}

func newFakeRepository(seed ...competitionsdomain.CompetitionEntry) *fakeRepository {
	m := make(map[string]competitionsdomain.CompetitionEntry, len(seed))
	for _, e := range seed {
		m[e.ID] = e
	}
	return &fakeRepository{entries: m}
}

func (f *fakeRepository) Create(context.Context, competitionsdomain.Competition) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, nil
}
func (f *fakeRepository) GetByID(context.Context, string) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *fakeRepository) GetByShareToken(context.Context, string) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *fakeRepository) UpdateStatus(context.Context, string, competitionsdomain.Status) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *fakeRepository) CreateEntry(_ context.Context, e competitionsdomain.CompetitionEntry) (competitionsdomain.CompetitionEntry, error) {
	f.entries[e.ID] = e
	return e, nil
}
func (f *fakeRepository) ListActiveEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *fakeRepository) ListEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *fakeRepository) ListCompetitions(context.Context, competitionsport.CompetitionListingFilter) ([]competitionsport.CompetitionListing, error) {
	return nil, nil
}
func (f *fakeRepository) GetEntryByID(_ context.Context, id string) (competitionsdomain.CompetitionEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
	}
	return e, nil
}
func (f *fakeRepository) UpdateEntryPaymentStatus(_ context.Context, id string, status competitionsdomain.PaymentStatus) (competitionsdomain.CompetitionEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
	}
	e.PaymentStatus = status
	f.entries[id] = e
	return e, nil
}

// CancelAllActiveForCompetition (T16.3) is unexercised here — this file's
// MarkCompetitionEntryPaymentStatus test never cancels a Competition — so
// it returns a zero value, matching the never-exercised-dependency stubs
// below.
func (f *fakeRepository) CancelAllActiveForCompetition(context.Context, string) (int, error) {
	return 0, nil
}

// --- other never-exercised ServiceOptions dependencies -------------------
//
// MarkCompetitionEntryPaymentStatus never touches any of these, but
// competitionsapp.NewService's ServiceOptions requires all five fields
// (none is optional, unlike Payments' RegistrationUpdater/EntryUpdater —
// see ServiceOptions's own doc comment), mirroring
// registration_updater_test.go's fakeGames/fakeWaitlist stubs.
type fakeIDs struct{}

func (fakeIDs) NewID() string { return "unused" }

type fakeReservation struct{}

func (fakeReservation) ReserveCourt(context.Context, string, time.Time, time.Time, string, string) (string, error) {
	return "", nil
}
func (fakeReservation) ReleaseCourt(context.Context, string, string) error { return nil }

type fakeFacilityLookup struct{}

func (fakeFacilityLookup) FacilityExists(context.Context, string) error { return nil }

type fakeShareTokens struct{}

func (fakeShareTokens) NewShareToken() (string, error) { return "unused", nil }

func newTestCompetitionsService(repo *fakeRepository) *competitionsapp.Service {
	return competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions: repo,
		IDs:          fakeIDs{},
		Reservation:  fakeReservation{},
		Facilities:   fakeFacilityLookup{},
		ShareTokens:  fakeShareTokens{},
	})
}

// TestEntryUpdater_ImplementsPort is a compile-time proof that
// *paymentscompetitions.EntryUpdater satisfies
// competitionsport.CompetitionEntryPaymentUpdater — the whole point of this
// package (mirror image of internal/payments/adapter/socialplay implementing
// internal/socialplay/port.RegistrationPaymentUpdater against Social Play's
// real app.Service).
var _ competitionsport.CompetitionEntryPaymentUpdater = (*paymentscompetitions.EntryUpdater)(nil)

// TestUpdatePaymentStatus_Succeeds proves the happy path calls through to
// the real Competitions app.Service and the change is observable via a
// fresh GetEntryByID — not just that no error was returned.
func TestUpdatePaymentStatus_Succeeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository(competitionsdomain.CompetitionEntry{
		ID: "entry-1", CompetitionID: "comp-1", PlayerID: "player-1",
		Status:        competitionsdomain.EntryStatusEntered,
		PaymentStatus: competitionsdomain.PaymentStatusUnpaid,
	})
	svc := newTestCompetitionsService(repo)
	updater := paymentscompetitions.NewEntryUpdater(svc)

	if err := updater.UpdatePaymentStatus(context.Background(), "entry-1", competitionsdomain.PaymentStatusPaid); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	stored, err := repo.GetEntryByID(context.Background(), "entry-1")
	if err != nil {
		t.Fatalf("GetEntryByID err: %v", err)
	}
	if stored.PaymentStatus != competitionsdomain.PaymentStatusPaid {
		t.Fatalf("PaymentStatus = %v, want paid", stored.PaymentStatus)
	}
}

// TestUpdatePaymentStatus_NotFound proves a bogus entry id surfaces
// competitionsdomain.ErrCompetitionEntryNotFound unchanged — the exact
// sentinel port.CompetitionEntryPaymentUpdater's doc comment promises, not
// some Payments-side wrapper error.
func TestUpdatePaymentStatus_NotFound(t *testing.T) {
	t.Parallel()

	svc := newTestCompetitionsService(newFakeRepository())
	updater := paymentscompetitions.NewEntryUpdater(svc)

	err := updater.UpdatePaymentStatus(context.Background(), "no-such-entry", competitionsdomain.PaymentStatusPaid)
	if !errors.Is(err, competitionsdomain.ErrCompetitionEntryNotFound) {
		t.Fatalf("got err %v, want %v", err, competitionsdomain.ErrCompetitionEntryNotFound)
	}
}
