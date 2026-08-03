<script setup lang="ts">
// Online checkout flow (T8.10, docs/process/t8-sprint-plan.md), mounted at
// /games/:id/checkout (replacing T8.1's placeholder route) — the Stripe-stub
// half of T8.10's Payments UI, reached from GameJoinPanel.vue's "Pay online
// now" button via DiscoverGames.vue's router push (see that file's
// `onPayOnline`). `:id` is the Game id; the Registration id (the actual
// Payment `payableId`) travels as `?registrationId=` since a Payment is
// keyed to a Registration, not a Game — see router/index.ts's route
// comment.
//
// Game context (host/court/time, for the review step) is fetched via
// `ListGames` and matched by id — same "no separate GetGame fetch" gap
// GameDetailPanel.vue's header comment already documents (Social Play has
// no per-Game detail RPC); this view accepts the same limitation rather
// than inventing a new one just for this screen.
//
// **Disclosed gap: the checkout amount is a flat placeholder, not a real
// price** — see models/payment.ts's header comment for the full
// disclosure (Social Play's Game/Registration has no price/fee field at
// all, despite the design handoff's Flow 4/6 calling for one). The review
// step below labels it plainly as a placeholder rather than presenting it
// as a real charge.
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useGameList } from '../composables/useGameList'
import { useGamePayment } from '../composables/useGamePayment'
import { formatGameRange } from '../models/game'
import { PLACEHOLDER_REGISTRATION_FEE_CENTS, formatMoneyCents } from '../models/payment'
import type { SocialPlayClient } from '../api/socialplayClient'
import type { PaymentsClient } from '../api/paymentsClient'

const props = defineProps<{
  /** Injectable for tests; defaults to the real socialplayClient. */
  client?: SocialPlayClient
  /** Injectable for tests; defaults to the real paymentsClient. */
  paymentsClient?: PaymentsClient
}>()

const route = useRoute()
const gameId = computed(() => String(route.params.id ?? ''))
const registrationId = computed(() => {
  const raw = route.query.registrationId
  return typeof raw === 'string' ? raw : ''
})

const { games, search } = useGameList(props.client)
const game = computed(() => games.value.find((g) => g.id === gameId.value) ?? null)

const { step, payment, createError, confirming, confirmError, confirmedPayment, startCheckout, confirmPayment } =
  useGamePayment(props.paymentsClient)

onMounted(() => {
  void search()
  if (registrationId.value) {
    void startCheckout(registrationId.value, PLACEHOLDER_REGISTRATION_FEE_CENTS)
  }
})
</script>

<template>
  <section class="game-checkout" aria-label="Checkout">
    <h1 class="game-checkout__heading">Checkout</h1>

    <p v-if="!registrationId" class="game-checkout__status game-checkout__status--error" role="alert">
      Missing registration — start from the Games list and join a game first.
    </p>

    <template v-else>
      <p v-if="step === 'preparing' && !createError" class="game-checkout__status" role="status">
        Preparing checkout…
      </p>

      <p v-if="createError" class="game-checkout__status game-checkout__status--error" role="alert">
        {{ createError }}
      </p>

      <!-- REVIEW/CONFIRM STEP (WCAG 3.3.4 Error Prevention, same pattern
           CourtBookingFlow.vue already uses for CreateBooking): ConfirmOnlinePayment
           can never fire before this step is on screen — useGamePayment's
           own confirm-step gate enforces that regardless of this template. -->
      <div v-if="step === 'review' && payment" class="game-checkout__review">
        <h2 class="game-checkout__review-heading">Review your payment</h2>
        <dl class="game-checkout__summary">
          <div v-if="game" class="game-checkout__summary-row">
            <dt>Game</dt>
            <dd>{{ formatGameRange(game.startsAt, game.endsAt) }}</dd>
          </div>
          <div class="game-checkout__summary-row">
            <dt>Amount</dt>
            <dd>
              {{ formatMoneyCents(payment.amountCents) }}
              <span class="game-checkout__note">(flat placeholder rate — no per-game price yet)</span>
            </dd>
          </div>
        </dl>

        <p v-if="confirmError" class="game-checkout__status game-checkout__status--error" role="alert">
          {{ confirmError }}
        </p>

        <!-- Stripe-stub: no card-shaped input field anywhere (CLAUDE.md
             rule 11 / PCI guardrail) — a stub confirm button is the entire
             "payment form". -->
        <button type="button" class="game-checkout__primary" :disabled="confirming" @click="confirmPayment">
          {{ confirming ? 'Confirming…' : 'Confirm payment (stub)' }}
        </button>
      </div>

      <!-- SUCCESS: ARIA live region (WCAG 4.1.3) -->
      <div
        v-else-if="step === 'success' && confirmedPayment"
        class="game-checkout__success"
        role="status"
        aria-live="polite"
      >
        <p>Payment confirmed. Reference: {{ confirmedPayment.id }}.</p>
      </div>
    </template>
  </section>
</template>

<style scoped>
.game-checkout {
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 480px;
  margin: 0 auto;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.game-checkout__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.game-checkout__status {
  color: var(--ink-soft);
}

.game-checkout__status--error {
  color: var(--ink-warning);
}

.game-checkout__review {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  padding: 1rem;
}

.game-checkout__review-heading {
  font-size: var(--font-size-base);
  margin: 0;
}

.game-checkout__summary {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.game-checkout__summary-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.game-checkout__summary-row dt {
  color: var(--ink-soft);
}

.game-checkout__summary-row dd {
  margin: 0;
  font-weight: 600;
  text-align: right;
}

.game-checkout__note {
  display: block;
  font-weight: 400;
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.game-checkout__primary {
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

.game-checkout__primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.game-checkout__success {
  color: var(--ink-success);
}

/* iPad/web: a bit more breathing room, mirroring CourtBookingFlow.vue's
   identical wider-not-restructured approach at these breakpoints. */
@media (min-width: 768px) {
  .game-checkout {
    max-width: 560px;
    padding: 2rem 1.5rem;
  }
}
</style>
