package domain

import (
	"math"
	"sort"
)

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
//
// Level's RANGE is deliberately unconstrained: this package does not own the
// scale (Identity does) and re-asserting its bounds here would be a second,
// drifting copy of someone else's invariant. A computed Level, a
// self-reported seed or a manual override all work.
//
// Its FINITENESS is not optional, and SuggestMatchups refuses a NaN or an
// infinity (ErrInvalidPlayerLevel). That is not fastidiousness: NaN makes
// the level comparator stop being a strict weak ordering — `!=` is true
// while both `<` are false, so the id tie-break never runs — which is
// undefined behaviour for sort.Slice. T65.2's review fed one set in three
// input orders and got three different arrangements, which would have made
// the determinism promised below simply false. Identity's own Level refuses
// NaN (Level.IsValid), so this guard catches a level that came from
// somewhere else — which this field's unconstrained type explicitly invites.
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
// THE ALGORITHM, and the objective it optimises — named, because the loose
// version of this paragraph claimed more than the code delivers. Sort by
// level and pair neighbours. For pairing a sorted set into couples,
// adjacent pairing minimises the TOTAL of the pairwise gaps, so this is not
// a heuristic standing in for an optimum. With an odd count one player must
// sit out, and the player chosen is the one whose removal leaves the
// smallest total gap (found by trying each candidate — cheap at a Game's
// scale and exact; the sit-out can only be at an even sorted position,
// since any other choice leaves a pair straddling it). Taking the last
// player in the list instead would sit out whoever happens to be strongest,
// which is both arbitrary and worse.
//
// THE OBJECTIVE IS THE TOTAL GAP, NOT THE WORST MATCH. These come apart on
// an odd set, and T65.2's review brute-forced a counterexample:
//
//	levels 2.0, 3.2, 3.2, 3.7, 5.0
//	this code: sit out 2.0 -> (3.2,3.2) and (3.7,5.0)   total 1.3, worst 1.3
//	minimax:   sit out 5.0 -> (2.0,3.2) and (3.2,3.7)   total 1.7, worst 1.2
//
// Total was chosen because it is the whole set's evenness rather than one
// pair's, and because an organiser reading a suggestion sees every match,
// not only the worst.
//
// The evidence, with its provenance, since the two halves differ.
// Total-gap minimality is checked in-tree against an exhaustive
// enumeration, over eight deterministic level sets covering ties, clusters
// and odd counts (TestSuggestMatchupsMinimisesTheTotalGap); the
// counterexample above is pinned by TestTheWorstMatchIsNotTheObjective. The
// wider figure — 0 total-gap failures and ~1.3% minimax failures across
// 280,000 random sets of 2 to 8 players — comes from T65.2's review, not
// from a command in this repository, and is quoted as the review's
// measurement rather than as a standing claim this tree can reproduce.
//
// Determinism: equal levels tie-break by player id, so the same set always
// produces the same advice. An organiser comparing today's suggestion with
// yesterday's needs the difference to mean something.
//
// Errors, in the order they are actually checked — which is what this list
// claimed and did not describe until T65.2's review ran it:
//
//  1. ErrEmptyPlayers — no players at all.
//  2. ErrTooFewPlayers — one player, who cannot be matched by anything.
//  3. ErrEmptyPlayerID — a blank id in the player set or in a pin. This
//     was ErrEmptyPlayers, whose own message ("at least one player is
//     required") told a caller with a blank id to add a player; the
//     sentinel for a blank id already existed in this package.
//  4. ErrInvalidPlayerLevel — a NaN or infinite level.
//  5. ErrDuplicatePlayer — a player listed twice in the set.
//  6. ErrUnknownPinnedPlayer — any pinned id not in the set. Checked across
//     EVERY pin before duplicates are considered, so that two faults in one
//     call give the same answer regardless of which slot the bad id sits
//     in. It did not: `[{a,b},{stranger,b}]` and `[{a,b},{b,stranger}]` are
//     the same two faults and returned different sentinels.
//  7. ErrDuplicatePlayer again — a player pinned twice, or a pin naming one
//     player on both sides.
//
// A caller with several faults therefore gets the most fundamental one, and
// gets it deterministically.
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
			return MatchupSuggestion{}, ErrEmptyPlayerID
		}
		if math.IsNaN(p.Level) || math.IsInf(p.Level, 0) {
			return MatchupSuggestion{}, ErrInvalidPlayerLevel
		}
		if _, seen := byID[p.PlayerID]; seen {
			return MatchupSuggestion{}, ErrDuplicatePlayer
		}
		byID[p.PlayerID] = p
	}

	// Every pinned id is checked for emptiness and membership BEFORE any
	// duplicate is considered, so the answer does not depend on which slot a
	// bad id happens to occupy. Two passes rather than one is the whole cost.
	for _, pair := range pinned {
		for _, id := range pair {
			if id == "" {
				return MatchupSuggestion{}, ErrEmptyPlayerID
			}
			if _, ok := byID[id]; !ok {
				return MatchupSuggestion{}, ErrUnknownPinnedPlayer
			}
		}
	}

	matchups := make([]Matchup, 0, len(players)/2+len(pinned))
	spokenFor := make(map[string]bool, len(pinned)*2)
	for _, pair := range pinned {
		var sides [2]RatedPlayer
		for i, id := range pair {
			if spokenFor[id] {
				return MatchupSuggestion{}, ErrDuplicatePlayer
			}
			spokenFor[id] = true
			sides[i] = byID[id]
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
