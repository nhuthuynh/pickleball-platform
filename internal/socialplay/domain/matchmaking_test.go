package domain_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestMatchWinners is the win rule, answered by the Product Owner on
// 2026-10-07: *highest points wins, and a tie counts for everyone tied.*
//
// It is implemented here rather than in internal/identity/domain because
// Match is Social Play's aggregate (docs/agent-operating-handbook.md A1) and
// Identity must not import this package (CLAUDE.md rule 3). Identity's
// formula consumes a games/wins count; this is what produces the "win" half
// of it.
//
// The rule is deliberately indifferent to team structure, which is why the
// answer works for doubles unchanged: partners share one side's score, so
// both appear in Score with the same value and both are winners (or both
// are not). Nothing about a side, a team, or a partner needs to be stored.
func TestMatchWinners(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		score map[string]int
		want  []string
		why   string
	}{
		{
			name:  "singles, a clear winner",
			score: map[string]int{"p1": 11, "p2": 7},
			want:  []string{"p1"},
			why:   "highest points wins",
		},
		{
			name:  "singles, a tie counts for both",
			score: map[string]int{"p1": 9, "p2": 9},
			want:  []string{"p1", "p2"},
			why:   "a tie counts for everyone tied",
		},
		{
			name:  "doubles, both members of the winning side win",
			score: map[string]int{"p1": 11, "p2": 11, "p3": 8, "p4": 8},
			want:  []string{"p1", "p2"},
			why:   "partners share a side score, so the rule needs no notion of a team",
		},
		{
			name:  "doubles, a tied match counts for all four",
			score: map[string]int{"p1": 10, "p2": 10, "p3": 10, "p4": 10},
			want:  []string{"p1", "p2", "p3", "p4"},
			why:   "everyone tied at the top wins, however many that is",
		},
		{
			name:  "a three-way tie above a fourth player",
			score: map[string]int{"p1": 11, "p2": 11, "p3": 11, "p4": 2},
			want:  []string{"p1", "p2", "p3"},
			why:   "the tie set is whoever holds the maximum, not a pair",
		},
		{
			name:  "zero-all is still a tie, not a no-result",
			score: map[string]int{"p1": 0, "p2": 0},
			want:  []string{"p1", "p2"},
			why:   "the rule is about the maximum, and 0 can be the maximum",
		},
		{
			name:  "negative scores do not break the maximum",
			score: map[string]int{"p1": -3, "p2": -7},
			want:  []string{"p1"},
			why:   "no score floor is enforced anywhere, so the rule must not assume one",
		},
		{
			name:  "a single scored player wins by default",
			score: map[string]int{"p1": 11},
			want:  []string{"p1"},
			why:   "RecordMatch does not require Score to cover every player, so this shape reaches here",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, err := domain.RecordMatch("g1", []string{"p1", "p2"}, tc.score, time.Now())
			if err != nil {
				t.Fatalf("RecordMatch: %v", err)
			}
			got := m.Winners()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Winners() = %v, want %v (%s)", got, tc.want, tc.why)
			}
			for _, w := range tc.want {
				if !m.Won(w) {
					t.Fatalf("Won(%q) = false, but %q is a winner", w, w)
				}
			}
		})
	}
}

// TestWinnersIsSortedAndStable: the result feeds a counting loop, so a
// non-deterministic order would make a derived win count non-reproducible
// even though the rule is deterministic. Go randomises map iteration, so
// this is a real risk, not a theoretical one.
func TestWinnersIsSortedAndStable(t *testing.T) {
	t.Parallel()

	score := map[string]int{"zoe": 11, "amy": 11, "max": 11, "bob": 3}
	m, err := domain.RecordMatch("g1", []string{"amy", "bob"}, score, time.Now())
	if err != nil {
		t.Fatalf("RecordMatch: %v", err)
	}

	want := []string{"amy", "max", "zoe"}
	for i := 0; i < 50; i++ {
		if got := m.Winners(); !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: Winners() = %v, want %v", i, got, want)
		}
	}
}

