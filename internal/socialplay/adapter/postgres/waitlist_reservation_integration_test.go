//go:build integration

// T61 — the regression test for the half of `db/migrations/0030` that had no
// test anywhere, on either side of the boundary.
//
// # What this pins
//
// `0007` (T6.6) taught `enforce_game_capacity()` that an unexpired `promoted`
// waitlist entry belonging to a DIFFERENT player occupies a slot: when a
// registration is cancelled and the next waitlisted player is promoted, that
// player's slot is held for `domain.PromotionResponseWindow` and a third
// party's direct registration must not take it. `0012` (T8.7) then rewrote the
// function for guest weighting, building its body from `0006`'s version, and
// the reservation vanished without being mentioned.
//
// `domain.SlotReservedByPromotion` has enforced the same rule in Go the whole
// time, so from T8.7 the authoritative half (CLAUDE.md rule 4) did not. Anything
// reaching the repository outside `app.Service.RegisterForGame`'s pre-check —
// and any concurrent registration racing that pre-check, which is exactly what
// the pre-check cannot close — could take a promoted player's slot.
//
// # Why this test did not exist
//
// `0007` shipped the guard with no test on either side. `0012` dropped it with
// no test to fail. Nothing noticed for 18 migrations. Both facts are the same
// fact: a DB-level guard with no DB-level test is a comment.
//
// This test drives the repository directly rather than `app.Service`, for the
// reason `0007`'s own header gives — the app layer's `SlotReservedByPromotion`
// pre-check would refuse the registration before Postgres ever saw it, so a
// test going through `app.Service` would pass against a trigger that had no
// reservation logic at all. That is precisely how this regression stayed
// invisible.
//
// Requires Docker. Shares newTestPool/mustRange/seedCourtID with this package's
// other integration tests and seedSocialplayUser with
// identity_fixtures_integration_test.go.
package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestRegistrationCannotTakeASlotReservedByAnotherPlayersPromotion is the
// assertion `0012` removed.
func TestRegistrationCannotTakeASlotReservedByAnotherPlayersPromotion(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)

	// Capacity 1, so the single slot is unambiguous: whoever holds it, holds
	// all of it.
	hostID := seedSocialplayUser(t, ctx, pool, "reservation-host")
	activeID := seedSocialplayUser(t, ctx, pool, "reservation-active")
	promotedID := seedSocialplayUser(t, ctx, pool, "reservation-promoted")
	interloperID := seedSocialplayUser(t, ctx, pool, "reservation-interloper")

	game, err := domain.NewGame(
		"44444444-4444-4444-4444-610000000001", hostID, "facility-x", "",
		[]string{seedCourtID}, mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z"),
		1, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	if _, err := gameRepo.Create(ctx, game); err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	// The slot is taken, so the waitlist is legal to join.
	active, err := regRepo.Create(ctx, domain.Registration{
		ID:            "44444444-4444-4444-4444-610000000002",
		GameID:        game.ID,
		PlayerID:      activeID,
		Source:        domain.RegistrationSourceApp,
		Status:        domain.RegistrationStatusRegistered,
		PaymentStatus: domain.PaymentStatusUnpaid,
	})
	if err != nil {
		t.Fatalf("failed to fill the single slot: %v", err)
	}

	if _, err := waitlistRepo.Create(ctx, domain.WaitlistEntry{
		ID:       "44444444-4444-4444-4444-610000000003",
		GameID:   game.ID,
		PlayerID: promotedID,
		Status:   domain.WaitlistStatusWaiting,
	}); err != nil {
		t.Fatalf("failed to join the waitlist: %v", err)
	}

	// The slot frees and the waiting player is promoted — their response window
	// starts now, and for its duration the slot is theirs.
	active.Status = domain.RegistrationStatusCancelled
	if _, err := regRepo.Update(ctx, active); err != nil {
		t.Fatalf("failed to cancel the active registration: %v", err)
	}
	promoted, err := waitlistRepo.PromoteNext(ctx, game.ID, time.Now())
	if err != nil {
		t.Fatalf("PromoteNext: %v", err)
	}
	if promoted.PlayerID != promotedID {
		t.Fatalf("PromoteNext promoted %q, want %q", promoted.PlayerID, promotedID)
	}

	// THE ASSERTION. A third party registering directly must be refused: the
	// slot is held. Driven at the repository, deliberately — see this file's
	// header for why going through app.Service would pass regardless of what
	// the trigger does.
	_, err = regRepo.Create(ctx, domain.Registration{
		ID:            "44444444-4444-4444-4444-610000000004",
		GameID:        game.ID,
		PlayerID:      interloperID,
		Source:        domain.RegistrationSourceApp,
		Status:        domain.RegistrationStatusRegistered,
		PaymentStatus: domain.PaymentStatusUnpaid,
	})
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("a third player registered into a slot reserved by another player's promotion: err = %v, want %v.\n\n"+
			"enforce_game_capacity() has lost 0007's reserved_by_others count — see "+
			"db/migrations/0030_socialplay_restore_weighted_capacity_guard.sql for the last "+
			"time a redefinition dropped it.", err, domain.ErrGameFull)
	}

	// The promoted player themself is NOT blocked: their own registration is
	// the confirmation the window exists for. Without this, a trigger that
	// refused everyone would satisfy the assertion above — and the exemption is
	// the subtle half of 0007's rule, the half a reconstruction from memory
	// would omit.
	if _, err := regRepo.Create(ctx, domain.Registration{
		ID:            "44444444-4444-4444-4444-610000000005",
		GameID:        game.ID,
		PlayerID:      promotedID,
		Source:        domain.RegistrationSourceApp,
		Status:        domain.RegistrationStatusRegistered,
		PaymentStatus: domain.PaymentStatusUnpaid,
	}); err != nil {
		t.Fatalf("the promoted player could not claim their own reserved slot: %v — "+
			"the reservation must exempt its own holder (domain.SlotReservedByPromotion does)", err)
	}
}

