import { ref, type Ref } from 'vue'
import { paymentsClient, type PaymentsClient } from '../api/paymentsClient'
import { mapToPayment, toMoneyRequest, type PaymentSummary } from '../models/payment'

export type CheckoutStep = 'preparing' | 'review' | 'success'

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
  /** Calls `CreateOnlinePayment` for `payableId` (a Registration id) and
   * `amountCents`, moving to the `'review'` step on success. */
  startCheckout: (payableId: string, amountCents: number, currencyCode?: string) => Promise<void>
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
 * The **confirm-step gate** (WCAG 3.3.4 Error Prevention) lives here, not
 * just in the UI, mirroring `useCourtBooking.confirmBooking`'s identical
 * gate: `confirmPayment` is a no-op unless `step` is already `'review'`
 * with a `payment` already created — `ConfirmOnlinePayment` can never fire
 * before `CreateOnlinePayment` has produced a Payment to confirm and the
 * review step has been shown, regardless of what the calling component
 * does.
 */
export function useGamePayment(client: PaymentsClient = paymentsClient): UseGamePaymentResult {
  const step = ref<CheckoutStep>('preparing')
  const payment = ref<PaymentSummary | null>(null)
  const createError = ref<string | null>(null)
  const confirming = ref(false)
  const confirmError = ref<string | null>(null)
  const confirmedPayment = ref<PaymentSummary | null>(null)

  async function startCheckout(payableId: string, amountCents: number, currencyCode?: string): Promise<void> {
    createError.value = null
    try {
      const { data, error } = await client.POST('/v1/payments:createOnline', {
        body: {
          payableType: 'PAYABLE_TYPE_REGISTRATION',
          payableId,
          amount: toMoneyRequest(amountCents, currencyCode),
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
