import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DisplayName from '../DisplayName.vue'
import type { IdentityClient } from '../../../api/identityClient'

function fakeClient(handler: () => unknown): IdentityClient {
  return { GET: vi.fn(handler), POST: vi.fn() } as unknown as IdentityClient
}

describe('DisplayName', () => {
  it('renders the resolved DisplayName for a known user', async () => {
    const client = fakeClient(() => ({ data: { user: { displayName: 'Ada Lovelace' } } }))
    const wrapper = mount(DisplayName, { props: { userId: 'u1', client } })
    await flushPromises()

    expect(wrapper.text()).toBe('Ada Lovelace')
  })

  it('shows a caller-supplied fallback (not a fabricated name) for an unknown user', async () => {
    const client = fakeClient(() => ({ data: undefined, error: { code: 5, message: 'not found' } }))
    const wrapper = mount(DisplayName, { props: { userId: 'gone', fallback: 'Unknown host', client } })
    await flushPromises()

    expect(wrapper.text()).toBe('Unknown host')
  })

  it('shows the fallback for an empty user id, without making a network call', async () => {
    const client = fakeClient(() => ({ data: { user: { displayName: 'should not be reached' } } }))
    const wrapper = mount(DisplayName, { props: { userId: '', fallback: 'Unknown player', client } })
    await flushPromises()

    expect(wrapper.text()).toBe('Unknown player')
    expect(client.GET).not.toHaveBeenCalled()
  })

  it('shows a loading label while the lookup is in flight', async () => {
    let resolveGet: (v: unknown) => void = () => {}
    const client = {
      GET: vi.fn(() => new Promise((resolve) => { resolveGet = resolve })),
      POST: vi.fn(),
    } as unknown as IdentityClient
    const wrapper = mount(DisplayName, { props: { userId: 'u1', client } })

    expect(wrapper.text()).toBe('Loading…')

    resolveGet({ data: { user: { displayName: 'Ada' } } })
    await flushPromises()
    expect(wrapper.text()).toBe('Ada')
  })

  it('re-resolves when userId changes, against the same injected client', async () => {
    const names: Record<string, string> = { u1: 'Ada Lovelace', u2: 'Grace Hopper' }
    const GET = vi.fn(async (_path: string, opts: { params: { path: { userId: string } } }) => ({
      data: { user: { displayName: names[opts.params.path.userId] } },
    }))
    const client = { GET, POST: vi.fn() } as unknown as IdentityClient
    const wrapper = mount(DisplayName, { props: { userId: 'u1', client } })
    await flushPromises()
    expect(wrapper.text()).toBe('Ada Lovelace')

    await wrapper.setProps({ userId: 'u2' })
    await flushPromises()
    expect(wrapper.text()).toBe('Grace Hopper')
  })
})
