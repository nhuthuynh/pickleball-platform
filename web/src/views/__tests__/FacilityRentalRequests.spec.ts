// T11.6 — the Facility Owner's incoming rental-requests panel. Mirrors
// FacilityDiscounts.spec.ts: a hand-rolled fake standing in for the real
// BookingClient, injected via the component's `client` prop, so these tests
// never touch the ambient `fetch`.
//
// The load-bearing tests here are the PARTIAL-APPROVAL ones (ticket
// instruction #2). T11.5's approval books each week independently and the
// template becomes `approved` regardless of how many weeks were skipped, so a
// screen that renders "Approved" and stops would be telling an owner they
// booked twelve weeks when they booked ten. Both the counted summary and the
// per-week list are asserted, plus the negative case: a template approved
// before this screen loaded has NO per-week record anywhere, and the screen
// must say so rather than reconstruct one.
import { describe, it, expect, vi, beforeAll } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FacilityRentalRequests from '../FacilityRentalRequests.vue'
import type { BookingClient } from '../../api/bookingClient'
import { findControlsMatching } from '../../test-support/semanticControlAssertions'
import { expectNoA11yViolations, COMPONENT_MOUNT_OPTIONS } from '../../test-support/axe'

const LIST_PATH = '/v1/facilities/{facilityId}/recurring-hire-templates'
const APPROVE_PATH = '/v1/recurring-hire-templates/{templateId}:approve'
const REJECT_PATH = '/v1/recurring-hire-templates/{templateId}:reject'
const OWNER = 'owner-mock-1'

