package app_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// fakeIdentityLookup implements port.IdentityLookup for every test in this
// package that does NOT itself exercise the identifier-space resolution
// seam (T29.2, closing the Social Play third of #164). It exists solely to
// satisfy NewService's now-required Identity dependency (see
// app.ServiceOptions.Identity's own doc comment for why this package treats
// Identity as required rather than optional/fail-closed, unlike Payments'
// identical-looking field) — no test in this file calls
// app.Service.ResolveActorUserID directly (that seam is exercised by
// internal/socialplay/adapter/grpcapi's own tests, against the real
// adapter/identity.Lookup), so this fake's own resolution rule (a
// deterministic subject == User.ID passthrough) is never actually invoked
// by anything below; it only needs to exist so construction succeeds.
type fakeIdentityLookup struct{}

func (fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	if subject == "" {
		return "", domain.ErrUserNotFound
	}
	return subject, nil
}

// fakeReservation is a minimal in-memory port.CourtReservation fake — no
// real Booking context involved (T5.3 is deliberately app-layer-only, no
// Postgres/proto, mirroring how Booking's own cross-source overlap test was
// proven before any adapter existed). It lets tests seed a specific court as
// already unavailable (simulating an existing overlapping Booking of any
// source) and records enough call history to prove rollback behaviour.
type fakeReservation struct {
	unavailable map[string]bool // courtID -> already reserved by someone else

	reserveCalls []string // courtIDs ReserveCourt was called for, in order
	released     []string // bookingIDs ReleaseCourt was called for, in order
	releaseErr   error    // optional: simulate a rollback call itself failing

	n int
}

func newFakeReservation(unavailableCourts ...string) *fakeReservation {
	f := &fakeReservation{unavailable: make(map[string]bool)}
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
	return fmt.Sprintf("booking-%d", f.n), nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, bookingID string) error {
	f.released = append(f.released, bookingID)
	return f.releaseErr
}

// fakeFacilityLookup is a minimal in-memory port.FacilityLookup fake (T8.3),
// mirroring fakeReservation's shape. known lets a test seed which facility
// IDs "exist" in the (faked) Facilities context; a facility ID absent from
// known is treated as not found. Most ScheduleGame tests don't set
// ScheduleGameInput.VenueFacilityID at all (it's optional — see
// app.Service.ScheduleGame's doc comment), so an empty newFakeFacilityLookup()
// with nothing seeded is the right default: Service skips the lookup
// entirely for an empty VenueFacilityID, so this fake is never actually
// consulted by those tests.
type fakeFacilityLookup struct {
	known map[string]bool
}

func newFakeFacilityLookup(knownIDs ...string) *fakeFacilityLookup {
	f := &fakeFacilityLookup{known: make(map[string]bool)}
	for _, id := range knownIDs {
		f.known[id] = true
	}
	return f
}

func (f *fakeFacilityLookup) FacilityExists(_ context.Context, facilityID string) error {
	if f.known[facilityID] {
		return nil
	}
	return domain.ErrFacilityNotFound
}

// sequentialIDs is a deterministic port.IDGenerator fake, mirroring
// internal/booking/app's test fake of the same shape. Shared by both Game
// and Registration IDs (real deployments use a random UUID generator, see
// internal/platform/idgen) so its prefix is generic rather than "game-".
type sequentialIDs struct{ n int }

// NewID mints UUID-shaped ids (T10.7, closing issue #97). It used to return
// "id-1", "id-2", ... — a shape internal/platform/idgen.UUID never produces
// and the Postgres adapter's mustUUID panics on. This is the exact fixture
// infidelity LESSONS.md's T9 entry names ("g-1" in the same family): T10.7
// adds a uuidShape guard to RegisterForGame/CancelRegistration/JoinWaitlist
// (found by that ticket's required inspection sweep, alongside its six
// named handlers), and every existing happy-path test in this file that
// creates a Game/Registration via this generator and then registers/
// cancels/joins-waitlist against it would otherwise start failing the new
// guard — the same reason internal/booking/app's identical generator was
// fixed by the same ticket. The "b000" group keeps it visually distinct
// from malformed_id_test.go's own gameID() helper ("a000"), though a value
// collision between the two would not actually be a bug (different
// counters, and Game/Registration ids live in unrelated keyspaces here).
func (g *sequentialIDs) NewID() string {
	g.n++
	return fmt.Sprintf("00000000-0000-4000-b000-%012d", g.n)
}

// fakeGameRepository is a minimal in-memory port.GameRepository fake, mirroring
// internal/booking/app's inMemoryRepo. createErr lets a test simulate
// persistence itself failing (e.g. a DB outage) after reservations already
// succeeded, to prove ScheduleGame's rollback-on-persist-failure path.
type fakeGameRepository struct {
	games     map[string]domain.Game
	createErr error
	// registrations backs ListGames' SpotsLeft computation only (T8.9) — a
	// direct field a test can populate, standing in for the real query's
	// LEFT JOIN against the registrations table (this fake has no
	// connection to fakeRegistrationRepository).
	registrations map[string]domain.Registration

	// getByIDCalls counts real invocations of GetByID, so a test can prove
	// a malformed-shape gameID never reaches the repository at all (T10.7,
	// closing issue #97 — RegisterForGame/JoinWaitlist's own guards). See
	// internal/booking/app's identical listActiveForCourtCalls for why
	// atomic.Int64 rather than a plain int.
	getByIDCalls atomic.Int64
}

func newFakeGameRepository() *fakeGameRepository {
	return &fakeGameRepository{games: make(map[string]domain.Game), registrations: make(map[string]domain.Registration)}
}

func (r *fakeGameRepository) Create(_ context.Context, g domain.Game) (domain.Game, error) {
	if r.createErr != nil {
		return domain.Game{}, r.createErr
	}
	r.games[g.ID] = g
	return g, nil
}

func (r *fakeGameRepository) GetByID(_ context.Context, id string) (domain.Game, error) {
	r.getByIDCalls.Add(1)
	g, ok := r.games[id]
	if !ok {
		return domain.Game{}, domain.ErrGameNotFound
	}
	return g, nil
}

// UpdateStatus is the in-memory port.GameRepository.UpdateStatus fake
// (T12.4). Like the real single-column query it stands in for, it writes
// only Status — every other field on the stored Game is left exactly as it
// was, so a test can prove a cancellation didn't quietly clobber anything
// else.
func (r *fakeGameRepository) UpdateStatus(_ context.Context, id string, status domain.Status) (domain.Game, error) {
	g, ok := r.games[id]
	if !ok {
		return domain.Game{}, domain.ErrGameNotFound
	}
	g.Status = status
	r.games[id] = g
	return g, nil
}

