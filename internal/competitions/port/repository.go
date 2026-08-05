package port

import (
	"context"
	"time"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// CompetitionListingFilter is ListCompetitions' optional filter (T9.4).
// Every field is optional and an unset field means "no filter on this
// dimension", mirroring socialplay's GameListingFilter convention exactly:
// an empty VenueFacilityID and a zero time.Time are the "unset" spellings,
// which the Postgres adapter converts into the explicitly-invalid
// (Valid: false) values sqlc's narg-generated params expect.
//
// StartsAfter/StartsBefore filter on a Competition's EARLIEST session start,
// not on "any session falls in the window". A Competition runs across dates
// (domain.Session), so "starts after" has to mean something specific, and
// the first sitting is the date a player thinks of as when the Competition
// starts — it is also the value ListCompetitions orders by, so the filter
// and the ordering agree on one definition rather than two.
type CompetitionListingFilter struct {
	VenueFacilityID string
	StartsAfter     time.Time
	StartsBefore    time.Time
}

// CompetitionListing pairs a Competition with SpotsLeft — its capacity minus
// the **weighted** sum of (1 + GuestCount) across its active (non-cancelled)
// entries, the identical weighting domain.Enter's own capacity check applies
// (see internal/competitions/domain/entry.go's countActiveEntries).
//
// The weighting is not optional and not cosmetic: an unweighted "spots left"
// would visibly lie the moment any entrant brings a guest — a Competition
// with capacity 8 holding one entry with 3 guests has 4 places left, not 7.
// Computing it server-side (in SQL, as one aggregate per row) rather than
// making each client re-derive it is what keeps every client agreeing with
// the invariant the database actually enforces.
//
// A sibling wrapper rather than a field on domain.Competition, because
// SpotsLeft is a derived read-model value for this one browse path, not part
// of the aggregate — the same reasoning socialplay's port.GameListing
// documents.
type CompetitionListing struct {
	Competition domain.Competition
	SpotsLeft   int
}

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

	// ListCompetitions is T9.4's browse/list read path, returning each
	// matching Competition alongside its server-computed SpotsLeft (see
	// CompetitionListing). Only scheduled Competitions are ever returned: a
	// cancelled Competition can't be entered (domain.Enter rejects it), so
	// it has no place in a player-facing browse list — the same rule
	// socialplay's ListGames applies to a cancelled Game.
	ListCompetitions(ctx context.Context, filter CompetitionListingFilter) ([]CompetitionListing, error)

	// ListEntriesForCompetition returns EVERY entry for competitionID,
	// cancelled ones included — deliberately distinct from
	// ListActiveEntriesForCompetition above, which exists to feed
	// domain.Enter's capacity rule and must therefore exclude them.
	//
	// This one backs T9.4's Host-facing roster read
	// (ListEntriesForCompetition RPC), where a cancelled entry is real
	// information a Host needs to see rather than noise to hide: "who
	// withdrew" and "who never entered" are different answers, and a roster
	// that silently drops the former can't be reconciled against payments.
	// Two methods rather than one with a boolean, so neither caller can
	// accidentally get the other's semantics.
	ListEntriesForCompetition(ctx context.Context, competitionID string) ([]domain.CompetitionEntry, error)
}
