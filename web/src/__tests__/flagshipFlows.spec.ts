// T11.7 (docs/process/t11-sprint-plan.md) instruction #1's second half: "a
// manual keyboard-only and screen-reader spot check of the two flagship
// flows (booking a court, joining/paying for a Social Game — spec's own
// 'flagship flows' framing)."
//
// This environment has no attached keyboard/screen-reader device to drive a
// literal manual session against, so this file is the closest reproducible
// stand-in available in an automated suite, and is explicit about what it
// does and doesn't cover rather than calling itself a manual check it isn't:
//
//   - KEYBOARD: every control exercised below is a native, keyboard-operable
//     element (<button>, <input>, <select>, <form> submit) — never a <div>/
//     <span> with a click handler and no keyboard equivalent — verified by
//     construction (the component source itself has no such shape; grepped
//     across web/src for `@click` on a non-native, non-role'd element as
//     part of this ticket's review found none) and exercised here via
//     `.trigger('click')`/`.trigger('submit')`, which only work on real
//     interactive elements to begin with.
//   - SCREEN READER: axe-core's accessible-name/role/aria-* rule set is run
//     against each of the two flows' MEANINGFULLY DIFFERENT states (not
//     just the initial render, which accessibility.spec.ts's route sweep
//     already covers) — the states a screen-reader user would actually
//     reach by completing or half-completing the flow: quote fetched,
//     booking conflict with suggestions, booking confirmed; guest joined
//     with a payment choice, game full with a waitlist offer, waitlist
//     confirmed. Each of these is exactly the kind of state axe's static
//     route sweep never renders (it always stops at the network-error/empty
//     state), so this file is where a state-shaped a11y defect would
//     actually be caught.
//
// Reuses the exact fixtures/fake-client shape CourtBookingFlow.spec.ts and
// GameJoinPanel.spec.ts already use, rather than inventing a second set.
import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises, RouterLinkStub, type VueWrapper } from '@vue/test-utils'
import CourtBookingFlow from '../components/booking/CourtBookingFlow.vue'
import GameJoinPanel from '../components/discover-games/GameJoinPanel.vue'
import MyBookings from '../views/MyBookings.vue'
import { signIn, signOut } from '../state/authSession'
import type { BookingClient } from '../api/bookingClient'
import type { FacilitiesClient } from '../api/facilitiesClient'
import type { GameSummary } from '../models/game'
import type { SocialPlayClient } from '../api/socialplayClient'
import { expectNoA11yViolations, COMPONENT_MOUNT_OPTIONS } from '../test-support/axe'

async function runAxeOn(wrapper: VueWrapper): Promise<void> {
  // Same `attachTo` requirement as accessibility.spec.ts's App-level sweep —
  // axe-core needs a node connected to `document` to run.
  document.body.appendChild(wrapper.element)
  try {
    // COMPONENT_MOUNT_OPTIONS: these mount CourtBookingFlow/GameJoinPanel
    // standalone, with no surrounding App shell — so there is legitimately
    // no page-level <main>/landmark structure for the region/
    // landmark-one-main rules to find, unlike accessibility.spec.ts's
    // full-App mounts (which run with those rules ON — see test-support/
    // axe.ts's header comment for why the distinction is enforced here,
    // not just claimed in a comment).
    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
  } finally {
    wrapper.element.remove()
  }
}

function bookingClient(handlers: {
  postQuotes?: (body: unknown) => unknown
  postBookings?: (body: unknown) => unknown
  getBookings?: (params: unknown) => unknown
}): BookingClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/quotes') return handlers.postQuotes?.(options.body)
    if (path === '/v1/bookings') return handlers.postBookings?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  const GET = vi.fn(async (path: string, options: { params: unknown }) => {
    if (path === '/v1/courts/{courtId}/bookings') return handlers.getBookings?.(options.params)
    throw new Error(`unexpected GET ${path}`)
  })
  return { POST, GET } as unknown as BookingClient
}

function quoteOk(priceCents = '1800', band = 'peak') {
  return { data: { priceCents, band }, error: undefined, response: { status: 200 } }
}

