package domain

// EntrySource identifies how a CompetitionEntry came to exist.
//
// These are the **same two values** Social Play's RegistrationSource
// already uses — `app` and `social` — not a new vocabulary such as
// `shared_link`. That is CLAUDE.md rule 7 (one ubiquitous language) and the
// BA/PE ruling recorded in docs/process/t9-sprint-plan.md §A3: inventing a
// third word for the same fact is exactly the "two different terms for what
// should be the same concept" failure the proto-review checklist exists to
// catch.
//
//   - EntrySourceApp: the entrant found the Competition inside the platform
//     and entered directly.
//   - EntrySourceSocial: the entrant arrived via a shareable registration
//     link the Host published to a channel outside the app (T9.5). Unlike
//     Social Play's identically-named value, which has had no producer
//     since T5.2, this one gets a real producer in this sprint — T9.5's
//     share-link flow sends it and the server validates it against this
//     closed enum rather than inferring it.
type EntrySource string

const (
	EntrySourceApp    EntrySource = "app"
	EntrySourceSocial EntrySource = "social"
)

// IsValid reports whether s is one of EntrySource's closed enum values.
// Enter uses this so a client-supplied source can never write an
// unrecognised value to the field.
func (s EntrySource) IsValid() bool {
	switch s {
	case EntrySourceApp, EntrySourceSocial:
		return true
	default:
		return false
	}
}

// EntryStatus is a CompetitionEntry's lifecycle state. Named distinctly
// from Competition's Status (both live in this package) rather than reusing
// that type.
//
// EntryStatusCancelled has no transition method in T9.1 — no T9 ticket
// exposes an "withdraw from a competition" RPC (T9.4's RPC list is
// Create/Get/List/Enter/CancelCompetition/ListEntries), so a Cancel method
// here would be a domain method with no caller. It is modelled now because
// the weighted capacity rule must already know to exclude cancelled entries
// (see countActiveEntries): the counting rule is correct from the day the
// first cancelled entry exists, rather than being a second thing to
// remember when withdrawal is built. Adding the transition later is a
// method plus its tests, with no change to the invariant.
type EntryStatus string

const (
	EntryStatusEntered   EntryStatus = "entered"
	EntryStatusCancelled EntryStatus = "cancelled"
)

// PaymentStatus tracks whether an entry's place has been paid for,
// mirroring internal/payments/domain.Status's unpaid -> paid -> refunded
// machine exactly (and socialplay.PaymentStatus's projection of it, T6.5) —
// "never paid" and "paid, then refunded" are different facts a Host or
// support view must be able to tell apart.
//
// T9 ships this as modelling only: Enter sets the initial "unpaid" and
// nothing in this sprint writes it again, because wiring Competitions to
// the Payments context is not a T9 ticket. That is the same path Social
// Play took (modelled in T5.2, given its writer in T6.5) and is stated here
// so the missing writer reads as sequencing rather than an oversight. When
// that integration lands, Payments must be its single writer, via a port —
// not a field any client can set.
type PaymentStatus string

