// T11.6 — the Facility Owner's queue composable. The tests that matter here
// are about `approvalResults`: it is the ONLY record of which weeks an
// approval actually booked (T11.5 returns it once, in the approval response,
// and no read endpoint replays it), so it must be written from a real response
// and never synthesised.
import { describe, it, expect, vi } from 'vitest'
import { useFacilityRentalRequests } from '../useFacilityRentalRequests'
import type { BookingClient } from '../../api/bookingClient'

const LIST_PATH = '/v1/facilities/{facilityId}/recurring-hire-templates'
const APPROVE_PATH = '/v1/recurring-hire-templates/{templateId}:approve'
const REJECT_PATH = '/v1/recurring-hire-templates/{templateId}:reject'
const OWNER = 'owner-mock-1'

function template(overrides: Record<string, unknown> = {}) {
  return {
    id: 'template-1',
    requestedByUserId: 'club-1',
    courtId: 'court-1',
    weekday: 1,
    startMinute: 540,
    endMinute: 600,
    startsAt: '2026-09-07T00:00:00Z',
    endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' },
    status: 'RECURRING_HIRE_STATUS_REQUESTED',
    ...overrides,
  }
}

function fakeClient(
  handlers: { list?: () => unknown; approve?: () => unknown; reject?: () => unknown } = {},
): BookingClient {
  const GET = vi.fn(
    async () =>
      handlers.list?.() ?? { data: { templates: [template()] }, error: undefined, response: { status: 200 } },
  )
  const POST = vi.fn(async (path: string) => {
    if (path === APPROVE_PATH) {
      return (
        handlers.approve?.() ?? {
          data: { template: template({ status: 'RECURRING_HIRE_STATUS_APPROVED' }), occurrences: [] },
          error: undefined,
          response: { status: 200 },
        }
      )
    }
    if (path === REJECT_PATH) {
      return (
        handlers.reject?.() ?? {
          data: { template: template({ status: 'RECURRING_HIRE_STATUS_REJECTED' }) },
          error: undefined,
          response: { status: 200 },
        }
      )
    }
    throw new Error(`unexpected POST ${path}`)
  })
  return { GET, POST } as unknown as BookingClient
}

describe('useFacilityRentalRequests.load', () => {
  it('reads the owner-scoped queue', async () => {
    const client = fakeClient()
    const queue = useFacilityRentalRequests(client)

    await queue.load('facility-1', OWNER)

    expect(client.GET).toHaveBeenCalledWith(LIST_PATH, {
      params: { path: { facilityId: 'facility-1' }, query: { actorUserId: OWNER } },
    })
    expect(queue.templates.value).toHaveLength(1)
  })

  it('distinguishes "not the owner" from a generic failure', async () => {
    const queue = useFacilityRentalRequests(
      fakeClient({ list: () => ({ data: undefined, error: { message: 'no' }, response: { status: 403 } }) }),
    )

    await queue.load('facility-1', OWNER)

    expect(queue.error.value).toMatch(/only the owner/i)
  })
})

describe('useFacilityRentalRequests.approve', () => {
  it('records the per-week outcomes exactly as the server reported them', async () => {
    const occurrences = [
      {
        startsAt: '2026-09-07T09:00:00Z',
        endsAt: '2026-09-07T10:00:00Z',
        outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED',
        bookingId: 'booking-1',
      },
      {
        startsAt: '2026-09-14T09:00:00Z',
        endsAt: '2026-09-14T10:00:00Z',
        outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT',
        reason: 'court double booked',
      },
    ]
    const queue = useFacilityRentalRequests(
      fakeClient({
        approve: () => ({
          data: { template: template({ status: 'RECURRING_HIRE_STATUS_APPROVED' }), occurrences },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await queue.load('facility-1', OWNER)

    const ok = await queue.approve('template-1', OWNER)

    expect(ok).toBe(true)
    const recorded = queue.approvalResults.value['template-1'] ?? []
    expect(recorded).toHaveLength(2)
    expect(recorded[1]?.reason).toBe('court double booked')
    // A skipped week has no booking id, and none is invented for it.
    expect(recorded[1]?.bookingId).toBe('')
    expect(queue.statusMessage.value).toContain('1 of 2 weeks booked')
  })

  // A partial approval still transitions the template — the ticket's whole
  // point. The status must follow the server, not the occurrence outcomes.
  it('marks the template approved even when weeks were skipped', async () => {
    const queue = useFacilityRentalRequests(
      fakeClient({
        approve: () => ({
          data: {
            template: template({ status: 'RECURRING_HIRE_STATUS_APPROVED' }),
            occurrences: [
              {
                startsAt: '2026-09-07T09:00:00Z',
                outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT',
                reason: 'taken',
              },
            ],
          },
          error: undefined,
          response: { status: 200 },
        }),
      }),
    )
    await queue.load('facility-1', OWNER)

    await queue.approve('template-1', OWNER)

    expect(queue.templates.value[0]?.status).toBe('RECURRING_HIRE_STATUS_APPROVED')
    expect(queue.statusMessage.value).toContain('0 of 1 weeks booked')
  })

  it('records nothing when the approval failed', async () => {
    const queue = useFacilityRentalRequests(
      fakeClient({ approve: () => ({ data: undefined, error: { message: 'no' }, response: { status: 403 } }) }),
    )
    await queue.load('facility-1', OWNER)

    const ok = await queue.approve('template-1', OWNER)

    expect(ok).toBe(false)
    expect(queue.approvalResults.value['template-1']).toBeUndefined()
    expect(queue.decisionError.value).toMatch(/only the owner/i)
    expect(queue.templates.value[0]?.status).toBe('RECURRING_HIRE_STATUS_REQUESTED')
  })

  it('never seeds an approval record from a plain list load', async () => {
    const queue = useFacilityRentalRequests(
      fakeClient({
        list: () => ({
          data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_APPROVED' })] },
          error: undefined,
        }),
      }),
    )

    await queue.load('facility-1', OWNER)

    expect(queue.approvalResults.value).toEqual({})
  })
})

describe('useFacilityRentalRequests.reject', () => {
  it('updates the status and states that the decision is closed', async () => {
    const queue = useFacilityRentalRequests(fakeClient())
    await queue.load('facility-1', OWNER)

    const ok = await queue.reject('template-1', OWNER)

    expect(ok).toBe(true)
    expect(queue.templates.value[0]?.status).toBe('RECURRING_HIRE_STATUS_REJECTED')
    expect(queue.statusMessage.value).toMatch(/no courts were booked/i)
    expect(queue.statusMessage.value).toMatch(/new one/i)
    expect(queue.approvalResults.value['template-1']).toBeUndefined()
  })

  it('explains an already-answered request specifically', async () => {
    const queue = useFacilityRentalRequests(
      fakeClient({ reject: () => ({ data: undefined, error: { message: 'no' }, response: { status: 400 } }) }),
    )
    await queue.load('facility-1', OWNER)

    await queue.reject('template-1', OWNER)

    expect(queue.decisionError.value).toMatch(/already been answered/i)
  })
})
