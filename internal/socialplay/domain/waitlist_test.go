package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestJoinWaitlist_RejectsWhenGameNotFull proves a player cannot join the
// waitlist of a Game that still has an open registration slot — they should
// register directly (domain.Register) instead. This is JoinWaitlist's own
// rejection (ErrGameNotFull), not one it borrows from Register.
func TestJoinWaitlist_RejectsWhenGameNotFull(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 4)
	_, err := domain.JoinWaitlist(g, nil, nil, "player-1")
	if !errors.Is(err, domain.ErrGameNotFull) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameNotFull)
	}
}

// TestJoinWaitlist_AcceptsWhenGameFull is the happy path: a Game at
// capacity accepts a waitlist join, producing a "waiting" entry at
// Position 1 when the waitlist was previously empty. This is the exact
// scenario JoinWaitlist's doc comment says reuses Register's own
// capacity-counting logic (via countActiveRegistrations) rather than a
// second, independently-written count.
func TestJoinWaitlist_AcceptsWhenGameFull(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existingRegs := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}

	entry, err := domain.JoinWaitlist(g, existingRegs, nil, "player-2")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry.GameID != g.ID || entry.PlayerID != "player-2" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Position != 1 {
		t.Fatalf("Position = %d, want 1", entry.Position)
	}
	if entry.Status != domain.WaitlistStatusWaiting {
		t.Fatalf("Status = %v, want waiting", entry.Status)
	}
}

// TestJoinWaitlist_EmptyPlayerID mirrors Register's own boundary check.
func TestJoinWaitlist_EmptyPlayerID(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existingRegs := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}
	_, err := domain.JoinWaitlist(g, existingRegs, nil, "")
	if !errors.Is(err, domain.ErrEmptyPlayerID) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyPlayerID)
	}
}

// TestJoinWaitlist_AlreadyActivelyRegistered proves a player who already
// holds an active Registration for this Game cannot also join its
// waitlist — this is the exact ErrAlreadyRegistered Register itself would
// return, checked first (more specific/actionable than "the game is full").
func TestJoinWaitlist_AlreadyActivelyRegistered(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existingRegs := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}
	_, err := domain.JoinWaitlist(g, existingRegs, nil, "player-1")
	if !errors.Is(err, domain.ErrAlreadyRegistered) {
		t.Fatalf("got err %v, want %v", err, domain.ErrAlreadyRegistered)
	}
}