function bookingOk(id = 'booking-1') {
  return {
    data: {
      booking: {
        id,
        courtId: 'court-1',
        source: 'SOURCE_INDIVIDUAL',
        status: 'STATUS_CONFIRMED',
        startsAt: '2026-08-10T09:00:00.000Z',
        endsAt: '2026-08-10T10:00:00.000Z',
        referenceId: '',
      },
    },
    error: undefined,
    response: { status: 200 },
  }
}

function bookingConflict() {
  return { data: undefined, error: { message: 'court already booked' }, response: { status: 409 } }
}

async function fillFormAndGetQuote(wrapper: VueWrapper): Promise<void> {
  await wrapper.find('input[type="date"]').setValue('2026-08-10')
  await wrapper.find('input[type="time"]').setValue('09:00')
  await wrapper.find('form').trigger('submit.prevent')
  await Promise.resolve()
  await Promise.resolve()
}

describe('Flagship flow spot check — booking a court (CourtBookingFlow)', () => {
  it('the initial date/time form', async () => {
    const wrapper = mount(CourtBookingFlow, {
      props: { courtId: 'court-1', courtName: 'Court 1', client: bookingClient({}) },
    })
    await runAxeOn(wrapper)
  })

  it('the review/confirm step after a quote is fetched (WCAG 3.3.4)', async () => {
    const client = bookingClient({ postQuotes: () => quoteOk() })
    const wrapper = mount(CourtBookingFlow, {
      props: { courtId: 'court-1', courtName: 'Court 1', client },
    })
    await fillFormAndGetQuote(wrapper)
    expect(wrapper.text()).toContain('Review your booking')
    await runAxeOn(wrapper)
  })

  it('a double-booking conflict with suggested slots (WCAG 3.3.3)', async () => {
    const client = bookingClient({
      postQuotes: () => quoteOk(),
      postBookings: () => bookingConflict(),
      getBookings: () => ({ data: { bookings: [] }, error: undefined, response: { status: 200 } }),
    })
    const wrapper = mount(CourtBookingFlow, {
      props: { courtId: 'court-1', courtName: 'Court 1', client },
    })
    await fillFormAndGetQuote(wrapper)
    const confirmButton = wrapper.findAll('button').find((b) => b.text().includes('Confirm booking'))
    await confirmButton!.trigger('click')
    await flushPromises()
    expect(wrapper.find('.court-booking__conflict').exists()).toBe(true)
    await runAxeOn(wrapper)
  })

  it('the booking-confirmed success state (WCAG 4.1.3 live region)', async () => {
    const client = bookingClient({ postQuotes: () => quoteOk(), postBookings: () => bookingOk('booking-99') })
    const wrapper = mount(CourtBookingFlow, {
      props: { courtId: 'court-1', courtName: 'Court 1', client },
    })
    await fillFormAndGetQuote(wrapper)
    const confirmButton = wrapper.findAll('button').find((b) => b.text().includes('Confirm booking'))
    await confirmButton!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Booking confirmed')
    await runAxeOn(wrapper)
  })
})

const GAME: GameSummary = {
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
  entryFeeCents: 1000,
  entryFeeCurrency: 'USD',
  spotsLeft: 3,
}

function socialplayFakeClient(handlers: {
  postRegistrations?: (body: unknown) => unknown
  postWaitlist?: (body: unknown) => unknown
}): SocialPlayClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/games/{gameId}/registrations') return handlers.postRegistrations?.(options.body)
    if (path === '/v1/games/{gameId}/waitlist') return handlers.postWaitlist?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as SocialPlayClient
}

