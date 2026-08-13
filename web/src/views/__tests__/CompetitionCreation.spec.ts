import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import CompetitionCreation from '../CompetitionCreation.vue'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import type { CompetitionsClient } from '../../api/competitionsClient'
import { hasHostEvidence, __resetHostEvidenceForTests } from '../../state/roleEvidence'

// jsdom doesn't implement matchMedia — same stub GameCreation.spec.ts uses.
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

beforeEach(() => {
  __resetHostEvidenceForTests()
})

/** A hand-rolled fake standing in for FacilitiesClient — only the two GET
 * operations the venue step actually calls (ListFacilities, GetFacility),
 * mirroring GameCreation.spec.ts's identical fake. */
function makeFacilitiesClient() {
  const GET = vi.fn(async (path: string, init: { params?: { path?: { facilityId?: string } } }) => {
    if (path === '/v1/facilities') {
      return {
        data: {
          facilities: [
            { id: 'fac-1', name: 'Sunset Courts', description: '', address: '123 Main St' },
          ],
        },
      }
    }
    if (path === '/v1/facilities/{facilityId}') {
      return {
        data: {
          facility: {
            id: init.params?.path?.facilityId ?? '',
            name: 'Sunset Courts',
            description: '',
            address: '123 Main St',
            photoUrls: [],
          },
          courts: [
            { id: 'court-1', name: 'Court 1' },
            { id: 'court-2', name: 'Court 2' },
          ],
        },
      }
    }
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { GET, POST: vi.fn() } as unknown as FacilitiesClient
}

/** The REAL share token this fake's CreateCompetition returns. Tests read
 * the promo's URL off THIS value rather than asserting a hardcoded string
 * in the component — a placeholder would pass an "is there a URL?" check
 * while pointing nowhere. */
const REAL_SHARE_TOKEN = 'A7fQ2xR9-tokenFromTheServer_zZ'

function makeCompetitionsClient(
  overrides: Partial<{
    createCompetition: (body: Record<string, unknown>) => { data?: unknown; error?: unknown }
  }> = {},
) {
  const createCompetition =
    overrides.createCompetition ??
    ((body: Record<string, unknown>) => ({
      data: {
        competition: {
          id: 'comp-1',
          hostId: body.hostId,
          name: body.name,
          venueFacilityId: body.venueFacilityId,
          sessions: body.sessions,
          capacity: body.capacity,
          guestAllowance: body.guestAllowance,
          paymentMethod: body.paymentMethod,
          entryFee: body.entryFee,
          format: body.format,
          status: 'COMPETITION_STATUS_SCHEDULED',
        },
        shareToken: REAL_SHARE_TOKEN,
      },
    }))

  const POST = vi.fn(async (path: string, init: { body?: unknown }) => {
    if (path === '/v1/competitions') return createCompetition(init.body as Record<string, unknown>)
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { POST, GET: vi.fn() } as unknown as CompetitionsClient
}

function mountCreation(
  competitionsClient = makeCompetitionsClient(),
  facilitiesClient = makeFacilitiesClient(),
) {
  return mount(CompetitionCreation, {
    props: { client: facilitiesClient, competitionsClient },
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
  })
}

/** Venue step: name the Competition, pick the facility and both courts. */
async function completeVenueStep(wrapper: VueWrapper) {
  await flushPromises() // onMounted's ListFacilities call
  await wrapper.get('[data-testid="competition-name"]').setValue('Autumn Open')
  await wrapper.get('.facility-list__item').trigger('click')
  await flushPromises() // GetFacility call

  const courts = wrapper.findAll('[data-testid="venue-court-option"] input[type="checkbox"]')
  await courts[0]!.setValue(true)
  await courts[1]!.setValue(true)
  await wrapper.get('[data-testid="venue-next"]').trigger('click')
}

/** Fills session row `index` with a date/time and one court. */
async function fillSessionRow(
  wrapper: VueWrapper,
  index: number,
  values: { date: string; startTime: string; endTime: string; court: number },
) {
  await wrapper.get(`[data-testid="session-date-${index}"]`).setValue(values.date)
  await wrapper.get(`[data-testid="session-start-${index}"]`).setValue(values.startTime)
  await wrapper.get(`[data-testid="session-end-${index}"]`).setValue(values.endTime)
  const courts = wrapper.findAll(`[data-testid="session-court-${index}"] input[type="checkbox"]`)
  await courts[values.court]!.setValue(true)
}

/** Walks a fresh wrapper all the way to the review step with one valid
 * session, so tests that care about publish/advertise don't re-derive the
 * whole path. */
async function advanceToReview(wrapper: VueWrapper) {
  await completeVenueStep(wrapper)

  await fillSessionRow(wrapper, 0, {
    date: '2026-09-01',
    startTime: '10:00',
    endTime: '12:00',
    court: 0,
  })
  await wrapper.get('[data-testid="sessions-next"]').trigger('click')

  await wrapper.get('[data-testid="capacity-input"]').setValue(16)
  await wrapper.get('input[type="radio"][value="COMPETITION_FORMAT_DOUBLES"]').setValue(true)
  await wrapper.get('[data-testid="details-next"]').trigger('click')

  await wrapper.get('input[type="radio"][value="online"]').setValue(true)
  await wrapper.get('[data-testid="payment-next"]').trigger('click')
}

async function publish(wrapper: VueWrapper) {
  await wrapper.get('[data-testid="publish-button"]').trigger('click')
  await flushPromises()
}

// T9.6's routes. Asserted against the real route table (not a copy) so a
// rename or a shadowing route is caught here rather than in a browser.
// NOTE for the T9.7 merge: that ticket adds `/competitions` and
// `/competitions/:id`. `/competitions/new` still wins over
// `/competitions/:id` in vue-router regardless of registration order,
// because a static segment outranks a dynamic one.
describe('CompetitionCreation — routing', () => {
  it('is mounted at /competitions/new, and the roster at /competitions/:id/manage', async () => {
    const { createRouter, createMemoryHistory } = await import('vue-router')
    const { routes } = await import('../../router')
    const router = createRouter({ history: createMemoryHistory(), routes })

    await router.push('/competitions/new')
    expect(router.currentRoute.value.name).toBe('competitions-new')

    await router.push('/competitions/comp-1/manage')
    expect(router.currentRoute.value.name).toBe('competition-manage')
    // `props: true` — the view takes the id as a plain prop.
    expect(router.currentRoute.value.params.id).toBe('comp-1')
  })
})

describe('CompetitionCreation — sessions step (the one step that is NOT a copy of GameCreation)', () => {
  it('starts with one session row and adds another on demand', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(1)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(2)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(3)
  })

  it('removes the right row, keeping the values of the rows either side of it', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await wrapper.get('[data-testid="add-session"]').trigger('click')

    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '09:00', endTime: '10:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-02', startTime: '09:00', endTime: '10:00', court: 0 })
    await fillSessionRow(wrapper, 2, { date: '2026-09-03', startTime: '09:00', endTime: '10:00', court: 0 })

    await wrapper.get('[data-testid="remove-session-1"]').trigger('click')

    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(2)
    expect((wrapper.get('[data-testid="session-date-0"]').element as HTMLInputElement).value).toBe(
      '2026-09-01',
    )
    // The third row's values survived, shifted into index 1 — the removal
    // took the middle row's data with it, not the last row's.
    expect((wrapper.get('[data-testid="session-date-1"]').element as HTMLInputElement).value).toBe(
      '2026-09-03',
    )
  })

  it('never removes the last remaining row — a Competition must have at least one session', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(1)
    // No remove control is offered for a lone row, and calling the handler
    // directly (bypassing the DOM) still can't empty the list — the floor
    // is a real code path, not just a missing button.
    expect(wrapper.find('[data-testid="remove-session-0"]').exists()).toBe(false)

    const exposed = wrapper.vm as unknown as { removeSession: (i: number) => void }
    exposed.removeSession(0)
    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(1)
  })

  // T9.6's explicitly required test: state retention across forward/back.
  it('retains every session row and its values across forward and back navigation', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-02', startTime: '14:00', endTime: '16:00', court: 1 })

    // Forward to details, then payment, then back again.
    await wrapper.get('[data-testid="sessions-next"]').trigger('click')
    await wrapper.get('[data-testid="capacity-input"]').setValue(16)
    await wrapper.get('input[type="radio"][value="COMPETITION_FORMAT_DOUBLES"]').setValue(true)
    await wrapper.get('[data-testid="details-next"]').trigger('click')
    expect(wrapper.find('[data-testid="payment-step"]').exists()).toBe(true)

    await wrapper.get('[data-testid="payment-back"]').trigger('click')
    await wrapper.get('[data-testid="details-back"]').trigger('click')

    expect(wrapper.find('[data-testid="sessions-step"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="session-row"]')).toHaveLength(2)
    expect((wrapper.get('[data-testid="session-date-0"]').element as HTMLInputElement).value).toBe('2026-09-01')
    expect((wrapper.get('[data-testid="session-start-0"]').element as HTMLInputElement).value).toBe('10:00')
    expect((wrapper.get('[data-testid="session-end-0"]').element as HTMLInputElement).value).toBe('12:00')
    expect((wrapper.get('[data-testid="session-date-1"]').element as HTMLInputElement).value).toBe('2026-09-02')
    expect((wrapper.get('[data-testid="session-start-1"]').element as HTMLInputElement).value).toBe('14:00')
    expect((wrapper.get('[data-testid="session-end-1"]').element as HTMLInputElement).value).toBe('16:00')

    // Court selections survived too, per row.
    const row0Courts = wrapper.findAll('[data-testid="session-court-0"] input[type="checkbox"]')
    expect((row0Courts[0]!.element as HTMLInputElement).checked).toBe(true)
    const row1Courts = wrapper.findAll('[data-testid="session-court-1"] input[type="checkbox"]')
    expect((row1Courts[1]!.element as HTMLInputElement).checked).toBe(true)
  })

  it('only offers the courts selected on the venue step', async () => {
    const wrapper = mountCreation()
    await flushPromises()
    await wrapper.get('[data-testid="competition-name"]').setValue('Autumn Open')
    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    // Select ONE of the facility's two courts.
    await wrapper.findAll('[data-testid="venue-court-option"] input[type="checkbox"]')[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')

    expect(wrapper.findAll('[data-testid="session-court-0"] input[type="checkbox"]')).toHaveLength(1)
  })
})

