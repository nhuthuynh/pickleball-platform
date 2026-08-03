import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RoleIndicator from '../RoleIndicator.vue'

describe('RoleIndicator', () => {
  it('defaults to the first mock role (Player) and only lists roles with mock "evidence"', () => {
    const wrapper = mount(RoleIndicator)
    const options = wrapper.findAll('option').map((o) => o.text())

    // Placeholder data per decision #1 (t7-sprint-plan.md kickoff note): a
    // Player who has also hosted (both have "evidence"), but never
    // onboarded a facility, so "Owner" must not appear.
    expect(options).toEqual(['Player', 'Host'])
    expect(options).not.toContain('Owner')
    expect(wrapper.get('.role-indicator__badge').text()).toBe('Player')
  })

  it('switches the displayed role when a different option is selected', async () => {
    const wrapper = mount(RoleIndicator)
    await wrapper.get('select').setValue('host')

    expect(wrapper.get('.role-indicator__badge').text()).toBe('Host')
  })
})
