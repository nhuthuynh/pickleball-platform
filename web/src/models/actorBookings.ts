// View models + labels for the player's own Bookings tab — T68.3, the read
// half of issue #353.
//
// This is the first view model in `web/src/models` that spans TWO bounded
// contexts, because the screen does: `GET /v1/bookings`
// (booking.ListBookingsForActor, T67.1) and `GET /v1/registrations`
// (socialplay.ListRegistrationsForActor, T67.2). They stay two separate
// types with two separate mapping functions rather than one merged "thing I
// am committed to" — a Booking and a Registration are different aggregates
// with different statuses, different cancellation semantics and different
// owners, and CLAUDE.md rule 7's one-ubiquitous-language point cuts against
// inventing a client-only union noun that exists nowhere in the protos, the
// DB, or the other two clients.
//
// Mirrors models/booking.ts/models/game.ts's established shape: a `RawX`
// alias per wire message, plus one pure `mapToX` per view model that copies
// only the fields it declares (no `...raw` spreads), so a field added to the
// wire later cannot leak into a view unless a mapping function is
// deliberately updated to carry it.
//
// ── THE RULE THIS FILE EXISTS TO ENFORCE ────────────────────────────────────
//
// **Nothing here fabricates a state the read did not supply, and nothing
// here drops a row.** Both protos are unusually explicit that filtering is
// the failure mode:
//
//   - `ListBookingsForActorRequest` has no status filter, deliberately: "a
//     status filter would invite a client to show only the live bookings,
//     which is the exact shape that hides a cancellation instead of
//     reporting it."
//   - `RegistrationListing.game_status` is "the most load-bearing field on
//     this message": `ListGames` filters `WHERE g.status = 'scheduled'` and
//     socialplay has NO `GetGame` RPC, so this read is the only way a player
//     can ever be told their Game was cancelled. "A client MUST render a
//     cancelled Game as cancelled rather than filtering it out."
//
// So every `*Label` function below is total over its input: an enum value
// this client has never heard of, and the UNSPECIFIED zero value, both
// resolve to an explicit "not recorded" rather than to the healthy branch.
// Every branch returns TEXT, never a colour or an icon alone (WCAG 1.4.1).
import type { components as BookingComponents } from '../api/generated/booking'
import type { components as SocialPlayComponents } from '../api/generated/socialplay'

export type RawActorBooking = BookingComponents['schemas']['v1ActorBooking']
export type RawRegistrationListing = SocialPlayComponents['schemas']['v1RegistrationListing']

/** One row of the court-bookings section: a Booking plus the Facility that
 *  owns its Court, which is what lets the screen show a court NAME instead
 *  of a uuid (#342, closed — `ActorBooking.facility_id` is what keeps it
 *  closed here). */
export interface ActorBookingView {
  id: string
  courtId: string
  /** `''` is a REAL, expected answer, not a failure — a Court's facility is
   *  nullable in this schema and both seeded Courts have none. See
   *  `ActorBooking.facility_id`'s doc comment. The screen must still render
   *  the row. */
  facilityId: string
  source: string
  status: string
  startsAt: string
  endsAt: string
  referenceId: string
}

export function mapToActorBooking(raw: RawActorBooking): ActorBookingView {
  // protojson omits an absent message field entirely rather than sending
  // `{}`, so `raw.booking` really can be undefined on the wire.
  const booking = raw.booking ?? {}
  return {
    id: booking.id ?? '',
    courtId: booking.courtId ?? '',
    facilityId: raw.facilityId ?? '',
    source: booking.source ?? 'SOURCE_UNSPECIFIED',
    status: booking.status ?? 'STATUS_UNSPECIFIED',
    startsAt: booking.startsAt ?? '',
    endsAt: booking.endsAt ?? '',
    referenceId: booking.referenceId ?? '',
  }
}

/** One row of the games section: the player's Registration plus the five
 *  `game_*` facts `RegistrationListing` projects for exactly this screen. */
export interface ActorRegistrationView {
  id: string
  gameId: string
  status: string
  paymentStatus: string
  guestCount: number
  /** Minor units, parsed from the int64-as-string wire form. FROZEN at
   *  registration time by the server — never recomputed here from a current
   *  entry fee, which would show a price this player never agreed to. */
  amountOwedCents: number
  gameStartsAt: string
  gameEndsAt: string
  gameVenueFacilityId: string
  /** REPEATED on the wire: "a Game spanning more than one court is normal
   *  here (unlike a Booking)". Carried as a list and passed to
   *  `CourtNames.vue` as a list — see that component's `courtIds` prop,
   *  which has always been plural. */
  gameCourtIds: string[]
  /** The GAME's status, not the Registration's. See this file's header. */
  gameStatus: string
}

export function mapToActorRegistration(raw: RawRegistrationListing): ActorRegistrationView {
  const registration = raw.registration ?? {}
  return {
    id: registration.id ?? '',
    gameId: registration.gameId ?? '',
    status: registration.status ?? 'REGISTRATION_STATUS_UNSPECIFIED',
    paymentStatus: registration.paymentStatus ?? 'PAYMENT_STATUS_UNSPECIFIED',
    guestCount: registration.guestCount ?? 0,
    // `amount_cents` is an int64, which protojson (and so the generated TS
    // types) represents as a *string* — the same convention
    // models/booking.ts's `priceCents` note and models/payment.ts's
    // `toMoneyRequest` both document. `Number(undefined)` is NaN, so the
    // `?? 0` has to sit inside the call, not after it.
    amountOwedCents: Number(registration.amountOwed?.amountCents ?? 0),
    gameStartsAt: raw.gameStartsAt ?? '',
    gameEndsAt: raw.gameEndsAt ?? '',
    gameVenueFacilityId: raw.gameVenueFacilityId ?? '',
    gameCourtIds: raw.gameCourtIds ?? [],
    gameStatus: raw.gameStatus ?? 'GAME_STATUS_UNSPECIFIED',
  }
}

