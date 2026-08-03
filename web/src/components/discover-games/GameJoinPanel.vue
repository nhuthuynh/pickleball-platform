<script setup lang="ts">
// Join-a-game flow (T8.9 requirement #2): a guest-count stepper (bounded by
// the Game's GuestAllowance as a client-side UX nicety — domain.Register's
// weighted capacity invariant is the authoritative server-side check),
// RegisterForGame, and — only on a capacity conflict (mapped ErrGameFull,
// 409) — a specific "This game just filled up" message plus a JoinWaitlist
// action (WCAG 3.3.3 Error Suggestion). Owns its own network calls (like
// CourtBookingFlow.vue owns useCourtBooking) rather than being purely
// presentational, since the join flow itself is the network-calling half
// of this screen — GameDetailPanel.vue (this component's parent) stays
// presentational for the read-only Game fields around it.
//
// Keyed by `game.id` at the call site (GameDetailPanel.vue) so switching
// the selected Game always mounts a fresh instance — no stale guest count
// or leftover success/error state from a previously viewed Game.
import { computed } from 'vue'
import { useJoinGame, MOCK_PLAYER_ID } from '../../composables/useJoinGame'
import type { GameSummary } from '../../models/game'
import type { SocialPlayClient } from '../../api/socialplayClient'

const props = defineProps<{
  game: GameSummary
  /** Injectable for tests; defaults to the real socialplayClient. */
  client?: SocialPlayClient
  /**
   * When true, the Game's already-known SpotsLeft (from the last ListGames
   * fetch) is 0 — skip straight to the "this game is full, join the
   * waitlist" state instead of rendering a guest-count/Join form that
   * would predictably fail. This is a UX nicety only, same caveat as
   * GuestAllowance-bounded stepper: SpotsLeft is a snapshot, so it can be
   * stale in *either* direction (a spot may have freed up, or the Game may
   * have filled further) — a genuine race is still handled by the same
   * 409 -> gameFull path this prop just pre-seeds, not bypassed by it.
   */
  startFull?: boolean
}>()

const {
  guestCount,
  registering,
  registerError,
  gameFull,
  confirmedRegistration,
  joiningWaitlist,
  waitlistError,
  confirmedWaitlistEntry,
  incrementGuests,
  decrementGuests,
  register,
  joinWaitlist,
} = useJoinGame(props.game.id, props.client)

if (props.startFull) {
  gameFull.value = true
  registerError.value = 'This game just filled up.'
}

const canIncrement = computed(() => guestCount.value < props.game.guestAllowance)
const canDecrement = computed(() => guestCount.value > 0)

function onIncrement(): void {
  incrementGuests(props.game.guestAllowance)
}

function onJoin(): void {
  void register(MOCK_PLAYER_ID)
}

function onJoinWaitlist(): void {
  void joinWaitlist(MOCK_PLAYER_ID)
}
</script>

<template>
  <section class="game-join" aria-label="Join this game">
    <!-- SUCCESS: registration confirmed (ARIA live region, WCAG 4.1.3) -->
    <div
      v-if="confirmedRegistration"
      class="game-join__success"
      role="status"
      aria-live="polite"
    >
      <p>
        You're in! Registered with {{ confirmedRegistration.guestCount }}
        {{ confirmedRegistration.guestCount === 1 ? 'guest' : 'guests' }}.
      </p>
    </div>

    <!-- WAITLIST SUCCESS -->
    <div
      v-else-if="confirmedWaitlistEntry"
      class="game-join__success"
      role="status"
      aria-live="polite"
    >
      <p>You're on the waitlist at position {{ confirmedWaitlistEntry.position }}.</p>
    </div>

    <!-- GAME FULL: specific, actionable message + waitlist offer (WCAG
         3.3.3 Error Suggestion) -->
    <div v-else-if="gameFull" class="game-join__conflict" role="alert">
      <p class="game-join__conflict-message">{{ registerError }}</p>
      <p v-if="waitlistError" class="game-join__status game-join__status--error">{{ waitlistError }}</p>
      <button
        type="button"
        class="game-join__primary"
        :disabled="joiningWaitlist"
        @click="onJoinWaitlist"
      >
        {{ joiningWaitlist ? 'Joining waitlist…' : 'Join waitlist' }}
      </button>
    </div>

    <!-- JOIN FORM: guest-count stepper + Join -->
    <form v-else class="game-join__form" @submit.prevent="onJoin">
      <div class="game-join__stepper">
        <span class="game-join__stepper-label" id="guest-stepper-label">Guests</span>
        <div class="game-join__stepper-controls" role="group" aria-labelledby="guest-stepper-label">
          <button
            type="button"
            class="game-join__stepper-button"
            aria-label="Remove a guest"
            :disabled="!canDecrement"
            @click="decrementGuests"
          >
            −
          </button>
          <span class="game-join__stepper-count" aria-live="polite">{{ guestCount }}</span>
          <button
            type="button"
            class="game-join__stepper-button"
            aria-label="Add a guest"
            :disabled="!canIncrement"
            @click="onIncrement"
          >
            +
          </button>
        </div>
        <span class="game-join__stepper-hint">Up to {{ game.guestAllowance }} guests allowed</span>
      </div>

      <p v-if="registerError" class="game-join__status game-join__status--error" role="alert">
        {{ registerError }}
      </p>

      <button type="submit" class="game-join__primary" :disabled="registering">
        {{ registering ? 'Joining…' : 'Join game' }}
      </button>
    </form>
  </section>
</template>

<style scoped>
.game-join {
  font-family: var(--font-family-ui);
  color: var(--ink);
  border-top: 1px solid var(--hs-border);
  margin-top: 1rem;
  padding-top: 1rem;
}

.game-join__form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.game-join__stepper {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.game-join__stepper-label {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.game-join__stepper-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.game-join__stepper-button {
  font: inherit;
  min-height: 44px;
  min-width: 44px;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--court);
  cursor: pointer;
  font-size: var(--font-size-lg);
  line-height: 1;
}

.game-join__stepper-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.game-join__stepper-count {
  min-width: 2ch;
  text-align: center;
  font-weight: 600;
}

.game-join__stepper-hint {
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.game-join__status {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.game-join__status--error {
  color: var(--ink-warning);
}

.game-join__primary {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
  align-self: flex-start;
}

.game-join__primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.game-join__conflict {
  border: 1px solid var(--pill-critical-bg);
  background: var(--pill-critical-bg);
  color: var(--ink);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.game-join__conflict-message {
  font-weight: 600;
  margin: 0;
}

.game-join__success {
  color: var(--ink-success);
}
</style>
