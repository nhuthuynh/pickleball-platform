import { describe, it, expect, vi } from 'vitest'
import { useEnterCompetition } from '../useEnterCompetition'
import type { CompetitionsClient } from '../../api/competitionsClient'

function fakeClient(post: (body: unknown) => unknown): CompetitionsClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/competitions/{competitionId}/entries') return post(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as CompetitionsClient
}

const OK_ENTRY = {
  data: {
    entry: {
      id: 'e1',
      competitionId: 'c1',
      playerId: 'player-mock-1',
      guestCount: 0,
      source: 'ENTRY_SOURCE_APP',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    },
  },
  error: undefined,
  response: { status: 200 },
}

describe('useEnterCompetition — guest stepper', () => {
  it('never exceeds the guest allowance passed in', () => {
    const { guestCount, incrementGuests } = useEnterCompetition('c1', 'ENTRY_SOURCE_APP', fakeClient(() => OK_ENTRY))
    incrementGuests(2)
    incrementGuests(2)
    incrementGuests(2)
    expect(guestCount.value).toBe(2)
  })

  it('never goes below zero', () => {
    const { guestCount, decrementGuests } = useEnterCompetition('c1', 'ENTRY_SOURCE_APP', fakeClient(() => OK_ENTRY))
    decrementGuests()
    expect(guestCount.value).toBe(0)
  })
})

describe('useEnterCompetition — source attribution', () => {
  it('sends the source it was constructed with on the wire', async () => {
    const client = fakeClient(() => OK_ENTRY)
    const { enter } = useEnterCompetition('c1', 'ENTRY_SOURCE_SOCIAL', client)
    await enter('player-mock-1')

    expect(client.POST).toHaveBeenCalledWith('/v1/competitions/{competitionId}/entries', {
      params: { path: { competitionId: 'c1' } },
      body: { playerId: 'player-mock-1', guestCount: 0, source: 'ENTRY_SOURCE_SOCIAL' },
    })
  })
})

describe('useEnterCompetition — rejection paths', () => {
  // ErrCompetitionFull maps to codes.AlreadyExists (409) — confirmed against
  // internal/competitions/adapter/grpcapi/handler.go's toStatus, which the
  // T9.4 review changed from FailedPrecondition.
  it('a 409 naming capacity is the competition-full path with a specific message', async () => {
    const { enter, competitionFull, enterError, alreadyEntered } = useEnterCompetition(
      'c1',
      'ENTRY_SOURCE_APP',
      fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: competition is at capacity' },
        response: { status: 409 },
      })),
    )
    await enter('player-mock-1')

    expect(competitionFull.value).toBe(true)
    expect(alreadyEntered.value).toBe(false)
    expect(enterError.value).toContain('This competition just filled up')
  })

  // ErrAlreadyEntered ALSO maps to AlreadyExists/409, so the status code
  // alone cannot tell the two apart — the message has to be read.
  it('a 409 naming a duplicate entry is NOT reported as the competition being full', async () => {
    const { enter, competitionFull, alreadyEntered, enterError } = useEnterCompetition(
      'c1',
      'ENTRY_SOURCE_APP',
      fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: player has already entered this competition' },
        response: { status: 409 },
      })),
    )
    await enter('player-mock-1')

    expect(alreadyEntered.value).toBe(true)
    expect(competitionFull.value).toBe(false)
    expect(enterError.value).toContain("already entered")
  })

  it('a cancelled competition gets its own message', async () => {
    const { enter, enterError, competitionFull } = useEnterCompetition(
      'c1',
      'ENTRY_SOURCE_APP',
      fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: competition is cancelled' },
        response: { status: 400 },
      })),
    )
    await enter('player-mock-1')

    expect(competitionFull.value).toBe(false)
    expect(enterError.value).toContain('cancelled')
  })

  it('a guest-allowance rejection says what to change', async () => {
    const { enter, enterError } = useEnterCompetition(
      'c1',
      'ENTRY_SOURCE_APP',
      fakeClient(() => ({
        data: undefined,
        error: { message: "competitions: guest count exceeds this competition's guest allowance" },
        response: { status: 400 },
      })),
    )
    await enter('player-mock-1')

    expect(enterError.value).toContain('guests')
  })

  it('an unreachable server says so rather than failing silently', async () => {
    const client = {
      POST: vi.fn(async () => {
        throw new TypeError('Failed to fetch')
      }),
      GET: vi.fn(),
    } as unknown as CompetitionsClient
    const { enter, enterError } = useEnterCompetition('c1', 'ENTRY_SOURCE_APP', client)
    await enter('player-mock-1')

    expect(enterError.value).toContain('Could not reach the server')
  })
})
