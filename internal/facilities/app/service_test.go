package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// inMemoryRepo is a minimal port.Repository fake, mirroring
// internal/booking/app's inMemoryRepo — no persistence, no concurrency
// guarantees. It exists purely to let the app-layer orchestration be
// tested without a database.
type inMemoryRepo struct {
	facilities map[string]domain.Facility
	courts     map[string]domain.Court
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		facilities: make(map[string]domain.Facility),
		courts:     make(map[string]domain.Court),
	}
}

func (r *inMemoryRepo) CreateFacility(_ context.Context, f domain.Facility) (domain.Facility, error) {
	r.facilities[f.ID] = f
	return f, nil
}

func (r *inMemoryRepo) GetFacilityByID(_ context.Context, id string) (domain.Facility, error) {
	f, ok := r.facilities[id]
	if !ok {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}
	return f, nil
}

func (r *inMemoryRepo) ListFacilities(_ context.Context, nameFilter string) ([]domain.Facility, error) {
	var out []domain.Facility
	for _, f := range r.facilities {
		if nameFilter == "" || containsFold(f.Name, nameFilter) {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *inMemoryRepo) AddCourt(_ context.Context, c domain.Court) (domain.Court, error) {
	r.courts[c.ID] = c
	return c, nil
}

func (r *inMemoryRepo) AddCameraLink(_ context.Context, facilityID string, link domain.CameraLink) (domain.CameraLink, error) {
	f, ok := r.facilities[facilityID]
	if !ok {
		return domain.CameraLink{}, domain.ErrFacilityNotFound
	}
	f.CameraLinks = append(f.CameraLinks, link)
	r.facilities[facilityID] = f
	return link, nil
}

func (r *inMemoryRepo) AttestCameraConsent(_ context.Context, facilityID string) error {
	f, ok := r.facilities[facilityID]
	if !ok {
		return domain.ErrFacilityNotFound
	}
	f.CameraConsentAttested = true
	r.facilities[facilityID] = f
	return nil
}

// containsFold is a tiny case-insensitive substring check, just enough to
// fake ListFacilities' ILIKE filter in-memory without pulling in strings
// import ceremony for one call site.
func containsFold(s, substr string) bool {
	return len(substr) == 0 || indexFold(s, substr) >= 0
}

func indexFold(s, substr string) int {
	sl, subl := len(s), len(substr)
	if subl == 0 {
		return 0
	}
	for i := 0; i+subl <= sl; i++ {
		if equalFold(s[i:i+subl], substr) {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// sequentialIDs is a deterministic port.IDGenerator fake, mirroring
// internal/booking/app's sequentialIDs.
type sequentialIDs struct{ n int }

func (g *sequentialIDs) NewID() string {
	g.n++
	return fmt.Sprintf("id-%d", g.n)
}

func TestCreateFacility_Valid(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID:     "owner-1",
		Name:        "Riverside Courts",
		Description: "Nice courts",
		Address:     "123 Main St",
		PhotoURLs:   []string{"https://example.com/p.jpg"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if f.ID != "id-1" {
		t.Fatalf("ID = %q, want id-1", f.ID)
	}
	if f.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = true, want false by default")
	}
}

func TestCreateFacility_ValidationRejectedBeforeTouchingRepo(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	_, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID: "", Name: "Riverside Courts", Address: "123 Main St",
	})
	if !errors.Is(err, domain.ErrEmptyFacilityField) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyFacilityField)
	}
	if len(repo.facilities) != 0 {
		t.Fatalf("invalid facility must not be persisted, repo has %d entries", len(repo.facilities))
	}
}

func TestGetFacility_UnknownReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	_, err := svc.GetFacility(ctx, "does-not-exist")
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrFacilityNotFound)
	}
}

func TestListFacilities_NameFilter(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	if _, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Lakeside Courts", Address: "2 St"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got, err := svc.ListFacilities(ctx, "river")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Riverside Courts" {
		t.Fatalf("got %v, want only Riverside Courts", got)
	}

	all, err := svc.ListFacilities(ctx, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d facilities, want 2 with empty filter", len(all))
	}
}

