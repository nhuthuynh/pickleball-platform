// T11.6 — the Club rental request + status screen. Mirrors
// FacilityDiscounts.spec.ts: hand-rolled fakes standing in for the real
// clients, injected via the component's props, so these tests never touch the
// ambient `fetch`.
//
// The headline test in this file is the ABSENCE ASSERTION (ticket instruction
// #3, sprint plan A6, T10 retro finding 5): the "Request a recurring rental"
// control must not exist for an actor whose real Roles do not include `club`.
// It uses `findControlsMatching` — the generalized form of
// `findGenderControls`'s multi-signal scan (test-support/
// semanticControlAssertions.ts), NOT a screen-local id/name check — so a
// control identified by `aria-label`, by `<label>` association, or by an ARIA/
// implicit role would still be caught. The paired "IS present for a club"
// test is what proves the assertion can actually see the control: an absence
// test that would pass against an empty page is worth nothing.
import { describe, it, expect, vi, beforeAll } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import ClubRentals from '../ClubRentals.vue'
import type { BookingClient } from '../../api/bookingClient'
import type { IdentityClient } from '../../api/identityClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import { findControlsMatching } from '../../test-support/semanticControlAssertions'
import { expectNoA11yViolations, COMPONENT_MOUNT_OPTIONS } from '../../test-support/axe'

const TEMPLATES_PATH = '/v1/recurring-hire-templates'
const USER_PATH = '/v1/users/{userId}'
const ACTOR = '00000000-0000-4000-b000-000000000010'
const COURT = '00000000-0000-4000-a000-000000000001'

/** The concept the request control names. Anything that offers to start a
 * recurring rental must be findable by it. */
const RENTAL_CONTROL_PATTERN = /recurring rental/i

beforeAll(() => {
  // jsdom doesn't implement matchMedia; useBreakpoint() needs it. Same stub as
  // FacilityDiscounts.spec.ts.
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
    requestedByUserId: ACTOR,
    courtId: COURT,
    weekday: 1,
    startMinute: 540,
    endMinute: 600,
    startsAt: '2026-09-07T00:00:00Z',
    endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 12 },
    status: 'RECURRING_HIRE_STATUS_REQUESTED',
    ...overrides,
  }
}

function fakeBookingClient(
  handlers: { list?: () => unknown; request?: (body: unknown) => unknown } = {},
): BookingClient {
  const GET = vi.fn(async (path: string) => {
    if (path === TEMPLATES_PATH) return handlers.list?.() ?? { data: { templates: [] }, error: undefined }
    throw new Error(`unexpected GET ${path}`)
  })
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === TEMPLATES_PATH) {
      return (
        handlers.request?.(options.body) ?? {
          data: { template: template() },
          error: undefined,
          response: { status: 200 },
        }
      )
    }
    throw new Error(`unexpected POST ${path}`)
  })
  return { GET, POST } as unknown as BookingClient
}

/** `roles` null means "the lookup itself failed" (a thrown fetch), which is a
 * different state from "resolved, and this user is not a club". */
function fakeIdentityClient(roles: string[] | null): IdentityClient {
  const GET = vi.fn(async (path: string) => {
    if (path !== USER_PATH) throw new Error(`unexpected GET ${path}`)
    if (roles === null) throw new Error('network down')
    return {
      data: { user: { id: ACTOR, displayName: 'Riverside Club', roles } },
      error: undefined,
      response: { status: 200 },
    }
  })
  return { GET } as unknown as IdentityClient
}

function fakeFacilitiesClient(): FacilitiesClient {
  const GET = vi.fn(async (path: string) => {
    if (path === '/v1/facilities') {
      return { data: { facilities: [{ id: 'facility-1', name: 'Riverside' }] }, error: undefined }
    }
    if (path === '/v1/facilities/{facilityId}') {
      return {
        data: { facility: { id: 'facility-1', name: 'Riverside' }, courts: [{ id: COURT, name: 'Court 1' }] },
        error: undefined,
      }
    }
    throw new Error(`unexpected GET ${path}`)
  })
  return { GET } as unknown as FacilitiesClient
}

