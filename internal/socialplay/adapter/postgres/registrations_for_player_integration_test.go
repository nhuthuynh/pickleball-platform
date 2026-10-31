//go:build integration

// T67.2 (part of issue #353) — the DB-level half of ListRegistrationsForActor.
//
// # Why this test has to exist, and why the unit tests do not cover it
//
// Every other test for this feature runs against an in-memory fake. CLAUDE.md
// is explicit about what that buys and what it does not: "an in-memory fake is
// more permissive than Postgres, and that asymmetry hides storage bugs" —
// `0031`'s defect passed every unit-level test for 22 sprints because the fake
// had no CHECK constraint to violate.
//
// Three things in this ticket exist only in SQL and are therefore invisible to
// every fake:
//
//  1. The JOIN. `ListRegistrationsForPlayer` is one statement over two tables;
//     a fake models it with a map lookup. Only a real database can tell us the
//     join condition, the column list and the `g.status AS game_status` alias
//     actually produce the right row — sqlc mints a distinct `...Row` struct
//     per query (CLAUDE.md gotcha), so the adapter's field-by-field conversion
//     is unchecked by anything else.
//  2. The `(r.status <> 'cancelled' OR g.status = 'cancelled')` filter. The
//     fakes reimplement this predicate in Go; a test driving a fake proves the
//     *fake's* copy is right, which is precisely the failure mode
//     `waitlist_reservation_integration_test.go` was written about.
//  3. `db/migrations/0033`'s `registrations_player_idx`. Nothing but a real
//     database has an index at all.
//
// This test drives the REPOSITORY, not `app.Service` — the same reason
// `waitlist_reservation_integration_test.go` gives. The app layer's uuidShape
// guard and its own filtering sit above the statement; going through it would
// let a query with the wrong WHERE clause still pass.
//
// Requires Docker. Shares newTestPool/mustRange/seedCourtID with this
// package's other integration tests and seedSocialplayUser with
// identity_fixtures_integration_test.go.
package postgres_test

