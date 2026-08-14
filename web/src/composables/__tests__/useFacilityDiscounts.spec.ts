// T11.3 — network/state half of the Facility Owner's discount panel.
// Same "fake the injected client, never touch fetch" shape as
// useCourtBooking.spec.ts / FacilityOnboarding.spec.ts.
import { describe, it, expect, vi } from 'vitest'
import { useFacilityDiscounts } from '../useFacilityDiscounts'
import { emptyDiscountForm, type DiscountFormInput } from '../../models/discountForm'
import type { BookingClient } from '../../api/bookingClient'

const DISCOUNT_PATH = '/v1/facilities/{facilityId}/discount-rules'

function fakeClient(handlers: {
  list?: () => unknown
  create?: (body: unknown) => unknown
}): BookingClient {
  const GET = vi.fn(async (path: string) => {
    if (path === DISCOUNT_PATH) return handlers.list?.()
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === DISCOUNT_PATH) return handlers.create?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { GET, POST } as unknown as BookingClient
}

function validForm(overrides: Partial<DiscountFormInput> = {}): DiscountFormInput {
  return {
    ...emptyDiscountForm(),
    discountType: 'DISCOUNT_TYPE_PERCENT',
    percent: '15',
    appliesTo: ['SOURCE_INDIVIDUAL'],
    startsAt: '2026-09-01',
    endKind: 'END_CONDITION_KIND_NO_END',
    ...overrides,
  }
}

const RULE = {
  id: 'discount-1',
  facilityId: 'facility-1',
  discountType: 'DISCOUNT_TYPE_PERCENT',
  percent: 15,
  appliesTo: ['SOURCE_INDIVIDUAL'],
  startsAt: '2026-09-01T00:00:00Z',
  endCondition: { kind: 'END_CONDITION_KIND_NO_END' },
}

describe('useFacilityDiscounts', () => {
  it('load maps ListDiscountRulesForFacility into view models', async () => {
    const client = fakeClient({ list: () => ({ data: { discountRules: [RULE] }, error: undefined }) })
    const { discounts, loading, error, load } = useFacilityDiscounts(client)

    await load('facility-1')

    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
    expect(discounts.value).toHaveLength(1)
    expect(discounts.value[0]!.id).toBe('discount-1')
    expect(client.GET).toHaveBeenCalledWith(DISCOUNT_PATH, {
      params: { path: { facilityId: 'facility-1' } },
    })
  })

  it('load treats an absent discountRules array as an empty list, not a failure', async () => {
    const client = fakeClient({ list: () => ({ data: {}, error: undefined }) })
    const { discounts, error, load } = useFacilityDiscounts(client)

    await load('facility-1')

    expect(discounts.value).toEqual([])
    expect(error.value).toBeNull()
  })

  it('load surfaces a readable error when the request fails', async () => {
    const client = fakeClient({ list: () => ({ data: undefined, error: { message: 'boom' } }) })
    const { error, load } = useFacilityDiscounts(client)

    await load('facility-1')

    expect(error.value).toBe('Could not load discounts for this facility. Please try again.')
  })

  it('load surfaces a connection error when the request throws', async () => {
    const client = { GET: vi.fn(async () => { throw new Error('offline') }), POST: vi.fn() } as unknown as BookingClient
    const { error, load } = useFacilityDiscounts(client)

    await load('facility-1')

    expect(error.value).toBe('Could not reach the server. Check your connection and try again.')
  })

  it('create refuses to call the API at all when the form is invalid (validation is a real code gate)', async () => {
    // Same discipline as FacilityOnboarding's consent gate: the guard lives
    // in code, not only in a disabled button, so a programmatic call can't
    // bypass it either.
    const client = fakeClient({ create: () => ({ data: { discountRule: RULE }, error: undefined }) })
    const { create, fieldErrors } = useFacilityDiscounts(client)

    const ok = await create('facility-1', validForm({ percent: '0' }), 'owner-1')

    expect(ok).toBe(false)
    expect(client.POST).not.toHaveBeenCalled()
    expect(fieldErrors.value.percent).toContain('Enter a percentage greater than 0')
  })

  it('create posts a valid rule, adds it to the list, and confirms in a status message', async () => {
    const client = fakeClient({ create: () => ({ data: { discountRule: RULE }, error: undefined, response: { status: 200 } }) })
    const { create, discounts, statusMessage, fieldErrors } = useFacilityDiscounts(client)

    const ok = await create('facility-1', validForm(), 'owner-1')

    expect(ok).toBe(true)
    expect(fieldErrors.value).toEqual({})
    expect(client.POST).toHaveBeenCalledWith(DISCOUNT_PATH, {
      params: { path: { facilityId: 'facility-1' } },
      body: expect.objectContaining({ actorUserId: 'owner-1', percent: 15 }),
    })
    // The created rule appears without needing a manual refresh.
    expect(discounts.value).toHaveLength(1)
    expect(statusMessage.value).toContain('Discount created')
  })

  it('create surfaces a 403 as a specific ownership message, not a generic failure', async () => {
    const client = fakeClient({
      create: () => ({ data: undefined, error: { message: 'not the owner' }, response: { status: 403 } }),
    })
    const { create, formError, discounts } = useFacilityDiscounts(client)

    const ok = await create('facility-1', validForm(), 'someone-else')

    expect(ok).toBe(false)
    expect(formError.value).toBe(
      'Only the owner of this facility can create a discount for it. Check you are signed in as the owner.',
    )
    expect(discounts.value).toEqual([])
  })

  it('create surfaces any other server error inline rather than silently doing nothing', async () => {
    const client = fakeClient({
      create: () => ({ data: undefined, error: { message: 'bad request' }, response: { status: 400 } }),
    })
    const { create, formError } = useFacilityDiscounts(client)

    expect(await create('facility-1', validForm(), 'owner-1')).toBe(false)
    expect(formError.value).toContain('Could not create this discount')
  })
})