// TestWonIsFalseForAnUnscoredPlayer: a player listed in Players but absent
// from Score has no result, and "absent" must not read as "won".
func TestWonIsFalseForAnUnscoredPlayer(t *testing.T) {
	t.Parallel()

	m, err := domain.RecordMatch("g1", []string{"p1", "p2", "ghost"},
		map[string]int{"p1": 11, "p2": 7}, time.Now())
	if err != nil {
		t.Fatalf("RecordMatch: %v", err)
	}
	if m.Won("ghost") {
		t.Fatalf("Won(ghost) = true for a player with no recorded score")
	}
	if m.Won("") {
		t.Fatalf(`Won("") = true`)
	}

	// The assertions above pass whether or not Won actually checks for
	// presence: an absent player reads as 0 points, and 0 loses to 11.
	// Mutation-verified and the mutant SURVIVED, which is this block's
	// reason for existing. Where every recorded score is at or below zero,
	// a missing player's implicit 0 is the maximum, and the only thing
	// standing between that and "the ghost won" is the presence check.
	for _, score := range []map[string]int{
		{"p1": 0, "p2": 0},
		{"p1": -3, "p2": -7},
	} {
		zeroish, err := domain.RecordMatch("g1", []string{"p1", "p2"}, score, time.Now())
		if err != nil {
			t.Fatalf("RecordMatch: %v", err)
		}
		if zeroish.Won("ghost") {
			t.Fatalf("Won(ghost) = true against %v: an absent player's implicit 0 was treated as a score", score)
		}
		for _, w := range zeroish.Winners() {
			if w == "ghost" {
				t.Fatalf("Winners() = %v against %v: includes a player who never scored",
					zeroish.Winners(), score)
			}
		}
	}
}

// TestWinnersOfAnEmptyScore: RecordMatch refuses an empty Score, but a
// zero-value Match can reach here from a repository row or a test fixture,
// and "nobody scored" must be no winners rather than a panic.
func TestWinnersOfAnEmptyScore(t *testing.T) {
	t.Parallel()

	var m domain.Match
	got := m.Winners()
	if len(got) != 0 {
		t.Fatalf("Winners() = %v, want empty", got)
	}
	// Non-nil, because the doc comment now promises "an empty, non-nil
	// slice" and reinstating the deleted early return was a surviving
	// mutant. A caller comparing with reflect.DeepEqual against []string{}
	// is the shape that would break on nil.
	if got == nil {
		t.Fatalf("Winners() = nil; the doc promises an empty, non-nil slice")
	}
}

// ---------------------------------------------------------------------
// Level-only match suggestion (ADR-0012's trigger, Q1 half)
// ---------------------------------------------------------------------

func rated(id string, level float64) domain.RatedPlayer {
	return domain.RatedPlayer{PlayerID: id, Level: level}
}

func pairIDs(s domain.MatchupSuggestion) [][2]string {
	out := make([][2]string, 0, len(s.Matchups))
	for _, m := range s.Matchups {
		out = append(out, [2]string{m.A.PlayerID, m.B.PlayerID})
	}
	return out
}

