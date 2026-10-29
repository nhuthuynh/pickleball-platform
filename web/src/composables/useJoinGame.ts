import { ref, type Ref } from 'vue'
import { socialplayClient, type SocialPlayClient } from '../api/socialplayClient'
import {
  mapToRegistration,
  mapToWaitlistEntry,
  type ConfirmedRegistration,
  type ConfirmedWaitlistEntry,
} from '../models/game'

// The server's own conflict messages, matched to tell apart the two 409s
// this screen must report differently (#345).
//
// These are a contract, not prose:
// internal/socialplay/adapter/grpcapi/conflict_message_contract_test.go
// asserts RegisterForGame returns exactly these strings and fails if either
// is reworded — because a reword would silently return this UI to the wrong
// message, silently since every fixture in the web suite would still match
// itself.
const CONFLICT_GAME_FULL = 'game is at capacity'
const CONFLICT_ALREADY_REGISTERED = 'already registered for this game'

// There is no JWT/session layer yet (HANDOFF.md cross-cutting Auth
// backlog) — `RegisterForGameRequest.player_id`/`JoinWaitlistRequest.
// player_id` are necessarily just request fields a caller supplies, the
// same known gap `proto/pickleball/socialplay/v1/socialplay.proto`'s own
// doc comments document. Mirrors `FacilityOnboarding.vue`'s identical
// `MOCK_OWNER_ID` placeholder for the Host side of the same gap.
export const MOCK_PLAYER_ID = 'player-mock-1'

export interface UseJoinGameResult {
  /** Number of guests the Player is bringing (T8.9's guest-count stepper).
   * Bounding this to the Game's `GuestAllowance` is the caller's job (a
   * UX nicety only — see `incrementGuests`'s doc comment); the server-side
   * check via `domain.Register`'s weighted capacity invariant remains
   * authoritative regardless of what this value is set to. */
  guestCount: Ref<number>
  registering: Ref<boolean>
  /** Human-readable message for a `RegisterForGame` failure that is *not*
   * the capacity conflict (that has its own `gameFull` state below). */
  registerError: Ref<string | null>
  /** Set when `RegisterForGame` comes back 409 (mapped `ErrGameFull`) — a
   * specific, actionable message (WCAG 3.3.3 Error Suggestion) plus the
   * signal to offer `JoinWaitlist` as the next action. */
  gameFull: Ref<boolean>
  /** Set when the 409 was ErrAlreadyRegistered — this player already holds
   *  a registration for this Game (#345). Distinct from `gameFull` because
   *  the remedy is different: there is nothing to do, and offering a
   *  waitlist for a game you are already in is nonsense. */
  alreadyRegistered: Ref<boolean>
  confirmedRegistration: Ref<ConfirmedRegistration | null>
  joiningWaitlist: Ref<boolean>
  waitlistError: Ref<string | null>
  confirmedWaitlistEntry: Ref<ConfirmedWaitlistEntry | null>
  /** Increments `guestCount`, but never past `max` (the Game's
   * `GuestAllowance`) — a client-side UX nicety only, mirroring T8.6's
   * domain rule; `domain.Register`'s own guest-allowance/capacity checks
   * remain the authoritative guard server-side regardless of what this
   * stepper allows. */
  incrementGuests: (max: number) => void
  decrementGuests: () => void
  register: (playerId: string) => Promise<void>
  joinWaitlist: (playerId: string) => Promise<void>
  reset: () => void
}

/**
 * Drives the join-a-game flow (T8.9): a guest-count stepper plus
 * `RegisterForGame`, and — only once a capacity conflict is hit —
 * `JoinWaitlist` as the offered next action. `client` is injectable
 * (defaults to the real `socialplayClient`), same pattern as
 * `useCourtBooking`/`useFacilityList`.
 */
