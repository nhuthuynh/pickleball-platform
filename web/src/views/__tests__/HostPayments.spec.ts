import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import HostPayments from '../HostPayments.vue'
import { MOCK_HOST_ID } from '../../composables/useHostPayments'
import { rememberUserId, signOut } from '../../state/authSession'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { PaymentsClient } from '../../api/paymentsClient'

function matchMediaForWidth(width: number): Pick<Window, 'matchMedia'> {
  return {
    matchMedia: (query: string) => {
      const min = Number(query.match(/min-width:\s*(\d+)px/)?.[1] ?? -Infinity)
      return { matches: width >= min, media: query, addEventListener: () => {}, removeEventListener: () => {} } as unknown as MediaQueryList
    },
  }
}

function gameListing(id: string, hostId: string, paymentMethod: string) {
  return {
    game: {
      id,
      hostId,
      venueFacilityId: 'facility-1',
      courtIds: ['court-1'],
      startsAt: '2026-09-01T10:00:00Z',
      endsAt: '2026-09-01T11:00:00Z',
      capacity: 8,
      status: 'GAME_STATUS_SCHEDULED',
      paymentMethod,
      guestAllowance: 2,
      entryFee: { amountCents: '1000', currencyCode: 'USD' },
    },
    spotsLeft: 5,
  }
}

/** The id a signed-in host's Games actually carry: the server sets
 *  Game.host_id from the verified caller, never from the wire (T12.9), so it
 *  is a users.id uuid and can never be MOCK_HOST_ID. */
const REAL_HOST_ID = 'c6b3598f-fa52-4a7b-848c-7e5e96cdf188'

function socialplayClientStub(): SocialPlayClient {
  return socialplayClientStubFor(MOCK_HOST_ID)
}

/** Same stub, with the host id the fixture Game is hosted by made explicit —
 *  so a test can seed a Game hosted by a REAL user id. */
function socialplayClientStubFor(hostId: string): SocialPlayClient {
  return {
    GET: vi.fn(async (path: string, options: { params?: { path?: { gameId?: string } } }) => {
      if (path === '/v1/games') {
        return {
          data: { games: [gameListing('g1', hostId, 'PAYMENT_METHOD_CASH')] },
          error: undefined,
          response: { status: 200 },
        }
      }
      if (path === '/v1/games/{gameId}/registrations' && options.params?.path?.gameId === 'g1') {
        return {
          data: {
            registrations: [
              // T56.1: two heads at this Game's entry fee — the shape a
              // real server returns. Without amount_owed, T58's filter
              // (a row whose owed amount is unknown cannot be paid for,
              // so it is not offered) correctly drops this fixture.
              { id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 1, amountOwed: { amountCents: '2000', currencyCode: 'USD' } },
            ],
          },
          error: undefined,
          response: { status: 200 },
        }
      }
      return { data: { registrations: [] }, error: undefined, response: { status: 200 } }
    }),
    POST: vi.fn(),
  } as unknown as SocialPlayClient
}

function emptyClient(): SocialPlayClient {
  return {
    GET: vi.fn(async () => ({ data: { games: [] }, error: undefined, response: { status: 200 } })),
    POST: vi.fn(),
  } as unknown as SocialPlayClient
}

function paymentsClientStub(): PaymentsClient {
  return {
    POST: vi.fn(async () => ({
      data: {
        payment: {
          id: 'pay-1',
          payableType: 'PAYABLE_TYPE_REGISTRATION',
          payableId: 'r1',
          amount: { amountCents: '1000', currencyCode: 'USD' },
          method: 'PAYMENT_METHOD_OFFLINE',
          status: 'PAYMENT_STATUS_PAID',
          stripeReference: '',
          recordedByUserId: MOCK_HOST_ID,
        },
      },
      error: undefined,
      response: { status: 200 },
    })),
    GET: vi.fn(),
  } as unknown as PaymentsClient
}

