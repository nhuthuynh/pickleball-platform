package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// ---------------------------------------------------------------------------
// Fakes — one per port, all in-memory. T9.3 is deliberately app-layer-only
// (no Postgres, no proto), mirroring how T5.3 proved Social Play's identical
// reservation/rollback shape before any adapter existed.
// ---------------------------------------------------------------------------

// fakeReservation is an in-memory port.CourtReservation.
//
// The important design choice here is `active`: it tracks the bookings this
// fake currently *holds*, keyed by booking ID — ReserveCourt inserts,
// ReleaseCourt deletes. That makes "did the rollback actually undo
// everything?" a question about observable state (how many bookings exist
// now) rather than about a call log. The sprint plan's non-functional
// requirement for this ticket is explicit that the partial-state rollback
// test must assert on state, not on a call sequence — a test that only
// checks "ReleaseCourt was called twice" passes just as happily against an
// implementation whose releases all silently failed.
//
// reserveCalls is kept as well, but only for the narrower question the
// state map genuinely cannot answer: "was the port touched at all?" An
// empty `active` is consistent both with "never reserved anything" and with
// "reserved then released", and the venue-validation-ordering test needs to
// distinguish those two.
type fakeReservation struct {
	unavailable map[string]bool // courtID -> already reserved by someone else

	active       map[string]string // bookingID -> courtID currently held
	reserveCalls []string          // courtIDs ReserveCourt was called for, in order
	releaseErr   error             // optional: simulate a rollback call itself failing

	n int
}

func newFakeReservation(unavailableCourts ...string) *fakeReservation {
	f := &fakeReservation{
		unavailable: make(map[string]bool),
		active:      make(map[string]string),
	}
	for _, c := range unavailableCourts {
		f.unavailable[c] = true
	}
	return f
}

func (f *fakeReservation) ReserveCourt(_ context.Context, courtID string, _, _ time.Time, _ string) (string, error) {
	f.reserveCalls = append(f.reserveCalls, courtID)
	if f.unavailable[courtID] {
		return "", domain.ErrCourtUnavailable
	}
	f.n++
	bookingID := fmt.Sprintf("booking-%d", f.n)
	f.active[bookingID] = courtID
	return bookingID, nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, bookingID string) error {
	if f.releaseErr != nil {
		// A failing release leaves the booking held — which is exactly the
		// dangling-Booking residue port.CourtReservation's doc comment calls
		// out as the known cost of best-effort rollback.
		return f.releaseErr
	}
	delete(f.active, bookingID)
	return nil
}

// fakeFacilityLookup is an in-memory port.FacilityLookup: `known` seeds
// which facility IDs "exist" in the (faked) Facilities context.
type fakeFacilityLookup struct {
	known  map[string]bool
	called []string
}

func newFakeFacilityLookup(knownIDs ...string) *fakeFacilityLookup {
	f := &fakeFacilityLookup{known: make(map[string]bool)}
	for _, id := range knownIDs {
		f.known[id] = true
	}
	return f
}

func (f *fakeFacilityLookup) FacilityExists(_ context.Context, facilityID string) error {
	f.called = append(f.called, facilityID)
	if f.known[facilityID] {
		return nil
	}
	return domain.ErrFacilityNotFound
}

// sequentialIDs is a deterministic port.IDGenerator, mirroring the fake of
// the same name in internal/socialplay/app and internal/booking/app.
type sequentialIDs struct{ n int }

func (g *sequentialIDs) NewID() string {
	g.n++
	return fmt.Sprintf("id-%d", g.n)
}

// fakeShareTokens is a deterministic port.ShareTokenGenerator. err lets a
// test simulate the real (T9.5, crypto/rand-backed) generator failing.
type fakeShareTokens struct {
	n   int
	err error
}

func (g *fakeShareTokens) NewShareToken() (string, error) {
	if g.err != nil {
		return "", g.err
	}
	g.n++
	return fmt.Sprintf("token-%d", g.n), nil
}

