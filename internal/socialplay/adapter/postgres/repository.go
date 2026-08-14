// Package postgres is the Social Play context's Postgres adapter. It
// implements port.GameRepository and port.RegistrationRepository against
// sqlc-generated queries and translates Postgres errors into domain errors
// (CLAUDE.md rule 5). It only compiles after `make generate` has produced
// internal/gen/socialplaydb (see CLAUDE.md gotchas); that package is
// gitignored and not committed.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	socialplaydb "github.com/nhuthuynh/white-label/internal/gen/socialplaydb"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// Postgres error codes this adapter cares about. 23505 is the unique
// violation on registrations_active_player_per_game_idx (db/migrations/
// 0005_socialplay.sql) firing — the DB-level mirror of
// domain.ErrAlreadyRegistered per CLAUDE.md rule 4. P0001 is the
// registrations_capacity_guard trigger's RAISE EXCEPTION (db/migrations/
// 0006_socialplay_capacity_guard.sql, rewritten by db/migrations/
// 0012_socialplay_guest_capacity.sql T8.7 to a weighted sum) firing — the
// DB-level mirror of domain.ErrGameFull, closing the race domain.Register's
// in-process count check alone can't (PR #14 loop-1 review finding).
const (
	pgUniqueViolation  = "23505"
	pgCapacityExceeded = "P0001"
	// pgForeignKeyViolation (T10.4) is matches.game_id's REFERENCES games (id)
	// constraint firing — see translateMatchErr's doc comment for when this
	// is actually reachable in practice.
	pgForeignKeyViolation = "23503"
)

// GameRepository implements port.GameRepository.
type GameRepository struct {
	q *socialplaydb.Queries
}

func NewGameRepository(pool *pgxpool.Pool) *GameRepository {
	return &GameRepository{q: socialplaydb.New(pool)}
}

func (r *GameRepository) Create(ctx context.Context, g domain.Game) (domain.Game, error) {
	row, err := r.q.CreateGame(ctx, socialplaydb.CreateGameParams{
		ID:              mustUUID(g.ID),
		HostID:          g.HostID,
		FacilityID:      g.FacilityID,
		VenueFacilityID: nullableUUID(g.VenueFacilityID),
		CourtIds:        mustUUIDs(g.CourtIDs),
		StartsAt:        toTimestamptz(g.Range.Start),
		EndsAt:          toTimestamptz(g.Range.End),
		Capacity:        int32(g.Capacity),
		Status:          string(g.Status),
		PaymentMethod:   string(g.PaymentMethod),
		GuestAllowance:  int32(g.GuestAllowance),
		// EntryFee (T9.2) is always written explicitly, including a 0 for a
		// free Game — the column DEFAULT exists only to backfill rows that
		// predate the columns (db/migrations/0013_socialplay_entry_fee.sql).
		EntryFeeCents:    g.EntryFee.Cents,
		EntryFeeCurrency: g.EntryFee.Currency,
	})
	if err != nil {
		return domain.Game{}, translateGameErr(err)
	}
	return gameFromFields(row.ID, row.HostID, row.FacilityID, row.VenueFacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status, row.PaymentMethod, row.GuestAllowance, row.EntryFeeCents, row.EntryFeeCurrency), nil
}

func (r *GameRepository) GetByID(ctx context.Context, id string) (domain.Game, error) {
	row, err := r.q.GetGameByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Game{}, translateGameErr(err)
	}
	return gameFromFields(row.ID, row.HostID, row.FacilityID, row.VenueFacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status, row.PaymentMethod, row.GuestAllowance, row.EntryFeeCents, row.EntryFeeCurrency), nil
}