// TestAddCourt_InsertsIntoExistingCourtsTable proves AddCourt's app-layer
// contract: it builds a domain.Court scoped to the given facility and
// persists it via the same Repository.AddCourt port that the Postgres
// adapter wires to the *existing* courts table (HANDOFF.md T7.3 AC 5) — no
// second courts table, no duplicate rows. The in-memory fake here can't
// prove the "existing table" part (that's the Postgres adapter's job,
// exercised via the integration/smoke test), but it does prove the service
// calls through with the right FacilityID/Name and surfaces AddCourt's
// return.
func TestAddCourt_InsertsIntoExistingCourtsTable(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	c, err := svc.AddCourt(ctx, f.ID, "owner-1", "Court 1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.FacilityID != f.ID {
		t.Fatalf("FacilityID = %q, want %q", c.FacilityID, f.ID)
	}
	if c.Name != "Court 1" {
		t.Fatalf("Name = %q, want Court 1", c.Name)
	}
	if _, ok := repo.courts[c.ID]; !ok {
		t.Fatalf("court was not persisted")
	}
}

// TestAddCourt_SucceedsWithoutCameraConsent is the AC from HANDOFF.md T7.3
// bullet 6: creating a facility with CameraConsentAttested false and then
// calling AddCourt must succeed — AddCourt is unrelated to the camera
// consent invariant, which only gates AddCameraLink.
func TestAddCourt_SucceedsWithoutCameraConsent(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if f.CameraConsentAttested {
		t.Fatalf("fixture CameraConsentAttested = true, want false")
	}

	if _, err := svc.AddCourt(ctx, f.ID, "owner-1", "Court 1"); err != nil {
		t.Fatalf("AddCourt should succeed regardless of camera consent, got %v", err)
	}
}

// TestAddCourt_RejectsNonOwner is T7.7's app-level proof that AddCourt
// rejects an actor who does not own the target Facility: it fetches the
// Facility first (which it previously never did at all) and calls
// domain.Facility.EnsureOwner before ever touching Repository.AddCourt —
// so a mismatched actorUserID returns domain.ErrNotFacilityOwner and the
// repo's court map is left untouched, not silently populated.
func TestAddCourt_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, err = svc.AddCourt(ctx, f.ID, "attacker", "Court 1")
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotFacilityOwner)
	}
	if len(repo.courts) != 0 {
		t.Fatalf("court must not be persisted for a rejected non-owner AddCourt, repo has %d entries", len(repo.courts))
	}
}

// TestAddCameraLink_RequiresConsent is the app-level proof of the AC from
// HANDOFF.md T7.3 bullet 6: adding a camera link before consent is
// attested must be rejected with domain.ErrCameraConsentRequired (which
// grpcapi maps to a 4xx, not a 500) — and the rejection must happen before
// any repository write, i.e. purely via domain.Facility.AddCameraLink's
// existing invariant (T7.2), not a second ad hoc check.
func TestAddCameraLink_RequiresConsent(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, err = svc.AddCameraLink(ctx, f.ID, "owner-1", "https://example.com/cam1.m3u8")
	if !errors.Is(err, domain.ErrCameraConsentRequired) {
		t.Fatalf("got err %v, want %v", err, domain.ErrCameraConsentRequired)
	}

	stored, err := svc.GetFacility(ctx, f.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(stored.CameraLinks) != 0 {
		t.Fatalf("CameraLinks = %v, want empty after rejected AddCameraLink", stored.CameraLinks)
	}
}

// TestAddCameraLink_SucceedsOnceConsentAttested is the positive path: once
// CameraConsentAttested is true, AddCameraLink persists the link via the
// repository and the caller can observe it on a subsequent GetFacility.
func TestAddCameraLink_SucceedsOnceConsentAttested(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	f.CameraConsentAttested = true
	if _, err := repo.CreateFacility(ctx, f); err != nil {
		t.Fatalf("unexpected err updating fixture consent: %v", err)
	}

	if _, err := svc.AddCameraLink(ctx, f.ID, "owner-1", "https://example.com/cam1.m3u8"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	stored, err := svc.GetFacility(ctx, f.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(stored.CameraLinks) != 1 || stored.CameraLinks[0].URL != "https://example.com/cam1.m3u8" {
		t.Fatalf("CameraLinks = %v, want one link with the added URL", stored.CameraLinks)
	}
}

func TestAddCameraLink_UnknownFacilityReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	_, err := svc.AddCameraLink(ctx, "does-not-exist", "owner-1", "https://example.com/cam1.m3u8")
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrFacilityNotFound)
	}
}

