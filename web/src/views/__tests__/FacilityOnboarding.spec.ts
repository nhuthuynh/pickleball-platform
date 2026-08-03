import { describe, it, expect, vi, beforeAll } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import FacilityOnboarding from '../FacilityOnboarding.vue'
import type { FacilitiesClient } from '../../api/facilitiesClient'

// jsdom doesn't implement matchMedia. useBreakpoint() (src/composables/
// useBreakpoint.ts) is unit-tested against its own injectable `win` mock
// (see useBreakpoint.spec.ts) — here FacilityOnboarding calls it with the
// real ambient `window` (matching how it's actually used in the app), so a
// minimal always-"no match" stub is enough for these tests, which don't
// assert breakpoint-specific rendering.
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

/**
 * A hand-rolled fake standing in for the real `FacilitiesClient`
 * (openapi-fetch's `Client<FacilitiesPaths>`), injected via the
 * component's `client` prop so these tests never touch the ambient
 * `fetch` or need `npm run generate:client` to have run — same "test
 * against a fixture, not the real generated module" spirit as
 * src/api/__tests__/client.spec.ts.
 */
function makeFakeClient(
  overrides: Partial<{
    createFacility: (body: Record<string, unknown>) => { data?: unknown; error?: unknown }
    addCameraLink: (body: Record<string, unknown>) => { data?: unknown; error?: unknown }
    addCourt: (body: Record<string, unknown>) => { data?: unknown; error?: unknown }
  }> = {},
) {
  const createFacility =
    overrides.createFacility ??
    ((body: Record<string, unknown>) => ({
      data: {
        facility: {
          id: 'fac-1',
          ownerId: 'owner-mock-1',
          name: body.name,
          description: body.description,
          address: body.address,
          photoUrls: body.photoUrls ?? [],
          cameraConsentAttested: false,
          cameraLinks: [],
        },
      },
    }))

  const addCameraLink =
    overrides.addCameraLink ??
    ((body: Record<string, unknown>) => ({
      data: {
        facility: {
          id: 'fac-1',
          ownerId: 'owner-mock-1',
          name: 'Sunset Courts',
          description: '',
          address: '123 Main St',
          photoUrls: [],
          cameraConsentAttested: true,
          cameraLinks: [{ url: body.url, courtId: '' }],
        },
      },
    }))

  const addCourt =
    overrides.addCourt ??
    ((body: Record<string, unknown>) => ({
      data: { court: { id: 'court-1', facilityId: 'fac-1', name: body.name } },
    }))

  const POST = vi.fn(async (path: string, init: { params?: unknown; body?: unknown }) => {
    if (path === '/v1/facilities') {
      return createFacility(init.body as Record<string, unknown>)
    }
    if (path === '/v1/facilities/{facilityId}/cameraLinks') {
      return addCameraLink(init.body as Record<string, unknown>)
    }
    if (path === '/v1/facilities/{facilityId}/courts') {
      return addCourt(init.body as Record<string, unknown>)
    }
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { POST, GET: vi.fn() } as unknown as FacilitiesClient
}

/** Fills Details and advances through Photos, triggering the (mocked)
 * `CreateFacility` call, landing on the Cameras step. */
async function advanceToCameras(wrapper: VueWrapper, name = 'Sunset Courts', address = '123 Main St') {
  await wrapper.get('#facility-name').setValue(name)
  await wrapper.get('#facility-address').setValue(address)
  await wrapper.get('[data-testid="details-next"]').trigger('click')
  await wrapper.get('[data-testid="photos-next"]').trigger('click')
  await flushPromises()
}

describe('FacilityOnboarding — camera-consent checkbox (round-10 §2b, load-bearing AC)', () => {
  it('defaults to unchecked: the checkbox has no `checked` attribute and its DOM state is false', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    const checkbox = wrapper.get('#camera-consent-checkbox')
    expect(checkbox.attributes('checked')).toBeUndefined()
    expect((checkbox.element as HTMLInputElement).checked).toBe(false)
  })

  it('does not call AddCameraLink when submitting a camera URL via the UI while consent is unchecked', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    await wrapper.get('#camera-url-input').setValue('https://cam.example.com/1')

    const button = wrapper.get('[data-testid="add-camera-button"]')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    await flushPromises()

    const cameraLinkCalls = (client.POST as ReturnType<typeof vi.fn>).mock.calls.filter(
      (call: unknown[]) => call[0] === '/v1/facilities/{facilityId}/cameraLinks',
    )
    expect(cameraLinkCalls).toHaveLength(0)
  })

  it('the internal submit handler itself refuses to call the API while unchecked (client-side gate, not just a disabled button)', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)
    await wrapper.get('#camera-url-input').setValue('https://cam.example.com/1')

    // Call the handler directly, bypassing any button `disabled` state —
    // this is the "client-side gate too, not just relying on the server
    // 400" requirement from the ticket.
    const exposed = wrapper.vm as unknown as { submitCameraLink: () => Promise<void> }
    await exposed.submitCameraLink()
    await flushPromises()

    const cameraLinkCalls = (client.POST as ReturnType<typeof vi.fn>).mock.calls.filter(
      (call: unknown[]) => call[0] === '/v1/facilities/{facilityId}/cameraLinks',
    )
    expect(cameraLinkCalls).toHaveLength(0)
    expect(wrapper.text()).toContain('Check the consent box')
  })

  it('does call AddCameraLink once consent is actively checked', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    await wrapper.get('#camera-consent-checkbox').setValue(true)
    await wrapper.get('#camera-url-input').setValue('https://cam.example.com/1')
    await wrapper.get('[data-testid="add-camera-button"]').trigger('click')
    await flushPromises()

    const cameraLinkCalls = (client.POST as ReturnType<typeof vi.fn>).mock.calls.filter(
      (call: unknown[]) => call[0] === '/v1/facilities/{facilityId}/cameraLinks',
    )
    expect(cameraLinkCalls).toHaveLength(1)
    expect(wrapper.get('[role="status"]').text()).toBe('Camera link added.')
  })
})

