import { describe, expect, it } from 'vitest'

import { DEFAULT_API_BASE_URL, createTypedClient } from '../client'

/**
 * Same reason `client.spec.ts` carries one: the generated `paths` type is
 * gitignored, so a test that must pass on a fresh clone cannot import it.
 */
interface FixturePaths {
  '/v1/facilities': {
    get: {
      responses: { 200: { content: { 'application/json': { facilities: unknown[] } } } }
    }
  }
}

/**
 * #339. The default base URL must be relative, i.e. same-origin.
 *
 * It was `http://localhost:8080` — a different port from whatever serves the
 * app (`:5173` under both `vite` and the compose stack's nginx) — so every
 * request was cross-origin, and neither front door proxied `/v1` nor did the
 * Go server send CORS headers. A browser blocked the lot: `make up` produced
 * a rendered app that could not reach its own backend.
 *
 * No test caught it because every other test in this suite passes an
 * explicit `baseUrl` or a fixture `fetch`, so the default was never
 * exercised. These two assert the default itself.
 */
describe('the default API base URL', () => {
  it('is relative, so the page and the API share an origin', () => {
    expect(DEFAULT_API_BASE_URL).toBe('')
  })

  it('produces a same-origin request path rather than an absolute URL', async () => {
    let seen: string | undefined
    const client = createTypedClient<FixturePaths>({
      fetch: async (request: Request) => {
        seen = request.url
        return new Response('{}', { status: 200, headers: { 'content-type': 'application/json' } })
      },
    })

    await client.GET('/v1/facilities')

    // Relative input is resolved against the document origin by Request,
    // which in this environment is localhost:3000 — the point is that it is
    // the PAGE's origin and not a hard-coded :8080.
    expect(seen).toMatch(/\/v1\/facilities$/)
    expect(seen).not.toContain(':8080')
  })
})
