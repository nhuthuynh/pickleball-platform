import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import CompetitionCheckout from '../CompetitionCheckout.vue'
import { routes } from '../../router'
import type { CompetitionsClient } from '../../api/competitionsClient'
import type { PaymentsClient } from '../../api/paymentsClient'

const COMPETITION_LISTING = {
  competition: {
    id: 'c1',
    hostId: 'host-1',
    name: 'Autumn Doubles Ladder',
    venueFacilityId: 'facility-1',
    sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
    capacity: 16,
    guestAllowance: 2,
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    entryFee: { amountCents: '1000', currencyCode: 'USD' },
    format: 'COMPETITION_FORMAT_DOUBLES',
    status: 'COMPETITION_STATUS_SCHEDULED',
  },
  spotsLeft: 5,
}

function competitionsClientStub(): CompetitionsClient {
  return {
    GET: vi.fn(async () => ({ data: { competitions: [COMPETITION_LISTING] }, error: undefined, response: { status: 200 } })),
    POST: vi.fn(),
  } as unknown as CompetitionsClient
}

function paymentsClientStub(handlers: {
  createOnline?: (body: unknown) => unknown
  confirmOnline?: () => unknown
}): PaymentsClient {
  const POST = vi.fn(async (path: string, options: { body?: unknown }) => {
    if (path === '/v1/payments:createOnline') return handlers.createOnline?.(options.body)
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
        payableType: 'PAYABLE_TYPE_COMPETITION_ENTRY',
        payableId: 'entry-1',
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

async function mountCheckout(paymentsClient: PaymentsClient, query: Record<string, string> = { entryId: 'entry-1' }) {
  const router = createRouter({ history: createMemoryHistory(), routes })
  await router.push({ name: 'competition-checkout', params: { id: 'c1' }, query })
  await router.isReady()
  const wrapper = mount(CompetitionCheckout, {
    props: { client: competitionsClientStub(), paymentsClient },
    global: { plugins: [router] },
  })
  await flushPromises()
  return wrapper
}

describe('CompetitionCheckout', () => {
  it('shows an error and never starts checkout when no entryId is present', async () => {
    const paymentsClient = paymentsClientStub({})
    const wrapper = await mountCheckout(paymentsClient, {})

    expect(wrapper.find('[role="alert"]').text()).toContain('Missing entry')
    expect(paymentsClient.POST).not.toHaveBeenCalled()
  })

  // T10.6's own authorization requirement (closes #96): a CompetitionEntry
  // checkout carries actorUserId/entrantPlayerId, both the entrant's mock
  // identity, so CreateOnlinePayment's new authorizeOnlineCreation check
  // (internal/payments/app/service.go) accepts it — mirrors
  // authorizeOfflineRecording's same actor-claim caveat every other
  // ActorUserID field in this codebase already carries.
  it("creates the Payment for the entry's real fee, carrying the entrant's actor claim", async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountCheckout(paymentsClient)

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls.length).toBe(1)
    expect(calls[0]![1].body).toEqual({
      payableType: 'PAYABLE_TYPE_COMPETITION_ENTRY',
      payableId: 'entry-1',
      amount: { amountCents: '1000', currencyCode: 'USD' },
      actorUserId: 'player-mock-1',
      entrantPlayerId: 'player-mock-1',
    })
  })

  // T10.6 required test: confirm-step gate — ConfirmOnlinePayment must
  // never be called before the review step is shown, mirrors
  // GameCheckout.spec.ts's identical T8.10 test.
  it('never calls ConfirmOnlinePayment before the review step renders', async () => {
    let resolveCreate: (value: unknown) => void = () => {}
    const paymentsClient = paymentsClientStub({
      createOnline: () => new Promise((resolve) => { resolveCreate = resolve }),
    })
    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({ name: 'competition-checkout', params: { id: 'c1' }, query: { entryId: 'entry-1' } })
    await router.isReady()
    const wrapper = mount(CompetitionCheckout, {
      props: { client: competitionsClientStub(), paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(wrapper.find('.competition-checkout__review').exists()).toBe(false)
    expect(paymentsClient.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())

    resolveCreate(paymentOk('PAYMENT_STATUS_UNPAID'))
    await flushPromises()
    expect(wrapper.find('.competition-checkout__review').exists()).toBe(true)
    expect(paymentsClient.POST).not.toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', expect.anything())
  })

  // T10.6 required test: "the online happy path (checkout -> confirm -> paid)".
  it('happy path: prepares checkout, shows the review step, then confirms and shows a success message', async () => {
    const paymentsClient = paymentsClientStub({
      createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID'),
      confirmOnline: () => paymentOk('PAYMENT_STATUS_PAID'),
    })
    const wrapper = await mountCheckout(paymentsClient)

    expect(wrapper.find('.competition-checkout__review').exists()).toBe(true)
    expect(wrapper.text()).toContain('$10.00')

    await wrapper.find('.competition-checkout__primary').trigger('click')
    await flushPromises()

    expect(paymentsClient.POST).toHaveBeenCalledWith('/v1/payments/{paymentId}:confirmOnline', {
      params: { path: { paymentId: 'pay-1' } },
    })
    const success = wrapper.find('[role="status"][aria-live="polite"]')
    expect(success.exists()).toBe(true)
    expect(success.text()).toContain('Payment confirmed')
  })

  it('creates no Payment for a free competition and says so in words', async () => {
    const listing = {
      ...COMPETITION_LISTING,
      competition: { ...COMPETITION_LISTING.competition, entryFee: { amountCents: '0', currencyCode: 'USD' } },
    }
    const client = {
      GET: vi.fn(async () => ({ data: { competitions: [listing] }, error: undefined, response: { status: 200 } })),
      POST: vi.fn(),
    } as unknown as CompetitionsClient
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })

    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({ name: 'competition-checkout', params: { id: 'c1' }, query: { entryId: 'entry-1' } })
    await router.isReady()
    const wrapper = mount(CompetitionCheckout, {
      props: { client, paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    const notice = wrapper.get('[data-testid="free-competition-notice"]').text()
    expect(notice).toContain('free')
    expect(notice).not.toContain('$0.00')
  })
})
