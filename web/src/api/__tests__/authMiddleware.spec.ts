import { afterEach, describe, expect, it } from 'vitest'

import { createTypedClient } from '../client'
import { authRejected, authToken, clearAuthRejected, signIn, signOut } from '../../state/authSession'

/**
 * T65.1. These are the tests that would have failed before this ticket, and
 * the reason they are worth having is in `state/authSession.ts`'s header: the
 * server has enforced authentication on 31 RPCs since T55.1 and this client
 * could not send a token, so every write path in the shipped UI answered
 * `Unauthenticated` — while 717 green tests mounted a fixture client that
 * never looked at a header.
 *
 * So each test here asserts something about the **wire**, not about a
 * composable's return value. That distinction is the whole finding.
 */
type Probe = { capturedAuth: string | null; fetch: (input: Request) => Promise<Response> }

function probe(status = 200): Probe {
  const p: Probe = {
    capturedAuth: null,
    fetch: async (input: Request) => {
      p.capturedAuth = input.headers.get('Authorization')
      return new Response(JSON.stringify({}), {
        status,
        headers: { 'content-type': 'application/json' },
      })
    },
  }
  return p
}

// A minimal `paths` shape: this file must not depend on generated output
// being present, per client.ts's own rationale.
type Paths = {
  '/v1/thing': { post: { responses: { 200: { content: { 'application/json': unknown } } } } }
}

afterEach(() => {
  signOut()
  clearAuthRejected()
})

describe('auth middleware', () => {
  it('sends no Authorization header when nobody is signed in', async () => {
    const p = probe()
    const client = createTypedClient<Paths>({ fetch: p.fetch })

    await client.POST('/v1/thing')

    expect(p.capturedAuth).toBeNull()
  })

  it('attaches the session token as a bearer header once signed in', async () => {
    const p = probe()
    signIn('header.payload.signature')
    const client = createTypedClient<Paths>({ fetch: p.fetch })

    await client.POST('/v1/thing')

    expect(p.capturedAuth).toBe('Bearer header.payload.signature')
  })

  it('attaches the token on a client created BEFORE sign-in', async () => {
    // The regression that matters for a SPA: the client is constructed once at
    // module load, and a user signs in later. A middleware that captured the
    // token at creation time would send nothing for the rest of the session.
    const p = probe()
    const client = createTypedClient<Paths>({ fetch: p.fetch })

    signIn('late.signed.in')
    await client.POST('/v1/thing')

    expect(p.capturedAuth).toBe('Bearer late.signed.in')
  })

  it('clears the token and raises one signal on a 401', async () => {
    const p = probe(401)
    signIn('stale.token.here', 'user-1')
    const client = createTypedClient<Paths>({ fetch: p.fetch })

    await client.POST('/v1/thing')

    expect(authRejected.value).toBe(true)
    // A stale token left in place makes every later call fail identically with
    // no explanation, which is the shape of bug a user cannot report usefully.
    expect(authToken.value).toBeNull()
  })

  it('does not raise the signal on a non-401 failure', async () => {
    const p = probe(500)
    signIn('good.token.here')
    const client = createTypedClient<Paths>({ fetch: p.fetch })

    await client.POST('/v1/thing')

    expect(authRejected.value).toBe(false)
    expect(authToken.value).toBe('good.token.here')
  })

  it('can be opted out of, for tests that assert the unauthed wire shape', async () => {
    const p = probe()
    signIn('present.but.unused')
    const client = createTypedClient<Paths>({ fetch: p.fetch, withoutAuth: true })

    await client.POST('/v1/thing')

    expect(p.capturedAuth).toBeNull()
  })
})
