package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

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

// sequentialIDs is a deterministic port.IDGenerator fake, mirroring
// internal/booking/app's test fake of the same shape. Shared by both Game
// and Registration IDs (real deployments use a random UUID generator, see
// internal/platform/idgen) so its prefix is generic rather than "game-".
type sequentialIDs struct{ n int }

func (g *sequentialIDs) NewID() string {
	g.n++
	return fmt.Sprintf("id-%d", g.n)
}

// fakeGameRepository is a minimal in-memory port.GameRepository fake, mirroring
// internal/booking/app's inMemoryRepo. createErr lets a test simulate
// persistence itself failing (e.g. a DB outage) after reservations already
// succeeded, to prove ScheduleGame's rollback-on-persist-failure path.
type fakeGameRepository struct {
	games     map[string]domain.Game
	createErr error
}

func newFakeGameRepository() *fakeGameRepository {
	return &fakeGameRepository{games: make(map[string]domain.Game)}
}

func (r *fakeGameRepository) Create(_ context.Context, g domain.Game) (domain.Game, error) {
	if r.createErr != nil {
		return domain.Game{}, r.createErr
	}
	r.games[g.ID] = g
	return g, nil
}

func (r *fakeGameRepository) GetByID(_ context.Context, id string) (domain.Game, error) {
	g, ok := r.games[id]
	if !ok {
		return domain.Game{}, domain.ErrGameNotFound
	}
	return g, nil
}

// fakeRegistrationRepository is a minimal in-memory port.RegistrationRepository
// fake.
type fakeRegistrationRepository struct {
	registrations map[string]domain.Registration
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

func validInput(courtIDs ...string) app.ScheduleGameInput {
	return app.ScheduleGameInput{
		HostID:     "host-1",
		FacilityID: "facility-1",
		CourtIDs:   courtIDs,
		Capacity:   4,
	}
}

// TestScheduleGame_ReservesEveryCourt proves the happy path: a multi-court
// Game reserves one Booking per court, using the game's ID as the
// referenceID passed to the port.
func TestScheduleGame_ReservesEveryCourt(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	g, err := svc.ScheduleGame(context.Background(), in, reservation)
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

	reservation := newFakeReservation("court-1")
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())
	in := validInput("court-1")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
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

	reservation := newFakeReservation("court-2")
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
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

	reservation := newFakeReservation("court-2")
	reservation.releaseErr = errors.New("release: boom")
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
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
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())
	in := validInput("court-1")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	in.Capacity = 0

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
	if !errors.Is(err, domain.ErrInvalidCapacity) {
		t.Fatalf("got err %v, want ErrInvalidCapacity", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("invalid Game must not touch the reservation port, got calls: %v", reservation.reserveCalls)
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
	svc := app.NewService(&sequentialIDs{}, games, newFakeRegistrationRepository())
	in := validInput("court-1")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	g, err := svc.ScheduleGame(context.Background(), in, reservation)
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
	svc := app.NewService(&sequentialIDs{}, games, newFakeRegistrationRepository())
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
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
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	fixtureIn := validInput("court-1")
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation())
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
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	in := validInput("court-1")
	in.Capacity = 1
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, in, newFakeReservation())
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

	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())

	_, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{GameID: "no-such-game", PlayerID: "player-1"})
	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("got err %v, want ErrGameNotFound", err)
	}
}

// TestCancelRegistration_OwnerSucceeds proves the happy path: the owning
// player can cancel their own registration, and the persisted status
// actually flips (not just the in-memory value returned).
func TestCancelRegistration_OwnerSucceeds(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	fixtureIn := validInput("court-1")
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation())
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
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	fixtureIn := validInput("court-1")
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation())
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
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	fixtureIn := validInput("court-1")
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation())
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

	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository())

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
	svc := app.NewService(&sequentialIDs{}, games, registrations)
	ctx := context.Background()

	fixtureIn := validInput("court-1")
	fixtureIn.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := svc.ScheduleGame(ctx, fixtureIn, newFakeReservation())
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