// T9.6's explicitly required test: overlapping-session validation fires AT
// ENTRY, next to the offending row, before submit.
describe('CompetitionCreation — overlapping-session validation (mirrors T9.1, server stays authoritative)', () => {
  it('flags both offending rows as soon as the overlapping value is typed, with no submit attempt', async () => {
    const competitionsClient = makeCompetitionsClient()
    const wrapper = mountCreation(competitionsClient)
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-01', startTime: '11:00', endTime: '13:00', court: 0 })

    // The message is next to each offending row (WCAG 3.3.1), visible text.
    const rowError0 = wrapper.get('[data-testid="session-error-0"]')
    const rowError1 = wrapper.get('[data-testid="session-error-1"]')
    expect(rowError0.text()).toContain('overlaps')
    expect(rowError1.text()).toContain('overlaps')
    expect(rowError0.attributes('role')).toBe('alert')

    // Fired at entry — nothing was submitted to get here.
    expect(competitionsClient.POST).not.toHaveBeenCalled()
    // And the Host can't carry the conflict forward.
    expect(wrapper.get('[data-testid="sessions-next"]').attributes('disabled')).toBeDefined()
  })

  it('clears the flag the moment the conflict is resolved', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-01', startTime: '11:00', endTime: '13:00', court: 0 })
    expect(wrapper.find('[data-testid="session-error-0"]').exists()).toBe(true)

    // Move the second sitting to the next day.
    await wrapper.get('[data-testid="session-date-1"]').setValue('2026-09-02')

    expect(wrapper.find('[data-testid="session-error-0"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="session-error-1"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sessions-next"]').attributes('disabled')).toBeUndefined()
  })

  it('does not flag back-to-back sittings on one court — ranges are half-open', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-01', startTime: '12:00', endTime: '14:00', court: 0 })

    expect(wrapper.find('[data-testid="session-error-0"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sessions-next"]').attributes('disabled')).toBeUndefined()
  })

  it('still surfaces the SERVER\'s overlap rejection on the sessions step — the client check is not a substitute', async () => {
    const competitionsClient = makeCompetitionsClient({
      createCompetition: () => ({
        error: { code: 3, message: 'competitions: sessions overlap on the same court' },
      }),
    })
    const wrapper = mountCreation(competitionsClient)
    await advanceToReview(wrapper)
    await publish(wrapper)

    expect(wrapper.find('[data-testid="sessions-step"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sessions-form-error"]').text()).toContain('overlap')
  })
})

