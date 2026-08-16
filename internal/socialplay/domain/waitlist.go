package domain

import "time"

// WaitlistStatus is a WaitlistEntry's lifecycle state.
type WaitlistStatus string

const (
	// WaitlistStatusWaiting is a queued entry not yet offered a slot.
	WaitlistStatusWaiting WaitlistStatus = "waiting"
	// WaitlistStatusPromoted is an entry currently offered an open slot,
	// within its response window (see PromotionResponseWindow) — it does
	// not yet consume a Registration's capacity slot, it reserves the
	// right to claim one (app.Service.RegisterForGame honours this
	// reservation; see that method's doc comment).
	WaitlistStatusPromoted WaitlistStatus = "promoted"
	// WaitlistStatusExpired is a promoted entry whose response window
	// elapsed without the player registering — terminal, never re-promoted
	// itself (a fresh JoinWaitlist call is required to queue again).
	WaitlistStatusExpired WaitlistStatus = "expired"
	// WaitlistStatusCancelled is an entry the player voluntarily withdrew
	// (WaitlistEntry.Cancel) — terminal, mirrors RegistrationStatusCancelled.
	WaitlistStatusCancelled WaitlistStatus = "cancelled"
)

// PromotionResponseWindow is how long a promoted WaitlistEntry has to
// convert into a real Registration before app.Service.ExpireWaitlistPromotion
// (or an equivalent sweep) skips it and promotes the next waiting entry
// instead. Product-tunable (PM/PO call, ADR-0006's own deferral of the exact
// duration to backlog refinement) — deliberately NOT load-bearing on the
// promotion-ordering invariant itself: that invariant is closed at the DB
// level by db/migrations/0008_socialplay_waitlist_promotion.sql regardless
// of what this constant is set to. Changing this value is a product
// decision, not a correctness one.
const PromotionResponseWindow = 30 * time.Minute

// WaitlistEntry is a Player's queued position for a Game that was full at
// the time they tried to join (T6.6 / ADR-0006). Lives in this package (not
// a new bounded context) because it only ever makes sense scoped to a Game,
// the same way Registration does — see ADR-0006's status line and the T6
// sprint plan's PE scoping note (standalone court/slot waitlists are a
// separate, deferred design).
type WaitlistEntry struct {
	ID       string
	GameID   string
	PlayerID string
	// Position is the entry's FIFO ordinal within its Game's waitlist,
	// assigned once at JoinWaitlist time and never renumbered — see
	// JoinWaitlist's doc comment for why a promoted/expired entry still
	// counts toward the next joiner's Position.
	Position int
	Status   WaitlistStatus
	// PromotedAt is the instant this entry last transitioned to
	// WaitlistStatusPromoted — the zero value until that happens. Used by
	// HasExpired to evaluate PromotionResponseWindow.
	PromotedAt time.Time
}

// JoinWaitlist builds a new WaitlistEntry for playerID against game, legal
// only when the Game is actually full for this player.
//
// Cancelled Game (T19.1, closing #212): a Game whose Status is
// StatusCancelled returns ErrGameCancelled, checked FIRST — before the
// not-full/already-registered/already-on-waitlist checks below — the
// identical ordering rationale and identical sentinel domain.Register's own
// cancelled-Game check uses (see that function's doc comment): the game not
// being in a bookable state at all is the more fundamental fact.
//
// Reuse, not duplication (CLAUDE.md rule 4 / T6.6 ticket instruction):
// "actually full for this player" is derived from the exact same
// countActiveRegistrations helper Register itself calls — JoinWaitlist does
// not maintain a second, independently-written capacity-counting loop that
// could silently drift from Register's. A player already actively
// registered gets the same ErrAlreadyRegistered Register would give them
// (checked first, same precedence Register itself uses — the more specific,
// actionable error wins); a player with an existing active
// (waiting/promoted) waitlist entry gets ErrAlreadyOnWaitlist; a player
// trying to join the waitlist of a Game that still has an open slot gets
// ErrGameNotFull (they should register directly instead).
//
// Position is assigned as len(existing non-cancelled entries) + 1 (FIFO,
// ADR-0006's v1 default) — "non-cancelled" deliberately includes expired and
// promoted entries in the count (only a voluntary Cancel removes an entry
// from this tally), so Position reflects queue-join order/history, not a
// live "how many people are still ahead of me" rank; a caller wanting the
// latter derives it from the still-waiting subset instead of Position
// itself.
func JoinWaitlist(game Game, existingRegistrations []Registration, existingEntries []WaitlistEntry, playerID string) (WaitlistEntry, error) {
	if game.Status == StatusCancelled {
		return WaitlistEntry{}, ErrGameCancelled
	}
	if playerID == "" {
		return WaitlistEntry{}, ErrEmptyPlayerID
	}

	activeWeight, playerAlreadyActive := countActiveRegistrations(game.ID, existingRegistrations, playerID)
	if playerAlreadyActive {
		return WaitlistEntry{}, ErrAlreadyRegistered
	}
	if activeWeight < game.Capacity {
		return WaitlistEntry{}, ErrGameNotFull
	}

	nonCancelled := 0
	for _, e := range existingEntries {
		if e.GameID != game.ID {
			continue
		}
		if e.Status == WaitlistStatusCancelled {
			continue
		}
		if e.PlayerID == playerID && (e.Status == WaitlistStatusWaiting || e.Status == WaitlistStatusPromoted) {
			return WaitlistEntry{}, ErrAlreadyOnWaitlist
		}
		nonCancelled++
	}

	return WaitlistEntry{
		GameID:   game.ID,
		PlayerID: playerID,
		Position: nonCancelled + 1,
		Status:   WaitlistStatusWaiting,
	}, nil
}