import (
	"context"
	"testing"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// seedPlayerGame creates one Game and returns it.
func seedPlayerGame(t *testing.T, ctx context.Context, repo *socialplaypg.GameRepository,
	id, hostID, start, end string) domain.Game {
	t.Helper()

	game, err := domain.NewGame(
		id, hostID, "facility-x", "",
		[]string{seedCourtID}, mustRange(t, start, end),
		4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	stored, err := repo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}
	return stored
}

// seedPlayerRegistration registers playerID for gameID.
func seedPlayerRegistration(t *testing.T, ctx context.Context,
	repo *socialplaypg.RegistrationRepository, id, gameID, playerID string) domain.Registration {
	t.Helper()

	reg, err := repo.Create(ctx, domain.Registration{
		ID:            id,
		GameID:        gameID,
		PlayerID:      playerID,
		Source:        domain.RegistrationSourceApp,
		Status:        domain.RegistrationStatusRegistered,
		PaymentStatus: domain.PaymentStatusUnpaid,
		AmountOwed:    domain.Money{Cents: 1500, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("failed to create fixture registration: %v", err)
	}
	return reg
}

// TestListForPlayer_ScopesToThePlayerAndProjectsTheGame is the happy path
// against a real database: the caller's row comes back, another player's does
// not, and all five projected Game columns survive the round trip.
func TestListForPlayer_ScopesToThePlayerAndProjectsTheGame(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	hostID := seedSocialplayUser(t, ctx, pool, "forplayer-host")
	playerID := seedSocialplayUser(t, ctx, pool, "forplayer-player")
	otherID := seedSocialplayUser(t, ctx, pool, "forplayer-other")

	game := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000001", hostID,
		"2026-12-01T09:00:00Z", "2026-12-01T10:30:00Z")

	seedPlayerRegistration(t, ctx, regRepo,
		"44444444-4444-4444-4444-670000000002", game.ID, playerID)
	// A second player on the SAME Game. If the WHERE clause were wrong, this
	// row would come back too — which a single-registration fixture could not
	// detect.
	seedPlayerRegistration(t, ctx, regRepo,
		"44444444-4444-4444-4444-670000000003", game.ID, otherID)

	got, err := regRepo.ListForPlayer(ctx, playerID)
	if err != nil {
		t.Fatalf("ListForPlayer: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListForPlayer returned %d listings, want 1 (the caller's own) — "+
			"two players are registered for this Game", len(got))
	}
	l := got[0]

	if l.Registration.PlayerID != playerID {
		t.Errorf("Registration.PlayerID = %q, want %q", l.Registration.PlayerID, playerID)
	}
	if l.Registration.GameID != game.ID {
		t.Errorf("Registration.GameID = %q, want %q", l.Registration.GameID, game.ID)
	}
	if l.Registration.Status != domain.RegistrationStatusRegistered {
		t.Errorf("Registration.Status = %q, want %q",
			l.Registration.Status, domain.RegistrationStatusRegistered)
	}
	if l.Registration.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Errorf("Registration.PaymentStatus = %q, want %q",
			l.Registration.PaymentStatus, domain.PaymentStatusUnpaid)
	}
	if l.Registration.AmountOwed.Cents != 1500 {
		t.Errorf("Registration.AmountOwed.Cents = %d, want 1500", l.Registration.AmountOwed.Cents)
	}

	// The projected Game columns. `game_status` in particular is the aliased
	// one — `r.status` already claims `status`, so a missing alias would make
	// sqlc produce two same-named fields and this assertion is what proves the
	// right one landed here.
	if !l.GameStartsAt.Equal(game.Range.Start) {
		t.Errorf("GameStartsAt = %v, want %v", l.GameStartsAt, game.Range.Start)
	}
	if !l.GameEndsAt.Equal(game.Range.End) {
		t.Errorf("GameEndsAt = %v, want %v", l.GameEndsAt, game.Range.End)
	}
	if l.GameStatus != domain.StatusScheduled {
		t.Errorf("GameStatus = %q, want %q", l.GameStatus, domain.StatusScheduled)
	}
	if len(l.GameCourtIDs) != 1 || l.GameCourtIDs[0] != seedCourtID {
		t.Errorf("GameCourtIDs = %v, want [%s]", l.GameCourtIDs, seedCourtID)
	}
	// venue_facility_id was seeded empty (nullable column). It must come back
	// as the empty string, NOT the all-zeros uuid a bare pgtype.UUID.String()
	// would render — a client would read that as a real Facility.
	if l.GameVenueFacilityID != "" {
		t.Errorf("GameVenueFacilityID = %q, want \"\" — the column was NULL, and "+
			"the all-zeros uuid would read as a real Facility", l.GameVenueFacilityID)
	}
}

// TestListForPlayer_HostCancelledGameIsReturnedButSelfCancelledIsNot is the
// assertion that justifies the whole feature, and the one that needs a real
// database most.
//
// The two cases are INDISTINGUISHABLE by the registration's own status —
// app.Service.CancelGame cascades CancelAllActiveRegistrationsForGame, so both
// rows read `status = 'cancelled'`. Only the Game's status separates them, and
// only the query's `(r.status <> 'cancelled' OR g.status = 'cancelled')`
// disjunct acts on that. A query filtering on the registration alone passes
// every other test in this file and silently drops the host-cancelled row — the
// one fact a Player cannot learn any other way, since ListGames filters
// `WHERE g.status = 'scheduled'` and this service has no GetGame RPC.
func TestListForPlayer_HostCancelledGameIsReturnedButSelfCancelledIsNot(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	hostID := seedSocialplayUser(t, ctx, pool, "cancel-host")
	playerID := seedSocialplayUser(t, ctx, pool, "cancel-player")

	// Game A: the Host cancels it. Its registration is cascaded to cancelled.
	gameA := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000011", hostID,
		"2026-12-05T09:00:00Z", "2026-12-05T10:00:00Z")
	seedPlayerRegistration(t, ctx, regRepo,
		"44444444-4444-4444-4444-670000000012", gameA.ID, playerID)

	// Game B: the Player withdraws. Still scheduled.
	gameB := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000013", hostID,
		"2026-12-06T09:00:00Z", "2026-12-06T10:00:00Z")
	regB := seedPlayerRegistration(t, ctx, regRepo,
		"44444444-4444-4444-4444-670000000014", gameB.ID, playerID)

	// Game C: untouched, the control. Without it a query returning nothing at
	// all would satisfy half the assertions below.
	gameC := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000015", hostID,
		"2026-12-07T09:00:00Z", "2026-12-07T10:00:00Z")
	seedPlayerRegistration(t, ctx, regRepo,
		"44444444-4444-4444-4444-670000000016", gameC.ID, playerID)

	// Cancel Game A the way app.Service.CancelGame does: the Game's own status
	// write, then the bulk cascade over its Registrations. Driven through the
	// repository methods rather than raw SQL so the fixture cannot drift from
	// what production actually writes.
	if _, err := gameRepo.UpdateStatus(ctx, gameA.ID, domain.StatusCancelled); err != nil {
		t.Fatalf("cancelling Game A: %v", err)
	}
	n, err := regRepo.CancelAllActiveForGame(ctx, gameA.ID)
	if err != nil {
		t.Fatalf("cascading Game A's cancellation: %v", err)
	}
	if n != 1 {
		t.Fatalf("the cascade transitioned %d registrations, want 1", n)
	}

	// The Player withdraws from Game B.
	regB.Status = domain.RegistrationStatusCancelled
	if _, err := regRepo.Update(ctx, regB); err != nil {
		t.Fatalf("the Player withdrawing from Game B: %v", err)
	}

	got, err := regRepo.ListForPlayer(ctx, playerID)
	if err != nil {
		t.Fatalf("ListForPlayer: %v", err)
	}

	byGame := map[string]domain.Status{}
	for _, l := range got {
		byGame[l.Registration.GameID] = l.GameStatus
	}

	if len(got) != 2 {
		t.Fatalf("ListForPlayer returned %d listings, want 2 "+
			"(the host-cancelled Game A and the untouched Game C, not the "+
			"self-cancelled Game B): got %v", len(got), byGame)
	}
	if st, ok := byGame[gameA.ID]; !ok {
		t.Errorf("the host-cancelled Game A is MISSING — this is the one fact a "+
			"Player cannot learn any other way (ListGames filters cancelled Games "+
			"out and there is no GetGame RPC). Got %v", byGame)
	} else if st != domain.StatusCancelled {
		t.Errorf("Game A's GameStatus = %q, want %q", st, domain.StatusCancelled)
	}
	if _, ok := byGame[gameB.ID]; ok {
		t.Errorf("the self-cancelled Game B came back — the Player withdrew and "+
			"already knows. Got %v", byGame)
	}
	if st, ok := byGame[gameC.ID]; !ok {
		t.Errorf("the untouched Game C is missing. Got %v", byGame)
	} else if st != domain.StatusScheduled {
		t.Errorf("Game C's GameStatus = %q, want %q", st, domain.StatusScheduled)
	}
}