const (
	PaymentStatusUnpaid   PaymentStatus = "unpaid"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// IsValid reports whether s is one of PaymentStatus's closed enum values.
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusUnpaid, PaymentStatusPaid, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

// CompetitionEntry is a Player's claim on a Competition's places — one for
// themself plus one per guest they bring. The Competitions analogue of
// socialplay.Registration.
type CompetitionEntry struct {
	ID            string
	CompetitionID string
	PlayerID      string
	// GuestCount is how many guests this entrant brings; 0 means none.
	// Each guest occupies a place exactly as the entrant does — see
	// Enter's weighted capacity rule. Must be >= 0 and <= the owning
	// Competition's GuestAllowance.
	GuestCount    int
	Source        EntrySource
	PaymentStatus PaymentStatus
	Status        EntryStatus
}

// Enter builds a new CompetitionEntry for playerID against competition,
// enforcing every invariant that needs only the Competition and its
// existing entries (never infrastructure):
//
//   - playerID must be non-empty, and source must be one of EntrySource's
//     closed enum values — the server validates the caller's claimed
//     source, it does not infer it (T9.5);
//   - the Competition must still be scheduled: entering a cancelled
//     Competition returns ErrCompetitionCancelled. (This is the one place
//     Enter deliberately does more than socialplay.Register, which has no
//     equivalent check: T9.5 requires a cancelled Competition's share token
//     to still resolve for reading while entry is rejected, and the only
//     place that rejection can live without being re-implemented per
//     transport is here.)
//   - guest allowance: guestCount must be >= 0 and <= the Competition's
//     GuestAllowance (ErrGuestAllowanceExceeded). Checked before capacity,
//     so a caller who asked for more guests than the Host permits gets that
//     specific, actionable error rather than a generic "full" — even on a
//     Competition that is also out of room;
//   - no double entry: a player with an existing active entry returns
//     ErrAlreadyEntered. Checked before capacity (same precedence
//     socialplay.Register uses), since a player who is already in must
//     never be told the competition is full because of their own entry;
//   - capacity, **weighted**: an entry and its guests all occupy places, so
//     this sums (1 + e.GuestCount) across existing active entries, adds
//     this call's own weight (1 + guestCount), and rejects with
//     ErrCompetitionFull if the total would exceed competition.Capacity.
//
// The weighting is here from day one, not retrofitted: T8.6 had to go back
// and reweight Social Play's identical rule after guests were introduced
// against a plain headcount, and T5's retro finding 1 says to bake the
// weighted rule into the ticket that introduces the invariant. A single
// entry bringing enough guests fills a Competition on its own, long before
// the entry *count* reaches Capacity — a headcount check would let it
// oversell.
//
// The returned entry's ID is intentionally left empty: Enter is a pure
// domain constructor and assigning a durable ID is the app/adapter layer's
// job at persistence time (mirrors socialplay.Register).
func Enter(competition Competition, existing []CompetitionEntry, playerID string, guestCount int, source EntrySource) (CompetitionEntry, error) {
	if playerID == "" {
		return CompetitionEntry{}, ErrEmptyPlayerID
	}
	if !source.IsValid() {
		return CompetitionEntry{}, ErrInvalidEntrySource
	}
	if competition.Status != StatusScheduled {
		return CompetitionEntry{}, ErrCompetitionCancelled
	}
	if guestCount < 0 || guestCount > competition.GuestAllowance {
		return CompetitionEntry{}, ErrGuestAllowanceExceeded
	}

	activeWeight, playerAlreadyActive := countActiveEntries(competition.ID, existing, playerID)
	if playerAlreadyActive {
		return CompetitionEntry{}, ErrAlreadyEntered
	}
	if activeWeight+(1+guestCount) > competition.Capacity {
		return CompetitionEntry{}, ErrCompetitionFull
	}

	return CompetitionEntry{
		CompetitionID: competition.ID,
		PlayerID:      playerID,
		GuestCount:    guestCount,
		Source:        source,
		PaymentStatus: PaymentStatusUnpaid,
		Status:        EntryStatusEntered,
	}, nil
}

// SpotsLeft reports how many places remain free in competition given its
// existing entries: capacity minus the **weighted** occupancy, floored at
// zero.
//
// "Weighted" is the whole point, and it is the same rule Enter enforces —
// both delegate to countActiveEntries, so a spots-left projection can never
// drift from the capacity check that actually admits or rejects an entry.
// An unweighted (headcount) version would advertise 7 free places on a
// capacity-8 Competition holding one entry that brought 3 guests, when the
// truth is 4: a visible lie to every player reading the listing, and one
// they'd only discover when their entry was rejected. That is exactly the
// drift T5's retro finding 1 and T8.6/T8.7's reweighting exist to prevent,
// which is why this ships in the same ticket as the listing that needs it
// rather than as a follow-up.
//
// The floor at zero is defensive only: the DB-level capacity guard
// (db/migrations/0014_competitions.sql) should make an over-occupied
// Competition unreachable. But a read path must never show a player a
// negative number of free places even if that invariant were somehow
// violated, so this mirrors the GREATEST(..., 0) floor the SQL applies
// (db/queries/competitions.sql's ListCompetitions).
//
// This is the pure-Go half of CLAUDE.md rule 4's dual enforcement for the
// same derived quantity: the browse list computes it in SQL (one aggregate
// per row, rather than an N+1 fetch of every Competition's entries), and
// the -tags=integration test asserts the two implementations agree against a
// real Postgres.
//
// existing may be unfiltered — entries belonging to other Competitions are
// ignored, exactly as in Enter, so a caller that hands over everything it
// loaded cannot accidentally count another Competition's entrants here.
func SpotsLeft(competition Competition, existing []CompetitionEntry) int {
	activeWeight, _ := countActiveEntries(competition.ID, existing, "")
	if remaining := competition.Capacity - activeWeight; remaining > 0 {
		return remaining
	}
	return 0
}

// countActiveEntries scans existing for non-cancelled entries scoped to
// competitionID, returning both the total *weighted* number of places those
// entries occupy (each counts as 1 + its own GuestCount) and whether
// playerID already holds one of them.
//
// The competitionID filter mirrors booking.EnsureNoConflict's own-scope
// filtering, so a caller may safely pass an unfiltered slice — a T9.3
// adapter that hands over every entry it loaded cannot accidentally count
// another Competition's entrants against this one's capacity.
//
// Kept as one helper (rather than counting inline in Enter) so that any
// later question of the form "is this Competition full?" — a spots-left
// projection, a waitlist, an admin view — is answered by the same rule
// Enter enforces, instead of a second copy that can drift from it. That is
// exactly the reuse socialplay's countActiveRegistrations was extracted for
// in T6.6.
// MarkPaymentStatus sets PaymentStatus to a new value reported by the
// Payments context (T10.6), via port.CompetitionEntryPaymentUpdater ->
// internal/payments/adapter/competitions ->
// Service.MarkCompetitionEntryPaymentStatus. This is the only path, besides
// Enter's initial "unpaid" default, that may change PaymentStatus — mirrors
// socialplay.Registration.MarkPaymentStatus exactly, including its scope:
// Payments is the single source of truth for whether a status transition
// itself (unpaid -> paid -> refunded) is legal, so this method deliberately
// does not re-derive that state machine — it only guards that status is a
// recognised PaymentStatus value at all (IsValid), so garbage input can
// never be written to the field.
func (e *CompetitionEntry) MarkPaymentStatus(status PaymentStatus) error {
	if !status.IsValid() {
		return ErrInvalidPaymentStatus
	}
	e.PaymentStatus = status
	return nil
}

func countActiveEntries(competitionID string, existing []CompetitionEntry, playerID string) (activeWeight int, playerAlreadyActive bool) {
	for _, e := range existing {
		if e.CompetitionID != competitionID {
			continue
		}
		if e.Status == EntryStatusCancelled {
			continue
		}
		if e.PlayerID == playerID {
			playerAlreadyActive = true
		}
		activeWeight += 1 + e.GuestCount
	}
	return activeWeight, playerAlreadyActive
}