// TestAddCameraLink_RejectsNonOwner is T7.7's app-level proof that
// AddCameraLink rejects an actor who does not own the target Facility —
// even when CameraConsentAttested is already true, so this test can't be
// confused with TestAddCameraLink_RequiresConsent above. The rejected call
// must not persist a link via Repository.AddCameraLink.
func TestAddCameraLink_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	f.CameraConsentAttested = true
	if _, err := repo.CreateFacility(ctx, f); err != nil {
		t.Fatalf("unexpected err updating fixture consent: %v", err)
	}

	_, err = svc.AddCameraLink(ctx, f.ID, "attacker", "https://example.com/cam1.m3u8")
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotFacilityOwner)
	}

	stored, err := svc.GetFacility(ctx, f.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(stored.CameraLinks) != 0 {
		t.Fatalf("CameraLinks = %v, want empty after rejected non-owner AddCameraLink", stored.CameraLinks)
	}
}

// --- T8.4: AttestCameraConsent ---------------------------------------------

// TestAttestCameraConsent_EnablesAddCameraLink is the app-level, end-to-end
// proof this ticket exists for: attesting consent as the owner, then adding
// a camera link, both succeed through the real service — closing the gap
// where every correct client submission to AddCameraLink was rejected
// because nothing could ever set CameraConsentAttested to true.
func TestAttestCameraConsent_EnablesAddCameraLink(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	attested, err := svc.AttestCameraConsent(ctx, f.ID, "owner-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !attested.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = false after AttestCameraConsent, want true")
	}

	stored, err := svc.GetFacility(ctx, f.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !stored.CameraConsentAttested {
		t.Fatalf("persisted CameraConsentAttested = false, want true")
	}

	if _, err := svc.AddCameraLink(ctx, f.ID, "owner-1", "https://example.com/cam1.m3u8"); err != nil {
		t.Fatalf("AddCameraLink after AttestCameraConsent should succeed, got: %v", err)
	}
}

// TestAttestCameraConsent_Idempotent mirrors the domain-level idempotency
// test at the app layer: attesting twice must not error.
func TestAttestCameraConsent_Idempotent(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if _, err := svc.AttestCameraConsent(ctx, f.ID, "owner-1"); err != nil {
		t.Fatalf("first AttestCameraConsent: unexpected err: %v", err)
	}
	if _, err := svc.AttestCameraConsent(ctx, f.ID, "owner-1"); err != nil {
		t.Fatalf("second AttestCameraConsent: unexpected err: %v (want idempotent)", err)
	}
}

// TestAttestCameraConsent_UnknownFacilityReturnsNotFound mirrors
// TestAddCameraLink_UnknownFacilityReturnsNotFound for the new RPC.
func TestAttestCameraConsent_UnknownFacilityReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	_, err := svc.AttestCameraConsent(ctx, "does-not-exist", "owner-1")
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrFacilityNotFound)
	}
}

// TestAttestCameraConsent_RejectsNonOwner is T8.4's app-level proof that
// AttestCameraConsent rejects an actor who does not own the target
// Facility, and does not persist any change to CameraConsentAttested.
func TestAttestCameraConsent_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &sequentialIDs{})
	ctx := context.Background()

	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{OwnerID: "owner-1", Name: "Riverside Courts", Address: "1 St"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, err = svc.AttestCameraConsent(ctx, f.ID, "attacker")
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotFacilityOwner)
	}

	stored, err := svc.GetFacility(ctx, f.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if stored.CameraConsentAttested {
		t.Fatalf("CameraConsentAttested = true after rejected non-owner attempt, want false (untouched)")
	}
}