describe('CompetitionCreation — publish', () => {
  it('calls CreateCompetition with every session as its own wire entry', async () => {
    const competitionsClient = makeCompetitionsClient()
    const wrapper = mountCreation(competitionsClient)
    await completeVenueStep(wrapper)

    await wrapper.get('[data-testid="add-session"]').trigger('click')
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await fillSessionRow(wrapper, 1, { date: '2026-09-02', startTime: '10:00', endTime: '12:00', court: 1 })
    await wrapper.get('[data-testid="sessions-next"]').trigger('click')

    await wrapper.get('[data-testid="capacity-input"]').setValue(16)
    await wrapper.get('[data-testid="guest-allowance-increment"]').trigger('click')
    await wrapper.get('input[type="radio"][value="COMPETITION_FORMAT_SINGLES"]').setValue(true)
    await wrapper.get('[data-testid="details-next"]').trigger('click')

    await wrapper.get('input[type="radio"][value="cash"]').setValue(true)
    await wrapper.get('[data-testid="entry-fee-input"]').setValue('25')
    await wrapper.get('[data-testid="payment-next"]').trigger('click')

    await publish(wrapper)

    const calls = (competitionsClient.POST as ReturnType<typeof vi.fn>).mock.calls as unknown as [
      string,
      { body: Record<string, unknown> },
    ][]
    expect(calls).toHaveLength(1)
    const [path, init] = calls[0]!
    expect(path).toBe('/v1/competitions')
    expect(init.body).toMatchObject({
      name: 'Autumn Open',
      venueFacilityId: 'fac-1',
      capacity: 16,
      guestAllowance: 1,
      paymentMethod: 'PAYMENT_METHOD_CASH',
      format: 'COMPETITION_FORMAT_SINGLES',
      entryFee: { amountCents: '2500', currencyCode: 'USD' },
    })

    const sessions = init.body.sessions as { startsAt: string; endsAt: string; courtIds: string[] }[]
    expect(sessions).toHaveLength(2)
    expect(sessions[0]!.courtIds).toEqual(['court-1'])
    expect(sessions[1]!.courtIds).toEqual(['court-2'])
    // Real instants, not the raw strings the Host typed.
    expect(new Date(sessions[0]!.startsAt).toISOString()).toBe(sessions[0]!.startsAt)
  })

  it('publishes a FREE competition as an explicit zero amount, not an omitted field', async () => {
    const competitionsClient = makeCompetitionsClient()
    const wrapper = mountCreation(competitionsClient)
    await advanceToReview(wrapper) // leaves the fee at its default of 0
    await publish(wrapper)

    const body = (competitionsClient.POST as ReturnType<typeof vi.fn>).mock.calls[0]![1].body
    expect(body.entryFee).toEqual({ amountCents: '0', currencyCode: 'USD' })
  })

  // Kickoff note decision #1 / T8.8's precedent: the EXISTING role-evidence
  // mechanism gains a second real signal, not a second mechanism.
  it('records Host evidence for RoleIndicator, only after a real successful response', async () => {
    expect(hasHostEvidence.value).toBe(false)

    const failing = makeCompetitionsClient({
      createCompetition: () => ({ error: { code: 13, message: 'competitions: boom' } }),
    })
    const wrapper = mountCreation(failing)
    await advanceToReview(wrapper)
    await publish(wrapper)
    expect(hasHostEvidence.value).toBe(false)

    const ok = mountCreation()
    await advanceToReview(ok)
    await publish(ok)
    expect(hasHostEvidence.value).toBe(true)
  })

  it('maps a capacity rejection back onto the Capacity field and returns to that step (WCAG 3.3.1)', async () => {
    const competitionsClient = makeCompetitionsClient({
      createCompetition: () => ({
        error: { code: 3, message: 'competitions: capacity must be greater than zero' },
      }),
    })
    const wrapper = mountCreation(competitionsClient)
    await advanceToReview(wrapper)
    await publish(wrapper)

    expect(wrapper.find('[data-testid="details-step"]').exists()).toBe(true)
    expect(wrapper.get('#capacity-error').text()).toContain('Capacity')
    expect(wrapper.get('#capacity-error').attributes('role')).toBe('alert')
  })

  it('maps a court-unavailable rejection onto the sessions step, where the courts were chosen', async () => {
    const competitionsClient = makeCompetitionsClient({
      createCompetition: () => ({
        error: { code: 9, message: 'competitions: court is unavailable for the requested time' },
      }),
    })
    const wrapper = mountCreation(competitionsClient)
    await advanceToReview(wrapper)
    await publish(wrapper)

    expect(wrapper.find('[data-testid="sessions-step"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sessions-form-error"]').text().toLowerCase()).toContain('unavailable')
  })
})

