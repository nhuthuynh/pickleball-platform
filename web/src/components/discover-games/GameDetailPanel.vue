<script setup lang="ts">
// Game detail/review view (T8.9 requirement #1): host, court, facility,
// spots left, and a payment-method label (see paymentMethodLabel's doc
// comment for why this is a label, not a price — Payments UI is T8.10).
// Read-only fields are purely presentational, driven by the already-loaded
// `game` prop (ListGames returns everything this view needs — no separate
// GetGame fetch, since Social Play has no per-Game detail RPC and nothing
// here needs more than ListGames already returns). The Join flow itself
// (network-calling) is delegated to GameJoinPanel.vue — see that file's
// header comment for why it isn't purely presentational too.
//
// Host/Facility names (T10.8, closes #98): this panel stays presentational
// for `game`'s own fields, but Host and Facility are resolved from raw ids
// (`hostId`/`venueFacilityId`) via two small network-calling child
// components, DisplayName.vue/VenueName.vue — the same "network-calling
// child of a presentational parent" split GameJoinPanel already uses, just
// applied to a read instead of a write.
import type { GameSummary } from '../../models/game'
import { formatGameRange, paymentMethodLabel, spotsLeftLabel, entryFeeLabel } from '../../models/game'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { IdentityClient } from '../../api/identityClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import GameJoinPanel from './GameJoinPanel.vue'
import DisplayName from '../identity/DisplayName.vue'
import VenueName from '../facilities/VenueName.vue'

defineProps<{
  game: GameSummary | null
  loading: boolean
  error: string | null
  /** Whether a game is currently selected at all (vs. no selection yet —
   * iPad/web only; iPhone never renders this component without a
   * selection) — mirrors FacilityDetailPanel.vue's identical prop. */
  hasSelection: boolean
  /** Injectable for tests; defaults to the real socialplayClient. Passed
   * through to GameJoinPanel. */
  client?: SocialPlayClient
  /** Injectable for tests (T10.8); defaults to the real identityClient.
   * Passed through to DisplayName. */
  identityClient?: IdentityClient
  /** Injectable for tests (T10.8); defaults to the real facilitiesClient.
   * Passed through to VenueName. */
  facilitiesClient?: FacilitiesClient
}>()

const emit = defineEmits<{
  retry: []
  /** Forwarded from GameJoinPanel's identical event (T8.10) — this
   * component stays presentational/router-free, same as its own `retry`
   * event; the caller that actually knows about routing (DiscoverGames.vue)
   * is what turns this into a real navigation. */
  payOnline: [registrationId: string]
}>()
</script>

<template>
  <section class="game-detail" aria-label="Game details">
    <p v-if="!hasSelection" class="game-detail__status game-detail__status--placeholder">
      Select a game to see its details.
    </p>

    <!-- Loading state (T8.9 requirement #3) -->
    <p v-else-if="loading" class="game-detail__status game-detail__status--loading" role="status">
      Loading game details…
    </p>

    <!-- Error state: API unreachable, or a stale selection that no longer
         resolves to a real Game (e.g. it was cancelled/removed since the
         list was last fetched) -->
    <div v-else-if="error" class="game-detail__status game-detail__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="game-detail__retry" @click="emit('retry')">Try again</button>
    </div>

    <div v-else-if="game" class="game-detail__content">
      <h2 class="game-detail__heading">{{ formatGameRange(game.startsAt, game.endsAt) }}</h2>

      <dl class="game-detail__fields">
        <div class="game-detail__field">
          <dt>Host</dt>
          <dd>
            <DisplayName :user-id="game.hostId" fallback="Unknown host" :client="identityClient" />
          </dd>
        </div>
        <div class="game-detail__field">
          <dt>Facility</dt>
          <dd>
            <VenueName :facility-id="game.venueFacilityId" :client="facilitiesClient" />
          </dd>
        </div>
        <div class="game-detail__field">
          <dt>Court{{ game.courtIds.length > 1 ? 's' : '' }}</dt>
          <dd>{{ game.courtIds.join(', ') || 'Not set' }}</dd>
        </div>
        <div class="game-detail__field">
          <!-- WCAG 1.4.1: spots-left is text, never a color-only cue -->
          <dt>Spots left</dt>
          <dd>{{ spotsLeftLabel(game.spotsLeft) }}</dd>
        </div>
        <div class="game-detail__field">
          <!-- WCAG 1.4.1: "Cash at facility" / "Online" / "Online or cash"
               is text, never a color-only cue -->
          <dt>Payment</dt>
          <dd>{{ paymentMethodLabel(game.paymentMethod) }}</dd>
        </div>
        <div class="game-detail__field">
          <!-- Entry fee (T9.2): the Host's real price, replacing T8.10's
               flat placeholder. A free game reads as the word "Free", never
               "$0.00" or a blank — WCAG 1.4.1 (text, not a color cue) and
               NN/g heuristic #2 (a zero price is a real product state). -->
          <dt>Entry fee</dt>
          <dd data-testid="game-detail-entry-fee">{{ entryFeeLabel(game.entryFeeCents) }}</dd>
        </div>
      </dl>

      <GameJoinPanel
        :key="game.id"
        :game="game"
        :client="client"
        :start-full="game.spotsLeft <= 0"
        @pay-online="(id) => emit('payOnline', id)"
      />
    </div>
  </section>
</template>

<style scoped>
.game-detail {
  font-family: var(--font-family-ui);
  color: var(--ink);
}

.game-detail__status {
  color: var(--ink-soft);
}

.game-detail__status--error {
  color: var(--ink-warning);
}

.game-detail__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.game-detail__heading {
  font-size: var(--font-size-2xl);
  margin: 0 0 0.75rem;
  color: var(--court);
}

.game-detail__fields {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.game-detail__field {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.4rem 0;
  border-bottom: 1px solid var(--hs-border);
}

.game-detail__field dt {
  color: var(--ink-soft);
}

.game-detail__field dd {
  margin: 0;
  font-weight: 600;
  text-align: right;
}

.game-detail__full-notice {
  margin-top: 1rem;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

/* Web/iPad: give the field list a bit more breathing room, mirroring
   FacilityDetailPanel.vue's approach of widening rather than restructuring
   at wider breakpoints. */
@media (min-width: 768px) {
  .game-detail__field {
    padding: 0.5rem 0;
  }
}
</style>
