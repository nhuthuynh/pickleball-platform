// View models + mapping functions for the quote/book flow (T7.6,
// docs/process/t7-sprint-plan.md), built from the raw Booking API response
// (web/src/api/generated/booking.d.ts's `components['schemas']`, produced
// from proto/pickleball/booking/v1/booking.proto).
//
// IMPORTANT — `price_cents` is an `int64` in the proto, which protojson (and
// therefore the generated OpenAPI/TS types) represents as a *string*, not a
// `number` (see `v1GetQuoteResponse.priceCents?: string` in
// `api/generated/booking.d.ts`). `mapToQuote` is the one place that parses
// it to a number; nothing else in this flow re-derives or recomputes a
// price — every displayed price is exactly what `GetQuote` last returned
// (T7.6 non-functional requirement: "no client-side price computation").
import type { components } from '../api/generated/booking'
import { mapToDiscountRule, type DiscountRule } from './discount'

export type RawQuote = components['schemas']['v1GetQuoteResponse']
export type RawBooking = components['schemas']['v1Booking']

/** A `GetQuote` result, plus when it was fetched — `fetchedAt` drives the
 * ~60s staleness check in `useCourtBooking.ts` (never submit a price the
 * server hasn't just confirmed).
 *
 * `bandPriceCents` and `discount` are T11.2 additions: the pre-discount band
 * price, and the rule that was applied. Both exist precisely so a client can
 * show what was taken off instead of swapping one number for another with no
 * indication a discount was applied (booking.proto's own words) — T11.3's
 * honest-UI requirement. */
export interface Quote {
  /** What the Player actually pays — band price with any discount applied. */
  priceCents: number
  /** The pre-discount band price. Equal to `priceCents` when no discount
   * applied (booking.proto), which is also what this maps an absent field to
   * rather than a fabricated 0. */
  bandPriceCents: number
  band: string
  /** The ISO 4217 code `priceCents` and `bandPriceCents` are both in
   * (T68.2, ADR-0005, #357). One field for both amounts because the server
   * sends one: they are the same resolved band's price.
   *
   * Always non-empty on a real `GetQuote` response — the server refuses to
   * quote a court whose pricing rules name no usable code or disagree about
   * one (FAILED_PRECONDITION, `domain.ErrInvalidPricingCurrency` /
   * `ErrMixedPricingCurrency`). `''` here therefore means a response that
   * predates the field or a hand-built fixture, not a real currency.
   *
   * NOT YET USED FOR DISPLAY, deliberately: see `formatPriceCents` below for
   * what is still owed and why it is a separate decision. */
  currency: string
  /** `null` — not a zero-valued rule — when no discount applied. */
  discount: DiscountRule | null
  fetchedAt: number
}

export function mapToQuote(raw: RawQuote, fetchedAt: number = Date.now()): Quote {
  const priceCents = Number(raw.priceCents ?? 0)
  return {
    priceCents,
    // A response carrying no bandPriceCents at all is not a $0.00 band price
    // — booking.proto states the two are equal when no discount applied, so
    // fall back to that rather than letting the UI strike through a zero.
    bandPriceCents: raw.bandPriceCents === undefined ? priceCents : Number(raw.bandPriceCents),
    band: raw.band ?? '',
    currency: raw.currency ?? '',
    discount: raw.discount ? mapToDiscountRule(raw.discount) : null,
    fetchedAt,
  }
}

/**
 * How much a discount actually took off this quote, in cents — 0 when no
 * discount applied, and 0 (never negative) for a discount that reduced
 * nothing. The UI uses this to decide whether there is a real saving to
 * show, so it can never render a phantom or negative one. This is not a
 * price computation: both numbers come straight from `GetQuote`, and the
 * price shown to the Player is always `priceCents` as the server sent it.
 */
export function quoteSavingCents(quote: Quote): number {
  return Math.max(0, quote.bandPriceCents - quote.priceCents)
}

