import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import GameCreation from '../GameCreation.vue'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import type { SocialPlayClient } from '../../api/socialplayClient'
import { hasHostEvidence, __resetHostEvidenceForTests } from '../../state/roleEvidence'

// jsdom doesn't implement matchMedia — same stub FacilityOnboarding.spec.ts
// uses, since GameCreation calls useBreakpoint() with the real ambient
// `window` and these tests don't assert breakpoint-specific rendering.
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

/** A hand-rolled fake standing in for FacilitiesClient — mirrors
 * FacilityOnboarding.spec.ts's makeFakeClient. Only the two GET operations
 * GameCreation's venue step actually calls (ListFacilities, GetFacility)
 * are implemented. */
function makeFacilitiesClient(
  overrides: Partial<{
    listFacilities: () => { data?: unknown; error?: unknown }
    getFacility: (facilityId: string) => { data?: unknown; error?: unknown }
  }> = {},
) {
  const listFacilities =
    overrides.listFacilities ??
    (() => ({
      data: {
        facilities: [
          { id: 'fac-1', name: 'Sunset Courts', description: '', address: '123 Main St' },
        ],
      },
    }))

  const getFacility =
    overrides.getFacility ??
    ((facilityId: string) => ({
      data: {
        facility: {
          id: facilityId,
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
    }))

  const GET = vi.fn(async (path: string, init: { params?: { path?: { facilityId?: string } } }) => {
    if (path === '/v1/facilities') return listFacilities()
    if (path === '/v1/facilities/{facilityId}') {
      return getFacility(init.params?.path?.facilityId ?? '')
    }
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { GET, POST: vi.fn() } as unknown as FacilitiesClient
}

/** A hand-rolled fake standing in for SocialPlayClient — only CreateGame
 * (the one operation GameCreation calls) is implemented. */
function makeSocialPlayClient(
  overrides: Partial<{
    createGame: (body: Record<string, unknown>) => { data?: unknown; error?: unknown }
  }> = {},
) {
  const createGame =
    overrides.createGame ??
    ((body: Record<string, unknown>) => ({
      data: {
        game: {
          id: 'game-1',
          hostId: body.hostId,
          venueFacilityId: body.venueFacilityId,
          courtIds: body.courtIds,
          startsAt: body.startsAt,
          endsAt: body.endsAt,
          capacity: body.capacity,
          paymentMethod: body.paymentMethod,
          guestAllowance: body.guestAllowance,
        },
      },
    }))

  const POST = vi.fn(async (path: string, init: { body?: unknown }) => {
    if (path === '/v1/games') return createGame(init.body as Record<string, unknown>)
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { POST, GET: vi.fn() } as unknown as SocialPlayClient
}

function mountGameCreation(
  facilitiesClient = makeFacilitiesClient(),
  socialClient = makeSocialPlayClient(),
) {
  return mount(GameCreation, { props: { client: facilitiesClient, socialClient } })
}

/** Advances a freshly-mounted wrapper through Venue -> Schedule -> Payment
 * -> Guests, landing on Review, with one facility + one court selected and
 * valid schedule/payment values. Reused by every test that needs to reach
 * Review or Publish without re-deriving the whole path each time. */
async function advanceToReview(wrapper: VueWrapper) {
  await flushPromises() // onMounted's ListFacilities call

  await wrapper.get('.facility-list__item').trigger('click')
  await flushPromises() // GetFacility call

  const courtCheckboxes = wrapper.findAll('.gc-court-option input[type="checkbox"]')
  await courtCheckboxes[0]!.setValue(true)
  await wrapper.get('[data-testid="venue-next"]').trigger('click')

  await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
  await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
  await wrapper.get('#game-capacity').setValue(4)
  await wrapper.get('[data-testid="schedule-next"]').trigger('click')

  await wrapper.get('input[type="radio"][value="online"]').setValue(true)
  await wrapper.get('[data-testid="payment-next"]').trigger('click')

  await wrapper.get('[data-testid="guests-next"]').trigger('click')
}

describe('GameCreation — omitted matching step (T8.8 kickoff note decision #2)', () => {
  it('renders the "matching isn\'t available yet" note on the review step', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)

    expect(wrapper.find('[data-testid="review-step"]').exists()).toBe(true)
    const note = wrapper.get('[data-testid="matching-note"]')
    expect(note.text()).toContain("Automated matching isn't available yet")
    expect(note.text()).toContain('players join directly')
  })

  // T10.5: the note now names precisely what's built and what's still
  // blocked (ADR-0012), not just "not available yet" — see
  // src/copy/matchingDisclosure.ts.
  it('upgrades the note to name that Identity now exists and the two specific escalated decisions still pending', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)

    const noteText = wrapper.get('[data-testid="matching-note"]').text()
    expect(noteText).toContain('Identity now exists')
    expect(noteText).toContain('Player Level formula is weighted')
    expect(noteText).toContain('gender-mix matching is in scope')
    expect(noteText).toContain('platform owner')
    expect(noteText.toLowerCase()).not.toContain('coming soon')
    expect(noteText.toLowerCase()).not.toContain('next sprint')
  })

  it('has no auto-match toggle, level-range slider, gender-mix selector, or any matching control anywhere in the form', async () => {
    const wrapper = mountGameCreation()

    // Check every step, not just Review — the omission must hold
    // throughout the whole flow, not merely on the step that carries the
    // note.
    await flushPromises()
    expect(wrapper.find('input[type="range"]').exists()).toBe(false)

    await advanceToReview(wrapper)
    expect(wrapper.find('input[type="range"]').exists()).toBe(false)

    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('auto-match')
    expect(text).not.toContain('automatch')
    expect(text).not.toContain('level range')
    expect(text).not.toContain('skill level')

    // "gender" now legitimately appears in the T10.5/ADR-0012 disclosure
    // note itself (naming Q2 — "whether gender-mix matching is in scope" —
    // as a specific blocked decision, not a live feature), so a blanket
    // ban on the word would fail on the honest disclosure text this ticket
    // requires. What must still be absent is any actual gender FIELD or
    // CONTROL — asserted directly, not by banning the word everywhere.
    const genderControls = wrapper
      .findAll('input, select')
      .filter((el) => /gender/i.test(el.attributes('name') ?? '') || /gender/i.test(el.attributes('id') ?? ''))
    expect(genderControls).toHaveLength(0)
  })
})

describe('GameCreation — payment method (3-way selector, required before submit)', () => {
  it('disables Next until a payment method is selected', async () => {
    const wrapper = mountGameCreation()
    await flushPromises()

    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    const courtCheckboxes = wrapper.findAll('.gc-court-option input[type="checkbox"]')
    await courtCheckboxes[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')

    await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
    await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
    await wrapper.get('[data-testid="schedule-next"]').trigger('click')

    expect(wrapper.find('[data-testid="payment-step"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="payment-next"]').attributes('disabled')).toBeDefined()

    await wrapper.get('input[type="radio"][value="cash"]').setValue(true)
    expect(wrapper.get('[data-testid="payment-next"]').attributes('disabled')).toBeUndefined()
  })

  it('offers exactly Online / Cash / Either, no more, no fewer', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)
    await wrapper.get('[data-testid="review-back"]').trigger('click')
    await wrapper.get('[data-testid="guests-back"]').trigger('click')

    const labels = wrapper.findAll('.gc-radio-option span').map((el) => el.text())
    expect(labels).toEqual(['Online', 'Cash', 'Either'])
  })
})

describe('GameCreation — guest allowance stepper (minimum of 0)', () => {
  it('cannot go below 0 via the decrement button, even called directly bypassing its disabled state', async () => {
    const wrapper = mountGameCreation()
    await flushPromises()

    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    const courtCheckboxes = wrapper.findAll('.gc-court-option input[type="checkbox"]')
    await courtCheckboxes[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')
    await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
    await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
    await wrapper.get('[data-testid="schedule-next"]').trigger('click')
    await wrapper.get('input[type="radio"][value="either"]').setValue(true)
    await wrapper.get('[data-testid="payment-next"]').trigger('click')

    expect(wrapper.find('[data-testid="guests-step"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="guest-allowance-value"]').text()).toBe('0')
    expect(wrapper.get('[data-testid="guest-allowance-decrement"]').attributes('disabled')).toBeDefined()

    // Bypass the disabled button entirely (same "the gate is a real code
    // path, not just a UI affordance" reasoning as
    // FacilityOnboarding.spec.ts's camera-consent tests).
    const exposed = wrapper.vm as unknown as { decrementGuestAllowance: () => void }
    exposed.decrementGuestAllowance()
    exposed.decrementGuestAllowance()
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-testid="guest-allowance-value"]').text()).toBe('0')
  })

  it('increments normally and the decrement button re-enables once above 0', async () => {
    const wrapper = mountGameCreation()
    await flushPromises()

    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    const courtCheckboxes = wrapper.findAll('.gc-court-option input[type="checkbox"]')
    await courtCheckboxes[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')
    await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
    await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
    await wrapper.get('[data-testid="schedule-next"]').trigger('click')
    await wrapper.get('input[type="radio"][value="either"]').setValue(true)
    await wrapper.get('[data-testid="payment-next"]').trigger('click')

    await wrapper.get('[data-testid="guest-allowance-increment"]').trigger('click')
    await wrapper.get('[data-testid="guest-allowance-increment"]').trigger('click')

    expect(wrapper.get('[data-testid="guest-allowance-value"]').text()).toBe('2')
    expect(wrapper.get('[data-testid="guest-allowance-decrement"]').attributes('disabled')).toBeUndefined()
  })
})

describe('GameCreation — multi-step forward/back state retention', () => {
  it('retains facility/court selection, schedule, payment method, and guest allowance across forward and back navigation', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)

    // Sanity: review summary reflects everything entered so far.
    expect(wrapper.text()).toContain('Sunset Courts')
    expect(wrapper.text()).toContain('1 selected')

    await wrapper.get('[data-testid="review-back"]').trigger('click')
    expect(wrapper.get('[data-testid="guest-allowance-value"]').text()).toBe('0')

    await wrapper.get('[data-testid="guests-back"]').trigger('click')
    expect(
      (wrapper.get('input[type="radio"][value="online"]').element as HTMLInputElement).checked,
    ).toBe(true)

    await wrapper.get('[data-testid="payment-back"]').trigger('click')
    expect((wrapper.get('#game-starts-at').element as HTMLInputElement).value).toBe(
      '2026-09-01T10:00',
    )
    expect((wrapper.get('#game-ends-at').element as HTMLInputElement).value).toBe(
      '2026-09-01T11:00',
    )
    expect((wrapper.get('#game-capacity').element as HTMLInputElement).value).toBe('4')

    await wrapper.get('[data-testid="schedule-back"]').trigger('click')
    expect(wrapper.find('[data-testid="venue-step"]').exists()).toBe(true)
    const courtCheckboxes = wrapper.findAll('.gc-court-option input[type="checkbox"]')
    expect((courtCheckboxes[0]!.element as HTMLInputElement).checked).toBe(true)
  })
})

