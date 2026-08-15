//go:build integration

// T17.2 (part of #195) — the load-bearing half of this ticket's regression
// proof that no in-memory fixture can supply.
//
// The gap: registrations.game_id and games.venue_facility_id are both
// FK-backed writes guarded by an app-level read immediately beforehand
// (app.Service.RegisterForGame's s.games.GetByID; ScheduleGame's
// facilities.FacilityExists). In the non-racing case that read's own
// not-found sentinel wins and the FK never fires — translate_test.go's unit
// tests already pin what translateRegistrationErr/translateGameErr do WHEN a
// 23503 arrives, using a synthetic *pgconn.PgError, but they cannot prove
// Postgres actually raises 23503 for these two writes, and no in-memory port
// fake models a table that can be deleted out from under a caller mid-request
// — the same fixture-infidelity trap #185 was caught in twice, per #195's own
// body. This file is where that assumption is checked against a real
// database, in the SPECIFIC narrow window #195 is about: the parent deleted
// between the guarding read and the insert that follows it, not merely an
// insert against an id that was already unknown from the start.
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker. NOT run in this environment (no Docker
// daemon) — see this ticket's PR description for the mutation-check
// (temporarily removing translateRegistrationErr/translateGameErr's new
// arms and confirming translate_test.go's unit-level cases fail) that stands
// in for actually executing this file here, per this ticket's own
// instruction 5.
//
// Shares waitForReady/applyMigrations/mustRange/seedCourtID
// (concurrency_integration_test.go) and newGameAdminTestPool
// (game_admin_integration_test.go) with this package's other integration
// tests rather than re-declaring container boot/migrate boilerplate a fourth
// time.
package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestCreateRegistration_GameDeletedBetweenGuardReadAndInsertIsGameNotFound
// is registrations.game_id's half of #195's race.
//
// Sequence, deliberately linear rather than driven by goroutines: Go's own
// ordering already produces the interleaving #195 describes (guarding read
// completes and observes the Game, THEN the Game is removed, THEN the insert
// that read believed was safe runs) without adding a synchronization
// primitive whose own correctness would need proving. A goroutine-based race
// would non-deterministically also cover the "delete never lands in the
// window" case, which is not the case this ticket is about and is already
// covered by TestCreateRegistration_ExactlyOneWinsUnderConcurrency-style
// tests elsewhere in this package; this test's job is to pin what happens
// when the delete DOES land in the window, every run.
func TestCreateRegistration_GameDeletedBetweenGuardReadAndInsertIsGameNotFound(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	t.Run("game deleted after the guarding read answers ErrGameNotFound, not a raw 23503", func(t *testing.T) {
		r := mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T10:00:00Z")
		game, err := domain.NewGame(
			"33333333-3333-3333-3333-330000000001", "race-host", "facility-x", "",
			[]string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
		)
		if err != nil {
			t.Fatalf("bad fixture game: %v", err)
		}
		game, err = gameRepo.Create(ctx, game)
		if err != nil {
			t.Fatalf("failed to create fixture game: %v", err)
		}

		// The guarding read app.Service.RegisterForGame performs before its
		// own insert — proven to succeed here, exactly as it would for a
		// real request that arrives before the race window closes.
		if _, err := gameRepo.GetByID(ctx, game.ID); err != nil {
			t.Fatalf("guarding read: %v", err)
		}

		// The race: the Game is deleted in the gap between that read
		// succeeding and the insert below running. A raw SQL DELETE —
		// neither this context nor Booking exposes a domain-level "delete a
		// Game" (#195's own body: "none of these contexts currently ships a
		// bulk-delete or cascading-delete flow"), so this is the only way
		// the window can be produced at all.
		if _, err := pool.Exec(ctx, "DELETE FROM games WHERE id = $1", game.ID); err != nil {
			t.Fatalf("failed to delete fixture game: %v", err)
		}

		reg := domain.Registration{
			ID:            "33333333-3333-3333-3333-330000000002",
			GameID:        game.ID,
			PlayerID:      "race-player",
			Source:        domain.RegistrationSourceApp,
			Status:        domain.RegistrationStatusRegistered,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}
		_, err = regRepo.Create(ctx, reg)
		if !errors.Is(err, domain.ErrGameNotFound) {
			t.Fatalf("Create(registration against a concurrently-deleted game) = %v, want domain.ErrGameNotFound", err)
		}

		// The inverse guard: a raw, untranslated driver error must never
		// survive to the caller — proves this went through
		// translateRegistrationErr's 23503 arm, not its wrapped default,
		// which would still satisfy errors.As for *pgconn.PgError because
		// fmt.Errorf's %w keeps the chain intact.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			t.Fatalf("Create leaked a raw *pgconn.PgError instead of a translated domain error: %v", err)
		}
	})

	// Control: without it, a repository that rejected every insert would
	// satisfy the subtest above for the wrong reason.
	t.Run("control: registering against a game that was never deleted still succeeds", func(t *testing.T) {
		r := mustRange(t, "2026-09-05T13:00:00Z", "2026-09-05T14:00:00Z")
		game, err := domain.NewGame(
			"33333333-3333-3333-3333-330000000003", "race-host-2", "facility-x", "",
			[]string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
		)
		if err != nil {
			t.Fatalf("bad fixture game: %v", err)
		}
		game, err = gameRepo.Create(ctx, game)
		if err != nil {
			t.Fatalf("failed to create fixture game: %v", err)
		}

		reg := domain.Registration{
			ID:            "33333333-3333-3333-3333-330000000004",
			GameID:        game.ID,
			PlayerID:      "race-player-2",
			Source:        domain.RegistrationSourceApp,
			Status:        domain.RegistrationStatusRegistered,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}
		if _, err := regRepo.Create(ctx, reg); err != nil {
			t.Fatalf("Create(registration against a real game) = %v, want success", err)
		}
	})
}

