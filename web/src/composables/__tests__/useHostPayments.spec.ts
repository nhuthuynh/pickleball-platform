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

// T56.1/T58 — a real server always returns amount_owed on a Registration
// (entry fee x heads, frozen at registration). This fixture fills it in
// when a test does not state one, so every registration here has the shape
// a live server produces rather than a pre-T56.1 one.
//
// That matters more than it looks: T58 makes `load` skip a registration
// recording no owed amount, because the server would refuse any figure we
// sent for it. Without this defaulting, every fixture in this file would
// silently vanish from the dashboard and the tests would be asserting
// against an empty list for reasons unrelated to what they are about — the
// fixture-infidelity failure docs/LESSONS.md's T9 entry describes.
//
// A test that specifically wants a pre-T56.1 row states `amountOwed: null`
// and gets one.
function withOwedAmount(raw: Record<string, unknown>, entryFeeCents: number): Record<string, unknown> {
  if ('amountOwed' in raw) {
    const { amountOwed, ...rest } = raw
    return amountOwed === null ? rest : raw
  }
  const heads = 1 + Number(raw.guestCount ?? 0)
  return { ...raw, amountOwed: { amountCents: String(entryFeeCents * heads), currencyCode: 'USD' } }
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
      const listing = (handlers.games ?? []).find(
        (g) => (g as { game: { id: string } }).game.id === gameId,
      ) as { game: { entryFee?: { amountCents?: string } } } | undefined
      const fee = Number(listing?.game.entryFee?.amountCents ?? 0)
      const regs = (handlers.registrationsByGame?.[gameId] ?? []).map((r) =>
        withOwedAmount(r as Record<string, unknown>, fee),
      )
      return {
        data: { registrations: regs },
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

  // RETIRED by T58 (issue #299), deliberately left as a note rather than
  // deleted silently.
  //
  // T56.1 gave a Registration recording no owed amount a fallback to the
  // Game's per-player entry fee, and this test pinned it. T58 makes the
  // server validate the recorded amount, which refuses that fallback
  // figure (the payable owes 0, not the entry fee) — and refuses 0 too, as
  // an invalid amount. The row became unpayable by any figure, so the
  // fallback stopped being a rescue and became a guaranteed failure.
  //
  // The replacement behaviour — such rows are filtered out of the
  // dashboard, like a free Game's — is pinned by
  // 'excludes a registration that records no owed amount on a paid game'
  // in the T58 block at the end of this file.
})

// T58 (issue #299) — the server now validates the recorded amount, and that
// makes one of this dashboard's own fallbacks unsafe.
//
// Before T58, a Registration carrying no owed amount (migration 0028's
// default of 0, i.e. a row written before T56.1) fell back to the Game's
// per-player entry fee, and RecordOfflinePayment accepted whatever it was
// sent. Now the server compares against what the payable owes — which for
// such a row is 0 — so the fallback figure is REFUSED, and 0 is refused too
// (domain.NewPayment rejects a zero amount). The row is unpayable either
// way.
//
// That is precisely the condition the free-Game filter above already exists
// to prevent: a "Mark paid" button that could only ever fail. Same
// treatment, same reason.
describe('useHostPayments — rows whose owed amount is unknown (T58)', () => {
  it('excludes a registration that records no owed amount on a paid game', async () => {
    const client = fakeClient({
      games: [gameListing({ id: 'g1', hostId: 'host-1', paymentMethod: 'PAYMENT_METHOD_CASH', entryFeeCents: 1000 })],
      registrationsByGame: {
        g1: [
          // Pre-T56.1: no amount_owed on the wire at all.
          { id: 'r-legacy', gameId: 'g1', playerId: 'p1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0, amountOwed: null },
          // Post-T56.1: a real frozen figure.
          { id: 'r-current', gameId: 'g1', playerId: 'p2', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 1, amountOwed: { amountCents: '2000', currencyCode: 'USD' } },
        ],
      },
    })

    const { pending, load } = useHostPayments(client, fakePaymentsClient({}))
    await load('host-1')

    // The payable row survives; the unpayable one does not. Both filtered
    // and kept, so a filter that dropped everything would not pass.
    expect(pending.value.map((p) => p.registrationId)).toEqual(['r-current'])
    expect(pending.value[0]!.amountOwedCents).toBe(2000)
  })
})
