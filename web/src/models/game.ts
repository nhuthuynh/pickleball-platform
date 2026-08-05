// Player-facing view models for the Discover & Join Games screen (T8.9,
// docs/process/t8-sprint-plan.md), plus the mapping functions that build
// them from the raw Social Play API response
// (web/src/api/generated/socialplay.d.ts's `components['schemas']`,
// produced from proto/pickleball/socialplay/v1/socialplay.proto).
//
// Mirrors models/facility.ts/models/booking.ts's shape: a `RawX` type
// alias per wire message, plus one pure `mapToX` function per view model
// that only ever copies the specific fields it declares (no `...raw`
// spreads), so a future field added to the wire message never leaks into a
// view unless a mapping function is deliberately updated to carry it.
import type { components } from '../api/generated/socialplay'

export type RawGameListing = components['schemas']['v1GameListing']
export type RawGame = components['schemas']['v1Game']
export type RawRegistration = components['schemas']['v1Registration']
export type RawWaitlistEntry = components['schemas']['v1WaitlistEntry']

/** One row in the games list/search view (`ListGames`, T8.9). */
export interface GameSummary {
  id: string
  hostId: string
  venueFacilityId: string
  courtIds: string[]
  startsAt: string
  endsAt: string
  capacity: number
  status: string
  paymentMethod: string
  guestAllowance: number
  /** The Host's real entry fee for this Game (T9.2), in integer minor
   * units. 0 means the Game is FREE — a real, Host-chosen value, never
   * "unset" (see `entryFeeLabel`). Replaces T8.10's retired
   * `PLACEHOLDER_REGISTRATION_FEE_CENTS`. */
  entryFeeCents: number
  entryFeeCurrency: string
  /** Server-computed (capacity minus weighted active registrations) — see
   * `v1GameListing`'s doc comment. Never negative. */
  spotsLeft: number
}

export function mapToGameSummary(raw: RawGameListing): GameSummary {
  const game = raw.game ?? {}
  return {
    id: game.id ?? '',
    hostId: game.hostId ?? '',
    venueFacilityId: game.venueFacilityId ?? '',
    courtIds: game.courtIds ?? [],
    startsAt: game.startsAt ?? '',
    endsAt: game.endsAt ?? '',
    capacity: game.capacity ?? 0,
    status: game.status ?? 'GAME_STATUS_UNSPECIFIED',
    paymentMethod: game.paymentMethod ?? 'PAYMENT_METHOD_UNSPECIFIED',
    guestAllowance: game.guestAllowance ?? 0,
    // `amountCents` is an int64 in the proto, which protojson (and so the
    // generated TS types) represents as a *string* — same convention
    // models/payment.ts's `toMoneyRequest` documents. An absent entry_fee
    // (an old server, or a Game created before T9.2) reads as 0 = free,
    // matching both the wire default and the migration's backfill.
    entryFeeCents: Number(game.entryFee?.amountCents ?? 0),
    entryFeeCurrency: game.entryFee?.currencyCode ?? '',
    spotsLeft: raw.spotsLeft ?? 0,
  }
}

/**
 * Payment-method label (T8.9 kickoff note decision #3): T8's scope doesn't
 * wire real Payments UI yet (that's T8.10), so the detail view shows
 * `Game.PaymentMethod` as a plain-text label rather than a price — WCAG
 * 1.4.1 (Use of Color): this label is text, never color-only, and every
 * branch (including the unrecognized-value fallback) returns text.
 */
export function paymentMethodLabel(method: string): string {
  switch (method) {
    case 'PAYMENT_METHOD_ONLINE':
      return 'Online'
    case 'PAYMENT_METHOD_CASH':
      return 'Cash at facility'
    case 'PAYMENT_METHOD_EITHER':
      return 'Online or cash'
    default:
      return 'Payment method not set'
  }
}

/** A "spots left" label that always carries text (WCAG 1.4.1) — a bare
 * number is deliberately not used on its own anywhere in the UI so the
 * urgency signal never depends on styling (e.g. a red number) alone. */
export function spotsLeftLabel(spotsLeft: number): string {
  if (spotsLeft <= 0) return 'Full'
  if (spotsLeft === 1) return '1 spot left'
  return `${spotsLeft} spots left`
}

/** A confirmed (or rejected) `Registration`, as returned by
 * `RegisterForGame`. */
export interface ConfirmedRegistration {
  id: string
  gameId: string
  playerId: string
  status: string
  paymentStatus: string
  guestCount: number
}

export function mapToRegistration(raw: RawRegistration): ConfirmedRegistration {
  return {
    id: raw.id ?? '',
    gameId: raw.gameId ?? '',
    playerId: raw.playerId ?? '',
    status: raw.status ?? 'REGISTRATION_STATUS_UNSPECIFIED',
    paymentStatus: raw.paymentStatus ?? 'PAYMENT_STATUS_UNSPECIFIED',
    guestCount: raw.guestCount ?? 0,
  }
}

/** A confirmed `WaitlistEntry`, as returned by `JoinWaitlist`. */
export interface ConfirmedWaitlistEntry {
  id: string
  gameId: string
  playerId: string
  position: number
  status: string
}

export function mapToWaitlistEntry(raw: RawWaitlistEntry): ConfirmedWaitlistEntry {
  return {
    id: raw.id ?? '',
    gameId: raw.gameId ?? '',
    playerId: raw.playerId ?? '',
    position: raw.position ?? 0,
    status: raw.status ?? 'WAITLIST_STATUS_UNSPECIFIED',
  }
}

/** Formats an ISO start/end range for display, mirroring
 * `CourtBookingFlow.vue`'s local `formatRange` helper (booking.ts has no
 * shared formatter to reuse — this is Games' own copy of the same
 * `Intl.DateTimeFormat` approach). */
export function formatGameRange(startsAt: string, endsAt: string): string {
  const start = new Date(startsAt)
  const end = new Date(endsAt)
  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
  return `${dateFmt.format(start)}, ${timeFmt.format(start)}–${timeFmt.format(end)}`
}

/**
 * The player-facing entry-fee label (T9.2).
 *
 * A zero fee renders as the word **"Free"**, never as `"$0.00"` and never
 * as an empty space: a free Game is a real product state a Host
 * deliberately chose, not a missing or pending value (NN/g heuristic #2,
 * match between the system and the real world). This is also the whole
 * point of the field that replaced `PLACEHOLDER_REGISTRATION_FEE_CENTS` —
 * the placeholder could not express "free" at all.
 *
 * WCAG 1.4.1 (Use of Color): every branch returns text, so the price — and
 * the fact that a Game is free — is never conveyed by styling alone.
 */
export function entryFeeLabel(cents: number): string {
  if (cents <= 0) return 'Free'
  return `$${(cents / 100).toFixed(2)}`
}