describe('GameCreation — field-level error rendering (WCAG 3.3.1)', () => {
  it('surfaces a capacity<=0 rejection next to the Capacity field and routes back to the schedule step', async () => {
    const socialClient = makeSocialPlayClient({
      createGame: () => ({
        error: { code: 3, message: 'socialplay: capacity must be greater than zero' },
      }),
    })
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper)

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="schedule-step"]').exists()).toBe(true)
    expect(wrapper.get('#capacity-error').text()).toContain('Capacity')
    expect(wrapper.get('#capacity-error').attributes('role')).toBe('alert')
  })

  it('surfaces an unknown venue_facility_id rejection next to the Facility field and routes back to the venue step', async () => {
    const socialClient = makeSocialPlayClient({
      createGame: () => ({
        error: { code: 5, message: 'socialplay: venue facility not found' },
      }),
    })
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper)

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="venue-step"]').exists()).toBe(true)
    expect(wrapper.get('#venue-error').text()).toContain('facility')
    expect(wrapper.get('#venue-error').attributes('role')).toBe('alert')
  })

  it('falls back to a generic review-step message for an unrecognized error', async () => {
    const socialClient = makeSocialPlayClient({
      createGame: () => ({ error: { code: 13, message: 'socialplay: something unexpected' } }),
    })
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper)

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="review-step"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('socialplay: something unexpected')
  })
})

