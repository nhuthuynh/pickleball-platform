package domain

import (
	"sort"
	"time"
)

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
	// now. Not required to cover every entry in Players — RecordMatch does
	// not check that every Players entry has a Score key, or that a Score
	// key names a real Players entry — only that the map itself is
	// non-empty (ErrEmptyScore, T10.4's own error-handling requirement:
	// "empty score -> InvalidArgument"). Nothing downstream reads Score's
	// per-player contents yet, so there is nothing here for a stricter,
	// per-key check to protect; a wholesale-empty Score is different — it
	// isn't a result at all.
	Score map[string]int
	// RecordedAt is when this result was recorded — the input timestamp
	// any future rating/matching algorithm would need, though nothing
	// computes on it yet (ADR-0012).
	RecordedAt time.Time
}

// RecordMatch constructs a new Match, validating the invariants that don't
// require any infrastructure: gameID must be non-empty (ErrEmptyGameID),
// players must be non-empty (ErrEmptyPlayers), players must hold at least
// two entries (ErrTooFewPlayers), and score must be non-empty (ErrEmptyScore,
// T10.4) — checked in that order, so a caller with multiple invalid fields
// gets the most fundamental error first (mirrors NewGame's own precedence
// discipline, e.g. capacity checked ahead of the time range). ErrEmptyScore
// is checked last since T10.3 already established GameID/Players as the
// more fundamental identity/participant facts; a Match with no players at
// all is a more basic problem than one with no score.
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
	if len(score) == 0 {
		return Match{}, ErrEmptyScore
	}

	return Match{
		GameID:     gameID,
		Players:    players,
		Score:      score,
		RecordedAt: recordedAt,
	}, nil
}

// Winners returns the player ids holding the highest score in this Match,
// sorted so the result is reproducible (Go randomises map iteration, and a
// caller counting wins across a history must get the same answer twice).
//
// THE RULE, answered by the Product Owner on 2026-10-07: *highest points
// wins, and a tie counts for everyone tied.* So a tied match returns every
// tied player and there is no draw category — which is exactly why the
// answer needed no change to what is stored. In particular it works for
// doubles unchanged: partners share one side's point total, so both appear
// in Score with the same value and either both win or neither does. This
// type needs no notion of a side, a team or a partner, and deliberately
// does not gain one here.
//
// Scope, stated because it is easy to over-read: a "win" is a fact about
// this one Match. Accumulating wins into a player's record, and weighting
// that record into a Level, is Identity's job
// (internal/identity/domain.PlayerRecord/ComputeLevel) — this package does
// not import that one, and nothing here computes a rating (CLAUDE.md
// rule 3; ADR-0012's "no PlayerRating field" still holds, since a
// PlayerRecord is a count, not a stored rating).
//
// Score is not required to cover every entry in Players (see the field's
// own doc comment), so Winners reports on who was actually scored. An empty
// or nil Score — which RecordMatch refuses but a zero-value Match can
// carry — returns an empty slice rather than panicking.
func (m Match) Winners() []string {
	if len(m.Score) == 0 {
		return nil
	}

	best := 0
	first := true
	for _, points := range m.Score {
		if first || points > best {
			best, first = points, false
		}
	}

	winners := make([]string, 0, len(m.Score))
	for playerID, points := range m.Score {
		if points == best {
			winners = append(winners, playerID)
		}
	}
	sort.Strings(winners)
	return winners
}

// Won reports whether playerID is among this Match's Winners. A player with
// no recorded score has no result, so Won is false for them — "absent from
// Score" must never read as "won", which is the shape a win-counting loop
// over a Game's Players would otherwise get wrong.
func (m Match) Won(playerID string) bool {
	if playerID == "" {
		return false
	}
	points, ok := m.Score[playerID]
	if !ok {
		return false
	}
	for _, other := range m.Score {
		if other > points {
			return false
		}
	}
	return true
}
