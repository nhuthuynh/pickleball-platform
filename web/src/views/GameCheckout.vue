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
// T9.2: the checkout amount is now the REAL fee the Host set on this Game
// (`Game.EntryFee`), replacing T8.10's flat placeholder rate and the
// "placeholder" label that had to accompany it. Two consequences worth
// stating, because both are real product states rather than edge cases:
//
//   - The amount is read off the Game, so checkout cannot start until the
//     Game has actually loaded. If it can't be found, this view says so
//     instead of falling back to an invented amount — charging a guessed
//     figure is precisely the failure this ticket exists to remove.
//   - A FREE game (entry fee 0) creates no Payment at all: there is
//     nothing to charge, and the Payments domain rightly rejects a
//     zero-amount Payment. The player is told they're already in.
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useGameList } from '../composables/useGameList'
import { useGamePayment } from '../composables/useGamePayment'
import { formatGameRange, entryFeeLabel } from '../models/game'
import { formatMoneyCents, DEFAULT_CURRENCY_CODE } from '../models/payment'
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

/** The FROZEN amount this Registration owes (T56.1, issue #126), carried
 * here from the join flow — see DiscoverGames.vue's `onPayOnline`. `null`
 * means the route did not supply one, which this view treats as "unknown",
 * never as zero. */
const amountOwedCents = computed<number | null>(() => {
  const raw = route.query.amountOwedCents
  if (typeof raw !== 'string' || raw === '') return null
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null
})

const amountOwedCurrency = computed(() => {
  const raw = route.query.amountOwedCurrency
  return typeof raw === 'string' && raw !== '' ? raw : DEFAULT_CURRENCY_CODE
})

const { games, search } = useGameList(props.client)
const game = computed(() => games.value.find((g) => g.id === gameId.value) ?? null)

const { step, payment, createError, confirming, confirmError, confirmedPayment, startCheckout, confirmPayment } =
  useGamePayment(props.paymentsClient)

/** Nothing is owed, so there is nothing to pay and no Payment to create —
 * the Payments domain rightly rejects a zero-amount Payment (T9.2).
 *
 * T56.1: decided by the FROZEN owed amount rather than the Game's entry
 * fee. For a free Game the two agree (zero times any number of heads is
 * zero), so this is not a behaviour change there; it is the right
 * question to ask now that "what is owed" and "what one player costs" are
 * no longer the same number. */
const isFreeGame = computed(() => amountOwedCents.value === 0)

/** The Game loaded but wasn't found, so we can't show what it is. We
 * refuse to invent it — see the file header. */
const gameMissing = ref(false)

/** T56.1/T56.2: the route carried no owed amount, so this view does not
 * know what to charge.
 *
 * It deliberately does NOT fall back to the Game's entry fee. That
 * fallback would be correct only for a player who brought nobody, and
 * silently wrong for everyone else — and since T56.2 (#297) the server
 * compares the amount against the Registration's own record, so "silently
 * wrong" is now "refused with a message about nothing the player did".
 * Saying we can't tell is the honest answer, and it mirrors what this view
 * already does for a Game it couldn't load. */
const amountUnknown = computed(() => registrationId.value !== '' && amountOwedCents.value === null)

onMounted(async () => {
  await search()
  if (!registrationId.value) return

  const owed = amountOwedCents.value
  if (owed === null) return
  if (owed <= 0) return

  if (!game.value) {
    // The amount is known, so checkout could technically proceed — but the
    // review step (WCAG 3.3.4 Error Prevention) has nothing to review
    // without the Game's time and court. Same refusal as before T56.1,
    // for the same reason, now on a narrower trigger.
    gameMissing.value = true
    return
  }

  void startCheckout(registrationId.value, owed, amountOwedCurrency.value)
})
</script>

<template>
  <section class="game-checkout" aria-label="Checkout">
    <h1 class="game-checkout__heading">Checkout</h1>

    <p v-if="!registrationId" class="game-checkout__status game-checkout__status--error" role="alert">
      Missing registration — start from the Games list and join a game first.
    </p>

    <template v-else>
      <!-- FREE GAME (T9.2): a real product state, stated in words. No
           Payment is created and no amount is shown, because none is
           owed. -->
      <div v-if="isFreeGame" class="game-checkout__success" role="status" aria-live="polite">
        <p data-testid="free-game-notice">This game is free — there's nothing to pay. You're all set.</p>
      </div>

      <!-- T56.1/T56.2: the owed amount never arrived. We say so rather
           than charging a guessed figure the server would refuse. -->
      <p
        v-else-if="amountUnknown"
        class="game-checkout__status game-checkout__status--error"
        role="alert"
      >
        We can't tell what this registration owes, so we won't guess an amount. Go back to the Games list
        and start your payment from the game you joined.
      </p>

      <p
        v-else-if="gameMissing"
        class="game-checkout__status game-checkout__status--error"
        role="alert"
      >
        We couldn't load this game, so we can't show you its details. Go back to the Games list and try again.
      </p>

      <p v-else-if="step === 'preparing' && !createError && !amountUnknown" class="game-checkout__status" role="status">
        Preparing checkout…
      </p>

      <p
        v-if="createError && !isFreeGame && !gameMissing && !amountUnknown"
        class="game-checkout__status game-checkout__status--error"
        role="alert"
      >
        {{ createError }}
      </p>

      <!-- REVIEW/CONFIRM STEP (WCAG 3.3.4 Error Prevention, same pattern
           CourtBookingFlow.vue already uses for CreateBooking): ConfirmOnlinePayment
           can never fire before this step is on screen — useGamePayment's
           own confirm-step gate enforces that regardless of this template. -->
      <div v-if="!isFreeGame && !gameMissing && !amountUnknown && step === 'review' && payment" class="game-checkout__review">
        <h2 class="game-checkout__review-heading">Review your payment</h2>
        <dl class="game-checkout__summary">
          <div v-if="game" class="game-checkout__summary-row">
            <dt>Game</dt>
            <dd>{{ formatGameRange(game.startsAt, game.endsAt) }}</dd>
          </div>
          <div v-if="game" class="game-checkout__summary-row">
            <dt>Entry fee</dt>
            <dd>{{ entryFeeLabel(game.entryFeeCents) }}</dd>
          </div>
          <div class="game-checkout__summary-row">
            <dt>Amount</dt>
            <dd data-testid="checkout-amount">{{ formatMoneyCents(payment.amountCents) }}</dd>
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
        v-else-if="!isFreeGame && !gameMissing && !amountUnknown && step === 'success' && confirmedPayment"
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
