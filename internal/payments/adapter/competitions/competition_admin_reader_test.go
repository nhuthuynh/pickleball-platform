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

// adminFakeRepository is a minimal but FUNCTIONAL competitions/port.
// Repository — unlike entry_updater_test.go's fakeRepository (whose GetByID
// always answers ErrCompetitionNotFound, because
// MarkCompetitionEntryPaymentStatus never touches it), GetByID here must
// actually resolve a seeded Competition: AssignCompetitionAdmin,
// RevokeCompetitionAdmin and ListCompetitionAdmins all load the Competition
// first. A second, differently-named fake rather than editing the existing
// one, since entry_updater_test.go's fakeRepository is exercised by tests
// that depend on its always-not-found GetByID.
type adminFakeRepository struct {
	competitions map[string]competitionsdomain.Competition
}

func newAdminFakeRepository(seed ...competitionsdomain.Competition) *adminFakeRepository {
	m := make(map[string]competitionsdomain.Competition, len(seed))
	for _, c := range seed {
		m[c.ID] = c
	}
	return &adminFakeRepository{competitions: m}
}

func (f *adminFakeRepository) Create(_ context.Context, c competitionsdomain.Competition) (competitionsdomain.Competition, error) {
	f.competitions[c.ID] = c
	return c, nil
}
func (f *adminFakeRepository) GetByID(_ context.Context, id string) (competitionsdomain.Competition, error) {
	c, ok := f.competitions[id]
	if !ok {
		return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
	}
	return c, nil
}
func (f *adminFakeRepository) GetByShareToken(context.Context, string) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *adminFakeRepository) UpdateStatus(context.Context, string, competitionsdomain.Status) (competitionsdomain.Competition, error) {
	return competitionsdomain.Competition{}, competitionsdomain.ErrCompetitionNotFound
}
func (f *adminFakeRepository) CreateEntry(context.Context, competitionsdomain.CompetitionEntry) (competitionsdomain.CompetitionEntry, error) {
	return competitionsdomain.CompetitionEntry{}, nil
}
func (f *adminFakeRepository) ListActiveEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *adminFakeRepository) ListEntriesForCompetition(context.Context, string) ([]competitionsdomain.CompetitionEntry, error) {
	return nil, nil
}
func (f *adminFakeRepository) ListCompetitions(context.Context, competitionsport.CompetitionListingFilter) ([]competitionsport.CompetitionListing, error) {
	return nil, nil
}
func (f *adminFakeRepository) GetEntryByID(context.Context, string) (competitionsdomain.CompetitionEntry, error) {
	return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
}
func (f *adminFakeRepository) UpdateEntryPaymentStatus(context.Context, string, competitionsdomain.PaymentStatus) (competitionsdomain.CompetitionEntry, error) {
	return competitionsdomain.CompetitionEntry{}, competitionsdomain.ErrCompetitionEntryNotFound
}

// adminFakeCompetitionAdmins is a minimal but FUNCTIONAL port.
// CompetitionAdminRepository — the in-memory store this test actually
// exercises Assign/Revoke/ListCompetitionAdmins against. Mirrors
// internal/competitions/app/competition_admin_test.go's own
// fakeCompetitionAdminRepository (unreachable from here: package app_test,
// unexported), reproduced independently. Deliberately real, not a stub that
// "returns whatever it's told" — T14.8's cross-context-fake trap this
// ticket's own instruction 7 names.
type adminFakeCompetitionAdmins struct {
	admins map[string][]competitionsdomain.CompetitionAdmin
}

func newAdminFakeCompetitionAdmins() *adminFakeCompetitionAdmins {
	return &adminFakeCompetitionAdmins{admins: make(map[string][]competitionsdomain.CompetitionAdmin)}
}

func (r *adminFakeCompetitionAdmins) Assign(_ context.Context, a competitionsdomain.CompetitionAdmin) (competitionsdomain.CompetitionAdmin, error) {
	for _, existing := range r.admins[a.CompetitionID] {
		if existing.UserID == a.UserID {
			return competitionsdomain.CompetitionAdmin{}, competitionsdomain.ErrAlreadyCompetitionAdmin
		}
	}
	r.admins[a.CompetitionID] = append(r.admins[a.CompetitionID], a)
	return a, nil
}

func (r *adminFakeCompetitionAdmins) Revoke(_ context.Context, competitionID, userID string) error {
	for i, existing := range r.admins[competitionID] {
		if existing.UserID == userID {
			r.admins[competitionID] = append(r.admins[competitionID][:i], r.admins[competitionID][i+1:]...)
			return nil
		}
	}
	return competitionsdomain.ErrCompetitionAdminNotFound
}

func (r *adminFakeCompetitionAdmins) ListCompetitionAdmins(_ context.Context, competitionID string) ([]competitionsdomain.CompetitionAdmin, error) {
	out := make([]competitionsdomain.CompetitionAdmin, len(r.admins[competitionID]))
	copy(out, r.admins[competitionID])
	return out, nil
}

