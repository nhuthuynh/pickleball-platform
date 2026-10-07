package domain

// Player Level — the computed, history-derived value ADR-0012's Q1 asked
// about and, until 2026-10-07, blocked.
//
// WHAT WAS DECIDED AND BY WHOM. The Product Owner answered Q1 on
// 2026-10-07: *balance win rate with experience — win rate is the signal,
// but a player needs a reasonable number of games before the rating is
// trusted, and early results move it less.* That answer fixes the
// formula's CHARACTER. It does not fix its CONSTANTS: ConfidenceGames and
// the shape of the ramp below were chosen by the engineer who wrote this
// file, with the reasoning stated on each, and either may be retuned
// without going back to the Product Owner. Do not read them as decided.
//
// WHAT THIS IS NOT. Q2 — whether gender-mix matching is in scope at all —
// is deliberately still unanswered (it is the protected-attribute
// question, and nothing in this file needs it). ADR-0012's trigger says an
// answered Q1 with an unanswered Q2 ships level-only matching, so there is
// no Gender field here, in the schema, or in any proto; see
// player_level_test.go's TestNoGenderFieldOnAnyLevelType, which asserts
// that by reflection rather than leaving it to a reviewer's eye.
//
// WHERE THE INPUT COMES FROM — and the gap, named rather than invented.
// This package computes a Level from a PlayerRecord it is handed. Nothing
// in this repository can yet BUILD that record for a real player: a Match
// (internal/socialplay/domain) stores Score as a per-player point map with
// no winner concept until T65's Match.Winners, there is no query that
// reads one player's matches (db/queries/socialplay.sql has CreateMatch
// and ListMatchesForGame only), and socialplay's player ids are opaque
// registration ids rather than identity_users.id. Those three gaps are
// filed as #333, not papered over. The formula is pure and testable
// without them, which is why it ships now: ADR-0012's trigger requires the
// sprint immediately following the answer to build what the answer
// unblocks.
//
// RULE 2. This file imports nothing — not even math; the ramp is
// deliberately arithmetic a reader can check by hand.

// Level is a computed player level on the SAME closed 1..5 scale as
// SelfReportedStartingLevel, so the seed and the computed value are
// directly comparable and a player's number never changes meaning when
// their history starts. It is a float64 rather than an int because the
// whole point of the answer is that the value moves gradually; rounding to
// an int would make the confidence ramp invisible for the first several
// games.
type Level float64

const (
	// MinLevel and MaxLevel bound the scale. They match
	// MinSelfReportedStartingLevel/MaxSelfReportedStartingLevel by
	// construction — see levelFromWinRate.
	MinLevel Level = Level(MinSelfReportedStartingLevel)
	MaxLevel Level = Level(MaxSelfReportedStartingLevel)
)

// IsValid reports whether l falls within the closed [MinLevel, MaxLevel]
// range. A NaN is not valid: every comparison against it is false, which is
// the behaviour WithManualOverride relies on to refuse one.
func (l Level) IsValid() bool {
	return l >= MinLevel && l <= MaxLevel
}

// ConfidenceGames is the number of recorded games after which a player's
// own results alone set their Level, and below which the value stays
// weighted toward their self-reported starting level.
//
// 20 was chosen here, not handed down, for three reasons:
//
//  1. It is where the statistics stop being embarrassing. The standard
//     error of a win rate at 20 games is at worst sqrt(0.25/20) ≈ 0.112,
//     i.e. about ±0.9 of a level at 95% confidence on this 4-wide scale.
//     That is still coarse — which is the honest argument for the ramp
//     existing at all — but it is the point where one more win stops
//     moving the number by a visible step.
//  2. It is reachable. A weekly recreational player clears 20 games in a
//     season; a league regular clears it in weeks. A threshold nobody
//     reaches would make every level permanently provisional, which is the
//     same as having no ramp.
//  3. It is explainable in one sentence to a player: "after 20 games your
//     level is entirely your results; before that we blend in the level
//     you started at."
//
// Retuning it is a code change with no product sign-off needed. What WOULD
// need sign-off is changing the character of the formula — e.g. making win
// rate not the signal, or removing the ramp.
const ConfidenceGames = 20

