import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useCourtBooking, QUOTE_STALE_MS } from '../useCourtBooking'
import type { BookingClient } from '../../api/bookingClient'

/**
 * A fake BookingClient whose POST/GET dispatch on the request path, since
 * (unlike useFacilityDetail's single-endpoint fakeClient) this composable
 * calls three different endpoints (GetQuote, CreateBooking,
 * ListCourtBookings).
 */
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

function bookingOk(id = 'b1') {
  return {
    data: {
      booking: {
        id,
        courtId: 'court-1',
        source: 'SOURCE_INDIVIDUAL',
        status: 'STATUS_CONFIRMED',
        startsAt: '2026-08-10T09:00:00Z',
        endsAt: '2026-08-10T10:00:00Z',
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

describe('useCourtBooking', () => {
  beforeEach(() => {
    vi.useRealTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('starts on the form step with no quote, error, or booking', () => {
    const { step, quote, quoteError, bookingError, conflict, confirmedBooking } = useCourtBooking(fakeClient({}))
    expect(step.value).toBe('form')
    expect(quote.value).toBeNull()
    expect(quoteError.value).toBeNull()
    expect(bookingError.value).toBeNull()
    expect(conflict.value).toBeNull()
    expect(confirmedBooking.value).toBeNull()
  })

  it('getQuote success moves to the review step with the mapped quote', async () => {
    const client = fakeClient({ postQuotes: () => quoteOk('1800', 'peak') })
    const { step, quote, getQuote } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')

    expect(step.value).toBe('review')
    expect(quote.value).toMatchObject({ priceCents: 1800, band: 'peak' })
    // `source` added T11.3: GetQuoteRequest gained it in T11.2 to select
    // which DiscountRules may apply, and this composable sends the same
    // SOURCE_INDIVIDUAL its own CreateBooking call uses rather than relying
    // on the unspecified-means-individual default.
    expect(client.POST).toHaveBeenCalledWith('/v1/quotes', {
      body: {
        courtId: 'court-1',
        startsAt: '2026-08-10T09:00:00Z',
        endsAt: '2026-08-10T10:00:00Z',
        source: 'SOURCE_INDIVIDUAL',
      },
    })
  })

  it('getQuote failure stays on the form step and sets quoteError', async () => {
    const client = fakeClient({ postQuotes: () => ({ data: undefined, error: { message: 'bad' }, response: { status: 400 } }) })
    const { step, quote, quoteError, getQuote } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')

    expect(step.value).toBe('form')
    expect(quote.value).toBeNull()
    expect(quoteError.value).toBeTruthy()
  })

  describe('confirm-step gate', () => {
    it('never calls CreateBooking if confirmBooking is called before a quote is reviewed', async () => {
      const client = fakeClient({ postBookings: () => bookingOk() })
      const { confirmBooking, step } = useCourtBooking(client)

      await confirmBooking()

      expect(step.value).toBe('form')
      expect(client.POST).not.toHaveBeenCalledWith('/v1/bookings', expect.anything())
    })
  })

  it('happy path: getQuote then confirmBooking books with SOURCE_INDIVIDUAL and moves to success', async () => {
    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingOk('booking-42'),
    })
    const { step, confirmedBooking, getQuote, confirmBooking } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
    await confirmBooking()

    expect(client.POST).toHaveBeenCalledWith('/v1/bookings', {
      body: {
        courtId: 'court-1',
        source: 'SOURCE_INDIVIDUAL',
        startsAt: '2026-08-10T09:00:00Z',
        endsAt: '2026-08-10T10:00:00Z',
      },
    })
    expect(step.value).toBe('success')
    expect(confirmedBooking.value?.id).toBe('booking-42')
  })

  it('re-fetches a stale quote instead of booking (never submits a price the server has not just confirmed)', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-01T00:00:00Z'))

    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingOk(),
    })
    const { step, quote, quoteRefreshedNotice, getQuote, confirmBooking } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
    const firstFetchedAt = quote.value?.fetchedAt

    vi.setSystemTime(new Date('2026-08-01T00:00:00Z').getTime() + QUOTE_STALE_MS + 1000)
    await confirmBooking()

    // Re-fetched the quote (POST /v1/quotes called twice), never called
    // CreateBooking.
    expect(client.POST).toHaveBeenNthCalledWith(1, '/v1/quotes', expect.anything())
    expect(client.POST).toHaveBeenNthCalledWith(2, '/v1/quotes', expect.anything())
    expect(client.POST).not.toHaveBeenCalledWith('/v1/bookings', expect.anything())
    expect(step.value).toBe('review')
    expect(quote.value?.fetchedAt).not.toBe(firstFetchedAt)
    expect(quoteRefreshedNotice.value).toBeTruthy()
  })

  it('does not re-fetch when the quote is fresh (within the staleness window)', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-01T00:00:00Z'))

    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingOk(),
    })
    const { step, getQuote, confirmBooking } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
    vi.setSystemTime(new Date('2026-08-01T00:00:00Z').getTime() + QUOTE_STALE_MS - 1000)
    await confirmBooking()

    expect(client.POST).toHaveBeenCalledTimes(2) // one quote, one booking
    expect(step.value).toBe('success')
  })

  describe('double-booking conflict', () => {
    it('surfaces the specific message and computed suggested slots, not just a generic error', async () => {
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
      const { step, conflict, bookingError, confirmedBooking, getQuote, confirmBooking } = useCourtBooking(client)

      await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
      await confirmBooking()

      expect(confirmedBooking.value).toBeNull()
      expect(bookingError.value).toBeNull() // conflict is its own state, not the generic bookingError
      expect(step.value).toBe('review') // stays on review so the Player can pick a suggestion
      expect(conflict.value?.message).toBe('This slot was just booked — pick another time')
      expect(conflict.value?.suggestionsLoading).toBe(false)
      expect(conflict.value?.suggestions.length).toBeGreaterThan(0)
      // The suggestion must not be the slot that just conflicted.
      expect(conflict.value?.suggestions[0]).not.toEqual({
        startsAt: '2026-08-10T09:00:00.000Z',
        endsAt: '2026-08-10T10:00:00.000Z',
      })

      expect(client.GET).toHaveBeenCalledWith('/v1/courts/{courtId}/bookings', expect.anything())
    })

    it('still surfaces the conflict message even if the ListCourtBookings suggestion lookup fails', async () => {
      const client = fakeClient({
        postQuotes: () => quoteOk('1800', 'peak'),
        postBookings: () => bookingConflict(),
        getBookings: () => {
          throw new TypeError('network down')
        },
      })
      const { conflict, getQuote, confirmBooking } = useCourtBooking(client)

      await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
      await confirmBooking()

      expect(conflict.value?.message).toBe('This slot was just booked — pick another time')
      expect(conflict.value?.suggestions).toEqual([])
    })
  })

  it('sets a generic bookingError for a non-409 CreateBooking failure', async () => {
    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
    })
    const { step, bookingError, conflict, getQuote, confirmBooking } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
    await confirmBooking()

    expect(step.value).toBe('review')
    expect(bookingError.value).toBeTruthy()
    expect(conflict.value).toBeNull()
  })

  it('backToForm clears conflict/error state and returns to the form step', async () => {
    const client = fakeClient({
      postQuotes: () => quoteOk('1800', 'peak'),
      postBookings: () => bookingConflict(),
      getBookings: () => ({ data: { bookings: [] }, error: undefined }),
    })
    const { step, conflict, getQuote, confirmBooking, backToForm } = useCourtBooking(client)

    await getQuote('court-1', '2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')
    await confirmBooking()
    expect(conflict.value).not.toBeNull()

    backToForm()

    expect(step.value).toBe('form')
    expect(conflict.value).toBeNull()
  })
})