// ListGames is a minimal in-memory port.GameRepository.ListGames fake
// (T8.9): filters by VenueFacilityID/StartsAfter/StartsBefore exactly like
// the real sqlc query (db/queries/socialplay.sql's ListGames), and derives
// SpotsLeft from r.registrations using the same (1 + GuestCount) weighting
// domain.Register's own capacity check uses — see
// countActiveRegistrationsWeight's doc comment. Only 'scheduled' Games are
// ever returned, mirroring the real query's WHERE status = 'scheduled'.
// Results are sorted by start time to match the real query's ORDER BY
// starts_at, so callers (and this fake's own tests) can assert on order.
func (r *fakeGameRepository) ListGames(_ context.Context, filter port.GameListingFilter) ([]port.GameListing, error) {
	var out []port.GameListing
	for _, g := range r.games {
		if g.Status != domain.StatusScheduled {
			continue
		}
		if filter.VenueFacilityID != "" && g.VenueFacilityID != filter.VenueFacilityID {
			continue
		}
		if !filter.StartsAfter.IsZero() && g.Range.Start.Before(filter.StartsAfter) {
			continue
		}
		if !filter.StartsBefore.IsZero() && g.Range.Start.After(filter.StartsBefore) {
			continue
		}
		out = append(out, port.GameListing{
			Game:      g,
			SpotsLeft: g.Capacity - countActiveRegistrationsWeight(r.registrations, g.ID),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Game.Range.Start.Before(out[j].Game.Range.Start) })
	return out, nil
}

// countActiveRegistrationsWeight sums (1 + GuestCount) across gameID's
// active (non-cancelled) registrations — the same weighting
// domain.Register's own capacity check uses (registration.go's
// countActiveRegistrations), reimplemented here since this fake has no
// access to the domain package's unexported helper.
func countActiveRegistrationsWeight(registrations map[string]domain.Registration, gameID string) int {
	weight := 0
	for _, reg := range registrations {
		if reg.GameID != gameID || reg.Status == domain.RegistrationStatusCancelled {
			continue
		}
		weight += 1 + reg.GuestCount
	}
	return weight
}

// fakeRegistrationRepository is a minimal in-memory port.RegistrationRepository
// fake.
type fakeRegistrationRepository struct {
	registrations map[string]domain.Registration

	// getByIDCalls counts real invocations of GetByID, so a test can prove
	// a malformed-shape registrationID never reaches the repository at all
	// (T10.7, closing issue #97 — CancelRegistration's own guard).
	getByIDCalls atomic.Int64
}

func newFakeRegistrationRepository() *fakeRegistrationRepository {
	return &fakeRegistrationRepository{registrations: make(map[string]domain.Registration)}
}

func (r *fakeRegistrationRepository) Create(_ context.Context, reg domain.Registration) (domain.Registration, error) {
	for _, existing := range r.registrations {
		if existing.GameID == reg.GameID && existing.PlayerID == reg.PlayerID && existing.Status != domain.RegistrationStatusCancelled {
			return domain.Registration{}, domain.ErrAlreadyRegistered
		}
	}
	r.registrations[reg.ID] = reg
	return reg, nil
}

func (r *fakeRegistrationRepository) GetByID(_ context.Context, id string) (domain.Registration, error) {
	r.getByIDCalls.Add(1)
	reg, ok := r.registrations[id]
	if !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	return reg, nil
}

func (r *fakeRegistrationRepository) ListActiveForGame(_ context.Context, gameID string) ([]domain.Registration, error) {
	var out []domain.Registration
	for _, reg := range r.registrations {
		if reg.GameID != gameID || reg.Status == domain.RegistrationStatusCancelled {
			continue
		}
		out = append(out, reg)
	}
	return out, nil
}

func (r *fakeRegistrationRepository) Update(_ context.Context, reg domain.Registration) (domain.Registration, error) {
	if _, ok := r.registrations[reg.ID]; !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	r.registrations[reg.ID] = reg
	return reg, nil
}

func (r *fakeRegistrationRepository) UpdatePaymentStatus(_ context.Context, id string, status domain.PaymentStatus) (domain.Registration, error) {
	reg, ok := r.registrations[id]
	if !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	reg.PaymentStatus = status
	r.registrations[id] = reg
	return reg, nil
}

// CancelAllActiveForGame mirrors the real Postgres adapter's contract
// (T16.3): bulk-cancels every non-cancelled registration scoped to gameID
// and returns the count actually transitioned. An already-cancelled
// registration is left untouched and not counted, mirroring the real
// query's `WHERE ... AND status <> 'cancelled'` clause — this fake would
// otherwise silently disagree with the production adapter about what
// "actually transitioned" means.
func (r *fakeRegistrationRepository) CancelAllActiveForGame(_ context.Context, gameID string) (int, error) {
	n := 0
	for id, reg := range r.registrations {
		if reg.GameID != gameID || reg.Status == domain.RegistrationStatusCancelled {
			continue
		}
		reg.Status = domain.RegistrationStatusCancelled
		r.registrations[id] = reg
		n++
	}
	return n, nil
}

// fakeWaitlistRepository is a minimal in-memory port.WaitlistRepository
// fake, mirroring fakeRegistrationRepository's shape. PromoteNext and
// ExpirePromotion are implemented as straightforward in-memory
// compare-and-swap operations (no concurrency guarantees are needed here —
// this fake backs single-goroutine app-layer tests only; the real
// concurrency proof lives in internal/socialplay/adapter/postgres against a
// real Postgres, per the T6.6 ticket's own DB-level requirement).
type fakeWaitlistRepository struct {
	entries map[string]domain.WaitlistEntry
}

func newFakeWaitlistRepository() *fakeWaitlistRepository {
	return &fakeWaitlistRepository{entries: make(map[string]domain.WaitlistEntry)}
}

func (r *fakeWaitlistRepository) Create(_ context.Context, e domain.WaitlistEntry) (domain.WaitlistEntry, error) {
	for _, existing := range r.entries {
		if existing.GameID == e.GameID && existing.PlayerID == e.PlayerID &&
			(existing.Status == domain.WaitlistStatusWaiting || existing.Status == domain.WaitlistStatusPromoted) {
			return domain.WaitlistEntry{}, domain.ErrAlreadyOnWaitlist
		}
	}
	r.entries[e.ID] = e
	return e, nil
}

func (r *fakeWaitlistRepository) GetByID(_ context.Context, id string) (domain.WaitlistEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return domain.WaitlistEntry{}, domain.ErrWaitlistEntryNotFound
	}
	return e, nil
}

func (r *fakeWaitlistRepository) ListForGame(_ context.Context, gameID string) ([]domain.WaitlistEntry, error) {
	var out []domain.WaitlistEntry
	for _, e := range r.entries {
		if e.GameID == gameID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (r *fakeWaitlistRepository) PromoteNext(_ context.Context, gameID string, now time.Time) (domain.WaitlistEntry, error) {
	var oldest *domain.WaitlistEntry
	for id, e := range r.entries {
		if e.GameID != gameID || e.Status != domain.WaitlistStatusWaiting {
			continue
		}
		if oldest == nil || e.Position < oldest.Position {
			cp := r.entries[id]
			oldest = &cp
		}
	}
	if oldest == nil {
		return domain.WaitlistEntry{}, domain.ErrNoWaitingEntries
	}
	if err := oldest.Promote(now); err != nil {
		return domain.WaitlistEntry{}, err
	}
	r.entries[oldest.ID] = *oldest
	return *oldest, nil
}

func (r *fakeWaitlistRepository) ExpirePromotion(_ context.Context, id string, now time.Time) (domain.WaitlistEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return domain.WaitlistEntry{}, domain.ErrWaitlistEntryNotFound
	}
	if err := e.Expire(); err != nil {
		return domain.WaitlistEntry{}, err
	}
	r.entries[id] = e
	return e, nil
}

// fakeMatchRepository is a minimal in-memory port.MatchRepository fake
// (T10.4), mirroring fakeWaitlistRepository's shape.
type fakeMatchRepository struct {
	matches map[string]domain.Match

	// createCalls counts real invocations of Create, so a test can prove a
	// rejected RecordMatchResult call (malformed GameID, unauthorized actor,
	// cancelled Game, or invalid domain input) never reaches the repository
	// at all — mirrors fakeGameRepository.getByIDCalls' identical role for
	// T10.7's guard tests. atomic.Int64 for the same reason those use it:
	// tests in this file run t.Parallel().
	createCalls atomic.Int64
}

func newFakeMatchRepository() *fakeMatchRepository {
	return &fakeMatchRepository{matches: make(map[string]domain.Match)}
}

func (r *fakeMatchRepository) Create(_ context.Context, m domain.Match) (domain.Match, error) {
	r.createCalls.Add(1)
	r.matches[m.ID] = m
	return m, nil
}

func (r *fakeMatchRepository) ListForGame(_ context.Context, gameID string) ([]domain.Match, error) {
	var out []domain.Match
	for _, m := range r.matches {
		if m.GameID == gameID {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.Before(out[j].RecordedAt) })
	return out, nil
}

// fakeGameAdminRepository is a minimal in-memory port.GameAdminRepository fake
// (T14.4), mirroring fakeMatchRepository's shape.
//
// It enforces the composite (game_id, user_id) uniqueness the real
// game_admins primary key does, so the CLAUDE.md-rule-4 relationship between
// domain.AssignGameAdmin's pre-check and the authoritative DB constraint is
// represented here rather than only in Postgres — otherwise a test could pass
// against a fake that silently accepted the duplicate the real store rejects.
type fakeGameAdminRepository struct {
	admins map[string][]domain.GameAdmin
}

func newFakeGameAdminRepository() *fakeGameAdminRepository {
	return &fakeGameAdminRepository{admins: make(map[string][]domain.GameAdmin)}
}

func (r *fakeGameAdminRepository) Assign(_ context.Context, a domain.GameAdmin) (domain.GameAdmin, error) {
	for _, existing := range r.admins[a.GameID] {
		if existing.UserID == a.UserID {
			return domain.GameAdmin{}, domain.ErrAlreadyGameAdmin
		}
	}
	r.admins[a.GameID] = append(r.admins[a.GameID], a)
	return a, nil
}

func (r *fakeGameAdminRepository) Revoke(_ context.Context, gameID, userID string) error {
	for i, existing := range r.admins[gameID] {
		if existing.UserID == userID {
			r.admins[gameID] = append(r.admins[gameID][:i], r.admins[gameID][i+1:]...)
			return nil
		}
	}
	return domain.ErrGameAdminNotFound
}

func (r *fakeGameAdminRepository) ListGameAdmins(_ context.Context, gameID string) ([]domain.GameAdmin, error) {
	out := make([]domain.GameAdmin, len(r.admins[gameID]))
	copy(out, r.admins[gameID])
	sort.Slice(out, func(i, j int) bool { return out[i].AssignedAt.Before(out[j].AssignedAt) })
	return out, nil
}

func mustRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatalf("bad fixture start: %v", err)
	}
	e, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatalf("bad fixture end: %v", err)
	}
	r, err := domain.NewTimeRange(s, e)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}