// TestJoinWaitlist_AlreadyOnWaitlist proves a player with an existing
// waiting or promoted entry cannot join a second time, but a player whose
// prior entry is expired or cancelled can rejoin.
func TestJoinWaitlist_AlreadyOnWaitlist(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existingRegs := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}

	tests := []struct {
		name    string
		entries []domain.WaitlistEntry
		wantErr error
	}{
		{
			name: "waiting entry blocks rejoin",
			entries: []domain.WaitlistEntry{
				{ID: "w1", GameID: "g1", PlayerID: "player-2", Position: 1, Status: domain.WaitlistStatusWaiting},
			},
			wantErr: domain.ErrAlreadyOnWaitlist,
		},
		{
			name: "promoted entry blocks rejoin",
			entries: []domain.WaitlistEntry{
				{ID: "w1", GameID: "g1", PlayerID: "player-2", Position: 1, Status: domain.WaitlistStatusPromoted},
			},
			wantErr: domain.ErrAlreadyOnWaitlist,
		},
		{
			name: "expired entry allows rejoin",
			entries: []domain.WaitlistEntry{
				{ID: "w1", GameID: "g1", PlayerID: "player-2", Position: 1, Status: domain.WaitlistStatusExpired},
			},
			wantErr: nil,
		},
		{
			name: "cancelled entry allows rejoin",
			entries: []domain.WaitlistEntry{
				{ID: "w1", GameID: "g1", PlayerID: "player-2", Position: 1, Status: domain.WaitlistStatusCancelled},
			},
			wantErr: nil,
		},
		{
			name: "another game's waiting entry for the same player does not block",
			entries: []domain.WaitlistEntry{
				{ID: "w1", GameID: "some-other-game", PlayerID: "player-2", Position: 1, Status: domain.WaitlistStatusWaiting},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.JoinWaitlist(g, existingRegs, tt.entries, "player-2")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestJoinWaitlist_PositionCountsNonCancelledEntries proves Position is
// len(existing non-cancelled entries) + 1 — expired/promoted entries still
// count (per the doc comment: Position is queue-join order/history, not a
// live rank), only cancelled entries are excluded from the tally.
func TestJoinWaitlist_PositionCountsNonCancelledEntries(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existingRegs := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}
	existingEntries := []domain.WaitlistEntry{
		{ID: "w1", GameID: "g1", PlayerID: "player-a", Position: 1, Status: domain.WaitlistStatusWaiting},
		{ID: "w2", GameID: "g1", PlayerID: "player-b", Position: 2, Status: domain.WaitlistStatusExpired},
		{ID: "w3", GameID: "g1", PlayerID: "player-c", Position: 3, Status: domain.WaitlistStatusCancelled},
		{ID: "w4", GameID: "some-other-game", PlayerID: "player-d", Position: 1, Status: domain.WaitlistStatusWaiting},
	}

	entry, err := domain.JoinWaitlist(g, existingRegs, existingEntries, "player-e")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// 2 non-cancelled entries scoped to g1 (w1 waiting, w2 expired) -- w3 is
	// cancelled (excluded), w4 is scoped to a different game (excluded).
	if entry.Position != 3 {
		t.Fatalf("Position = %d, want 3", entry.Position)
	}
}

// TestWaitlistEntry_Promote proves the waiting -> promoted transition
// records PromotedAt, and that an illegal source state is rejected rather
// than silently accepted.
func TestWaitlistEntry_Promote(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-1", Position: 1, Status: domain.WaitlistStatusWaiting}
	if err := entry.Promote(now); err != nil {
		t.Fatalf("promote from waiting should succeed, got %v", err)
	}
	if entry.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("Status = %v, want promoted", entry.Status)
	}
	if !entry.PromotedAt.Equal(now) {
		t.Fatalf("PromotedAt = %v, want %v", entry.PromotedAt, now)
	}

	if err := entry.Promote(now); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("re-promoting an already-promoted entry should be rejected, got %v", err)
	}
}

// TestWaitlistEntry_Expire proves promoted -> expired, and that expiring a
// non-promoted entry (waiting, or already expired) is rejected.
func TestWaitlistEntry_Expire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  domain.WaitlistStatus
		wantErr error
	}{
		{"promoted expires cleanly", domain.WaitlistStatusPromoted, nil},
		{"waiting cannot expire directly", domain.WaitlistStatusWaiting, domain.ErrIllegalStatusTransition},
		{"already expired cannot expire again", domain.WaitlistStatusExpired, domain.ErrIllegalStatusTransition},
		{"cancelled cannot expire", domain.WaitlistStatusCancelled, domain.ErrIllegalStatusTransition},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-1", Status: tt.status}
			err := entry.Expire()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && entry.Status != domain.WaitlistStatusExpired {
				t.Fatalf("Status = %v, want expired", entry.Status)
			}
		})
	}
}