// TestSuggestMatchupsPairsNearestLevels is the suggestion itself: sort by
// level and pair neighbours, which is the arrangement that minimises the
// total level gap across the set.
func TestSuggestMatchupsPairsNearestLevels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		players      []domain.RatedPlayer
		want         [][2]string
		wantUnpaired []string
		why          string
	}{
		{
			name:    "four players split into two close pairs",
			players: []domain.RatedPlayer{rated("a", 4.5), rated("b", 2.0), rated("c", 2.2), rated("d", 4.0)},
			want:    [][2]string{{"b", "c"}, {"d", "a"}},
			why:     "gaps 0.2 + 0.5 beat any other split of this set",
		},
		{
			name: "six players",
			players: []domain.RatedPlayer{
				rated("a", 1.0), rated("b", 1.1), rated("c", 3.0),
				rated("d", 3.2), rated("e", 4.9), rated("f", 5.0),
			},
			want: [][2]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
			why:  "sorted neighbours",
		},
		{
			name:    "two players always pair, however far apart",
			players: []domain.RatedPlayer{rated("a", 1.0), rated("b", 5.0)},
			want:    [][2]string{{"a", "b"}},
			why:     "a suggestion is advice, not a refusal — the organiser decides whether to use it",
		},
		{
			name:         "the odd player out is chosen to minimise the total gap, not taken from the end",
			players:      []domain.RatedPlayer{rated("lonely", 1.0), rated("a", 3.0), rated("b", 3.1)},
			want:         [][2]string{{"a", "b"}},
			wantUnpaired: []string{"lonely"},
			why:          "sitting out the outlier costs 0.1 of gap; sitting out b costs 2.0",
		},
		{
			name: "five players, the sit-out comes from the middle when that is cheapest",
			players: []domain.RatedPlayer{
				rated("a", 1.0), rated("b", 1.1), rated("mid", 3.0), rated("d", 4.9), rated("e", 5.0),
			},
			want:         [][2]string{{"a", "b"}, {"d", "e"}},
			wantUnpaired: []string{"mid"},
			why:          "0.1 + 0.1 beats every arrangement that pairs the middle player",
		},
		{
			name:    "equal levels tie-break by player id, so the suggestion is reproducible",
			players: []domain.RatedPlayer{rated("d", 3.0), rated("c", 3.0), rated("b", 3.0), rated("a", 3.0)},
			want:    [][2]string{{"a", "b"}, {"c", "d"}},
			why:     "Go's sort is not stable by default and the input order here is adversarial",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.SuggestMatchups(tc.players, nil)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !reflect.DeepEqual(pairIDs(got), tc.want) {
				t.Fatalf("matchups = %v, want %v (%s)", pairIDs(got), tc.want, tc.why)
			}
			var unpaired []string
			for _, p := range got.Unpaired {
				unpaired = append(unpaired, p.PlayerID)
			}
			if len(unpaired) != len(tc.wantUnpaired) {
				t.Fatalf("unpaired = %v, want %v", unpaired, tc.wantUnpaired)
			}
			for i := range unpaired {
				if unpaired[i] != tc.wantUnpaired[i] {
					t.Fatalf("unpaired = %v, want %v", unpaired, tc.wantUnpaired)
				}
			}
			for _, m := range got.Matchups {
				if m.Pinned {
					t.Fatalf("matchup %v is flagged Pinned but nothing was pinned", m)
				}
			}
		})
	}
}