// UpdateStatus implements port.GameRepository.UpdateStatus (T12.4): the
// single-column status write app.Service.CancelGame needs, via its own
// dedicated query (db/queries/socialplay.sql's UpdateGameStatus) which
// cannot touch any other column. An unknown id produces pgx.ErrNoRows,
// which translateGameErr already maps to domain.ErrGameNotFound (CLAUDE.md
// rule 5 — upper layers only ever see domain errors).
func (r *GameRepository) UpdateStatus(ctx context.Context, id string, status domain.Status) (domain.Game, error) {
	row, err := r.q.UpdateGameStatus(ctx, socialplaydb.UpdateGameStatusParams{
		ID:     mustUUID(id),
		Status: string(status),
	})
	if err != nil {
		return domain.Game{}, translateGameErr(err)
	}
	return gameFromFields(row.ID, row.HostID, row.FacilityID, row.VenueFacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status, row.PaymentMethod, row.GuestAllowance, row.EntryFeeCents, row.EntryFeeCurrency), nil
}

// ListGames implements port.GameRepository.ListGames (T8.9): the Discover &
// Join Games browse/filter read. venue_facility_id/starts_after/
// starts_before are all optional (nullableUUID/nullableTimestamptz below
// convert an unset domain filter value into the explicitly-invalid,
// Valid: false zero value the sqlc.narg-generated params expect for "no
// filter on this dimension" — see socialplaydb.sql's ListGames query). The
// query itself computes SpotsLeft, so this adapter has nothing left to do
// beyond mapping columns onto port.GameListing.
func (r *GameRepository) ListGames(ctx context.Context, filter port.GameListingFilter) ([]port.GameListing, error) {
	rows, err := r.q.ListGames(ctx, socialplaydb.ListGamesParams{
		VenueFacilityID: nullableUUID(filter.VenueFacilityID),
		StartsAfter:     nullableTimestamptz(filter.StartsAfter),
		StartsBefore:    nullableTimestamptz(filter.StartsBefore),
	})
	if err != nil {
		return nil, translateGameErr(err)
	}
	out := make([]port.GameListing, 0, len(rows))
	for _, row := range rows {
		out = append(out, port.GameListing{
			Game:      gameFromFields(row.ID, row.HostID, row.FacilityID, row.VenueFacilityID, row.CourtIds, row.StartsAt, row.EndsAt, row.Capacity, row.Status, row.PaymentMethod, row.GuestAllowance, row.EntryFeeCents, row.EntryFeeCurrency),
			SpotsLeft: int(row.SpotsLeft),
		})
	}
	return out, nil
}

func translateGameErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrGameNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (games): %w", err)
}

// gameFromFields builds a domain.Game from the 13 columns every games query
// selects (venue_facility_id, T8.3; payment_method/guest_allowance, T8.7;
// entry_fee_cents/entry_fee_currency, T9.2).
// sqlc generates a distinct Row struct per query (CreateGameRow,
// GetGameByIDRow, ...) rather than reusing socialplaydb.Game, mirroring the
// fromFields pattern in internal/booking/adapter/postgres/repository.go
// (CLAUDE.md gotcha: sqlc emits a distinct ...Row type per query).
func gameFromFields(id pgtype.UUID, hostID, facilityID string, venueFacilityID pgtype.UUID, courtIDs []pgtype.UUID, startsAt, endsAt pgtype.Timestamptz, capacity int32, status, paymentMethod string, guestAllowance int32, entryFeeCents int64, entryFeeCurrency string) domain.Game {
	ids := make([]string, 0, len(courtIDs))
	for _, c := range courtIDs {
		ids = append(ids, c.String())
	}
	return domain.Game{
		ID:              id.String(),
		HostID:          hostID,
		FacilityID:      facilityID,
		VenueFacilityID: fromNullableUUID(venueFacilityID),
		CourtIDs:        ids,
		Range: domain.TimeRange{
			Start: startsAt.Time,
			End:   endsAt.Time,
		},
		Capacity:       int(capacity),
		Status:         domain.Status(status),
		PaymentMethod:  domain.PaymentMethod(paymentMethod),
		GuestAllowance: int(guestAllowance),
		EntryFee: domain.Money{
			Cents:    entryFeeCents,
			Currency: entryFeeCurrency,
		},
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
		GuestCount:    int32(reg.GuestCount),
	})
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus, row.GuestCount), nil
}

