import { ref, type Ref } from 'vue'
import { bookingClient, type BookingClient } from '../api/bookingClient'
import {
  mapToQuote,
  mapToBooking,
  computeNextAvailableSlots,
  type Quote,
  type ConfirmedBooking,
  type BookingSlot,
} from '../models/booking'

/**
 * How stale a displayed quote is allowed to be before `confirmBooking`
 * silently refuses to book against it and re-fetches instead (T7.6
 * non-functional requirement: never submit a price the server hasn't just
 * confirmed — a Quote can change between viewing and confirming).
 */
export const QUOTE_STALE_MS = 60_000

export type BookingStep = 'form' | 'review' | 'success'

export interface ConflictInfo {
  message: string
  suggestions: BookingSlot[]
  suggestionsLoading: boolean
}

export interface UseCourtBookingResult {
  step: Ref<BookingStep>
  quote: Ref<Quote | null>
  quoteLoading: Ref<boolean>
  /** Human-readable message when `GetQuote` itself failed. */
  quoteError: Ref<string | null>
  bookingLoading: Ref<boolean>
  /** Human-readable message for a `CreateBooking` failure that is *not* the
   * double-booking conflict (that has its own `conflict` state below). */
  bookingError: Ref<string | null>
  /** Set when `CreateBooking` comes back 409 (`ErrCourtDoubleBooked`) — the
   * specific message plus computed next-available-slot suggestions (WCAG
   * 3.3.3 Error Suggestion). */
  conflict: Ref<ConflictInfo | null>
  confirmedBooking: Ref<ConfirmedBooking | null>
  /** Set when `confirmBooking` silently refreshed a stale quote instead of
   * booking, so the UI can tell the Player why nothing happened yet. */
  quoteRefreshedNotice: Ref<string | null>
  getQuote: (courtId: string, startsAt: string, endsAt: string) => Promise<void>
  confirmBooking: () => Promise<void>
  backToForm: () => void
  reset: () => void
}

/**
 * Drives the quote -> review/confirm -> book flow (T7.6): `GetQuote`,
 * `CreateBooking` (always with `source: SOURCE_INDIVIDUAL` — the only
 * source a bare Player-initiated booking can legitimately use, per
 * booking.proto's `Source` doc comment), and — only on a 409 double-booking
 * conflict — `ListCourtBookings` to compute suggested next slots.
 *
 * `client` is injectable (defaults to the real `bookingClient`), same
 * pattern as `useFacilityDetail`/`useFacilityList`.
 *
 * The **confirm-step gate** (WCAG 3.3.4 Error Prevention) lives here, not
 * just in the UI: `confirmBooking` is a no-op unless `step` is already
 * `'review'` with a fetched `quote` — `CreateBooking` can never fire before
 * a quote has been reviewed, regardless of what the calling component does.
 */
export function useCourtBooking(client: BookingClient = bookingClient): UseCourtBookingResult {
  const step = ref<BookingStep>('form')
  const quote = ref<Quote | null>(null)
  const quoteLoading = ref(false)
  const quoteError = ref<string | null>(null)
  const bookingLoading = ref(false)
  const bookingError = ref<string | null>(null)
  const conflict = ref<ConflictInfo | null>(null)
  const confirmedBooking = ref<ConfirmedBooking | null>(null)
  const quoteRefreshedNotice = ref<string | null>(null)

  // The most recently quoted slot — needed by confirmBooking (to re-fetch a
  // stale quote against the same slot) and by the conflict-suggestion
  // search (to know the duration/window to search around).
  let lastCourtId = ''
  let lastStartsAt = ''
  let lastEndsAt = ''

  async function getQuote(courtId: string, startsAt: string, endsAt: string): Promise<void> {
    lastCourtId = courtId
    lastStartsAt = startsAt
    lastEndsAt = endsAt
    quoteLoading.value = true
    quoteError.value = null
    conflict.value = null
    bookingError.value = null
    try {
      const { data, error } = await client.POST('/v1/quotes', {
        body: { courtId, startsAt, endsAt },
      })
      if (error || !data) {
        quoteError.value = 'Could not get a quote for this slot. Please try again.'
        return
      }
      quote.value = mapToQuote(data)
      step.value = 'review'
    } catch {
      quoteError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      quoteLoading.value = false
    }
  }

  async function confirmBooking(): Promise<void> {
    // Confirm-step gate: never call CreateBooking without a reviewed quote.
    if (step.value !== 'review' || !quote.value) return

    if (Date.now() - quote.value.fetchedAt > QUOTE_STALE_MS) {
      // Never submit against a price the server hasn't *just* confirmed —
      // refresh it and make the Player re-confirm against the fresh number
      // instead of booking silently at a possibly-changed price.
      await getQuote(lastCourtId, lastStartsAt, lastEndsAt)
      if (!quoteError.value) {
        quoteRefreshedNotice.value =
          'Your quote may have changed, so we refreshed it — please review and confirm again.'
      }
      return
    }

    bookingLoading.value = true
    bookingError.value = null
    conflict.value = null
    quoteRefreshedNotice.value = null
    try {
      const { data, error, response } = await client.POST('/v1/bookings', {
        body: {
          courtId: lastCourtId,
          source: 'SOURCE_INDIVIDUAL',
          startsAt: lastStartsAt,
          endsAt: lastEndsAt,
        },
      })
      if (response.status === 409) {
        await loadConflictSuggestions()
        return
      }
      if (error || !data?.booking) {
        bookingError.value = 'Could not complete this booking. Please try again.'
        return
      }
      confirmedBooking.value = mapToBooking(data.booking)
      step.value = 'success'
    } catch {
      bookingError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      bookingLoading.value = false
    }
  }

  async function loadConflictSuggestions(): Promise<void> {
    // Specific, actionable message per T7.6 (WCAG 3.3.3 Error Suggestion) —
    // set immediately so the Player sees it even while suggestions load.
    conflict.value = {
      message: 'This slot was just booked — pick another time',
      suggestions: [],
      suggestionsLoading: true,
    }
    try {
      const searchFrom = new Date(lastStartsAt)
      const durationMs = new Date(lastEndsAt).getTime() - searchFrom.getTime()
      const searchUntil = new Date(searchFrom.getTime() + 24 * 60 * 60 * 1000)
      const { data, error } = await client.GET('/v1/courts/{courtId}/bookings', {
        params: {
          path: { courtId: lastCourtId },
          query: { from: searchFrom.toISOString(), to: searchUntil.toISOString() },
        },
      })
      const bookings = !error && data ? (data.bookings ?? []).map(mapToBooking) : []
      const suggestions = computeNextAvailableSlots({
        bookings,
        durationMs,
        searchFrom,
        searchUntil,
        limit: 3,
      })
      conflict.value = {
        message: 'This slot was just booked — pick another time',
        suggestions,
        suggestionsLoading: false,
      }
    } catch {
      conflict.value = {
        message: 'This slot was just booked — pick another time',
        suggestions: [],
        suggestionsLoading: false,
      }
    }
  }

  function backToForm(): void {
    step.value = 'form'
    conflict.value = null
    bookingError.value = null
    quoteRefreshedNotice.value = null
  }

  function reset(): void {
    step.value = 'form'
    quote.value = null
    quoteError.value = null
    bookingError.value = null
    conflict.value = null
    confirmedBooking.value = null
    quoteRefreshedNotice.value = null
  }

  return {
    step,
    quote,
    quoteLoading,
    quoteError,
    bookingLoading,
    bookingError,
    conflict,
    confirmedBooking,
    quoteRefreshedNotice,
    getQuote,
    confirmBooking,
    backToForm,
    reset,
  }
}
