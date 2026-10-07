package domain

import "sort"

// Level-only automated match suggestion, manually overridable.
//
// WHY THIS EXISTS NOW. ADR-0012 blocked automated matching on two escalated
// product decisions. Q1 (how the Player Level formula is weighted) was
// answered 2026-10-07; Q2 (whether gender-mix matching is in scope at all —
// collecting and acting on a protected attribute) was deliberately NOT
// asked. ADR-0012's trigger covers exactly this case: *"If only one of
// Q1/Q2 is answered, build the part that answer unblocks (e.g., an answered
// Q1 with an unanswered Q2 ships level-only automated matching, still with
// no Gender field)."* So: level only, nothing else, and no Gender field
// anywhere here — asserted by TestNoGenderAnywhereInMatchmaking rather than
// left to a reviewer's eye.
//
// MANUALLY OVERRIDABLE, AS A PROPERTY RATHER THAN A PROMISE. CLAUDE.md's
// locked decision is that matchmaking is "automated from history, always
// manually overridable". "The caller may ignore the return value" would
// satisfy that sentence vacuously — every pure function is ignorable — so
// overridability is built in as pinning: an organiser's own pairings are
// passed in, carried through untouched and flagged, and the algorithm
// arranges only the players they left to it.
//
// WHAT IS NOT HERE. No RPC, no repository read, no persistence: this is the
// pure half. Wiring it needs three things this repository does not yet have
// — a query that reads one player's match history, a bridge from Social
// Play's opaque player ids to identity_users.id, and a place to put a
// computed level — all three filed as #333 rather than invented here.
// The Level values below are therefore supplied by the caller.
//
// Rule 2: this package imports nothing outside the standard library, and
// this file in particular does not import internal/identity/domain (rule 3)
// — hence Level as a plain float64 rather than identity's Level type.

// RatedPlayer is a player plus the level the caller wants them arranged by.
// Level is deliberately an unconstrained float64: this package does not own
// the scale (Identity does) and re-asserting its bounds here would be a
// second, drifting copy of someone else's invariant. Anything comparable
// works — a computed Level, a self-reported seed, a manual override.
type RatedPlayer struct {
	PlayerID string
	Level    float64
}

// Matchup is one suggested or pinned pairing. Singles-shaped (two sides of
// one player each) because that is the only shape the level-only answer
// determines; balancing a doubles pairing is a different problem — two
// partners' levels combine, and how they combine is an unanswered product
// question — so it is not guessed at here.
type Matchup struct {
	A, B RatedPlayer

	// Pinned is true when an organiser fixed this pairing and the
	// algorithm merely carried it through. A UI that cannot tell a human's
	// choice from the machine's advice will eventually present one as the
	// other, which is the failure this flag exists to prevent.
	Pinned bool
}

// LevelGap is the absolute level difference across the matchup — the one
// number that says how even a suggestion is. Absolute, so it does not
// depend on which player happened to land in A.
func (m Matchup) LevelGap() float64 {
	gap := m.A.Level - m.B.Level
	if gap < 0 {
		return -gap
	}
	return gap
}

// MatchupSuggestion is the whole answer: the pairings (pinned ones first,
// in the order they were pinned, then the suggested ones in ascending
// level), and whoever could not be paired.
type MatchupSuggestion struct {
	Matchups []Matchup

	// Unpaired holds the players left over — at most one, since only an
	// odd remainder can be left. Reported rather than dropped: a player
	// who silently disappears from a schedule is a bug the organiser finds
	// out about at the court.
	Unpaired []RatedPlayer
}

// TotalLevelGap is the sum of every matchup's gap, including pinned ones.
// It is what makes "the suggestion is balanced" a checkable claim rather
// than an adjective.
func (s MatchupSuggestion) TotalLevelGap() float64 {
	var total float64
	for _, m := range s.Matchups {
		total += m.LevelGap()
	}
	return total
}