describe('FacilityOnboarding — field-level error rendering (WCAG 3.3.1)', () => {
  it('surfaces a failed CreateFacility validation error as text next to the specific field, not a generic toast', async () => {
    const client = makeFakeClient({
      createFacility: () => ({
        error: { code: 3, message: 'facilities: required facility field is empty: Name' },
      }),
    })
    const wrapper = mount(FacilityOnboarding, { props: { client } })

    // Leave name blank on purpose; fill address so only Name fails.
    await wrapper.get('#facility-address').setValue('123 Main St')
    await wrapper.get('[data-testid="details-next"]').trigger('click')
    await wrapper.get('[data-testid="photos-next"]').trigger('click')
    await flushPromises()

    // The wizard must route back to Details so the error is visible next
    // to the field it's about.
    expect(wrapper.find('[data-testid="details-step"]').exists()).toBe(true)

    const nameError = wrapper.get('#name-error')
    expect(nameError.text()).toContain('name')
    expect(nameError.attributes('role')).toBe('alert')

    // No generic toast/banner anywhere in the rendered output.
    expect(wrapper.find('.toast').exists()).toBe(false)
    expect(wrapper.find('#address-error').exists()).toBe(false)
  })

  it('surfaces an Address validation error next to the address field specifically', async () => {
    const client = makeFakeClient({
      createFacility: () => ({
        error: { code: 3, message: 'facilities: required facility field is empty: Address' },
      }),
    })
    const wrapper = mount(FacilityOnboarding, { props: { client } })

    await wrapper.get('#facility-name').setValue('Sunset Courts')
    await wrapper.get('[data-testid="details-next"]').trigger('click')
    await wrapper.get('[data-testid="photos-next"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('#address-error').text()).toContain('Address')
    expect(wrapper.find('#name-error').exists()).toBe(false)
  })
})

describe('FacilityOnboarding — multi-step forward/back state retention', () => {
  it('retains entered Details data after navigating forward to Photos and back', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })

    await wrapper.get('#facility-name').setValue('Sunset Courts')
    await wrapper.get('#facility-address').setValue('123 Main St')
    await wrapper.get('#facility-description').setValue('Great courts by the river')

    await wrapper.get('[data-testid="details-next"]').trigger('click')
    expect(wrapper.find('[data-testid="photos-step"]').exists()).toBe(true)

    await wrapper.get('[data-testid="photos-back"]').trigger('click')
    expect(wrapper.find('[data-testid="details-step"]').exists()).toBe(true)

    expect((wrapper.get('#facility-name').element as HTMLInputElement).value).toBe(
      'Sunset Courts',
    )
    expect((wrapper.get('#facility-address').element as HTMLInputElement).value).toBe(
      '123 Main St',
    )
    expect((wrapper.get('#facility-description').element as HTMLTextAreaElement).value).toBe(
      'Great courts by the river',
    )
  })

  it('retains photo URLs already added after creating the facility and navigating back from Cameras', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })

    await wrapper.get('#facility-name').setValue('Sunset Courts')
    await wrapper.get('#facility-address').setValue('123 Main St')
    await wrapper.get('[data-testid="details-next"]').trigger('click')

    await wrapper.get('#facility-photo-url').setValue('https://example.com/a.jpg')
    await wrapper.get('.fo-add-row button').trigger('click')
    expect(wrapper.text()).toContain('https://example.com/a.jpg')

    await wrapper.get('[data-testid="photos-next"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="cameras-step"]').exists()).toBe(true)

    await wrapper.get('[data-testid="cameras-back"]').trigger('click')
    expect(wrapper.find('[data-testid="photos-step"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('https://example.com/a.jpg')
  })
})

