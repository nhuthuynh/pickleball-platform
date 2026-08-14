import { describe, it, expect, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import CourtBookingFlow from '../CourtBookingFlow.vue'
import type { BookingClient } from '../../../api/bookingClient'
import { expectNoA11yViolations, COMPONENT_MOUNT_OPTIONS } from '../../../test-support/axe'

function fakeClient(handlers: {
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

async function fillFormAndGetQuote(wrapper: VueWrapper) {
  await wrapper.find('input[type="date"]').setValue('2026-08-10')
  await wrapper.find('input[type="time"]').setValue('09:00')
  await wrapper.find('form').trigger('submit.prevent')
  await flush()
}

async function flush() {
  await Promise.resolve()
  await Promise.resolve()
}

describe('CourtBookingFlow', () => {
  it('confirm-step gate: the review/confirm step (and CreateBooking) never appears before a quote is fetched', async () => {
    const client = fakeClient({ postBookings: () => bookingOk() })
    const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })

    // No "Confirm booking" button exists yet — only the quote form.
    expect(wrapper.text()).not.toContain('Confirm booking')
    expect(wrapper.text()).not.toContain('Review your booking')
    expect(client.POST).not.toHaveBeenCalledWith('/v1/bookings', expect.anything())
  })

  it('happy path: quote -> review -> confirm -> booked, using SOURCE_INDIVIDUAL', async () => {
    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingOk('booking-99'),
    })
    const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', courtName: 'Court 1', client } })

    await fillFormAndGetQuote(wrapper)

    expect(wrapper.text()).toContain('Review your booking')
    expect(wrapper.text()).toContain('$18.00')
    expect(wrapper.text()).toContain('peak')
    // CreateBooking still must not have fired merely from reaching review.
    expect(client.POST).not.toHaveBeenCalledWith('/v1/bookings', expect.anything())

    const confirmButton = wrapper.findAll('button').find((b) => b.text().includes('Confirm booking'))
    expect(confirmButton).toBeTruthy()
    await confirmButton!.trigger('click')
    await flush()

    expect(client.POST).toHaveBeenCalledWith('/v1/bookings', {
      body: expect.objectContaining({ courtId: 'court-1', source: 'SOURCE_INDIVIDUAL' }),
    })
    expect(wrapper.text()).toContain('Booking confirmed')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
    expect(wrapper.find('[role="status"]').text()).toContain('booking-99')
  })

  it('double-booking conflict: shows the specific message and suggested slots, not just a generic error', async () => {
    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingConflict(),
      getBookings: () => ({
        data: {
          bookings: [
            {
              id: 'existing',
              courtId: 'court-1',
              status: 'STATUS_CONFIRMED',
              startsAt: '2026-08-10T09:00:00Z',
              endsAt: '2026-08-10T10:00:00Z',
            },
          ],
        },
        error: undefined,
      }),
    })
    const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })

    await fillFormAndGetQuote(wrapper)
    const confirmButton = wrapper.findAll('button').find((b) => b.text().includes('Confirm booking'))
    await confirmButton!.trigger('click')
    await flush()

    expect(wrapper.text()).toContain('This slot was just booked — pick another time')
    expect(wrapper.text()).not.toContain('Booking confirmed')
    const conflictAlert = wrapper.find('[role="alert"]')
    expect(conflictAlert.exists()).toBe(true)
    expect(conflictAlert.text()).toContain('This slot was just booked')

    // At least one suggested next-available-slot button rendered (not just
    // a rejection).
    const suggestionButtons = wrapper.findAll('.court-booking__suggestion')
    expect(suggestionButtons.length).toBeGreaterThan(0)
  })

  // ── T11.3: honest discounted-price display ────────────────────────────
  //
  // The requirement (t11-sprint-plan.md T11.3 instruction #2) is that a
  // discounted price is NEVER silently substituted for the band price — the
  // original must remain visible and labelled, the same discipline T8.10
  // applied to its placeholder fee. These tests assert the labelling in
  // TEXT, not via CSS, because a line-through style alone is exactly the
  // "no indication a discount was applied" failure the ticket forbids (and
  // is invisible to a screen reader).
  describe('discounted quote (T11.3)', () => {
    function quoteWithDiscount() {
      return {
        data: {
          priceCents: '1530',
          band: 'peak',
          bandPriceCents: '1800',
          discount: {
            id: 'discount-1',
            facilityId: 'facility-1',
            discountType: 'DISCOUNT_TYPE_PERCENT',
            percent: 15,
            appliesTo: ['SOURCE_INDIVIDUAL'],
            startsAt: '2026-08-01T00:00:00Z',
            endCondition: { kind: 'END_CONDITION_KIND_NO_END' },
          },
        },
        error: undefined,
        response: { status: 200 },
      }
    }

    it('shows the discounted price AND the original band price, both labelled in text', async () => {
      const client = fakeClient({ postQuotes: () => quoteWithDiscount() })
      const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })
      await fillFormAndGetQuote(wrapper)

      const text = wrapper.text()
      // The price actually charged.
      expect(text).toContain('$15.30')
      // The original is still on screen — not silently replaced.
      expect(text).toContain('$18.00')
      // ...and is identified as the original IN WORDS, not by styling alone.
      expect(text).toContain('Original price')
      expect(text).toContain('Discounted price')
      // The discount itself is described, so the Player can see WHY.
      expect(text).toContain('15% off')
      expect(text).toContain('You save $2.70')
    })

    it('marks up the struck-through original with <s>, but never relies on that alone', async () => {
      const client = fakeClient({ postQuotes: () => quoteWithDiscount() })
      const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })
      await fillFormAndGetQuote(wrapper)

      const struck = wrapper.find('.court-booking__original-price')
      expect(struck.exists()).toBe(true)
      expect(struck.element.tagName.toLowerCase()).toBe('s')
      expect(struck.text()).toContain('$18.00')

      // The <s> is decorative reinforcement: its meaning is carried by a
      // real <dt> label that is present in the text regardless.
      const labels = wrapper.findAll('dt').map((dt) => dt.text())
      expect(labels).toContain('Original price')
    })

    it('renders an END_AFTER_OCCURRENCES rule as a TOTAL, never as a remaining count', async () => {
      // The API has no remaining-count field (see models/discount.ts) — this
      // is the component-level guard for T11.3 instruction #3.
      const client = fakeClient({
        postQuotes: () => ({
          data: {
            priceCents: '1700',
            band: 'peak',
            bandPriceCents: '1800',
            discount: {
              id: 'discount-2',
              discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT',
              fixedAmountCents: '100',
              currency: 'USD',
              endCondition: { kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 10 },
            },
          },
          error: undefined,
          response: { status: 200 },
        }),
      })
      const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })
      await fillFormAndGetQuote(wrapper)

      expect(wrapper.text()).toContain('Ends after 10 total uses')
      expect(wrapper.text().toLowerCase()).not.toContain('remaining')
    })

    it('shows a plain single price with NO discount wording when none applied', async () => {
      const client = fakeClient({ postQuotes: () => quoteOk('1800', 'peak') })
      const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })
      await fillFormAndGetQuote(wrapper)

      const text = wrapper.text()
      expect(text).toContain('$18.00')
      expect(text).not.toContain('Original price')
      expect(text).not.toContain('Discounted price')
      expect(text).not.toContain('You save')
      expect(wrapper.find('.court-booking__original-price').exists()).toBe(false)
    })

    it('quotes with an explicit SOURCE_INDIVIDUAL, matching the booking it will create', async () => {
      // GetQuoteRequest gained `source` in T11.2 to select which rules may
      // apply. Every booking this flow creates is SOURCE_INDIVIDUAL
      // (confirmBooking hardcodes it), so the quote must ask about the same
      // source rather than leaning on the unspecified-means-individual
      // default.
      const client = fakeClient({ postQuotes: () => quoteOk('1800', 'peak') })
      const wrapper = mount(CourtBookingFlow, { props: { courtId: 'court-1', client } })
      await fillFormAndGetQuote(wrapper)

      expect(client.POST).toHaveBeenCalledWith('/v1/quotes', {
        body: expect.objectContaining({ source: 'SOURCE_INDIVIDUAL' }),
      })
    })

    it('has no automated a11y violations while showing a discounted quote', async () => {
      const client = fakeClient({ postQuotes: () => quoteWithDiscount() })
      const wrapper = mount(CourtBookingFlow, {
        props: { courtId: 'court-1', client },
        attachTo: document.body,
      })
      try {
        await fillFormAndGetQuote(wrapper)
        await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
      } finally {
        wrapper.unmount()
      }
    })
  })
})
