// T68.3 (part of #353): the view models behind the Bookings tab — the read
// half of "what did I book and what did I join".
//
// Table-driven per CLAUDE.md rule 1. The properties under test are the ones
// the ticket's functional requirements name as load-bearing, and each table
// row is a wire shape the server can really send:
//
//   - a CANCELLED booking and a HOST-CANCELLED game both map to a rendered
//     row with an honest label, never to `null`/dropped (requirement 3).
//     Both protos say so in as many words; `ListBookingsForActor` has no
//     status filter precisely so the client cannot hide a cancellation, and
//     `RegistrationListing.game_status` is the ONLY way a player can learn
//     their Game was cancelled (there is no GetGame RPC).
//   - an UNSPECIFIED status maps to "not recorded", never to a fabricated
//     Confirmed/Scheduled/Unpaid (requirement 7's "do not fabricate a
//     paid/unpaid state the read did not supply", applied to every enum).
//   - `amount_owed` shows ONLY for unpaid + non-zero (requirement 7).
import { describe, it, expect } from 'vitest'

import {
  mapToActorBooking,
  mapToActorRegistration,
  bookingStatusLabel,
  bookingSourceLabel,
  gameStatusLabel,
  registrationStatusLabel,
  paymentStatusLabel,
  amountOwedLabel,
  formatBookingWindow,
  guestCountLabel,
  type RawActorBooking,
  type RawRegistrationListing,
} from '../actorBookings'

describe('mapToActorBooking', () => {
  it('copies every field the screen renders', () => {
    const raw: RawActorBooking = {
      booking: {
        id: 'bk-1',
        courtId: 'court-1',
        source: 'SOURCE_INDIVIDUAL',
        status: 'STATUS_CONFIRMED',
        startsAt: '2026-11-10T09:00:00Z',
        endsAt: '2026-11-10T10:00:00Z',
        referenceId: 'ref-1',
      },
      facilityId: 'fac-1',
    }

    expect(mapToActorBooking(raw)).toEqual({
      id: 'bk-1',
      courtId: 'court-1',
      facilityId: 'fac-1',
      source: 'SOURCE_INDIVIDUAL',
      status: 'STATUS_CONFIRMED',
      startsAt: '2026-11-10T09:00:00Z',
      endsAt: '2026-11-10T10:00:00Z',
      referenceId: 'ref-1',
    })
  })

  // ActorBooking.facility_id's own doc comment: "**Empty is a real, expected
  // answer**... A client must render such a booking, not hide it — the
  // booking is real." Both courts seeded by 0002_seed.sql have no Facility,
  // so on a freshly migrated database this is the COMMON case, not an edge.
  it('maps an absent facility to an empty string, not to a dropped row', () => {
    const mapped = mapToActorBooking({ booking: { id: 'bk-2', courtId: 'court-9' } })

    expect(mapped.id).toBe('bk-2')
    expect(mapped.facilityId).toBe('')
    // Never invented. An unknown status is unknown, not confirmed.
    expect(mapped.status).toBe('STATUS_UNSPECIFIED')
  })

  // protojson omits a message field entirely rather than sending `{}`, so a
  // row with no `booking` at all is a shape the wire can produce.
  it('survives a listing with no booking message', () => {
    expect(mapToActorBooking({ facilityId: 'fac-1' })).toEqual({
      id: '',
      courtId: '',
      facilityId: 'fac-1',
      source: 'SOURCE_UNSPECIFIED',
      status: 'STATUS_UNSPECIFIED',
      startsAt: '',
      endsAt: '',
      referenceId: '',
    })
  })
})