export function useJoinGame(gameId: string, client: SocialPlayClient = socialplayClient): UseJoinGameResult {
  const guestCount = ref(0)
  const registering = ref(false)
  const registerError = ref<string | null>(null)
  const gameFull = ref(false)
  const alreadyRegistered = ref(false)
  const confirmedRegistration = ref<ConfirmedRegistration | null>(null)
  const joiningWaitlist = ref(false)
  const waitlistError = ref<string | null>(null)
  const confirmedWaitlistEntry = ref<ConfirmedWaitlistEntry | null>(null)

  function incrementGuests(max: number): void {
    if (guestCount.value < max) {
      guestCount.value += 1
    }
  }

  function decrementGuests(): void {
    if (guestCount.value > 0) {
      guestCount.value -= 1
    }
  }

  async function register(playerId: string): Promise<void> {
    registering.value = true
    registerError.value = null
    gameFull.value = false
    alreadyRegistered.value = false
    try {
      const { data, error, response } = await client.POST('/v1/games/{gameId}/registrations', {
        params: { path: { gameId } },
        body: { playerId, guestCount: guestCount.value },
      })
      if (response.status === 409) {
        // A 409 is not necessarily "full" (#345). Five sentinels share
        // AlreadyExists on registration paths — ErrGameFull,
        // ErrAlreadyRegistered, ErrCourtUnavailable, ErrAlreadyOnWaitlist,
        // ErrAlreadyGameAdmin — and that grouping is deliberate and
        // documented in the handler's toStatus, so the status alone cannot
        // say which happened. This used to report every one of them as
        // "This game just filled up" and offer a waitlist, which told a
        // player who was already registered to wait for a game they were
        // already in.
        //
        // So the message decides. It is the server's own, pinned on the Go
        // side by conflict_message_contract_test.go, which fails if either
        // string is reworded without this matcher changing too.
        const serverMessage = (error as { message?: string } | undefined)?.message ?? ''

        if (serverMessage.includes(CONFLICT_ALREADY_REGISTERED)) {
          alreadyRegistered.value = true
          registerError.value = 'You are already registered for this game.'
          return
        }
        if (serverMessage.includes(CONFLICT_GAME_FULL)) {
          // Specific, actionable message (WCAG 3.3.3 Error Suggestion) —
          // the waitlist offer itself is rendered by the caller when
          // gameFull is true, joinWaitlist below is what it calls.
          gameFull.value = true
          registerError.value = 'This game just filled up.'
          return
        }
        // A conflict this client has no specific message for. Saying
        // "filled up" here would claim a cause we did not establish, which
        // is the defect this block exists to fix — so say only what is
        // true, and offer no waitlist.
        registerError.value = 'Could not complete this registration.'
        return
      }
      if (error || !data?.registration) {
        registerError.value = 'Could not complete this registration. Please try again.'
        return
      }
      confirmedRegistration.value = mapToRegistration(data.registration)
    } catch {
      registerError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      registering.value = false
    }
  }

  async function joinWaitlist(playerId: string): Promise<void> {
    joiningWaitlist.value = true
    waitlistError.value = null
    try {
      const { data, error } = await client.POST('/v1/games/{gameId}/waitlist', {
        params: { path: { gameId } },
        body: { playerId },
      })
      if (error || !data?.entry) {
        waitlistError.value = 'Could not join the waitlist. Please try again.'
        return
      }
      confirmedWaitlistEntry.value = mapToWaitlistEntry(data.entry)
    } catch {
      waitlistError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      joiningWaitlist.value = false
    }
  }

  function reset(): void {
    guestCount.value = 0
    registering.value = false
    registerError.value = null
    gameFull.value = false
    confirmedRegistration.value = null
    joiningWaitlist.value = false
    waitlistError.value = null
    confirmedWaitlistEntry.value = null
  }

  return {
    guestCount,
    registering,
    registerError,
    gameFull,
    alreadyRegistered,
    confirmedRegistration,
    joiningWaitlist,
    waitlistError,
    confirmedWaitlistEntry,
    incrementGuests,
    decrementGuests,
    register,
    joinWaitlist,
    reset,
  }
}