describe('GameCreation — successful publish', () => {
  it('calls CreateGame with the expected fields, shows an ARIA live-region success confirmation, and records Host evidence', async () => {
    expect(hasHostEvidence.value).toBe(false)

    const socialClient = makeSocialPlayClient()
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper)

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    const postCalls = (socialClient.POST as ReturnType<typeof vi.fn>).mock.calls as unknown as [
      string,
      { body: Record<string, unknown> },
    ][]
    expect(postCalls).toHaveLength(1)
    const [path, init] = postCalls[0]!
    expect(path).toBe('/v1/games')
    expect(init.body).toMatchObject({
      venueFacilityId: 'fac-1',
      courtIds: ['court-1'],
      capacity: 4,
      paymentMethod: 'PAYMENT_METHOD_ONLINE',
      guestAllowance: 0,
    })
    expect(init.body.hostId).toBeTruthy()

    expect(wrapper.find('[data-testid="published-step"]').exists()).toBe(true)
    const status = wrapper.get('[role="status"]')
    expect(status.text()).toContain('Game published')
    expect(status.text()).toContain("Automated matching isn’t available yet")

    // T8.8 kickoff note decision #1: RoleIndicator now has evidence Host
    // is an available role for this browser session.
    expect(hasHostEvidence.value).toBe(true)
  })

  it('lets the Host create another game after a successful publish, resetting back to the venue step', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)
    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="create-another"]').trigger('click')

    expect(wrapper.find('[data-testid="published-step"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="venue-step"]').exists()).toBe(true)
  })
})