// TestSuggestMatchupsHonoursPinnedPairings is CLAUDE.md's locked "always
// manually overridable" decision, expressed in code rather than in a
// comment: an organiser's own pairings are carried through untouched and the
// algorithm arranges only the players they left to it.
//
// Overridability as a returned-value-nobody-has-to-use would be unfalsifiable
// (every pure function is ignorable). Pinning is the version a test can
// hold: the manual choice survives a call that would not have made it.
func TestSuggestMatchupsHonoursPinnedPairings(t *testing.T) {
	t.Parallel()

	players := []domain.RatedPlayer{
		rated("beginner", 1.0), rated("novice", 1.2),
		rated("strong", 4.8), rated("expert", 5.0),
	}

	// The organiser deliberately pairs across the level gap — a coaching
	// pairing the formula would never suggest.
	got, err := domain.SuggestMatchups(players, [][2]string{{"beginner", "expert"}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	want := [][2]string{{"beginner", "expert"}, {"novice", "strong"}}
	if !reflect.DeepEqual(pairIDs(got), want) {
		t.Fatalf("matchups = %v, want %v — a pinned pairing must survive", pairIDs(got), want)
	}
	if !got.Matchups[0].Pinned {
		t.Fatalf("the pinned matchup is not flagged Pinned, so a UI cannot tell it apart from advice")
	}
	if got.Matchups[1].Pinned {
		t.Fatalf("a suggested matchup is flagged Pinned")
	}
	if len(got.Unpaired) != 0 {
		t.Fatalf("unpaired = %v, want none", got.Unpaired)
	}
}

// TestSuggestMatchupsPinnedCanLeaveAnOddRemainder: pinning changes who is
// left over, and the leftover must be reported rather than dropped.
func TestSuggestMatchupsPinnedCanLeaveAnOddRemainder(t *testing.T) {
	t.Parallel()

	players := []domain.RatedPlayer{rated("a", 1.0), rated("b", 1.1), rated("c", 3.0)}

	got, err := domain.SuggestMatchups(players, [][2]string{{"a", "c"}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if want := [][2]string{{"a", "c"}}; !reflect.DeepEqual(pairIDs(got), want) {
		t.Fatalf("matchups = %v, want %v", pairIDs(got), want)
	}
	if len(got.Unpaired) != 1 || got.Unpaired[0].PlayerID != "b" {
		t.Fatalf("unpaired = %v, want [b]", got.Unpaired)
	}
}

// TestSuggestMatchupsRejects covers every input the suggestion must refuse.
func TestSuggestMatchupsRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		players []domain.RatedPlayer
		pinned  [][2]string
		want    error
	}{
		{"no players", nil, nil, domain.ErrEmptyPlayers},
		{"one player", []domain.RatedPlayer{rated("a", 3)}, nil, domain.ErrTooFewPlayers},
		{
			// ErrEmptyPlayerID, not ErrEmptyPlayers, which this case
			// expected until T65.2's review pointed out that
			// ErrEmptyPlayers' own message — "at least one player is
			// required" — tells a caller with a blank id to add a player.
			// The right sentinel already existed in this package.
			"an empty player id",
			[]domain.RatedPlayer{rated("a", 3), rated("", 3)}, nil, domain.ErrEmptyPlayerID,
		},
		{
			"a single player with a blank id is still too few players",
			[]domain.RatedPlayer{rated("", 3)}, nil, domain.ErrTooFewPlayers,
		},
		{
			"a NaN level",
			[]domain.RatedPlayer{rated("a", 3), rated("b", math.NaN())}, nil, domain.ErrInvalidPlayerLevel,
		},
		{
			"an infinite level",
			[]domain.RatedPlayer{rated("a", 3), rated("b", math.Inf(1))}, nil, domain.ErrInvalidPlayerLevel,
		},
		{
			"the same player listed twice",
			[]domain.RatedPlayer{rated("a", 3), rated("a", 4)}, nil, domain.ErrDuplicatePlayer,
		},
		{
			"a pinned player who is not in the set",
			[]domain.RatedPlayer{rated("a", 3), rated("b", 3)},
			[][2]string{{"a", "stranger"}}, domain.ErrUnknownPinnedPlayer,
		},
		{
			"a pinned pair naming one player twice",
			[]domain.RatedPlayer{rated("a", 3), rated("b", 3)},
			[][2]string{{"a", "a"}}, domain.ErrDuplicatePlayer,
		},
		{
			"a player pinned into two different pairs",
			[]domain.RatedPlayer{rated("a", 3), rated("b", 3), rated("c", 3), rated("d", 3)},
			[][2]string{{"a", "b"}, {"a", "c"}}, domain.ErrDuplicatePlayer,
		},
		{
			"a pinned pair with an empty id",
			[]domain.RatedPlayer{rated("a", 3), rated("b", 3)},
			[][2]string{{"a", ""}}, domain.ErrEmptyPlayerID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.SuggestMatchups(tc.players, tc.pinned); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestSuggestMatchupsDoesNotMutateItsInput: the implementation sorts, and
// sorting the caller's slice in place would silently reorder a Game's own
// player list at the call site. Pure means pure (CLAUDE.md rule 2).
func TestSuggestMatchupsDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()

	players := []domain.RatedPlayer{rated("z", 5.0), rated("a", 1.0), rated("m", 3.0), rated("b", 1.5)}
	before := append([]domain.RatedPlayer(nil), players...)
	pinned := [][2]string{{"z", "a"}}
	pinnedBefore := [][2]string{{"z", "a"}}

	if _, err := domain.SuggestMatchups(players, pinned); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(players, before) {
		t.Fatalf("input was reordered: %v, want %v", players, before)
	}
	if !reflect.DeepEqual(pinned, pinnedBefore) {
		t.Fatalf("pinned was modified: %v, want %v", pinned, pinnedBefore)
	}
}

// TestMatchupLevelGapIsAbsolute: the gap is what a UI shows next to a
// suggestion, and it must not depend on which player landed in A.
func TestMatchupLevelGapIsAbsolute(t *testing.T) {
	t.Parallel()

	a := domain.Matchup{A: rated("a", 2.0), B: rated("b", 4.5)}
	b := domain.Matchup{A: rated("b", 4.5), B: rated("a", 2.0)}
	if got := a.LevelGap(); got < 2.5-1e-9 || got > 2.5+1e-9 {
		t.Fatalf("LevelGap() = %v, want 2.5", got)
	}
	if a.LevelGap() != b.LevelGap() {
		t.Fatalf("LevelGap() depends on argument order: %v vs %v", a.LevelGap(), b.LevelGap())
	}
}

// TestPinnedFaultPrecedenceIsPositionIndependent: two faults in one call
// must give the same answer however the bad id is arranged. They did not —
// `[{a,b},{stranger,b}]` returned ErrUnknownPinnedPlayer and
// `[{a,b},{b,stranger}]` returned ErrDuplicatePlayer, because both checks
// sat in the same per-id loop.
func TestPinnedFaultPrecedenceIsPositionIndependent(t *testing.T) {
	t.Parallel()

	players := []domain.RatedPlayer{rated("a", 1), rated("b", 2), rated("c", 3), rated("d", 4)}

	for _, pinned := range [][][2]string{
		{{"a", "b"}, {"stranger", "b"}},
		{{"a", "b"}, {"b", "stranger"}},
		{{"stranger", "b"}, {"a", "b"}},
	} {
		_, err := domain.SuggestMatchups(players, pinned)
		if !errors.Is(err, domain.ErrUnknownPinnedPlayer) {
			t.Fatalf("pinned %v: err = %v, want ErrUnknownPinnedPlayer — membership is checked across every "+
				"pin before any duplicate", pinned, err)
		}
	}
}

// TestFaultPrecedenceHoldsAcrossCategories is the test the one above should
// have been. Fixing the two reported positional cases left the CLASS intact:
// with validation interleaved per element, a NaN on the first player beat a
// blank id on the second, because each player was validated completely
// before the next was looked at. Four such cases survived the first fix, and
// the doc comment had meanwhile been strengthened to claim the class.
//
// Every case here puts two faults in DIFFERENT elements, so the only way to
// pass is for each category to be its own pass over the whole input.
func TestFaultPrecedenceHoldsAcrossCategories(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		players []domain.RatedPlayer
		pinned  [][2]string
		want    error
	}{
		{
			"a blank id on a later player beats a NaN on an earlier one",
			[]domain.RatedPlayer{rated("a", math.NaN()), rated("", 3)}, nil,
			domain.ErrEmptyPlayerID,
		},
		{
			"a blank id beats a duplicate that appears before it",
			[]domain.RatedPlayer{rated("a", 3), rated("a", 3), rated("", 3)}, nil,
			domain.ErrEmptyPlayerID,
		},
		{
			"a NaN on a later player beats a duplicate that appears before it",
			[]domain.RatedPlayer{rated("a", 3), rated("a", 3), rated("b", math.NaN())}, nil,
			domain.ErrInvalidPlayerLevel,
		},
		{
			"a blank id in a later pin beats an unknown id in an earlier one",
			[]domain.RatedPlayer{rated("a", 1), rated("b", 2)},
			[][2]string{{"a", "stranger"}, {"", "b"}},
			domain.ErrEmptyPlayerID,
		},
		{
			"a blank id in a pin beats a NaN in the player set",
			[]domain.RatedPlayer{rated("a", 1), rated("b", math.NaN())},
			[][2]string{{"a", ""}},
			domain.ErrEmptyPlayerID,
		},
		{
			"an unknown pinned id beats a pin duplicate that appears before it",
			[]domain.RatedPlayer{rated("a", 1), rated("b", 2), rated("c", 3)},
			[][2]string{{"a", "b"}, {"a", "stranger"}},
			domain.ErrUnknownPinnedPlayer,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.SuggestMatchups(tc.players, tc.pinned)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v — each category must be its own pass over the whole input, "+
					"or the documented precedence is only a description of one loop's body", err, tc.want)
			}
		})
	}
}