describe('mapToActorRegistration', () => {
  it('copies the registration and the projected game facts', () => {
    const raw: RawRegistrationListing = {
      registration: {
        id: 'reg-1',
        gameId: 'g-1',
        playerId: 'player-1',
        status: 'REGISTRATION_STATUS_REGISTERED',
        paymentStatus: 'PAYMENT_STATUS_UNPAID',
        guestCount: 2,
        amountOwed: { amountCents: '3000', currencyCode: 'USD' },
      },
      gameStartsAt: '2026-11-12T18:00:00Z',
      gameEndsAt: '2026-11-12T19:30:00Z',
      gameVenueFacilityId: 'fac-2',
      gameCourtIds: ['court-3', 'court-4'],
      gameStatus: 'GAME_STATUS_SCHEDULED',
    }

    expect(mapToActorRegistration(raw)).toEqual({
      id: 'reg-1',
      gameId: 'g-1',
      status: 'REGISTRATION_STATUS_REGISTERED',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      guestCount: 2,
      amountOwedCents: 3000,
      gameStartsAt: '2026-11-12T18:00:00Z',
      gameEndsAt: '2026-11-12T19:30:00Z',
      gameVenueFacilityId: 'fac-2',
      // REPEATED on the wire, and a multi-court Game is normal in socialplay
      // — carried as a list, not collapsed to the first id.
      gameCourtIds: ['court-3', 'court-4'],
      gameStatus: 'GAME_STATUS_SCHEDULED',
    })
  })

  // `amount_cents` is an int64, which protojson represents as a STRING —
  // same convention models/booking.ts's `priceCents` note documents. A
  // mapping that forgot this would hand the formatter "3000" and render
  // "$NaN".
  it('parses the int64-as-string amount into a number', () => {
    const mapped = mapToActorRegistration({
      registration: { id: 'r', amountOwed: { amountCents: '1250' } },
    })
    expect(mapped.amountOwedCents).toBe(1250)
  })

  it('maps an absent amount to 0 rather than NaN', () => {
    expect(mapToActorRegistration({ registration: { id: 'r' } }).amountOwedCents).toBe(0)
  })

  it('maps an absent court list to an empty list, not undefined', () => {
    expect(mapToActorRegistration({}).gameCourtIds).toEqual([])
  })
})

// ── Labels: every branch returns TEXT (WCAG 1.4.1), and no branch invents a
// state the read did not supply. ────────────────────────────────────────────

describe('bookingStatusLabel', () => {
  it.each([
    ['STATUS_CONFIRMED', 'Confirmed'],
    ['STATUS_CANCELLED', 'Cancelled'],
    ['STATUS_UNSPECIFIED', 'Status not recorded'],
    ['SOMETHING_NEWER_THAN_THIS_CLIENT', 'Status not recorded'],
  ])('%s -> %s', (status, expected) => {
    expect(bookingStatusLabel(status)).toBe(expected)
  })
})

describe('gameStatusLabel', () => {
  // The most load-bearing label on this screen. ListGames filters
  // `WHERE g.status = 'scheduled'` and socialplay has NO GetGame RPC, so
  // this read is the only place a player is ever told the host cancelled.
  it.each([
    ['GAME_STATUS_SCHEDULED', 'Scheduled'],
    ['GAME_STATUS_CANCELLED', 'Cancelled by the host'],
    ['GAME_STATUS_UNSPECIFIED', 'Status not recorded'],
    ['', 'Status not recorded'],
  ])('%s -> %s', (status, expected) => {
    expect(gameStatusLabel(status)).toBe(expected)
  })
})

describe('registrationStatusLabel', () => {
  it.each([
    ['REGISTRATION_STATUS_REGISTERED', 'You are registered'],
    ['REGISTRATION_STATUS_CANCELLED', 'You cancelled your place'],
    ['REGISTRATION_STATUS_UNSPECIFIED', 'Status not recorded'],
  ])('%s -> %s', (status, expected) => {
    expect(registrationStatusLabel(status)).toBe(expected)
  })
})

describe('paymentStatusLabel', () => {
  // Requirement 7, stated as a property: an UNSPECIFIED payment status is
  // NOT "Unpaid". Collapsing the two would tell a player they owe money on
  // evidence the read never supplied.
  it.each([
    ['PAYMENT_STATUS_UNPAID', 'Unpaid'],
    ['PAYMENT_STATUS_PAID', 'Paid'],
    ['PAYMENT_STATUS_REFUNDED', 'Refunded'],
    ['PAYMENT_STATUS_UNSPECIFIED', 'Payment status not recorded'],
    ['', 'Payment status not recorded'],
  ])('%s -> %s', (status, expected) => {
    expect(paymentStatusLabel(status)).toBe(expected)
  })
})

describe('amountOwedLabel', () => {
  // Requirement 7: unpaid AND non-zero. Every other combination returns null
  // — there is nothing honest to say about an amount owed on a paid, refunded
  // or unknown-status registration.
  it.each([
    ['PAYMENT_STATUS_UNPAID', 3000, '$30.00'],
    ['PAYMENT_STATUS_UNPAID', 1, '$0.01'],
    ['PAYMENT_STATUS_UNPAID', 0, null],
    ['PAYMENT_STATUS_PAID', 3000, null],
    ['PAYMENT_STATUS_REFUNDED', 3000, null],
    ['PAYMENT_STATUS_UNSPECIFIED', 3000, null],
  ])('%s with %i cents -> %s', (paymentStatus, cents, expected) => {
    expect(amountOwedLabel(paymentStatus as string, cents as number)).toBe(expected)
  })
})