// validInput's PaymentMethod is explicitly PaymentMethodEither (T8.7) —
// mirrors the resolved value internal/socialplay/adapter/grpcapi.
// fromProtoPaymentMethod produces for a request that never sets
// payment_method, so these app-layer tests exercise ScheduleGame with the
// same effective input a real unset-field request would produce. GuestAllowance
// stays at its zero value (no guests) unless a specific test overrides it.
func validInput(courtIDs ...string) app.ScheduleGameInput {
	return app.ScheduleGameInput{
		HostID:        "host-1",
		FacilityID:    "facility-1",
		CourtIDs:      courtIDs,
		Capacity:      4,
		PaymentMethod: domain.PaymentMethodEither,
	}
}

// TestScheduleGame_ReservesEveryCourt proves the happy path: a multi-court
// Game reserves one Booking per court, using the game's ID as the
// referenceID passed to the port.
func TestScheduleGame_ReservesEveryCourt(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1), courtID(2))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	g, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.Status != domain.StatusScheduled {
		t.Fatalf("status = %v, want scheduled", g.Status)
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("ReserveCourt called %d times, want 2", len(reservation.reserveCalls))
	}
	if len(reservation.released) != 0 {
		t.Fatalf("no rollback expected on the happy path, got releases: %v", reservation.released)
	}
}

// TestScheduleGame_RejectsCourtAlreadyReserved is the literal HANDOFF AC:
// a Game cannot be scheduled onto a court the port reports as already
// reserved (standing in for a confirmed Booking of any source — individual,
// recurring hire, competition, or another game; the fake doesn't need to
// model all four, it just needs to prove the conflict propagates through
// ScheduleGame's returned error unchanged).
func TestScheduleGame_RejectsCourtAlreadyReserved(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation(courtID(1))
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
}

// TestScheduleGame_RollsBackEarlierCourtsOnConflict proves the no-half-
// scheduled-Game requirement: when court 2 of 2 conflicts, court 1's
// reservation must not be left dangling. This is the chosen resolution to
// the multi-court-atomicity question (rollback via best-effort compensating
// ReleaseCourt calls) rather than restricting Games to a single court — see
// the PR description for why.
func TestScheduleGame_RollsBackEarlierCourtsOnConflict(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation(courtID(2))
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1), courtID(2))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("ReserveCourt called %d times, want 2 (court-1 then court-2)", len(reservation.reserveCalls))
	}
	if len(reservation.released) != 1 || reservation.released[0] != "booking-1" {
		t.Fatalf("expected court-1's reservation (booking-1) to be released, got %v", reservation.released)
	}
}

// TestScheduleGame_RollbackFailureDoesNotMaskOriginalError proves the
// documented best-effort semantics: even if the compensating ReleaseCourt
// call itself fails, ScheduleGame still surfaces the original conflict
// error rather than a rollback-specific one.
func TestScheduleGame_RollbackFailureDoesNotMaskOriginalError(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation(courtID(2))
	reservation.releaseErr = errors.New("release: boom")
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1), courtID(2))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable even though rollback itself failed", err)
	}
	if len(reservation.released) != 1 {
		t.Fatalf("rollback should still be attempted once, got %v", reservation.released)
	}
}

// TestScheduleGame_InvalidInputRejectedBeforeTouchingPort proves domain
// validation (T5.1's NewGame) runs before any court is reserved — mirroring
// TestCreateBooking_InvalidSourceRejectedBeforeTouchingRepo's style in
// internal/booking/app.
func TestScheduleGame_InvalidInputRejectedBeforeTouchingPort(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	in.Capacity = 0

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrInvalidCapacity) {
		t.Fatalf("got err %v, want ErrInvalidCapacity", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("invalid Game must not touch the reservation port, got calls: %v", reservation.reserveCalls)
	}
}

// TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts is
// T8.3's core proof: a VenueFacilityID that the Facilities context doesn't
// recognise is rejected with domain.ErrFacilityNotFound, and — same
// no-partial-state requirement as
// TestScheduleGame_InvalidInputRejectedBeforeTouchingPort above — no court
// reservation is attempted and no Game is persisted, so a bogus
// VenueFacilityID can never leave a dangling Booking or Game behind.
func TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	in.VenueFacilityID = "no-such-facility"

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("got err %v, want ErrFacilityNotFound", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("an unknown venue facility must not touch the reservation port, got calls: %v", reservation.reserveCalls)
	}
	if len(games.games) != 0 {
		t.Fatalf("an unknown venue facility must not leave a persisted Game behind, got %d", len(games.games))
	}
}

// TestScheduleGame_KnownVenueFacilityAccepted proves the happy path: a
// VenueFacilityID the fake FacilityLookup recognises does not block
// scheduling, and the Game returned carries it through.
func TestScheduleGame_KnownVenueFacilityAccepted(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	in.VenueFacilityID = "facility-1"

	g, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup("facility-1"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.VenueFacilityID != "facility-1" {
		t.Fatalf("VenueFacilityID = %q, want facility-1", g.VenueFacilityID)
	}
}

// TestScheduleGame_EmptyVenueFacilitySkipsLookup proves VenueFacilityID is
// optional (the migration adds venue_facility_id as a nullable column,
// db/migrations/0011_socialplay_facility_fk.sql, precisely so existing/
// legacy Games aren't forced to have one): an empty VenueFacilityID never
// even calls the FacilityLookup port, so a Game can still be scheduled
// without one.
func TestScheduleGame_EmptyVenueFacilitySkipsLookup(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	// newFakeFacilityLookup() with no known IDs seeded: if ScheduleGame
	// called it for an empty VenueFacilityID, this would fail with
	// ErrFacilityNotFound.
	g, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.VenueFacilityID != "" {
		t.Fatalf("VenueFacilityID = %q, want empty", g.VenueFacilityID)
	}
}

// TestScheduleGame_PersistsGame is T5.4's new coverage: proves ScheduleGame
// actually persists the Game via GameRepository (T5.3's original version
// never touched a repository at all), and that the persisted Game is what's
// returned.
func TestScheduleGame_PersistsGame(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	g, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	stored, err := games.GetByID(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("game should be persisted, GetByID err: %v", err)
	}
	if stored.ID != g.ID || stored.Status != domain.StatusScheduled {
		t.Fatalf("stored game = %+v, want a match for returned game %+v", stored, g)
	}
}

// TestScheduleGame_RollsBackReservationsWhenPersistFails proves the T5.4
// addition to the "no half-scheduled Game" requirement: if every court
// reserves successfully but persisting the Game itself fails (e.g. a DB
// outage), the reservations already made must not be left dangling.
func TestScheduleGame_RollsBackReservationsWhenPersistFails(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	games := newFakeGameRepository()
	games.createErr = errors.New("persist: boom")
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	in := validInput(courtID(1), courtID(2))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if err == nil {
		t.Fatalf("expected an error when persistence fails")
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("both courts should have been reserved before persistence ran, got %v", reservation.reserveCalls)
	}
	if len(reservation.released) != 2 {
		t.Fatalf("both reservations should be rolled back when persistence fails, got %v", reservation.released)
	}
}

// TestRegisterForGame_Valid proves the happy path end-to-end through the
// app layer: a registration is looked up against the persisted Game and its
// active registrations, then itself persisted.
func TestRegisterForGame_Valid(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reg.ID == "" {
		t.Fatalf("registration should have a generated ID")
	}
	if reg.GameID != g.ID || reg.PlayerID != "player-1" {
		t.Fatalf("unexpected registration: %+v", reg)
	}

	stored, err := registrations.GetByID(ctx, reg.ID)
	if err != nil {
		t.Fatalf("registration should be persisted, GetByID err: %v", err)
	}
	if stored.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("stored status = %v, want registered", stored.Status)
	}
}

// TestRegisterForGame_GameFull is the app-level proof of T5.2's capacity
// invariant, now running through real persistence instead of a hand-built
// []domain.Registration slice.
func TestRegisterForGame_GameFull(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	in := validInput(courtID(1))
	in.Capacity = 1
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	if _, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"}); err != nil {
		t.Fatalf("first registration should succeed, got %v", err)
	}

	_, err = svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-2"})
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("got err %v, want ErrGameFull", err)
	}
}

// TestRegisterForGame_GameNotFound proves a bogus GameID surfaces
// domain.ErrGameNotFound rather than a generic/opaque error.
func TestRegisterForGame_GameNotFound(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	_, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{GameID: "no-such-game", PlayerID: "player-1"})
	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want ErrGameNotFound", err)
	}
}