// T9.6's explicitly required test: the share payload carries the REAL token
// URL, read off an actual CreateCompetition response.
describe('CompetitionCreation — advertise step', () => {
  it('builds the promo from the real CreateCompetition response, including the real share token', async () => {
    const competitionsClient = makeCompetitionsClient()
    const wrapper = mountCreation(competitionsClient)
    await advanceToReview(wrapper)
    await publish(wrapper)

    expect(wrapper.find('[data-testid="advertise-step"]').exists()).toBe(true)
    const promo = wrapper.get('[data-testid="promo-text"]').text()

    // The token in the promo is the one the SERVER returned, not a
    // placeholder — asserted against the fake's constant, not a literal.
    expect(promo).toContain(`/c/${REAL_SHARE_TOKEN}`)
    expect(promo).not.toContain('example.com')
    expect(promo).not.toContain('SHARE_TOKEN')
    expect(promo).not.toContain('undefined')

    // ...and the rest of the promo is real, submitted data.
    expect(promo).toContain('Autumn Open')
    expect(promo).toContain('Doubles')
    expect(promo).toContain('Sunset Courts')
    expect(promo).toContain('Free') // the default 0 fee, stated as a word
    expect(promo).toContain('16 spots')
  })

  it('puts the entry fee in the promo when the Host set one', async () => {
    const wrapper = mountCreation()
    await completeVenueStep(wrapper)
    await fillSessionRow(wrapper, 0, { date: '2026-09-01', startTime: '10:00', endTime: '12:00', court: 0 })
    await wrapper.get('[data-testid="sessions-next"]').trigger('click')
    await wrapper.get('[data-testid="capacity-input"]').setValue(16)
    await wrapper.get('input[type="radio"][value="COMPETITION_FORMAT_DOUBLES"]').setValue(true)
    await wrapper.get('[data-testid="details-next"]').trigger('click')
    await wrapper.get('input[type="radio"][value="online"]').setValue(true)
    await wrapper.get('[data-testid="entry-fee-input"]').setValue('25')
    await wrapper.get('[data-testid="payment-next"]').trigger('click')
    await publish(wrapper)

    expect(wrapper.get('[data-testid="promo-text"]').text()).toContain('$25.00')
  })

  // T9.6's explicitly required test: the copy action announces success.
  it('copies the promo and announces it in an ARIA live region (WCAG 4.1.3)', async () => {
    const writeText = vi.fn(async (_text: string) => {})
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
      writable: true,
    })

    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    await publish(wrapper)

    const status = wrapper.get('[data-testid="advertise-status"]')
    expect(status.attributes('role')).toBe('status')
    expect(status.text()).toBe('')

    await wrapper.get('[data-testid="copy-promo"]').trigger('click')
    await flushPromises()

    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText.mock.calls[0]![0]).toContain(`/c/${REAL_SHARE_TOKEN}`)
    expect(wrapper.get('[data-testid="advertise-status"]').text()).toContain('Copied')
  })

  it('says so — in the same live region — when the clipboard write fails, rather than silently claiming success', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: vi.fn(async () => {
          throw new Error('denied')
        }),
      },
      configurable: true,
      writable: true,
    })

    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    await publish(wrapper)
    await wrapper.get('[data-testid="copy-promo"]').trigger('click')
    await flushPromises()

    const status = wrapper.get('[data-testid="advertise-status"]').text()
    expect(status).not.toContain('Copied')
    expect(status.toLowerCase()).toContain('copy')
  })

  it('offers the Web Share API when the browser has one, passing the real share URL', async () => {
    const share = vi.fn(async (_data: { url?: string; text?: string; title?: string }) => {})
    Object.defineProperty(navigator, 'share', { value: share, configurable: true, writable: true })

    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    await publish(wrapper)

    await wrapper.get('[data-testid="share-promo"]').trigger('click')
    await flushPromises()

    expect(share).toHaveBeenCalledTimes(1)
    const payload = share.mock.calls[0]![0]
    expect(payload.url).toContain(`/c/${REAL_SHARE_TOKEN}`)

    Reflect.deleteProperty(navigator, 'share')
  })

  it('hides the Web Share button entirely where the API is absent, rather than offering a dead control', async () => {
    Reflect.deleteProperty(navigator, 'share')

    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    await publish(wrapper)

    expect(wrapper.find('[data-testid="share-promo"]').exists()).toBe(false)
    // The Copy path still exists, so the step is never a dead end.
    expect(wrapper.find('[data-testid="copy-promo"]').exists()).toBe(true)
  })
})