// The defect this guards, found by an audit of what a user cannot do:
// HostPayments.vue passed MOCK_HOST_ID to `load`/`markPaid` RAW, while every
// other screen had been migrated to `actingUserId(MOCK_*)`. Since the server
// sets Game.host_id from the verified caller, a signed-in host's games carry
// a uuid — so the dashboard filtered on 'host-mock-1', matched nothing, and
// was permanently empty with a permanently-zero nav badge. Offline cash,
// which is the only payment path here that moves real money, was unreachable.
//
// Every existing test in this file seeded its fixture game with
// MOCK_HOST_ID, so the filter matched in tests and could not in production —
// the same fixture-agrees-with-the-bug shape as #342 and #345.
describe('a signed-in host (the regression #3 from the MVP gap audit)', () => {
  afterEach(() => {
    signOut()
  })

  it('shows a signed-in host their pending cash, filtering on the REAL user id', async () => {
    rememberUserId(REAL_HOST_ID)
    const wrapper = mount(HostPayments, {
      props: {
        client: socialplayClientStubFor(REAL_HOST_ID),
        paymentsClient: paymentsClientStub(),
        win: matchMediaForWidth(1440),
      },
    })
    await flushPromises()

    // Before the fix the filter compared against MOCK_HOST_ID, matched
    // nothing, and this screen rendered its empty state instead.
    expect(wrapper.findAll('.host-payments__row')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('No pending cash payments right now.')
  })

  it('still works for a browse-only session, falling back to the placeholder', async () => {
    // Not signed in: actingUserId returns the fallback, so behaviour is
    // exactly what it was before — that is the contract of the helper.
    const wrapper = mount(HostPayments, {
      props: {
        client: socialplayClientStub(),
        paymentsClient: paymentsClientStub(),
        win: matchMediaForWidth(1440),
      },
    })
    await flushPromises()

    expect(wrapper.findAll('.host-payments__row')).toHaveLength(1)
  })
})

describe('HostPayments', () => {
  it('shows an empty state when there are no pending cash payments', async () => {
    const wrapper = mount(HostPayments, {
      props: { client: emptyClient(), paymentsClient: paymentsClientStub(), win: matchMediaForWidth(1440) },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('No pending cash payments right now.')
  })

  it('lists pending cash registrations for this Host', async () => {
    const client = socialplayClientStub()
    const wrapper = mount(HostPayments, {
      props: { client, paymentsClient: paymentsClientStub(), win: matchMediaForWidth(1440) },
    })
    await flushPromises()

    expect(wrapper.findAll('.host-payments__row')).toHaveLength(1)
    expect(wrapper.text()).toContain('player-1')
    expect(wrapper.text()).toContain('1 guest')
    expect(wrapper.text()).toContain('cash at facility')
  })

  // T8.10 required test: "the Host mark-paid action".
  it('clicking Mark paid calls RecordOfflinePayment and removes the row on success', async () => {
    const client = socialplayClientStub()
    const paymentsClient = paymentsClientStub()
    const wrapper = mount(HostPayments, {
      props: { client, paymentsClient, win: matchMediaForWidth(1440) },
    })
    await flushPromises()
    expect(wrapper.findAll('.host-payments__row')).toHaveLength(1)

    await wrapper.find('.host-payments__mark-paid').trigger('click')
    await flushPromises()

    // T56.1/T58: 2000, not 1000. This registration brings a guest, so it
    // owes two heads at the Game's 1000 entry fee. The old assertion of
    // 1000 was the #126 defect itself — it kept passing after T56.1 only
    // because the fixture carried no amount_owed and the composable fell
    // back to the per-player fee. T58 removed that fallback (the server
    // now validates the figure and would refuse it), so the fixture states
    // what a real server returns and the assertion states what is owed.
    expect(paymentsClient.POST).toHaveBeenCalledWith('/v1/payments:recordOffline', {
      body: {
        payableType: 'PAYABLE_TYPE_REGISTRATION',
        payableId: 'r1',
        amount: { amountCents: '2000', currencyCode: 'USD' },
        actorUserId: MOCK_HOST_ID,
        gameHostId: MOCK_HOST_ID,
      },
    })
    expect(wrapper.findAll('.host-payments__row')).toHaveLength(0)
    expect(wrapper.text()).toContain('No pending cash payments right now.')
  })
})
