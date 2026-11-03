// T68.3 (part of #353): the Bookings tab — "what did I book, and what did I
// join".
//
// T67 shipped both actor-scoped reads and cut the screen, so `/bookings`
// pointed at `ComingSoonView` and rendered literally `Coming soon.` while
// `AppNav` advertised Bookings as a top-level tab. This spec is the
// behavioural half of replacing it.
//
// Three properties carry most of the weight here, because all three are
// failure modes the protos call out by name:
//
//   1. **Nothing is filtered.** A cancelled booking, a host-cancelled game,
//      and a booking with no facility all RENDER, with an honest label.
//   2. **No raw uuid reaches the user.** #342 is closed and must stay
//      closed, so there is a property test over the whole rendered text
//      rather than per-field spot checks.
//   3. **The two reads are independent.** One failing read leaves the other
//      section rendered.
//
// `facilitiesClient` is injected and shared by VenueName/CourtNames, the same
// way GameDetailPanel.spec.ts does it — nothing here stubs ambient `fetch`.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, RouterLinkStub, type VueWrapper } from '@vue/test-utils'

import MyBookings from '../MyBookings.vue'
import { signIn, signOut } from '../../state/authSession'
import type { BookingClient } from '../../api/bookingClient'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'

const COURT_1 = 'caa9daca-7efd-4d5c-a9e3-705e34e3f874'
const COURT_3 = '89195969-6bb3-433a-9cd5-77381a8fa053'
const COURT_4 = '0b3a6c19-9d0e-4b60-8a1f-2d7c5e8f1a44'
const FACILITY_1 = 'f1111111-1111-4111-8111-111111111111'
const FACILITY_2 = 'f2222222-2222-4222-8222-222222222222'

function bookingRow(overrides: Record<string, unknown> = {}) {
  return {
    booking: {
      id: 'bk-1',
      courtId: COURT_1,
      source: 'SOURCE_INDIVIDUAL',
      status: 'STATUS_CONFIRMED',
      startsAt: '2026-11-10T09:00:00Z',
      endsAt: '2026-11-10T10:00:00Z',
      referenceId: '',
      ...((overrides.booking as Record<string, unknown>) ?? {}),
    },
    facilityId: FACILITY_1,
    ...overrides,
  }
}

function registrationRow(overrides: Record<string, unknown> = {}) {
  return {
    registration: {
      id: 'reg-1',
      gameId: 'g-1',
      playerId: 'p-1',
      status: 'REGISTRATION_STATUS_REGISTERED',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      guestCount: 0,
      amountOwed: { amountCents: '2000', currencyCode: 'USD' },
      ...((overrides.registration as Record<string, unknown>) ?? {}),
    },
    gameStartsAt: '2026-11-12T18:00:00Z',
    gameEndsAt: '2026-11-12T19:00:00Z',
    gameVenueFacilityId: FACILITY_2,
    gameCourtIds: [COURT_3],
    gameStatus: 'GAME_STATUS_SCHEDULED',
    ...overrides,
  }
}

function ok(data: unknown) {
  return { data, error: undefined, response: { status: 200 } }
}
function fail(status = 500) {
  return { data: undefined, error: { message: 'boom' }, response: { status } }
}

function bookingClient(result: unknown): BookingClient {
  return { GET: vi.fn(async () => result), POST: vi.fn() } as unknown as BookingClient
}
function socialplayClient(result: unknown): SocialPlayClient {
  return { GET: vi.fn(async () => result), POST: vi.fn() } as unknown as SocialPlayClient
}

/** One GetFacility response per facility id, so VenueName and CourtNames
 *  resolve real names for both sections' rows. */
function facilitiesClient(): FacilitiesClient {
  const byId: Record<string, unknown> = {
    [FACILITY_1]: {
      facility: { id: FACILITY_1, name: 'Riverside Courts' },
      courts: [{ id: COURT_1, name: 'Court 1' }],
    },
    [FACILITY_2]: {
      facility: { id: FACILITY_2, name: 'Eastside Club' },
      courts: [
        { id: COURT_3, name: 'Court 3' },
        { id: COURT_4, name: 'Court 4' },
      ],
    },
  }
  const GET = vi.fn(async (_path: string, options: { params: { path: { facilityId: string } } }) => {
    const found = byId[options.params.path.facilityId]
    return found ? { data: found, error: undefined } : { data: undefined, error: { message: '404' } }
  })
  return { GET, POST: vi.fn() } as unknown as FacilitiesClient
}

