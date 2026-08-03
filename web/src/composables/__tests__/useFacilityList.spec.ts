import { describe, it, expect, vi } from 'vitest'
import { useFacilityList } from '../useFacilityList'
import type { FacilitiesClient } from '../../api/facilitiesClient'

function fakeClient(get: (...args: unknown[]) => unknown): FacilitiesClient {
  return { GET: get } as unknown as FacilitiesClient
}

describe('useFacilityList', () => {
  it('starts empty, not loading, no error', () => {
    const { facilities, loading, error } = useFacilityList(fakeClient(vi.fn()))
    expect(facilities.value).toEqual([])
    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('sets loading true while the request is in flight, then false once resolved', async () => {
    let resolveGet!: (v: unknown) => void
    const get = vi.fn(() => new Promise((resolve) => (resolveGet = resolve)))
    const { loading, search } = useFacilityList(fakeClient(get))

    const promise = search()
    expect(loading.value).toBe(true)

    resolveGet({ data: { facilities: [] }, error: undefined })
    await promise
    expect(loading.value).toBe(false)
  })

  it('populates facilities and clears error on a successful search (no facilities yet -> empty list, not an error)', async () => {
    const get = vi.fn(async () => ({ data: { facilities: [] }, error: undefined }))
    const { facilities, error, search } = useFacilityList(fakeClient(get))

    await search()

    expect(facilities.value).toEqual([])
    expect(error.value).toBeNull()
  })

  it('maps a successful response into facility summaries', async () => {
    const get = vi.fn(async () => ({
      data: {
        facilities: [
          { id: 'f1', name: 'Riverside Courts', description: 'Nice courts', address: '1 River Rd' },
        ],
      },
      error: undefined,
    }))
    const { facilities, search } = useFacilityList(fakeClient(get))

    await search()

    expect(facilities.value).toEqual([
      { id: 'f1', name: 'Riverside Courts', description: 'Nice courts', address: '1 River Rd' },
    ])
  })

  it('sets an error message and clears facilities when the API returns a non-2xx error', async () => {
    const get = vi.fn(async () => ({ data: undefined, error: { message: 'boom' } }))
    const { facilities, error, search } = useFacilityList(fakeClient(get))

    await search()

    expect(facilities.value).toEqual([])
    expect(error.value).toBeTruthy()
  })

  it('sets an error message when the API is unreachable (fetch throws)', async () => {
    const get = vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    const { facilities, error, loading, search } = useFacilityList(fakeClient(get))

    await search()

    expect(facilities.value).toEqual([])
    expect(error.value).toBeTruthy()
    expect(loading.value).toBe(false)
  })

  it('passes the current nameFilter as a query param, and an explicit filter argument updates it', async () => {
    const get = vi.fn(async () => ({ data: { facilities: [] }, error: undefined }))
    const { nameFilter, search } = useFacilityList(fakeClient(get))

    await search('river')

    expect(nameFilter.value).toBe('river')
    expect(get).toHaveBeenCalledWith('/v1/facilities', { params: { query: { nameFilter: 'river' } } })
  })

  it('omits the nameFilter query param entirely when the filter is empty', async () => {
    const get = vi.fn(async () => ({ data: { facilities: [] }, error: undefined }))
    const { search } = useFacilityList(fakeClient(get))

    await search('')

    expect(get).toHaveBeenCalledWith('/v1/facilities', { params: { query: {} } })
  })
})