// TestRegistrationMayTakeASlotWhoseReservationHasExpired is the other edge of
// the same rule, and the control for the window itself: a reservation that has
// outlived PromotionResponseWindow holds nothing. Without this, a trigger
// treating every `promoted` row as a permanent claim would pass the test above
// while quietly making an unanswered promotion block the slot forever.
func TestRegistrationMayTakeASlotWhoseReservationHasExpired(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	hostID := seedSocialplayUser(t, ctx, pool, "expired-reservation-host")
	promotedID := seedSocialplayUser(t, ctx, pool, "expired-reservation-promoted")
	interloperID := seedSocialplayUser(t, ctx, pool, "expired-reservation-interloper")

	game, err := domain.NewGame(
		"44444444-4444-4444-4444-610000000011", hostID, "facility-x", "",
		[]string{seedCourtID}, mustRange(t, "2026-09-01T11:00:00Z", "2026-09-01T12:00:00Z"),
		1, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	if _, err := gameRepo.Create(ctx, game); err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	// A promotion that happened longer ago than PromotionResponseWindow (30
	// minutes). Written with raw SQL because no repository method backdates a
	// promotion, and the window is measured against `now()` inside the trigger
	// — so the only way to be on the far side of it without sleeping for half an
	// hour is to place promoted_at in the past.
	if _, err := pool.Exec(ctx, `
		INSERT INTO waitlist_entries (id, game_id, player_id, position, status, promoted_at)
		VALUES ($1, $2, $3, 1, 'promoted', now() - interval '31 minutes')
	`, "44444444-4444-4444-4444-610000000012", game.ID, promotedID); err != nil {
		t.Fatalf("failed to seed an expired promotion: %v", err)
	}

	if _, err := regRepo.Create(ctx, domain.Registration{
		ID:            "44444444-4444-4444-4444-610000000013",
		GameID:        game.ID,
		PlayerID:      interloperID,
		Source:        domain.RegistrationSourceApp,
		Status:        domain.RegistrationStatusRegistered,
		PaymentStatus: domain.PaymentStatusUnpaid,
	}); err != nil {
		t.Fatalf("an expired promotion blocked the slot: %v — "+
			"a reservation past PromotionResponseWindow reserves nothing "+
			"(domain.WaitlistEntry.HasExpired), or the 30-minute literal in "+
			"enforce_game_capacity() no longer matches domain.PromotionResponseWindow", err)
	}
}
