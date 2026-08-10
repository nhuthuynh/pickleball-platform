import { ref, type Ref } from 'vue'
import { paymentsClient, type PaymentsClient } from '../api/paymentsClient'
import { mapToPayment, toMoneyRequest, type PaymentSummary } from '../models/payment'

export type CheckoutStep = 'preparing' | 'review' | 'success'

/** The two payable types the online checkout flow drives today.
 * PAYABLE_TYPE_REGISTRATION is the T8.10 original (a Social Play Game);
 * PAYABLE_TYPE_COMPETITION_ENTRY is T10.6's addition (closes #96) — same
 * composable, same flow, a different payable type routed to a different
 * context's reconciliation on the server (see
 * internal/payments/app.Service.reconcileCompetitionEntryPaymentStatus). */
export type CheckoutPayableType = 'PAYABLE_TYPE_REGISTRATION' | 'PAYABLE_TYPE_COMPETITION_ENTRY'

/** Authorization facts for a PAYABLE_TYPE_COMPETITION_ENTRY checkout
 * (T10.6): the actor claiming to start the payment, and — since
 * internal/payments has no live join to Competitions' database — the
 * entrant id the caller asserts this actor matches (mirrors
 * internal/payments/app.CreateOnlinePaymentInput's own caller-supplied
 * fields exactly). Ignored server-side for every other payable type, so
 * this is a no-op to pass for the PAYABLE_TYPE_REGISTRATION path. */
export interface CheckoutActor {
  actorUserId: string
  entrantPlayerId: string
}

export interface UseGamePaymentResult {
  step: Ref<CheckoutStep>
  /** The unpaid Payment `CreateOnlinePayment` returned — set once checkout
   * has been prepared, cleared again only by `reset()`. */
  payment: Ref<PaymentSummary | null>
  /** Human-readable message when `CreateOnlinePayment` itself failed. */
  createError: Ref<string | null>
  confirming: Ref<boolean>
  /** Human-readable message for a `ConfirmOnlinePayment` failure. */
  confirmError: Ref<string | null>
  confirmedPayment: Ref<PaymentSummary | null>
  /** Calls `CreateOnlinePayment` for `payableId` (a Registration id, or —
   * T10.6 — a CompetitionEntry id, per the `payableType` this composable
   * was constructed with) and `amountCents`, moving to the `'review'` step
   * on success. `actor` is required (and validated server-side) only for
   * the PAYABLE_TYPE_COMPETITION_ENTRY path — see `CheckoutActor`. */
  startCheckout: (payableId: string, amountCents: number, currencyCode?: string, actor?: CheckoutActor) => Promise<void>
  /** Calls `ConfirmOnlinePayment` (Stripe-stub) — but only once the review
   * step has actually been shown, see the doc comment below. */
  confirmPayment: () => Promise<void>
  reset: () => void
}

/**
 * Drives the online checkout flow (T8.10): `CreateOnlinePayment` (builds an
 * unpaid Payment + authorizes a Stripe-stub intent) followed by, only after
 * a review step, `ConfirmOnlinePayment` (captures funds, Stripe-stub — see
 * models/payment.ts's header comment on why no real card form or price
 * exists yet). `client` is injectable (defaults to the real
 * `paymentsClient`), same pattern as `useCourtBooking`/`useJoinGame`.
 *
 * `payableType` (T10.6, closes #96) defaults to `'PAYABLE_TYPE_REGISTRATION'`
 * — GameCheckout.vue's original T8.10 behaviour, unchanged — so every
 * existing call site keeps working without passing it. CompetitionCheckout.vue
 * is the one caller that passes `'PAYABLE_TYPE_COMPETITION_ENTRY'`: this is
 * the "reuse the pattern, don't fork the component" extraction the ticket's
 * own instructions require (mirrors T9.6's UnpaidCashAmount.vue extraction),
 * applied to the checkout *composable* rather than a presentational
 * component, since GameCheckout.vue's own template is otherwise
 * Game-specific (its review step renders `game.startsAt`/`entryFeeLabel`,
 * neither of which a CompetitionEntry has) and forking the flow LOGIC would
 * have reproduced the exact bug #96 is about — two independently-maintained
 * copies of the create -> review -> confirm state machine, one per context,
 * free to drift.
 *
 * The **confirm-step gate** (WCAG 3.3.4 Error Prevention) lives here, not
 * just in the UI, mirroring `useCourtBooking.confirmBooking`'s identical
 * gate: `confirmPayment` is a no-op unless `step` is already `'review'`
 * with a `payment` already created — `ConfirmOnlinePayment` can never fire
 * before `CreateOnlinePayment` has produced a Payment to confirm and the
 * review step has been shown, regardless of what the calling component
 * does.
 */
export function useGamePayment(
  client: PaymentsClient = paymentsClient,
  payableType: CheckoutPayableType = 'PAYABLE_TYPE_REGISTRATION',
): UseGamePaymentResult {
  const step = ref<CheckoutStep>('preparing')
  const payment = ref<PaymentSummary | null>(null)
  const createError = ref<string | null>(null)
  const confirming = ref(false)
  const confirmError = ref<string | null>(null)
  const confirmedPayment = ref<PaymentSummary | null>(null)

  async function startCheckout(
    payableId: string,
    amountCents: number,
    currencyCode?: string,
    actor?: CheckoutActor,
  ): Promise<void> {
    createError.value = null
    try {
      const { data, error } = await client.POST('/v1/payments:createOnline', {
        body: {
          payableType,
          payableId,
          amount: toMoneyRequest(amountCents, currencyCode),
          ...(actor
            ? { actorUserId: actor.actorUserId, entrantPlayerId: actor.entrantPlayerId }
            : {}),
        },
      })
      if (error || !data?.payment) {
        createError.value = 'Could not start checkout. Please try again.'
        return
      }
      payment.value = mapToPayment(data.payment)
      step.value = 'review'
    } catch {
      createError.value = 'Could not reach the server. Check your connection and try again.'
    }
  }

  async function confirmPayment(): Promise<void> {
    // Confirm-step gate: never call ConfirmOnlinePayment without a reviewed,
    // already-created Payment.
    if (step.value !== 'review' || !payment.value) return

    confirming.value = true
    confirmError.value = null
    try {
      const { data, error } = await client.POST('/v1/payments/{paymentId}:confirmOnline', {
        params: { path: { paymentId: payment.value.id } },
      })
      if (error || !data?.payment) {
        confirmError.value = 'Could not confirm this payment. Please try again.'
        return
      }
      confirmedPayment.value = mapToPayment(data.payment)
      step.value = 'success'
    } catch {
      confirmError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      confirming.value = false
    }
  }

  function reset(): void {
    step.value = 'preparing'
    payment.value = null
    createError.value = null
    confirming.value = false
    confirmError.value = null
    confirmedPayment.value = null
  }

  return {
    step,
    payment,
    createError,
    confirming,
    confirmError,
    confirmedPayment,
    startCheckout,
    confirmPayment,
    reset,
  }
}