// fakeRepository is an in-memory port.Repository holding both Competitions
// and their entries — see that port's doc comment for why one interface
// covers both.
type fakeRepository struct {
	competitions map[string]domain.Competition
	entries      map[string]domain.CompetitionEntry

	// competitionOrder/entryOrder track insertion order so the list read
	// paths are deterministic — Go map iteration order is unspecified, and
	// an assertion on a roster or a browse list needs a stable sequence.
	// Mirrors internal/facilities/app/service_test.go's inMemoryRepo.courtOrder.
	competitionOrder []string
	entryOrder       []string

	createErr error // simulate persistence failing after reservations succeeded
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		competitions: make(map[string]domain.Competition),
		entries:      make(map[string]domain.CompetitionEntry),
	}
}

func (r *fakeRepository) Create(_ context.Context, c domain.Competition) (domain.Competition, error) {
	if r.createErr != nil {
		return domain.Competition{}, r.createErr
	}
	if _, exists := r.competitions[c.ID]; !exists {
		r.competitionOrder = append(r.competitionOrder, c.ID)
	}
	r.competitions[c.ID] = c
	return c, nil
}

func (r *fakeRepository) GetByID(_ context.Context, id string) (domain.Competition, error) {
	c, ok := r.competitions[id]
	if !ok {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	return c, nil
}

// GetByShareToken mirrors the real adapter's contract: a linear scan over
// share tokens with NO status filter (a cancelled Competition's link still
// resolves), returning the same unwrapped ErrCompetitionNotFound sentinel
// GetByID returns for every miss — see port.Repository.GetByShareToken for
// why a distinct "malformed token" error would be a security bug rather than
// a nicety.
func (r *fakeRepository) GetByShareToken(_ context.Context, shareToken string) (domain.Competition, error) {
	for _, id := range r.competitionOrder {
		if c := r.competitions[id]; c.ShareToken == shareToken {
			return c, nil
		}
	}
	return domain.Competition{}, domain.ErrCompetitionNotFound
}

func (r *fakeRepository) UpdateStatus(_ context.Context, id string, status domain.Status) (domain.Competition, error) {
	c, ok := r.competitions[id]
	if !ok {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	c.Status = status
	r.competitions[id] = c
	return c, nil
}

func (r *fakeRepository) CreateEntry(_ context.Context, e domain.CompetitionEntry) (domain.CompetitionEntry, error) {
	if _, exists := r.entries[e.ID]; !exists {
		r.entryOrder = append(r.entryOrder, e.ID)
	}
	r.entries[e.ID] = e
	return e, nil
}

func (r *fakeRepository) ListActiveEntriesForCompetition(_ context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	var out []domain.CompetitionEntry
	for _, e := range r.entries {
		if e.CompetitionID != competitionID {
			continue
		}
		if e.Status == domain.EntryStatusCancelled {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// ListEntriesForCompetition returns EVERY entry for competitionID, cancelled
// ones included — the deliberate difference from
// ListActiveEntriesForCompetition above (see port.Repository's doc comments).
// entryOrder gives it a deterministic order, since Go map iteration order is
// unspecified and a roster assertion needs to be stable.
func (r *fakeRepository) ListEntriesForCompetition(_ context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	out := make([]domain.CompetitionEntry, 0)
	for _, id := range r.entryOrder {
		if e := r.entries[id]; e.CompetitionID == competitionID {
			out = append(out, e)
		}
	}
	return out, nil
}

// ListCompetitions mirrors the real Postgres adapter's contract: only
// scheduled Competitions, each paired with its WEIGHTED SpotsLeft. It
// deliberately delegates that computation to domain.SpotsLeft — the same
// function the production read path's SQL mirrors — so this fake can never
// quietly disagree with the rule under test.
func (r *fakeRepository) ListCompetitions(_ context.Context, filter port.CompetitionListingFilter) ([]port.CompetitionListing, error) {
	out := make([]port.CompetitionListing, 0)
	for _, id := range r.competitionOrder {
		c := r.competitions[id]
		if c.Status != domain.StatusScheduled {
			continue
		}
		if filter.VenueFacilityID != "" && c.VenueFacilityID != filter.VenueFacilityID {
			continue
		}
		var entries []domain.CompetitionEntry
		for _, e := range r.entries {
			if e.CompetitionID == c.ID {
				entries = append(entries, e)
			}
		}
		out = append(out, port.CompetitionListing{
			Competition: c,
			SpotsLeft:   domain.SpotsLeft(c, entries),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q in test setup: %v", s, err)
	}
	return parsed
}

func session(t *testing.T, start, end string, courtIDs ...string) domain.Session {
	t.Helper()
	r, err := domain.NewTimeRange(mustTime(t, start), mustTime(t, end))
	if err != nil {
		t.Fatalf("bad range in test setup: %v", err)
	}
	return domain.Session{Range: r, CourtIDs: courtIDs}
}

func validInput(t *testing.T, sessions ...domain.Session) app.ScheduleCompetitionInput {
	t.Helper()
	if len(sessions) == 0 {
		sessions = []domain.Session{session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")}
	}
	return app.ScheduleCompetitionInput{
		HostID:         "host-1",
		Name:           "Spring Doubles Open",
		Sessions:       sessions,
		Capacity:       16,
		GuestAllowance: 1,
		PaymentMethod:  domain.PaymentMethodEither,
		EntryFee:       domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		Format:         domain.FormatDoubles,
	}
}

// newTestService wires a Service from the fakes via the options struct —
// which is the point of ServiceOptions: five dependencies named at the call
// site instead of five positional arguments whose order a reader has to
// look up (HANDOFF.md's Cross-cutting note on socialplayapp.NewService).
func newTestService(repo *fakeRepository, reservation *fakeReservation, facilities *fakeFacilityLookup, tokens *fakeShareTokens) *app.Service {
	return app.NewService(app.ServiceOptions{
		Competitions: repo,
		IDs:          &sequentialIDs{},
		Reservation:  reservation,
		Facilities:   facilities,
		ShareTokens:  tokens,
	})
}

// ---------------------------------------------------------------------------
// ScheduleCompetition
// ---------------------------------------------------------------------------

// TestScheduleCompetition_ReservesEveryCourtInEverySession is the core
// difference from Social Play's ScheduleGame: a Competition reserves across
// sessions × courts, not a single range × courts, so the reservation loop is
// nested and the count that matters is the product.
func TestScheduleCompetition_ReservesEveryCourtInEverySession(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	reservation := newFakeReservation()
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1", "court-2"),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", "court-1", "court-2"),
	)

	c, err := svc.ScheduleCompetition(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(reservation.active) != 4 {
		t.Fatalf("holding %d bookings, want 4 (2 sessions x 2 courts): %v", len(reservation.active), reservation.active)
	}
	if len(repo.competitions) != 1 {
		t.Fatalf("persisted %d competitions, want 1", len(repo.competitions))
	}
	if repo.competitions[c.ID].ID != c.ID {
		t.Fatalf("returned Competition %q is not the persisted one", c.ID)
	}
	if c.Status != domain.StatusScheduled {
		t.Fatalf("Status = %q, want scheduled", c.Status)
	}
}

// TestScheduleCompetition_RollsBackEverySessionWhenLaterCourtUnavailable is
// this ticket's required partial-state proof: two sessions, the SECOND
// session's court already booked. The assertions are deliberately about
// observable state — zero bookings held, zero Competitions persisted — not
// about how many times ReleaseCourt was called, so that an implementation
// that "rolled back" by calling a release that didn't actually free anything
// would still fail this test.
func TestScheduleCompetition_RollsBackEverySessionWhenLaterCourtUnavailable(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	reservation := newFakeReservation("court-2")
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1"),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", "court-2"),
	)

	_, err := svc.ScheduleCompetition(context.Background(), in)
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
	if len(reservation.active) != 0 {
		t.Fatalf("a failed ScheduleCompetition left %d booking(s) behind: %v", len(reservation.active), reservation.active)
	}
	if len(repo.competitions) != 0 {
		t.Fatalf("a failed ScheduleCompetition persisted %d competition(s), want 0", len(repo.competitions))
	}
}

// TestScheduleCompetition_RollsBackInReverseOrder pins the documented
// reverse-order rollback. This one legitimately inspects the order releases
// happened in, because ordering is the behaviour under test and no amount of
// end-state inspection can observe it — the end state (zero held bookings)
// is identical either way. It complements, rather than replaces, the
// state-based proof above.
func TestScheduleCompetition_RollsBackInReverseOrder(t *testing.T) {
	t.Parallel()

	// The recorder wraps the ordinary fake, noting the order ReleaseCourt is
	// called in without changing its state semantics.
	var releaseOrder []string
	recorder := &orderRecordingReservation{inner: newFakeReservation("court-3"), order: &releaseOrder}
	svc := app.NewService(app.ServiceOptions{
		Competitions: newFakeRepository(),
		IDs:          &sequentialIDs{},
		Reservation:  recorder,
		Facilities:   newFakeFacilityLookup(),
		ShareTokens:  &fakeShareTokens{},
	})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1", "court-2"),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", "court-3"),
	)

	if _, err := svc.ScheduleCompetition(context.Background(), in); !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
	want := []string{"booking-2", "booking-1"}
	if len(releaseOrder) != len(want) {
		t.Fatalf("released %v, want %v", releaseOrder, want)
	}
	for i := range want {
		if releaseOrder[i] != want[i] {
			t.Fatalf("released %v, want reverse order %v", releaseOrder, want)
		}
	}
}

// orderRecordingReservation delegates to an inner fake, recording the order
// ReleaseCourt is called in.
type orderRecordingReservation struct {
	inner *fakeReservation
	order *[]string
}

func (r *orderRecordingReservation) ReserveCourt(ctx context.Context, courtID string, start, end time.Time, referenceID string) (string, error) {
	return r.inner.ReserveCourt(ctx, courtID, start, end, referenceID)
}

func (r *orderRecordingReservation) ReleaseCourt(ctx context.Context, bookingID string) error {
	*r.order = append(*r.order, bookingID)
	return r.inner.ReleaseCourt(ctx, bookingID)
}

// TestScheduleCompetition_RollbackFailureDoesNotMaskOriginalError proves the
// documented best-effort semantics: a failing compensating release must not
// replace the conflict the caller actually needs to see.
func TestScheduleCompetition_RollbackFailureDoesNotMaskOriginalError(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation("court-2")
	reservation.releaseErr = errors.New("release: boom")
	svc := newTestService(newFakeRepository(), reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1"),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", "court-2"),
	)

	_, err := svc.ScheduleCompetition(context.Background(), in)
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable even though rollback itself failed", err)
	}
}

// TestScheduleCompetition_UnknownVenueFacilityRejectsBeforeReservingCourts
// is the required venue-validation-ordering proof, mirroring T8.3's
// TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts: an
// unknown venue reserves nothing at all (not "reserves then rolls back"),
// and persists nothing.
func TestScheduleCompetition_UnknownVenueFacilityRejectsBeforeReservingCourts(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	reservation := newFakeReservation()
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t)
	in.VenueFacilityID = "no-such-facility"

	_, err := svc.ScheduleCompetition(context.Background(), in)
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("got err %v, want ErrFacilityNotFound", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("an unknown venue facility must not touch the reservation port, got calls: %v", reservation.reserveCalls)
	}
	if len(reservation.active) != 0 {
		t.Fatalf("an unknown venue facility left %d booking(s) behind", len(reservation.active))
	}
	if len(repo.competitions) != 0 {
		t.Fatalf("an unknown venue facility persisted %d competition(s), want 0", len(repo.competitions))
	}
}

