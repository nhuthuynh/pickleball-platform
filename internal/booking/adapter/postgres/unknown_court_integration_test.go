//go:build integration

// T15.6 (closes issue #185). This is the half of the regression proof that
// NO in-memory fixture can supply, and saying why is the point of this file
// existing separately from foreign_key_test.go beside it.
//
// The defect: a well-formed but unknown court id — a syntactically valid UUID
// naming no row in `courts` — answered codes.Internal (HTTP 500) instead of a
// client error. app.Service.CreateBooking's guard is `uuidShape.MatchString`,
// a SHAPE check; nothing above the database knows which courts exist. The
// only thing that can tell an unknown court from a real one is
// `bookings.court_id uuid NOT NULL REFERENCES courts (id)`
// (db/migrations/0001_init.sql:24), and that constraint lives in Postgres.
//
// #185's testing note names the consequence precisely: an in-memory fake
// accepts the unknown court id and the test passes whether or not the guard
// exists. That is why T14.8's tests did not surface this while closing the
// malformed half of the same defect class (#156), and it is the
// fixture-infidelity trap docs/LESSONS.md' T9 entry describes. The unit tests
// in foreign_key_test.go pin translateErr's 23503 -> sentinel mapping, and the
// FK-modelling fakes in the three grpcapi packages pin sentinel -> gRPC code,
// but BOTH assume Postgres raises 23503 here at all. This file is where that
// assumption is checked against a real database rather than asserted.
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker.
package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	bookinggrpc "github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	competitionsbooking "github.com/nhuthuynh/white-label/internal/competitions/adapter/booking"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	socialplaybooking "github.com/nhuthuynh/white-label/internal/socialplay/adapter/booking"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"

	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// unknownCourtID is uuid-shaped and inserted into `courts` by no migration.
// seedCourtID (concurrency_integration_test.go) is the one 0002_seed.sql
// really creates, so the two differ only in whether the FK accepts them.
const unknownCourtID = "11111111-1111-1111-1111-1111111111ff"

// startPostgres brings up a real Postgres with the repo's own migrations
// applied, reusing concurrency_integration_test.go's waitForReady and
// applyMigrations so both files test against the same schema a real deploy
// gets rather than a hand-rolled test schema that can drift.
func startPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("pickleball"),
		tcpostgres.WithUsername("pickleball"),
		tcpostgres.WithPassword("pickleball"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	waitForReady(t, ctx, pool)
	applyMigrations(t, ctx, pool)
	return pool
}

func newBookingService(pool *pgxpool.Pool) *bookingapp.Service {
	return bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       bookingpg.NewRepository(pool),
		PricingRules:   bookingpg.NewPricingRuleRepository(pool),
		DiscountRules:  bookingpg.NewDiscountRuleRepository(pool),
		RecurringHires: unusedRecurringHireRepository{},
		Facilities:     unusedFacilityLookup{},
		Identity:       unusedIdentityLookup{},
		IDs:            idgen.UUID{},
	})
}

// TestCreateBooking_UnknownCourtIsTranslatedByRealPostgres is the load-bearing
// claim: against a REAL database, a well-formed unknown court id produces a
// 23503 foreign-key violation that the adapter translates into a domain
// sentinel — not a raw, wrapped infra error.
func TestCreateBooking_UnknownCourtIsTranslatedByRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	svc := newBookingService(pool)
	owner := seedBookingOwner(t, ctx, pool)
	rng := mustRange(t, "2026-09-02T09:00:00Z", "2026-09-02T10:00:00Z")

	_, err := svc.CreateBooking(ctx, bookingapp.CreateBookingInput{
		CourtID:     unknownCourtID,
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: owner,
	})

	if !errors.Is(err, domain.ErrInvalidCourtReference) {
		t.Fatalf("CreateBooking(unknown court) = %v, want domain.ErrInvalidCourtReference", err)
	}
	// The inverse guard: a real FK violation must not be mistaken for the
	// EXCLUDE constraint. 23503 and 23P01 are different constraints with
	// different answers (404 vs 409) and the same 23xxx prefix.
	if errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("an unknown court was misclassified as a double booking: %v", err)
	}
}

