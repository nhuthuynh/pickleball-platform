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

// T57.1 (#126): the default query now carries the FROZEN owed amount
// alongside the entry id, because that is what the enter flow actually
// pushes (DiscoverCompetitions.vue's and CompetitionLanding.vue's
// `onPayOnline`) and what the view now charges. 1000 matches
// COMPETITION_LISTING's entry fee — i.e. an entrant who brought no guests
// — so every test written before this ticket keeps exercising the same
// figure it always did.
async function mountCheckout(
  paymentsClient: PaymentsClient,
  query: Record<string, string> = { entryId: 'entry-1', amountOwedCents: '1000', amountOwedCurrency: 'USD' },
) {
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
  // T57.1 amended the amount's SOURCE (the entry's frozen AmountOwed, not
  // the Competition's entry fee) — the actor-claim assertion this test
  // exists for is untouched, and 1000 is still the figure because an
  // entrant with no guests owes exactly one entry fee.
  it("creates the Payment for a real fee, carrying the entrant's actor claim", async () => {
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
    await router.push({
      name: 'competition-checkout',
      params: { id: 'c1' },
      query: { entryId: 'entry-1', amountOwedCents: '1000', amountOwedCurrency: 'USD' },
    })
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

  // T57.1 AMENDED this test. It seeded a Competition with a zero entry fee
  // and asserted no Payment was created; the view no longer reads the price
  // off the Competition at all, so the zero now arrives as the entry's own
  // owed amount. The Competition is left free too, because the two agree by
  // construction for a free Competition and a fixture where they disagreed
  // would be describing a state the domain cannot produce. What the test
  // proves is unchanged: nothing owed means no Payment and a notice in
  // words.
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
    await router.push({
      name: 'competition-checkout',
      params: { id: 'c1' },
      query: { entryId: 'entry-1', amountOwedCents: '0', amountOwedCurrency: 'USD' },
    })
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

// T57.1/T57.2 — Competitions' mirror of T56's per-head pricing: checkout
// charges the FROZEN amount the entry recorded, not a figure re-derived
// here.
//
// The two halves are one behaviour, exactly as they are for a Game. T57.1
// made the owed amount entry_fee × (1 + guest_count) and froze it onto the
// CompetitionEntry; T57.2 made the server validate every online entry
// payment against that figure. So a client that keeps sending the bare
// per-entrant fee is not merely undercharging — it is now REFUSED, for
// every entrant who brought anyone.
describe('CompetitionCheckout — frozen per-head amount (T57.1/T57.2)', () => {
  async function mountWithQuery(query: Record<string, string>, paymentsClient: PaymentsClient) {
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

  // The defect, stated as a test. COMPETITION_LISTING's entry fee is 1000;
  // an entrant who brought two guests owes 3000. Before this, the client
  // sent 1000 and the other two heads were free.
  it('charges the amount the entry owes, not the per-entrant fee', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountWithQuery(
      { entryId: 'entry-1', amountOwedCents: '3000', amountOwedCurrency: 'USD' },
      paymentsClient,
    )

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls.length).toBe(1)
    expect(calls[0]![1].body.amount).toEqual({ amountCents: '3000', currencyCode: 'USD' })
  })

  // The currency travels with the amount (ADR-0005) rather than being
  // defaulted here — T57.2 compares both, so a right number in the wrong
  // currency is refused just as hard as a wrong number.
  it('sends the currency the entry recorded', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountWithQuery(
      { entryId: 'entry-1', amountOwedCents: '3000', amountOwedCurrency: 'GBP' },
      paymentsClient,
    )

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls[0]![1].body.amount.currencyCode).toBe('GBP')
  })

  // The important refusal. Falling back to the Competition's entry fee when
  // the owed amount is missing would be right only for an entrant who
  // brought nobody and silently wrong — now server-refused — for everyone
  // else. A checkout that cannot know what is owed says so instead of
  // guessing, exactly as it already does for a Competition it could not
  // load.
  it('refuses to guess an amount when the owed figure is absent', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const wrapper = await mountWithQuery({ entryId: 'entry-1' }, paymentsClient)

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    expect(wrapper.find('[role="alert"]').text()).toContain("can't tell what this entry owes")
  })

  // A free Competition is still free for a whole party — ExpectedAmount
  // multiplies zero by any number of heads and gets zero — and still
  // creates no Payment, since the Payments domain rejects a zero-amount one.
  it('creates no Payment when the frozen amount is zero, however many guests', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const wrapper = await mountWithQuery(
      { entryId: 'entry-1', amountOwedCents: '0', amountOwedCurrency: 'USD' },
      paymentsClient,
    )

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    expect(wrapper.get('[data-testid="free-competition-notice"]').text()).toContain('free')
  })
})
