import { describe, it, expect, vi } from 'vitest'
import { useJoinGame } from '../useJoinGame'
import type { SocialPlayClient } from '../../api/socialplayClient'

function fakeClient(handlers: {
  postRegistrations?: (body: unknown) => unknown
  postWaitlist?: (body: unknown) => unknown
}): SocialPlayClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/games/{gameId}/registrations') return handlers.postRegistrations?.(options.body)
    if (path === '/v1/games/{gameId}/waitlist') return handlers.postWaitlist?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as SocialPlayClient
}

function registrationOk(guestCount = 1) {
  return {
    data: {
      registration: {
        id: 'r1',
        gameId: 'g1',
        playerId: 'player-mock-1',
        status: 'REGISTRATION_STATUS_REGISTERED',
        paymentStatus: 'PAYMENT_STATUS_UNPAID',
        guestCount,
      },
    },
    error: undefined,
    response: { status: 200 },
  }
}

function gameFullConflict() {
  return { data: undefined, error: { message: 'game is full' }, response: { status: 409 } }
}

function waitlistOk(position = 2) {
  return {
    data: { entry: { id: 'w1', gameId: 'g1', playerId: 'player-mock-1', position, status: 'WAITLIST_STATUS_WAITING' } },
    error: undefined,
    response: { status: 200 },
  }
}

describe('useJoinGame', () => {
  it('starts with 0 guests and no result/error state', () => {
    const { guestCount, registering, registerError, gameFull, confirmedRegistration } = useJoinGame(
      'g1',
      fakeClient({}),
    )
    expect(guestCount.value).toBe(0)
    expect(registering.value).toBe(false)
    expect(registerError.value).toBeNull()
    expect(gameFull.value).toBe(false)
    expect(confirmedRegistration.value).toBeNull()
  })

  // T8.9 requirement: "guest-count stepper respects GuestAllowance".
  describe('guest-count stepper bounds', () => {
    it('incrementGuests never goes past the given max (GuestAllowance)', () => {
      const { guestCount, incrementGuests } = useJoinGame('g1', fakeClient({}))
      incrementGuests(2)
      incrementGuests(2)
      incrementGuests(2) // third increment should be a no-op at max=2
      expect(guestCount.value).toBe(2)
    })

    it('incrementGuests is a no-op when max is 0 (no guests allowed)', () => {
      const { guestCount, incrementGuests } = useJoinGame('g1', fakeClient({}))
      incrementGuests(0)
      expect(guestCount.value).toBe(0)
    })

    it('decrementGuests never goes below 0', () => {
      const { guestCount, decrementGuests } = useJoinGame('g1', fakeClient({}))
      decrementGuests()
      expect(guestCount.value).toBe(0)
    })

    it('increments and decrements within bounds', () => {
      const { guestCount, incrementGuests, decrementGuests } = useJoinGame('g1', fakeClient({}))
      incrementGuests(3)
      incrementGuests(3)
      expect(guestCount.value).toBe(2)
      decrementGuests()
      expect(guestCount.value).toBe(1)
    })
  })

  it('register success stores the confirmed registration, including guestCount', async () => {
    const client = fakeClient({ postRegistrations: () => registrationOk(2) })
    const { guestCount, incrementGuests, register, confirmedRegistration } = useJoinGame('g1', client)
    incrementGuests(2)
    incrementGuests(2)

    await register('player-mock-1')

    expect(confirmedRegistration.value).toMatchObject({ id: 'r1', guestCount: 2 })
    expect(client.POST).toHaveBeenCalledWith('/v1/games/{gameId}/registrations', {
      params: { path: { gameId: 'g1' } },
      body: { playerId: 'player-mock-1', guestCount: 2 },
    })
  })

  // T8.9 requirement: the game-full/waitlist-offer path.
  describe('capacity-exceeded (game full) path', () => {
    it('a 409 response sets gameFull with a specific, actionable message', async () => {
      const client = fakeClient({ postRegistrations: () => gameFullConflict() })
      const { register, gameFull, registerError, confirmedRegistration } = useJoinGame('g1', client)

      await register('player-mock-1')

      expect(gameFull.value).toBe(true)
      expect(registerError.value).toBe('This game just filled up.')
      expect(confirmedRegistration.value).toBeNull()
    })

    it('joinWaitlist after a game-full rejection stores the confirmed waitlist entry', async () => {
      const client = fakeClient({
        postRegistrations: () => gameFullConflict(),
        postWaitlist: () => waitlistOk(3),
      })
      const { register, joinWaitlist, gameFull, confirmedWaitlistEntry } = useJoinGame('g1', client)

      await register('player-mock-1')
      expect(gameFull.value).toBe(true)

      await joinWaitlist('player-mock-1')

      expect(confirmedWaitlistEntry.value).toMatchObject({ id: 'w1', position: 3 })
      expect(client.POST).toHaveBeenCalledWith('/v1/games/{gameId}/waitlist', {
        params: { path: { gameId: 'g1' } },
        body: { playerId: 'player-mock-1' },
      })
    })

    it('a joinWaitlist failure sets a distinct waitlistError, not registerError', async () => {
      const client = fakeClient({
        postRegistrations: () => gameFullConflict(),
        postWaitlist: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
      })
      const { register, joinWaitlist, waitlistError, confirmedWaitlistEntry } = useJoinGame('g1', client)

      await register('player-mock-1')
      await joinWaitlist('player-mock-1')

      expect(confirmedWaitlistEntry.value).toBeNull()
      expect(waitlistError.value).toBeTruthy()
    })
  })

  it('a non-conflict registration failure sets a generic registerError, not gameFull', async () => {
    const client = fakeClient({
      postRegistrations: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
    })
    const { register, registerError, gameFull } = useJoinGame('g1', client)

    await register('player-mock-1')

    expect(gameFull.value).toBe(false)
    expect(registerError.value).toBeTruthy()
  })

  it('sets an error message when the API is unreachable (fetch throws)', async () => {
    const client = fakeClient({
      postRegistrations: () => {
        throw new TypeError('Failed to fetch')
      },
    })
    const { register, registerError, registering } = useJoinGame('g1', client)

    await register('player-mock-1')

    expect(registerError.value).toBeTruthy()
    expect(registering.value).toBe(false)
  })
})
