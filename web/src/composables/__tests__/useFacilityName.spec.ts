import { describe, it, expect, vi } from 'vitest'
import { useFacilityName } from '../useFacilityName'
import type { FacilitiesClient } from '../../api/facilitiesClient'

function fakeClient(handler: () => unknown): FacilitiesClient {
  return { GET: vi.fn(handler), POST: vi.fn() } as unknown as FacilitiesClient
}

describe('useFacilityName', () => {
  it('resolves a real Name for a known facility id', async () => {
    const client = fakeClient(() => ({ data: { facility: { id: 'f1', name: 'Riverside Courts' } } }))
    const { name, failed, load } = useFacilityName(client)

    await load('f1')

    expect(name.value).toBe('Riverside Courts')
    expect(failed.value).toBe(false)
    expect(client.GET).toHaveBeenCalledWith('/v1/facilities/{facilityId}', {
      params: { path: { facilityId: 'f1' } },
    })
  })

  it('treats an empty facility id as "no venue set" — honest, not a failure, and no API call', async () => {
    const client = fakeClient(() => ({ data: { facility: { name: 'should not be reached' } } }))
    const { name, failed, load } = useFacilityName(client)

    await load('')

    expect(name.value).toBeNull()
    expect(failed.value).toBe(false)
    expect(client.GET).not.toHaveBeenCalled()
  })

  it('degrades honestly (failed=true) for an unknown/deleted facility', async () => {
    const client = fakeClient(() => ({ data: undefined, error: { code: 5, message: 'not found' } }))
    const { name, failed, load } = useFacilityName(client)

    await load('gone')

    expect(name.value).toBeNull()
    expect(failed.value).toBe(true)
  })

  // T10.8 PR review: VenueName.vue is never remounted when its facilityId
  // prop changes, so a component instance's `load()` can be called again
  // for a new id while an earlier call is still in flight. Without the
  // request-sequencing guard, this test fails: the earlier ('fac-a')
  // response lands last and overwrites the correct, newer ('fac-b') state.
  it('discards a stale response that resolves after a newer request has already started', async () => {
    const resolvers: Array<(v: unknown) => void> = []
    const GET = vi.fn(() => new Promise((resolve) => { resolvers.push(resolve) }))
    const client = { GET, POST: vi.fn() } as unknown as FacilitiesClient
    const { name, failed, load } = useFacilityName(client)

    const first = load('fac-a') // starts first, resolves SLOWER
    const second = load('fac-b') // starts second, resolves FASTER

    expect(resolvers).toHaveLength(2)

    resolvers[1]!({ data: { facility: { name: 'Lakeside Courts' } } })
    await second
    expect(name.value).toBe('Lakeside Courts')
    expect(failed.value).toBe(false)

    // The earlier, now-stale request (fac-a) finally resolves — must be
    // discarded, not clobber the newer state.
    resolvers[0]!({ data: { facility: { name: 'Riverside Courts' } } })
    await first
    expect(name.value).toBe('Lakeside Courts')
    expect(failed.value).toBe(false)
  })

  it('degrades honestly on a network error, never throwing', async () => {
    const client = {
      GET: vi.fn(async () => { throw new TypeError('Failed to fetch') }),
      POST: vi.fn(),
    } as unknown as FacilitiesClient
    const { name, failed, load } = useFacilityName(client)

    await expect(load('f1')).resolves.toBeUndefined()
    expect(name.value).toBeNull()
    expect(failed.value).toBe(true)
  })
})