// TestRegisterForGame_RejectsCancelledGame is T19.1's app-level proof
// (closes #212): a Player calling RegisterForGame against an already
// cancelled Game gets domain.ErrGameCancelled, propagated bare from
// domain.Register through app.Service.RegisterForGame unchanged.
func TestRegisterForGame_RejectsCancelledGame(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	if _, err := svc.CancelGame(ctx, g.ID, "host-1"); err != nil {
		t.Fatalf("fixture game should cancel, got %v", err)
	}

	_, err = svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"})
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want ErrGameCancelled", err)
	}
}

// TestCancelRegistration_OwnerSucceeds proves the happy path: the owning
// player can cancel their own registration, and the persisted status
// actually flips (not just the in-memory value returned).
func TestCancelRegistration_OwnerSucceeds(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-a"})
	if err != nil {
		t.Fatalf("fixture registration should succeed, got %v", err)
	}

	cancelled, err := svc.CancelRegistration(ctx, reg.ID, "player-a")
	if err != nil {
		t.Fatalf("owner cancel should succeed, got %v", err)
	}
	if cancelled.Status != domain.RegistrationStatusCancelled {
		t.Fatalf("status = %v, want cancelled", cancelled.Status)
	}

	stored, err := registrations.GetByID(ctx, reg.ID)
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	if stored.Status != domain.RegistrationStatusCancelled {
		t.Fatalf("persisted status = %v, want cancelled", stored.Status)
	}
}

// TestCancelRegistration_WrongActorRejected is the app-level BOLA regression
// (P1 #6 / T5.5 groundwork, proven here through the full app-layer use case
// rather than only the domain method): Player A cannot cancel Player B's
// registration, and the rejection is the distinct ErrNotRegistrationOwner,
// not a silent success or a generic error.
func TestCancelRegistration_WrongActorRejected(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-b"})
	if err != nil {
		t.Fatalf("fixture registration should succeed, got %v", err)
	}

	_, err = svc.CancelRegistration(ctx, reg.ID, "player-a")
	if !errors.Is(err, domain.ErrNotRegistrationOwner) {
		t.Fatalf("got err %v, want ErrNotRegistrationOwner", err)
	}

	stored, getErr := registrations.GetByID(ctx, reg.ID)
	if getErr != nil {
		t.Fatalf("GetByID err: %v", getErr)
	}
	if stored.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("registration must be untouched by the rejected cancel, status = %v", stored.Status)
	}
}

// --- T6.5: MarkRegistrationPaymentStatus (port.RegistrationPaymentUpdater's
// real implementation calls this) --------------------------------------

// TestMarkRegistrationPaymentStatus_Succeeds proves the happy path: an
// existing Registration's PaymentStatus is updated and the change is
// actually persisted (not just returned in-memory) — mirrors
// TestCancelRegistration_OwnerSucceeds's "prove it via a fresh GetByID"
// standard.
func TestMarkRegistrationPaymentStatus_Succeeds(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-a"})
	if err != nil {
		t.Fatalf("fixture registration should succeed, got %v", err)
	}
	if reg.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("fixture PaymentStatus = %v, want unpaid", reg.PaymentStatus)
	}

	if err := svc.MarkRegistrationPaymentStatus(ctx, reg.ID, domain.PaymentStatusPaid); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	stored, err := registrations.GetByID(ctx, reg.ID)
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	if stored.PaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("persisted PaymentStatus = %v, want paid", stored.PaymentStatus)
	}
}

// TestMarkRegistrationPaymentStatus_NotFound proves a bogus registration id
// surfaces domain.ErrRegistrationNotFound, the sentinel
// port.RegistrationPaymentUpdater's doc comment promises callers.
func TestMarkRegistrationPaymentStatus_NotFound(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	err := svc.MarkRegistrationPaymentStatus(context.Background(), "no-such-registration", domain.PaymentStatusPaid)
	if !errors.Is(err, domain.ErrRegistrationNotFound) {
		t.Fatalf("got err %v, want ErrRegistrationNotFound", err)
	}
}

// TestMarkRegistrationPaymentStatus_InvalidStatusRejected proves the
// domain's closed-enum guard (Registration.MarkPaymentStatus) is actually
// wired, not bypassed, at the app layer.
func TestMarkRegistrationPaymentStatus_InvalidStatusRejected(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-a"})
	if err != nil {
		t.Fatalf("fixture registration should succeed, got %v", err)
	}

	err = svc.MarkRegistrationPaymentStatus(ctx, reg.ID, domain.PaymentStatus("not-a-real-status"))
	if !errors.Is(err, domain.ErrInvalidPaymentStatus) {
		t.Fatalf("got err %v, want ErrInvalidPaymentStatus", err)
	}

	stored, getErr := registrations.GetByID(ctx, reg.ID)
	if getErr != nil {
		t.Fatalf("GetByID err: %v", getErr)
	}
	if stored.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("registration must be untouched by the rejected update, PaymentStatus = %v", stored.PaymentStatus)
	}
}

// --- T6.6: waitlist ---------------------------------------------------------

// fixtureFullGame schedules and fills a Game of the given capacity, returning
// it plus the fixture player IDs occupying every slot, so waitlist tests can
// start from "the game is definitely full" without repeating the setup.
func fixtureFullGame(t *testing.T, ctx context.Context, svc *app.Service, capacity int) domain.Game {
	t.Helper()
	in := validInput(courtID(1))
	in.Capacity = capacity
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	for i := 0; i < capacity; i++ {
		playerID := fmt.Sprintf("filler-%d", i)
		if _, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: playerID}); err != nil {
			t.Fatalf("fixture filler registration %d should succeed, got %v", i, err)
		}
	}
	return g
}

// TestJoinWaitlist_Valid proves the happy path through the app layer: a full
// Game accepts a waitlist join, persisted with a generated ID.
func TestJoinWaitlist_Valid(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	entry, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-waiting"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry.ID == "" {
		t.Fatalf("waitlist entry should have a generated ID")
	}
	if entry.Status != domain.WaitlistStatusWaiting {
		t.Fatalf("Status = %v, want waiting", entry.Status)
	}

	stored, err := waitlist.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("entry should be persisted, GetByID err: %v", err)
	}
	if stored.PlayerID != "player-waiting" {
		t.Fatalf("stored entry = %+v, want PlayerID player-waiting", stored)
	}
}

// TestJoinWaitlist_GameNotFull proves the app layer surfaces
// domain.ErrGameNotFull for a Game that still has an open slot, mirroring
// the domain-level test but through the full use case (repository round
// trips included).
func TestJoinWaitlist_GameNotFull(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	in := validInput(courtID(1))
	in.Capacity = 4
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	_, err = svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-1"})
	if !errors.Is(err, domain.ErrGameNotFull) {
		t.Fatalf("got err %v, want ErrGameNotFull", err)
	}
}

// TestJoinWaitlist_RejectsCancelledGame is T19.1's app-level proof (closes
// #212): a Player calling JoinWaitlist against an already cancelled Game
// gets domain.ErrGameCancelled, propagated bare from domain.JoinWaitlist
// through app.Service.JoinWaitlist unchanged.
func TestJoinWaitlist_RejectsCancelledGame(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	if _, err := svc.CancelGame(ctx, g.ID, "host-1"); err != nil {
		t.Fatalf("fixture game should cancel, got %v", err)
	}

	_, err = svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-1"})
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want ErrGameCancelled", err)
	}
}

// TestCancelRegistration_PromotesOldestWaiting is the auto-promotion
// requirement's core proof: cancelling a Registration on a full Game
// promotes the oldest waiting entry, and that entry's promotion is
// persisted (not just returned).
func TestCancelRegistration_PromotesOldestWaiting(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	first, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-first"})
	if err != nil {
		t.Fatalf("first join should succeed, got %v", err)
	}
	second, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-second"})
	if err != nil {
		t.Fatalf("second join should succeed, got %v", err)
	}
	if first.Position >= second.Position {
		t.Fatalf("fixture setup bug: first.Position (%d) should be < second.Position (%d)", first.Position, second.Position)
	}

	all, err := registrations.ListActiveForGame(ctx, g.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("fixture setup bug: expected exactly 1 active registration, got %d (err %v)", len(all), err)
	}
	var fillerRegID string
	for id := range registrations.registrations {
		fillerRegID = id
	}

	if _, err := svc.CancelRegistration(ctx, fillerRegID, "filler-0"); err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}

	storedFirst, err := waitlist.GetByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetByID(first) err: %v", err)
	}
	if storedFirst.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("first (oldest) entry Status = %v, want promoted", storedFirst.Status)
	}
	if storedFirst.PromotedAt.IsZero() {
		t.Fatalf("first entry PromotedAt should be set")
	}

	storedSecond, err := waitlist.GetByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("GetByID(second) err: %v", err)
	}
	if storedSecond.Status != domain.WaitlistStatusWaiting {
		t.Fatalf("second entry Status = %v, want still waiting (only one slot freed)", storedSecond.Status)
	}
}

