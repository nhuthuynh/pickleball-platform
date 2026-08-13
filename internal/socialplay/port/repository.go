package port

import (
	"context"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameRepository is Social Play's persistence boundary for the Game
// aggregate (T5.1). The domain and app layers only ever see this interface;
// internal/socialplay/adapter/postgres implements it against the real
// database (T5.4), and tests implement it in-memory. Mirrors
// internal/booking/port.Repository's shape.
type GameRepository interface {
	// Create persists a new scheduled Game.
	Create(ctx context.Context, g domain.Game) (domain.Game, error)

	// GetByID returns a single Game, or domain.ErrGameNotFound.
	GetByID(ctx context.Context, id string) (domain.Game, error)

	// ListGames returns scheduled Games matching filter, each paired with
	// its SpotsLeft (T8.9) — the Discover & Join Games browse/filter read
	// path. See GameListingFilter/GameListing's doc comments for the exact
	// filter/shape semantics.
	ListGames(ctx context.Context, filter GameListingFilter) ([]GameListing, error)
}

// GameListingFilter is ListGames' optional filter (T8.9). VenueFacilityID
// empty means no facility filter; StartsAfter/StartsBefore zero (time.Time's
// zero value, i.e. IsZero()) means no bound on that side — mirrors
// ListFacilities' name_filter empty-string-means-unfiltered convention
// (internal/facilities/app/service.go), using time.Time's zero value as the
// analogous "unset" sentinel for the two timestamp bounds since an empty
// string isn't a meaningful sentinel for a timestamp.
type GameListingFilter struct {
	VenueFacilityID string
	StartsAfter     time.Time
	StartsBefore    time.Time
}

// GameListing pairs a Game with SpotsLeft — capacity minus the weighted sum
// of (1 + guest_count) across that Game's active (non-cancelled)
// registrations, computed by the repository/query itself (T8.9) rather than
// requiring the app layer to separately fetch and sum registrations per
// Game (an N+1 query per Game in the list). Deliberately not a field on
// domain.Game itself: SpotsLeft is a derived read-model value for this one
// browse/list path, not part of the Game aggregate's own invariants
// (CLAUDE.md rule 2 — keep the domain pure/free of read-path-specific
// projections). Mirrors the identically-named proto message
// (proto/pickleball/socialplay/v1/socialplay.proto) field-for-field.
type GameListing struct {
	Game      domain.Game
	SpotsLeft int
}

// RegistrationRepository is Social Play's persistence boundary for the
// Registration aggregate (T5.2).
type RegistrationRepository interface {
	// Create persists a new registration. Implementations backed by
	// Postgres rely on a unique constraint on (game_id, player_id) scoped
	// to non-cancelled rows as the authoritative guard against double
	// registration under concurrency; domain.Register's own-scope check is
	// a pre-check only, the same relationship booking.EnsureNoConflict has
	// with the EXCLUDE constraint (CLAUDE.md rule 4).
	Create(ctx context.Context, r domain.Registration) (domain.Registration, error)

	// GetByID returns a single registration, or domain.ErrRegistrationNotFound.
	GetByID(ctx context.Context, id string) (domain.Registration, error)

	// ListActiveForGame returns the non-cancelled registrations for gameID —
	// the capacity-safe read path domain.Register needs to re-derive the
	// active count/players before enforcing the capacity invariant.
	ListActiveForGame(ctx context.Context, gameID string) ([]domain.Registration, error)

	// Update persists changes to an existing registration (e.g. a status
	// transition from Cancel).
	Update(ctx context.Context, r domain.Registration) (domain.Registration, error)

	// UpdatePaymentStatus persists a PaymentStatus change for the
	// Registration identified by id (T6.5) — a dedicated, single-purpose
	// write path (mirroring UpdateRegistrationStatus/UpdateBookingStatus's
	// one-method-per-updatable-field convention) rather than overloading
	// Update, so the Postgres adapter's query for it is scoped to exactly
	// the payment_status column and can't accidentally clobber Status (or
	// vice versa). Returns domain.ErrRegistrationNotFound for an unknown id.
	UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus) (domain.Registration, error)
}

// MatchRepository is Social Play's persistence boundary for the Match
// aggregate (T10.4). Mirrors GameRepository/RegistrationRepository's shape:
// the domain and app layers only ever see this interface;
// internal/socialplay/adapter/postgres implements it against the real
// database, and tests implement it in-memory.
type MatchRepository interface {
	// Create persists a new Match. Callers (app.Service.RecordMatchResult)
	// have already assigned m.ID (mirrors RegistrationRepository.Create /
	// WaitlistRepository.Create's identical relationship with
	// Register/JoinWaitlist — see those constructors' doc comments for why
	// the ID is assigned at the app layer, not by this method or by
	// domain.RecordMatch itself).
	Create(ctx context.Context, m domain.Match) (domain.Match, error)

	// ListForGame returns every Match recorded against gameID, ordered by
	// RecordedAt ascending (see db/queries/socialplay.sql's
	// ListMatchesForGame). An unknown gameID returns an empty slice, not an
	// error, at THIS layer — unlike ListRegistrationsForGame's identical
	// "empty roster for an unknown game" read, T10.4's own error-handling
	// instructions require ListMatchesForGame's RPC to answer an unknown
	// GameID with NotFound, so app.Service.ListMatchesForGame does its own
	// GameRepository.GetByID existence check before calling this method —
	// see that method's doc comment.
	ListForGame(ctx context.Context, gameID string) ([]domain.Match, error)
}
