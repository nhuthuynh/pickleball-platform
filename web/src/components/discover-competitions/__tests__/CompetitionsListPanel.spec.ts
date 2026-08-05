import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CompetitionsListPanel from '../CompetitionsListPanel.vue'
import { mapToCompetitionListing, type CompetitionSummary } from '../../../models/competition'

const COMPETITION: CompetitionSummary = mapToCompetitionListing({
  competition: {
    id: 'c1',
    hostId: 'host-1',
    name: 'Autumn Doubles Ladder',
    venueFacilityId: 'facility-1',
    sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
    capacity: 16,
    guestAllowance: 2,
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    entryFee: { amountCents: '2500', currencyCode: 'USD' },
    format: 'COMPETITION_FORMAT_DOUBLES',
    status: 'COMPETITION_STATUS_SCHEDULED',
  },
  spotsLeft: 3,
} as never)

function props(overrides: Record<string, unknown> = {}) {
  return {
    competitions: [COMPETITION],
    loading: false,
    error: null,
    selectedId: null,
    facilityFilter: '',
    dateFilter: '',
    ...overrides,
  }
}

describe('CompetitionsListPanel', () => {
  it('renders a row per competition with its name and a text spots-left label', () => {
    const wrapper = mount(CompetitionsListPanel, { props: props() })
    const row = wrapper.get('.competition-list__item')
    expect(row.text()).toContain('Autumn Doubles Ladder')
    // WCAG 1.4.1: urgency is carried by words, not by a colour on a bare number.
    expect(row.text()).toContain('3 spots left')
  })

  it('shows a designed loading state, not a blank panel', () => {
    const wrapper = mount(CompetitionsListPanel, { props: props({ loading: true, competitions: [] }) })
    expect(wrapper.get('[role="status"]').text()).toContain('Loading competitions')
    expect(wrapper.find('.competition-list__items').exists()).toBe(false)
  })

  it('shows an error state with a retry action', async () => {
    const wrapper = mount(
      CompetitionsListPanel,
      { props: props({ error: 'Could not load competitions. Please try again.', competitions: [] }) },
    )
    expect(wrapper.get('[role="alert"]').text()).toContain('Could not load competitions')
    await wrapper.get('.competition-list__retry').trigger('click')
    expect(wrapper.emitted('search')).toHaveLength(1)
  })

  it('distinguishes "nothing scheduled" from "nothing matches your filters"', () => {
    const bare = mount(CompetitionsListPanel, { props: props({ competitions: [] }) })
    expect(bare.text()).toContain('No competitions scheduled yet')

    const filtered = mount(
      CompetitionsListPanel,
      { props: props({ competitions: [], facilityFilter: 'facility-9' }) },
    )
    expect(filtered.text()).toContain('No competitions match these filters')
  })

  it('emits the filter values and a search request', async () => {
    const wrapper = mount(CompetitionsListPanel, { props: props() })
    await wrapper.get('#competition-facility-filter').setValue('facility-2')
    await wrapper.get('#competition-date-filter').setValue('2026-09-01')
    await wrapper.get('.competition-list__search').trigger('submit')

    expect(wrapper.emitted('update:facilityFilter')).toEqual([['facility-2']])
    expect(wrapper.emitted('update:dateFilter')).toEqual([['2026-09-01']])
    expect(wrapper.emitted('search')).toHaveLength(1)
  })

  it('emits the selected competition id', async () => {
    const wrapper = mount(CompetitionsListPanel, { props: props() })
    await wrapper.get('.competition-list__item').trigger('click')
    expect(wrapper.emitted('select')).toEqual([['c1']])
  })
})
