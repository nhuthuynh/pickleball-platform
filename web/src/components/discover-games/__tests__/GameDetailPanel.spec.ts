import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import GameDetailPanel from '../GameDetailPanel.vue'
import type { GameSummary } from '../../../models/game'
import type { SocialPlayClient } from '../../../api/socialplayClient'

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
  spotsLeft: 3,
}

function fakeClient(): SocialPlayClient {
  return { GET: vi.fn(), POST: vi.fn() } as unknown as SocialPlayClient
}

function baseProps() {
  return {
    game: null as GameSummary | null,
    loading: false,
    error: null as string | null,
    hasSelection: false,
    client: fakeClient(),
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

  it('renders host, court, facility, spots left, and a payment-method label (not a price) for a selected game', () => {
    const wrapper = mount(GameDetailPanel, { props: { ...baseProps(), hasSelection: true, game } })

    expect(wrapper.text()).toContain('host-1')
    expect(wrapper.text()).toContain('facility-1')
    expect(wrapper.text()).toContain('court-1, court-2')
    // WCAG 1.4.1: spots-left and payment method are text, not color-only.
    expect(wrapper.text()).toContain('3 spots left')
    expect(wrapper.text()).toContain('Cash at facility')
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