describe('GameCreation — entry fee (T9.2)', () => {
  it('sends the Host\'s real entry fee, in minor units, on publish', async () => {
    const socialClient = makeSocialPlayClient()
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await flushPromises()

    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    await wrapper.findAll('.gc-court-option input[type="checkbox"]')[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')

    await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
    await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
    await wrapper.get('#game-capacity').setValue(4)
    await wrapper.get('[data-testid="schedule-next"]').trigger('click')

    await wrapper.get('input[type="radio"][value="online"]').setValue(true)
    await wrapper.get('[data-testid="entry-fee-input"]').setValue('12.50')
    await wrapper.get('[data-testid="payment-next"]').trigger('click')
    await wrapper.get('[data-testid="guests-next"]').trigger('click')

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    const body = (socialClient.POST as ReturnType<typeof vi.fn>).mock.calls[0]![1].body
    expect(body.entryFee).toEqual({ amountCents: '1250', currencyCode: 'USD' })
  })

  it('publishes a FREE game as an explicit zero amount, not an omitted field', async () => {
    const socialClient = makeSocialPlayClient()
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper) // leaves the fee at its default of 0

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    const body = (socialClient.POST as ReturnType<typeof vi.fn>).mock.calls[0]![1].body
    expect(body.entryFee).toEqual({ amountCents: '0', currencyCode: 'USD' })
  })

  it('shows the fee as the word "Free" at zero on the review step', async () => {
    const wrapper = mountGameCreation()
    await advanceToReview(wrapper)

    expect(wrapper.get('[data-testid="review-entry-fee"]').text()).toBe('Free')
  })

  // WCAG 1.4.1 (Use of Color) + 3.3.1 (Error Identification): the rejection
  // must be readable TEXT sitting next to the field, not a red border.
  it('renders a negative-fee validation error as visible text next to the field', async () => {
    const socialClient = makeSocialPlayClient()
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await flushPromises()

    await wrapper.get('.facility-list__item').trigger('click')
    await flushPromises()
    await wrapper.findAll('.gc-court-option input[type="checkbox"]')[0]!.setValue(true)
    await wrapper.get('[data-testid="venue-next"]').trigger('click')

    await wrapper.get('#game-starts-at').setValue('2026-09-01T10:00')
    await wrapper.get('#game-ends-at').setValue('2026-09-01T11:00')
    await wrapper.get('#game-capacity').setValue(4)
    await wrapper.get('[data-testid="schedule-next"]').trigger('click')

    await wrapper.get('input[type="radio"][value="online"]').setValue(true)
    await wrapper.get('[data-testid="entry-fee-input"]').setValue('-5')
    await flushPromises()

    // The message is visible text, next to the field, as soon as the value
    // is invalid — not hidden behind a submit attempt, and not conveyed by
    // the disabled Next button alone.
    const error = wrapper.get('#entry-fee-error')
    expect(error.text()).toBe("Entry fee can't be negative.")
    expect(error.attributes('role')).toBe('alert')
    expect(wrapper.get('[data-testid="entry-fee-input"]').attributes('aria-describedby')).toBe(
      'entry-fee-error',
    )
    expect(wrapper.get('[data-testid="payment-next"]').attributes('disabled')).toBeDefined()
  })

  it('maps the server\'s ErrInvalidMoney rejection onto the fee field and returns to the payment step', async () => {
    const socialClient = makeSocialPlayClient({
      createGame: () => ({
        error: { code: 3, message: 'socialplay: invalid money amount' },
      }),
    })
    const wrapper = mountGameCreation(makeFacilitiesClient(), socialClient)
    await advanceToReview(wrapper)

    await wrapper.get('[data-testid="publish-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-step"]').exists()).toBe(true)
    const error = wrapper.get('#entry-fee-error')
    expect(error.text()).toBe("Entry fee can't be negative.")
    expect(error.attributes('role')).toBe('alert')
  })
})