async function mountScreen(
  bookings: unknown,
  registrations: unknown,
): Promise<VueWrapper> {
  const wrapper = mount(MyBookings, {
    props: {
      client: bookingClient(bookings),
      socialplayClient: socialplayClient(registrations),
      facilitiesClient: facilitiesClient(),
    },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  // VenueName/CourtNames each start their own lookup once the row exists, so
  // a second flush is what lets those resolve.
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  signIn('dev-token', 'user-1')
})

afterEach(() => {
  signOut()
})

describe('MyBookings — the signed-in, populated screen', () => {
  it('renders both sections with date, time, venue name and court name', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [bookingRow()] }),
      ok({ registrations: [registrationRow()] }),
    )

    expect(wrapper.find('[data-testid="court-bookings-section"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="games-section"]').exists()).toBe(true)

    const text = wrapper.text()
    // Date and time, from the real timestamps.
    expect(text).toContain('2026')
    // Venue names, resolved through useFacilityName (VenueName.vue).
    expect(text).toContain('Riverside Courts')
    expect(text).toContain('Eastside Club')
    // Court names, resolved through useCourtNames (CourtNames.vue).
    expect(text).toContain('Court 1')
    expect(text).toContain('Court 3')
  })

  // #342 is closed and must stay closed. Asserted over the WHOLE rendered
  // text as a property, not per field: a future row that adds an id-bearing
  // field fails this without anyone remembering to add a case.
  it('renders no raw uuid anywhere', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [bookingRow()] }),
      ok({ registrations: [registrationRow({ gameCourtIds: [COURT_3, COURT_4] })] }),
    )

    const UUID = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i
    expect(wrapper.text()).not.toMatch(UUID)
  })

  // The ticket's cardinality check: `RegistrationListing.game_court_ids` is
  // REPEATED, and a multi-court game is normal in socialplay.
  it('renders every court of a multi-court game', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [] }),
      ok({ registrations: [registrationRow({ gameCourtIds: [COURT_3, COURT_4] })] }),
    )

    const text = wrapper.text()
    expect(text).toContain('Court 3')
    expect(text).toContain('Court 4')
  })

  // Requirement 7: show the amount owed, and only for unpaid + non-zero.
  it('shows the amount owed for an unpaid registration', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [] }),
      ok({ registrations: [registrationRow()] }),
    )

    expect(wrapper.text()).toContain('Unpaid')
    expect(wrapper.text()).toContain('$20.00')
  })

  it('shows no amount owed once paid', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [] }),
      ok({
        registrations: [
          registrationRow({ registration: { paymentStatus: 'PAYMENT_STATUS_PAID' } }),
        ],
      }),
    )

    expect(wrapper.text()).toContain('Paid')
    expect(wrapper.text()).not.toContain('$20.00')
  })

  // Requirement 7's "do not fabricate": an UNSPECIFIED payment status is not
  // "Unpaid", and carries no amount.
  it('does not claim unpaid when the read supplied no payment status', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [] }),
      ok({
        registrations: [
          registrationRow({ registration: { paymentStatus: 'PAYMENT_STATUS_UNSPECIFIED' } }),
        ],
      }),
    )

    expect(wrapper.text()).toContain('Payment status not recorded')
    expect(wrapper.text()).not.toContain('$20.00')
  })
})

describe('MyBookings — rows that must render rather than disappear', () => {
  // ListBookingsForActor deliberately has no status filter so a client cannot
  // hide a cancellation.
  it('renders a cancelled booking, labelled as cancelled', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [bookingRow({ booking: { status: 'STATUS_CANCELLED' } })] }),
      ok({ registrations: [] }),
    )

    expect(wrapper.findAll('[data-testid^="booking-row-"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Cancelled')
  })

  // The one fact a player cannot learn anywhere else: ListGames filters
  // cancelled Games out and socialplay has no GetGame RPC.
  it('renders a host-cancelled game, labelled as cancelled by the host', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [] }),
      ok({ registrations: [registrationRow({ gameStatus: 'GAME_STATUS_CANCELLED' })] }),
    )

    expect(wrapper.findAll('[data-testid^="registration-row-"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Cancelled by the host')
  })

  // ActorBooking.facility_id: "Empty is a real, expected answer... A client
  // must render such a booking, not hide it — the booking is real."
  it('renders a booking whose facility is empty, with an honest venue label', async () => {
    const wrapper = await mountScreen(
      ok({ bookings: [bookingRow({ facilityId: '' })] }),
      ok({ registrations: [] }),
    )

    expect(wrapper.findAll('[data-testid^="booking-row-"]')).toHaveLength(1)
    // An honest label, not a uuid and not a fabricated venue.
    expect(wrapper.text()).toContain('No venue on record')
    expect(wrapper.text()).not.toContain(COURT_1)
  })
})

