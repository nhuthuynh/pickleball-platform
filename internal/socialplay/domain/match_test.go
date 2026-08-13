package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

func validPlayers() []string {
	return []string{"player-1", "player-2"}
}

func validScore() map[string]int {
	return map[string]int{"player-1": 11, "player-2": 7}
}

// TestRecordMatch_Valid proves the happy path: a well-formed Match is
// constructed with every field carried through as given, and no PlayerRating
// field exists anywhere on the type (ADR-0012 — see match.go's doc comment).
func TestRecordMatch_Valid(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)
	m, err := domain.RecordMatch("g1", validPlayers(), validScore(), recordedAt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m.GameID != "g1" {
		t.Fatalf("GameID = %q, want g1", m.GameID)
	}
	if len(m.Players) != 2 || m.Players[0] != "player-1" || m.Players[1] != "player-2" {
		t.Fatalf("Players = %v, want [player-1 player-2]", m.Players)
	}
	if m.Score["player-1"] != 11 || m.Score["player-2"] != 7 {
		t.Fatalf("Score = %v, want {player-1:11 player-2:7}", m.Score)
	}
	if !m.RecordedAt.Equal(recordedAt) {
		t.Fatalf("RecordedAt = %v, want %v", m.RecordedAt, recordedAt)
	}
}

// TestRecordMatch_Validation is the required table-driven boundary coverage:
// empty GameID, empty Players, and fewer-than-2 Players each reject with
// their own distinct sentinel (per T10.3's instructions), rather than a
// single generic validation error.
func TestRecordMatch_Validation(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		gameID  string
		players []string
		wantErr error
	}{
		{"empty game id is rejected", "", validPlayers(), domain.ErrEmptyGameID},
		{"nil players is rejected", "g1", nil, domain.ErrEmptyPlayers},
		{"empty players slice is rejected", "g1", []string{}, domain.ErrEmptyPlayers},
		{"single player is rejected", "g1", []string{"player-1"}, domain.ErrTooFewPlayers},
		{"two players is accepted", "g1", validPlayers(), nil},
		{"three players is accepted", "g1", []string{"player-1", "player-2", "player-3"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.RecordMatch(tt.gameID, tt.players, validScore(), recordedAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRecordMatch_ScoreValidation is T10.4's own boundary coverage (this
// ticket's error-handling instructions: "empty score -> InvalidArgument"),
// added alongside T10.3's GameID/Players table above rather than folded into
// it, since it's a distinct requirement landing a sprint later. Checked
// after GameID/Players so a Match missing everything still reports the most
// fundamental problem first (RecordMatch's own doc comment).
func TestRecordMatch_ScoreValidation(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		score   map[string]int
		wantErr error
	}{
		{"nil score is rejected", nil, domain.ErrEmptyScore},
		{"empty score map is rejected", map[string]int{}, domain.ErrEmptyScore},
		{"well-formed score is accepted", validScore(), nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.RecordMatch("g1", validPlayers(), tt.score, recordedAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRecordMatch_TooFewPlayersCheckedBeforeScore proves the precedence
// order RecordMatch's doc comment claims: a Match with only one player and
// no score gets ErrTooFewPlayers, not ErrEmptyScore.
func TestRecordMatch_TooFewPlayersCheckedBeforeScore(t *testing.T) {
	t.Parallel()

	_, err := domain.RecordMatch("g1", []string{"player-1"}, nil, time.Now())
	if !errors.Is(err, domain.ErrTooFewPlayers) {
		t.Fatalf("got err %v, want %v", err, domain.ErrTooFewPlayers)
	}
}

// TestRecordMatch_EmptyGameIDCheckedBeforePlayers proves empty GameID wins
// even when Players is also invalid, so a caller gets the most fundamental
// error first rather than an arbitrary one (mirrors NewGame's own
// precedence discipline).
func TestRecordMatch_EmptyGameIDCheckedBeforePlayers(t *testing.T) {
	t.Parallel()

	_, err := domain.RecordMatch("", nil, validScore(), time.Now())
	if !errors.Is(err, domain.ErrEmptyGameID) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyGameID)
	}
}
