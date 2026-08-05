import { ref, type Ref } from 'vue'
import { competitionsClient, type CompetitionsClient } from '../api/competitionsClient'
import { mapToCompetitionEntry, type ConfirmedEntry, type EntrySource } from '../models/competition'
// Reused rather than redeclared: there is still no JWT/session layer
// (HANDOFF.md's open cross-cutting Auth item), so `player_id` is a request
// field a caller supplies. One mock identity across Games and Competitions
// keeps the gap visible in one place instead of two.
import { MOCK_PLAYER_ID } from './useJoinGame'

export { MOCK_PLAYER_ID }

export interface UseEnterCompetitionResult {
  /** Guests the entrant is bringing. Bounding this to the Competition's
   * `GuestAllowance` is a CLIENT-SIDE UX NICETY ONLY — the server's own
   * guest-allowance and weighted-capacity checks (`domain.Enter`, plus the
   * DB-level trigger in db/migrations/0014_competitions.sql) remain
   * authoritative regardless of what this stepper permits. */
  guestCount: Ref<number>
  entering: Ref<boolean>
  /** Human-readable message for the most recent failure, whichever kind. */
  enterError: Ref<string | null>
  /** The Competition is at capacity. */
  competitionFull: Ref<boolean>
  /** This player already holds an entry. */
  alreadyEntered: Ref<boolean>
  confirmedEntry: Ref<ConfirmedEntry | null>
  /** Increments `guestCount`, never past `max`. */
  incrementGuests: (max: number) => void
  decrementGuests: () => void
  enter: (playerId: string) => Promise<void>
  reset: () => void
}

/**
 * Reads the human-readable message out of a grpc-gateway error body. The
 * gateway serialises a `status.Error` as `{"code":…, "message":…}`, and
 * openapi-fetch hands that parsed body back as `error`.
 */
function messageOf(apiError: unknown): string {
  if (apiError && typeof apiError === 'object' && 'message' in apiError) {
    return String((apiError as { message?: unknown }).message ?? '')
  }
  return ''
}

/**
 * Drives the enter-a-competition flow (T9.7 requirement #1): a guest-count
 * stepper plus `EnterCompetition`, carrying the `source` attribution for how
 * this entrant arrived.
 *
 * `source` is supplied by the CALLER, not inferred here: the whole point of
 * the field is that `/c/:shareToken` and `/competitions` are different
 * arrival paths, and only the screen knows which one it is. The server
 * validates the claimed value against a closed enum
 * (proto/pickleball/competitions/v1/competitions.proto's `EntrySource`).
 *
 * **There is deliberately no waitlist here.** Social Play's waitlist is
 * Game-scoped (T6.6 / ADR-0006) and belongs to a different bounded context;
 * T9 gives Competitions none, so a full Competition offers browsing rather
 * than a queue. Its absence is a decision, not an omission.
 */
export function useEnterCompetition(
  competitionId: string,
  source: EntrySource,
  client: CompetitionsClient = competitionsClient,
): UseEnterCompetitionResult {
  const guestCount = ref(0)
  const entering = ref(false)
  const enterError = ref<string | null>(null)
  const competitionFull = ref(false)
  const alreadyEntered = ref(false)
  const confirmedEntry = ref<ConfirmedEntry | null>(null)

  function incrementGuests(max: number): void {
    if (guestCount.value < max) guestCount.value += 1
  }

  function decrementGuests(): void {
    if (guestCount.value > 0) guestCount.value -= 1
  }

  /**
   * Turns a rejection into a specific, actionable message (WCAG 3.3.3 Error
   * Suggestion — the same rule T7.6 and T8.9 applied).
   *
   * The status code alone is NOT enough here, and that is the reason this
   * function exists: `internal/competitions/adapter/grpcapi`'s `toStatus`
   * maps BOTH `ErrCompetitionFull` and `ErrAlreadyEntered` to
   * `codes.AlreadyExists`, so both arrive as HTTP 409. Telling an entrant
   * who already has a place that "this competition just filled up" would be
   * both wrong and unactionable, so the message is read too. (`toStatus`
   * moved `ErrCompetitionFull` from FailedPrecondition to AlreadyExists
   * during T9.4's review — checked against the current handler, not assumed
   * from the proto comment, which still says "409-shaped
   * FAILED_PRECONDITION".)
   */
  function classifyFailure(status: number, message: string): void {
    const text = message.toLowerCase()

    if (text.includes('already entered')) {
      alreadyEntered.value = true
      enterError.value = "You've already entered this competition — there's nothing more to do."
      return
    }
    if (status === 409 || text.includes('at capacity')) {
      competitionFull.value = true
      enterError.value = 'This competition just filled up. Browse other competitions to find another one.'
      return
    }
    if (text.includes('cancelled')) {
      enterError.value = 'This competition has been cancelled, so entries are closed.'
      return
    }
    if (text.includes('guest allowance')) {
      enterError.value = "That's more guests than this competition allows. Lower the guest count and try again."
      return
    }
    enterError.value = 'Could not complete this entry. Please try again.'
  }

  async function enter(playerId: string): Promise<void> {
    entering.value = true
    enterError.value = null
    competitionFull.value = false
    alreadyEntered.value = false
    try {
      const { data, error: apiError, response } = await client.POST(
        '/v1/competitions/{competitionId}/entries',
        {
          params: { path: { competitionId } },
          body: { playerId, guestCount: guestCount.value, source },
        },
      )
      if (apiError || !data?.entry) {
        classifyFailure(response?.status ?? 0, messageOf(apiError))
        return
      }
      confirmedEntry.value = mapToCompetitionEntry(data.entry)
    } catch {
      enterError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      entering.value = false
    }
  }

  function reset(): void {
    guestCount.value = 0
    entering.value = false
    enterError.value = null
    competitionFull.value = false
    alreadyEntered.value = false
    confirmedEntry.value = null
  }

  return {
    guestCount,
    entering,
    enterError,
    competitionFull,
    alreadyEntered,
    confirmedEntry,
    incrementGuests,
    decrementGuests,
    enter,
    reset,
  }
}
