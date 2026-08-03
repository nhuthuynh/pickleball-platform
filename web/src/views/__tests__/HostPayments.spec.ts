import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import HostPayments from '../HostPayments.vue'
import { MOCK_HOST_ID } from '../../composables/useHostPayments'
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
    },
    spotsLeft: 5,
  }
}

function socialplayClientStub(): SocialPlayClient {
  return {
    GET: vi.fn(async (path: string, options: { params?: { path?: { gameId?: string } } }) => {
      if (path === '/v1/games') {
        return {
          data: { games: [gameListing('g1', MOCK_HOST_ID, 'PAYMENT_METHOD_CASH')] },
          error: undefined,
          response: { status: 200 },
        }
      }
      if (path === '/v1/games/{gameId}/registrations' && options.params?.path?.gameId === 'g1') {
        return {
          data: {
            registrations: [
              { id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 1 },
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

    expect(paymentsClient.POST).toHaveBeenCalledWith('/v1/payments:recordOffline', {
      body: {
        payableType: 'PAYABLE_TYPE_REGISTRATION',
        payableId: 'r1',
        amount: { amountCents: '1000', currencyCode: 'USD' },
        actorUserId: MOCK_HOST_ID,
        gameHostId: MOCK_HOST_ID,
      },
    })
    expect(wrapper.findAll('.host-payments__row')).toHaveLength(0)
    expect(wrapper.text()).toContain('No pending cash payments right now.')
  })
})
