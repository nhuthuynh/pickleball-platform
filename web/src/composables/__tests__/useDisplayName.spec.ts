import { describe, it, expect, vi } from 'vitest'
import { useDisplayName } from '../useDisplayName'
import type { IdentityClient } from '../../api/identityClient'

function fakeClient(handler: () => unknown): IdentityClient {
  return { GET: vi.fn(handler), POST: vi.fn() } as unknown as IdentityClient
}

describe('useDisplayName', () => {
  it('resolves a real DisplayName for a known user id', async () => {
    const client = fakeClient(() => ({ data: { user: { id: 'u1', displayName: 'Ada Lovelace' } } }))
    const { displayName, failed, load } = useDisplayName(client)

    await load('u1')

    expect(displayName.value).toBe('Ada Lovelace')
    expect(failed.value).toBe(false)
    expect(client.GET).toHaveBeenCalledWith('/v1/users/{userId}', { params: { path: { userId: 'u1' } } })
  })

  it('degrades honestly (failed=true, no displayName) for an empty user id — never calls the API', async () => {
    const client = fakeClient(() => ({ data: { user: { displayName: 'should not be reached' } } }))
    const { displayName, failed, load } = useDisplayName(client)

    await load('')

    expect(displayName.value).toBeNull()
    expect(failed.value).toBe(true)
    expect(client.GET).not.toHaveBeenCalled()
  })

  it('degrades honestly for an unknown/deleted user (NotFound)', async () => {
    const client = fakeClient(() => ({ data: undefined, error: { code: 5, message: 'not found' } }))
    const { displayName, failed, load } = useDisplayName(client)

    await load('gone')

    expect(displayName.value).toBeNull()
    expect(failed.value).toBe(true)
  })

  it('degrades honestly on a network error, never throwing', async () => {
    const client = { GET: vi.fn(async () => { throw new TypeError('Failed to fetch') }), POST: vi.fn() } as unknown as IdentityClient
    const { displayName, failed, load } = useDisplayName(client)

    await expect(load('u1')).resolves.toBeUndefined()
    expect(displayName.value).toBeNull()
    expect(failed.value).toBe(true)
  })

  it('tracks loading state across the request', async () => {
    let resolveGet: (v: unknown) => void = () => {}
    const client = {
      GET: vi.fn(() => new Promise((resolve) => { resolveGet = resolve })),
      POST: vi.fn(),
    } as unknown as IdentityClient
    const { loading, load } = useDisplayName(client)

    const pending = load('u1')
    expect(loading.value).toBe(true)

    resolveGet({ data: { user: { displayName: 'Ada' } } })
    await pending

    expect(loading.value).toBe(false)
  })
})
