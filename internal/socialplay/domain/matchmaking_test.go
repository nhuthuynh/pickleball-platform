package domain_test

import (
	"errors"
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
	if got := m.Winners(); len(got) != 0 {
		t.Fatalf("Winners() = %v, want empty", got)
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
			"an empty player id",
			[]domain.RatedPlayer{rated("a", 3), rated("", 3)}, nil, domain.ErrEmptyPlayers,
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
			[][2]string{{"a", ""}}, domain.ErrEmptyPlayers,
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

// TestNoGenderAnywhereInMatchmaking is ADR-0012 Q2's guard on this side of
// the boundary: Q1 is answered and Q2 is not, so matching here is
// level-only and no type may carry a protected attribute. Identity has the
// same assertion over its own types.
func TestNoGenderAnywhereInMatchmaking(t *testing.T) {
	t.Parallel()

	for _, v := range []any{domain.RatedPlayer{}, domain.Matchup{}, domain.MatchupSuggestion{}, domain.Match{}} {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			if name := typ.Field(i).Name; strings.Contains(strings.ToLower(name), "gender") {
				t.Fatalf("%s.%s: ADR-0012 Q2 is unanswered, so matching here must stay level-only",
					typ.Name(), name)
			}
		}
	}
}