// TestListForPlayer_OrdersByTheGamesStart pins `ORDER BY g.starts_at`. The
// rows are inserted in the reverse of the expected order, so a query with no
// ORDER BY would have to be right by accident twice.
func TestListForPlayer_OrdersByTheGamesStart(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	hostID := seedSocialplayUser(t, ctx, pool, "order-host")
	playerID := seedSocialplayUser(t, ctx, pool, "order-player")

	later := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000021", hostID,
		"2026-12-20T09:00:00Z", "2026-12-20T10:00:00Z")
	middle := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000022", hostID,
		"2026-12-15T09:00:00Z", "2026-12-15T10:00:00Z")
	sooner := seedPlayerGame(t, ctx, gameRepo,
		"44444444-4444-4444-4444-670000000023", hostID,
		"2026-12-10T09:00:00Z", "2026-12-10T10:00:00Z")

	for i, g := range []domain.Game{later, middle, sooner} {
		seedPlayerRegistration(t, ctx, regRepo,
			[]string{
				"44444444-4444-4444-4444-670000000031",
				"44444444-4444-4444-4444-670000000032",
				"44444444-4444-4444-4444-670000000033",
			}[i], g.ID, playerID)
	}

	got, err := regRepo.ListForPlayer(ctx, playerID)
	if err != nil {
		t.Fatalf("ListForPlayer: %v", err)
	}
	want := []string{sooner.ID, middle.ID, later.ID}
	if len(got) != len(want) {
		t.Fatalf("ListForPlayer returned %d listings, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].Registration.GameID != id {
			t.Errorf("listing[%d].Registration.GameID = %q, want %q",
				i, got[i].Registration.GameID, id)
		}
	}
}