// TestCreateBooking_KnownCourtStillSucceedsAgainstRealPostgres is the control.
// Without it, an adapter that rejected every INSERT would satisfy the test
// above.
func TestCreateBooking_KnownCourtStillSucceedsAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	svc := newBookingService(pool)
	owner := seedBookingOwner(t, ctx, pool)
	rng := mustRange(t, "2026-09-02T09:00:00Z", "2026-09-02T10:00:00Z")

	b, err := svc.CreateBooking(ctx, bookingapp.CreateBookingInput{
		CourtID:     seedCourtID,
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: owner,
	})
	if err != nil {
		t.Fatalf("CreateBooking(seeded court) = %v, want success", err)
	}
	if b.CourtID != seedCourtID {
		t.Fatalf("persisted booking court = %q, want %q", b.CourtID, seedCourtID)
	}
}

// TestCreateBookingRPC_UnknownCourtIsNotFoundAgainstRealPostgres walks the
// whole Booking stack a REST client walks — grpcapi.Handler -> app.Service ->
// the real Postgres adapter -> a real FK — and asserts the status code, not
// just the sentinel. grpc-gateway maps NotFound to HTTP 404, so this is the
// end-to-end statement of what #185 asked for on Booking's own RPC.
func TestCreateBookingRPC_UnknownCourtIsNotFoundAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	owner := seedBookingOwner(t, ctx, pool)

	// CreateBooking became an authenticated RPC in T55.1 (DECISION D1), so
	// this end-to-end walk now needs both halves of ADR-0014's seam: a
	// principal on the context, and an IdentityLookup that resolves its
	// subject to the seeded User.ID. unusedIdentityLookup (which fails closed)
	// cannot serve that, so this one test builds its service with a lookup
	// that can. The court-resolution behaviour under test is unchanged.
	h := bookinggrpc.NewHandler(bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       bookingpg.NewRepository(pool),
		PricingRules:   bookingpg.NewPricingRuleRepository(pool),
		DiscountRules:  bookingpg.NewDiscountRuleRepository(pool),
		RecurringHires: unusedRecurringHireRepository{},
		Facilities:     unusedFacilityLookup{},
		Identity:       fixedIdentityLookup{userID: owner},
		IDs:            idgen.UUID{},
	}))
	rng := mustRange(t, "2026-09-02T09:00:00Z", "2026-09-02T10:00:00Z")

	authed := auth.ContextWithPrincipal(ctx, auth.Principal{
		Subject: "auth0|t55-booking-owner",
		Issuer:  "https://issuer.test/",
	})

	_, err := h.CreateBooking(authed, &bookingv1.CreateBookingRequest{
		CourtId:  unknownCourtID,
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
		StartsAt: timestamppb.New(rng.Start),
		EndsAt:   timestamppb.New(rng.End),
	})

	if got := status.Code(err); got == codes.Internal {
		t.Fatalf("CreateBooking RPC(unknown court) still answers Internal — #185 is not fixed (err: %v)", err)
	} else if got != codes.NotFound {
		t.Fatalf("CreateBooking RPC(unknown court) code = %v (err: %v), want NotFound", got, err)
	}
}