// TestCancelRegistration_NoWaitlistIsNotAnError proves cancelling on a Game
// with an empty waitlist is not treated as a failure — domain.ErrNoWaitingEntries
// must be swallowed, not propagated as an error from CancelRegistration.
func TestCancelRegistration_NoWaitlistIsNotAnError(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-a"})
	if err != nil {
		t.Fatalf("fixture registration should succeed, got %v", err)
	}

	if _, err := svc.CancelRegistration(ctx, reg.ID, "player-a"); err != nil {
		t.Fatalf("cancel on a game with no waitlist should succeed cleanly, got %v", err)
	}
}

// TestRegisterForGame_UnexpiredPromotionReservesSlot is the ticket's
// required "confirm-then-register-fails-full" proof: while a promoted
// entry's response window is still open, a *different* (non-promoted)
// player's direct RegisterForGame call for the freed slot fails with
// ErrGameFull, but the promoted player's own call succeeds (that is how a
// promotion gets "confirmed" — see RegisterForGame's doc comment).
func TestRegisterForGame_UnexpiredPromotionReservesSlot(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	promotedEntry, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-promoted"})
	if err != nil {
		t.Fatalf("join should succeed, got %v", err)
	}

	var fillerRegID string
	for id := range registrations.registrations {
		fillerRegID = id
	}
	if _, err := svc.CancelRegistration(ctx, fillerRegID, "filler-0"); err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}

	stored, err := waitlist.GetByID(ctx, promotedEntry.ID)
	if err != nil || stored.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("fixture setup bug: entry should now be promoted, got %+v (err %v)", stored, err)
	}

	// A different, non-waitlisted player tries to take the freed slot
	// directly -- must be rejected while the promotion window is open.
	_, err = svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-other"})
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("other player's direct register got err %v, want ErrGameFull (slot should be reserved for the promoted player)", err)
	}

	// The promoted player's own call succeeds -- this is the "confirm".
	confirmed, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-promoted"})
	if err != nil {
		t.Fatalf("promoted player's own register should succeed, got %v", err)
	}
	if confirmed.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("confirmed.Status = %v, want registered", confirmed.Status)
	}
}

// TestRegisterForGame_ExpiredPromotionDoesNotBlock proves an expired
// promotion does not reserve the slot for anyone -- a different, unrelated
// player can register directly once the response window has elapsed. The
// fake repository's PromotedAt is set directly to a time far enough in the
// past that HasExpired is true, since app.Service itself uses the real wall
// clock (no injected Clock port for this ticket -- see the PR description).
func TestRegisterForGame_ExpiredPromotionDoesNotBlock(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	promotedEntry, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-promoted"})
	if err != nil {
		t.Fatalf("join should succeed, got %v", err)
	}

	var fillerRegID string
	for id := range registrations.registrations {
		fillerRegID = id
	}
	if _, err := svc.CancelRegistration(ctx, fillerRegID, "filler-0"); err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}

	// Force the promotion's PromotedAt far enough into the past that its
	// response window has already elapsed relative to real time.Now().
	stored, err := waitlist.GetByID(ctx, promotedEntry.ID)
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	stored.PromotedAt = time.Now().Add(-2 * domain.PromotionResponseWindow)
	waitlist.entries[stored.ID] = stored

	// A different player's direct register must now succeed -- the expired
	// promotion reserves the slot for nobody.
	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-other"})
	if err != nil {
		t.Fatalf("other player's direct register should succeed once the promotion has expired, got %v", err)
	}
	if reg.PlayerID != "player-other" {
		t.Fatalf("reg.PlayerID = %q, want player-other", reg.PlayerID)
	}
}

// TestExpireWaitlistPromotion_CascadesToNext is the required cascade proof:
// player 1's promotion expires, and player 2 (the next waiting entry) gets
// promoted in their place, not left stuck.
func TestExpireWaitlistPromotion_CascadesToNext(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	first, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-first"})
	if err != nil {
		t.Fatalf("first join should succeed, got %v", err)
	}
	second, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-second"})
	if err != nil {
		t.Fatalf("second join should succeed, got %v", err)
	}

	var fillerRegID string
	for id := range registrations.registrations {
		fillerRegID = id
	}
	if _, err := svc.CancelRegistration(ctx, fillerRegID, "filler-0"); err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}

	// player-first is now promoted. Force its PromotedAt into the past so
	// ExpireWaitlistPromotion sees an actually-elapsed window.
	stored, err := waitlist.GetByID(ctx, first.ID)
	if err != nil || stored.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("fixture setup bug: first entry should be promoted, got %+v (err %v)", stored, err)
	}
	stored.PromotedAt = time.Now().Add(-2 * domain.PromotionResponseWindow)
	waitlist.entries[stored.ID] = stored

	expired, err := svc.ExpireWaitlistPromotion(ctx, first.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if expired.Status != domain.WaitlistStatusExpired {
		t.Fatalf("expired.Status = %v, want expired", expired.Status)
	}

	storedSecond, err := waitlist.GetByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("GetByID(second) err: %v", err)
	}
	if storedSecond.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("second entry Status = %v, want promoted (cascade should have skipped the expired first entry)", storedSecond.Status)
	}
	if storedSecond.PromotedAt.IsZero() {
		t.Fatalf("second entry PromotedAt should be set")
	}
}

// TestExpireWaitlistPromotion_RejectsPremature proves calling
// ExpireWaitlistPromotion before the response window has actually elapsed
// is rejected, not silently honoured.
func TestExpireWaitlistPromotion_RejectsPremature(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	entry, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-first"})
	if err != nil {
		t.Fatalf("join should succeed, got %v", err)
	}

	var fillerRegID string
	for id := range registrations.registrations {
		fillerRegID = id
	}
	if _, err := svc.CancelRegistration(ctx, fillerRegID, "filler-0"); err != nil {
		t.Fatalf("cancel should succeed, got %v", err)
	}
	// entry is now promoted, PromotedAt == time.Now() (just now) -- its
	// window has not elapsed.

	_, err = svc.ExpireWaitlistPromotion(ctx, entry.ID)
	if !errors.Is(err, domain.ErrWaitlistPromotionNotExpired) {
		t.Fatalf("got err %v, want ErrWaitlistPromotionNotExpired", err)
	}

	stored, err := waitlist.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	if stored.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("status = %v, want unchanged (still promoted)", stored.Status)
	}
}

// TestListGames_FiltersByVenueFacility proves ListGames scopes results to
// filter.VenueFacilityID when set, per T8.9's Discover & Join Games
// facility filter.
func TestListGames_FiltersByVenueFacility(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	rng1 := mustRange(t, "2026-09-01T10:00:00Z", "2026-09-01T11:00:00Z")
	rng2 := mustRange(t, "2026-09-02T10:00:00Z", "2026-09-02T11:00:00Z")
	games.games["g-a"] = mustGame(t, "g-a", "facility-a", rng1)
	games.games["g-b"] = mustGame(t, "g-b", "facility-b", rng2)

	got, err := svc.ListGames(ctx, port.GameListingFilter{VenueFacilityID: "facility-a"})
	if err != nil {
		t.Fatalf("ListGames err: %v", err)
	}
	if len(got) != 1 || got[0].Game.ID != "g-a" {
		t.Fatalf("ListGames(facility-a) = %+v, want only g-a", got)
	}
}

// TestListGames_FiltersByDateRange proves ListGames scopes results to
// filter.StartsAfter/StartsBefore when set, per T8.9's Discover & Join Games
// date filter.
func TestListGames_FiltersByDateRange(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	early := mustRange(t, "2026-09-01T10:00:00Z", "2026-09-01T11:00:00Z")
	late := mustRange(t, "2026-09-10T10:00:00Z", "2026-09-10T11:00:00Z")
	games.games["g-early"] = mustGame(t, "g-early", "facility-a", early)
	games.games["g-late"] = mustGame(t, "g-late", "facility-a", late)

	after, err := time.Parse(time.RFC3339, "2026-09-05T00:00:00Z")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	got, err := svc.ListGames(ctx, port.GameListingFilter{StartsAfter: after})
	if err != nil {
		t.Fatalf("ListGames err: %v", err)
	}
	if len(got) != 1 || got[0].Game.ID != "g-late" {
		t.Fatalf("ListGames(starts_after=%v) = %+v, want only g-late", after, got)
	}
}