func TestScheduleCompetition_KnownVenueFacilityAccepted(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepository(), newFakeReservation(), newFakeFacilityLookup("facility-1"), &fakeShareTokens{})

	in := validInput(t)
	in.VenueFacilityID = "facility-1"

	c, err := svc.ScheduleCompetition(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.VenueFacilityID != "facility-1" {
		t.Fatalf("VenueFacilityID = %q, want facility-1", c.VenueFacilityID)
	}
}

// TestScheduleCompetition_EmptyVenueFacilitySkipsLookup proves the venue is
// optional (domain.Competition.VenueFacilityID's doc comment): an empty one
// never consults the port at all, so an unseeded fake lookup — which would
// answer ErrFacilityNotFound for anything — doesn't block scheduling.
func TestScheduleCompetition_EmptyVenueFacilitySkipsLookup(t *testing.T) {
	t.Parallel()

	facilities := newFakeFacilityLookup()
	svc := newTestService(newFakeRepository(), newFakeReservation(), facilities, &fakeShareTokens{})

	c, err := svc.ScheduleCompetition(context.Background(), validInput(t))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.VenueFacilityID != "" {
		t.Fatalf("VenueFacilityID = %q, want empty", c.VenueFacilityID)
	}
	if len(facilities.called) != 0 {
		t.Fatalf("an empty VenueFacilityID must not consult the lookup port, got %v", facilities.called)
	}
}

