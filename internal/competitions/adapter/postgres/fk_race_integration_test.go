//go:build integration

// T17.3 (part of #195, the Competitions mirror of T17.2's identical fix for
// Social Play) — the load-bearing half of this ticket's regression proof
// that no in-memory fixture can supply.
//
// The gap: competition_entries.competition_id and
// competitions.venue_facility_id are both FK-backed writes guarded by an
// app-level read immediately beforehand
// (app.Service.EnterCompetition's s.competitions.GetByID;
// ScheduleCompetition's s.facilities.FacilityExists). In the non-racing
// case that read's own not-found sentinel wins and the FK never fires —
// translate_test.go's unit tests already pin what
// translateEntryErr/translateErr do WHEN a 23503 arrives, using a synthetic
// *pgconn.PgError, but they cannot prove Postgres actually raises 23503 for
// these two writes, and no in-memory port fake models a table that can be
// deleted out from under a caller mid-request — the same fixture-infidelity
// trap #185 was caught in twice, per #195's own body. This file is where
// that assumption is checked against a real database, in the SPECIFIC
// narrow window #195 is about: the parent deleted between the guarding read
// and the insert that follows it, not merely an insert against an id that
// was already unknown from the start.
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker. NOT run in this environment (no Docker
// daemon) — see this ticket's PR description for the mutation-check
// (temporarily removing translateEntryErr/translateErr's new 23503 arms and
// confirming translate_test.go's unit-level cases fail) that stands in for
// actually executing this file here, per this ticket's own instruction 5.
//
// Shares newTestPool/waitForReady/applyMigrations/mustRange/seedCourtID
// (capacity_concurrency_integration_test.go) with this package's other
// integration tests rather than re-declaring container boot/migrate
// boilerplate a third time.
package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	competitionspg "github.com/nhuthuynh/white-label/internal/competitions/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestCreateCompetitionEntry_CompetitionDeletedBetweenGuardReadAndInsertIsCompetitionNotFound
// is competition_entries.competition_id's half of #195's race.
//
// Sequence, deliberately linear rather than driven by goroutines: Go's own
// ordering already produces the interleaving #195 describes (guarding read
// completes and observes the Competition, THEN the Competition is removed,
// THEN the insert that read believed was safe runs) without adding a
// synchronization primitive whose own correctness would need proving. A
// goroutine-based race would non-deterministically also cover the "delete
// never lands in the window" case, which is not the case this ticket is
// about and is already covered by the
// TestEnterCompetition_*UnderConcurrency-style tests in
// capacity_concurrency_integration_test.go; this test's job is to pin what
// happens when the delete DOES land in the window, every run.
func TestCreateCompetitionEntry_CompetitionDeletedBetweenGuardReadAndInsertIsCompetitionNotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	repo := competitionspg.NewRepository(pool)

	t.Run("competition deleted after the guarding read answers ErrCompetitionNotFound, not a raw 23503", func(t *testing.T) {
		competition := seedCompetition(t, ctx, repo, "66666666-6666-6666-6666-660000000001", 16, 2, "fk-race-entries-token-1")

		// The guarding read app.Service.EnterCompetition performs before its
		// own insert — proven to succeed here, exactly as it would for a
		// real request that arrives before the race window closes.
		if _, err := repo.GetByID(ctx, competition.ID); err != nil {
			t.Fatalf("guarding read: %v", err)
		}

		// The race: the Competition is deleted in the gap between that read
		// succeeding and the insert below running. A raw SQL DELETE —
		// neither this context nor any other exposes a domain-level "delete
		// a Competition" (#195's own body: "none of these contexts
		// currently ships a bulk-delete or cascading-delete flow"), so this
		// is the only way the window can be produced at all. Cascades
		// cleanly to competition_sessions (ON DELETE CASCADE,
		// db/migrations/0014_competitions.sql), leaving no orphaned row
		// behind for a later test to trip over.
		if _, err := pool.Exec(ctx, "DELETE FROM competitions WHERE id = $1", competition.ID); err != nil {
			t.Fatalf("failed to delete fixture competition: %v", err)
		}

		entry := domain.CompetitionEntry{
			ID:            "66666666-6666-6666-6666-660000000002",
			CompetitionID: competition.ID,
			PlayerID:      "race-player",
			Source:        domain.EntrySourceApp,
			Status:        domain.EntryStatusEntered,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}
		_, err := repo.CreateEntry(ctx, entry)
		if !errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Fatalf("CreateEntry(entry against a concurrently-deleted competition) = %v, want domain.ErrCompetitionNotFound", err)
		}

		// The inverse guard: a raw, untranslated driver error must never
		// survive to the caller — proves this went through
		// translateEntryErr's 23503 arm, not its wrapped default, which
		// would still satisfy errors.As for *pgconn.PgError because
		// fmt.Errorf's %w keeps the chain intact.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			t.Fatalf("CreateEntry leaked a raw *pgconn.PgError instead of a translated domain error: %v", err)
		}

		// The inverse-inverse guard: three different constraints fire 23xxx
		// codes on this same table (23505 on the active-player unique
		// index, P0001 from the capacity trigger, and now this 23503) — a
		// mismap onto either sibling sentinel must fail this test too.
		if errors.Is(err, domain.ErrCompetitionFull) || errors.Is(err, domain.ErrAlreadyEntered) {
			t.Fatalf("CreateEntry(entry against a concurrently-deleted competition) = %v, must not be mistaken for the capacity guard or the unique index", err)
		}
	})

	// Control: without it, a repository that rejected every insert would
	// satisfy the subtest above for the wrong reason.
	t.Run("control: entering a competition that was never deleted still succeeds", func(t *testing.T) {
		competition := seedCompetition(t, ctx, repo, "66666666-6666-6666-6666-660000000003", 16, 2, "fk-race-entries-token-2")

		entry := domain.CompetitionEntry{
			ID:            "66666666-6666-6666-6666-660000000004",
			CompetitionID: competition.ID,
			PlayerID:      "race-player-2",
			Source:        domain.EntrySourceApp,
			Status:        domain.EntryStatusEntered,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}
		if _, err := repo.CreateEntry(ctx, entry); err != nil {
			t.Fatalf("CreateEntry(entry against a real competition) = %v, want success", err)
		}
	})
}