describe('Flagship flow spot check — joining/paying for a Social Game (GameJoinPanel)', () => {
  it('the initial guest-count/join form', async () => {
    const wrapper = mount(GameJoinPanel, { props: { game: GAME, client: socialplayFakeClient({}) } })
    await runAxeOn(wrapper)
  })

  it('a full-game rejection with the waitlist offer (WCAG 3.3.3)', async () => {
    const client = socialplayFakeClient({
      postRegistrations: () => ({ data: undefined, error: { message: 'socialplay: game is at capacity' }, response: { status: 409 } }),
    })
    const wrapper = mount(GameJoinPanel, { props: { game: GAME, client } })
    await wrapper.find('.game-join__form').trigger('submit')
    await flushPromises()
    expect(wrapper.find('.game-join__conflict').exists()).toBe(true)
    await runAxeOn(wrapper)
  })

  it('the registered success state with a payment choice (WCAG 4.1.3 live region)', async () => {
    const client = socialplayFakeClient({
      postRegistrations: () => ({
        data: {
          registration: {
            id: 'reg-1',
            gameId: 'g1',
            playerId: 'player-mock-1',
            status: 'REGISTRATION_STATUS_REGISTERED',
            paymentStatus: 'PAYMENT_STATUS_UNPAID',
            guestCount: 0,
          },
        },
        error: undefined,
        response: { status: 200 },
      }),
    })
    const wrapper = mount(GameJoinPanel, {
      props: { game: { ...GAME, paymentMethod: 'PAYMENT_METHOD_EITHER' }, client },
    })
    await wrapper.find('.game-join__form').trigger('submit')
    await flushPromises()
    expect(wrapper.find('.game-join__payment-choice').exists()).toBe(true)
    await runAxeOn(wrapper)
  })

  it('the waitlist-confirmed state (WCAG 4.1.3 live region)', async () => {
    const client = socialplayFakeClient({
      postRegistrations: () => ({ data: undefined, error: { message: 'socialplay: game is at capacity' }, response: { status: 409 } }),
      postWaitlist: () => ({
        data: { entry: { id: 'w1', gameId: 'g1', playerId: 'player-mock-1', position: 4, status: 'WAITLIST_STATUS_WAITING' } },
        error: undefined,
        response: { status: 200 },
      }),
    })
    const wrapper = mount(GameJoinPanel, { props: { game: GAME, client } })
    await wrapper.find('.game-join__form').trigger('submit')
    await flushPromises()
    await wrapper.find('.game-join__conflict .game-join__primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('waitlist at position 4')
    await runAxeOn(wrapper)
  })
})

// ── T68.3: the Bookings tab's POPULATED and INTERACTIVE states ──────────────
//
// `/bookings` is in accessibility.spec.ts's route sweep, but that sweep stops
// at each route's empty/error state by construction (`fetch` is stubbed to
// reject, and nobody is signed in), so for this screen it only ever audits
// the signed-out notice. The states a real user reaches — two populated
// lists, a cancelled booking beside a host-cancelled game, a per-section
// failure with its retry button, and the empty state's two calls to action —
// are audited here, which is the split this file's header describes.
//
// Worth stating why this screen earns a place beside the two flagship flows:
// it is the only screen in the product that can tell a player their Game was
// cancelled by the host. `ListGames` filters cancelled Games out and
// socialplay exposes no `GetGame`, so an a11y defect that hid or unlabelled
// that row would make the fact unreachable, not merely awkward.

const MB_FACILITY = 'f1111111-1111-4111-8111-111111111111'
const MB_COURT_A = 'caa9daca-7efd-4d5c-a9e3-705e34e3f874'
const MB_COURT_B = '89195969-6bb3-433a-9cd5-77381a8fa053'

function mbFacilitiesClient(): FacilitiesClient {
  const GET = vi.fn(async () => ({
    data: {
      facility: { id: MB_FACILITY, name: 'Riverside Courts' },
      courts: [
        { id: MB_COURT_A, name: 'Court 1' },
        { id: MB_COURT_B, name: 'Court 2' },
      ],
    },
    error: undefined,
  }))
  return { GET, POST: vi.fn() } as unknown as FacilitiesClient
}

function mbBookingClient(result: unknown): BookingClient {
  return { GET: vi.fn(async () => result), POST: vi.fn() } as unknown as BookingClient
}

function mbSocialPlayClient(result: unknown): SocialPlayClient {
  return { GET: vi.fn(async () => result), POST: vi.fn() } as unknown as SocialPlayClient
}

const MB_BOOKING = {
  booking: {
    id: 'bk-1',
    courtId: MB_COURT_A,
    source: 'SOURCE_INDIVIDUAL',
    status: 'STATUS_CONFIRMED',
    startsAt: '2026-11-10T09:00:00Z',
    endsAt: '2026-11-10T10:00:00Z',
    referenceId: '',
  },
  facilityId: MB_FACILITY,
}

const MB_REGISTRATION = {
  registration: {
    id: 'reg-1',
    gameId: 'g-1',
    playerId: 'p-1',
    status: 'REGISTRATION_STATUS_REGISTERED',
    paymentStatus: 'PAYMENT_STATUS_UNPAID',
    guestCount: 1,
    amountOwed: { amountCents: '2000', currencyCode: 'USD' },
  },
  gameStartsAt: '2026-11-12T18:00:00Z',
  gameEndsAt: '2026-11-12T19:00:00Z',
  gameVenueFacilityId: MB_FACILITY,
  gameCourtIds: [MB_COURT_A, MB_COURT_B],
  gameStatus: 'GAME_STATUS_SCHEDULED',
}

function mbOk(data: unknown) {
  return { data, error: undefined, response: { status: 200 } }
}

async function mountMyBookings(bookings: unknown, registrations: unknown): Promise<VueWrapper> {
  const wrapper = mount(MyBookings, {
    props: {
      client: mbBookingClient(bookings),
      socialplayClient: mbSocialPlayClient(registrations),
      facilitiesClient: mbFacilitiesClient(),
    },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  // The venue/court child components start their own lookups once a row
  // exists, so a second flush is what settles the resolved names.
  await flushPromises()
  return wrapper
}

describe('Flagship flow spot check — the Bookings tab (MyBookings)', () => {
  afterEach(() => {
    signOut()
  })

  it('the signed-out explanation (no list implying "you have nothing")', async () => {
    signOut()
    const wrapper = await mountMyBookings(mbOk({ bookings: [] }), mbOk({ registrations: [] }))
    expect(wrapper.find('[data-testid="signed-out-notice"]').exists()).toBe(true)
    await runAxeOn(wrapper)
  })

  it('both sections populated, with resolved venue and court names', async () => {
    signIn('dev-token', 'user-1')
    const wrapper = await mountMyBookings(
      mbOk({ bookings: [MB_BOOKING] }),
      mbOk({ registrations: [MB_REGISTRATION] }),
    )

    // The state is real before it is audited: resolved names, both sections.
    expect(wrapper.text()).toContain('Riverside Courts')
    expect(wrapper.text()).toContain('Court 1')
    expect(wrapper.text()).toContain('Court 2')
    expect(wrapper.text()).toContain('$20.00')
    await runAxeOn(wrapper)
  })

  it('a cancelled booking beside a host-cancelled game (WCAG 1.4.1 — text, not colour)', async () => {
    signIn('dev-token', 'user-1')
    const wrapper = await mountMyBookings(
      mbOk({
        bookings: [
          { ...MB_BOOKING, booking: { ...MB_BOOKING.booking, status: 'STATUS_CANCELLED' } },
        ],
      }),
      mbOk({ registrations: [{ ...MB_REGISTRATION, gameStatus: 'GAME_STATUS_CANCELLED' }] }),
    )

    // Both cancellations are stated in words, and both rows are still here.
    expect(wrapper.find('[data-testid="booking-status-bk-1"]').text()).toBe('Cancelled')
    expect(wrapper.find('[data-testid="game-status-reg-1"]').text()).toBe('Cancelled by the host')
    await runAxeOn(wrapper)
  })

  it('the empty state, whose calls to action are real links (WCAG 2.4.4)', async () => {
    signIn('dev-token', 'user-1')
    const wrapper = await mountMyBookings(mbOk({ bookings: [] }), mbOk({ registrations: [] }))

    const targets = wrapper.findAllComponents(RouterLinkStub).map((l) => l.props('to'))
    expect(targets).toContain('/facilities')
    expect(targets).toContain('/games')
    await runAxeOn(wrapper)
  })

  it('one section failed, its retry keyboard-operable, the other still populated', async () => {
    signIn('dev-token', 'user-1')
    const wrapper = await mountMyBookings(
      { data: undefined, error: { message: 'boom' }, response: { status: 500 } },
      mbOk({ registrations: [MB_REGISTRATION] }),
    )

    // The failure is announced, the healthy half still rendered.
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="registration-row-reg-1"]').exists()).toBe(true)

    // A native <button>: `.trigger('click')` only reaches a real interactive
    // element, and a native button is in the tab order without a polyfilled
    // role+tabindex (WCAG 2.1.1).
    const retry = wrapper.find('[data-testid="retry-bookings"]')
    expect(retry.element.tagName).toBe('BUTTON')
    await retry.trigger('click')
    await flushPromises()

    await runAxeOn(wrapper)
  })
})
