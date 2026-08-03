import { describe, it, expect, vi } from 'vitest'
import { useGameList } from '../useGameList'
import type { SocialPlayClient } from '../../api/socialplayClient'

function fakeClient(get: (...args: unknown[]) => unknown): SocialPlayClient {
  return { GET: get, POST: vi.fn() } as unknown as SocialPlayClient
}

const GAME_LISTING = {
  game: {
    id: 'g1',
    hostId: 'host-1',
    venueFacilityId: 'facility-1',
    courtIds: ['court-1'],
    startsAt: '2026-09-01T10:00:00Z',
    endsAt: '2026-09-01T11:00:00Z',
    capacity: 8,
    status: 'GAME_STATUS_SCHEDULED',
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    guestAllowance: 2,
  },
  spotsLeft: 5,
}

describe('useGameList', () => {
  it('starts empty, not loading, no error', () => {
    const { games, loading, error } = useGameList(fakeClient(vi.fn()))
    expect(games.value).toEqual([])
    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('sets loading true while the request is in flight, then false once resolved', async () => {
    let resolveGet!: (v: unknown) => void
    const get = vi.fn(() => new Promise((resolve) => (resolveGet = resolve)))
    const { loading, search } = useGameList(fakeClient(get))

    const promise = search()
    expect(loading.value).toBe(true)

    resolveGet({ data: { games: [] }, error: undefined })
    await promise
    expect(loading.value).toBe(false)
  })

  it('maps a successful response into game summaries, including SpotsLeft', async () => {
    const get = vi.fn(async () => ({ data: { games: [GAME_LISTING] }, error: undefined }))
    const { games, search } = useGameList(fakeClient(get))

    await search()

    expect(games.value).toEqual([
      {
        id: 'g1',
        hostId: 'host-1',
        venueFacilityId: 'facility-1',
        courtIds: ['court-1'],
        startsAt: '2026-09-01T10:00:00Z',
        endsAt: '2026-09-01T11:00:00Z',
        capacity: 8,
        status: 'GAME_STATUS_SCHEDULED',
        paymentMethod: 'PAYMENT_METHOD_EITHER',
        guestAllowance: 2,
        spotsLeft: 5,
      },
    ])
  })

  it('no games yet -> empty list, not an error', async () => {
    const get = vi.fn(async () => ({ data: { games: [] }, error: undefined }))
    const { games, error, search } = useGameList(fakeClient(get))

    await search()

    expect(games.value).toEqual([])
    expect(error.value).toBeNull()
  })

  it('sets an error message and clears games when the API returns a non-2xx error', async () => {
    const get = vi.fn(async () => ({ data: undefined, error: { message: 'boom' } }))
    const { games, error, search } = useGameList(fakeClient(get))

    await search()

    expect(games.value).toEqual([])
    expect(error.value).toBeTruthy()
  })

  it('sets an error message when the API is unreachable (fetch throws)', async () => {
    const get = vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    const { games, error, loading, search } = useGameList(fakeClient(get))

    await search()

    expect(games.value).toEqual([])
    expect(error.value).toBeTruthy()
    expect(loading.value).toBe(false)
  })

  it('omits venueFacilityId/startsAfter/startsBefore query params when both filters are empty', async () => {
    const get = vi.fn(async () => ({ data: { games: [] }, error: undefined }))
    const { search } = useGameList(fakeClient(get))

    await search()

    expect(get).toHaveBeenCalledWith('/v1/games', { params: { query: {} } })
  })

  it('passes venueFacilityId when facilityFilter is set', async () => {
    const get = vi.fn(async () => ({ data: { games: [] }, error: undefined }))
    const { facilityFilter, search } = useGameList(fakeClient(get))
    facilityFilter.value = 'facility-1'

    await search()

    expect(get).toHaveBeenCalledWith('/v1/games', {
      params: { query: { venueFacilityId: 'facility-1' } },
    })
  })

  it('converts dateFilter into a startsAfter/startsBefore local-day window', async () => {
    let capturedQuery: Record<string, string> | undefined
    const get = vi.fn(async (...args: unknown[]) => {
      const options = args[1] as { params: { query: Record<string, string> } }
      capturedQuery = options.params.query
      return { data: { games: [] }, error: undefined }
    })
    const { dateFilter, search } = useGameList(fakeClient(get))
    dateFilter.value = '2026-09-01'

    await search()

    expect(capturedQuery?.startsAfter).toBeTruthy()
    expect(capturedQuery?.startsBefore).toBeTruthy()
    const start = new Date(capturedQuery!.startsAfter!)
    const end = new Date(capturedQuery!.startsBefore!)
    expect(end.getTime() - start.getTime()).toBe(24 * 60 * 60 * 1000)
  })
})
