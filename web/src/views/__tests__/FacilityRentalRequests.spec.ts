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

/**
 * Drives a decision all the way through the T12.5 confirm step (WCAG 3.3.4):
 * choose the decision, then confirm it. Every test below that cares about
 * what happens AFTER a decision commits goes through here, so none of them
 * silently depends on the pre-T12.5 single-click behaviour — and if the
 * confirm step were ever removed, they would fail here rather than quietly
 * keep passing against a screen that lost its error prevention.
 */
async function confirmDecision(
  wrapper: Awaited<ReturnType<typeof mountPanel>>,
  decision: 'approve' | 'reject',
  templateId = 'template-1',
): Promise<void> {
  await wrapper.find(`[data-testid="${decision}-${templateId}"]`).trigger('click')
  await flushPromises()
  await wrapper.find(`[data-testid="confirm-${decision}-${templateId}"]`).trigger('click')
  await flushPromises()
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

    await confirmDecision(wrapper, 'approve')

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

    await confirmDecision(wrapper, 'approve')

    expect(wrapper.find('[role="status"]').text()).toContain('1 of 1 weeks booked')
  })

  it('updates the request’s status from the server’s response', async () => {
    const wrapper = await mountPanel(fakeClient())

    await confirmDecision(wrapper, 'approve')

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

    await confirmDecision(wrapper, 'reject')

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

    await confirmDecision(wrapper, 'approve')

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

// ── T12.5 — WCAG 2.2 AA 3.3.4 Error Prevention (Legal/Financial/Data) ──────
//
// Approving generates real Bookings across every implied week, and T11.4
// models BOTH decisions as one-way transitions (see this screen's own header
// comment). T11.6 shipped them as single-click, no-confirm, no-undo controls,
// which is the 3.3.4 failure this block pins shut. The fix follows the
// convention CourtBookingFlow/useCourtBooking already established for this
// criterion — a review/confirm step whose GATE lives in the composable, not
// only in the markup — so the two screens answer 3.3.4 the same way.
//
// WHY THESE ASSERTIONS ARE A PROPERTY, NOT A SIGNAL LIST (T11 retro finding
// 5 / recommendation 7, restated by this ticket's instruction #4). The
// property is:
//
//   for EVERY terminal decision this screen offers, the first activation of
//   its control commits NOTHING to the server, a confirmation control and a
//   way to back out both appear, and only activating the confirmation
//   commits.
//
// It is quantified over the decisions (`it.each`), and every control is
// located with `findControlsMatching` — the shared multi-signal scan from
// test-support/semanticControlAssertions.ts — rather than by the
// `data-testid` this implementation happens to use. That matters in both
// directions: a re-implementation that identifies its confirm button by
// aria-label, by a wrapping <label>, or as a bare `<button>Yes, approve…`
// with no id/name at all still satisfies this test, and a re-implementation
// that quietly drops the confirm step fails it no matter which shape it
// used. No `data-testid` is asserted on anywhere in this block.
const TERMINAL_DECISIONS = [
  {
    decision: 'approve',
    trigger: /^approve request$/i,
    confirm: /yes, approve this request/i,
    backOut: /keep this request open/i,
    path: APPROVE_PATH,
  },
  {
    decision: 'reject',
    trigger: /^reject request$/i,
    confirm: /yes, reject this request/i,
    backOut: /keep this request open/i,
    path: REJECT_PATH,
  },
] as const

describe('FacilityRentalRequests — WCAG 3.3.4 Error Prevention: confirm before commit', () => {
  it.each(TERMINAL_DECISIONS)(
    '$decision: activating the control commits nothing until it is confirmed',
    async ({ trigger, confirm, backOut, path }) => {
      const client = fakeClient()
      const wrapper = await mountPanel(client)

      // 1. The trigger exists, under whatever shape it is identified by.
      const [triggerControl] = findControlsMatching(wrapper, trigger)
      expect(triggerControl, 'no control matching the decision was found').toBeDefined()

      await triggerControl!.trigger('click')
      await flushPromises()

      // 2. THE PROPERTY: the first activation commits NOTHING. Not "did not
      // call this path" — no write of any kind reached the server.
      expect(client.POST).not.toHaveBeenCalled()

      // 3. A confirmation control and a way to back out are both offered.
      // 3.3.4 is satisfied by reversal/checking/confirmation; an
      // acknowledgement with no escape is not a confirmation step.
      expect(findControlsMatching(wrapper, confirm).length).toBeGreaterThan(0)
      expect(findControlsMatching(wrapper, backOut).length).toBeGreaterThan(0)

      // 4. The confirm step SAYS what is about to happen, in text — an
      // owner cannot check a decision the screen never described.
      expect(wrapper.text()).toMatch(/cannot be undone|permanent|final/i)

      // 5. Only confirming commits.
      const [confirmControl] = findControlsMatching(wrapper, confirm)
      await confirmControl!.trigger('click')
      await flushPromises()

      expect(client.POST).toHaveBeenCalledTimes(1)
      expect(client.POST).toHaveBeenCalledWith(path, {
        params: { path: { templateId: 'template-1' } },
        body: { actorUserId: OWNER },
      })
    },
  )

  it.each(TERMINAL_DECISIONS)(
    '$decision: backing out commits nothing and restores the original choice',
    async ({ trigger, backOut }) => {
      const client = fakeClient()
      const wrapper = await mountPanel(client)

      await findControlsMatching(wrapper, trigger)[0]!.trigger('click')
      await flushPromises()
      await findControlsMatching(wrapper, backOut)[0]!.trigger('click')
      await flushPromises()

      expect(client.POST).not.toHaveBeenCalled()
      // Backing out is not a dead end: the request is still answerable.
      expect(findControlsMatching(wrapper, trigger).length).toBeGreaterThan(0)
    },
  )

  it('staging one decision never commits the other', async () => {
    const client = fakeClient()
    const wrapper = await mountPanel(client)

    await findControlsMatching(wrapper, /^approve request$/i)[0]!.trigger('click')
    await flushPromises()
    // While an approval is staged, confirming must not be reachable for the
    // decision that was NOT chosen.
    expect(findControlsMatching(wrapper, /yes, reject this request/i)).toHaveLength(0)
  })

  // THE GATE IS IN THE COMPOSABLE, NOT THE MARKUP. This is the half a
  // markup-only confirm step would leave open, and the reason
  // useCourtBooking puts its own 3.3.4 gate in the composable: the component
  // exposes `approve`/`reject` (defineExpose), so a caller that never
  // touches the buttons could otherwise still fire the irreversible write.
  it.each(TERMINAL_DECISIONS)(
    '$decision: calling the exposed method directly, with nothing staged, commits nothing',
    async ({ decision }) => {
      const client = fakeClient()
      const wrapper = await mountPanel(client)

      const vm = wrapper.vm as unknown as Record<string, (id: string, actor: string) => Promise<boolean>>
      const committed = await vm[decision]!('template-1', OWNER)
      await flushPromises()

      expect(committed).toBe(false)
      expect(client.POST).not.toHaveBeenCalled()
    },
  )

  // WCAG 2.4.3 Focus Order / 2.1.1 Keyboard: the trigger button is REPLACED
  // by the confirm step, so the element holding focus leaves the DOM. Left
  // alone that drops focus to <body> and a keyboard-only owner has to
  // re-traverse the whole queue to reach the confirmation they just asked
  // for. Focus is moved onto the confirm panel instead.
  it('moves focus into the confirm step, and back to the trigger on cancel', async () => {
    const wrapper = await mountPanel(fakeClient(), true)

    await findControlsMatching(wrapper, /^approve request$/i)[0]!.trigger('click')
    await flushPromises()

    const panel = wrapper.get('.frr-confirm')
    expect(panel.attributes('tabindex')).toBe('-1')
    expect(document.activeElement).toBe(panel.element)

    await findControlsMatching(wrapper, /keep this request open/i)[0]!.trigger('click')
    await flushPromises()

    // Focus comes back to the control the owner left, not to <body>.
    expect(document.activeElement).toBe(findControlsMatching(wrapper, /^approve request$/i)[0]!.element)
    wrapper.unmount()
  })
})

describe('FacilityRentalRequests — accessibility', () => {
  it('has no axe violations with a pending request and an approval result', async () => {
    const wrapper = await mountPanel(fakeClient(), true)
    await confirmDecision(wrapper, 'approve')

    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    wrapper.unmount()
  })

  it('has no axe violations while a decision is awaiting confirmation', async () => {
    const wrapper = await mountPanel(fakeClient(), true)
    await findControlsMatching(wrapper, /^approve request$/i)[0]!.trigger('click')
    await flushPromises()

    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    wrapper.unmount()
  })
})