// TestListGames_ExcludesCancelled proves a cancelled Game never appears in
// the Discover & Join Games browse list (T8.9) — it isn't joinable, so it
// has no place there, mirroring ListCourtBookings' cancelled-exclusion
// (internal/booking/app/list_bookings_test.go).
func TestListGames_ExcludesCancelled(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	rng := mustRange(t, "2026-09-01T10:00:00Z", "2026-09-01T11:00:00Z")
	live := mustGame(t, "g-live", "facility-a", rng)
	cancelled := mustGame(t, "g-cancelled", "facility-a", rng)
	if err := cancelled.Cancel(); err != nil {
		t.Fatalf("fixture Cancel err: %v", err)
	}
	games.games["g-live"] = live
	games.games["g-cancelled"] = cancelled

	got, err := svc.ListGames(ctx, port.GameListingFilter{})
	if err != nil {
		t.Fatalf("ListGames err: %v", err)
	}
	if len(got) != 1 || got[0].Game.ID != "g-live" {
		t.Fatalf("ListGames() = %+v, want only g-live (cancelled excluded)", got)
	}
}

// TestListGames_ComputesSpotsLeft proves each returned GameListing's
// SpotsLeft reflects the Game's Capacity minus the *weighted* sum of
// (1 + GuestCount) across its active registrations — the same weighting
// domain.Register's own capacity check uses (T8.9's requirement that the
// detail view show a real "spots left", not a static Capacity).
func TestListGames_ComputesSpotsLeft(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	rng := mustRange(t, "2026-09-01T10:00:00Z", "2026-09-01T11:00:00Z")
	g := mustGame(t, "g-1", "facility-a", rng)
	g.Capacity = 6
	games.games["g-1"] = g

	// One registration with 2 guests occupies 3 of the 6 slots.
	games.registrations["r-1"] = domain.Registration{
		ID: "r-1", GameID: "g-1", PlayerID: "player-1",
		Status: domain.RegistrationStatusRegistered, GuestCount: 2,
	}
	// A cancelled registration must not count against capacity.
	games.registrations["r-2"] = domain.Registration{
		ID: "r-2", GameID: "g-1", PlayerID: "player-2",
		Status: domain.RegistrationStatusCancelled, GuestCount: 5,
	}

	got, err := svc.ListGames(ctx, port.GameListingFilter{})
	if err != nil {
		t.Fatalf("ListGames err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListGames() returned %d listings, want 1", len(got))
	}
	if got[0].SpotsLeft != 3 {
		t.Fatalf("SpotsLeft = %d, want 3 (6 capacity - 3 weight)", got[0].SpotsLeft)
	}
}

// mustGame builds a minimal valid, scheduled domain.Game fixture for
// ListGames tests, bypassing app.Service.ScheduleGame (which requires a
// working CourtReservation/FacilityLookup this suite of tests has no need
// for) by calling domain.NewGame directly and seeding it straight into the
// fake repository.
func mustGame(t *testing.T, id, venueFacilityID string, r domain.TimeRange) domain.Game {
	t.Helper()
	g, err := domain.NewGame(id, "host-1", "", venueFacilityID, []string{courtID(1)}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("fixture NewGame err: %v", err)
	}
	return g
}

// TestListRegistrationsForGame_ReturnsActiveOnly proves
// ListRegistrationsForGame (T8.10, added as a disclosed backend finding —
// see that ticket's PR description) is a thin passthrough to
// RegistrationRepository.ListActiveForGame: it returns every non-cancelled
// registration for the given game and none for an unrelated game, so the
// Host pending-payments dashboard (T8.10) can enumerate Registrations —
// something no RPC on this service exposed before this ticket (confirmed
// by inspection: RegistrationRepository.ListActiveForGame was previously
// only reachable internally, from RegisterForGame's own capacity
// pre-check).
func TestListRegistrationsForGame_ReturnsActiveOnly(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	// Game IDs are UUID-shaped because that is what the real system mints and
	// what the Postgres adapter's mustUUID requires; "g-1" was a fixture shape
	// no real Game ever has, which is why no test could see the malformed-ID
	// crash this file's malformed_id_test.go sibling now covers.
	gameOne, gameTwo := gameID(1), gameID(2)

	// T13.6: the read is Host-only, so the Game must exist and the caller must
	// be its Host for this scoping assertion to be reachable at all. The
	// authorization behaviour itself is proved in the grpcapi package's
	// roster_authz_test.go; here it is only a precondition.
	const host = "auth0|host-1"
	games.games[gameOne] = domain.Game{ID: gameOne, HostID: host}

	registrations.registrations["r-active"] = domain.Registration{
		ID: "r-active", GameID: gameOne, PlayerID: "player-1",
		Status: domain.RegistrationStatusRegistered, PaymentStatus: domain.PaymentStatusUnpaid,
	}
	registrations.registrations["r-cancelled"] = domain.Registration{
		ID: "r-cancelled", GameID: gameOne, PlayerID: "player-2",
		Status: domain.RegistrationStatusCancelled, PaymentStatus: domain.PaymentStatusUnpaid,
	}
	registrations.registrations["r-other-game"] = domain.Registration{
		ID: "r-other-game", GameID: gameTwo, PlayerID: "player-3",
		Status: domain.RegistrationStatusRegistered, PaymentStatus: domain.PaymentStatusUnpaid,
	}

	got, err := svc.ListRegistrationsForGame(ctx, gameOne, host)
	if err != nil {
		t.Fatalf("ListRegistrationsForGame err: %v", err)
	}
	if len(got) != 1 || got[0].ID != "r-active" {
		t.Fatalf("ListRegistrationsForGame(%s) = %+v, want only r-active", gameOne, got)
	}
}

// --- T10.4: RecordMatchResult / ListMatchesForGame -------------------------

func validMatchPlayers() []string {
	return []string{"player-1", "player-2"}
}

func validMatchScore() map[string]int {
	return map[string]int{"player-1": 11, "player-2": 7}
}

// newMatchTestService wires a real app.Service against fresh in-memory fakes
// and seeds one scheduled fixture Game (HostID "host-1", per validInput())
// via the real ScheduleGame path, mirroring TestRegisterForGame_Valid's
// setup. Returns the fixture Game plus every fake a test might need to seed
// or inspect further state on.
func newMatchTestService(t *testing.T) (*app.Service, domain.Game, *fakeGameRepository, *fakeMatchRepository) {
	t.Helper()
	games := newFakeGameRepository()
	matches := newFakeMatchRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       matches,
		GameAdmins:    newFakeGameAdminRepository(),
	})

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(context.Background(), fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}
	return svc, g, games, matches
}

// TestRecordMatchResult_HostSucceeds proves the happy path: the Game's own
// Host may record a Match result, and it's persisted with a generated ID.
func TestRecordMatchResult_HostSucceeds(t *testing.T) {
	t.Parallel()

	svc, g, _, matches := newMatchTestService(t)

	m, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m.ID == "" {
		t.Fatalf("recorded match should have a generated ID")
	}
	if m.GameID != g.ID {
		t.Fatalf("GameID = %q, want %q", m.GameID, g.ID)
	}
	if matches.createCalls.Load() != 1 {
		t.Fatalf("Create should be called exactly once, got %d", matches.createCalls.Load())
	}
}

// assignMatchAdmin grants adminUserID Game-Admin authority over gameID through
// the real Host-only write path (T14.5).
//
// Before T14.5 these tests expressed "is an admin" by putting a string in
// RecordMatchResultInput.AssignedGameAdminUserIDs — which is precisely the
// forgeable shape #168 was about, and which the input struct no longer has a
// field for. Going through Service.AssignGameAdmin means the fixture is
// established by the same rule production uses, so a test can no longer assert
// a permission the real system would not grant.
func assignMatchAdmin(t *testing.T, svc *app.Service, gameID, adminUserID string) {
	t.Helper()

	if _, err := svc.AssignGameAdmin(context.Background(), app.AssignGameAdminInput{
		GameID:      gameID,
		ActorUserID: "host-1", // validInput's HostID — the fixture Game's Host
		AdminUserID: adminUserID,
	}); err != nil {
		t.Fatalf("the Host assigning %q as a Game Admin: %v", adminUserID, err)
	}
}

// TestRecordMatchResult_AssignedGameAdminSucceeds is the symmetric
// positive-path case for the "or an assigned Game Admin" half of the
// authorization check. As of T14.5 the admin is a stored assignment rather than
// a name the caller supplied.
func TestRecordMatchResult_AssignedGameAdminSucceeds(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)
	assignMatchAdmin(t, svc, g.ID, "admin-2")

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "admin-2",
	})
	if err != nil {
		t.Fatalf("an assigned game admin should be allowed to record a match, got: %v", err)
	}
}