beforeAll(() => {
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

function template(overrides: Record<string, unknown> = {}) {
  return {
    id: 'template-1',
    requestedByUserId: 'club-1',
    courtId: 'court-1',
    weekday: 1,
    startMinute: 540,
    endMinute: 600,
    startsAt: '2026-09-07T00:00:00Z',
    endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 3 },
    status: 'RECURRING_HIRE_STATUS_REQUESTED',
    ...overrides,
  }
}

function occurrence(overrides: Record<string, unknown> = {}) {
  return {
    startsAt: '2026-09-07T09:00:00Z',
    endsAt: '2026-09-07T10:00:00Z',
    outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED',
    bookingId: 'booking-1',
    ...overrides,
  }
}

function fakeClient(
  handlers: {
    list?: () => unknown
    approve?: (body: unknown) => unknown
    reject?: (body: unknown) => unknown
  } = {},
): BookingClient {
  const GET = vi.fn(async (path: string) => {
    if (path === LIST_PATH) {
      return handlers.list?.() ?? { data: { templates: [template()] }, error: undefined, response: { status: 200 } }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === APPROVE_PATH) {
      return (
        handlers.approve?.(options.body) ?? {
          data: {
            template: template({ status: 'RECURRING_HIRE_STATUS_APPROVED' }),
            occurrences: [occurrence()],
          },
          error: undefined,
          response: { status: 200 },
        }
      )
    }
    if (path === REJECT_PATH) {
      return (
        handlers.reject?.(options.body) ?? {
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

async function mountPanel(client: BookingClient, attach = false) {
  const wrapper = mount(FacilityRentalRequests, {
    props: { facilityId: 'facility-1', client },
    ...(attach ? { attachTo: document.body } : {}),
  })
  await flushPromises()
  return wrapper
}

describe('FacilityRentalRequests — the queue', () => {
  it('loads the facility’s incoming requests as the owner', async () => {
    const client = fakeClient()
    const wrapper = await mountPanel(client)

    expect(client.GET).toHaveBeenCalledWith(LIST_PATH, {
      params: { path: { facilityId: 'facility-1' }, query: { actorUserId: OWNER } },
    })
    expect(wrapper.text()).toContain('Mondays')
    expect(wrapper.text()).toContain('09:00–10:00')
  })

  it('shows an empty state rather than an error when nothing has been requested', async () => {
    const wrapper = await mountPanel(fakeClient({ list: () => ({ data: { templates: [] }, error: undefined }) }))

    expect(wrapper.text()).toMatch(/no club has requested/i)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  // This read is owner-only (unlike ListDiscountRulesForFacility), so a 403
  // means something specific the owner can act on.
  it('explains a 403 as "you are not the owner", not as a generic failure', async () => {
    const wrapper = await mountPanel(
      fakeClient({ list: () => ({ data: undefined, error: { message: 'denied' }, response: { status: 403 } }) }),
    )

    expect(wrapper.find('[role="alert"]').text()).toMatch(/only the owner of this facility/i)
  })
})

describe('FacilityRentalRequests — approving shows every week', () => {
  it('reports the per-week outcome, not a bare "approved"', async () => {
    const client = fakeClient({
      approve: () => ({
        data: {
          template: template({ status: 'RECURRING_HIRE_STATUS_APPROVED' }),
          occurrences: [
            occurrence(),
            occurrence({
              startsAt: '2026-09-14T09:00:00Z',
              outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT',
              bookingId: '',
              reason: 'court double booked',
            }),
            occurrence({
              startsAt: '2026-09-21T09:00:00Z',
              outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR',
              bookingId: '',
              reason: 'pricing rule missing',
            }),
          ],
        },
        error: undefined,
        response: { status: 200 },
      }),
    })
    const wrapper = await mountPanel(client)

    await wrapper.find('[data-testid="approve-template-1"]').trigger('click')
    await flushPromises()

    expect(client.POST).toHaveBeenCalledWith(APPROVE_PATH, {
      params: { path: { templateId: 'template-1' } },
      body: { actorUserId: OWNER },
    })

    // The live region carries the counts (WCAG 4.1.3) — never a bare
    // "Approved".
    const status = wrapper.find('[role="status"]').text()
    expect(status).toContain('1 of 3 weeks booked')
    expect(status).toMatch(/1 skipped because the court was already booked/i)
    expect(status).toMatch(/1 could not be booked/i)

    // And every week is listed individually, with its own outcome and the
    // server's own reason.
    const occurrences = wrapper.find('[data-testid="occurrences-template-1"]')
    expect(occurrences.exists()).toBe(true)
    expect(occurrences.text()).toContain('2026-09-07 09:00')
    expect(occurrences.text()).toContain('Booked')
    expect(occurrences.text()).toMatch(/Skipped — court already booked/)
    expect(occurrences.text()).toContain('court double booked')
    expect(occurrences.text()).toMatch(/Skipped — could not be booked/)
    expect(occurrences.text()).toContain('pricing rule missing')
  })

  it('states the count even when every week succeeded', async () => {
    const wrapper = await mountPanel(fakeClient())

    await wrapper.find('[data-testid="approve-template-1"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[role="status"]').text()).toContain('1 of 1 weeks booked')
  })

  it('updates the request’s status from the server’s response', async () => {
    const wrapper = await mountPanel(fakeClient())

    await wrapper.find('[data-testid="approve-template-1"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="request-status-template-1"]').text()).toBe('Approved')
  })
})

describe('FacilityRentalRequests — no fabricated data', () => {
  // The per-week outcomes only exist in the approval response. A template
  // approved before this screen loaded has none, and inventing "all weeks
  // booked" for it is exactly the fabrication instruction #4 forbids.
  it('does not invent a week-by-week record for a previously approved request', async () => {
    const wrapper = await mountPanel(
      fakeClient({
        list: () => ({
          data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_APPROVED' })] },
          error: undefined,
        }),
      }),
    )

    expect(wrapper.find('[data-testid="occurrences-template-1"]').exists()).toBe(false)
    const note = wrapper.find('[data-testid="no-occurrence-record-template-1"]')
    expect(note.exists()).toBe(true)
    expect(note.text()).toMatch(/is not stored, so it cannot be shown here/i)
    expect(wrapper.text()).not.toMatch(/3 of 3 weeks booked/)
  })

  it('treats a rejection as terminal: no approve control is offered afterwards', async () => {
    const wrapper = await mountPanel(fakeClient())

    await wrapper.find('[data-testid="reject-template-1"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="request-status-template-1"]').text()).toBe('Rejected')
    // The controls are ABSENT, not disabled — scanned with the same
    // multi-signal helper the Club screen's absence assertion uses.
    expect(wrapper.find('[data-testid="approve-template-1"]').exists()).toBe(false)
    expect(findControlsMatching(wrapper, /approve request/i)).toHaveLength(0)

    const status = wrapper.find('[role="status"]').text()
    expect(status).toMatch(/no courts were booked/i)
    expect(status).toMatch(/new one/i)
  })

  it('offers no decision controls for a request that arrives already decided', async () => {
    const wrapper = await mountPanel(
      fakeClient({
        list: () => ({
          data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_REJECTED' })] },
          error: undefined,
        }),
      }),
    )

    expect(findControlsMatching(wrapper, /approve request|reject request/i)).toHaveLength(0)
    expect(wrapper.text()).toMatch(/final/i)
  })

  it('explains an already-answered request rather than a generic failure', async () => {
    const wrapper = await mountPanel(
      fakeClient({
        approve: () => ({ data: undefined, error: { message: 'bad state' }, response: { status: 400 } }),
      }),
    )

    await wrapper.find('[data-testid="approve-template-1"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[role="alert"]').text()).toMatch(/already been answered/i)
  })

  it('renders an unspecified schedule as unspecified rather than as Sunday', async () => {
    const wrapper = await mountPanel(
      fakeClient({
        list: () => ({ data: { templates: [{ id: 'template-9', status: 'RECURRING_HIRE_STATUS_REQUESTED' }] }, error: undefined }),
      }),
    )

    expect(wrapper.text()).toContain('Not specified')
    expect(wrapper.text()).not.toContain('Sundays')
  })
})

describe('FacilityRentalRequests — accessibility', () => {
  it('has no axe violations with a pending request and an approval result', async () => {
    const wrapper = await mountPanel(fakeClient(), true)
    await wrapper.find('[data-testid="approve-template-1"]').trigger('click')
    await flushPromises()

    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    wrapper.unmount()
  })
})
