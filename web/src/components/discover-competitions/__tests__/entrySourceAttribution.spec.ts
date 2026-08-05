// T9.7's key required test: an entry started from the DEEP LINK
// (/c/:shareToken) and an entry started IN-APP (/competitions) must produce
// DIFFERENT `source` values on the wire.
//
// This asserts the actual EnterCompetition request payload each path sends —
// not merely that both paths can enter successfully — because "the entry
// succeeded" is true for both and would prove nothing about attribution.
// `EntrySource` is a closed enum the server validates rather than infers
// (proto/pickleball/competitions/v1/competitions.proto), so the client is
// the only place this distinction can be made, and the only place a
// regression in it could hide.
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import DiscoverCompetitions from '../DiscoverCompetitions.vue'
import CompetitionLanding from '../../../views/CompetitionLanding.vue'
import type { CompetitionsClient } from '../../../api/competitionsClient'

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

const LISTING = { competition: COMPETITION, spotsLeft: 4 }

const ENTRY_OK = {
  data: {
    entry: {
      id: 'e1',
      competitionId: 'c1',
      playerId: 'player-mock-1',
      guestCount: 0,
      source: 'ENTRY_SOURCE_APP',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    },
  },
  error: undefined,
  response: { status: 200 },
}

/** Records every EnterCompetition body this client is asked to send. */
function recordingClient() {
  const sentBodies: Array<Record<string, unknown>> = []
  const GET = vi.fn(async (path: string) => {
    if (path === '/v1/competitions') {
      return { data: { competitions: [LISTING] }, error: undefined, response: { status: 200 } }
    }
    if (path === '/v1/competitions/by-share-token/{shareToken}') {
      return { data: { competition: COMPETITION }, error: undefined, response: { status: 200 } }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { body: Record<string, unknown> }) => {
    if (path === '/v1/competitions/{competitionId}/entries') {
      sentBodies.push(options.body)
      return ENTRY_OK
    }
    throw new Error(`unexpected POST ${path}`)
  })
  return { client: { GET, POST } as unknown as CompetitionsClient, sentBodies }
}

function matchMediaForWidth(width: number): Pick<Window, 'matchMedia'> {
  return {
    matchMedia: (query: string) => {
      const min = Number(query.match(/min-width:\s*(\d+)px/)?.[1] ?? -Infinity)
      const max = Number(query.match(/max-width:\s*(\d+)px/)?.[1] ?? Infinity)
      return {
        matches: width >= min && width <= max,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      } as unknown as MediaQueryList
    },
  }
}

async function enterFromInApp() {
  const { client, sentBodies } = recordingClient()
  const wrapper = mount(DiscoverCompetitions, {
    props: { client, win: matchMediaForWidth(1024), competitionId: 'c1' },
  })
  await flushPromises()
  await wrapper.get('.competition-entry__form').trigger('submit')
  await flushPromises()
  return sentBodies
}

async function enterFromDeepLink() {
  const { client, sentBodies } = recordingClient()
  const wrapper = mount(CompetitionLanding, {
    props: { client, shareToken: 'tok-123' },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  await wrapper.get('.competition-entry__form').trigger('submit')
  await flushPromises()
  return sentBodies
}

describe('entry source attribution: deep link vs. in-app', () => {
  it('an entry from /competitions sends source: ENTRY_SOURCE_APP', async () => {
    const bodies = await enterFromInApp()
    expect(bodies).toHaveLength(1)
    expect(bodies[0]).toEqual({ playerId: 'player-mock-1', guestCount: 0, source: 'ENTRY_SOURCE_APP' })
  })

  it('an entry from /c/:shareToken sends source: ENTRY_SOURCE_SOCIAL', async () => {
    const bodies = await enterFromDeepLink()
    expect(bodies).toHaveLength(1)
    expect(bodies[0]).toEqual({ playerId: 'player-mock-1', guestCount: 0, source: 'ENTRY_SOURCE_SOCIAL' })
  })

  it('the two paths differ on the wire, in the source field specifically', async () => {
    const [inApp] = await enterFromInApp()
    const [deepLink] = await enterFromDeepLink()

    expect(inApp.source).not.toBe(deepLink.source)
    // Everything ELSE about the two requests is identical — the only thing
    // the arrival path changes is the attribution.
    expect({ ...inApp, source: undefined }).toEqual({ ...deepLink, source: undefined })
  })
})
