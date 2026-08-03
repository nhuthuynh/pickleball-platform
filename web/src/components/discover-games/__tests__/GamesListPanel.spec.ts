import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GamesListPanel from '../GamesListPanel.vue'
import type { GameSummary } from '../../../models/game'

const games: GameSummary[] = [
  {
    id: 'g1',
    hostId: 'host-1',
    venueFacilityId: 'facility-1',
    courtIds: ['court-1'],
    startsAt: '2026-09-01T10:00:00Z',
    endsAt: '2026-09-01T11:00:00Z',
    capacity: 8,
    status: 'GAME_STATUS_SCHEDULED',
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    guestAllowance: 2,
    spotsLeft: 5,
  },
  {
    id: 'g2',
    hostId: 'host-2',
    venueFacilityId: 'facility-2',
    courtIds: ['court-2'],
    startsAt: '2026-09-02T10:00:00Z',
    endsAt: '2026-09-02T11:00:00Z',
    capacity: 4,
    status: 'GAME_STATUS_SCHEDULED',
    paymentMethod: 'PAYMENT_METHOD_CASH',
    guestAllowance: 0,
    spotsLeft: 0,
  },
]

function baseProps() {
  return {
    games: [] as GameSummary[],
    loading: false,
    error: null as string | null,
    selectedId: null as string | null,
    facilityFilter: '',
    dateFilter: '',
  }
}

describe('GamesListPanel', () => {
  it('shows a loading state with real UI (not a blank screen)', () => {
    const wrapper = mount(GamesListPanel, { props: { ...baseProps(), loading: true } })
    expect(wrapper.text()).toContain('Loading games')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('shows an error state with a retry action when the API is unreachable', async () => {
    const wrapper = mount(GamesListPanel, {
      props: { ...baseProps(), error: 'Could not reach the server. Check your connection and try again.' },
    })
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not reach the server')
    const retryButton = wrapper.find('.game-list__retry')
    expect(retryButton.exists()).toBe(true)

    await retryButton.trigger('click')
    expect(wrapper.emitted('search')).toBeTruthy()
  })

  it('shows an empty state (no games yet) with real UI, not a blank screen', () => {
    const wrapper = mount(GamesListPanel, { props: baseProps() })
    expect(wrapper.text()).toContain('No games yet')
    expect(wrapper.findAll('.game-list__item')).toHaveLength(0)
  })

  it('shows a distinct empty-state message when a facility/date filter matches nothing (e.g. a facility with zero games)', () => {
    const wrapper = mount(GamesListPanel, { props: { ...baseProps(), facilityFilter: 'facility-9' } })
    expect(wrapper.text()).toContain('No games match these filters')
  })

  it('renders each game with its time range and a text spots-left indicator', () => {
    const wrapper = mount(GamesListPanel, { props: { ...baseProps(), games } })
    const items = wrapper.findAll('.game-list__item')
    expect(items).toHaveLength(2)
    expect(items[0]!.text()).toContain('5 spots left')
    // WCAG 1.4.1: a full game's status is still conveyed as text ("Full"),
    // not merely a color/style change.
    expect(items[1]!.text()).toContain('Full')
  })

  it('marks the selected game', () => {
    const wrapper = mount(GamesListPanel, { props: { ...baseProps(), games, selectedId: 'g2' } })
    const items = wrapper.findAll('.game-list__item')
    expect(items[1]!.attributes('aria-current')).toBe('true')
    expect(items[0]!.attributes('aria-current')).toBeUndefined()
  })

  it('emits select with the game id when a row is clicked', async () => {
    const wrapper = mount(GamesListPanel, { props: { ...baseProps(), games } })
    await wrapper.findAll('.game-list__item')[1]!.trigger('click')
    expect(wrapper.emitted('select')).toEqual([['g2']])
  })

  it('emits update:facilityFilter/update:dateFilter as the user types', async () => {
    const wrapper = mount(GamesListPanel, { props: baseProps() })
    await wrapper.find('#game-facility-filter').setValue('facility-1')
    expect(wrapper.emitted('update:facilityFilter')).toEqual([['facility-1']])

    await wrapper.find('#game-date-filter').setValue('2026-09-01')
    expect(wrapper.emitted('update:dateFilter')).toEqual([['2026-09-01']])
  })

  it('emits search on form submit', async () => {
    const wrapper = mount(GamesListPanel, { props: baseProps() })
    await wrapper.find('.game-list__search').trigger('submit')
    expect(wrapper.emitted('search')).toBeTruthy()
  })
})