// Promote transitions a WaitlistEntry from waiting to promoted, recording
// now as PromotedAt (the anchor HasExpired measures PromotionResponseWindow
// from). Legal only from WaitlistStatusWaiting — mirrors
// Registration.Cancel/Game.Cancel's "illegal transition rejected, not
// silently accepted" pattern.
func (w *WaitlistEntry) Promote(now time.Time) error {
	if w.Status != WaitlistStatusWaiting {
		return ErrIllegalStatusTransition
	}
	w.Status = WaitlistStatusPromoted
	w.PromotedAt = now
	return nil
}

// Expire transitions a promoted WaitlistEntry to expired. Legal only from
// WaitlistStatusPromoted. Does not itself check PromotionResponseWindow —
// that's app.Service.ExpireWaitlistPromotion's job (it must reject an
// early/premature expiry attempt with ErrWaitlistPromotionNotExpired before
// ever calling this method), the same domain/app split ScheduleGame uses:
// this method only enforces the state-machine legality of the transition,
// not the timing policy around when it should be called.
func (w *WaitlistEntry) Expire() error {
	if w.Status != WaitlistStatusPromoted {
		return ErrIllegalStatusTransition
	}
	w.Status = WaitlistStatusExpired
	return nil
}

// Cancel transitions a WaitlistEntry to cancelled, but only for its owner —
// mirrors Registration.Cancel's object-level (BOLA-shaped) ownership check
// exactly, including checking ownership before the status transition so a
// wrong actor gets a consistent answer regardless of the entry's current
// state, and leaving the entry untouched on rejection. Legal from either
// waiting or promoted (a player can withdraw even after being promoted, not
// just while still queued); cancelling an already-terminal entry (expired or
// cancelled) is rejected via ErrIllegalStatusTransition.
//
// actorPlayerID is a verified principal as of T12.8, and — as of T29.2,
// closing the Social Play third of #164 — a resolved **User.ID** (uuid), the
// same identifier space w.PlayerID holds. This method's comparison logic is
// unchanged by that migration — see domain.Registration.Cancel's identical
// note.
func (w *WaitlistEntry) Cancel(actorPlayerID string) error {
	if actorPlayerID != w.PlayerID {
		return ErrNotWaitlistEntryOwner
	}
	if w.Status != WaitlistStatusWaiting && w.Status != WaitlistStatusPromoted {
		return ErrIllegalStatusTransition
	}
	w.Status = WaitlistStatusCancelled
	return nil
}

// SlotReservedByPromotion reports whether the freed capacity slot behind
// entries is currently reserved by an unexpired promotion belonging to a
// player other than playerID. Used by app.Service.RegisterForGame (T6.6) as
// a pre-check ahead of domain.Register: a promoted entry does not consume a
// Registration row (so Register's own active-registration count alone can't
// see it), but its promoted player is meant to have first claim on the slot
// until PromotionResponseWindow elapses — a *different* player's direct
// Register attempt during that window must not be able to take it instead.
// The promoted player themself is deliberately excluded from this check (it
// reports false for them) — their own Register call is how a promotion gets
// confirmed, and Register's normal capacity/double-registration rules still
// apply to it unchanged.
func SlotReservedByPromotion(entries []WaitlistEntry, playerID string, now time.Time) bool {
	for _, e := range entries {
		if e.Status != WaitlistStatusPromoted {
			continue
		}
		if e.PlayerID == playerID {
			continue
		}
		if e.HasExpired(now) {
			continue
		}
		return true
	}
	return false
}

// HasExpired reports whether a promoted entry's PromotionResponseWindow has
// elapsed as of now. Always false for an entry that isn't currently
// promoted (a waiting/expired/cancelled entry has no active response
// window to expire).
func (w *WaitlistEntry) HasExpired(now time.Time) bool {
	if w.Status != WaitlistStatusPromoted {
		return false
	}
	return !now.Before(w.PromotedAt.Add(PromotionResponseWindow))
}
