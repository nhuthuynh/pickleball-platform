// Package postgres is the Social Play context's Postgres adapter. It
// implements port.GameRepository and port.RegistrationRepository against
// sqlc-generated queries and translates Postgres errors into domain errors
// (CLAUDE.md rule 5). It only compiles after `make generate` has produced
// internal/gen/socialplaydb (see CLAUDE.md gotchas); that package is
// gitignored and not committed.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	socialplaydb "github.com/nhuthuynh/white-label/internal/gen/socialplaydb"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// Postgres error codes this adapter cares about. 23505 is the unique
// violation on registrations_active_player_per_game_idx (db/migrations/
// 0005_socialplay.sql) firing — the DB-level mirror of
// domain.ErrAlreadyRegistered per CLAUDE.md rule 4.
const pgUniqueViolation = "23505"

// GameRepository implements port.GameRepository.
type GameRepository struct {
	q *socialplaydb.Queries
}

func NewGameRepository(pool *pgxpool.Pool) *GameRepository {
	return &GameRepository{q: socialplaydb.New(pool)}
}

func (r *GameRepository) Create(ctx context.Context, g domain.Game) (domain.Game, error) {
	row, err := r.q.CreateGame(ctx, socialplaydb.CreateGameParams{
		ID:         mustUUID(g.ID),
		HostID:     g.HostID,
		FacilityID: g.FacilityID,
		CourtIds:   mustUUIDs(g.CourtIDs),
		StartsAt:   toTimestamptz(g.Range.Start),
		EndsAt:     toTimestamptz(g.Range.End),
		Capacity:   int32(g.Capacity),
		Status:     string(g.Status),
	})
	if err != nil {
		return domain.Game{}, translateGameErr(err)
	}
	return gameFromFields(row.ID, row.HostID, row.FacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status), nil
}

func (r *GameRepository) GetByID(ctx context.Context, id string) (domain.Game, error) {
	row, err := r.q.GetGameByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Game{}, translateGameErr(err)
	}
	return gameFromFields(row.ID, row.HostID, row.FacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status), nil
}

func translateGameErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrGameNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (games): %w", err)
}

// gameFromFields builds a domain.Game from the 8 columns every games query
// selects. sqlc generates a distinct Row struct per query (CreateGameRow,
// GetGameByIDRow, ...) rather than reusing socialplaydb.Game, mirroring the
// fromFields pattern in internal/booking/adapter/postgres/repository.go
// (CLAUDE.md gotcha: sqlc emits a distinct ...Row type per query).
func gameFromFields(id pgtype.UUID, hostID, facilityID string, courtIDs []pgtype.UUID, startsAt, endsAt pgtype.Timestamptz, capacity int32, status string) domain.Game {
	ids := make([]string, 0, len(courtIDs))
	for _, c := range courtIDs {
		ids = append(ids, c.String())
	}
	return domain.Game{
		ID:         id.String(),
		HostID:     hostID,
		FacilityID: facilityID,
		CourtIDs:   ids,
		Range: domain.TimeRange{
			Start: startsAt.Time,
			End:   endsAt.Time,
		},
		Capacity: int(capacity),
		Status:   domain.Status(status),
	}
}

// RegistrationRepository implements port.RegistrationRepository.
type RegistrationRepository struct {
	q *socialplaydb.Queries
}

func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{q: socialplaydb.New(pool)}
}

func (r *RegistrationRepository) Create(ctx context.Context, reg domain.Registration) (domain.Registration, error) {
	row, err := r.q.CreateRegistration(ctx, socialplaydb.CreateRegistrationParams{
		ID:            mustUUID(reg.ID),
		GameID:        mustUUID(reg.GameID),
		PlayerID:      reg.PlayerID,
		Source:        string(reg.Source),
		Status:        string(reg.Status),
		PaymentStatus: string(reg.PaymentStatus),
	})
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus), nil
}

func (r *RegistrationRepository) GetByID(ctx context.Context, id string) (domain.Registration, error) {
	row, err := r.q.GetRegistrationByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus), nil
}

func (r *RegistrationRepository) ListActiveForGame(ctx context.Context, gameID string) ([]domain.Registration, error) {
	rows, err := r.q.ListActiveRegistrationsForGame(ctx, mustUUID(gameID))
	if err != nil {
		return nil, translateRegistrationErr(err)
	}
	out := make([]domain.Registration, 0, len(rows))
	for _, row := range rows {
		out = append(out, registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus))
	}
	return out, nil
}

func (r *RegistrationRepository) Update(ctx context.Context, reg domain.Registration) (domain.Registration, error) {
	row, err := r.q.UpdateRegistrationStatus(ctx, socialplaydb.UpdateRegistrationStatusParams{
		ID:     mustUUID(reg.ID),
		Status: string(reg.Status),
	})
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus), nil
}

// translateRegistrationErr maps infrastructure failures onto domain errors
// — the only errors allowed to cross out of this package (CLAUDE.md rule
// 5). A 23505 unique_violation on registrations_active_player_per_game_idx
// becomes domain.ErrAlreadyRegistered, the DB-level guard's counterpart to
// domain.Register's own pre-check (CLAUDE.md rule 4).
func translateRegistrationErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrAlreadyRegistered
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRegistrationNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (registrations): %w", err)
}

// registrationFromFields builds a domain.Registration from the 6 columns
// every registrations query selects — see gameFromFields's doc comment for
// why this pattern exists.
func registrationFromFields(id, gameID pgtype.UUID, playerID, source, status, paymentStatus string) domain.Registration {
	return domain.Registration{
		ID:            id.String(),
		GameID:        gameID.String(),
		PlayerID:      playerID,
		Source:        domain.RegistrationSource(source),
		Status:        domain.RegistrationStatus(status),
		PaymentStatus: domain.PaymentStatus(paymentStatus),
	}
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		// Social Play IDs are generated by port.IDGenerator (adapter/idgen)
		// as UUIDs; a malformed ID here means an upstream invariant was
		// already violated, which is a programmer error, not a runtime one
		// — mirrors internal/booking/adapter/postgres/repository.go's
		// mustUUID.
		panic(fmt.Sprintf("socialplay postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}

func mustUUIDs(ss []string) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ss))
	for _, s := range ss {
		out = append(out, mustUUID(s))
	}
	return out
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
