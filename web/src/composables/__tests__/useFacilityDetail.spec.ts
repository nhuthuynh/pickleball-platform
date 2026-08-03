import { describe, it, expect, vi } from 'vitest'
import { useFacilityDetail } from '../useFacilityDetail'
import type { FacilitiesClient } from '../../api/facilitiesClient'

function fakeClient(get: (...args: unknown[]) => unknown): FacilitiesClient {
  return { GET: get } as unknown as FacilitiesClient
}

describe('useFacilityDetail', () => {
  it('starts with no facility, not loading, no error', () => {
    const { facility, loading, error } = useFacilityDetail(fakeClient(vi.fn()))
    expect(facility.value).toBeNull()
    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('sets loading true while in flight, then false once resolved', async () => {
    let resolveGet!: (v: unknown) => void
    const get = vi.fn(() => new Promise((resolve) => (resolveGet = resolve)))
    const { loading, load } = useFacilityDetail(fakeClient(get))

    const promise = load('f1')
    expect(loading.value).toBe(true)

    resolveGet({
      data: { facility: { id: 'f1', name: 'Riverside Courts', description: '', address: '', photoUrls: [] } },
      error: undefined,
    })
    await promise
    expect(loading.value).toBe(false)
  })

  it('maps a successful GetFacility response, including a facility with zero courts', async () => {
    const get = vi.fn(async () => ({
      data: {
        facility: {
          id: 'f1',
          name: 'Riverside Courts',
          description: 'Six outdoor courts.',
          address: '1 River Rd',
          photoUrls: ['https://cdn.example.test/1.jpg'],
        },
      },
      error: undefined,
    }))
    const { facility, error, load } = useFacilityDetail(fakeClient(get))

    await load('f1')

    expect(error.value).toBeNull()
    expect(facility.value).toEqual({
      id: 'f1',
      name: 'Riverside Courts',
      description: 'Six outdoor courts.',
      address: '1 River Rd',
      photoUrls: ['https://cdn.example.test/1.jpg'],
      courts: [],
    })
  })

  // T8.2: GetFacilityResponse now carries a sibling `courts` list —
  // previously this always mapped to `[]` (see models/facility.ts's prior
  // doc comment) since no endpoint returned a facility's courts at all.
  // This proves the composable actually forwards `data.courts` into
  // `mapToFacilityDetail`, not just that the mapping function itself can
  // handle a populated list (models/__tests__/facility.spec.ts covers
  // that in isolation).
  it('maps a populated courts list from GetFacilityResponse.courts onto the loaded facility', async () => {
    const get = vi.fn(async () => ({
      data: {
        facility: {
          id: 'f1',
          name: 'Riverside Courts',
          description: 'Six outdoor courts.',
          address: '1 River Rd',
          photoUrls: [],
        },
        courts: [
          { id: 'c1', facilityId: 'f1', name: 'Court 1' },
          { id: 'c2', facilityId: 'f1', name: 'Court 2' },
        ],
      },
      error: undefined,
    }))
    const { facility, error, load } = useFacilityDetail(fakeClient(get))

    await load('f1')

    expect(error.value).toBeNull()
    expect(facility.value?.courts).toEqual([
      { id: 'c1', name: 'Court 1' },
      { id: 'c2', name: 'Court 2' },
    ])
  })

  it('never carries camera-link data onto the loaded facility, even when the mocked API response includes it', async () => {
    const get = vi.fn(async () => ({
      data: {
        facility: {
          id: 'f1',
          name: 'Riverside Courts',
          description: '',
          address: '1 River Rd',
          photoUrls: [],
          cameraConsentAttested: true,
          cameraLinks: [{ url: 'https://cameras.example.test/riverside/secret-feed', courtId: '' }],
        },
      },
      error: undefined,
    }))
    const { facility, load } = useFacilityDetail(fakeClient(get))

    await load('f1')

    expect(facility.value).not.toHaveProperty('cameraLinks')
    expect(facility.value).not.toHaveProperty('cameraConsentAttested')
    expect(JSON.stringify(facility.value)).not.toContain('cameras.example.test')
  })

  it('sets an error and leaves facility null when the API returns a non-2xx error', async () => {
    const get = vi.fn(async () => ({ data: undefined, error: { message: 'not found' } }))
    const { facility, error, load } = useFacilityDetail(fakeClient(get))

    await load('missing')

    expect(facility.value).toBeNull()
    expect(error.value).toBeTruthy()
  })

  it('sets an error when the API is unreachable (fetch throws)', async () => {
    const get = vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    const { facility, error, loading, load } = useFacilityDetail(fakeClient(get))

    await load('f1')

    expect(facility.value).toBeNull()
    expect(error.value).toBeTruthy()
    expect(loading.value).toBe(false)
  })

  it('resets facility/error at the start of a new load', async () => {
    const get = vi
      .fn()
      .mockResolvedValueOnce({ data: undefined, error: { message: 'not found' } })
      .mockImplementationOnce(() => new Promise(() => {})) // never resolves for this assertion
    const { facility, error, load } = useFacilityDetail(fakeClient(get))

    await load('missing')
    expect(error.value).toBeTruthy()

    void load('f1')
    expect(facility.value).toBeNull()
    expect(error.value).toBeNull()
  })
})
