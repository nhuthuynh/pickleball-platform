import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FacilityListPanel from '../FacilityListPanel.vue'
import type { FacilitySummary } from '../../../models/facility'

const facilities: FacilitySummary[] = [
  { id: 'f1', name: 'Riverside Courts', description: 'Nice courts', address: '1 River Rd' },
  { id: 'f2', name: 'Uptown Pickleball', description: 'Indoor courts', address: '2 Main St' },
]

function baseProps() {
  return {
    facilities: [] as FacilitySummary[],
    loading: false,
    error: null as string | null,
    selectedId: null as string | null,
    nameFilter: '',
  }
}

describe('FacilityListPanel', () => {
  it('shows a loading state with real UI (not a blank screen)', () => {
    const wrapper = mount(FacilityListPanel, { props: { ...baseProps(), loading: true } })
    expect(wrapper.text()).toContain('Loading facilities')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('shows an error state with a retry action when the API is unreachable', async () => {
    const wrapper = mount(FacilityListPanel, {
      props: { ...baseProps(), error: 'Could not reach the server. Check your connection and try again.' },
    })
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not reach the server')
    const retryButton = wrapper.find('.facility-list__retry')
    expect(retryButton.exists()).toBe(true)

    await retryButton.trigger('click')
    expect(wrapper.emitted('search')).toBeTruthy()
  })

  it('shows an empty state (no facilities yet) with real UI, not a blank screen', () => {
    const wrapper = mount(FacilityListPanel, { props: baseProps() })
    expect(wrapper.text()).toContain('No facilities yet')
    expect(wrapper.findAll('.facility-list__item')).toHaveLength(0)
  })

  it('shows a distinct empty-state message when a search filter matches nothing', () => {
    const wrapper = mount(FacilityListPanel, { props: { ...baseProps(), nameFilter: 'nonexistent' } })
    expect(wrapper.text()).toContain('No facilities match "nonexistent"')
  })

  it('renders each facility with its name and address', () => {
    const wrapper = mount(FacilityListPanel, { props: { ...baseProps(), facilities } })
    const items = wrapper.findAll('.facility-list__item')
    expect(items).toHaveLength(2)
    expect(items[0]!.text()).toContain('Riverside Courts')
    expect(items[0]!.text()).toContain('1 River Rd')
  })

  it('marks the selected facility', () => {
    const wrapper = mount(FacilityListPanel, { props: { ...baseProps(), facilities, selectedId: 'f2' } })
    const items = wrapper.findAll('.facility-list__item')
    expect(items[1]!.attributes('aria-current')).toBe('true')
    expect(items[0]!.attributes('aria-current')).toBeUndefined()
  })

  it('emits select with the facility id when a row is clicked', async () => {
    const wrapper = mount(FacilityListPanel, { props: { ...baseProps(), facilities } })
    await wrapper.findAll('.facility-list__item')[1]!.trigger('click')
    expect(wrapper.emitted('select')).toEqual([['f2']])
  })

  it('emits update:nameFilter as the user types', async () => {
    const wrapper = mount(FacilityListPanel, { props: baseProps() })
    await wrapper.find('.facility-list__search-input').setValue('river')
    expect(wrapper.emitted('update:nameFilter')).toEqual([['river']])
  })

  it('emits search on form submit', async () => {
    const wrapper = mount(FacilityListPanel, { props: baseProps() })
    await wrapper.find('.facility-list__search').trigger('submit')
    expect(wrapper.emitted('search')).toBeTruthy()
  })
})
