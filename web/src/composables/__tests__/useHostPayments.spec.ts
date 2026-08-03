import { describe, it, expect, vi } from 'vitest'
import { useHostPayments } from '../useHostPayments'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { PaymentsClient } from '../../api/paymentsClient'

function gameListing(overrides: Partial<{
  id: string
  hostId: string
  paymentMethod: string
}> = {}) {
  return {
    game: {
      id: overrides.id ?? 'g1',
      hostId: overrides.hostId ?? 'host-1',
      venueFacilityId: 'facility-1',
      courtIds: ['court-1'],
      startsAt: '2026-09-01T10:00:00Z',
      endsAt: '2026-09-01T11:00:00Z',
      capacity: 8,
      status: 'GAME_STATUS_SCHEDULED',
      paymentMethod: overrides.paymentMethod ?? 'PAYMENT_METHOD_CASH',
      guestAllowance: 2,
    },
    spotsLeft: 5,
  }
}

function fakeClient(handlers: {
  games?: unknown[]
  registrationsByGame?: Record<string, unknown[]>
}): SocialPlayClient {
  const GET = vi.fn(async (path: string, options: { params?: { path?: { gameId?: string } } }) => {
    if (path === '/v1/games') {
      return { data: { games: handlers.games ?? [] }, error: undefined, response: { status: 200 } }
    }
    if (path === '/v1/games/{gameId}/registrations') {
      const gameId = options.params?.path?.gameId ?? ''
      return {
        data: { registrations: handlers.registrationsByGame?.[gameId] ?? [] },
        error: undefined,
        response: { status: 200 },
      }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  return { GET, POST: vi.fn() } as unknown as SocialPlayClient
}

function fakePaymentsClient(handlers: { recordOffline?: (body: unknown) => unknown }): PaymentsClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/payments:recordOffline') return handlers.recordOffline?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as PaymentsClient
}

describe('useHostPayments', () => {
  it('lists only this Host\'s cash-eligible Games\' unpaid Registrations', async () => {
    const client = fakeClient({
      games: [
        gameListing({ id: 'g-mine-cash', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH' }),
        gameListing({ id: 'g-mine-online-only', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_ONLINE' }),
        gameListing({ id: 'g-other-host', hostId: 'host-2', paymentMethod: 'PAYMENT_METHOD_EITHER' }),
      ],
      registrationsByGame: {
        'g-mine-cash': [
          { id: 'r-unpaid', gameId: 'g-mine-cash', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 1 },
          { id: 'r-paid', gameId: 'g-mine-cash', playerId: 'player-2', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_PAID', guestCount: 0 },
        ],
        'g-mine-online-only': [
          { id: 'r-online-unpaid', gameId: 'g-mine-online-only', playerId: 'player-3', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 },
        ],
        'g-other-host': [
          { id: 'r-not-mine', gameId: 'g-other-host', playerId: 'player-4', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 },
        ],
      },
    })

    const { pending, load } = useHostPayments(client, fakePaymentsClient({}))
    await load('host-1')

    // g-mine-online-only is excluded: an Online-only Game has no cash option
    // at all, so an unpaid Registration there isn't a *cash* payment
    // pending. g-other-host is excluded: not this Host's Game.
    expect(pending.value).toHaveLength(1)
    expect(pending.value[0]!.registrationId).toBe('r-unpaid')
    expect(pending.value[0]!.gameId).toBe('g-mine-cash')
    expect(pending.value[0]!.guestCount).toBe(1)
  })

  // T8.10 required test: "the Host mark-paid action".
  it('markPaid calls RecordOfflinePayment and removes the entry from pending on success', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH' })],
      registrationsByGame: {
        g1: [{ id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
      },
    })
    const payments = fakePaymentsClient({
      recordOffline: () => ({
        data: {
          payment: {
            id: 'pay-1',
            payableType: 'PAYABLE_TYPE_REGISTRATION',
            payableId: 'r1',
            amount: { amountCents: '1000', currencyCode: 'USD' },
            method: 'PAYMENT_METHOD_OFFLINE',
            status: 'PAYMENT_STATUS_PAID',
            stripeReference: '',
            recordedByUserId: 'host-1',
          },
        },
        error: undefined,
        response: { status: 200 },
      }),
    })

    const { pending, load, markPaid } = useHostPayments(client, payments)
    await load('host-1')
    expect(pending.value).toHaveLength(1)

    await markPaid(pending.value[0]!, 'host-1')

    expect(payments.POST).toHaveBeenCalledWith('/v1/payments:recordOffline', {
      body: {
        payableType: 'PAYABLE_TYPE_REGISTRATION',
        payableId: 'r1',
        amount: { amountCents: '1000', currencyCode: 'USD' },
        actorUserId: 'host-1',
        gameHostId: 'host-1',
      },
    })
    expect(pending.value).toHaveLength(0)
  })

  it('markPaid sets a human-readable error and keeps the entry pending when RecordOfflinePayment fails', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH' })],
      registrationsByGame: {
        g1: [{ id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
      },
    })
    const payments = fakePaymentsClient({
      recordOffline: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 403 } }),
    })

    const { pending, markPaidError, load, markPaid } = useHostPayments(client, payments)
    await load('host-1')
    await markPaid(pending.value[0]!, 'host-1')

    expect(markPaidError.value).toBeTruthy()
    expect(pending.value).toHaveLength(1)
  })
})