// TestScheduleCompetition_InvalidInputRejectedBeforeTouchingPorts pins the
// step ordering: domain construction (domain.NewCompetition) runs before any
// port is touched, so structurally invalid input costs no I/O and can leave
// no residue. Overlapping sessions on one court is the Competition-specific
// case — T9.1 catches it at construction precisely so this loop never has to
// discover it half-way through and roll back.
func TestScheduleCompetition_InvalidInputRejectedBeforeTouchingPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(in *app.ScheduleCompetitionInput)
		wantErr error
	}{
		{
			name:    "zero capacity",
			mutate:  func(in *app.ScheduleCompetitionInput) { in.Capacity = 0 },
			wantErr: domain.ErrInvalidCapacity,
		},
		{
			name:    "no sessions",
			mutate:  func(in *app.ScheduleCompetitionInput) { in.Sessions = nil },
			wantErr: domain.ErrEmptySessions,
		},
		{
			name:    "invalid format",
			mutate:  func(in *app.ScheduleCompetitionInput) { in.Format = domain.Format("mixed-quads") },
			wantErr: domain.ErrInvalidFormat,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			reservation := newFakeReservation()
			facilities := newFakeFacilityLookup("facility-1")
			svc := newTestService(repo, reservation, facilities, &fakeShareTokens{})

			in := validInput(t)
			in.VenueFacilityID = "facility-1"
			tc.mutate(&in)

			_, err := svc.ScheduleCompetition(context.Background(), in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got err %v, want %v", err, tc.wantErr)
			}
			if len(reservation.reserveCalls) != 0 {
				t.Fatalf("invalid input must not touch the reservation port, got %v", reservation.reserveCalls)
			}
			if len(facilities.called) != 0 {
				t.Fatalf("invalid input must not touch the facility lookup port, got %v", facilities.called)
			}
			if len(repo.competitions) != 0 {
				t.Fatalf("invalid input persisted %d competition(s), want 0", len(repo.competitions))
			}
		})
	}
}

