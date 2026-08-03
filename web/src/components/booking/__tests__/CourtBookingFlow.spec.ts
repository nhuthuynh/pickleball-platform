import { describe, it, expect, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import CourtBookingFlow from '../CourtBookingFlow.vue'
import type { BookingClient } from '../../../api/bookingClient'

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
})