/** A Booking's own status. `STATUS_CANCELLED` rows are deliberately in this
 *  read (see header) so a player can tell "I cancelled it" from "it was
 *  lost" — this label is what makes that distinction visible. */
export function bookingStatusLabel(status: string): string {
  switch (status) {
    case 'STATUS_CONFIRMED':
      return 'Confirmed'
    case 'STATUS_CANCELLED':
      return 'Cancelled'
    default:
      return 'Status not recorded'
  }
}

/**
 * Which of the four polymorphic Booking sources this row is.
 *
 * Why the screen shows it at all: a court booking held FOR a Game is the
 * same Booking aggregate as an individual reservation (the locked
 * polymorphic-Booking decision), so a player registered for a Game can see
 * the slot in BOTH sections of this screen. Unlabelled, that reads as a
 * double booking they did not make.
 */
export function bookingSourceLabel(source: string): string {
  switch (source) {
    case 'SOURCE_INDIVIDUAL':
      return 'Court booking'
    case 'SOURCE_RECURRING_HIRE':
      return 'Recurring hire'
    case 'SOURCE_GAME':
      return 'Held for a game'
    case 'SOURCE_COMPETITION':
      return 'Held for a competition'
    default:
      // Not "Unknown booking": the row IS a booking, and that much is known.
      return 'Booking'
  }
}

/**
 * The HOST's cancellation of the Game, which is the one fact a player cannot
 * learn anywhere else in this product (socialplay exposes no `GetGame`, and
 * `ListGames` filters cancelled Games out entirely).
 *
 * "Cancelled by the host" rather than a bare "Cancelled" deliberately: the
 * row right beside it carries the player's OWN registration status, which
 * can also read cancelled, and the two mean opposite things about who acted.
 */
export function gameStatusLabel(status: string): string {
  switch (status) {
    case 'GAME_STATUS_SCHEDULED':
      return 'Scheduled'
    case 'GAME_STATUS_CANCELLED':
      return 'Cancelled by the host'
    default:
      return 'Status not recorded'
  }
}

export function registrationStatusLabel(status: string): string {
  switch (status) {
    case 'REGISTRATION_STATUS_REGISTERED':
      return 'You are registered'
    case 'REGISTRATION_STATUS_CANCELLED':
      return 'You cancelled your place'
    default:
      return 'Status not recorded'
  }
}

/**
 * The payment state as the read supplied it — and "not recorded" when it did
 * not supply one.
 *
 * `PAYMENT_STATUS_UNSPECIFIED` must NOT collapse into "Unpaid": that would
 * tell a player they owe money on evidence this read never gave, which is
 * the ticket's requirement 7 ("do not fabricate a paid/unpaid state the read
 * did not supply") stated as a default branch.
 */
export function paymentStatusLabel(paymentStatus: string): string {
  switch (paymentStatus) {
    case 'PAYMENT_STATUS_UNPAID':
      return 'Unpaid'
    case 'PAYMENT_STATUS_PAID':
      return 'Paid'
    case 'PAYMENT_STATUS_REFUNDED':
      return 'Refunded'
    default:
      return 'Payment status not recorded'
  }
}

/**
 * The amount still owed, or `null` when there is nothing honest to say.
 *
 * Requirement 7's two conditions, and no third: UNPAID **and** a non-zero
 * `amount_owed`. A zero amount on an unpaid registration is a FREE Game, not
 * a debt of $0.00 (see `entryFeeLabel`'s identical reasoning), and a paid,
 * refunded or unknown-status registration has no outstanding amount this
 * read can vouch for.
 *
 * Returns `null` rather than `''` so a template's `v-if` cannot accidentally
 * render an empty money row.
 */
export function amountOwedLabel(paymentStatus: string, amountOwedCents: number): string | null {
  if (paymentStatus !== 'PAYMENT_STATUS_UNPAID') return null
  if (!Number.isFinite(amountOwedCents) || amountOwedCents <= 0) return null
  return `$${(amountOwedCents / 100).toFixed(2)}`
}

/**
 * A booking/game window as one human string: the date, then the time range.
 *
 * Guarded, and that guard is the point. `Intl.DateTimeFormat.format` THROWS
 * `RangeError: Invalid time value` on `new Date(NaN)` — it does not return
 * "Invalid Date" — so an unguarded formatter here would not merely print
 * something ugly, it would take the whole section's render down on one
 * malformed timestamp. `models/recurringHire.ts`'s `formatDateTime` draws
 * the same line (`''` rather than today's date as a stand-in); this returns
 * a sentence instead of `''` because it is rendered as a row's primary
 * heading, where an empty string would look like a layout bug.
 */
export function formatBookingWindow(startsAt: string, endsAt: string): string {
  const start = new Date(startsAt)
  const end = new Date(endsAt)
  if (!startsAt || !endsAt || Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return 'Time not recorded'
  }
  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
  return `${dateFmt.format(start)}, ${timeFmt.format(start)}–${timeFmt.format(end)}`
}

/** How many guests this registration brought, or `null` for none — so the
 *  template renders no guest row at all rather than "0 guests". */
export function guestCountLabel(guestCount: number): string | null {
  if (!Number.isFinite(guestCount) || guestCount <= 0) return null
  return guestCount === 1 ? '1 guest' : `${guestCount} guests`
}