func (r *RegistrationRepository) GetByID(ctx context.Context, id string) (domain.Registration, error) {
	row, err := r.q.GetRegistrationByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus, row.GuestCount), nil
}

func (r *RegistrationRepository) ListActiveForGame(ctx context.Context, gameID string) ([]domain.Registration, error) {
	rows, err := r.q.ListActiveRegistrationsForGame(ctx, mustUUID(gameID))
	if err != nil {
		return nil, translateRegistrationErr(err)
	}
	out := make([]domain.Registration, 0, len(rows))
	for _, row := range rows {
		out = append(out, registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus, row.GuestCount))
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
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus, row.GuestCount), nil
}

// UpdatePaymentStatus persists a PaymentStatus change (T6.5) via its own
// dedicated query — UpdateRegistrationStatus above only ever touches the
// status column, so it cannot be reused here without silently leaving
// PaymentStatus unpersisted (the exact gap this method exists to close).
func (r *RegistrationRepository) UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus) (domain.Registration, error) {
	row, err := r.q.UpdateRegistrationPaymentStatus(ctx, socialplaydb.UpdateRegistrationPaymentStatusParams{
		ID:            mustUUID(id),
		PaymentStatus: string(status),
	})
	if err != nil {
		return domain.Registration{}, translateRegistrationErr(err)
	}
	return registrationFromFields(row.ID, row.GameID, row.PlayerID, row.Source, row.Status, row.PaymentStatus, row.GuestCount), nil
}

// translateRegistrationErr maps infrastructure failures onto domain errors
// — the only errors allowed to cross out of this package (CLAUDE.md rule
// 5). A 23505 unique_violation on registrations_active_player_per_game_idx
// becomes domain.ErrAlreadyRegistered, the DB-level guard's counterpart to
// domain.Register's own pre-check (CLAUDE.md rule 4). A P0001 from the
// registrations_capacity_guard trigger (db/migrations/
// 0006_socialplay_capacity_guard.sql) becomes domain.ErrGameFull — the
// same sentinel domain.Register's own pre-check returns, so a caller sees
// one error type regardless of which layer caught the capacity conflict
// (mirrors translateErr's ErrCourtDoubleBooked handling in
// internal/booking/adapter/postgres/repository.go).
func translateRegistrationErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return domain.ErrAlreadyRegistered
		case pgCapacityExceeded:
			return domain.ErrGameFull
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRegistrationNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (registrations): %w", err)
}

// registrationFromFields builds a domain.Registration from the 7 columns
// every registrations query selects (6 pre-T8.7 + guest_count, T8.7) — see
// gameFromFields's doc comment for why this pattern exists.
func registrationFromFields(id, gameID pgtype.UUID, playerID, source, status, paymentStatus string, guestCount int32) domain.Registration {
	return domain.Registration{
		ID:            id.String(),
		GameID:        gameID.String(),
		PlayerID:      playerID,
		Source:        domain.RegistrationSource(source),
		Status:        domain.RegistrationStatus(status),
		PaymentStatus: domain.PaymentStatus(paymentStatus),
		GuestCount:    int(guestCount),
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

// nullableUUID converts an optional domain string (T8.3's
// domain.Game.VenueFacilityID: "" means "no venue Facility set") into a
// pgtype.UUID suitable for the nullable venue_facility_id column (db/
// migrations/0011_socialplay_facility_fk.sql) — an explicitly-invalid zero
// value (Valid: false), never an attempt to parse "" as a uuid via mustUUID
// (which would panic).
func nullableUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	return mustUUID(s)
}

// fromNullableUUID is nullableUUID's inverse: reads a possibly-NULL
// venue_facility_id column back into domain.Game.VenueFacilityID, mapping
// SQL NULL (u.Valid == false) to Go's "" rather than a formatted
// all-zeroes uuid string.
func fromNullableUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// nullableTimestamptz converts an optional filter bound (T8.9's
// GameListingFilter.StartsAfter/StartsBefore: time.Time's zero value means
// "no bound on this side") into a pgtype.Timestamptz suitable for the
// ListGames query's sqlc.narg params — an explicitly-invalid zero value
// (Valid: false), mirroring nullableUUID's identical convention for
// VenueFacilityID.
func nullableTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return toTimestamptz(t)
}

