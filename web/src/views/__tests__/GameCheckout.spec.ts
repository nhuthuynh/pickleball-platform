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

// T56.1 (#126): the default query now carries the FROZEN owed amount
// alongside the Registration id, because that is what the join flow
// actually pushes (DiscoverGames.vue's `onPayOnline`) and what the view
// now charges. 1000 matches GAME_LISTING's entry fee — i.e. a player who
// brought no guests — so every test written before this ticket keeps
// exercising the same figure it always did.
async function mountCheckout(
  paymentsClient: PaymentsClient,
  query: Record<string, string> = { registrationId: 'reg-1', amountOwedCents: '1000', amountOwedCurrency: 'USD' },
) {
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
    await router.push({
      name: 'game-checkout',
      params: { id: 'g1' },
      query: { registrationId: 'reg-1', amountOwedCents: '1000', amountOwedCurrency: 'USD' },
    })
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

// T9.2: checkout charges a real fee rather than T8.10's flat placeholder,
// and a free Game creates no Payment at all (a zero-amount Payment is
// rejected by the Payments domain, so offering one would be a button that
// could only fail).
//
// T56.1 (#126) AMENDED these two tests rather than leaving them as written,
// and the amendment is worth stating because it changes what they prove.
// They used to mount a Game with a given entry_fee and assert the client
// charged THAT number, read off the Game. The client no longer reads the
// price off the Game at all: it charges the Registration's frozen
// AmountOwed, which for a player with no guests is the same number and for
// everyone else is a multiple of it. So each test now supplies the owed
// figure the way the join flow does, and the property each one pins is
// unchanged — a real fee is charged, and nothing is charged when nothing
// is owed. What is NOT retained is the old tests' implicit claim that the
// Game's entry fee is the thing charged; that claim is now false, and
// `GameCheckout — frozen per-head amount` below is where the replacement
// claim lives.
describe('GameCheckout — real entry fee (T9.2, amended T56.1)', () => {
  /** Mounts a Game whose entry fee is `amountCents`, with a Registration
   * owing `owedCents` (defaulting to the same — one head, no guests). */
  async function mountWithFee(amountCents: string, paymentsClient: PaymentsClient, owedCents = amountCents) {
    const listing = {
      ...GAME_LISTING,
      game: { ...GAME_LISTING.game, entryFee: { amountCents, currencyCode: 'USD' } },
    }
    const social = {
      GET: vi.fn(async () => ({ data: { games: [listing] }, error: undefined, response: { status: 200 } })),
      POST: vi.fn(),
    } as unknown as SocialPlayClient

    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({
      name: 'game-checkout',
      params: { id: 'g1' },
      query: { registrationId: 'reg-1', amountOwedCents: owedCents, amountOwedCurrency: 'USD' },
    })
    await router.isReady()
    const wrapper = mount(GameCheckout, {
      props: { client: social, paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()
    return wrapper
  }

  it('creates the Payment for a real fee, not a placeholder rate', async () => {
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

// T56.1/T56.2 (issues #126, #297): checkout charges the FROZEN per-head
// amount the Registration recorded, not a figure re-derived here.
//
// The two halves are one behaviour. #126 made the owed amount
// entry_fee × (1 + guest_count) and froze it onto the Registration; #297
// made the server validate every online payment against exactly that
// figure. So a client that keeps sending the bare per-player entry fee is
// not merely undercharging — as of #297 it is REFUSED, for every player
// who brought anyone. These tests pin the client to the frozen number.
describe('GameCheckout — frozen per-head amount (T56.1/T56.2)', () => {
  async function mountWithQuery(query: Record<string, string>, paymentsClient: PaymentsClient) {
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

  // The defect, stated as a test. GAME_LISTING's entry fee is 1000; a
  // player who brought two guests owes 3000. Before this, the client sent
  // 1000 and the other two heads were free.
  it('charges the amount the Registration owes, not the per-player entry fee', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountWithQuery(
      { registrationId: 'reg-1', amountOwedCents: '3000', amountOwedCurrency: 'USD' },
      paymentsClient,
    )

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls.length).toBe(1)
    expect(calls[0]![1].body.amount).toEqual({ amountCents: '3000', currencyCode: 'USD' })
  })

  // The currency travels with the amount (ADR-0005) rather than being
  // defaulted here — #297 compares both, so a right number in the wrong
  // currency is refused just as hard as a wrong number.
  it('sends the currency the Registration recorded', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    await mountWithQuery(
      { registrationId: 'reg-1', amountOwedCents: '3000', amountOwedCurrency: 'EUR' },
      paymentsClient,
    )

    const calls = (paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls
    expect(calls[0]![1].body.amount.currencyCode).toBe('EUR')
  })

  // The important refusal. Falling back to the Game's entry fee when the
  // owed amount is missing would be right only for a player who brought
  // nobody and silently wrong — now, server-refused — for everyone else.
  // A checkout that cannot know what is owed says so instead of guessing,
  // exactly as it already does for a Game it could not load.
  it('refuses to guess an amount when the owed figure is absent', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const wrapper = await mountWithQuery({ registrationId: 'reg-1' }, paymentsClient)

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    expect(wrapper.find('[role="alert"]').text()).toContain("can't tell what this registration owes")
  })

  // A free Game is still free for a whole party — ExpectedAmount multiplies
  // zero by any number of heads and gets zero — and still creates no
  // Payment, since the Payments domain rejects a zero-amount one.
  it('creates no Payment when the frozen amount is zero, however many guests', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const wrapper = await mountWithQuery(
      { registrationId: 'reg-1', amountOwedCents: '0', amountOwedCurrency: 'USD' },
      paymentsClient,
    )

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    expect(wrapper.get('[data-testid="free-game-notice"]').text()).toContain('free')
  })
})

// A hand-edited URL carrying a fractional cents value is treated as
// unknown, not sent. amountCents is an int64 on the wire, so a fraction
// would come back as a parse error the player can make no sense of —
// "we can't tell what this owes" is the honest answer, and it is the same
// one a missing value gets.
describe('GameCheckout — malformed owed amount', () => {
  it('refuses a non-integer cents value rather than sending it', async () => {
    const paymentsClient = paymentsClientStub({ createOnline: () => paymentOk('PAYMENT_STATUS_UNPAID') })
    const router = createRouter({ history: createMemoryHistory(), routes })
    await router.push({
      name: 'game-checkout',
      params: { id: 'g1' },
      query: { registrationId: 'reg-1', amountOwedCents: '25.5', amountOwedCurrency: 'USD' },
    })
    await router.isReady()
    const wrapper = mount(GameCheckout, {
      props: { client: socialplayClientStub(), paymentsClient },
      global: { plugins: [router] },
    })
    await flushPromises()

    expect((paymentsClient.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)
    expect(wrapper.find('[role="alert"]').text()).toContain("can't tell what this registration owes")
  })
})