// TestCreateGame_VenueFacilityDeletedBetweenGuardReadAndInsertIsFacilityNotFound
// is games.venue_facility_id's half of #195's race — the direct sibling of
// the test above, over the Games repository and a Facilities-context
// fixture instead of a Social Play one.
//
// The guarding read is modelled at the DB layer port.FacilityLookup.
// FacilityExists ultimately relies on (facilitiespg.Repository.
// GetFacilityByID) rather than by wiring the full Facilities app.Service:
// the cross-context lookup wiring itself (facilitiesdomain.
// ErrFacilityNotFound -> this context's own domain.ErrFacilityNotFound) is
// already covered by internal/socialplay/adapter/facilities/lookup_test.go;
// this file's job is only the FK translation the DB layer beneath that
// lookup shares with it.
func TestCreateGame_VenueFacilityDeletedBetweenGuardReadAndInsertIsFacilityNotFound(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	facilityRepo := facilitiespg.NewRepository(pool)
	gameRepo := socialplaypg.NewGameRepository(pool)

	t.Run("facility deleted after the guarding read answers ErrFacilityNotFound, not a raw 23503", func(t *testing.T) {
		facility, err := facilitiesdomain.NewFacility(
			"44444444-4444-4444-4444-440000000001",
			"44444444-4444-4444-4444-4400000000f1",
			"Race Facility", "", "1 Race St", nil,
		)
		if err != nil {
			t.Fatalf("bad fixture facility: %v", err)
		}
		facility, err = facilityRepo.CreateFacility(ctx, facility)
		if err != nil {
			t.Fatalf("failed to create fixture facility: %v", err)
		}

		// The guarding read app.Service.ScheduleGame performs via
		// port.FacilityLookup.FacilityExists before its own insert.
		if _, err := facilityRepo.GetFacilityByID(ctx, facility.ID); err != nil {
			t.Fatalf("guarding read: %v", err)
		}

		// The race: the Facility is deleted in the gap between that read
		// succeeding and the insert below running. Raw SQL DELETE:
		// Facilities exposes no domain-level delete either (#195's own
		// body).
		if _, err := pool.Exec(ctx, "DELETE FROM facilities WHERE id = $1", facility.ID); err != nil {
			t.Fatalf("failed to delete fixture facility: %v", err)
		}

		r := mustRange(t, "2026-09-05T11:00:00Z", "2026-09-05T12:00:00Z")
		game, err := domain.NewGame(
			"44444444-4444-4444-4444-440000000002", "race-host", "facility-x", facility.ID,
			[]string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
		)
		if err != nil {
			t.Fatalf("bad fixture game: %v", err)
		}

		_, err = gameRepo.Create(ctx, game)
		if !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("Create(game against a concurrently-deleted facility) = %v, want domain.ErrFacilityNotFound", err)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			t.Fatalf("Create leaked a raw *pgconn.PgError instead of a translated domain error: %v", err)
		}

		// The inverse check the FK's own comment (repository.go) warns
		// about: a Game with no venue Facility set at all (VenueFacilityID
		// == "") must not be mistaken for this case — nullableUUID skips the
		// column entirely rather than sending an empty string through
		// mustUUID.
		gameNoVenue, err := domain.NewGame(
			"44444444-4444-4444-4444-440000000003", "race-host", "facility-x", "",
			[]string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
		)
		if err != nil {
			t.Fatalf("bad fixture game: %v", err)
		}
		if _, err := gameRepo.Create(ctx, gameNoVenue); err != nil {
			t.Fatalf("Create(game with no venue facility) = %v, want success", err)
		}
	})

	// Control: without it, a repository that rejected every insert would
	// satisfy the subtest above for the wrong reason.
	t.Run("control: a game against a facility that was never deleted still succeeds", func(t *testing.T) {
		facility, err := facilitiesdomain.NewFacility(
			"44444444-4444-4444-4444-440000000004",
			"44444444-4444-4444-4444-4400000000f2",
			"Race Facility 2", "", "2 Race St", nil,
		)
		if err != nil {
			t.Fatalf("bad fixture facility: %v", err)
		}
		facility, err = facilityRepo.CreateFacility(ctx, facility)
		if err != nil {
			t.Fatalf("failed to create fixture facility: %v", err)
		}

		r := mustRange(t, "2026-09-05T15:00:00Z", "2026-09-05T16:00:00Z")
		game, err := domain.NewGame(
			"44444444-4444-4444-4444-440000000005", "race-host-2", "facility-x", facility.ID,
			[]string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
		)
		if err != nil {
			t.Fatalf("bad fixture game: %v", err)
		}
		if _, err := gameRepo.Create(ctx, game); err != nil {
			t.Fatalf("Create(game against a real facility) = %v, want success", err)
		}
	})
}