async function mountScreen(
  options: {
    roles?: string[] | null
    booking?: BookingClient
    attach?: boolean
  } = {},
) {
  const booking = options.booking ?? fakeBookingClient()
  const wrapper = mount(ClubRentals, {
    props: {
      client: booking,
      identity: fakeIdentityClient(options.roles === undefined ? ['ROLE_CLUB'] : options.roles),
      facilities: fakeFacilitiesClient(),
      actorUserId: ACTOR,
    },
    ...(options.attach ? { attachTo: document.body } : {}),
  })
  await flushPromises()
  return { wrapper, booking }
}

async function openRequestStep(wrapper: VueWrapper) {
  const tab = wrapper.findAll('button').find((b) => b.text() === 'Request a recurring rental')
  await tab!.trigger('click')
  await flushPromises()
}

async function fillValidRequest(wrapper: VueWrapper) {
  await wrapper.find('#rental-facility').setValue('facility-1')
  await flushPromises()
  await wrapper.find('#rental-court').setValue(COURT)
  await wrapper.find('#rental-weekday').setValue('1')
  await wrapper.find('#rental-start-time').setValue('09:00')
  await wrapper.find('#rental-end-time').setValue('10:00')
  await wrapper.find('#rental-starts-at').setValue('2026-09-07')
}

describe('ClubRentals — the rental-request control is role-gated', () => {
  // ── THE ABSENCE ASSERTION ──────────────────────────────────────────────
  it('offers no rental-request control at all to an actor without the club role', async () => {
    const { wrapper, booking } = await mountScreen({ roles: ['ROLE_PLAYER'] })

    // The multi-signal scan: native id/name, aria-label, <label> association,
    // and ARIA/implicit role + accessible text. Not one signal type.
    expect(findControlsMatching(wrapper, RENTAL_CONTROL_PATTERN)).toHaveLength(0)

    // Absent, not merely disabled — a disabled control is still a control,
    // and the scan above would (correctly) have found one.
    expect(wrapper.find('[data-testid="new-rental-step"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="request-rental-button"]').exists()).toBe(false)
    expect(wrapper.find('form').exists()).toBe(false)
    expect(wrapper.findAll('button[disabled]')).toHaveLength(0)

    // Nothing was sent on this actor's behalf either.
    expect(booking.POST).not.toHaveBeenCalled()
  })

  // The paired positive case. Without it, the assertion above would pass
  // against a screen that renders nothing at all, which is the classic way an
  // absence test rots into a tautology.
  it('DOES offer the control to an actor whose real roles include club', async () => {
    const { wrapper } = await mountScreen({ roles: ['ROLE_PLAYER', 'ROLE_CLUB'] })

    const controls = findControlsMatching(wrapper, RENTAL_CONTROL_PATTERN)
    expect(controls.length).toBeGreaterThan(0)
    expect(wrapper.find('[data-testid="no-club-role-notice"]').exists()).toBe(false)
  })

  // The gate is driven by the actor's REAL roles, read from Identity's
  // GetUser — not by anything the client decided for itself.
  it('reads the role from the identity API rather than assuming it', async () => {
    const identity = fakeIdentityClient(['ROLE_CLUB'])
    const wrapper = mount(ClubRentals, {
      props: {
        client: fakeBookingClient(),
        identity,
        facilities: fakeFacilitiesClient(),
        actorUserId: ACTOR,
      },
    })
    await flushPromises()

    expect(identity.GET).toHaveBeenCalledWith(USER_PATH, { params: { path: { userId: ACTOR } } })
  })

  // Fails CLOSED when the role lookup fails — but does NOT then assert the
  // actor is not a club, because that is not what was learned.
  it('hides the control when the role lookup fails, without claiming the actor is not a club', async () => {
    const { wrapper } = await mountScreen({ roles: null })

    expect(findControlsMatching(wrapper, RENTAL_CONTROL_PATTERN)).toHaveLength(0)

    const notice = wrapper.find('[data-testid="no-club-role-notice"]')
    expect(notice.exists()).toBe(true)
    expect(notice.text()).toMatch(/could not check|could not reach/i)
    expect(notice.text()).not.toMatch(/is not registered as a club/i)
  })

  it('says plainly that the account is not a club once the roles really were read', async () => {
    const { wrapper } = await mountScreen({ roles: ['ROLE_PLAYER'] })

    expect(wrapper.find('[data-testid="no-club-role-notice"]').text()).toMatch(
      /not registered as a club/i,
    )
  })

  // The backend's actor-scoped read is deliberately NOT role-gated (a club
  // that later lost the role must still be able to read its own history,
  // including rejections). The screen matches that.
  it('still shows a non-club actor their own request history', async () => {
    const booking = fakeBookingClient({
      list: () => ({
        data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_REJECTED' })] },
        error: undefined,
      }),
    })
    const { wrapper } = await mountScreen({ roles: ['ROLE_PLAYER'], booking })

    expect(booking.GET).toHaveBeenCalledWith(TEMPLATES_PATH, {
      params: { query: { actorUserId: ACTOR } },
    })
    expect(wrapper.find('[data-testid="rental-list-step"]').text()).toContain('Rejected')
  })
})

