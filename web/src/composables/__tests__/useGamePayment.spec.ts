import { describe, it, expect, vi } from 'vitest'
import { useGamePayment } from '../useGamePayment'
import type { PaymentsClient } from '../../api/paymentsClient'

function fakeClient(handlers: {
  createOnline?: (body: unknown) => unknown
  confirmOnline?: (paymentId: string) => unknown
}): PaymentsClient {
  const POST = vi.fn(async (path: string, options: { body?: unknown; params?: { path?: { paymentId?: string } } }) => {
    if (path === '/v1/payments:createOnline') return handlers.createOnline?.(options.body)
    if (path === '/v1/payments/{paymentId}:confirmOnline') {
      return handlers.confirmOnline?.(options.params?.path?.paymentId ?? '')
    }
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as PaymentsClient
}

function paymentOk(id = 'pay-1', status = 'PAYMENT_STATUS_UNPAID') {
  return {
    data: {
      payment: {
        id,
        payableType: 'PAYABLE_TYPE_REGISTRATION',
        payableId: 'reg-1',
        amount: { amountCents: '1000', currencyCode: 'USD' },
        method: 'PAYMENT_METHOD_ONLINE',
        status,
        stripeReference: 'stub-ref',
        recordedByUserId: '',
      },
    },
    error: undefined,
    response: { status: 200 },
  }
}

describe('useGamePayment', () => {
  // T8.10 required test: "the confirm-step gate (asserts ConfirmOnlinePayment
  // is never called before the review step is shown, mirroring T7.6's own
  // confirm-gate test)".
  describe('confirm-step gate', () => {
    it('never calls ConfirmOnlinePayment if confirmPayment is called before checkout has been started', async () => {
      const client = fakeClient({ confirmOnline: () => paymentOk('pay-1', 'PAYMENT_STATUS_PAID') })
      const { confirmPayment, step } = useGamePayment(client)

      await confirmPayment()

      expect(step.value).toBe('preparing')
      expect(client.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())
    })

    it('never calls ConfirmOnlinePayment while CreateOnlinePayment is still in flight (before the review step renders)', async () => {
      let resolveCreate: (value: unknown) => void = () => {}
      const client = fakeClient({
        createOnline: () => new Promise((resolve) => { resolveCreate = resolve }),
        confirmOnline: () => paymentOk('pay-1', 'PAYMENT_STATUS_PAID'),
      })
      const { startCheckout, confirmPayment, step } = useGamePayment(client)

      const checkoutPromise = startCheckout('reg-1', 1000)
      // Still 'preparing' — CreateOnlinePayment hasn't resolved yet.
      await confirmPayment()
      expect(client.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())

      resolveCreate(paymentOk())
      await checkoutPromise
      expect(step.value).toBe('review')
    })
  })

  // T8.10 required test: "the online happy path (checkout -> confirm -> paid)".
  it('happy path: startCheckout then confirmPayment moves to success with the paid Payment', async () => {
    const client = fakeClient({
      createOnline: () => paymentOk('pay-1', 'PAYMENT_STATUS_UNPAID'),
      confirmOnline: () => paymentOk('pay-1', 'PAYMENT_STATUS_PAID'),
    })
    const { step, payment, confirmedPayment, startCheckout, confirmPayment } = useGamePayment(client)

    await startCheckout('reg-1', 1000, 'USD')
    expect(step.value).toBe('review')
    expect(payment.value?.id).toBe('pay-1')
    expect(payment.value?.status).toBe('PAYMENT_STATUS_UNPAID')

    await confirmPayment()

    expect(client.POST).toHaveBeenNthCalledWith(1, '/v1/payments:createOnline', {
      body: {
        payableType: 'PAYABLE_TYPE_REGISTRATION',
        payableId: 'reg-1',
        amount: { amountCents: '1000', currencyCode: 'USD' },
      },
    })
    expect(client.POST).toHaveBeenNthCalledWith(2, '/v1/payments/{paymentId}:confirmOnline', {
      params: { path: { paymentId: 'pay-1' } },
    })
    expect(step.value).toBe('success')
    expect(confirmedPayment.value?.status).toBe('PAYMENT_STATUS_PAID')
  })

  it('startCheckout sets a human-readable error and stays on the preparing step when CreateOnlinePayment fails', async () => {
    const client = fakeClient({ createOnline: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }) })
    const { step, createError, startCheckout } = useGamePayment(client)

    await startCheckout('reg-1', 1000)

    expect(step.value).toBe('preparing')
    expect(createError.value).toBeTruthy()
  })

  it('confirmPayment sets a human-readable error and stays on the review step when ConfirmOnlinePayment fails', async () => {
    const client = fakeClient({
      createOnline: () => paymentOk(),
      confirmOnline: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
    })
    const { step, confirmError, startCheckout, confirmPayment } = useGamePayment(client)

    await startCheckout('reg-1', 1000)
    await confirmPayment()

    expect(step.value).toBe('review')
    expect(confirmError.value).toBeTruthy()
  })
})
