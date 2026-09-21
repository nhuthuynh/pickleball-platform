import { describe, it, expect, vi } from 'vitest'
import { useHostPayments } from '../useHostPayments'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { PaymentsClient } from '../../api/paymentsClient'

function gameListing(overrides: Partial<{
  id: string
  hostId: string
  paymentMethod: string
  entryFeeCents: number
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
      // T9.2: the Game's real entry fee. 1000 ($10.00) is the same figure
      // these tests asserted before, so their substance is unchanged — it
      // is now the Host's real price rather than a global placeholder.
      entryFee: { amountCents: String(overrides.entryFeeCents ?? 1000), currencyCode: 'USD' },
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

// T9.2: a free Game owes nothing, so it has no place on a "cash still owed"
// dashboard. This is correctness, not cosmetics: RecordOfflinePayment
// rejects a zero amount, so a free Game's row would render a "Mark paid"
// button that could only ever fail.
describe('useHostPayments — free games (T9.2)', () => {
  it('excludes a free Game\'s registrations from the pending-cash list', async () => {
    const client = fakeClient({
      games: [
        gameListing({ id: 'g-free', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 0 }),
        gameListing({ id: 'g-paid', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 1500 }),
      ],
      registrationsByGame: {
        'g-free': [{ id: 'r-free', gameId: 'g-free', playerId: 'p1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
        'g-paid': [{ id: 'r-paid', gameId: 'g-paid', playerId: 'p2', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
      },
    })

    const { pending, load } = useHostPayments(client, fakePaymentsClient({}))
    await load('host-1')

    expect(pending.value.map((p) => p.registrationId)).toEqual(['r-paid'])
    expect(pending.value[0]!.entryFeeCents).toBe(1500)
  })

  it('records the Game\'s real fee, not a flat placeholder', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 750 })],
      registrationsByGame: {
        g1: [{ id: 'r1', gameId: 'g1', playerId: 'p1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
      },
    })
    const payments = fakePaymentsClient({
      recordOffline: () => ({
        data: {
          payment: {
            id: 'pay-1',
            payableType: 'PAYABLE_TYPE_REGISTRATION',
            payableId: 'r1',
            amount: { amountCents: '750', currencyCode: 'USD' },
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
    await markPaid(pending.value[0]!, 'host-1')

    const body = (payments.POST as ReturnType<typeof vi.fn>).mock.calls[0]![1].body
    expect(body.amount).toEqual({ amountCents: '750', currencyCode: 'USD' })
  })
})

// T56.1 (issue #126) — the CASH path owes per head too.
//
// The online path was the loud half of #126, but a Game Admin marking cash
// paid records a Payment against the same Registration, and until this
// ticket it recorded the Game's per-PLAYER entry fee: a Host who took
// $30.00 in cash from a player and their two guests recorded $10.00, and
// reconciliation then marked the Registration paid in full. Same defect,
// quieter path.
//
// Note what this does NOT do: T56.2 (#297) validates the amount on
// CreateOnlinePayment only, so nothing server-side refuses a wrong figure
// here. This is the client sending the right number because it is the
// right number, not because it would be caught.
describe('useHostPayments — per-head amounts (T56.1)', () => {
  it('records the amount the Registration owes, not the per-player entry fee', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 1000 })],
      registrationsByGame: {
        g1: [{
          id: 'r1',
          gameId: 'g1',
          playerId: 'player-1',
          status: 'REGISTRATION_STATUS_REGISTERED',
          paymentStatus: 'PAYMENT_STATUS_UNPAID',
          guestCount: 2,
          amountOwed: { amountCents: '3000', currencyCode: 'USD' },
        }],
      },
    })
    const payments = fakePaymentsClient({
      recordOffline: () => ({
        data: { payment: { id: 'pay-1', payableId: 'r1', status: 'PAYMENT_STATUS_PAID' } },
        error: undefined,
        response: { status: 200 },
      }),
    })

    const { pending, load, markPaid } = useHostPayments(client, payments)
    await load('host-1')
    await markPaid(pending.value[0]!, 'host-1')

    const body = (payments.POST as ReturnType<typeof vi.fn>).mock.calls[0]![1].body
    expect(body.amount).toEqual({ amountCents: '3000', currencyCode: 'USD' })
  })

  // The dashboard must show the Host the figure they are about to record,
  // or "Mark paid" becomes a button whose effect differs from its label.
  it('surfaces the owed amount on the row, not the per-player fee', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 1000 })],
      registrationsByGame: {
        g1: [{
          id: 'r1',
          gameId: 'g1',
          playerId: 'player-1',
          status: 'REGISTRATION_STATUS_REGISTERED',
          paymentStatus: 'PAYMENT_STATUS_UNPAID',
          guestCount: 2,
          amountOwed: { amountCents: '3000', currencyCode: 'USD' },
        }],
      },
    })

    const { pending, load } = useHostPayments(client, fakePaymentsClient({}))
    await load('host-1')

    expect(pending.value[0]!.amountOwedCents).toBe(3000)
    expect(pending.value[0]!.amountOwedCurrency).toBe('USD')
  })

  // A Registration written before T56.1 carries amount_owed 0 (the
  // migration's default). Recording 0 would be refused by the Payments
  // domain and the row would be a button that can only fail — the exact
  // condition the free-Game filter above already exists to prevent — so
  // the per-player fee remains the fallback for those rows. It is the
  // figure they were created under, and it is what this dashboard
  // recorded for them before this ticket.
  it('falls back to the entry fee for a pre-T56.1 registration that records no owed amount', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 1000 })],
      registrationsByGame: {
        g1: [{ id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 }],
      },
    })

    const { pending, load } = useHostPayments(client, fakePaymentsClient({}))
    await load('host-1')

    expect(pending.value[0]!.amountOwedCents).toBe(1000)
  })
})
