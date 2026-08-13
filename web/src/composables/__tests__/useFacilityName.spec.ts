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