/** A confirmed (or cancelled) `Booking`, as returned by `CreateBooking` /
 * `ListCourtBookings`. */
export interface ConfirmedBooking {
  id: string
  courtId: string
  startsAt: string
  endsAt: string
  status: string
  referenceId: string
}

export function mapToBooking(raw: RawBooking): ConfirmedBooking {
  return {
    id: raw.id ?? '',
    courtId: raw.courtId ?? '',
    startsAt: raw.startsAt ?? '',
    endsAt: raw.endsAt ?? '',
    status: raw.status ?? 'STATUS_UNSPECIFIED',
    referenceId: raw.referenceId ?? '',
  }
}

/** Formats a price the server returned in minor units into a display string
 * (e.g. `1800` -> `"$18.00"`). Display formatting only, not a price
 * computation — see this file's header comment.
 *
 * KNOWN GAP (T68.2, ADR-0005, #357): this hardcodes a dollar sign and a
 * two-decimal divisor, and `Quote.currency` is now on the response and is
 * **not** consulted here. #357 was a complaint about the API contract and
 * that is the half T68.2 closed; making this function currency-aware is a
 * separate decision nobody has made, and it is not a one-liner:
 *   * the symbol is per-currency (`Intl.NumberFormat` already knows one, so
 *     the fix is to use it rather than to grow a symbol table), and
 *   * the DIVISOR is per-currency too. ISO 4217 exponents are not all 2 —
 *     VND, the currency `0035_pricing_rules_currency.sql` seeds the demo
 *     facility in, has an exponent of 0, so `/ 100` renders 2000 dong as
 *     "20.00". ADR-0005 assumes integer minor units throughout and says
 *     nothing about zero-decimal currencies, so settling that is an
 *     ADR-level change touching every context holding a `*_cents` column,
 *     not a tweak to this function.
 * Until then this renders correctly for two-decimal currencies only, which
 * is what it has always done — what changed is that it is written down
 * rather than asserted to be true by construction. */
export function formatPriceCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

export interface BookingSlot {
  startsAt: string
  endsAt: string
}

/**
 * Client-side "next available slot" search (T7.6's double-booking conflict
 * handling, WCAG 3.3.3 Error Suggestion) — `ListCourtBookings` (live since
 * T2) doesn't itself suggest a slot, so this scans candidate start times at
 * `stepMs` increments across `[searchFrom, searchUntil)`, of the same
 * duration as the slot that just conflicted, and returns the first `limit`
 * that don't overlap any of `bookings`.
 *
 * Cancelled bookings are excluded defensively even though
 * `ListCourtBookings` (per its own T2 tests) already excludes them
 * server-side — this function shouldn't silently treat a future backend
 * change as "this slot is booked" if that guarantee ever weakens.
 */
export function computeNextAvailableSlots(params: {
  bookings: ConfirmedBooking[]
  durationMs: number
  searchFrom: Date
  searchUntil: Date
  limit?: number
  stepMs?: number
}): BookingSlot[] {
  const { bookings, durationMs, searchFrom, searchUntil, limit = 3, stepMs = 30 * 60 * 1000 } = params
  const active = bookings.filter((b) => b.status !== 'STATUS_CANCELLED')

  const slots: BookingSlot[] = []
  let candidateStart = searchFrom.getTime()

  while (slots.length < limit && candidateStart + durationMs <= searchUntil.getTime()) {
    const candidateEnd = candidateStart + durationMs
    const overlaps = active.some((b) => {
      const bStart = new Date(b.startsAt).getTime()
      const bEnd = new Date(b.endsAt).getTime()
      return bStart < candidateEnd && bEnd > candidateStart
    })
    if (!overlaps) {
      slots.push({
        startsAt: new Date(candidateStart).toISOString(),
        endsAt: new Date(candidateEnd).toISOString(),
      })
    }
    candidateStart += stepMs
  }

  return slots
}
