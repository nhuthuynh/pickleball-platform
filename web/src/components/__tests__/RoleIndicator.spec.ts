import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import RoleIndicator from '../RoleIndicator.vue'
import { recordHostEvidence, __resetHostEvidenceForTests } from '../../state/roleEvidence'

describe('RoleIndicator', () => {
  beforeEach(() => {
    // src/state/roleEvidence.ts's evidence is module-scoped (by design —
    // see that file's header comment), so each test starts from a clean
    // "no evidence yet" state rather than leaking a previous test's
    // recordHostEvidence() call.
    __resetHostEvidenceForTests()
  })

  it('defaults to Player only when this browser session has no Host evidence yet', () => {
    const wrapper = mount(RoleIndicator)
    const options = wrapper.findAll('option').map((o) => o.text())

    expect(options).toEqual(['Player'])
    expect(options).not.toContain('Host')
    expect(options).not.toContain('Owner')
    expect(wrapper.get('.role-indicator__badge').text()).toBe('Player')
  })

  // T8.8 (docs/process/t8-sprint-plan.md kickoff note decision #1): "Host"
  // is listed once this browser session has real evidence for it — a
  // successful CreateGame call (src/views/GameCreation.vue), never
  // hardcoded present unconditionally.
  it('lists Host once this session has recorded Host evidence', () => {
    recordHostEvidence()
    const wrapper = mount(RoleIndicator)
    const options = wrapper.findAll('option').map((o) => o.text())

    expect(options).toEqual(['Player', 'Host'])
    expect(options).not.toContain('Owner')
  })

  it('reactively adds Host to an already-mounted indicator once evidence is recorded elsewhere', async () => {
    const wrapper = mount(RoleIndicator)
    expect(wrapper.findAll('option').map((o) => o.text())).toEqual(['Player'])

    recordHostEvidence()
    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('option').map((o) => o.text())).toEqual(['Player', 'Host'])
  })

  it('switches the displayed role when a different option is selected', async () => {
    recordHostEvidence()
    const wrapper = mount(RoleIndicator)
    await wrapper.get('select').setValue('host')

    expect(wrapper.get('.role-indicator__badge').text()).toBe('Host')
  })
})