// TestRecordMatchResult_RejectsNonHostNonAdmin is the unit-level companion
// to the handler-level BOLA regression test this ticket requires
// (internal/socialplay/adapter/grpcapi): a Player who is neither the Game's
// Host nor an assigned Game Admin is rejected with
// domain.ErrNotGameHostOrAdmin, and the rejected attempt never reaches the
// repository (belt-and-braces, mirroring T10.7's malformed-id call-counter
// discipline applied to the authorization check instead).
func TestRecordMatchResult_RejectsNonHostNonAdmin(t *testing.T) {
	t.Parallel()

	svc, g, _, matches := newMatchTestService(t)
	assignMatchAdmin(t, svc, g.ID, "admin-1")

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "random-player",
	})
	if !errors.Is(err, domain.ErrNotGameHostOrAdmin) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotGameHostOrAdmin)
	}
	if matches.createCalls.Load() != 0 {
		t.Errorf("rejected RecordMatchResult reached the repository (%d Create calls), want 0", matches.createCalls.Load())
	}
}

// TestRecordMatchResult_CancelledGameRejected proves the FailedPrecondition
// case: a cancelled Game may no longer have match results recorded against
// it, even by its own Host.
func TestRecordMatchResult_CancelledGameRejected(t *testing.T) {
	t.Parallel()

	svc, g, games, matches := newMatchTestService(t)

	cancelled := g
	if err := cancelled.Cancel(); err != nil {
		t.Fatalf("fixture cancel should succeed: %v", err)
	}
	games.games[cancelled.ID] = cancelled

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      cancelled.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "host-1",
	})
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameCancelled)
	}
	if matches.createCalls.Load() != 0 {
		t.Errorf("rejected RecordMatchResult reached the repository (%d Create calls), want 0", matches.createCalls.Load())
	}
}

// TestRecordMatchResult_AuthorizationCheckedBeforeCancelledPrecondition
// proves the ORDERING RecordMatchResult's own doc comment claims, not just
// the two checks individually: a non-Host/non-Game-Admin actor attempting
// to record a result against a *cancelled* Game gets ErrNotGameHostOrAdmin
// (PermissionDenied), not ErrGameCancelled (FailedPrecondition) —
// TestRecordMatchResult_RejectsNonHostNonAdmin above only proves the
// authorization check against a non-cancelled Game, and
// TestRecordMatchResult_CancelledGameRejected only proves the cancelled
// check against the Game's own Host, so neither alone pins down which
// check wins when both would independently reject the same call. Without
// this test, a future refactor that swapped the two checks would leak
// whether a Game is cancelled to an unauthorized caller (mirrors
// TestRecordOfflinePayment_AuthorizationCheckedBeforeDomainConstruction's
// identical reasoning, internal/payments/app/service_test.go) with nothing
// here failing red — exactly the gap this test closes.
func TestRecordMatchResult_AuthorizationCheckedBeforeCancelledPrecondition(t *testing.T) {
	t.Parallel()

	svc, g, games, matches := newMatchTestService(t)
	assignMatchAdmin(t, svc, g.ID, "admin-1")

	cancelled := g
	if err := cancelled.Cancel(); err != nil {
		t.Fatalf("fixture cancel should succeed: %v", err)
	}
	games.games[cancelled.ID] = cancelled

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      cancelled.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "random-player",
	})
	if !errors.Is(err, domain.ErrNotGameHostOrAdmin) {
		t.Fatalf("got err %v, want %v (an unauthorized actor must be rejected before the cancelled-Game precondition ever runs, so it never learns whether the Game is cancelled)", err, domain.ErrNotGameHostOrAdmin)
	}
	if errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v — must not be (or wrap) ErrGameCancelled, which would leak the Game's cancelled state to an unauthorized actor", err)
	}
	if matches.createCalls.Load() != 0 {
		t.Errorf("rejected RecordMatchResult reached the repository (%d Create calls), want 0", matches.createCalls.Load())
	}
}

// TestRecordMatchResult_TooFewPlayersRejected and
// TestRecordMatchResult_EmptyScoreRejected prove domain.RecordMatch's own
// validation surfaces through the app layer unchanged, for an authorized,
// non-cancelled request.
func TestRecordMatchResult_TooFewPlayersRejected(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     []string{"player-1"},
		Score:       validMatchScore(),
		ActorUserID: "host-1",
	})
	if !errors.Is(err, domain.ErrTooFewPlayers) {
		t.Fatalf("got err %v, want %v", err, domain.ErrTooFewPlayers)
	}
}

func TestRecordMatchResult_EmptyScoreRejected(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)

	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     validMatchPlayers(),
		Score:       nil,
		ActorUserID: "host-1",
	})
	if !errors.Is(err, domain.ErrEmptyScore) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyScore)
	}
}

// TestRecordMatchResult_UnknownGameRejected proves an unknown-but-well-formed
// GameID answers NotFound (domain.ErrGameNotFound) without ever reaching
// EnsureHostOrGameAdmin (there is no Game to check the actor against).
func TestRecordMatchResult_UnknownGameRejected(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	matches := newFakeMatchRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       matches,
		GameAdmins:    newFakeGameAdminRepository(),
	})

	unknown := gameID(9001)
	_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
		GameID:      unknown,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: "host-1",
	})
	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameNotFound)
	}
	if games.getByIDCalls.Load() != 1 {
		t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", games.getByIDCalls.Load())
	}
	if matches.createCalls.Load() != 0 {
		t.Errorf("rejected RecordMatchResult reached the match repository (%d Create calls), want 0", matches.createCalls.Load())
	}
}

// TestListMatchesForGame_ReturnsRecordedMatches proves the happy path: every
// Match recorded against a Game is returned.
func TestListMatchesForGame_ReturnsRecordedMatches(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	if _, err := svc.RecordMatchResult(ctx, app.RecordMatchResultInput{
		GameID: g.ID, Players: validMatchPlayers(), Score: validMatchScore(), ActorUserID: "host-1",
	}); err != nil {
		t.Fatalf("fixture RecordMatchResult failed: %v", err)
	}

	got, err := svc.ListMatchesForGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListMatchesForGame err: %v", err)
	}
	if len(got) != 1 || got[0].GameID != g.ID {
		t.Fatalf("ListMatchesForGame(%s) = %+v, want one match for this game", g.ID, got)
	}
}

// TestListMatchesForGame_UnknownGameRejected proves this RPC's own
// error-handling requirement — unlike ListRegistrationsForGame's "empty
// roster" convention, an unknown GameID here is NotFound.
func TestListMatchesForGame_UnknownGameRejected(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	matches := newFakeMatchRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       matches,
		GameAdmins:    newFakeGameAdminRepository(),
	})

	unknown := gameID(9002)
	_, err := svc.ListMatchesForGame(context.Background(), unknown)
	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameNotFound)
	}
	if games.getByIDCalls.Load() != 1 {
		t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", games.getByIDCalls.Load())
	}
}

// ---------------------------------------------------------------------------
// CancelGame (T12.4) — mirrors internal/competitions/app's CancelCompetition
// test block exactly, since CancelGame is the same shape (GetByID ->
// EnsureHost -> transition -> UpdateStatus) on the Game aggregate.
// ---------------------------------------------------------------------------

// TestCancelGame_HostSucceeds proves the happy path: the Game's own Host may
// cancel it, and the change is actually *persisted*, not just returned in
// memory (re-read through GetByID rather than trusting the return value —
// the standard the rest of this file's write-path tests already hold).
func TestCancelGame_HostSucceeds(t *testing.T) {
	t.Parallel()

	svc, g, games, _ := newMatchTestService(t)
	ctx := context.Background()

	cancelled, err := svc.CancelGame(ctx, g.ID, g.HostID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Status = %q, want cancelled", cancelled.Status)
	}

	stored, err := games.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("re-fetching the cancelled game failed: %v", err)
	}
	if stored.Status != domain.StatusCancelled {
		t.Fatalf("persisted Status = %q, want cancelled", stored.Status)
	}
}

// TestCancelGame_NonHostRejected is the required EnsureHost proof at the app
// layer. The error must be ErrNotGameHost *specifically* — asserting merely
// non-nil would pass against an implementation that rejected the call for
// some entirely unrelated reason, and the handler maps this exact sentinel
// to PermissionDenied rather than Internal. It also asserts no state
// changed, so a "rejected but cancelled anyway" bug cannot hide.
func TestCancelGame_NonHostRejected(t *testing.T) {
	t.Parallel()

	svc, g, games, _ := newMatchTestService(t)
	ctx := context.Background()

	_, err := svc.CancelGame(ctx, g.ID, "not-the-host")
	if !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("got err %v, want ErrNotGameHost specifically", err)
	}

	stored, err := games.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("re-fetching the game failed: %v", err)
	}
	if stored.Status != domain.StatusScheduled {
		t.Fatalf("a rejected cancel changed Status to %q; it must stay scheduled", stored.Status)
	}
}

