import { describe, it, expect, vi } from 'vitest'
import { createTypedClient, DEFAULT_API_BASE_URL } from '../client'

/**
 * A minimal, hand-written fixture standing in for the `paths` type
 * `openapi-typescript` would emit for a real swagger.json — shaped after
 * Booking's actual `GetQuote` REST endpoint
 * (proto/pickleball/booking/v1/booking.proto) so the fixture reads as
 * realistic, not arbitrary. The real, generated version lives at
 * web/src/api/generated/booking.d.ts once `npm run generate:client` has
 * run (gitignored — not present in a fresh checkout, which is exactly why
 * this test uses its own fixture instead of importing the real one: this
 * test must pass on a fresh clone with no toolchain run yet).
 */
interface FixturePaths {
  '/v1/courts/{courtId}/quote': {
    get: {
      parameters: { path: { courtId: string } }
      responses: {
        200: { content: { 'application/json': { priceCents: number; band: string } } }
      }
    }
  }
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('createTypedClient', () => {
  it('issues requests against DEFAULT_API_BASE_URL when no baseUrl override is given', async () => {
    const fetchMock = vi.fn(async (_req: Request) => jsonResponse({ priceCents: 1800, band: 'peak' }))
    const client = createTypedClient<FixturePaths>({ fetch: fetchMock })

    const { data, error } = await client.GET('/v1/courts/{courtId}/quote', {
      params: { path: { courtId: 'court-1' } },
    })

    expect(error).toBeUndefined()
    expect(data).toEqual({ priceCents: 1800, band: 'peak' })
    expect(fetchMock).toHaveBeenCalledTimes(1)

    const requestArg = fetchMock.mock.calls[0]![0]
    expect(requestArg.url.startsWith(DEFAULT_API_BASE_URL)).toBe(true)
    expect(requestArg.url).toContain('/v1/courts/court-1/quote')
  })

  it('honors an explicit baseUrl override instead of DEFAULT_API_BASE_URL', async () => {
    const fetchMock = vi.fn(async (_req: Request) => jsonResponse({ priceCents: 1200, band: 'off-peak' }))
    const client = createTypedClient<FixturePaths>({ baseUrl: 'https://api.example.test', fetch: fetchMock })

    await client.GET('/v1/courts/{courtId}/quote', { params: { path: { courtId: 'abc' } } })

    const requestArg = fetchMock.mock.calls[0]![0]
    expect(requestArg.url.startsWith('https://api.example.test')).toBe(true)
    expect(requestArg.url.startsWith(DEFAULT_API_BASE_URL)).toBe(false)
  })

  it('surfaces a non-2xx response as `error`, not a thrown exception', async () => {
    const fetchMock = vi.fn(async () => jsonResponse({ message: 'court not found' }, 404))
    const client = createTypedClient<FixturePaths>({ fetch: fetchMock })

    const { data, error } = await client.GET('/v1/courts/{courtId}/quote', {
      params: { path: { courtId: 'missing' } },
    })

    expect(data).toBeUndefined()
    expect(error).toEqual({ message: 'court not found' })
  })
})
