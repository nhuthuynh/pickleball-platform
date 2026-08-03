// Shared, reactive pending-cash-payments count (T8.10 kickoff note decision
// #3: "a Host-facing dashboard shows a pending-cash-payments count badge").
// Mirrors state/roleEvidence.ts's shape exactly: a module-scoped `ref` so
// AppNav.vue (mounted once, in App.vue's persistent chrome) and
// HostPayments.vue (mounted only when `/host/payments` is visited) both
// read the same live count without a shared store library (no Pinia in
// this repo) — `useHostPayments.ts`'s `load()` is the sole writer, calling
// `setPendingCashPaymentsCount` after every successful fetch or mark-paid
// action, so the badge never drifts from the dashboard's own list.
//
// Not persisted to localStorage (unlike roleEvidence.ts's Host-evidence
// flag): this is a live server-derived count, not durable client evidence,
// so re-deriving it via a fresh `load()` on next mount is correct, not a
// fallback.
import { ref } from 'vue'

export const pendingCashPaymentsCount = ref(0)

export function setPendingCashPaymentsCount(count: number): void {
  pendingCashPaymentsCount.value = count
}

/** Test-only: resets the shared count so specs don't leak state across each
 * other (this module's state is otherwise shared for the lifetime of the
 * page, by design) — mirrors roleEvidence.ts's identical test-reset helper. */
export function __resetPendingCashPaymentsCountForTests(): void {
  pendingCashPaymentsCount.value = 0
}
