import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import GameCheckout from '../GameCheckout.vue'
import { routes } from '../../router'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { PaymentsClient } from '../../api/paymentsClient'

const GAME_LISTING = {
  game: {
    id: 'g1',
    hostId: 'host-1',
    venueFacilityId: 'facility-1',
    courtIds: ['court-1'],
    startsAt: '2026-09-01T10:00:00Z',
    endsAt: '2026-09-01T11:00:00Z',
    capacity: 8,
    status: 'GAME_STATUS_SCHEDULED',
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    guestAllowance: 2,
    entryFee: { amountCents: '1000', currencyCode: 'USD' },
  },
  spotsLeft: 5,
}

function socialplayClientStub(): SocialPlayClient {
  return {
    GET: vi.fn(async () => ({ data: { games: [GAME_LISTING] }, error: undefined, response: { status: 200 } })),
    POST: vi.fn(),
  } as unknown as SocialPlayClient
}

function paymentsClientStub(handlers: {
  createOnline?: () => unknown
  confirmOnline?: () => unknown
}): PaymentsClient {
  const POST = vi.fn(async (path: string) => {
    if (path === '/v1/payments:createOnline') return handlers.createOnline?.()
    if (path === '/v1/payments/{paymentId}:confirmOnline') return handlers.confirmOnline?.()
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as PaymentsClient
}

function paymentOk(status: string) {
  return {
    data: {
      payment: {
        id: 'pay-1',
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

async function mountCheckout(paymentsClient: PaymentsClient, query: Record<string, string> = { registrationId: 'reg-1' }) {
  const router = createRouter({ history: createMemoryHistory(), routes })
  await router.push({ name: 'game-checkout', params: { id: 'g1' }, query })
  await router.isReady()
  const wrapper = mount(GameCheckout, {
    props: { client: socialplayClientStub(), paymentsClient },
    global: { plugins: [router] },
  })
  await flushPromises()
  return wrapper
}

describe('GameCheckout', () => {
  it('shows an error and never starts checkout when no registrationId is present', async () => {
    const paymentsClient = paymentsClientStub({})
    const wrapper = await mountCheckout(paymentsClient, {})

    expect(wrapper.find('[role="alert"]').text()).toContain('Missing registration')
    expect(paymentsClient.POST).not.toHaveBeenCalled()
  })

  // T8.10 required test: confirm-step gate — ConfirmOnlinePayment must
  // never be called before the review step is shown.
  it('never calls ConfirmOnlinePayment before the review step renders', async () => {
    let resolveCreate: (value: unknown) => void = () => {}
    const paymentsClient = paymentsClientStub({
      createOnline: () => new Promise((resolve) => { resolveCreate = resolve }),
    })
    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({ name: 'game-checkout', params: { id: 'g1' }, query: { registrationId: 'reg-1' } })
    await router.isReady()
    const wrapper = mount(GameCheckout, {
      props: { client: socialplayClientStub(), paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()

    // Still preparing — the review step (with its Confirm button) has not
    // rendered yet, so there is nothing to click; assert the button simply
    // isn't there rather than call it.
    expect(wrapper.find('.game-checkout__review').exists()).toBe(false)
    expect(paymentsClient.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())

    resolveCreate(paymentOk('PAYMENT_STATUS_UNPAID'))
    await flushPromises()
    expect(wrapper.find('.game-checkout__review').exists()).toBe(true)
    expect(paymentsClient.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())
  })

  // T8.10 required test: "the online happy path (checkout -> confirm -> paid)".
  it('happy path: prepares checkout, shows the review step, then confirms and shows a success message', async () => {
    const paymentsClient = paymentsClientStub({
      createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID'),
      confirmOnline: () => paymentOk('PAYMENT_STATUS_PAID'),
    })
    const wrapper = await mountCheckout(paymentsClient)

    expect(wrapper.find('.game-checkout__review').exists()).toBe(true)
    expect(wrapper.text()).toContain('$10.00')

    await wrapper.find('.game-checkout__primary').trigger('click')
    await flushPromises()

    expect(paymentsClient.POST).toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', {
      params: { path: { paymentId: 'pay-1' } },
    })
    const success = wrapper.find('[role="status"][aria-live="polite"]')
    expect(success.exists()).toBe(true)
    expect(success.text()).toContain('Payment confirmed')
  })
})

// T9.2: checkout charges the Game's real fee, and a free Game creates no
// Payment at all (a zero-amount Payment is rejected by the Payments domain,
// so offering one would be a button that could only fail).
describe('GameCheckout — real entry fee (T9.2)', () => {
  /** Mounts with a Game whose entry fee is `amountCents`. */
  async function mountWithFee(amountCents: string, paymentsClient: PaymentsClient) {
    const listing = {
      ...GAME_LISTING,
      game: { ...GAME_LISTING.game, entryFee: { amountCents, currencyCode: 'USD' } },
    }
    const social = {
      GET: vi.fn(async () => ({ data: { games: [listing] }, error: undefined, response: { status: 200 } })),
      POST: vi.fn(),
    } as unknown as SocialPlayClient

    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({ name: 'game-checkout', params: { id: 'g1' }, query: { registrationId: 'reg-1' } })
    await router.isReady()
    const wrapper = mount(GameCheckout, {
      props: { client: social, paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()
    return wrapper
  }

  it("creates the Payment for the Game's real entry fee", async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountWithFee('2500', paymentsClient)

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls.length).toBe(1)
    expect(calls[0]![1].body.amount).toEqual({ amountCents: '2500', currencyCode: 'USD' })
  })

  it('creates no Payment for a free game and says so in words', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const wrapper = await mountWithFee('0', paymentsClient)

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    const notice = wrapper.get('[data-testid="free-game-notice"]').text()
    expect(notice).toContain('free')
    expect(notice).not.toContain('$0.00')
  })
})
