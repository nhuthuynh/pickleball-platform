// T11.6 — the actor's real Roles, read from Identity's GetUser, which is what
// the Club screen's rental-request control is gated on.
//
// The whole point of this composable is that it distinguishes THREE states
// where a naive `isClub: boolean` would have two, and every test here pins one
// of them. Collapsing "we could not check" into "you are not a club" is how a
// screen ends up making a claim about someone's account that it never
// actually verified.
import { describe, it, expect, vi } from 'vitest'
import { useActorRoles } from '../useActorRoles'
import type { IdentityClient } from '../../api/identityClient'

const USER_PATH = '/v1/users/{userId}'
const ACTOR = '00000000-0000-4000-b000-000000000010'

function fakeClient(result: unknown | (() => never)): IdentityClient {
  const GET = vi.fn(async (path: string) => {
    if (path !== USER_PATH) throw new Error(`unexpected GET ${path}`)
    if (typeof result === 'function') return (result as () => never)()
    return result
  })
  return { GET } as unknown as IdentityClient
}

describe('useActorRoles', () => {
  it('reports the club role when the user really holds it', async () => {
    const roles = useActorRoles(
      fakeClient({
        data: { user: { id: ACTOR, roles: ['ROLE_PLAYER', 'ROLE_CLUB'] } },
        error: undefined,
        response: { status: 200 },
      }),
    )

    await roles.load(ACTOR)

    expect(roles.resolved.value).toBe(true)
    expect(roles.isClub.value).toBe(true)
    expect(roles.error.value).toBeNull()
  })

  it('resolves to "not a club" when the roles were read and club is absent', async () => {
    const roles = useActorRoles(
      fakeClient({
        data: { user: { id: ACTOR, roles: ['ROLE_PLAYER'] } },
        error: undefined,
        response: { status: 200 },
      }),
    )

    await roles.load(ACTOR)

    expect(roles.resolved.value).toBe(true)
    expect(roles.isClub.value).toBe(false)
  })

  // An absent `roles` array is not a licence to guess. It maps to [] and the
  // gate stays closed.
  it('treats a user with no roles field as holding no roles', async () => {
    const roles = useActorRoles(
      fakeClient({ data: { user: { id: ACTOR } }, error: undefined, response: { status: 200 } }),
    )

    await roles.load(ACTOR)

    expect(roles.roles.value).toEqual([])
    expect(roles.isClub.value).toBe(false)
    expect(roles.resolved.value).toBe(true)
  })

  // A 404 IS an answer: this id is nobody, so there is genuinely no club role
  // to find. Resolved, not an error to shout about — today's mock identity
  // takes exactly this path (see useProfile's own note).
  it('treats "no such user" as a resolved answer rather than a failure', async () => {
    const roles = useActorRoles(
      fakeClient({ data: undefined, error: { message: 'not found' }, response: { status: 404 } }),
    )

    await roles.load(ACTOR)

    expect(roles.notFound.value).toBe(true)
    expect(roles.resolved.value).toBe(true)
    expect(roles.isClub.value).toBe(false)
    expect(roles.error.value).toBeNull()
  })

  // The third state, and the one that matters: the lookup FAILED. The gate
  // fails closed (isClub false), but `resolved` stays false so no caller can
  // report "you are not a club" on the strength of a failed request.
  it.each([
    ['a server error', { data: undefined, error: { message: 'boom' }, response: { status: 500 } }],
    ['an unreachable server', () => { throw new Error('network down') }],
  ])('fails closed on %s without claiming the actor is not a club', async (_label, result) => {
    const roles = useActorRoles(fakeClient(result))

    await roles.load(ACTOR)

    expect(roles.isClub.value).toBe(false)
    expect(roles.resolved.value).toBe(false)
    expect(roles.error.value).toMatch(/could not (check|reach)/i)
  })

  it('is not a club before any lookup has happened', () => {
    const roles = useActorRoles(fakeClient({ data: undefined, error: undefined }))

    expect(roles.isClub.value).toBe(false)
    expect(roles.resolved.value).toBe(false)
  })
})