// TestSuggestMatchupsIsOrderIndependent is the determinism this function
// promises, driven over permutations of one set rather than asserted. It is
// here because NaN broke exactly this property before SuggestMatchups
// refused one: three input orders of a single set produced three different
// arrangements, since a NaN comparator is not a strict weak ordering and
// sort.Slice's behaviour is then undefined.
func TestSuggestMatchupsIsOrderIndependent(t *testing.T) {
	t.Parallel()

	base := []domain.RatedPlayer{
		rated("a", 1.0), rated("b", 1.0), rated("c", 3.0),
		rated("d", 3.0), rated("e", 4.9), rated("f", 5.0),
	}
	want, err := domain.SuggestMatchups(base, nil)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}

	for _, order := range [][]int{
		{5, 4, 3, 2, 1, 0},
		{1, 4, 0, 2, 5, 3},
		{2, 0, 5, 1, 3, 4},
	} {
		shuffled := make([]domain.RatedPlayer, 0, len(base))
		for _, i := range order {
			shuffled = append(shuffled, base[i])
		}
		got, err := domain.SuggestMatchups(shuffled, nil)
		if err != nil {
			t.Fatalf("order %v: %v", order, err)
		}
		if !reflect.DeepEqual(pairIDs(got), pairIDs(want)) {
			t.Fatalf("order %v gave %v, want %v", order, pairIDs(got), pairIDs(want))
		}
	}
}

