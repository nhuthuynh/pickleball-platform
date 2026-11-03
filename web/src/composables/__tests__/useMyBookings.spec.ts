// T68.3 (part of #353): the two actor-scoped reads behind the Bookings tab.
//
// The property this file exists to pin is the ticket's requirement 5:
// **the two reads are independent**. They are two different bounded contexts
// reached over two different typed clients, and either can fail on its own.
// A `Promise.all` over them — the obvious implementation — rejects the whole
// load on the first failure and would blank BOTH sections, which is exactly
// the "one failing read must not blank the screen" case. So there are tests
// for each read failing while the other still populates, in both directions,
// rather than one test for "an error is surfaced".
//
// Fake-client shape mirrors useCourtNames.spec.ts/useClubRentals.spec.ts:
// a `vi.fn()` GET per context, injected, so nothing stubs ambient `fetch`.
import { describe, it, expect, vi } from 'vitest'

import { useMyBookings } from '../useMyBookings'
import type { BookingClient } from '../../api/bookingClient'
import type { SocialPlayClient } from '../../api/socialplayClient'

const BOOKING_ROW = {
  booking: {
    id: 'bk-1',
    courtId: 'court-1',
    source: 'SOURCE_INDIVIDUAL',
    status: 'STATUS_CONFIRMED',
    startsAt: '2026-11-10T09:00:00Z',
    endsAt: '2026-11-10T10:00:00Z',
    referenceId: '',
  },
  facilityId: 'fac-1',
}

const REGISTRATION_ROW = {
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
  gameVenueFacilityId: 'fac-2',
  gameCourtIds: ['court-3', 'court-4'],
  gameStatus: 'GAME_STATUS_SCHEDULED',
}

function ok(data: unknown) {
  return { data, error: undefined, response: { status: 200 } }
}

function apiError(status = 500) {
  return { data: undefined, error: { message: 'boom' }, response: { status } }
}

function bookingClient(result: unknown | (() => unknown)): BookingClient {
  const GET = vi.fn(async (path: string) => {
    if (path !== '/v1/bookings') throw new Error(`unexpected GET ${path}`)
    return typeof result === 'function' ? (result as () => unknown)() : result
  })
  return { GET, POST: vi.fn() } as unknown as BookingClient
}

function socialplayClient(result: unknown | (() => unknown)): SocialPlayClient {
  const GET = vi.fn(async (path: string) => {
    if (path !== '/v1/registrations') throw new Error(`unexpected GET ${path}`)
    return typeof result === 'function' ? (result as () => unknown)() : result
  })
  return { GET, POST: vi.fn() } as unknown as SocialPlayClient
}