// TestListForPlayer_UnknownPlayerIsAnEmptyList pins the "empty result, not an
// error" contract at the storage layer. The id is well-formed and references no
// User — the Postgres path a malformed id never reaches (the app layer's
// uuidShape guard stops that one before mustUUID can panic).
func TestListForPlayer_UnknownPlayerIsAnEmptyList(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	regRepo := socialplaypg.NewRegistrationRepository(pool)

	got, err := regRepo.ListForPlayer(ctx, "44444444-4444-4444-4444-670000000099")
	if err != nil {
		t.Fatalf("ListForPlayer for an unknown player: %v", err)
	}
	if got == nil {
		t.Fatal("ListForPlayer returned nil, want an empty slice")
	}
	if len(got) != 0 {
		t.Errorf("ListForPlayer returned %d listings for an unknown player, want 0", len(got))
	}
}

// TestRegistrationsPlayerIndexExists pins db/migrations/0033 itself.
//
// CLAUDE.md's standing lesson is that a DB-level guard with no DB-level test is
// a comment; an index is the same shape — nothing in Go can observe it, so
// without this test the migration could be reverted, renamed, or never applied
// and every other test in this package would stay green.
//
// It asserts the index's indexed COLUMN SET rather than its name alone, read
// out of `pg_index`, so an index renamed-but-equivalent still passes while one
// that happens to carry the right name over the wrong column does not. It also
// asserts the index is NOT partial — a partial index on the active rows alone
// would silently stop covering the host-cancelled case above.
func TestRegistrationsPlayerIndexExists(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	// Every single-column index on registrations(player_id), with whether it
	// carries a WHERE predicate. Derived from the catalog rather than compared
	// against a hardcoded name list.
	rows, err := pool.Query(ctx, `
		SELECT c.relname, i.indpred IS NOT NULL AS is_partial
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = i.indkey[0]
		WHERE t.relname = 'registrations'
		  AND i.indnatts = 1
		  AND a.attname = 'player_id'`)
	if err != nil {
		t.Fatalf("querying pg_index: %v", err)
	}
	defer rows.Close()

	type idx struct {
		name    string
		partial bool
	}
	var found []idx
	for rows.Next() {
		var got idx
		if err := rows.Scan(&got.name, &got.partial); err != nil {
			t.Fatalf("scanning pg_index row: %v", err)
		}
		found = append(found, got)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating pg_index rows: %v", err)
	}

	if len(found) == 0 {
		t.Fatalf("no single-column index on registrations (player_id) exists — " +
			"db/migrations/0033 is the only thing that creates one, and without it " +
			"ListRegistrationsForPlayer sequentially scans the table on every call. " +
			"Note registrations_active_player_per_game_idx does NOT serve this: " +
			"game_id is its leading column.")
	}
	for _, got := range found {
		if got.partial {
			t.Errorf("index %q on registrations (player_id) is PARTIAL — "+
				"ListRegistrationsForPlayer returns cancelled registration rows "+
				"whose Game was cancelled, so a partial index would stop covering "+
				"exactly the case this read exists for", got.name)
		}
	}
}