// TestTotalLevelGap: production code whose own doc says it "is what makes
// 'the suggestion is balanced' a checkable claim rather than an adjective",
// and which nothing checked — a surviving mutant in T65.2's review.
func TestTotalLevelGap(t *testing.T) {
	t.Parallel()

	var empty domain.MatchupSuggestion
	if got := empty.TotalLevelGap(); got != 0 {
		t.Fatalf("TotalLevelGap() = %v on an empty suggestion, want 0", got)
	}

	s := domain.MatchupSuggestion{Matchups: []domain.Matchup{
		{A: rated("a", 1.0), B: rated("b", 1.5)},
		{A: rated("c", 4.0), B: rated("d", 3.0), Pinned: true},
	}}
	if got := s.TotalLevelGap(); got < 1.5-1e-9 || got > 1.5+1e-9 {
		t.Fatalf("TotalLevelGap() = %v, want 1.5 (0.5 + 1.0, pinned matchups included)", got)
	}
}

// TestSuggestMatchupsMinimisesTheTotalGap is the optimality claim in
// SuggestMatchups' doc comment, held by brute force rather than by the
// comment. Every arrangement of every small set is enumerated and compared
// against what the function returns.
//
// It exists because the claim had NO test evidence at all: the table above
// asserts specific pairings, which says nothing about whether another
// arrangement would have been better.
func TestSuggestMatchupsMinimisesTheTotalGap(t *testing.T) {
	t.Parallel()

	// A deterministic spread of levels, including ties and clusters, rather
	// than random input: a property test that cannot be re-run on the same
	// data is a property test whose failures cannot be reproduced.
	levelSets := [][]float64{
		{1, 2},
		{1, 1, 5, 5},
		{2, 3.7, 3.2, 3.2, 5},
		{1, 1.1, 3, 4.9, 5},
		{1, 2, 3, 4, 5, 5},
		{5, 1, 3, 3, 3, 1, 2},
		{2.5, 2.5, 2.5, 2.5},
		{1, 5, 3, 3.01, 2.99, 4, 1.5, 4.5},
	}

	for _, levels := range levelSets {
		players := make([]domain.RatedPlayer, 0, len(levels))
		for i, l := range levels {
			players = append(players, rated(string(rune('a'+i)), l))
		}

		got, err := domain.SuggestMatchups(players, nil)
		if err != nil {
			t.Fatalf("levels %v: %v", levels, err)
		}
		// The count is asserted because the gap check is one-sided: a
		// mutant returning FEWER matchups would have a smaller total and
		// pass. TestSuggestMatchupsPairsNearestLevels catches that today,
		// but this test's comment implies it stands alone, so it should.
		if wantPairs := len(levels) / 2; len(got.Matchups) != wantPairs {
			t.Fatalf("levels %v: %d matchup(s), want %d", levels, len(got.Matchups), wantPairs)
		}
		best := bestTotalGap(levels)
		if total := got.TotalLevelGap(); total > best+1e-9 {
			t.Fatalf("levels %v: total gap %v, but %v is achievable (%v)", levels, total, best, pairIDs(got))
		}
	}
}