describe('useMyBookings', () => {
  it('loads both sections from the two actor-scoped reads', async () => {
    const result = useMyBookings(
      bookingClient(ok({ bookings: [BOOKING_ROW] })),
      socialplayClient(ok({ registrations: [REGISTRATION_ROW] })),
    )

    await result.load()

    expect(result.bookings.value).toHaveLength(1)
    expect(result.bookings.value[0]!.id).toBe('bk-1')
    expect(result.registrations.value).toHaveLength(1)
    expect(result.registrations.value[0]!.gameCourtIds).toEqual(['court-3', 'court-4'])
    expect(result.bookingsError.value).toBeNull()
    expect(result.registrationsError.value).toBeNull()
  })

  // Both messages are EMPTY on the wire — the actor is the token. A request
  // carrying any body or query would mean this client had invented an actor
  // field the protos deliberately do not have.
  it('sends no actor of any kind on either request', async () => {
    const booking = bookingClient(ok({ bookings: [] }))
    const socialplay = socialplayClient(ok({ registrations: [] }))

    await useMyBookings(booking, socialplay).load()

    expect(booking.GET).toHaveBeenCalledTimes(1)
    expect(booking.GET).toHaveBeenCalledWith('/v1/bookings', {})
    expect(socialplay.GET).toHaveBeenCalledTimes(1)
    expect(socialplay.GET).toHaveBeenCalledWith('/v1/registrations', {})
  })

  // ── Requirement 5: independence, asserted in BOTH directions ────────────

  it('keeps the games section when the bookings read fails', async () => {
    const result = useMyBookings(
      bookingClient(apiError()),
      socialplayClient(ok({ registrations: [REGISTRATION_ROW] })),
    )

    await result.load()

    expect(result.bookingsError.value).not.toBeNull()
    expect(result.bookings.value).toEqual([])
    // The other half is untouched.
    expect(result.registrations.value).toHaveLength(1)
    expect(result.registrationsError.value).toBeNull()
  })

  it('keeps the bookings section when the games read fails', async () => {
    const result = useMyBookings(
      bookingClient(ok({ bookings: [BOOKING_ROW] })),
      socialplayClient(apiError()),
    )

    await result.load()

    expect(result.registrationsError.value).not.toBeNull()
    expect(result.registrations.value).toEqual([])
    expect(result.bookings.value).toHaveLength(1)
    expect(result.bookingsError.value).toBeNull()
  })

  // A THROWN rejection, not a populated `error` — openapi-fetch propagates a
  // rejected `fetch()` rather than resolving (see api/__tests__/client.spec.ts).
  // A `Promise.all` implementation would reject the whole `load()` here and
  // leave the other section's `loading` stuck true forever.
  it('keeps the games section when the bookings read throws', async () => {
    const result = useMyBookings(
      bookingClient(() => {
        throw new Error('network down')
      }),
      socialplayClient(ok({ registrations: [REGISTRATION_ROW] })),
    )

    await expect(result.load()).resolves.toBeUndefined()

    expect(result.bookingsError.value).not.toBeNull()
    expect(result.registrations.value).toHaveLength(1)
    expect(result.bookingsLoading.value).toBe(false)
    expect(result.registrationsLoading.value).toBe(false)
  })

  it('surfaces a failure in both sections when both reads fail', async () => {
    const result = useMyBookings(bookingClient(apiError()), socialplayClient(apiError()))

    await result.load()

    expect(result.bookingsError.value).not.toBeNull()
    expect(result.registrationsError.value).not.toBeNull()
  })

  // ── Nothing is filtered out ─────────────────────────────────────────────

  // Both protos say so in as many words. A cancelled booking and a
  // host-cancelled game are the rows the user most needs; dropping either
  // reinstates the blind spot the reads exist to remove.
  it('keeps cancelled rows on both sides', async () => {
    const cancelledBooking = {
      ...BOOKING_ROW,
      booking: { ...BOOKING_ROW.booking, id: 'bk-cancelled', status: 'STATUS_CANCELLED' },
    }
    const cancelledGame = { ...REGISTRATION_ROW, gameStatus: 'GAME_STATUS_CANCELLED' }

    const result = useMyBookings(
      bookingClient(ok({ bookings: [cancelledBooking] })),
      socialplayClient(ok({ registrations: [cancelledGame] })),
    )

    await result.load()

    expect(result.bookings.value.map((b) => b.status)).toEqual(['STATUS_CANCELLED'])
    expect(result.registrations.value.map((r) => r.gameStatus)).toEqual(['GAME_STATUS_CANCELLED'])
  })

  it('keeps a booking whose facility is empty', async () => {
    const result = useMyBookings(
      bookingClient(ok({ bookings: [{ booking: { id: 'bk-x', courtId: 'c-x' } }] })),
      socialplayClient(ok({ registrations: [] })),
    )

    await result.load()

    expect(result.bookings.value).toHaveLength(1)
    expect(result.bookings.value[0]!.facilityId).toBe('')
  })

  // An absent repeated field is an empty list, not a failure: a player who
  // has booked nothing is a normal state and must not read as an error.
  it('treats an absent list as empty rather than as an error', async () => {
    const result = useMyBookings(bookingClient(ok({})), socialplayClient(ok({})))

    await result.load()

    expect(result.bookings.value).toEqual([])
    expect(result.registrations.value).toEqual([])
    expect(result.bookingsError.value).toBeNull()
    expect(result.registrationsError.value).toBeNull()
  })

  it('clears loading on both sections once settled', async () => {
    const result = useMyBookings(
      bookingClient(ok({ bookings: [] })),
      socialplayClient(ok({ registrations: [] })),
    )

    const pending = result.load()
    expect(result.bookingsLoading.value).toBe(true)
    expect(result.registrationsLoading.value).toBe(true)
    await pending
    expect(result.bookingsLoading.value).toBe(false)
    expect(result.registrationsLoading.value).toBe(false)
  })

  // A 401 is a different thing to tell someone than "the server broke": the
  // remedy is signing in, not retrying. The middleware already clears the
  // token; this just has to not call it a server error.
  it('says the session expired on a 401 rather than reporting a server fault', async () => {
    const result = useMyBookings(bookingClient(apiError(401)), socialplayClient(apiError(401)))

    await result.load()

    expect(result.bookingsError.value).toMatch(/sign in/i)
    expect(result.registrationsError.value).toMatch(/sign in/i)
  })

  // Per-section errors imply a per-section retry: a "Try again" beside the
  // failed half must not re-fetch the half that already succeeded, or a
  // retry would throw away good data to re-request it.
  it('retries one section without touching the other', async () => {
    const booking = bookingClient(ok({ bookings: [BOOKING_ROW] }))
    const socialplay = socialplayClient(ok({ registrations: [REGISTRATION_ROW] }))
    const result = useMyBookings(booking, socialplay)

    await result.load()
    expect(booking.GET).toHaveBeenCalledTimes(1)
    expect(socialplay.GET).toHaveBeenCalledTimes(1)

    await result.loadRegistrations()

    expect(socialplay.GET).toHaveBeenCalledTimes(2)
    expect(booking.GET).toHaveBeenCalledTimes(1)
    expect(result.bookings.value).toHaveLength(1)
  })

  // A reload after a failure must not leave the old message on screen next
  // to fresh data.
  it('clears a previous error on a successful reload', async () => {
    let fail = true
    const result = useMyBookings(
      bookingClient(() => (fail ? apiError() : ok({ bookings: [BOOKING_ROW] }))),
      socialplayClient(ok({ registrations: [] })),
    )

    await result.load()
    expect(result.bookingsError.value).not.toBeNull()

    fail = false
    await result.load()

    expect(result.bookingsError.value).toBeNull()
    expect(result.bookings.value).toHaveLength(1)
  })
})
