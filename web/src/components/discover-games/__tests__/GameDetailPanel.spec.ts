import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GameDetailPanel from '../GameDetailPanel.vue'
import type { GameSummary } from '../../../models/game'
import type { SocialPlayClient } from '../../../api/socialplayClient'
import type { IdentityClient } from '../../../api/identityClient'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

const game: GameSummary = {
  id: 'g1',
  hostId: 'host-1',
  venueFacilityId: 'facility-1',
  courtIds: ['court-1', 'court-2'],
  startsAt: '2026-09-01T10:00:00Z',
  endsAt: '2026-09-01T11:00:00Z',
  capacity: 8,
  status: 'GAME_STATUS_SCHEDULED',
  paymentMethod: 'PAYMENT_METHOD_CASH',
  guestAllowance: 2,
  // T9.2 added entryFeeCents/entryFeeCurrency to GameSummary but this
  // fixture was never updated, so `vue-tsc` failed and the panel rendered
  // `entryFeeLabel(undefined)` -> "$NaN" at runtime. A real (non-free) fee
  // matches the sibling GameJoinPanel fixture. (SCRUM-6: first defect the
  // new CI type-check stage caught.)
  entryFeeCents: 1000,
  entryFeeCurrency: 'USD',
  spotsLeft: 3,
}

function fakeClient(): SocialPlayClient {
  return { GET: vi.fn(), POST: vi.fn() } as unknown as SocialPlayClient
}

/** T10.8: resolves any user id to a fixed DisplayName, so tests unrelated
 * to the Host join don't need to special-case the fetch. */
function fakeIdentityClient(): IdentityClient {
  return {
    GET: vi.fn(async () => ({ data: { user: { displayName: 'Ada Lovelace' } } })),
    POST: vi.fn(),
  } as unknown as IdentityClient
}

/** T10.8: resolves any facility id to a fixed Name, same reasoning as
 * fakeIdentityClient above. */
function fakeFacilitiesClient(): FacilitiesClient {
  return {
    GET: vi.fn(async () => ({ data: { facility: { name: 'Riverside Courts' } } })),
    POST: vi.fn(),
  } as unknown as FacilitiesClient
}

function baseProps() {
  return {
    game: null as GameSummary | null,
    loading: false,
    error: null as string | null,
    hasSelection: false,
    client: fakeClient(),
    identityClient: fakeIdentityClient(),
    facilitiesClient: fakeFacilitiesClient(),
  }
}

describe('GameDetailPanel', () => {
  it('shows a "select a game" placeholder before any selection', () => {
    const wrapper = mount(GameDetailPanel, { props: baseProps() })
    expect(wrapper.text()).toContain('Select a game')
  })

  it('shows a loading state with real UI (not a blank screen)', () => {
    const wrapper = mount(GameDetailPanel, { props: { ...baseProps(), hasSelection: true, loading: true } })
    expect(wrapper.text()).toContain('Loading game details')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('shows an error state with a retry action (API unreachable, or a stale/removed selection)', async () => {
    const wrapper = mount(GameDetailPanel, {
      props: { ...baseProps(), hasSelection: true, error: 'Could not load this game. Please try again.' },
    })
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not load this game')
    const retryButton = wrapper.find('.game-detail__retry')
    expect(retryButton.exists()).toBe(true)

    await retryButton.trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
  })

  it('renders host, court, facility, spots left, and a payment-method label (not a price) for a selected game', async () => {
    const wrapper = mount(GameDetailPanel, { props: { ...baseProps(), hasSelection: true, game } })
    await flushPromises()

    // T10.8 (closes #98): the resolved Host DisplayName and venue Name, not
    // the raw hostId/venueFacilityId.
    expect(wrapper.text()).toContain('Ada Lovelace')
    expect(wrapper.text()).toContain('Riverside Courts')
    expect(wrapper.text()).not.toContain('host-1')
    expect(wrapper.text()).not.toContain('facility-1')
    expect(wrapper.text()).toContain('court-1, court-2')
    // WCAG 1.4.1: spots-left and payment method are text, not color-only.
    expect(wrapper.text()).toContain('3 spots left')
    expect(wrapper.text()).toContain('Cash at facility')
  })

  it('degrades honestly to a named empty state when the Host lookup fails — never a fabricated name', async () => {
    const failingIdentityClient = {
      GET: vi.fn(async () => ({ data: undefined, error: { code: 5, message: 'not found' } })),
      POST: vi.fn(),
    } as unknown as IdentityClient
    const wrapper = mount(GameDetailPanel, {
      props: { ...baseProps(), hasSelection: true, game, identityClient: failingIdentityClient },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Unknown host')
  })

  it('shows an honest "Not set" for a Game with no venue, without a facility lookup', async () => {
    const facilitiesClient = fakeFacilitiesClient()
    const wrapper = mount(GameDetailPanel, {
      props: { ...baseProps(), hasSelection: true, game: { ...game, venueFacilityId: '' }, facilitiesClient },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Not set')
    expect(facilitiesClient.GET).not.toHaveBeenCalled()
  })

  it.each([
    ['PAYMENT_METHOD_ONLINE', 'Online'],
    ['PAYMENT_METHOD_CASH', 'Cash at facility'],
    ['PAYMENT_METHOD_EITHER', 'Online or cash'],
  ])('shows the %s payment method as the label %s', (paymentMethod, label) => {
    const wrapper = mount(GameDetailPanel, {
      props: { ...baseProps(), hasSelection: true, game: { ...game, paymentMethod } },
    })
    expect(wrapper.text()).toContain(label)
  })

  it('renders the join flow (GameJoinPanel) for a selected game', () => {
    const wrapper = mount(GameDetailPanel, { props: { ...baseProps(), hasSelection: true, game } })
    expect(wrapper.find('.game-join').exists()).toBe(true)
  })
})