// MatchRepository implements port.MatchRepository (T10.4).
type MatchRepository struct {
	q *socialplaydb.Queries
}

func NewMatchRepository(pool *pgxpool.Pool) *MatchRepository {
	return &MatchRepository{q: socialplaydb.New(pool)}
}

func (r *MatchRepository) Create(ctx context.Context, m domain.Match) (domain.Match, error) {
	score, err := marshalScore(m.Score)
	if err != nil {
		return domain.Match{}, fmt.Errorf("socialplay postgres adapter (matches): marshalling score for match %s: %w", m.ID, err)
	}

	row, err := r.q.CreateMatch(ctx, socialplaydb.CreateMatchParams{
		ID:         mustUUID(m.ID),
		GameID:     mustUUID(m.GameID),
		Players:    m.Players,
		Score:      score,
		RecordedAt: toTimestamptz(m.RecordedAt),
	})
	if err != nil {
		return domain.Match{}, translateMatchErr(err)
	}
	return matchFromFields(row.ID, row.GameID, row.Players, row.Score, row.RecordedAt)
}

func (r *MatchRepository) ListForGame(ctx context.Context, gameID string) ([]domain.Match, error) {
	rows, err := r.q.ListMatchesForGame(ctx, mustUUID(gameID))
	if err != nil {
		return nil, translateMatchErr(err)
	}
	out := make([]domain.Match, 0, len(rows))
	for _, row := range rows {
		m, err := matchFromFields(row.ID, row.GameID, row.Players, row.Score, row.RecordedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// translateMatchErr maps infrastructure failures onto domain errors
// (CLAUDE.md rule 5) — mirrors translateGameErr/translateRegistrationErr's
// shape. There is no unique-violation or capacity-guard case here (see
// db/migrations/0015_socialplay_matches.sql's doc comment on why matches
// carries no such invariant); a foreign-key violation on game_id (23503)
// would mean app.Service.RecordMatchResult's own prior GetByID existence
// check somehow missed a concurrently-deleted Game — this codebase never
// deletes Games (only cancels them), so this is defensive, not a case any
// current code path is known to hit.
func translateMatchErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return domain.ErrGameNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrGameNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (matches): %w", err)
}

// matchFromFields builds a domain.Match from the 5 columns every matches
// query selects — mirrors gameFromFields/registrationFromFields' pattern
// (CLAUDE.md gotcha: sqlc emits a distinct ...Row type per query, not a
// shared table model). Unlike those, this can fail: unmarshalling the
// jsonb score column back into map[string]int is fallible in principle
// (a hand-edited row with malformed JSON), so this returns an error rather
// than panicking — a corrupt row is a data problem to surface, not a
// programmer error like mustUUID's malformed-ID case.
func matchFromFields(id, gameID pgtype.UUID, players []string, score []byte, recordedAt pgtype.Timestamptz) (domain.Match, error) {
	m, err := unmarshalScore(score)
	if err != nil {
		return domain.Match{}, fmt.Errorf("socialplay postgres adapter (matches): unmarshalling score for match %s: %w", id.String(), err)
	}
	return domain.Match{
		ID:         id.String(),
		GameID:     gameID.String(),
		Players:    players,
		Score:      m,
		RecordedAt: recordedAt.Time,
	}, nil
}

// marshalScore/unmarshalScore convert domain.Match.Score (map[string]int)
// to/from the []byte shape sqlc's pgx/v5 driver uses for a jsonb column —
// see CreateMatchParams.Score's doc comment (socialplaydb, generated) for
// why this codebase's first jsonb column needs an explicit marshal step
// unlike every other column here.
func marshalScore(score map[string]int) ([]byte, error) {
	return json.Marshal(score)
}

func unmarshalScore(b []byte) (map[string]int, error) {
	var score map[string]int
	if err := json.Unmarshal(b, &score); err != nil {
		return nil, err
	}
	return score, nil
}