// TestCancelGame_AssignedGameAdminRejected is the app-layer half of the
// ErrNotGameHost/ErrNotGameHostOrAdmin distinction: the very same actor that
// RecordMatchResult accepts as an assigned Game Admin must not be able to
// cancel the Game.
//
// T14.5 strengthens this from a statement about arguments into a statement
// about authority. The admin is now a real stored assignment, and the fixture
// precondition below proves it is genuinely honoured elsewhere before asserting
// it buys nothing here — so a reader cannot dismiss the rejection as "CancelGame
// simply wasn't told about the admins".
func TestCancelGame_AssignedGameAdminRejected(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()
	const gameAdmin = "admin-2"

	assignMatchAdmin(t, svc, g.ID, gameAdmin)

	if _, err := svc.RecordMatchResult(ctx, app.RecordMatchResultInput{
		GameID:      g.ID,
		Players:     validMatchPlayers(),
		Score:       validMatchScore(),
		ActorUserID: gameAdmin,
	}); err != nil {
		t.Fatalf("fixture precondition: %q must be a legitimate game admin, got %v", gameAdmin, err)
	}

	if _, err := svc.CancelGame(ctx, g.ID, gameAdmin); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("a game admin must not be able to cancel the game: got err %v, want ErrNotGameHost", err)
	}
}

// TestCancelGame_EmptyActorRejected pins domain.EnsureHost's "an
// unidentified caller is never the host" rule at the app boundary.
func TestCancelGame_EmptyActorRejected(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)

	if _, err := svc.CancelGame(context.Background(), g.ID, ""); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("got err %v, want ErrNotGameHost", err)
	}
}

// TestCancelGame_UnknownGameNotFound covers the well-formed-but-unknown id
// case, and proves it actually reached the repository (rather than being
// short-circuited by the uuidShape guard, which the next test covers).
func TestCancelGame_UnknownGameNotFound(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	if _, err := svc.CancelGame(context.Background(), gameID(9101), "host-1"); !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want ErrGameNotFound", err)
	}
	if games.getByIDCalls.Load() != 1 {
		t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", games.getByIDCalls.Load())
	}
}

// TestCancelGame_MalformedGameIDNotFound is the T10.7-shaped boundary guard:
// a malformed id must produce exactly the same ErrGameNotFound an unknown
// one does, without ever reaching the repository (where the real Postgres
// adapter's mustUUID would panic on it).
func TestCancelGame_MalformedGameIDNotFound(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	if _, err := svc.CancelGame(context.Background(), "not-a-uuid", "host-1"); !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want ErrGameNotFound", err)
	}
	if games.getByIDCalls.Load() != 0 {
		t.Fatalf("malformed GameID reached the repository (%d calls); the boundary guard must stop it first", games.getByIDCalls.Load())
	}
}

// TestCancelGame_AlreadyCancelledRejected proves a second cancel is refused
// rather than silently accepted, and — specifically — with the
// ErrGameCancelled sentinel, which the handler maps to FailedPrecondition.
// See CancelGame's doc comment for why this path deliberately goes through
// Game.EnsureNotCancelled rather than relying on Game.Cancel's own
// ErrIllegalStatusTransition (which socialplay's toStatus maps to
// InvalidArgument, not FailedPrecondition).
func TestCancelGame_AlreadyCancelledRejected(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); err != nil {
		t.Fatalf("first cancel failed: %v", err)
	}
	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want ErrGameCancelled", err)
	}
}

// TestCancelGame_AuthorizationPrecedesStatusCheck pins the ordering: a
// non-Host acting on an already-cancelled Game gets ErrNotGameHost, not
// ErrGameCancelled. Leaking "this game exists and is already cancelled" to
// an unauthorized actor would be a (minor) information disclosure, and more
// importantly the ordering is what makes the authorization check
// unconditional rather than reachable only for Games in one particular
// state.
func TestCancelGame_AuthorizationPrecedesStatusCheck(t *testing.T) {
	t.Parallel()

	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); err != nil {
		t.Fatalf("fixture precondition: first cancel failed: %v", err)
	}
	if _, err := svc.CancelGame(ctx, g.ID, "not-the-host"); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("got err %v, want ErrNotGameHost (authorization must be checked before status)", err)
	}
}

// --- T16.3: cancelling a Game cascades to its active Registrations -------
// (partial fix for #124; the Competitions mirror lives in
// internal/competitions/app/service_test.go's identically-purposed tests)

// TestCancelGame_CancelsActiveRegistrations is T16.3's core proof:
// cancelling a Game must cascade to its active Registrations, not just
// flip the Game's own status and leave every registered player looking
// "registered" for something that no longer exists.
func TestCancelGame_CancelsActiveRegistrations(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	first, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"})
	if err != nil {
		t.Fatalf("fixture registration 1 failed: %v", err)
	}
	second, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-2"})
	if err != nil {
		t.Fatalf("fixture registration 2 failed: %v", err)
	}

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); err != nil {
		t.Fatalf("CancelGame failed: %v", err)
	}

	if got := registrations.registrations[first.ID].Status; got != domain.RegistrationStatusCancelled {
		t.Fatalf("registration 1 Status = %q after CancelGame, want cancelled", got)
	}
	if got := registrations.registrations[second.ID].Status; got != domain.RegistrationStatusCancelled {
		t.Fatalf("registration 2 Status = %q after CancelGame, want cancelled", got)
	}
}

// TestCancelGame_LeavesRegistrationPaymentStatusAlone proves the cascade
// only touches Status, never PaymentStatus (T16.3 instruction 4: this
// ticket does not call RefundPayment or decide the refund question) — a
// paid registration stays marked paid even once its Status flips to
// cancelled, so a support/admin view can still tell "this player paid,
// then the game was cancelled" from "this player never paid".
func TestCancelGame_LeavesRegistrationPaymentStatusAlone(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"})
	if err != nil {
		t.Fatalf("fixture registration failed: %v", err)
	}
	if err := svc.MarkRegistrationPaymentStatus(ctx, reg.ID, domain.PaymentStatusPaid); err != nil {
		t.Fatalf("fixture MarkRegistrationPaymentStatus failed: %v", err)
	}

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); err != nil {
		t.Fatalf("CancelGame failed: %v", err)
	}

	got := registrations.registrations[reg.ID]
	if got.Status != domain.RegistrationStatusCancelled {
		t.Fatalf("Status = %q, want cancelled", got.Status)
	}
	if got.PaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("PaymentStatus = %q, want paid (unchanged) — the cascade must not touch payment state", got.PaymentStatus)
	}
}

// TestCancelGame_AlreadyCancelledRegistrationNotRecancelled proves a
// registration a player cancelled before the Host cancelled the Game is
// left alone by the cascade — CancelAllActiveForGame's WHERE clause is
// `status <> 'cancelled'`, so this test would catch a fake or a query that
// swept every registration indiscriminately (and would over-count the
// affected-rows total the real query returns).
func TestCancelGame_AlreadyCancelledRegistrationNotRecancelled(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	fixtureIn := validInput(courtID(1))
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("fixture game should schedule, got %v", err)
	}

	reg, err := svc.RegisterForGame(ctx, app.RegisterForGameInput{GameID: g.ID, PlayerID: "player-1"})
	if err != nil {
		t.Fatalf("fixture registration failed: %v", err)
	}
	if _, err := svc.CancelRegistration(ctx, reg.ID, "player-1"); err != nil {
		t.Fatalf("fixture pre-cancel failed: %v", err)
	}

	n, err := registrations.CancelAllActiveForGame(ctx, g.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 0 {
		t.Fatalf("CancelAllActiveForGame transitioned %d rows, want 0 (the only registration was already cancelled)", n)
	}
}

// TestCancelGame_DoesNotPromoteWaitlist is T16.3 instruction 5's required
// proof, not merely an assertion: cancelling a full Game with a waiting
// entry must NOT promote that entry. CancelGame never calls
// promoteNextWaiting (contrast CancelRegistration, which does, because
// that Game is still live and the freed slot is real) — this test checks
// that behaviour against the actual code rather than trusting the doc
// comment.
func TestCancelGame_DoesNotPromoteWaitlist(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	waitlist := newFakeWaitlistRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      waitlist,
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	ctx := context.Background()

	g := fixtureFullGame(t, ctx, svc, 1)

	entry, err := svc.JoinWaitlist(ctx, app.JoinWaitlistInput{GameID: g.ID, PlayerID: "player-waiting"})
	if err != nil {
		t.Fatalf("fixture waitlist join failed: %v", err)
	}
	if entry.Status != domain.WaitlistStatusWaiting {
		t.Fatalf("fixture precondition: entry.Status = %q, want waiting", entry.Status)
	}

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID); err != nil {
		t.Fatalf("CancelGame failed: %v", err)
	}

	stored, err := waitlist.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("re-fetching the waitlist entry failed: %v", err)
	}
	if stored.Status != domain.WaitlistStatusWaiting {
		t.Fatalf("waitlist entry Status = %q after cancelling its Game, want waiting (unchanged) — a cancelled Game must not promote anyone", stored.Status)
	}
}
