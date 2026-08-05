import { describe, it, expect, vi } from 'vitest'
import { useCompetitionByShareToken } from '../useCompetitionByShareToken'
import type { CompetitionsClient } from '../../api/competitionsClient'

const COMPETITION = {
  id: 'c1',
  hostId: 'host-1',
  name: 'Autumn Doubles Ladder',
  venueFacilityId: 'facility-1',
  sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_CASH',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
}

function fakeClient(handlers: {
  byToken?: () => unknown
  list?: () => unknown
}): CompetitionsClient {
  const GET = vi.fn(async (path: string) => {
    if (path === '/v1/competitions/by-share-token/{shareToken}') {
      return handlers.byToken?.() ?? { data: undefined, error: { message: 'x' }, response: { status: 404 } }
    }
    if (path === '/v1/competitions') {
      return handlers.list?.() ?? { data: { competitions: [] }, error: undefined, response: { status: 200 } }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  return { GET, POST: vi.fn() } as unknown as CompetitionsClient
}

describe('useCompetitionByShareToken', () => {
  it('resolves a token into a Competition', async () => {
    const { load, competition, linkInvalid, error } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({ data: { competition: COMPETITION }, error: undefined, response: { status: 200 } }),
      }),
    )
    await load('tok-123')

    expect(competition.value?.name).toBe('Autumn Doubles Ladder')
    expect(linkInvalid.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('reports an unknown/expired token distinctly from a server failure', async () => {
    const { load, competition, linkInvalid, error } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({ data: undefined, error: { message: 'not found' }, response: { status: 404 } }),
      }),
    )
    await load('nope')

    expect(linkInvalid.value).toBe(true)
    expect(competition.value).toBeNull()
    expect(error.value).toBeNull()
  })

  it('reports a server failure as an error, not as an invalid link', async () => {
    const { load, linkInvalid, error } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
      }),
    )
    await load('tok-123')

    expect(linkInvalid.value).toBe(false)
    expect(error.value).toContain('Could not load')
  })

  it('a CANCELLED competition still resolves — a dead link and a cancelled event are different facts', async () => {
    const { load, competition, linkInvalid } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({
          data: { competition: { ...COMPETITION, status: 'COMPETITION_STATUS_CANCELLED' } },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await load('tok-123')

    expect(linkInvalid.value).toBe(false)
    expect(competition.value?.status).toBe('COMPETITION_STATUS_CANCELLED')
  })

  // GetCompetitionByShareToken deliberately returns no spots_left (its
  // response is field-for-field identical to GetCompetition). Rather than
  // invent a number, the composable makes a best-effort second read of the
  // public browse list, which DOES carry the server-computed value.
  it('enriches a scheduled competition with the real spots_left from ListCompetitions', async () => {
    const { load, competition } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({ data: { competition: COMPETITION }, error: undefined, response: { status: 200 } }),
        list: () => ({
          data: { competitions: [{ competition: COMPETITION, spotsLeft: 4 }] },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await load('tok-123')

    expect(competition.value?.spotsLeft).toBe(4)
  })

  it('leaves spotsLeft null — never a guess — when the browse read cannot supply one', async () => {
    const { load, competition } = useCompetitionByShareToken(
      fakeClient({
        byToken: () => ({ data: { competition: COMPETITION }, error: undefined, response: { status: 200 } }),
        list: () => ({ data: { competitions: [] }, error: undefined, response: { status: 200 } }),
      }),
    )
    await load('tok-123')

    expect(competition.value?.spotsLeft).toBeNull()
  })

  it('does not fail the whole landing when the best-effort spots_left read throws', async () => {
    const GET = vi.fn(async (path: string) => {
      if (path === '/v1/competitions/by-share-token/{shareToken}') {
        return { data: { competition: COMPETITION }, error: undefined, response: { status: 200 } }
      }
      throw new TypeError('Failed to fetch')
    })
    const { load, competition, error } = useCompetitionByShareToken({
      GET,
      POST: vi.fn(),
    } as unknown as CompetitionsClient)
    await load('tok-123')

    expect(competition.value?.name).toBe('Autumn Doubles Ladder')
    expect(competition.value?.spotsLeft).toBeNull()
    expect(error.value).toBeNull()
  })

  it('does not make the browse read at all for a cancelled competition', async () => {
    const client = fakeClient({
      byToken: () => ({
        data: { competition: { ...COMPETITION, status: 'COMPETITION_STATUS_CANCELLED' } },
        error: undefined,
        response: { status: 200 },
      }),
    })
    const { load } = useCompetitionByShareToken(client)
    await load('tok-123')

    expect(client.GET).toHaveBeenCalledTimes(1)
  })
})
