import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import VenueName from '../VenueName.vue'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

function fakeClient(handler: () => unknown): FacilitiesClient {
  return { GET: vi.fn(handler), POST: vi.fn() } as unknown as FacilitiesClient
}

describe('VenueName', () => {
  it('renders the resolved Name for a known facility', async () => {
    const client = fakeClient(() => ({ data: { facility: { name: 'Riverside Courts' } } }))
    const wrapper = mount(VenueName, { props: { facilityId: 'f1', client } })
    await flushPromises()

    expect(wrapper.text()).toBe('Riverside Courts')
  })

  it('shows the empty-state label for an unset facility id, without a network call', async () => {
    const client = fakeClient(() => ({ data: { facility: { name: 'should not be reached' } } }))
    const wrapper = mount(VenueName, { props: { facilityId: '', emptyLabel: 'No venue set', client } })
    await flushPromises()

    expect(wrapper.text()).toBe('No venue set')
    expect(client.GET).not.toHaveBeenCalled()
  })

  it('shows a distinct failed-lookup label for a real id that does not resolve — not the same message as "unset"', async () => {
    const client = fakeClient(() => ({ data: undefined, error: { code: 5, message: 'not found' } }))
    const wrapper = mount(VenueName, {
      props: { facilityId: 'gone', emptyLabel: 'No venue set', failedLabel: 'Unknown venue', client },
    })
    await flushPromises()

    expect(wrapper.text()).toBe('Unknown venue')
    expect(wrapper.text()).not.toBe('No venue set')
  })

  it('shows a loading label while the lookup is in flight', async () => {
    let resolveGet: (v: unknown) => void = () => {}
    const client = {
      GET: vi.fn(() => new Promise((resolve) => { resolveGet = resolve })),
      POST: vi.fn(),
    } as unknown as FacilitiesClient
    const wrapper = mount(VenueName, { props: { facilityId: 'f1', client } })

    expect(wrapper.text()).toBe('Loading…')

    resolveGet({ data: { facility: { name: 'Riverside Courts' } } })
    await flushPromises()
    expect(wrapper.text()).toBe('Riverside Courts')
  })
})