// PlayerRecord is the history the formula consumes: how many games a
// player has a recorded result for, and how many of them they won.
//
// WHAT COUNTS AS A WIN is a product answer, given 2026-10-07: *highest
// points wins, and a tie counts for everyone tied.* It is not implemented
// here — deriving wins from a Match's Score is Social Play's job, since
// Match is Social Play's aggregate (docs/agent-operating-handbook.md A1),
// and this package must not import that one (CLAUDE.md rule 3). See
// socialplay/domain.Match.Winners, which is where that rule lives. This
// type is the boundary the two meet at.
//
// Draws are therefore not a third category: a tied match is a win for
// every player on the tie, so Wins <= GamesPlayed always holds and a
// record that breaks it is refused (ErrImpossibleRecord) rather than
// clamped.
type PlayerRecord struct {
	GamesPlayed int
	Wins        int
}

// WinRate is Wins/GamesPlayed, or 0 for a player with no games. Callers
// should validate the record first (ComputeLevel does); this method does
// not refuse an impossible one, it only avoids dividing by zero.
func (r PlayerRecord) WinRate() float64 {
	if r.GamesPlayed <= 0 {
		return 0
	}
	return float64(r.Wins) / float64(r.GamesPlayed)
}

// Valid reports whether the record is arithmetically possible.
func (r PlayerRecord) Valid() bool {
	return r.GamesPlayed >= 0 && r.Wins >= 0 && r.Wins <= r.GamesPlayed
}

// PlayerLevel is a computed Level together with the two facts a consumer
// needs in order to use it honestly.
type PlayerLevel struct {
	// Value is the level itself, always within [MinLevel, MaxLevel].
	Value Level

	// GamesPlayed is the history Value was computed from. Carried on the
	// result so a consumer can show "based on N games" without going back
	// to the record, and so Provisional is auditable.
	GamesPlayed int

	// Provisional is true while GamesPlayed < ConfidenceGames — i.e. while
	// Value is still partly the player's own claim about themselves rather
	// than their results. A UI that shows a Level without showing this is
	// presenting a claim as a measurement; matchmaking may still use the
	// value, which is the point of blending rather than withholding it.
	Provisional bool

	// ManuallySet is true when Value came from WithManualOverride rather
	// than from the formula. CLAUDE.md's locked decision is that
	// matchmaking is "always manually overridable"; this flag is what makes
	// an override survive the next recompute instead of being silently
	// replaced the next time the player plays (see RecomputeLevel).
	ManuallySet bool
}

// levelFromWinRate maps a win rate in [0,1] onto the 1..5 scale: 0% is
// MinLevel, 100% is MaxLevel, linearly between. This is the "win rate is
// the signal" half of the answer, and it is linear because nothing in the
// answer justifies a curve — a curve would be an invented product opinion
// about which part of the field to spread out.
func levelFromWinRate(winRate float64) Level {
	return MinLevel + Level(winRate)*(MaxLevel-MinLevel)
}

// confidence is the weight the player's own results carry, ramping linearly
// from 0 at no games to 1 at ConfidenceGames and staying there.
//
// Linear, rather than sqrt or exponential, for one reason worth more than
// its mathematical elegance: it has an exact, quotable hand-over point.
// "After 20 games it is all your results" is checkable by a player and by
// a test (see the at-exactly-ConfidenceGames case); "asymptotically
// approaches your results" is neither.
func confidence(gamesPlayed int) float64 {
	if gamesPlayed >= ConfidenceGames {
		return 1
	}
	if gamesPlayed <= 0 {
		return 0
	}
	return float64(gamesPlayed) / float64(ConfidenceGames)
}