describe('ClubRentals — requesting a rental', () => {
  it('sends the real RequestRecurringHire body and confirms through the live region', async () => {
    const { wrapper, booking } = await mountScreen({})
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(booking.POST).toHaveBeenCalledWith(TEMPLATES_PATH, {
      body: {
        actorUserId: ACTOR,
        courtId: COURT,
        weekday: 1,
        startMinute: 540,
        endMinute: 600,
        startsAt: '2026-09-07T00:00:00.000Z',
        endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' },
      },
    })

    // Not "booked" — nothing is booked until the owner approves.
    const status = wrapper.find('[role="status"]').text()
    expect(status).toMatch(/request sent/i)
    expect(status).toMatch(/no courts are booked/i)
  })

  // WCAG 3.3.1 / 3.3.3, and the gate is real code, not a disabled button.
  it('identifies an invalid field in text next to it, with a fix, and sends nothing', async () => {
    const { wrapper, booking } = await mountScreen({})
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)
    await wrapper.find('#rental-end-time').setValue('08:00')

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(booking.POST).not.toHaveBeenCalled()

    const errorEl = wrapper.find('#rental-end-time-error')
    expect(errorEl.exists()).toBe(true)
    expect(errorEl.attributes('role')).toBe('alert')
    expect(errorEl.text()).toMatch(/later than the start time/i)
    expect(wrapper.find('#rental-end-time').attributes('aria-describedby')).toBe(
      'rental-end-time-error',
    )
    expect(wrapper.find('#rental-end-time').attributes('aria-invalid')).toBe('true')
  })

  // T12.5 — the same criterion, stated as a PROPERTY over every field rather
  // than pinned on the one field above. This is what caught the real defect:
  // `#rental-starts-at` renders `#rental-starts-at-error` but hard-coded
  // `aria-describedby="rental-starts-at-hint"`, so its error was the one
  // field error on this screen never programmatically associated with its
  // control. `role="alert"` announces such an error once as it appears, but a
  // screen-reader user who tabs back to the field afterwards to correct it
  // hears only the hint — the error is not part of the control's description
  // (WCAG 3.3.1 Error Identification / 1.3.1 Info and Relationships).
  //
  // The property: for EVERY error this form renders, the control it belongs
  // to must reference that error's id in aria-describedby. Quantified over
  // whatever errors actually render, so a field added later is covered
  // without this test being edited.
  it('associates every rendered field error with its own control, not just some', async () => {
    const { wrapper } = await mountScreen({})
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)
    // Clear two fields whose errors are rendered by DIFFERENT markup shapes:
    // end-time's describedby is error-only, starts-at's also carries a hint.
    await wrapper.find('#rental-starts-at').setValue('')
    await wrapper.find('#rental-end-time').setValue('')

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const errors = wrapper.findAll('.cr-field-error[id]')
    expect(errors.length).toBeGreaterThan(1)

    for (const error of errors) {
      const errorId = error.attributes('id')!
      // Convention on this screen: `<control-id>-error`.
      const control = wrapper.find(`#${errorId.replace(/-error$/, '')}`)
      expect(control.exists(), `no control found for ${errorId}`).toBe(true)

      const describedBy = (control.attributes('aria-describedby') ?? '').split(/\s+/).filter(Boolean)
      expect(describedBy, `${errorId} is rendered but not referenced by its control`).toContain(
        errorId,
      )
      expect(control.attributes('aria-invalid')).toBe('true')
    }
  })

  // The hint must SURVIVE the error rather than be replaced by it: it states
  // the server's own generation rule, which is exactly what a user needs
  // while correcting the field (WCAG 3.3.3 Error Suggestion).
  it('keeps the starts-at hint described alongside its error', async () => {
    const { wrapper } = await mountScreen({})
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)
    await wrapper.find('#rental-starts-at').setValue('')

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const describedBy = wrapper.find('#rental-starts-at').attributes('aria-describedby') ?? ''
    expect(describedBy.split(/\s+/)).toEqual(
      expect.arrayContaining(['rental-starts-at-hint', 'rental-starts-at-error']),
    )
  })

  // The server is still the authority on the role; a 403 gets its own
  // message rather than a generic failure the club cannot act on.
  it('explains a server-side role rejection specifically', async () => {
    const booking = fakeBookingClient({
      request: () => ({ data: undefined, error: { message: 'permission denied' }, response: { status: 403 } }),
    })
    const { wrapper } = await mountScreen({ booking })
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toMatch(/not registered as a club/i)
  })

  it('changing facility clears an already-chosen court', async () => {
    const { wrapper } = await mountScreen({})
    await openRequestStep(wrapper)
    await fillValidRequest(wrapper)
    expect((wrapper.find('#rental-court').element as HTMLSelectElement).value).toBe(COURT)

    await (wrapper.vm as unknown as { selectFacility: (id: string) => Promise<void> }).selectFacility('')
    await flushPromises()

    expect((wrapper.find('#rental-court').element as HTMLSelectElement).value).toBe('')
  })
})

