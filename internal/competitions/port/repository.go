package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// Repository is Competitions' persistence boundary. The domain and app
// layers only ever see this interface; internal/competitions/adapter/
// postgres will implement it against the real database (T9.4), and tests
// implement it in-memory.
//
// One interface covers both Competitions and their entries, where Social
// Play uses two (GameRepository, RegistrationRepository). The reason is
// that a CompetitionEntry is never read or written on its own here: every
// use case that touches entries is scoped to one owning Competition —
// EnterCompetition loads the Competition *and* its active entries together
// because domain.Enter's weighted capacity rule needs both, and T9.4's
// ListEntriesForCompetition is likewise per-Competition. Splitting them
// would produce two interfaces that are always constructed, wired, and
// called as a pair, which buys no isolation and costs a second type in
// every call site.
//
// Methods are added as the use cases that need them land, rather than
// up-front: T9.4 adds the listing read path and T9.5 adds a
// token-addressed lookup for the share link. Declaring those here now
// would ship interface methods no implementation is exercised against and
// no test covers — the same reasoning that kept T9.1's errors.go from
// pre-declaring this ticket's sentinels.
type Repository interface {
	// Create persists a new scheduled Competition, including its sessions.
	Create(ctx context.Context, c domain.Competition) (domain.Competition, error)

	// GetByID returns a single Competition, or domain.ErrCompetitionNotFound.
	GetByID(ctx context.Context, id string) (domain.Competition, error)

	// UpdateStatus persists a status transition for the Competition
	// identified by id (app.Service.CancelCompetition). A dedicated,
	// single-purpose write path rather than a general Update, following the
	// same one-method-per-updatable-field convention as Social Play's
	// UpdatePaymentStatus and Booking's UpdateBookingStatus — the Postgres
	// query behind it can then be scoped to exactly the status column and
	// cannot accidentally clobber a Competition's sessions or capacity.
	// Returns domain.ErrCompetitionNotFound for an unknown id.
	UpdateStatus(ctx context.Context, id string, status domain.Status) (domain.Competition, error)

	// CreateEntry persists a new CompetitionEntry. Implementations backed
	// by Postgres rely on a DB-level weighted capacity guard as the
	// authoritative protection against two simultaneous entries overfilling
	// a Competition (T9.4); domain.Enter's own check is a pre-check only,
	// the same relationship booking.EnsureNoConflict has with the EXCLUDE
	// constraint (CLAUDE.md rule 4).
	CreateEntry(ctx context.Context, e domain.CompetitionEntry) (domain.CompetitionEntry, error)

	// ListActiveEntriesForCompetition returns the non-cancelled entries for
	// competitionID — the read path domain.Enter needs to re-derive the
	// weighted occupied-places count and the already-entered players before
	// enforcing capacity.
	ListActiveEntriesForCompetition(ctx context.Context, competitionID string) ([]domain.CompetitionEntry, error)
}
