<script setup lang="ts">
// Host pending-cash-payments dashboard (T8.10 kickoff note decision #3),
// mounted at /host/payments (replacing T8.1's placeholder route): lists
// this Host's Games' Registrations where PaymentStatus == unpaid and the
// Game accepts cash, with a "Mark paid" action (RecordOfflinePayment).
//
// Responsive treatment: T8.10's Instructions ask this to "follow T7.5's
// list/detail responsive pattern" — this dashboard has no separate detail
// pane to split into (each pending row already shows everything relevant
// inline; there is no secondary "select a row to see more" concept the way
// FacilityDetailPanel/GameDetailPanel have), so the pattern actually
// applied is the shared underlying one (single stacked column on iPhone,
// wider breathing room at iPad/web), not a literal list+detail split for a
// screen with nothing to put in a detail pane.
import { onMounted } from 'vue'
import { useBreakpoint } from '../composables/useBreakpoint'
import { useHostPayments, MOCK_HOST_ID, type PendingCashPayment } from '../composables/useHostPayments'
import { setPendingCashPaymentsCount } from '../state/hostPendingPayments'
import { formatMoneyCents } from '../models/payment'
import type { SocialPlayClient } from '../api/socialplayClient'
import type { PaymentsClient } from '../api/paymentsClient'

const props = defineProps<{
  /** Injectable for tests; defaults to the real socialplayClient. */
  client?: SocialPlayClient
  /** Injectable for tests; defaults to the real paymentsClient. */
  paymentsClient?: PaymentsClient
  /** Injectable for tests; defaults to the ambient `window` (matchMedia). */
  win?: Pick<Window, 'matchMedia'>
}>()

const { isWeb } = useBreakpoint(props.win ?? window)

const { pending, loading, error, markingPaidId, markPaidError, load, markPaid } = useHostPayments(
  props.client,
  props.paymentsClient,
)

async function refresh(): Promise<void> {
  await load(MOCK_HOST_ID)
  setPendingCashPaymentsCount(pending.value.length)
}

function onMarkPaid(entry: PendingCashPayment): void {
  void (async () => {
    await markPaid(entry, MOCK_HOST_ID)
    setPendingCashPaymentsCount(pending.value.length)
  })()
}

onMounted(() => {
  void refresh()
})
</script>

<template>
  <section class="host-payments" :data-breakpoint="isWeb ? 'web' : 'compact'" aria-label="Pending cash payments">
    <h1 class="host-payments__heading">Payments</h1>
    <p class="host-payments__subheading">Cash registrations still marked unpaid, across your Games.</p>

    <p v-if="loading" class="host-payments__status" role="status">Loading pending payments…</p>

    <div v-else-if="error" class="host-payments__status host-payments__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="host-payments__retry" @click="refresh">Try again</button>
    </div>

    <p v-else-if="pending.length === 0" class="host-payments__empty">No pending cash payments right now.</p>

    <ul v-else class="host-payments__list">
      <li v-for="entry in pending" :key="entry.registrationId" class="host-payments__row">
        <div class="host-payments__row-details">
          <p class="host-payments__row-game">{{ entry.gameLabel }}</p>
          <p class="host-payments__row-player">
            Player {{ entry.playerId }}
            <span v-if="entry.guestCount > 0"> + {{ entry.guestCount }} guest{{ entry.guestCount === 1 ? '' : 's' }}</span>
          </p>
          <!-- WCAG 1.4.1: the amount-due/cash label is text, never color-only. -->
          <p class="host-payments__row-amount">
            {{ formatMoneyCents(entry.entryFeeCents) }} due (cash at facility)
          </p>
        </div>
        <button
          type="button"
          class="host-payments__mark-paid"
          :disabled="markingPaidId === entry.registrationId"
          @click="onMarkPaid(entry)"
        >
          {{ markingPaidId === entry.registrationId ? 'Marking…' : 'Mark paid' }}
        </button>
      </li>
    </ul>

    <p v-if="markPaidError" class="host-payments__status host-payments__status--error" role="alert">
      {{ markPaidError }}
    </p>
  </section>
</template>

<style scoped>
.host-payments {
  font-family: var(--font-family-ui);
  color: var(--ink);
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 720px;
}

.host-payments__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.host-payments__subheading {
  margin: 0;
  color: var(--ink-soft);
}

.host-payments__status {
  color: var(--ink-soft);
}

.host-payments__status--error {
  color: var(--ink-warning);
}

.host-payments__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.host-payments__empty {
  color: var(--ink-soft);
}

.host-payments__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.host-payments__row {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper-raised);
}

.host-payments__row-details p {
  margin: 0;
}

.host-payments__row-game {
  font-weight: 600;
}

.host-payments__row-player,
.host-payments__row-amount {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.host-payments__mark-paid {
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

.host-payments__mark-paid:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* iPad/web: rows go side-by-side (details left, action right) instead of
   the iPhone-stacked column, mirroring T7.5's FacilityListPanel row
   treatment at the same breakpoint. */
@media (min-width: 768px) {
  .host-payments__row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  .host-payments__mark-paid {
    align-self: center;
  }
}
</style>
