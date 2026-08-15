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

	// UpdateStatus persists a Game status transition (T12.4's
	// app.Service.CancelGame) via its own single-column write, following the
	// one-query-per-updatable-field convention UpdateRegistrationStatus and
	// competitions' UpdateCompetitionStatus already established — so this
	// path cannot accidentally clobber a Game's capacity, courts, or entry
	// fee. Returns domain.ErrGameNotFound for an unknown id.
	//
	// This is the Game aggregate's first write path other than Create:
	// before T12.4 a Game could be scheduled but never modified, which is
	// exactly why there was no CancelGame RPC to authorize.
	UpdateStatus(ctx context.Context, id string, status domain.Status) (domain.Game, error)
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

	// CancelAllActiveForGame bulk-cancels every non-cancelled Registration
	// scoped to gameID in ONE atomic statement (T16.3, partial fix for
	// #124) — app.Service.CancelGame's cascade, fired after the Game's own
	// status write persists so a cancelled Game never leaves an active
	// Registration behind. Returns the number of rows actually
	// transitioned (an already-cancelled Registration is not re-counted),
	// not merely "did this succeed" — the caller has no other way to
	// learn how many Registrations a Host's single cancel action affected.
	//
	// Deliberately NOT N sequential calls to Update/UpdatePaymentStatus:
	// this is a single-query bulk write, mirroring the
	// one-query-per-write-path convention UpdateRegistrationStatus and
	// waitlist.go's PromoteNext already establish, following
	// db/queries/socialplay.sql's CancelAllActiveRegistrationsForGame
	// (an `UPDATE ... WHERE game_id = $1 AND status <> 'cancelled'`). A
	// loop of individual updates would cost N round trips for one Host
	// action and — the sharper reason — could not close the same race
	// this one statement closes: a Registration created concurrently,
	// between the Game's status write and this call running, is still
	// caught by the same WHERE clause and cancelled in the same
	// statement, rather than possibly landing after a per-row loop had
	// already passed it by.
	//
	// Scope, matching CancelGame's own instructions (T16.3): this method
	// touches ONLY the status column. It does not call RefundPayment or
	// touch PaymentStatus (instruction 4 — refunding a cancelled
	// Registration's Payment is an explicitly open sub-question, not
	// this ticket's to answer), and it does not touch WaitlistEntry rows
	// (instruction 5 — a cancelled Game must not promote anyone off its
	// waitlist; app.Service.CancelGame simply never calls
	// promoteNextWaiting, so there is nothing for this method to
	// suppress).
	CancelAllActiveForGame(ctx context.Context, gameID string) (int, error)
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