// TestScheduleCompetition_PopulatesShareToken proves the ShareToken is set
// at construction. T9.5 consumes it; introducing the generator now is what
// lets that ticket be a read path plus a real crypto/rand implementation,
// rather than also having to invent a second write path to back-fill tokens
// onto Competitions that were created without one.
func TestScheduleCompetition_PopulatesShareToken(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := newTestService(repo, newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})

	c, err := svc.ScheduleCompetition(context.Background(), validInput(t))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.ShareToken != "token-1" {
		t.Fatalf("ShareToken = %q, want token-1", c.ShareToken)
	}
	if repo.competitions[c.ID].ShareToken != "token-1" {
		t.Fatalf("persisted ShareToken = %q, want token-1", repo.competitions[c.ID].ShareToken)
	}
}

// TestScheduleCompetition_ShareTokenFailureLeavesNoPartialState: the real
// generator (T9.5) reads from crypto/rand and can fail. Same no-partial-state
// rule as every other pre-reservation failure — nothing reserved, nothing
// persisted.
func TestScheduleCompetition_ShareTokenFailureLeavesNoPartialState(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	reservation := newFakeReservation()
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{err: errors.New("entropy: boom")})

	if _, err := svc.ScheduleCompetition(context.Background(), validInput(t)); err == nil {
		t.Fatal("expected an error when the share token generator fails")
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("a share-token failure must not touch the reservation port, got %v", reservation.reserveCalls)
	}
	if len(repo.competitions) != 0 {
		t.Fatalf("a share-token failure persisted %d competition(s), want 0", len(repo.competitions))
	}
}

