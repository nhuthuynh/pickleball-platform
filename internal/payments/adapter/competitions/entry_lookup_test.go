package competitions_test

import (
	"context"
	"testing"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsport "github.com/nhuthuynh/white-label/internal/competitions/port"
	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	paymentsport "github.com/nhuthuynh/white-label/internal/payments/port"
)

// entryLookupFakeRepository is a minimal but FUNCTIONAL competitions/port.
// Repository — unlike entry_updater_test.go's fakeRepository (whose
// GetEntryByID always answers ErrCompetitionEntryNotFound, because
// MarkCompetitionEntryPaymentStatus never touches it via this path) and
// competition_admin_reader_test.go's adminFakeRepository (whose GetEntryByID
// is the identical always-not-found stub, since it only needed GetByID on
// the Competition itself), GetEntryByID here must actually resolve a seeded
// CompetitionEntry — a third, differently-named fake rather than editing
// either existing one, since both are exercised by tests that depend on
// their own GetEntryByID shape.
//
// GetByID (on the Competition itself) is ALSO functional, unlike
// entry_updater_test.go's stub: authorization_boundary_test.go's mutation
// table needs AssignCompetitionAdmin/RevokeCompetitionAdmin to resolve a
// real seeded Competition (both load it first, to compare its HostID against
// the actor), not just its entries.
type entryLookupFakeRepository struct {
	entries      map[string]competitionsdomain.CompetitionEntry
	competitions map[string]competitionsdomain.Competition
}

func newEntryLookupFakeRepository(seed ...competitionsdomain.CompetitionEntry) *entryLookupFakeRepository {
	m := make(map[string]competitionsdomain.CompetitionEntry, len(seed))
	for _, e := range seed {
		m[e.ID] = e
	}
	return &entryLookupFakeRepository{entries: m, competitions: map[string]competitionsdomain.Competition{}}
}

func (f *entryLookupFakeRepository) Create(_ context.Context, c competitionsdomain.Competition) (competitionsdomain.Competition, error) {
	f.competitions[c.ID] = c
	return c, nil
}
func (f *entryLookupFakeRepository) GetByID(_ context.Context, id string) (competitionsdomain.Competition, error) {
	c, ok := f.competitions[id]
	if !ok {
		return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
	}
	return c, nil
}
func (f *entryLookupFakeRepository) GetByShareToken(context.Context, string) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *entryLookupFakeRepository) UpdateStatus(context.Context, string, competitionsdomain.Status) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *entryLookupFakeRepository) CreateEntry(_ context.Context, e competitionsdomain.CompetitionEntry) (competitionsdomain.CompetitionEntry, error) {
	f.entries[e.ID] = e
	return e, nil
}
func (f *entryLookupFakeRepository) ListActiveEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *entryLookupFakeRepository) ListEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *entryLookupFakeRepository) ListCompetitions(context.Context, competitionsport.CompetitionListingFilter) ([]competitionsport.CompetitionListing, error) {
	return nil, nil
}
func (f *entryLookupFakeRepository) GetEntryByID(_ context.Context, id string) (competitionsdomain.CompetitionEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
	}
	return e, nil
}
func (f *entryLookupFakeRepository) UpdateEntryPaymentStatus(_ context.Context, id string, status competitionsdomain.PaymentStatus) (competitionsdomain.CompetitionEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
	}
	e.PaymentStatus = status
	f.entries[id] = e
	return e, nil
}

func newEntryLookupTestService(repo *entryLookupFakeRepository) *competitionsapp.Service {
	return competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions:      repo,
		IDs:               fakeIDs{},
		Reservation:       fakeReservation{},
		Facilities:        fakeFacilityLookup{},
		ShareTokens:       fakeShareTokens{},
		CompetitionAdmins: newAdminFakeCompetitionAdmins(),
	})
}

// TestEntryLookup_ImplementsPort is a compile-time proof that
// *paymentscompetitions.EntryLookup satisfies paymentsport.EntryLookup.
var _ paymentsport.EntryLookup = (*paymentscompetitions.EntryLookup)(nil)

const (
	entryLookupCompetitionID = "6ba7b810-0000-4000-8000-0000000000f4"
	entryLookupEntryID       = "6ba7b810-0000-4000-8000-0000000000f5"
	entryLookupPlayerID      = "player-subject-t16-2"
)

// TestCompetitionIDAndPlayerIDForEntry_Succeeds proves the happy path calls
// through to the real competitionsapp.Service.GetEntryByID (T16.2 — the
// first exported single-entry read Competitions has offered outside its own
// package, see that method's own doc comment) and returns both the
// CompetitionID and PlayerID in one call — driven through the real service,
// not a fake app.Service, per T14.8/T15.5's cross-context-fake warning.
func TestCompetitionIDAndPlayerIDForEntry_Succeeds(t *testing.T) {
	t.Parallel()

	repo := newEntryLookupFakeRepository(competitionsdomain.CompetitionEntry{
		ID:            entryLookupEntryID,
		CompetitionID: entryLookupCompetitionID,
		PlayerID:      entryLookupPlayerID,
		Status:        competitionsdomain.EntryStatusEntered,
		PaymentStatus: competitionsdomain.PaymentStatusUnpaid,
	})
	svc := newEntryLookupTestService(repo)
	lookup := paymentscompetitions.NewEntryLookup(svc)

	competitionID, playerID, err := lookup.CompetitionIDAndPlayerIDForEntry(context.Background(), entryLookupEntryID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if competitionID != entryLookupCompetitionID {
		t.Fatalf("competitionID = %q, want %q", competitionID, entryLookupCompetitionID)
	}
	if playerID != entryLookupPlayerID {
		t.Fatalf("playerID = %q, want %q", playerID, entryLookupPlayerID)
	}
}

// TestCompetitionIDAndPlayerIDForEntry_UnknownEntry proves an unknown
// entryID answers ("", "", nil) — port.EntryLookup's own documented
// translation of competitionsdomain.ErrCompetitionEntryNotFound.
func TestCompetitionIDAndPlayerIDForEntry_UnknownEntry(t *testing.T) {
	t.Parallel()

	svc := newEntryLookupTestService(newEntryLookupFakeRepository())
	lookup := paymentscompetitions.NewEntryLookup(svc)

	competitionID, playerID, err := lookup.CompetitionIDAndPlayerIDForEntry(context.Background(), "00000000-0000-4000-8000-000000009999")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if competitionID != "" || playerID != "" {
		t.Fatalf("got (%q, %q), want both empty", competitionID, playerID)
	}
}
