// T11.6 — the Club's request + status-list composable, tested directly rather
// than only through ClubRentals.vue, so the validation gate is provably a code
// path (a programmatic call cannot bypass it) and not just a disabled button.
import { describe, it, expect, vi } from 'vitest'
import { useClubRentals } from '../useClubRentals'
import { emptyRecurringHireForm, type RecurringHireFormInput } from '../../models/recurringHireForm'
import type { BookingClient } from '../../api/bookingClient'

const TEMPLATES_PATH = '/v1/recurring-hire-templates'
const ACTOR = '00000000-0000-4000-b000-000000000010'
const COURT = '00000000-0000-4000-a000-000000000001'

function validForm(): RecurringHireFormInput {
  return {
    ...emptyRecurringHireForm(),
    courtId: COURT,
    weekday: 1,
    startTime: '09:00',
    endTime: '10:00',
    startsAt: '2026-09-07',
  }
}

function template(overrides: Record<string, unknown> = {}) {
  return {
    id: 'template-1',
    requestedByUserId: ACTOR,
    courtId: COURT,
    weekday: 1,
    startMinute: 540,
    endMinute: 600,
    startsAt: '2026-09-07T00:00:00Z',
    endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' },
    status: 'RECURRING_HIRE_STATUS_REQUESTED',
    ...overrides,
  }
}

function fakeClient(handlers: { list?: () => unknown; request?: () => unknown } = {}): BookingClient {
  const GET = vi.fn(async () => handlers.list?.() ?? { data: { templates: [] }, error: undefined })
  const POST = vi.fn(
    async () =>
      handlers.request?.() ?? { data: { template: template() }, error: undefined, response: { status: 200 } },
  )
  return { GET, POST } as unknown as BookingClient
}

describe('useClubRentals.load', () => {
  it('asks for the actor’s own templates and maps them', async () => {
    const client = fakeClient({ list: () => ({ data: { templates: [template()] }, error: undefined }) })
    const rentals = useClubRentals(client)

    await rentals.load(ACTOR)

    expect(client.GET).toHaveBeenCalledWith(TEMPLATES_PATH, {
      params: { query: { actorUserId: ACTOR } },
    })
    expect(rentals.templates.value).toHaveLength(1)
    expect(rentals.error.value).toBeNull()
  })

  it('treats an absent templates array as an empty history, not a failure', async () => {
    const rentals = useClubRentals(fakeClient({ list: () => ({ data: {}, error: undefined }) }))

    await rentals.load(ACTOR)

    expect(rentals.templates.value).toEqual([])
    expect(rentals.error.value).toBeNull()
  })

  it('reports a failed load as an error rather than an empty history', async () => {
    const rentals = useClubRentals(
      fakeClient({ list: () => ({ data: undefined, error: { message: 'boom' } }) }),
    )

    await rentals.load(ACTOR)

    expect(rentals.templates.value).toEqual([])
    expect(rentals.error.value).toMatch(/could not load/i)
  })

  // The status view must show decided requests too — filtering them out is
  // what would let a rejection quietly disappear (instruction #4).
  it('keeps rejected and approved templates in the list', async () => {
    const rentals = useClubRentals(
      fakeClient({
        list: () => ({
          data: {
            templates: [
              template({ id: 'a', status: 'RECURRING_HIRE_STATUS_REQUESTED' }),
              template({ id: 'b', status: 'RECURRING_HIRE_STATUS_APPROVED' }),
              template({ id: 'c', status: 'RECURRING_HIRE_STATUS_REJECTED' }),
            ],
          },
          error: undefined,
        }),
      }),
    )

    await rentals.load(ACTOR)

    expect(rentals.templates.value.map((t) => t.status)).toEqual([
      'RECURRING_HIRE_STATUS_REQUESTED',
      'RECURRING_HIRE_STATUS_APPROVED',
      'RECURRING_HIRE_STATUS_REJECTED',
    ])
  })
})

describe('useClubRentals.request', () => {
  it('sends nothing when the form is invalid, and reports the field errors', async () => {
    const client = fakeClient()
    const rentals = useClubRentals(client)

    const sent = await rentals.request({ ...validForm(), startTime: '' }, ACTOR)

    expect(sent).toBe(false)
    expect(client.POST).not.toHaveBeenCalled()
    expect(rentals.fieldErrors.value.startTime).toBeDefined()
  })

  it('adds the created template to the list and confirms honestly', async () => {
    const rentals = useClubRentals(fakeClient())

    const sent = await rentals.request(validForm(), ACTOR)

    expect(sent).toBe(true)
    expect(rentals.templates.value).toHaveLength(1)
    // "Request sent", never "court booked" — approval is what books anything.
    expect(rentals.statusMessage.value).toMatch(/request sent/i)
    expect(rentals.statusMessage.value).not.toMatch(/court booked|you have booked/i)
  })

  it.each([
    [403, /not registered as a club/i],
    [404, /court could not be found/i],
  ])('gives a specific message for a %i', async (status, pattern) => {
    const rentals = useClubRentals(
      fakeClient({
        request: () => ({ data: undefined, error: { message: 'nope' }, response: { status } }),
      }),
    )

    const sent = await rentals.request(validForm(), ACTOR)

    expect(sent).toBe(false)
    expect(rentals.formError.value).toMatch(pattern)
    expect(rentals.templates.value).toHaveLength(0)
  })

  it('does not add anything to the list when the server refuses', async () => {
    const rentals = useClubRentals(
      fakeClient({ request: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }) }),
    )

    await rentals.request(validForm(), ACTOR)

    expect(rentals.templates.value).toEqual([])
    expect(rentals.formError.value).toMatch(/could not send/i)
  })
})