// ComputeLevel is the formula, and the whole of it:
//
//	observed = 1 + winRate*4                      // win rate is the signal
//	c        = min(gamesPlayed, 20) / 20          // experience is the weight
//	level    = seed + c*(observed - seed)         // early results move it less
//
// At zero games the result is exactly the seed — the locked cold-start
// mechanism ("new players seeded by a self-reported starting level"). At
// ConfidenceGames and beyond the seed has no weight at all, so an inflated
// self-report is fully corrected by a real history rather than haunting the
// value forever.
//
// Because level is a weighted mean of two values that are both on the
// scale, the result is always on the scale; no clamp is needed and none is
// applied (a clamp would hide an arithmetic mistake rather than prevent
// one — the sweep in TestEveryComputedValueStaysOnTheScale is what holds
// this).
//
// Errors: ErrInvalidSelfReportedStartingLevel for a seed off the 1..5
// scale, ErrImpossibleRecord for a record that cannot have happened
// (negative counts, or more wins than games). The seed is checked first,
// matching this package's existing precedence discipline (see
// User.UpdateSelfReportedLevel): the more fundamental input is validated
// before the derived one.
func ComputeLevel(seed SelfReportedStartingLevel, rec PlayerRecord) (PlayerLevel, error) {
	if !seed.IsValid() {
		return PlayerLevel{}, ErrInvalidSelfReportedStartingLevel
	}
	if !rec.Valid() {
		return PlayerLevel{}, ErrImpossibleRecord
	}

	seedLevel := Level(seed)
	observed := levelFromWinRate(rec.WinRate())
	c := confidence(rec.GamesPlayed)

	return PlayerLevel{
		Value:       seedLevel + Level(c)*(observed-seedLevel),
		GamesPlayed: rec.GamesPlayed,
		Provisional: rec.GamesPlayed < ConfidenceGames,
	}, nil
}

// RecomputeLevel refreshes a PlayerLevel against a newer record, and is the
// entry point anything that reacts to a recorded Match should call rather
// than ComputeLevel.
//
// The difference is the one that makes "always manually overridable" true:
// if current.ManuallySet, the overridden Value is KEPT and only the
// history-derived facts (GamesPlayed, Provisional) advance. Calling
// ComputeLevel directly on a player who has an override would discard it at
// their next match, which is an override that lasts until the player does
// anything — not an override.
//
// A caller that genuinely wants to drop the override calls
// ClearManualOverride first; that is deliberately explicit.
func RecomputeLevel(current PlayerLevel, seed SelfReportedStartingLevel, rec PlayerRecord) (PlayerLevel, error) {
	fresh, err := ComputeLevel(seed, rec)
	if err != nil {
		return PlayerLevel{}, err
	}
	if current.ManuallySet {
		fresh.Value = current.Value
		fresh.ManuallySet = true
	}
	return fresh, nil
}

// WithManualOverride returns a copy of pl whose Value is v, flagged as
// manually set. v must be on the scale (ErrInvalidLevel otherwise) — an
// override is a human's judgement about where a player sits, not a licence
// to leave the range every consumer assumes. NaN is refused by the same
// check, since IsValid's comparisons are all false for it.
//
// GamesPlayed and Provisional are untouched: an override changes the value,
// not the history it was derived from, and a consumer still needs to know
// the history is short.
func (pl PlayerLevel) WithManualOverride(v Level) (PlayerLevel, error) {
	if !v.IsValid() {
		return PlayerLevel{}, ErrInvalidLevel
	}
	pl.Value = v
	pl.ManuallySet = true
	return pl, nil
}

// ClearManualOverride returns a copy of pl with the override flag dropped.
// It does NOT recompute Value — the next RecomputeLevel does that, and
// keeping the two steps separate means clearing an override never needs a
// record on hand.
func (pl PlayerLevel) ClearManualOverride() PlayerLevel {
	pl.ManuallySet = false
	return pl
}