describe('formatBookingWindow', () => {
  // The END DATE must survive (PR #369 review, promoted to required). The
  // first version formatted the start's date and then only the two CLOCK
  // times, so:
  //
  //   09:00 Nov 10 -> 10:00 Nov 12  rendered  "Nov 10, 2026, 9:00 AM–10:00 AM"
  //
  // A two-day hold displayed as a one-hour booking — byte-identical to the
  // genuine one-hour booking on the row above it. That is precisely the
  // "fabricating a state the read did not supply" this file's header forbids,
  // on the screen whose whole job is telling a person what they booked.
  //
  // Both halves are asserted against `Intl`-formatted expectations computed
  // the same way the function does, rather than against frozen strings:
  // pinning "Nov 12, 2026" would pin the runner's ICU data and locale, not
  // this function's behaviour.
  const mediumDate = (iso: string) =>
    new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(iso))

  it('carries the end date when the window spans more than one day', () => {
    const label = formatBookingWindow('2026-11-10T09:00:00Z', '2026-11-12T10:00:00Z')

    expect(label).toContain(mediumDate('2026-11-10T09:00:00Z'))
    expect(label).toContain(mediumDate('2026-11-12T10:00:00Z'))
  })

  // 23:00 -> 01:00 read as a backwards window: the end looked like it came
  // ten hours BEFORE the start, on the same stated day.
  it('carries the end date across midnight', () => {
    const label = formatBookingWindow('2026-11-10T23:00:00Z', '2026-11-11T01:00:00Z')

    expect(label).toContain(mediumDate('2026-11-10T23:00:00Z'))
    expect(label).toContain(mediumDate('2026-11-11T01:00:00Z'))
  })

  // The common case stays short — repeating the same date twice on every
  // ordinary one-hour booking would be noise, and this is what keeps the fix
  // from being "print both dates always".
  it('states the date once for a window inside a single day', () => {
    const date = mediumDate('2026-11-10T09:00:00Z')
    const label = formatBookingWindow('2026-11-10T09:00:00Z', '2026-11-10T10:00:00Z')

    expect(label.split(date)).toHaveLength(2) // i.e. the date occurs exactly once
  })

  it('renders a date and a time range', () => {
    // Asserted as a property rather than against a frozen locale string: the
    // formatter is Intl-backed, so pinning exact output would pin the test
    // runner's ICU data rather than this function's behaviour.
    const label = formatBookingWindow('2026-11-10T09:00:00Z', '2026-11-10T10:30:00Z')
    expect(label).toContain('2026')
    expect(label).toMatch(/\d/)
    expect(label).not.toMatch(/Invalid|NaN/)
  })

  // An honest label rather than a throw or "Invalid Date". Intl's
  // `format(new Date(NaN))` throws RangeError outright, so an unguarded
  // formatter would take the whole section down with it.
  it.each([
    ['', ''],
    ['not-a-date', '2026-11-10T10:00:00Z'],
    ['2026-11-10T09:00:00Z', ''],
  ])('says so rather than throwing for (%s, %s)', (startsAt, endsAt) => {
    expect(formatBookingWindow(startsAt, endsAt)).toBe('Time not recorded')
  })
})

describe('guestCountLabel', () => {
  // `null` for none, so the template renders no guest row at all rather than
  // the word "0 guests" — the same "a real state is not an empty one"
  // distinction `entryFeeLabel` draws between "Free" and a missing price.
  it.each([
    [0, null],
    [1, '1 guest'],
    [2, '2 guests'],
    [-1, null],
    [Number.NaN, null],
  ])('%s -> %s', (guestCount, expected) => {
    expect(guestCountLabel(guestCount as number)).toBe(expected)
  })
})

describe('bookingSourceLabel', () => {
  // Why this label exists at all: a court booking held FOR a Game appears in
  // the court-bookings section as well, and a player seeing the same slot
  // twice with no explanation would reasonably think they double-booked.
  it.each([
    ['SOURCE_INDIVIDUAL', 'Court booking'],
    ['SOURCE_RECURRING_HIRE', 'Recurring hire'],
    ['SOURCE_GAME', 'Held for a game'],
    ['SOURCE_COMPETITION', 'Held for a competition'],
    ['SOURCE_UNSPECIFIED', 'Booking'],
  ])('%s -> %s', (source, expected) => {
    expect(bookingSourceLabel(source)).toBe(expected)
  })
})