// T9.6's explicitly required ABSENCE assertions — following T8.8's
// precedent, because only an absence assertion catches a well-meaning
// future re-addition of a control that controls nothing.
describe('CompetitionCreation — honest omissions (ADR-0009 at the UI layer)', () => {
  it('has NO "Connect account" affordance and no per-platform Connected/Connect state, anywhere in the flow', async () => {
    const assertNoAccountLinking = (wrapper: VueWrapper) => {
      const text = wrapper.text().toLowerCase()
      expect(text).not.toContain('connect account')
      expect(text).not.toContain('connect your account')
      expect(text).not.toContain('link account')
      expect(text).not.toContain('connected')
      expect(text).not.toContain('disconnect')
      // No per-platform controls of any kind.
      expect(text).not.toContain('facebook')
      expect(text).not.toContain('instagram')
      expect(text).not.toContain('whatsapp')
      expect(text).not.toContain('twitter')
      expect(text).not.toContain('post to')
      expect(wrapper.find('[data-testid="connect-account"]').exists()).toBe(false)
    }

    // Every step of the flow, on its own wrapper, not just the last one.
    const onVenue = mountCreation()
    await flushPromises()
    assertNoAccountLinking(onVenue)

    const onSessions = mountCreation()
    await completeVenueStep(onSessions)
    assertNoAccountLinking(onSessions)

    const onReview = mountCreation()
    await advanceToReview(onReview)
    assertNoAccountLinking(onReview)

    // The advertise step is exactly where a "Connect account" button would
    // be tempting to add — it is the step that replaced Flow 5's "link
    // accounts". It must still not have one.
    await publish(onReview)
    expect(onReview.find('[data-testid="advertise-step"]').exists()).toBe(true)
    assertNoAccountLinking(onReview)
  })

  it('states in one line that automatic posting to linked accounts is not available yet', async () => {
    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    await publish(wrapper)

    const note = wrapper.get('[data-testid="auto-post-note"]')
    expect(note.text().toLowerCase()).toContain("isn't available yet")
    expect(note.text().toLowerCase()).toContain('post')
  })

  it('has NO matching control of any kind, and carries T8.8\'s honest note instead', async () => {
    const onSessions = mountCreation()
    await completeVenueStep(onSessions)
    expect(onSessions.find('input[type="range"]').exists()).toBe(false)

    const wrapper = mountCreation()
    await advanceToReview(wrapper)
    expect(wrapper.find('input[type="range"]').exists()).toBe(false)

    const note = wrapper.get('[data-testid="matching-note"]')
    expect(note.text()).toContain("Automated matching isn't available yet")
    expect(note.text()).toContain('players enter directly')

    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('auto-match')
    expect(text).not.toContain('automatch')
    expect(text).not.toContain('level range')
    expect(text).not.toContain('skill level')
    expect(text).not.toContain('seeding')
    expect(text).not.toContain('bracket')

    // "gender" now legitimately appears in the T10.5/ADR-0012 disclosure
    // note itself (naming Q2 as a specific blocked decision, not a live
    // feature) — see GameCreation.spec.ts's identical note on this same
    // change. What must still be absent is any actual gender FIELD or
    // CONTROL, asserted directly rather than by banning the word.
    const genderControls = wrapper
      .findAll('input, select')
      .filter((el) => /gender/i.test(el.attributes('name') ?? '') || /gender/i.test(el.attributes('id') ?? ''))
    expect(genderControls).toHaveLength(0)
  })

  // T10.5: the note now names precisely what's built and what's still
  // blocked (ADR-0012), not just "not available yet".
  it('upgrades the note to name that Identity now exists and the two specific escalated decisions still pending', async () => {
    const wrapper = mountCreation()
    await advanceToReview(wrapper)

    const noteText = wrapper.get('[data-testid="matching-note"]').text()
    expect(noteText).toContain('Identity now exists')
    expect(noteText).toContain('Player Level formula is weighted')
    expect(noteText).toContain('gender-mix matching is in scope')
    expect(noteText).toContain('platform owner')
    expect(noteText.toLowerCase()).not.toContain('coming soon')
    expect(noteText.toLowerCase()).not.toContain('next sprint')
  })
})
