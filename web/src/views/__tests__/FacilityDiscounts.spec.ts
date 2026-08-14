// T11.3 — Facility Owner discount panel. Mirrors FacilityOnboarding.spec.ts:
// a hand-rolled fake standing in for the real BookingClient, injected via the
// component's `client` prop, so these tests never touch the ambient `fetch`.
import { describe, it, expect, vi, beforeAll } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import FacilityDiscounts from '../FacilityDiscounts.vue'
import type { BookingClient } from '../../api/bookingClient'
import { expectNoA11yViolations, COMPONENT_MOUNT_OPTIONS } from '../../test-support/axe'

const DISCOUNT_PATH = '/v1/facilities/{facilityId}/discount-rules'

beforeAll(() => {
  // jsdom doesn't implement matchMedia; useBreakpoint() needs it. Same stub
  // as FacilityOnboarding.spec.ts.
  if (!window.matchMedia) {
    window.matchMedia = (query: string) =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList
  }
})

function fakeClient(handlers: { list?: () => unknown; create?: (body: unknown) => unknown } = {}): BookingClient {
  const GET = vi.fn(async (path: string) => {
    if (path === DISCOUNT_PATH) return handlers.list?.() ?? { data: { discountRules: [] }, error: undefined }
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === DISCOUNT_PATH) return handlers.create?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { GET, POST } as unknown as BookingClient
}

function rule(overrides: Record<string, unknown> = {}) {
  return {
    id: 'discount-1',
    facilityId: 'facility-1',
    discountType: 'DISCOUNT_TYPE_PERCENT',
    percent: 15,
    appliesTo: ['SOURCE_INDIVIDUAL'],
    startsAt: '2026-09-01T00:00:00Z',
    endCondition: { kind: 'END_CONDITION_KIND_NO_END' },
    ...overrides,
  }
}

async function mountPanel(client: BookingClient, attach = false) {
  const wrapper = mount(FacilityDiscounts, {
    props: { facilityId: 'facility-1', client },
    ...(attach ? { attachTo: document.body } : {}),
  })
  await flushPromises()
  return wrapper
}

async function showList(wrapper: VueWrapper) {
  const listTab = wrapper.findAll('button').find((b) => b.text() === 'All discounts')
  await listTab!.trigger('click')
  await flushPromises()
}

async function fillValidPercentDiscount(wrapper: VueWrapper) {
  await wrapper.find('#discount-percent').setValue('15')
  await wrapper.find('input[type="checkbox"][value="SOURCE_INDIVIDUAL"]').setValue(true)
  await wrapper.find('#discount-starts-at').setValue('2026-09-01')
}

describe('FacilityDiscounts', () => {
  it('lists the facility’s existing discounts on mount', async () => {
    const client = fakeClient({ list: () => ({ data: { discountRules: [rule()] }, error: undefined }) })
    const wrapper = await mountPanel(client)

    expect(client.GET).toHaveBeenCalledWith(DISCOUNT_PATH, {
      params: { path: { facilityId: 'facility-1' } },
    })

    await showList(wrapper)
    expect(wrapper.text()).toContain('15% off')
    expect(wrapper.text()).toContain('Individual bookings')
  })

  it('shows an empty state rather than an error when a facility has no discounts', async () => {
    const wrapper = await mountPanel(fakeClient())
    await showList(wrapper)
    expect(wrapper.text()).toContain('No discounts yet')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('creates a discount and confirms it through the ARIA live region', async () => {
    const client = fakeClient({ create: () => ({ data: { discountRule: rule() }, error: undefined, response: { status: 200 } }) })
    const wrapper = await mountPanel(client)

    await fillValidPercentDiscount(wrapper)
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(client.POST).toHaveBeenCalledWith(DISCOUNT_PATH, {
      params: { path: { facilityId: 'facility-1' } },
      body: expect.objectContaining({
        discountType: 'DISCOUNT_TYPE_PERCENT',
        percent: 15,
        appliesTo: ['SOURCE_INDIVIDUAL'],
        endCondition: { kind: 'END_CONDITION_KIND_NO_END' },
      }),
    })
    expect(wrapper.find('[role="status"]').text()).toContain('Discount created')
    // The new rule shows up without a manual refresh.
    expect(wrapper.text()).toContain('15% off')
  })

  // ── WCAG 3.3.1 / 3.3.3 ────────────────────────────────────────────────
  it('identifies an invalid field in TEXT next to that field, with a suggested fix', async () => {
    const client = fakeClient({ create: () => ({ data: { discountRule: rule() }, error: undefined }) })
    const wrapper = await mountPanel(client)

    // Everything valid except the percentage.
    await wrapper.find('#discount-percent').setValue('150')
    await wrapper.find('input[type="checkbox"][value="SOURCE_INDIVIDUAL"]').setValue(true)
    await wrapper.find('#discount-starts-at').setValue('2026-09-01')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    // Nothing was sent (the gate is real, in code — see useFacilityDiscounts).
    expect(client.POST).not.toHaveBeenCalled()

    const errorEl = wrapper.find('#discount-percent-error')
    expect(errorEl.exists()).toBe(true)
    // 3.3.1: identified in text, with role="alert" so it is announced.
    expect(errorEl.attributes('role')).toBe('alert')
    // 3.3.3: the message tells the Owner what a correct value looks like.
    expect(errorEl.text()).toContain('for example, 15 for 15% off')
    // 1.3.1/4.1.2: the input is programmatically tied to its message.
    const input = wrapper.find('#discount-percent')
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(input.attributes('aria-describedby')).toBe('discount-percent-error')
  })

  it('reports a missing "applies to" selection on the fieldset, not as a vague form error', async () => {
    const wrapper = await mountPanel(fakeClient({ create: () => ({ data: { discountRule: rule() } }) }))

    await wrapper.find('#discount-percent').setValue('15')
    await wrapper.find('#discount-starts-at').setValue('2026-09-01')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const errorEl = wrapper.find('#discount-applies-to-error')
    expect(errorEl.exists()).toBe(true)
    expect(errorEl.text()).toContain('Select at least one booking type')
  })

  it('surfaces a 403 as a specific ownership message', async () => {
    const client = fakeClient({
      create: () => ({ data: undefined, error: { message: 'forbidden' }, response: { status: 403 } }),
    })
    const wrapper = await mountPanel(client)

    await fillValidPercentDiscount(wrapper)
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Only the owner of this facility can create a discount')
  })

  // ── T11.3 instruction #3: no fabricated fields ────────────────────────
  it('renders a NoEnd rule as "No end date" and never a date', async () => {
    const client = fakeClient({
      list: () => ({
        data: { discountRules: [rule({ endCondition: { kind: 'END_CONDITION_KIND_NO_END' } })] },
        error: undefined,
      }),
    })
    const wrapper = await mountPanel(client)
    await showList(wrapper)

    expect(wrapper.text()).toContain('No end date')
    // No fabricated date anywhere in the rendered rule.
    expect(wrapper.text()).not.toMatch(/\d{4}/)
  })

  it('renders an EndAfterOccurrences rule as a TOTAL, never as a remaining count', async () => {
    const client = fakeClient({
      list: () => ({
        data: {
          discountRules: [
            rule({ endCondition: { kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 10 } }),
          ],
        },
        error: undefined,
      }),
    })
    const wrapper = await mountPanel(client)
    await showList(wrapper)

    expect(wrapper.text()).toContain('Ends after 10 total uses')
    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('remaining')
    expect(text).not.toContain('uses left')
  })

  it('labels the occurrences INPUT as a total, so the Owner is not promised a live counter', async () => {
    const wrapper = await mountPanel(fakeClient())
    await wrapper.find('input[value="END_CONDITION_KIND_END_AFTER_OCCURRENCES"]').setValue(true)
    await flushPromises()

    expect(wrapper.find('#discount-occurrences').exists()).toBe(true)
    expect(wrapper.text()).toContain('Total uses allowed')
    expect(wrapper.text().toLowerCase()).not.toContain('remaining')
  })

  // ── WCAG automated sweep ──────────────────────────────────────────────
  it('has no automated a11y violations in its create state', async () => {
    const wrapper = await mountPanel(fakeClient(), true)
    try {
      await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    } finally {
      wrapper.unmount()
    }
  })

  it('has no automated a11y violations while showing validation errors', async () => {
    const wrapper = await mountPanel(fakeClient({ create: () => ({ data: undefined }) }), true)
    try {
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()
      expect(wrapper.findAll('[role="alert"]').length).toBeGreaterThan(0)
      await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    } finally {
      wrapper.unmount()
    }
  })

  it('has no automated a11y violations in its list state', async () => {
    const client = fakeClient({ list: () => ({ data: { discountRules: [rule()] }, error: undefined }) })
    const wrapper = await mountPanel(client, true)
    try {
      await showList(wrapper)
      await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    } finally {
      wrapper.unmount()
    }
  })
})