describe('ClubRentals — the status view tells the truth', () => {
  it('shows a rejection as final, and never as something still awaiting a decision', async () => {
    const booking = fakeBookingClient({
      list: () => ({
        data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_REJECTED' })] },
        error: undefined,
      }),
    })
    const { wrapper } = await mountScreen({ booking })

    const text = wrapper.find('[data-testid="rental-list-step"]').text()
    expect(wrapper.find('[data-testid="rental-status-template-1"]').text()).toBe('Rejected')
    expect(text).toMatch(/final/i)
    expect(text).toMatch(/new request/i)
    expect(text).not.toMatch(/awaiting a decision/i)

    // And no control anywhere invites the club to await or chase this
    // decision as though it were still open.
    expect(findControlsMatching(wrapper, /approve|resubmit|awaiting/i)).toHaveLength(0)
  })

  it('does not claim an approved request booked every week', async () => {
    const booking = fakeBookingClient({
      list: () => ({
        data: { templates: [template({ status: 'RECURRING_HIRE_STATUS_APPROVED' })] },
        error: undefined,
      }),
    })
    const { wrapper } = await mountScreen({ booking })

    const text = wrapper.find('[data-testid="rental-list-step"]').text()
    expect(text).toMatch(/already booked were skipped/i)
    expect(text).toMatch(/cannot list which weeks/i)
    expect(text).not.toMatch(/all \d+ weeks|every week is booked/i)
  })

  it('renders a schedule the server did not fully specify as unspecified, not as Sunday', async () => {
    const booking = fakeBookingClient({
      list: () => ({ data: { templates: [{ id: 'template-2', status: 'RECURRING_HIRE_STATUS_REQUESTED' }] }, error: undefined }),
    })
    const { wrapper } = await mountScreen({ booking })

    const text = wrapper.find('[data-testid="rental-list-step"]').text()
    expect(text).toContain('Not specified')
    expect(text).toContain('Time not specified')
    expect(text).not.toContain('Sundays')
  })

  it('shows an empty state rather than an error when nothing has been requested', async () => {
    const { wrapper } = await mountScreen({})

    expect(wrapper.text()).toMatch(/not requested any recurring rentals yet/i)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('reports a failed list load as an error, not as an empty history', async () => {
    const booking = fakeBookingClient({
      list: () => ({ data: undefined, error: { message: 'boom' } }),
    })
    const { wrapper } = await mountScreen({ booking })

    expect(wrapper.find('[role="alert"]').text()).toMatch(/could not load your rental requests/i)
    expect(wrapper.text()).not.toMatch(/not requested any recurring rentals yet/i)
  })
})

describe('ClubRentals — accessibility', () => {
  it('has no axe violations in the club (form visible) state', async () => {
    const { wrapper } = await mountScreen({ attach: true })
    await openRequestStep(wrapper)

    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    wrapper.unmount()
  })

  it('has no axe violations in the non-club (form absent) state', async () => {
    const { wrapper } = await mountScreen({ roles: ['ROLE_PLAYER'], attach: true })

    await expectNoA11yViolations(wrapper.element, COMPONENT_MOUNT_OPTIONS)
    wrapper.unmount()
  })
})