// TestCreateCompetition_VenueFacilityDeletedBetweenGuardReadAndInsertIsFacilityNotFound
// is competitions.venue_facility_id's half of #195's race — the direct
// sibling of the test above, over the Competitions repository and a
// Facilities-context fixture instead of a Competition one.
//
// The guarding read is modelled at the DB layer
// port.FacilityLookup.FacilityExists ultimately relies on
// (facilitiespg.Repository.GetFacilityByID) rather than by wiring the full
// Facilities app.Service: the cross-context lookup wiring itself
// (facilitiesdomain.ErrFacilityNotFound -> this context's own
// domain.ErrFacilityNotFound) is already covered by
// internal/competitions/adapter/facilities/lookup_test.go; this file's job
// is only the FK translation the DB layer beneath that lookup shares with
// it.
func TestCreateCompetition_VenueFacilityDeletedBetweenGuardReadAndInsertIsFacilityNotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	facilityRepo := facilitiespg.NewRepository(pool)
	competitionRepo := competitionspg.NewRepository(pool)

	t.Run("facility deleted after the guarding read answers ErrFacilityNotFound, not a raw 23503", func(t *testing.T) {
		facility, err := facilitiesdomain.NewFacility(
			"77777777-7777-7777-7777-770000000001",
			"77777777-7777-7777-7777-7700000000f1",
			"Race Facility", "", "1 Race St", nil,
		)
		if err != nil {
			t.Fatalf("bad fixture facility: %v", err)
		}
		facility, err = facilityRepo.CreateFacility(ctx, facility)
		if err != nil {
			t.Fatalf("failed to create fixture facility: %v", err)
		}

		// The guarding read app.Service.ScheduleCompetition performs via
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

		competition, err := domain.NewCompetition(
			"77777777-7777-7777-7777-770000000002", "race-host", "Race Open", facility.ID,
			[]domain.Session{{
				Range:    mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T12:00:00Z"),
				CourtIDs: []string{seedCourtID},
			}},
			16, 2,
			domain.PaymentMethodEither,
			domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
			domain.FormatDoubles,
			"fk-race-venue-token-1",
		)
		if err != nil {
			t.Fatalf("bad fixture competition: %v", err)
		}

		_, err = competitionRepo.Create(ctx, competition)
		if !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("Create(competition against a concurrently-deleted facility) = %v, want domain.ErrFacilityNotFound", err)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			t.Fatalf("Create leaked a raw *pgconn.PgError instead of a translated domain error: %v", err)
		}
		if errors.Is(err, domain.ErrCompetitionNotFound) {
			t.Fatalf("Create(competition against a concurrently-deleted facility) = %v, must not be mistaken for the pgx.ErrNoRows arm (wrong condition)", err)
		}

		// The inverse check the FK's own comment (repository.go) warns
		// about: a Competition with no venue Facility set at all
		// (VenueFacilityID == "") must not be mistaken for this case —
		// nullableUUID skips the column entirely rather than sending an
		// empty string through mustUUID.
		competitionNoVenue, err := domain.NewCompetition(
			"77777777-7777-7777-7777-770000000003", "race-host", "Race Open No Venue", "",
			[]domain.Session{{
				Range:    mustRange(t, "2026-09-05T09:00:00Z", "2026-09-05T12:00:00Z"),
				CourtIDs: []string{seedCourtID},
			}},
			16, 2,
			domain.PaymentMethodEither,
			domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
			domain.FormatDoubles,
			"fk-race-venue-token-2",
		)
		if err != nil {
			t.Fatalf("bad fixture competition: %v", err)
		}
		if _, err := competitionRepo.Create(ctx, competitionNoVenue); err != nil {
			t.Fatalf("Create(competition with no venue facility) = %v, want success", err)
		}
	})

	// Control: without it, a repository that rejected every insert would
	// satisfy the subtest above for the wrong reason.
	t.Run("control: a competition against a facility that was never deleted still succeeds", func(t *testing.T) {
		facility, err := facilitiesdomain.NewFacility(
			"77777777-7777-7777-7777-770000000004",
			"77777777-7777-7777-7777-7700000000f2",
			"Race Facility 2", "", "2 Race St", nil,
		)
		if err != nil {
			t.Fatalf("bad fixture facility: %v", err)
		}
		facility, err = facilityRepo.CreateFacility(ctx, facility)
		if err != nil {
			t.Fatalf("failed to create fixture facility: %v", err)
		}

		competition, err := domain.NewCompetition(
			"77777777-7777-7777-7777-770000000005", "race-host-2", "Race Open 2", facility.ID,
			[]domain.Session{{
				Range:    mustRange(t, "2026-09-05T13:00:00Z", "2026-09-05T16:00:00Z"),
				CourtIDs: []string{seedCourtID},
			}},
			16, 2,
			domain.PaymentMethodEither,
			domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
			domain.FormatDoubles,
			"fk-race-venue-token-3",
		)
		if err != nil {
			t.Fatalf("bad fixture competition: %v", err)
		}
		if _, err := competitionRepo.Create(ctx, competition); err != nil {
			t.Fatalf("Create(competition against a real facility) = %v, want success", err)
		}
	})
}