// SuggestMatchups arranges players into level-matched pairings, honouring
// any the organiser has already pinned.
//
// THE ALGORITHM, and why it is this one: sort by level and pair neighbours.
// For pairing a sorted set into couples, adjacent pairing minimises the
// total of the pairwise gaps — so this is not a heuristic standing in for
// an optimum, it is the optimum for "make every match as even as possible"
// on a single dimension. With an odd count one player must sit out, and the
// player chosen is the one whose removal leaves the smallest total gap
// (found by trying each candidate, which is cheap at a Game's scale and
// exact; the sit-out can only be at an even sorted position, since any
// other choice leaves a pair straddling it). Taking the last player in the
// list instead would sit out whoever happens to be strongest, which is both
// arbitrary and worse.
//
// Determinism: equal levels tie-break by player id, so the same set always
// produces the same advice. An organiser comparing today's suggestion with
// yesterday's needs the difference to mean something.
//
// Errors: ErrEmptyPlayers (no players, or any empty player id — including
// inside a pinned pair), ErrTooFewPlayers (one player, which cannot be
// matched at all), ErrDuplicatePlayer (a player listed or pinned twice),
// ErrUnknownPinnedPlayer (a pin naming someone outside the set). Checked in
// that order, most fundamental first, mirroring RecordMatch's precedence
// discipline.
//
// Neither argument is modified (the sort runs on a copy) — a Game's own
// player list is a common thing to pass in, and reordering the caller's
// slice would be an invisible side effect in a package that promises purity.
func SuggestMatchups(players []RatedPlayer, pinned [][2]string) (MatchupSuggestion, error) {
	if len(players) == 0 {
		return MatchupSuggestion{}, ErrEmptyPlayers
	}
	if len(players) < 2 {
		return MatchupSuggestion{}, ErrTooFewPlayers
	}

	byID := make(map[string]RatedPlayer, len(players))
	for _, p := range players {
		if p.PlayerID == "" {
			return MatchupSuggestion{}, ErrEmptyPlayers
		}
		if _, seen := byID[p.PlayerID]; seen {
			return MatchupSuggestion{}, ErrDuplicatePlayer
		}
		byID[p.PlayerID] = p
	}

	matchups := make([]Matchup, 0, len(players)/2+len(pinned))
	spokenFor := make(map[string]bool, len(pinned)*2)
	for _, pair := range pinned {
		var sides [2]RatedPlayer
		for i, id := range pair {
			if id == "" {
				return MatchupSuggestion{}, ErrEmptyPlayers
			}
			p, ok := byID[id]
			if !ok {
				return MatchupSuggestion{}, ErrUnknownPinnedPlayer
			}
			if spokenFor[id] {
				return MatchupSuggestion{}, ErrDuplicatePlayer
			}
			spokenFor[id] = true
			sides[i] = p
		}
		matchups = append(matchups, Matchup{A: sides[0], B: sides[1], Pinned: true})
	}

	free := make([]RatedPlayer, 0, len(players))
	for _, p := range players {
		if !spokenFor[p.PlayerID] {
			free = append(free, p)
		}
	}
	sort.Slice(free, func(i, j int) bool {
		if free[i].Level != free[j].Level {
			return free[i].Level < free[j].Level
		}
		return free[i].PlayerID < free[j].PlayerID
	})

	sitOut := -1
	if len(free)%2 == 1 {
		sitOut = cheapestSitOut(free)
	}

	var unpaired []RatedPlayer
	remaining := make([]RatedPlayer, 0, len(free))
	for i, p := range free {
		if i == sitOut {
			unpaired = append(unpaired, p)
			continue
		}
		remaining = append(remaining, p)
	}
	for i := 0; i+1 < len(remaining); i += 2 {
		matchups = append(matchups, Matchup{A: remaining[i], B: remaining[i+1]})
	}

	return MatchupSuggestion{Matchups: matchups, Unpaired: unpaired}, nil
}

// cheapestSitOut returns the index into the level-sorted sorted slice of the
// player whose removal leaves the smallest total adjacent-pair gap. Only
// even indices are considered: removing an odd-indexed player leaves a pair
// straddling the gap it opened, which is never better than removing one of
// that pair's own neighbours. Ties go to the lowest index, so the answer is
// deterministic.
func cheapestSitOut(sorted []RatedPlayer) int {
	best, bestGap := 0, 0.0
	first := true
	for drop := 0; drop < len(sorted); drop += 2 {
		gap := 0.0
		prev := -1
		for i := range sorted {
			if i == drop {
				continue
			}
			if prev < 0 {
				prev = i
				continue
			}
			gap += sorted[i].Level - sorted[prev].Level
			prev = -1
		}
		if first || gap < bestGap {
			best, bestGap, first = drop, gap, false
		}
	}
	return best
}