// TestCrossContextReservations_UnknownCourtAgainstRealPostgres covers the
// other two RPCs #185 names. Both Social Play's ScheduleGame and Competitions'
// ScheduleCompetition reach the same FK through their own
// port.CourtReservation adapter over this same bookingapp.Service, and both
// adapters deliberately strip the Booking error's type at the boundary (%s,
// not %w — CLAUDE.md rule 5), which is exactly why nothing survived for their
// toStatus to classify before this ticket.
//
// The adapters are driven directly rather than through each context's full
// app.Service: the translation under test is theirs, the sentinel -> NotFound
// half is pinned by each context's own error_mapping_test table, and wiring
// two more contexts' repositories here would test their persistence rather
// than this FK.
func TestCrossContextReservations_UnknownCourtAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	svc := newBookingService(pool)
	owner := seedBookingOwner(t, ctx, pool)
	rng := mustRange(t, "2026-09-02T11:00:00Z", "2026-09-02T12:00:00Z")

	t.Run("social play", func(t *testing.T) {
		res := socialplaybooking.NewReservation(svc)

		_, err := res.ReserveCourt(ctx, unknownCourtID, rng.Start, rng.End, "game-ref", owner)
		if !errors.Is(err, socialplaydomain.ErrCourtNotFound) {
			t.Fatalf("ReserveCourt(unknown court) = %v, want socialplay ErrCourtNotFound", err)
		}
		if errors.Is(err, domain.ErrInvalidCourtReference) {
			t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
		}

		// Control: the seeded court still reserves cleanly through the same
		// adapter, so the assertion above is about this court id and not
		// about the adapter refusing everything.
		if _, err := res.ReserveCourt(ctx, seedCourtID, rng.Start, rng.End, "game-ref", owner); err != nil {
			t.Fatalf("ReserveCourt(seeded court) = %v, want success", err)
		}
	})

	t.Run("competitions", func(t *testing.T) {
		res := competitionsbooking.NewReservation(svc)
		// A slot that does not overlap Social Play's above, so the seeded
		// court's control reservation is rejected by the FK's absence rather
		// than by the EXCLUDE constraint.
		compRng := mustRange(t, "2026-09-02T13:00:00Z", "2026-09-02T14:00:00Z")

		_, err := res.ReserveCourt(ctx, unknownCourtID, compRng.Start, compRng.End, "comp-ref", owner)
		if !errors.Is(err, competitionsdomain.ErrCourtNotFound) {
			t.Fatalf("ReserveCourt(unknown court) = %v, want competitions ErrCourtNotFound", err)
		}
		if errors.Is(err, domain.ErrInvalidCourtReference) {
			t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
		}

		if _, err := res.ReserveCourt(ctx, seedCourtID, compRng.Start, compRng.End, "comp-ref", owner); err != nil {
			t.Fatalf("ReserveCourt(seeded court) = %v, want success", err)
		}
	})
}

// seedBookingOwner inserts an identity_users row and returns its id, for use
// as a Booking's OwnerUserID.
//
// Required since DECISION D1 (ADR-0015 option (a)): bookings.owner_user_id is
// `uuid NOT NULL REFERENCES identity_users (id)` (migration 0027), so an
// integration test that creates a Booking must have a real user row to point
// at — an invented uuid fails the FK, which is the constraint doing its job
// rather than a test-harness inconvenience to route around.
//
// It mints its own id rather than reusing the package's other user fixture so
// that tests using both do not collide on the primary key.
func seedBookingOwner(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()

	id := uuid.NewString()
	_, err := pool.Exec(ctx,
		`INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level) VALUES ($1, $2, $3, $4)`,
		id, "T55.1 booking owner", []string{"player"}, 3)
	if err != nil {
		t.Fatalf("seed booking owner: %v", err)
	}
	return id
}

// fixedIdentityLookup resolves any subject to one fixed User.ID.
//
// It exists for the single RPC-level test above, which needs ADR-0014's
// subject -> User.ID translation to actually succeed. unusedIdentityLookup
// deliberately fails closed and is still what every other test here uses;
// this one is scoped to the one case that genuinely walks the actor path, so
// a wiring mistake elsewhere still surfaces rather than being absorbed by a
// permissive stub.
type fixedIdentityLookup struct{ userID string }

func (l fixedIdentityLookup) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return l.userID, nil
}

func (fixedIdentityLookup) EnsureClubRole(_ context.Context, _ string) error {
	return domain.ErrNotClub
}