describe('MyBookings — empty states', () => {
  // Requirement 4: name the action that would fill the section, and link to it.
  it('names and links the action that would fill each empty section', async () => {
    const wrapper = await mountScreen(ok({ bookings: [] }), ok({ registrations: [] }))

    const text = wrapper.text()
    expect(text).toContain('have not booked a court')
    expect(text).toContain('have not joined any games')

    const targets = wrapper.findAllComponents(RouterLinkStub).map((l) => l.props('to'))
    expect(targets).toContain('/facilities')
    expect(targets).toContain('/games')
  })
})

describe('MyBookings — one failing read must not blank the screen', () => {
  it('reports the bookings failure and still renders the games section', async () => {
    const wrapper = await mountScreen(fail(), ok({ registrations: [registrationRow()] }))

    const alerts = wrapper.findAll('[role="alert"]').map((a) => a.text())
    expect(alerts.join(' ')).toContain('court bookings')
    // The other half rendered for real, not as an empty shell.
    expect(wrapper.findAll('[data-testid^="registration-row-"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Eastside Club')
  })

  it('reports the games failure and still renders the bookings section', async () => {
    const wrapper = await mountScreen(ok({ bookings: [bookingRow()] }), fail())

    const alerts = wrapper.findAll('[role="alert"]').map((a) => a.text())
    expect(alerts.join(' ')).toContain('games')
    expect(wrapper.findAll('[data-testid^="booking-row-"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Riverside Courts')
  })

  // A per-section error needs a per-section retry, and that retry must be a
  // real, labelled, keyboard-operable control.
  it('offers a retry on the failed section that re-reads only that section', async () => {
    const booking = bookingClient(fail())
    const socialplay = socialplayClient(ok({ registrations: [registrationRow()] }))
    const wrapper = mount(MyBookings, {
      props: { client: booking, socialplayClient: socialplay, facilitiesClient: facilitiesClient() },
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    const retry = wrapper.find('[data-testid="retry-bookings"]')
    expect(retry.exists()).toBe(true)
    expect(retry.element.tagName).toBe('BUTTON')
    expect(retry.text().length).toBeGreaterThan(0)

    await retry.trigger('click')
    await flushPromises()

    expect(booking.GET).toHaveBeenCalledTimes(2)
    // The healthy section was not re-requested.
    expect(socialplay.GET).toHaveBeenCalledTimes(1)
  })
})

describe('MyBookings — signed out', () => {
  // Requirement 6: an empty list would imply "you have nothing", which is a
  // different and false statement.
  it('explains the signed-out state and links to sign-in instead of listing nothing', async () => {
    signOut()
    const booking = bookingClient(ok({ bookings: [] }))
    const socialplay = socialplayClient(ok({ registrations: [] }))
    const wrapper = mount(MyBookings, {
      props: { client: booking, socialplayClient: socialplay, facilitiesClient: facilitiesClient() },
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="signed-out-notice"]').exists()).toBe(true)
    const targets = wrapper.findAllComponents(RouterLinkStub).map((l) => l.props('to'))
    expect(targets).toContain('/sign-in')

    // No list, no empty state, and — the point — no read attempted: these are
    // actor-scoped reads and there is no actor.
    expect(wrapper.find('[data-testid="court-bookings-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="games-section"]').exists()).toBe(false)
    expect(booking.GET).not.toHaveBeenCalled()
    expect(socialplay.GET).not.toHaveBeenCalled()
  })
})

describe('MyBookings — structure the route sweep depends on', () => {
  // accessibility.spec.ts asserts exactly one <h1> per route. Pinned here as
  // well so a failure points at this component rather than at the sweep.
  it('renders exactly one top-level heading in every state', async () => {
    const populated = await mountScreen(
      ok({ bookings: [bookingRow()] }),
      ok({ registrations: [registrationRow()] }),
    )
    expect(populated.findAll('h1')).toHaveLength(1)

    const failed = await mountScreen(fail(), fail())
    expect(failed.findAll('h1')).toHaveLength(1)

    signOut()
    const signedOut = mount(MyBookings, {
      props: {
        client: bookingClient(ok({})),
        socialplayClient: socialplayClient(ok({})),
        facilitiesClient: facilitiesClient(),
      },
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()
    expect(signedOut.findAll('h1')).toHaveLength(1)
  })
})
