import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FacilityDetailPanel from '../FacilityDetailPanel.vue'
import type { FacilityDetail } from '../../../models/facility'

const facility: FacilityDetail = {
  id: 'f1',
  name: 'Riverside Courts',
  description: 'Six outdoor courts by the river.',
  address: '100 River Rd, Springfield',
  photoUrls: ['https://cdn.example.test/riverside-1.jpg'],
  courts: [],
}

function baseProps() {
  return {
    facility: null as FacilityDetail | null,
    loading: false,
    error: null as string | null,
    hasSelection: false,
  }
}

describe('FacilityDetailPanel', () => {
  it('shows a placeholder when no facility is selected yet', () => {
    const wrapper = mount(FacilityDetailPanel, { props: baseProps() })
    expect(wrapper.text()).toContain('Select a facility')
  })

  it('shows a loading state with real UI (not a blank screen)', () => {
    const wrapper = mount(FacilityDetailPanel, {
      props: { ...baseProps(), hasSelection: true, loading: true },
    })
    expect(wrapper.text()).toContain('Loading facility details')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('shows an error state with a retry action when the API is unreachable', async () => {
    const wrapper = mount(FacilityDetailPanel, {
      props: { ...baseProps(), hasSelection: true, error: 'Could not load this facility. Please try again.' },
    })
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not load this facility')
    const retryButton = wrapper.find('.facility-detail__retry')
    await retryButton.trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
  })

  it('renders description, address, and photos for a loaded facility', () => {
    const wrapper = mount(FacilityDetailPanel, { props: { ...baseProps(), hasSelection: true, facility } })
    expect(wrapper.text()).toContain('Riverside Courts')
    expect(wrapper.text()).toContain('100 River Rd, Springfield')
    expect(wrapper.text()).toContain('Six outdoor courts by the river.')
    expect(wrapper.find('img').attributes('src')).toBe('https://cdn.example.test/riverside-1.jpg')
  })

  it('shows an empty state with real UI for a facility with zero courts (not a blank screen)', () => {
    const wrapper = mount(FacilityDetailPanel, {
      props: { ...baseProps(), hasSelection: true, facility: { ...facility, courts: [] } },
    })
    expect(wrapper.text()).toContain("hasn't listed any courts yet")
  })

  it('renders each court by name when the facility has courts', () => {
    const wrapper = mount(FacilityDetailPanel, {
      props: {
        ...baseProps(),
        hasSelection: true,
        facility: { ...facility, courts: [{ id: 'c1', name: 'Court 1' }, { id: 'c2', name: 'Court 2' }] },
      },
    })
    const items = wrapper.findAll('.facility-detail__court span')
    expect(items.map((i) => i.text())).toEqual(['Court 1', 'Court 2'])
  })

  it('emits book-court with the court id and name when a listed court is booked', async () => {
    const wrapper = mount(FacilityDetailPanel, {
      props: {
        ...baseProps(),
        hasSelection: true,
        facility: { ...facility, courts: [{ id: 'c1', name: 'Court 1' }] },
      },
    })
    const bookButton = wrapper.findAll('.facility-detail__court button')[0]
    expect(bookButton).toBeTruthy()
    await bookButton!.trigger('click')
    expect(wrapper.emitted('book-court')).toEqual([['c1', 'Court 1']])
  })

  it('always offers a manual court-ID entry fallback (T7.6) since the courts list is currently always empty', async () => {
    const wrapper = mount(FacilityDetailPanel, { props: { ...baseProps(), hasSelection: true, facility } })

    const bookButton = wrapper.find('.facility-detail__manual-booking button[type="submit"]')
    // Disabled until a court id has been entered.
    expect(bookButton.attributes('disabled')).toBeDefined()

    await wrapper.find('.facility-detail__manual-booking-input').setValue('court-123')
    expect(wrapper.find('.facility-detail__manual-booking button[type="submit"]').attributes('disabled')).toBeUndefined()

    await wrapper.find('.facility-detail__manual-booking').trigger('submit')
    expect(wrapper.emitted('book-court')).toEqual([['court-123']])
  })

  it('does not emit book-court for a blank manually-entered court id', async () => {
    const wrapper = mount(FacilityDetailPanel, { props: { ...baseProps(), hasSelection: true, facility } })

    await wrapper.find('.facility-detail__manual-booking-input').setValue('   ')
    await wrapper.find('.facility-detail__manual-booking').trigger('submit')

    expect(wrapper.emitted('book-court')).toBeFalsy()
  })

  it('never renders camera-link data, even if it were present on the facility object passed in', () => {
    // FacilityDetail (models/facility.ts) has no field for camera links, so
    // this simulates a hypothetical regression via a loose cast — the
    // point of this test is that the template has nothing that could ever
    // render such a field, not that TypeScript would catch the mistake.
    const facilityWithCameraLinkLeak = {
      ...facility,
      cameraLinks: [{ url: 'https://cameras.example.test/riverside/secret-feed', courtId: '' }],
    } as unknown as FacilityDetail

    const wrapper = mount(FacilityDetailPanel, {
      props: { ...baseProps(), hasSelection: true, facility: facilityWithCameraLinkLeak },
    })

    expect(wrapper.html()).not.toContain('cameras.example.test')
    expect(wrapper.html().toLowerCase()).not.toContain('camera')
  })
})
