<script setup lang="ts">
// Online checkout flow for a Competition entry (T10.6, closes #96), mounted
// at /competitions/:id/checkout — replacing CompetitionEntryPanel.vue's
// former "online payments aren't available yet" disclosure now that a real
// PAYABLE_TYPE_COMPETITION_ENTRY payable exists
// (internal/payments/domain.PayableTypeCompetitionEntry) and Payments has a
// port/adapter that routes a paid competition entry to Competitions, not
// Social Play's registrations table (the #96 bug this ticket closes).
//
// REUSE, NOT A FORK (per this ticket's own instructions, citing T8.10's
// GameCheckout.vue and T9.6's UnpaidCashAmount.vue extraction as the
// established discipline): this view is GameCheckout.vue's exact structural
// pattern — the same three-step preparing/review/success flow, the same
// confirm-step gate, the same "no card-shaped input anywhere" PCI posture —
// reusing the SAME composable (`useGamePayment`, constructed with
// `payableType: 'PAYABLE_TYPE_COMPETITION_ENTRY'`, T10.6's addition to that
// composable) rather than a second, independently-maintained checkout state
// machine. What's forked is only the presentational template, because a
// Competition's review step shows different fields (a Competition has
// sessions plural, not a single start/end range; there is no
// `entryFeeLabel`-shaped field name collision worth sharing) — the same
// "share the logic, not necessarily the markup" line T9.6's own extraction
// drew for UnpaidCashAmount.vue.
//
// `:id` is the Competition id; the CompetitionEntry id (the actual Payment
// payableId) travels as `?entryId=`, mirroring GameCheckout.vue's identical
// `?registrationId=` convention and reasoning (a Payment is keyed to the
// Entry, not the Competition).
//
// actorUserId/entrantPlayerId (T10.6's authorization requirement, closes
// #96): CreateOnlinePayment for a competition_entry payable now requires
// the caller to claim to be the entrant (or an assigned Competition Admin) —
// see internal/payments/app.authorizeOnlineCreation's doc comment. This
// screen is the entrant's own checkout, so both fields are MOCK_PLAYER_ID
// (the same caller-supplied-claim-not-real-auth placeholder
// CompetitionEntryPanel.vue's own `enter()` call already uses — there is no
// JWT/session layer yet, HANDOFF.md's open Auth item).
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useCompetitionList } from '../composables/useCompetitionList'
import { useGamePayment } from '../composables/useGamePayment'
import { MOCK_PLAYER_ID } from '../composables/useJoinGame'
import { entryFeeLabel } from '../models/competition'
import { formatMoneyCents, DEFAULT_CURRENCY_CODE } from '../models/payment'
import type { CompetitionsClient } from '../api/competitionsClient'
import type { PaymentsClient } from '../api/paymentsClient'

const props = defineProps<{
  /** Injectable for tests; defaults to the real competitionsClient. */
  client?: CompetitionsClient
  /** Injectable for tests; defaults to the real paymentsClient. */
  paymentsClient?: PaymentsClient
}>()

const route = useRoute()
const competitionId = computed(() => String(route.params.id ?? ''))
const entryId = computed(() => {
  const raw = route.query.entryId
  return typeof raw === 'string' ? raw : ''
})

const { competitions, search } = useCompetitionList(props.client)
const competition = computed(() => competitions.value.find((c) => c.id === competitionId.value) ?? null)

const { step, payment, createError, confirming, confirmError, confirmedPayment, startCheckout, confirmPayment } =
  useGamePayment(props.paymentsClient, 'PAYABLE_TYPE_COMPETITION_ENTRY')

/** True once the Competition has loaded and its entry fee is 0 — a free
 * Competition, so there is nothing to pay and no Payment to create. */
const isFreeCompetition = computed(() => competition.value !== null && competition.value.entryFeeCents <= 0)

/** The Competition loaded but wasn't found, so its real price is unknown.
 * We refuse to invent one — mirrors GameCheckout.vue's identical rule. */
const competitionMissing = ref(false)

onMounted(async () => {
  await search()
  if (!entryId.value) return

  const loaded = competition.value
  if (!loaded) {
    competitionMissing.value = true
    return
  }
  if (loaded.entryFeeCents <= 0) return

  void startCheckout(
    entryId.value,
    loaded.entryFeeCents,
    loaded.entryFeeCurrency || DEFAULT_CURRENCY_CODE,
    { actorUserId: MOCK_PLAYER_ID, entrantPlayerId: MOCK_PLAYER_ID },
  )
})
</script>

