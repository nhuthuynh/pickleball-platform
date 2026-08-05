<script setup lang="ts">
// The "money still owed, in cash, at the facility" line — T8.10's
// established way of surfacing an unpaid cash obligation, extracted from
// HostPayments.vue by T9.6 so the Competition roster surfaces unpaid-cash
// entries with the SAME affordance rather than a second, near-identical
// badge.
//
// T9.6's Instructions: "Unpaid-cash entries surface using the pattern
// HostPayments.vue already established (T8.10), reusing its components
// rather than duplicating the badge/list." That pattern wasn't a component
// yet — it was one inline `<p>` in HostPayments.vue — so this is that line
// lifted out, with HostPayments.vue rewired onto it. Its existing spec
// (HostPayments.spec.ts asserts the rendered text contains "cash at
// facility") passes unchanged, which is what makes the extraction safe.
//
// WCAG 1.4.1 (Use of Color): the obligation is TEXT — the amount and the
// words "due (cash at facility)". It is never a bare coloured dot, a red
// border, or a colour-coded row, because a Host who can't distinguish those
// colours would otherwise have no way to tell an unpaid row from a paid
// one.
import { formatMoneyCents } from '../../models/payment'

defineProps<{
  /** The amount owed, in integer minor units. */
  amountCents: number
}>()
</script>

<template>
  <p class="unpaid-cash" data-testid="unpaid-cash-amount">
    {{ formatMoneyCents(amountCents) }} due (cash at facility)
  </p>
</template>

<style scoped>
.unpaid-cash {
  margin: 0;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}
</style>
