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
// up-front: T9.4 added the listing read path and T9.5 adds
// GetByShareToken, the token-addressed lookup for the share link.
// Declaring those before their use cases existed would have shipped
// interface methods no implementation was exercised against and no test
// covered — the same reasoning that kept T9.1's errors.go from
// pre-declaring later sentinels.
type Repository interface {
	// Create persists a new scheduled Competition, including its sessions.
	Create(ctx context.Context, c domain.Competition) (domain.Competition, error)

	// GetByID returns a single Competition, or domain.ErrCompetitionNotFound.
	GetByID(ctx context.Context, id string) (domain.Competition, error)

	// GetByShareToken returns the Competition whose share_token equals
	// shareToken — the read behind T9.5's shareable registration link — or
	// domain.ErrCompetitionNotFound.
	//
	// Two rules bind every implementation of this method:
	//
	//  1. It returns THE SAME domain.ErrCompetitionNotFound sentinel that
	//     GetByID returns, unwrapped and with no token-specific detail. This
	//     read is unauthenticated and keyed by a secret, so any difference
	//     between "no such token" and "no such Competition" is an oracle an
	//     enumerator can use to confirm a guess without ever seeing a
	//     Competition. An implementation must not, for example, validate the
	//     token's shape and return a distinct "malformed" error.
	//  2. It does NOT filter by status. A cancelled Competition's token must
	//     still resolve, returning the Competition with Status cancelled —
	//     deliberately unlike ListCompetitions, which hides cancelled
	//     Competitions from the browse list. A link already published to a
	//     social channel outlives the Competition's scheduled state, and a
	//     Player who follows it deserves "this competition was cancelled"
	//     rather than a 404 that reads as a broken link. Entering it is
	//     separately rejected by domain.Enter.
	GetByShareToken(ctx context.Context, shareToken string) (domain.Competition, error)

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

	// GetEntryByID returns a single CompetitionEntry, or
	// domain.ErrCompetitionEntryNotFound (T10.6). Added alongside
	// UpdateEntryPaymentStatus for app.Service.MarkCompetitionEntryPaymentStatus,
	// which needs to load an entry before validating and writing its
	// PaymentStatus transition — mirrors
	// internal/socialplay/port.RegistrationRepository.GetByID's role for
	// Service.MarkRegistrationPaymentStatus exactly.
	GetEntryByID(ctx context.Context, id string) (domain.CompetitionEntry, error)

	// UpdateEntryPaymentStatus persists a PaymentStatus change for the
	// CompetitionEntry identified by id (T10.6) — a dedicated,
	// single-purpose write path (mirroring
	// RegistrationRepository.UpdatePaymentStatus's one-method-per-
	// updatable-field convention) rather than overloading a general Update,
	// so the Postgres adapter's query for it is scoped to exactly the
	// payment_status column and can't accidentally clobber Status (or vice
	// versa). Returns domain.ErrCompetitionEntryNotFound for an unknown id.
	UpdateEntryPaymentStatus(ctx context.Context, id string, status domain.PaymentStatus) (domain.CompetitionEntry, error)

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

	// CancelAllActiveForCompetition bulk-cancels every non-cancelled
	// CompetitionEntry scoped to competitionID in ONE atomic statement
	// (T16.3, closes the mirrored Competitions gap found this ceremony —
	// see socialplay.port.RegistrationRepository.CancelAllActiveForGame,
	// which this mirrors field-for-field apart from the owning
	// aggregate). app.Service.CancelCompetition's cascade, fired after the
	// Competition's own status write persists so a cancelled Competition
	// never leaves an active entry behind. Returns the number of rows
	// actually transitioned (an already-cancelled entry is not
	// re-counted).
	//
	// Deliberately NOT N sequential UpdateEntryPaymentStatus-shaped calls:
	// a single-query bulk write, following
	// db/queries/competitions.sql's CancelAllActiveEntriesForCompetition
	// (an `UPDATE ... WHERE competition_id = $1 AND status <>
	// 'cancelled'`) — one round trip per Host action instead of N, and the
	// set-based UPDATE closes the same concurrent-entry race
	// CancelAllActiveForGame's doc comment describes: an entry that
	// CreateEntry lands between the Competition's status write and this
	// call is still caught by the same WHERE clause.
	//
	// Scope, matching CancelCompetition's own instructions (T16.3): ONLY
	// the status column. It does not call RefundPayment or touch
	// PaymentStatus (instruction 4), and Competitions has no waitlist to
	// consider (instruction 5 scopes that half to Social Play only).
	CancelAllActiveForCompetition(ctx context.Context, competitionID string) (int, error)
}