// TestTheWorstMatchIsNotTheObjective pins the counterexample T65.2's review
// brute-forced, so the doc comment's narrowed claim stays honest. On an odd
// set, minimising the total gap and minimising the WORST match come apart,
// and this code chooses the total.
func TestTheWorstMatchIsNotTheObjective(t *testing.T) {
	t.Parallel()

	got, err := domain.SuggestMatchups([]domain.RatedPlayer{
		rated("low", 2.0), rated("mid1", 3.2), rated("mid2", 3.2),
		rated("mid3", 3.7), rated("high", 5.0),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(got.Unpaired) != 1 || got.Unpaired[0].PlayerID != "low" {
		t.Fatalf("unpaired = %v, want [low] — the total-gap optimum sits out the outlier", got.Unpaired)
	}
	var worst float64
	for _, m := range got.Matchups {
		if g := m.LevelGap(); g > worst {
			worst = g
		}
	}
	if worst < 1.3-1e-9 {
		t.Fatalf("worst gap = %v; this test records that the total-gap objective accepts 1.3 here "+
			"when a 1.2 arrangement exists. If it now fails, the objective changed and the doc "+
			"comment naming it is stale", worst)
	}
}

// bestTotalGap enumerates every pairing (and every sit-out, for an odd set)
// and returns the smallest achievable total gap. Exponential, which is fine
// for the sizes above and is the point: it shares no code or reasoning with
// the implementation it checks.
func bestTotalGap(levels []float64) float64 {
	if len(levels)%2 == 1 {
		best := math.Inf(1)
		for i := range levels {
			rest := make([]float64, 0, len(levels)-1)
			rest = append(rest, levels[:i]...)
			rest = append(rest, levels[i+1:]...)
			if g := bestTotalGap(rest); g < best {
				best = g
			}
		}
		return best
	}
	if len(levels) == 0 {
		return 0
	}
	best := math.Inf(1)
	for j := 1; j < len(levels); j++ {
		gap := math.Abs(levels[0] - levels[j])
		rest := make([]float64, 0, len(levels)-2)
		rest = append(rest, levels[1:j]...)
		rest = append(rest, levels[j+1:]...)
		if total := gap + bestTotalGap(rest); total < best {
			best = total
		}
	}
	return best
}

// TestWinnersAndWonAgreeOnAnEmptyPlayerID: RecordMatch validates only the
// LENGTH of players and score, so a blank id is a state this package will
// construct — and Winners reported it as the winner while Won denied it.
func TestWinnersAndWonAgreeOnAnEmptyPlayerID(t *testing.T) {
	t.Parallel()

	m, err := domain.RecordMatch("g1", []string{"", "p1"}, map[string]int{"": 11, "p1": 3}, time.Now())
	if err != nil {
		t.Fatalf("RecordMatch: %v — if this now errors, empty ids are refused at construction "+
			"and this test's premise is gone", err)
	}
	winners := m.Winners()
	if len(winners) != 1 || winners[0] != "" {
		t.Fatalf("Winners() = %q, want the blank id — it holds the maximum", winners)
	}
	if !m.Won("") {
		t.Fatalf(`Won("") = false while Winners() reports it; the two must agree for every input`)
	}
}

// TestAScoreKeyNamingNobodyStillWins records the other direction of the
// Players/Score mismatch, which is unconstrained in both and matters because
// a win count would be built from Score while a games-played count would be
// built from Players. Named in #333; pinned here so the behaviour is a
// recorded fact rather than a surprise.
func TestAScoreKeyNamingNobodyStillWins(t *testing.T) {
	t.Parallel()

	m, err := domain.RecordMatch("g1", []string{"p1", "p2"},
		map[string]int{"p1": 3, "ghost": 11}, time.Now())
	if err != nil {
		t.Fatalf("RecordMatch: %v", err)
	}
	if got := m.Winners(); len(got) != 1 || got[0] != "ghost" {
		t.Fatalf("Winners() = %v, want [ghost] — Winners reports what was scored, faithfully", got)
	}
	if !m.Won("ghost") {
		t.Fatalf("Won(ghost) = false while Winners() reports it")
	}
}

// TestNoGenderAnywhereInThisPackage is ADR-0012 Q2's guard on this side of
// the boundary: Q1 is answered and Q2 is not, so matching here is
// level-only and nothing may carry a protected attribute.
//
// It parses this package's own source rather than listing types, which is
// what it did until T65.2's review added a Gender field to Registration —
// the natural home for a gender-mix feature, since a Registration is how a
// player joins a Game — and watched the listed version pass. Identity holds
// the repo-wide version of this check, covering the schema and the protos
// too; this one is local, so a change to this package fails in this
// package.
func TestNoGenderAnywhereInThisPackage(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	parsed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), e.Name(), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", e.Name(), err)
		}
		parsed++
		name := e.Name()
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.StructType:
				for _, f := range node.Fields.List {
					for _, id := range f.Names {
						if strings.Contains(strings.ToLower(id.Name), "gender") {
							t.Errorf("%s: struct field %q — ADR-0012 Q2 is unanswered, so matching here "+
								"stays level-only", name, id.Name)
						}
					}
				}
			case *ast.TypeSpec:
				if strings.Contains(strings.ToLower(node.Name.Name), "gender") {
					t.Errorf("%s: type %q — same prohibition", name, node.Name.Name)
				}
			}
			return true
		})
	}
	if parsed < 10 {
		t.Fatalf("parsed only %d file(s) in this package; the scan is broken and proved nothing", parsed)
	}
}
