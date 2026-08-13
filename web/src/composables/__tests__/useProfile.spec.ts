import { describe, it, expect, vi } from 'vitest'
import { useProfile } from '../useProfile'
import type { IdentityClient } from '../../api/identityClient'

function fakeClient(handlers: {
  get?: (userId: string) => unknown
  post?: (userId: string, body: unknown) => unknown
}): IdentityClient {
  const GET = vi.fn(async (path: string, options: { params: { path: { userId: string } } }) => {
    if (path === '/v1/users/{userId}') return handlers.get?.(options.params.path.userId)
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { params: { path: { userId: string } }; body: unknown }) => {
    if (path === '/v1/users/{userId}/selfReportedLevel') {
      return handlers.post?.(options.params.path.userId, options.body)
    }
    throw new Error(`unexpected POST ${path}`)
  })
  return { GET, POST } as unknown as IdentityClient
}

function userOk(level: number) {
  return {
    data: { user: { id: 'u1', displayName: 'Alex Rivera', roles: ['ROLE_PLAYER'], selfReportedStartingLevel: level } },
    error: undefined,
    response: { status: 200 },
  }
}

function notFound() {
  return { data: undefined, error: { message: 'user not found' }, response: { status: 404 } }
}

function permissionDenied() {
  return { data: undefined, error: { message: 'actor is not this user' }, response: { status: 403 } }
}

describe('useProfile', () => {
  it('starts with no profile, not loading, no error, not found false', () => {
    const { profile, loading, error, notFound: nf } = useProfile(fakeClient({}))
    expect(profile.value).toBeNull()
    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
    expect(nf.value).toBe(false)
  })

  it('loads and maps a found User, including a real self-reported level', async () => {
    const client = fakeClient({ get: () => userOk(3) })
    const { profile, error, notFound: nf, load } = useProfile(client)

    await load('u1')

    expect(error.value).toBeNull()
    expect(nf.value).toBe(false)
    expect(profile.value).toEqual({ id: 'u1', displayName: 'Alex Rivera', selfReportedStartingLevel: 3 })
  })

  // T10.5 instructions #4: "no fabricated fields — if SelfReportedStartingLevel
  // is unset, render an honest empty state, not a default value implying the
  // player chose one." The wire's zero value (proto default / omitted field)
  // must map to `null`, never to a displayed "0" or "1".
  it('maps an unset (wire-zero) self-reported level to null, not a fabricated default', async () => {
    const client = fakeClient({ get: () => userOk(0) })
    const { profile, load } = useProfile(client)

    await load('u1')

    expect(profile.value?.selfReportedStartingLevel).toBeNull()
  })

  // GetUser's malformed-ID boundary guard (T10.2 item 4) answers a
  // malformed id exactly like an unknown one — a 404 either way. This
  // proves the composable treats that 404 as the honest "no profile yet"
  // state (`notFound`), not a generic `error`.
  it('sets notFound (not error) on a 404, and leaves profile null', async () => {
    const client = fakeClient({ get: () => notFound() })
    const { profile, error, notFound: nf, load } = useProfile(client)

    await load('unknown-or-malformed')

    expect(profile.value).toBeNull()
    expect(error.value).toBeNull()
    expect(nf.value).toBe(true)
  })

  it('sets a generic error (not notFound) on a non-404 failure', async () => {
    const client = fakeClient({ get: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }) })
    const { profile, error, notFound: nf, load } = useProfile(client)

    await load('u1')

    expect(profile.value).toBeNull()
    expect(nf.value).toBe(false)
    expect(error.value).toBeTruthy()
  })

  it('resets notFound/error at the start of a new load, before the new request resolves', async () => {
    let resolveSecondGet!: (v: unknown) => void
    const get = vi
      .fn()
      .mockResolvedValueOnce(notFound())
      .mockImplementationOnce(() => new Promise((resolve) => (resolveSecondGet = resolve)))
    const client = { GET: get, POST: vi.fn() } as unknown as IdentityClient
    const { profile, notFound: nf, load } = useProfile(client)

    await load('unknown')
    expect(nf.value).toBe(true)

    const secondLoad = load('u1')
    expect(nf.value).toBe(false)
    expect(profile.value).toBeNull()

    resolveSecondGet(userOk(3))
    await secondLoad
    expect(profile.value?.selfReportedStartingLevel).toBe(3)
  })

  it('updateLevel saves a new level and re-maps the returned User onto profile', async () => {
    const client = fakeClient({ get: () => userOk(2), post: () => userOk(4) })
    const { profile, saveError, load, updateLevel } = useProfile(client)

    await load('u1')
    expect(profile.value?.selfReportedStartingLevel).toBe(2)

    await updateLevel('u1', 'u1', 4)

    expect(saveError.value).toBeNull()
    expect(profile.value?.selfReportedStartingLevel).toBe(4)
  })

  it('sets a specific saveError on a 403 (EnsureSelf mismatch), and leaves the prior profile untouched', async () => {
    const client = fakeClient({ get: () => userOk(2), post: () => permissionDenied() })
    const { profile, saveError, load, updateLevel } = useProfile(client)

    await load('u1')
    await updateLevel('u1', 'someone-else', 4)

    expect(saveError.value).toBeTruthy()
    expect(profile.value?.selfReportedStartingLevel).toBe(2)
  })

  it('sets saving true while the update is in flight, then false once resolved', async () => {
    let resolvePost!: (v: unknown) => void
    const post = vi.fn(() => new Promise((resolve) => (resolvePost = resolve)))
    const client = fakeClient({ post })
    const { saving, updateLevel } = useProfile(client)

    const promise = updateLevel('u1', 'u1', 3)
    expect(saving.value).toBe(true)

    resolvePost(userOk(3))
    await promise
    expect(saving.value).toBe(false)
  })

  it('sets an error when the API is unreachable (fetch throws)', async () => {
    const client = fakeClient({
      get: () => {
        throw new TypeError('Failed to fetch')
      },
    })
    const { profile, error, loading, load } = useProfile(client)

    await load('u1')

    expect(profile.value).toBeNull()
    expect(error.value).toBeTruthy()
    expect(loading.value).toBe(false)
  })
})
