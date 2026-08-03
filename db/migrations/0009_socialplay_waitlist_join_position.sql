-- Waitlist queue-position race guard (T6.6-loop-2, PM+PE review finding on
-- PR #25): 0008_socialplay_waitlist_promotion.sql's own race analysis was
-- required by name to be repeated for queue-POSITION assignment too
-- (docs/process/t6-sprint-plan.md's T6.6 section, referencing
-- docs/LESSONS.md's T5 capacity-invariant lesson) — it wasn't done in loop 1.
-- This migration is that fix. Summarized here since it's the fix and should
-- be self-explanatory on its own, same discipline as 0006/0008's comments.
--
-- THE RACE: domain.JoinWaitlist computed Position as
-- len(existing non-cancelled entries) + 1 from an app-layer
-- WaitlistRepository.ListForGame read, with no lock held between that read
-- and the later WaitlistRepository.Create insert. Two concurrent
-- JoinWaitlist calls for the SAME game_id can both read the waitlist in the
-- same pre-insert state and both compute the same Position — reproduced
-- directly against a real local Postgres: 30 concurrent JoinWaitlist calls
-- for one Game produced 27 entries all sharing Position 1 (see the PR
-- description for the full "before" run log, two separate cold-start runs).
--
-- DISTINCTNESS VS ORDERING (the question docs/LESSONS.md's T5 entry asks by
-- name for every new count/order-limited invariant, same question 0008
-- already answered for the promotion trigger):
--
-- This is an ORDERING race, not a plain distinctness race, and a bare
-- UNIQUE(game_id, position) constraint is NOT the right fix on its own.
-- A unique index would correctly reject the SECOND of two colliding inserts
-- (turning silent data corruption into a visible 23505 conflict), but it
-- does not compute the CORRECT position for the rejected caller — the
-- rejected player's JoinWaitlist call would simply fail outright, even
-- though both players' joins are entirely legitimate (the game is full for
-- both, and both are allowed to queue). Forcing one of two valid concurrent
-- joiners to see an error rather than land at the correct next Position is
-- materially worse UX than the "block briefly, then get correctly
-- sequenced" behavior a lock buys — the same distinction 0008 already drew
-- for the promotion trigger: distinctness constraints answer "is this value
-- different from every other row's", ordering invariants answer "is this
-- value the single correct next one given everything that has already
-- committed", and only a lock that is held across the read-then-write can
-- answer the second question correctly.
--
-- NOT adding a UNIQUE(game_id, position) backstop here (unlike 0007's
-- active-player-per-game index, which IS a genuine distinctness invariant
-- and correctly has one): the existing Position formula (count of
-- non-cancelled entries + 1, unchanged by this migration — see below) is
-- only guaranteed collision-free against OTHER STILL-ACTIVE entries when no
-- entry has been cancelled out of the tally in between. A join that lands
-- after an earlier, lower-Position entry has been cancelled can legitimately
-- recompute a Position that matches an existing still-active entry's
-- Position (e.g. entries at Position 1 (waiting) and 2 (waiting), Position
-- 1 cancels, the next joiner's count-of-non-cancelled is now 1, landing them
-- back on Position 2 too) — a pre-existing quirk of the count-based formula
-- itself, not something concurrency introduces or that this migration's
-- locking fixes. This is a real, single-threaded, non-concurrency finding
-- surfaced while doing this race analysis; it is deliberately NOT fixed in
-- this migration (out of scope for a "close the concurrency race" fix — it
-- would mean changing the Position formula's semantics, which
-- domain.JoinWaitlist's own doc comment and
-- TestJoinWaitlist_PositionCountsNonCancelledEntries currently lock in as
-- intentional) — flagged instead for a follow-up ticket. Adding a bare
-- UNIQUE(game_id, position) constraint today would turn that pre-existing
-- quirk into a hard 23505 failure for an otherwise legitimate join, which
-- would be a worse regression than the quirk itself.
--
-- THE FIX, mirroring 0008's FOR UPDATE-locking function pattern exactly:
-- wrap "lock the game, count current non-cancelled entries, insert at the
-- next Position" as ONE atomic Postgres function holding a FOR UPDATE lock
-- on the owning games row for its whole duration — the SAME lock
-- promote_next_waiting and enforce_game_capacity already take, so a
-- JoinWaitlist call, a PromoteNext call, and a capacity check for the same
-- Game are all serialized against each other, not just against each other's
-- own kind. A second concurrent join_waitlist_entry call for the SAME
-- game_id blocks until the first call's insert commits, then counts against
-- fresh state and gets the correct next Position. The Position formula
-- itself (count of non-cancelled entries + 1) is unchanged from
-- domain.JoinWaitlist's — see that function's doc comment
-- (internal/socialplay/domain/waitlist.go) — this migration only makes its
-- computation atomic, it does not change what value gets computed for any
-- given fixed state.
--
-- REQUIRED CONCURRENCY PROOF (CLAUDE.md rule 10): see
-- internal/socialplay/adapter/postgres/waitlist_position_concurrency_integration_test.go
-- and the PR description's run log — N concurrent JoinWaitlist calls
-- against the same full Game, run multiple times including cold starts,
-- asserting the resulting positions are exactly 1..N with no collisions and
-- no gaps.

CREATE FUNCTION join_waitlist_entry(p_id uuid, p_game_id uuid, p_player_id text)
RETURNS SETOF waitlist_entries AS $$
DECLARE
    v_position integer;
    v_entry    waitlist_entries%ROWTYPE;
BEGIN
    -- The actual race-closing lock: every concurrent join for this
    -- game_id (and every promote_next_waiting / enforce_game_capacity call
    -- for it too, since they lock the same row) serializes here, so the
    -- count below always runs against a state that already reflects every
    -- earlier-committed join/promotion for this game, never a stale one.
    PERFORM 1 FROM games WHERE id = p_game_id FOR UPDATE;

    -- Mirrors domain.JoinWaitlist's Position formula exactly: every
    -- non-cancelled entry counts (waiting, promoted, AND expired — only a
    -- voluntary cancel is excluded from the tally), the new entry lands one
    -- past that count.
    SELECT count(*) INTO v_position
    FROM waitlist_entries
    WHERE game_id = p_game_id
      AND status <> 'cancelled';

    INSERT INTO waitlist_entries (id, game_id, player_id, position, status)
    VALUES (p_id, p_game_id, p_player_id, v_position + 1, 'waiting')
    RETURNING * INTO v_entry;

    RETURN NEXT v_entry;
END;
$$ LANGUAGE plpgsql;