// TestScheduleCompetition_RollsBackReservationsWhenPersistFails covers the
// second failure point: every court reserved fine, but the write failed. A
// Competition that failed to persist must not leave its Bookings holding
// courts nobody can now see a reason for.
func TestScheduleCompetition_RollsBackReservationsWhenPersistFails(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	repo.createErr = errors.New("db: connection refused")
	reservation := newFakeReservation()
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1", "court-2"),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", "court-1"),
	)

	if _, err := svc.ScheduleCompetition(context.Background(), in); err == nil {
		t.Fatal("expected an error when persistence fails")
	}
	if len(reservation.active) != 0 {
		t.Fatalf("a failed persist left %d booking(s) behind: %v", len(reservation.active), reservation.active)
	}
	if len(repo.competitions) != 0 {
		t.Fatalf("persisted %d competition(s), want 0", len(repo.competitions))
	}
}

// ---------------------------------------------------------------------------
// EnterCompetition
// ---------------------------------------------------------------------------

// scheduleFixture schedules one Competition through the service and returns
// it alongside the repo backing it, so entry tests start from a Competition
// that really went through the full write path.
func scheduleFixture(t *testing.T, capacity, guestAllowance int) (*fakeRepository, *app.Service, domain.Competition) {
	t.Helper()

	repo := newFakeRepository()
	svc := newTestService(repo, newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t)
	in.Capacity = capacity
	in.GuestAllowance = guestAllowance

	c, err := svc.ScheduleCompetition(context.Background(), in)
	if err != nil {
		t.Fatalf("fixture ScheduleCompetition failed: %v", err)
	}
	return repo, svc, c
}

func TestEnterCompetition_PersistsEntry(t *testing.T) {
	t.Parallel()

	repo, svc, c := scheduleFixture(t, 16, 1)

	entry, err := svc.EnterCompetition(context.Background(), app.EnterCompetitionInput{
		CompetitionID: c.ID,
		PlayerID:      "player-1",
		GuestCount:    1,
		Source:        domain.EntrySourceApp,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("entry was persisted without an ID")
	}
	if entry.CompetitionID != c.ID || entry.PlayerID != "player-1" || entry.GuestCount != 1 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("PaymentStatus = %q, want unpaid", entry.PaymentStatus)
	}
	if got, ok := repo.entries[entry.ID]; !ok || got.PlayerID != "player-1" {
		t.Fatalf("entry %q was not persisted", entry.ID)
	}
}

// TestEnterCompetition_AnyPlayerMayEnter documents the deliberate
// authorization asymmetry between this method and CancelCompetition: entering
// is open to any Player, exactly as socialplay's RegisterForGame is, because
// a Competition a Host has published is an invitation. Ownership only gates
// acts *on* the Competition itself (CancelCompetition -> EnsureHost).
func TestEnterCompetition_AnyPlayerMayEnter(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)

	if c.HostID == "player-99" {
		t.Fatal("fixture invalid: the entrant must not be the host")
	}
	if _, err := svc.EnterCompetition(context.Background(), app.EnterCompetitionInput{
		CompetitionID: c.ID,
		PlayerID:      "player-99",
		Source:        domain.EntrySourceApp,
	}); err != nil {
		t.Fatalf("a non-host Player must be able to enter, got err: %v", err)
	}
}

// TestEnterCompetition_WeightedCapacityEnforced proves the app layer really
// routes through domain.Enter's weighted rule: capacity 3, one entry with two
// guests fills it (1 + 2 = 3 places), so the next lone entrant is rejected
// even though only ONE entry exists. A headcount check would wrongly admit
// them.
func TestEnterCompetition_WeightedCapacityEnforced(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 3, 2)
	ctx := context.Background()

	if _, err := svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: c.ID, PlayerID: "player-1", GuestCount: 2, Source: domain.EntrySourceApp,
	}); err != nil {
		t.Fatalf("first entry should fit exactly: %v", err)
	}

	_, err := svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: c.ID, PlayerID: "player-2", Source: domain.EntrySourceApp,
	})
	if !errors.Is(err, domain.ErrCompetitionFull) {
		t.Fatalf("got err %v, want ErrCompetitionFull (1 entry + 2 guests already fills capacity 3)", err)
	}
}