describe('FacilityOnboarding — courts step', () => {
  it('shows a "Needs pricing" badge and a coming-soon note for a freshly-added court, not a dead pricing button', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    await wrapper.get('[data-testid="cameras-next"]').trigger('click')
    expect(wrapper.find('[data-testid="courts-step"]').exists()).toBe(true)

    await wrapper.get('#court-name-input').setValue('Court 1')
    await wrapper.get('[data-testid="add-court-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Court 1')
    expect(wrapper.get('.fo-badge--needs-pricing').text()).toBe('Needs pricing')
    expect(wrapper.text()).toContain('Pricing setup coming soon.')
    expect(wrapper.find('button[disabled][aria-disabled]').exists()).toBe(false)
  })
})

// T7.7 (proto/pickleball/facilities/v1/facilities.proto) added a required
// actor_user_id to AddCourtRequest/AddCameraLinkRequest, checked server-side
// by domain.Facility.EnsureOwner (403 on empty/mismatched actor). PR #44's
// review flagged that the existing suite gave "zero warning" here because
// no test inspected the request body for an actor field on these two
// endpoints — every test kept passing even though the live flow would
// 403 unconditionally once #43 landed. These two tests close that gap by
// asserting the same MOCK_OWNER_ID used for CreateFacility's ownerId is
// also sent as actorUserId on AddCourt and AddCameraLink, so the acting
// caller matches the facility's owner and EnsureOwner passes.
describe('FacilityOnboarding — actorUserId on AddCourt/AddCameraLink (T7.7 ownership check)', () => {
  it('sends actorUserId matching the facility owner on AddCameraLink', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    await wrapper.get('#camera-consent-checkbox').setValue(true)
    await wrapper.get('#camera-url-input').setValue('https://cam.example.com/1')
    await wrapper.get('[data-testid="add-camera-button"]').trigger('click')
    await flushPromises()

    const cameraLinkCalls = (client.POST as ReturnType<typeof vi.fn>).mock.calls.filter(
      (call: unknown[]) => call[0] === '/v1/facilities/{facilityId}/cameraLinks',
    )
    expect(cameraLinkCalls).toHaveLength(1)
    const [, init] = cameraLinkCalls[0] as [string, { body: Record<string, unknown> }]
    expect(init.body.actorUserId).toBe('owner-mock-1')
  })

  it('sends actorUserId matching the facility owner on AddCourt', async () => {
    const client = makeFakeClient()
    const wrapper = mount(FacilityOnboarding, { props: { client } })
    await advanceToCameras(wrapper)

    await wrapper.get('[data-testid="cameras-next"]').trigger('click')
    await wrapper.get('#court-name-input').setValue('Court 1')
    await wrapper.get('[data-testid="add-court-button"]').trigger('click')
    await flushPromises()

    const addCourtCalls = (client.POST as ReturnType<typeof vi.fn>).mock.calls.filter(
      (call: unknown[]) => call[0] === '/v1/facilities/{facilityId}/courts',
    )
    expect(addCourtCalls).toHaveLength(1)
    const [, init] = addCourtCalls[0] as [string, { body: Record<string, unknown> }]
    expect(init.body.actorUserId).toBe('owner-mock-1')
  })
})