// TestWaitlistEntry_Cancel proves the BOLA-shaped ownership check (mirrors
// Registration.Cancel's own test) and the waiting/promoted -> cancelled
// transition.
func TestWaitlistEntry_Cancel(t *testing.T) {
	t.Parallel()

	t.Run("owner can cancel from waiting", func(t *testing.T) {
		t.Parallel()
		entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-1", Status: domain.WaitlistStatusWaiting}
		if err := entry.Cancel("player-1"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if entry.Status != domain.WaitlistStatusCancelled {
			t.Fatalf("Status = %v, want cancelled", entry.Status)
		}
	})

	t.Run("owner can cancel from promoted", func(t *testing.T) {
		t.Parallel()
		entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-1", Status: domain.WaitlistStatusPromoted}
		if err := entry.Cancel("player-1"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if entry.Status != domain.WaitlistStatusCancelled {
			t.Fatalf("Status = %v, want cancelled", entry.Status)
		}
	})

	t.Run("wrong actor is rejected and entry is untouched", func(t *testing.T) {
		t.Parallel()
		entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-b", Status: domain.WaitlistStatusWaiting}
		err := entry.Cancel("player-a")
		if !errors.Is(err, domain.ErrNotWaitlistEntryOwner) {
			t.Fatalf("got err %v, want %v", err, domain.ErrNotWaitlistEntryOwner)
		}
		if entry.Status != domain.WaitlistStatusWaiting {
			t.Fatalf("Status = %v, want unchanged (waiting)", entry.Status)
		}
	})

	t.Run("already terminal entries reject cancel", func(t *testing.T) {
		t.Parallel()
		for _, s := range []domain.WaitlistStatus{domain.WaitlistStatusExpired, domain.WaitlistStatusCancelled} {
			entry := domain.WaitlistEntry{ID: "w1", GameID: "g1", PlayerID: "player-1", Status: s}
			if err := entry.Cancel("player-1"); !errors.Is(err, domain.ErrIllegalStatusTransition) {
				t.Fatalf("status %v: got err %v, want ErrIllegalStatusTransition", s, err)
			}
		}
	})
}

// TestSlotReservedByPromotion proves the reservation semantics
// app.Service.RegisterForGame relies on: an unexpired promotion blocks a
// *different* player, never the promoted player themselves, and an expired
// promotion blocks nobody.
func TestSlotReservedByPromotion(t *testing.T) {
	t.Parallel()

	promotedAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	stillOpen := promotedAt.Add(time.Minute)
	afterWindow := promotedAt.Add(domain.PromotionResponseWindow + time.Minute)

	tests := []struct {
		name     string
		entries  []domain.WaitlistEntry
		playerID string
		now      time.Time
		want     bool
	}{
		{
			name: "unexpired promotion blocks a different player",
			entries: []domain.WaitlistEntry{
				{ID: "w1", PlayerID: "player-promoted", Status: domain.WaitlistStatusPromoted, PromotedAt: promotedAt},
			},
			playerID: "player-other",
			now:      stillOpen,
			want:     true,
		},
		{
			name: "unexpired promotion does not block the promoted player themselves",
			entries: []domain.WaitlistEntry{
				{ID: "w1", PlayerID: "player-promoted", Status: domain.WaitlistStatusPromoted, PromotedAt: promotedAt},
			},
			playerID: "player-promoted",
			now:      stillOpen,
			want:     false,
		},
		{
			name: "expired promotion blocks nobody",
			entries: []domain.WaitlistEntry{
				{ID: "w1", PlayerID: "player-promoted", Status: domain.WaitlistStatusPromoted, PromotedAt: promotedAt},
			},
			playerID: "player-other",
			now:      afterWindow,
			want:     false,
		},
		{
			name: "a waiting (not yet promoted) entry never blocks",
			entries: []domain.WaitlistEntry{
				{ID: "w1", PlayerID: "player-waiting", Status: domain.WaitlistStatusWaiting},
			},
			playerID: "player-other",
			now:      stillOpen,
			want:     false,
		},
		{
			name:     "no entries at all never blocks",
			entries:  nil,
			playerID: "player-other",
			now:      stillOpen,
			want:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := domain.SlotReservedByPromotion(tt.entries, tt.playerID, tt.now); got != tt.want {
				t.Fatalf("SlotReservedByPromotion = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWaitlistEntry_HasExpired proves the response-timeout boundary: exactly
// at the window it's expired (>= not >), before it isn't, and a non-promoted
// entry is never "expired" regardless of PromotedAt.
func TestWaitlistEntry_HasExpired(t *testing.T) {
	t.Parallel()

	promotedAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status domain.WaitlistStatus
		now    time.Time
		want   bool
	}{
		{"just promoted, well within window", domain.WaitlistStatusPromoted, promotedAt.Add(time.Minute), false},
		{"one second before window ends", domain.WaitlistStatusPromoted, promotedAt.Add(domain.PromotionResponseWindow - time.Second), false},
		{"exactly at window boundary counts as expired", domain.WaitlistStatusPromoted, promotedAt.Add(domain.PromotionResponseWindow), true},
		{"well past window", domain.WaitlistStatusPromoted, promotedAt.Add(domain.PromotionResponseWindow + time.Hour), true},
		{"waiting entry is never expired even with an old PromotedAt", domain.WaitlistStatusWaiting, promotedAt.Add(24 * time.Hour), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := domain.WaitlistEntry{ID: "w1", Status: tt.status, PromotedAt: promotedAt}
			if got := entry.HasExpired(tt.now); got != tt.want {
				t.Fatalf("HasExpired = %v, want %v", got, tt.want)
			}
		})
	}
}