func newAdminTestCompetitionsService(repo *adminFakeRepository, admins *adminFakeCompetitionAdmins) *competitionsapp.Service {
	return competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions:      repo,
		IDs:               fakeIDs{},
		Reservation:       fakeReservation{},
		Facilities:        fakeFacilityLookup{},
		ShareTokens:       fakeShareTokens{},
		CompetitionAdmins: admins,
	})
}

// TestCompetitionAdminReader_ImplementsPort is a compile-time proof that
// *paymentscompetitions.CompetitionAdminReader satisfies
// paymentsport.CompetitionAdminReader — mirroring
// TestEntryUpdater_ImplementsPort's role (entry_updater_test.go) for the
// push-out adapter, now for the read-in direction (T15.5, §A12 GAP C).
var _ paymentsport.CompetitionAdminReader = (*paymentscompetitions.CompetitionAdminReader)(nil)

const (
	competitionAdminReaderCompetitionID = "6ba7b810-0000-4000-8000-0000000000f2"
	competitionAdminReaderHostID        = "host-subject-t15-5"
	competitionAdminReaderAdmin         = "admin-subject-t15-5"
)

func competitionAdminReaderFixture() (*competitionsapp.Service, *adminFakeCompetitionAdmins) {
	repo := newAdminFakeRepository(competitionsdomain.Competition{ID: competitionAdminReaderCompetitionID, HostID: competitionAdminReaderHostID})
	admins := newAdminFakeCompetitionAdmins()
	return newAdminTestCompetitionsService(repo, admins), admins
}

// TestListCompetitionAdmins_AssignThenRevoke is T15.5 instruction 7's
// headline positive-then-negative control, delivered at the seam this
// ticket could actually wire (see CompetitionAdminReader's own doc comment
// for why it stops short of internal/payments/app.Service): the SAME user
// appears in ListCompetitionAdmins once genuinely assigned through the real
// Competitions app.Service, and disappears again once revoked — driven
// through the real competitionsapp.Service, not a fake that "returns
// whatever it's told" (T14.8's cross-context-fake note).
func TestListCompetitionAdmins_AssignThenRevoke(t *testing.T) {
	t.Parallel()

	svc, _ := competitionAdminReaderFixture()
	reader := paymentscompetitions.NewCompetitionAdminReader(svc)
	ctx := context.Background()

	before, err := reader.ListCompetitionAdmins(ctx, competitionAdminReaderCompetitionID)
	if err != nil {
		t.Fatalf("ListCompetitionAdmins (before assign) unexpected error: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ListCompetitionAdmins (before assign) = %v, want empty", before)
	}

	if _, err := svc.AssignCompetitionAdmin(ctx, competitionsapp.AssignCompetitionAdminInput{
		CompetitionID: competitionAdminReaderCompetitionID,
		ActorUserID:   competitionAdminReaderHostID,
		AdminUserID:   competitionAdminReaderAdmin,
	}); err != nil {
		t.Fatalf("AssignCompetitionAdmin: %v", err)
	}

	assigned, err := reader.ListCompetitionAdmins(ctx, competitionAdminReaderCompetitionID)
	if err != nil {
		t.Fatalf("ListCompetitionAdmins (after assign) unexpected error: %v", err)
	}
	if !containsString(assigned, competitionAdminReaderAdmin) {
		t.Fatalf("ListCompetitionAdmins (after assign) = %v, want it to contain %s", assigned, competitionAdminReaderAdmin)
	}

	if err := svc.RevokeCompetitionAdmin(ctx, competitionsapp.RevokeCompetitionAdminInput{
		CompetitionID: competitionAdminReaderCompetitionID,
		ActorUserID:   competitionAdminReaderHostID,
		AdminUserID:   competitionAdminReaderAdmin,
	}); err != nil {
		t.Fatalf("RevokeCompetitionAdmin: %v", err)
	}

	revoked, err := reader.ListCompetitionAdmins(ctx, competitionAdminReaderCompetitionID)
	if err != nil {
		t.Fatalf("ListCompetitionAdmins (after revoke) unexpected error: %v", err)
	}
	if containsString(revoked, competitionAdminReaderAdmin) {
		t.Fatalf("ListCompetitionAdmins (after revoke) = %v, want it to NOT contain %s", revoked, competitionAdminReaderAdmin)
	}
}

// TestListCompetitionAdmins_UnknownCompetition proves an unknown
// competitionID answers an empty slice with no error —
// CompetitionAdminReader's own doc comment's translation, deliberately
// different from competitionsapp.Service.ListCompetitionAdmins' own
// domain.ErrCompetitionNotFound answer for the same input.
func TestListCompetitionAdmins_UnknownCompetition(t *testing.T) {
	t.Parallel()

	svc, _ := competitionAdminReaderFixture()
	reader := paymentscompetitions.NewCompetitionAdminReader(svc)

	got, err := reader.ListCompetitionAdmins(context.Background(), "00000000-0000-4000-8000-000000009999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

// containsString mirrors internal/payments/adapter/socialplay's identically
// named test helper — kept as an independent copy rather than a shared
// export, since these are test-only files in two different packages and
// neither depends on the other.
func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