func TestEnterCompetition_CompetitionNotFound(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepository(), newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})

	_, err := svc.EnterCompetition(context.Background(), app.EnterCompetitionInput{
		CompetitionID: "no-such-competition", PlayerID: "player-1", Source: domain.EntrySourceApp,
	})
	if !errors.Is(err, domain.ErrCompetitionNotFound) {
		t.Fatalf("got err %v, want ErrCompetitionNotFound", err)
	}
}

func TestEnterCompetition_CancelledCompetitionRejected(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ctx := context.Background()

	if _, err := svc.CancelCompetition(ctx, c.ID, c.HostID); err != nil {
		t.Fatalf("fixture cancel failed: %v", err)
	}

	_, err := svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: c.ID, PlayerID: "player-1", Source: domain.EntrySourceApp,
	})
	if !errors.Is(err, domain.ErrCompetitionCancelled) {
		t.Fatalf("got err %v, want ErrCompetitionCancelled", err)
	}
}

// ---------------------------------------------------------------------------
// CancelCompetition
// ---------------------------------------------------------------------------

func TestCancelCompetition_HostSucceeds(t *testing.T) {
	t.Parallel()

	repo, svc, c := scheduleFixture(t, 16, 0)

	cancelled, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Status = %q, want cancelled", cancelled.Status)
	}
	if repo.competitions[c.ID].Status != domain.StatusCancelled {
		t.Fatalf("persisted Status = %q, want cancelled", repo.competitions[c.ID].Status)
	}
}

// TestCancelCompetition_NonHostRejected is the required EnsureHost proof.
// The error must be ErrNotCompetitionHost *specifically* — merely asserting
// non-nil would pass against an implementation that rejected the call for
// some entirely unrelated reason (and, downstream, T9.4 maps this exact
// sentinel to PermissionDenied rather than Internal). It also asserts no
// state changed, so a "rejected but cancelled anyway" bug can't hide.
func TestCancelCompetition_NonHostRejected(t *testing.T) {
	t.Parallel()

	repo, svc, c := scheduleFixture(t, 16, 0)

	_, err := svc.CancelCompetition(context.Background(), c.ID, "not-the-host")
	if !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("got err %v, want ErrNotCompetitionHost specifically", err)
	}
	if repo.competitions[c.ID].Status != domain.StatusScheduled {
		t.Fatalf("a rejected cancel changed Status to %q; it must stay scheduled", repo.competitions[c.ID].Status)
	}
}

// TestCancelCompetition_EmptyActorRejected pins domain.EnsureHost's "an
// unidentified caller is never the host" rule at the app boundary.
func TestCancelCompetition_EmptyActorRejected(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)

	if _, err := svc.CancelCompetition(context.Background(), c.ID, ""); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("got err %v, want ErrNotCompetitionHost", err)
	}
}

func TestCancelCompetition_NotFound(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepository(), newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})

	if _, err := svc.CancelCompetition(context.Background(), "no-such-competition", "host-1"); !errors.Is(err, domain.ErrCompetitionNotFound) {
		t.Fatalf("got err %v, want ErrCompetitionNotFound", err)
	}
}

// TestCancelCompetition_AlreadyCancelledRejected mirrors
// booking.Booking.Cancel/socialplay.Game.Cancel: a second cancel is an
// illegal transition, not a silent no-op.
func TestCancelCompetition_AlreadyCancelledRejected(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ctx := context.Background()

	if _, err := svc.CancelCompetition(ctx, c.ID, c.HostID); err != nil {
		t.Fatalf("first cancel failed: %v", err)
	}
	if _, err := svc.CancelCompetition(ctx, c.ID, c.HostID); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want ErrIllegalStatusTransition", err)
	}
}
