-- Waitlist promotion race guard (T6.6). See the PR description for the
-- full race analysis; summarized here since this migration is the fix for
-- it and should be self-explanatory on its own, same discipline as
-- 0006_socialplay_capacity_guard.sql's comment.
--
-- THE RACE, AND WHY IT'S ORDERING-SHAPED, NOT DISTINCTNESS-SHAPED
-- (docs/LESSONS.md's T5 capacity-invariant lesson asks this question
-- explicitly for every new count/order-limited invariant):
--
-- app.Service's auto-promotion orchestration (CancelRegistration,
-- ExpireWaitlistPromotion) is, at the app layer, a "list the game's
-- waitlist entries, pick the oldest still-waiting one, transition it to
-- promoted" read-then-write sequence -- the exact shape LESSONS.md's T5
-- entry already flagged as *not* an invariant under concurrency ("a
-- domain-only count/order then act check... has a TOCTOU window the moment
-- two real requests interleave"). Two concurrent triggers for the SAME game
-- (e.g. two cancellations racing, or a cancellation racing a timeout-expiry
-- sweep) can both read the waitlist in its pre-promotion state, both
-- identify the SAME entry as "the oldest waiting one", and — if promotion
-- were implemented as an unconditional "list then UPDATE by id" pair of
-- queries — both attempt to promote it.
--
-- This is NOT closed by a unique constraint the way the no-double-
-- registration/no-double-waitlisting invariants are (those are genuine
-- *distinctness* races: two rows racing to claim the same (game_id,
-- player_id) key, which a unique index resolves for free by rejecting the
-- second INSERT). Promotion is a *counting/ordering* race: the question
-- being raced on is "which single row, among a set that can change size and
-- order as concurrent transactions commit, is 'next'?" -- ordering
-- information that a plain UPDATE...WHERE targeting one already-decided id
-- cannot re-derive after the fact. A targeted UPDATE's own row lock is
-- sufficient if every concurrent caller happens to target the literal same
-- row (Postgres would correctly serialize that and the loser's WHERE
-- status='waiting' clause would fail after the winner commits) — but that
-- only holds in the trivial single-entry case. With two or more waiting
-- entries, two concurrent callers picking "the oldest" via independent,
-- unlocked SELECTs can both land on entry #1 while entry #2 (which should
-- receive the SECOND freed slot) never gets promoted at all — an
-- under-promotion bug that would silently strand a paying player behind an
-- empty slot, exactly the kind of gap ADR-0006's response-timeout
-- requirement exists to prevent, just one step earlier in the pipeline.
--
-- THE FIX, mirroring 0006's FOR UPDATE-locking trigger pattern: wrap the
-- entire "lock the game, pick the true oldest waiting entry, promote it" as
-- ONE atomic Postgres function, holding a FOR UPDATE lock on the owning
-- games row for the function's whole duration. A second concurrent call for
-- the SAME game_id blocks on that lock until the first call's transaction
-- commits, and only then runs its own SELECT ... ORDER BY position LIMIT 1
-- -- by which point it sees the first call's promotion already committed,
-- so it correctly picks the NEXT still-waiting entry (or finds none). This
-- is what actually closes the race, the same way 0006's games-row lock is
-- what serializes concurrent registrations, not the count check that comes
-- after it.
--
-- REQUIRED CONCURRENCY PROOF (CLAUDE.md rule 10): see
-- internal/socialplay/adapter/postgres/waitlist_promotion_concurrency_integration_test.go
-- and the PR description's run log — two simultaneous CancelRegistration
-- calls against a capacity-1 Game with one waitlist entry, run multiple
-- times including a cold start, asserting exactly one promotion.

CREATE FUNCTION promote_next_waiting(p_game_id uuid, p_now timestamptz)
RETURNS SETOF waitlist_entries AS $$
DECLARE
    v_entry waitlist_entries%ROWTYPE;
BEGIN
    -- The actual race-closing lock: every concurrent promotion attempt for
    -- this game_id serializes here, so the SELECT below always runs against
    -- a state that already reflects every earlier-committed promotion for
    -- this game, never a stale/concurrent one.
    PERFORM 1 FROM games WHERE id = p_game_id FOR UPDATE;

    -- Belt-and-braces row lock on the entry itself too, even though the
    -- games-row lock above already serializes every caller that matters.
    SELECT * INTO v_entry
    FROM waitlist_entries
    WHERE game_id = p_game_id
      AND status = 'waiting'
    ORDER BY position ASC
    LIMIT 1
    FOR UPDATE;

    IF NOT FOUND THEN
        -- No one is waiting -- an empty result set, not an error. The
        -- caller (internal/socialplay/adapter/postgres) translates a
        -- zero-row result into domain.ErrNoWaitingEntries.
        RETURN;
    END IF;

    UPDATE waitlist_entries
    SET status = 'promoted', promoted_at = p_now
    WHERE id = v_entry.id
    RETURNING * INTO v_entry;

    RETURN NEXT v_entry;
END;
$$ LANGUAGE plpgsql;
