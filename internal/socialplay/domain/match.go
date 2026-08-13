package domain

import "time"

// Match records a real result played within an existing Game (T10.3): which
// players played, what the score was, and when the result was recorded.
// Lives in this package, not a new bounded context — Match and PlayerRating
// are Social Play concepts per docs/agent-operating-handbook.md A1; only the
// self-reported starting level lives in Identity/Users.
//
// Explicitly not built here, per ADR-0012
// (docs/adr/0012-identity-users-and-match-built-rating-and-matching-
// algorithm-blocked-on-escalated-decisions.md): no PlayerRating field or
// computation anywhere on this type, and nothing in this package reads a
// Match back into a rating. A Match is a recorded fact only — deriving a
// rating from Match history is blocked on ADR-0012's Q1 (how the Player
// Level formula is weighted), a product decision only the platform owner
// can make, not something this ticket guesses at.
type Match struct {
	// ID is assigned by the persistence layer (T10.4), the same way
	// Registration.ID and WaitlistEntry.ID are — RecordMatch deliberately
	// does not take an id parameter (see its own doc comment).
	ID string
	// GameID is the existing Game this Match's result belongs to — same
	// package, same context (docs/process/t10-sprint-plan.md T10.3's
	// cross-context check confirmed Game.ID is stable and already exists
	// for this to reference). Whether GameID actually refers to a real Game
	// requires the repository, which this pure constructor cannot call
	// (CLAUDE.md rule 2) — that existence check is app.Service's job at
	// wiring time (T10.4), mirroring Game.VenueFacilityID's existence-check
	// pattern. RecordMatch only validates non-emptiness.
	GameID string
	// Players is the set of participants, identified by plain player-ID
	// strings — the same representation Registration.PlayerID and
	// WaitlistEntry.PlayerID already use in this package. Social Play has
	// no distinct PlayerID type to reuse, and inventing one here would be
	// exactly the kind of speculative shape ADR-0012 warns against
	// building ahead of a real need. Must hold at least two entries;
	// RecordMatch rejects fewer.
	Players []string
	// Score is a minimal per-player point total, keyed by the same
	// player-ID strings used in Players. Deliberately the simplest shape
	// that records a real result (T10.3's own instruction: "keep it
	// minimal since nothing downstream consumes it this sprint") — a
	// richer shape (set-by-set breakdown, a winner/loser pair, etc.) is
	// left for whichever future ticket actually needs it, not guessed at
	// now. Not required to cover every entry in Players, and RecordMatch
	// does not validate its contents beyond accepting whatever is given —
	// nothing downstream reads Score yet, so there is nothing here for a
	// stricter check to protect.
	Score map[string]int
	// RecordedAt is when this result was recorded — the input timestamp
	// any future rating/matching algorithm would need, though nothing
	// computes on it yet (ADR-0012).
	RecordedAt time.Time
}

// RecordMatch constructs a new Match, validating the invariants that don't
// require any infrastructure: gameID must be non-empty (ErrEmptyGameID),
// players must be non-empty (ErrEmptyPlayers), and players must hold at
// least two entries (ErrTooFewPlayers) — checked in that order, so a caller
// with multiple invalid fields gets the most fundamental error first
// (mirrors NewGame's own precedence discipline, e.g. capacity checked ahead
// of the time range).
//
// Whether gameID refers to a real, non-cancelled Game, and whether the
// caller is that Game's Host or an assigned Game Admin, are both
// app.Service's job at wiring time (T10.4) — this pure constructor cannot
// call the repository or check authorization (CLAUDE.md rule 2/3).
//
// The returned Match's ID is intentionally left empty: like Register and
// JoinWaitlist, RecordMatch does not take an id parameter — assigning a
// durable ID is the app/adapter layer's job at persistence time.
func RecordMatch(gameID string, players []string, score map[string]int, recordedAt time.Time) (Match, error) {
	if gameID == "" {
		return Match{}, ErrEmptyGameID
	}
	if len(players) == 0 {
		return Match{}, ErrEmptyPlayers
	}
	if len(players) < 2 {
		return Match{}, ErrTooFewPlayers
	}

	return Match{
		GameID:     gameID,
		Players:    players,
		Score:      score,
		RecordedAt: recordedAt,
	}, nil
}