<template>
  <section class="competition-checkout" aria-label="Checkout">
    <h1 class="competition-checkout__heading">Checkout</h1>

    <p v-if="!entryId" class="competition-checkout__status competition-checkout__status--error" role="alert">
      Missing entry — start from the Competitions list and enter a competition first.
    </p>

    <template v-else>
      <!-- FREE COMPETITION: a real product state, stated in words. No
           Payment is created and no amount is shown, because none is
           owed. -->
      <div v-if="isFreeCompetition" class="competition-checkout__success" role="status" aria-live="polite">
        <p data-testid="free-competition-notice">This competition is free — there's nothing to pay. You're all set.</p>
      </div>

      <p
        v-else-if="competitionMissing"
        class="competition-checkout__status competition-checkout__status--error"
        role="alert"
      >
        We couldn't load this competition, so we can't show you its price. Go back to the Competitions list and try again.
      </p>

      <p v-else-if="step === 'preparing' && !createError" class="competition-checkout__status" role="status">
        Preparing checkout…
      </p>

      <p
        v-if="createError && !isFreeCompetition && !competitionMissing"
        class="competition-checkout__status competition-checkout__status--error"
        role="alert"
      >
        {{ createError }}
      </p>

      <!-- REVIEW/CONFIRM STEP (WCAG 3.3.4 Error Prevention, same pattern
           GameCheckout.vue already uses): ConfirmOnlinePayment can never
           fire before this step is on screen — useGamePayment's own
           confirm-step gate enforces that regardless of this template. -->
      <div v-if="!isFreeCompetition && !competitionMissing && step === 'review' && payment" class="competition-checkout__review">
        <h2 class="competition-checkout__review-heading">Review your payment</h2>
        <dl class="competition-checkout__summary">
          <div v-if="competition" class="competition-checkout__summary-row">
            <dt>Competition</dt>
            <dd>{{ competition.name }}</dd>
          </div>
          <div v-if="competition" class="competition-checkout__summary-row">
            <dt>Entry fee</dt>
            <dd>{{ entryFeeLabel(competition.entryFeeCents) }}</dd>
          </div>
          <div class="competition-checkout__summary-row">
            <dt>Amount</dt>
            <dd data-testid="checkout-amount">{{ formatMoneyCents(payment.amountCents) }}</dd>
          </div>
        </dl>

        <p v-if="confirmError" class="competition-checkout__status competition-checkout__status--error" role="alert">
          {{ confirmError }}
        </p>

        <!-- Stripe-stub: no card-shaped input field anywhere (CLAUDE.md
             rule 11 / PCI guardrail) — a stub confirm button is the entire
             "payment form". -->
        <button type="button" class="competition-checkout__primary" :disabled="confirming" @click="confirmPayment">
          {{ confirming ? 'Confirming…' : 'Confirm payment (stub)' }}
        </button>
      </div>

      <!-- SUCCESS: ARIA live region (WCAG 4.1.3) -->
      <div
        v-else-if="!isFreeCompetition && !competitionMissing && step === 'success' && confirmedPayment"
        class="competition-checkout__success"
        role="status"
        aria-live="polite"
      >
        <p>Payment confirmed. Reference: {{ confirmedPayment.id }}.</p>
      </div>
    </template>
  </section>
</template>

<style scoped>
.competition-checkout {
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 480px;
  margin: 0 auto;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.competition-checkout__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.competition-checkout__status {
  color: var(--ink-soft);
}

.competition-checkout__status--error {
  color: var(--ink-warning);
}

.competition-checkout__review {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  padding: 1rem;
}

.competition-checkout__review-heading {
  font-size: var(--font-size-base);
  margin: 0;
}

.competition-checkout__summary {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.competition-checkout__summary-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.competition-checkout__summary-row dt {
  color: var(--ink-soft);
}

.competition-checkout__summary-row dd {
  margin: 0;
  font-weight: 600;
  text-align: right;
}

.competition-checkout__primary {
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

.competition-checkout__primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.competition-checkout__success {
  color: var(--ink-success);
}

/* iPad/web: a bit more breathing room, mirroring GameCheckout.vue's
   identical wider-not-restructured approach at these breakpoints. */
@media (min-width: 768px) {
  .competition-checkout {
    max-width: 560px;
    padding: 2rem 1.5rem;
  }
}
</style>
