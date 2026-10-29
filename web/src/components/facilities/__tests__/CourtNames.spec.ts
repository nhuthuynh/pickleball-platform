// #342: the component that stopped the game detail card printing raw court
// uuids. Mirrors VenueName.spec.ts's shape, because the component mirrors
// VenueName.vue.
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CourtNames from '../CourtNames.vue'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

const COURT_A = 'caa9daca-7efd-4d5c-a9e3-705e34e3f874'
const COURT_B = '89195969-6bb3-433a-9cd5-77381a8fa053'

function fakeClient(handler: () => unknown): FacilitiesClient {
  return { GET: vi.fn(handler), POST: vi.fn() } as unknown as FacilitiesClient
}

const withCourts = (courts: Array<{ id: string; name: string }>) => () => ({
  data: { facility: { id: 'f1', name: 'Riverside Courts' }, courts },
  error: undefined,
})

describe('CourtNames', () => {
  it('renders the resolved name instead of the id — the whole point of #342', async () => {
    const client = fakeClient(withCourts([{ id: COURT_A, name: 'Court 1' }]))
    const wrapper = mount(CourtNames, { props: { facilityId: 'f1', courtIds: [COURT_A], client } })
    await flushPromises()

    expect(wrapper.text()).toBe('Court 1')
    expect(wrapper.text()).not.toContain(COURT_A)
  })

  it('joins several court names', async () => {
    const client = fakeClient(
      withCourts([
        { id: COURT_A, name: 'Court 1' },
        { id: COURT_B, name: 'Court 2' },
      ]),
    )
    const wrapper = mount(CourtNames, {
      props: { facilityId: 'f1', courtIds: [COURT_A, COURT_B], client },
    })
    await flushPromises()

    expect(wrapper.text()).toBe('Court 1, Court 2')
  })

  it('shows the empty-state label with no courts, and makes no network call', async () => {
    const client = fakeClient(withCourts([]))
    const wrapper = mount(CourtNames, {
      props: { facilityId: 'f1', courtIds: [], emptyLabel: 'Not set', client },
    })
    await flushPromises()

    expect(wrapper.text()).toBe('Not set')
    expect(client.GET).not.toHaveBeenCalled()
  })

  it('shows the empty-state label when there is no facility to resolve against', async () => {
    const client = fakeClient(withCourts([{ id: COURT_A, name: 'Court 1' }]))
    const wrapper = mount(CourtNames, {
      props: { facilityId: '', courtIds: [COURT_A], emptyLabel: 'Not set', client },
    })
    await flushPromises()

    expect(wrapper.text()).toBe('Not set')
    expect(client.GET).not.toHaveBeenCalled()
  })

  // A failed lookup is a different fact from "no court set" and must not
  // borrow its wording — the same rule VenueName follows.
  it('shows a distinct failed-lookup label, not the empty-state one', async () => {
    const client = fakeClient(() => ({ data: undefined, error: { code: 5, message: 'not found' } }))
    const wrapper = mount(CourtNames, {
      props: {
        facilityId: 'gone',
        courtIds: [COURT_A],
        emptyLabel: 'Not set',
        failedLabel: 'Unknown court',
        client,
      },
    })
    await flushPromises()

    expect(wrapper.text()).toBe('Unknown court')
    expect(wrapper.text()).not.toBe('Not set')
  })

  it('shows a loading label while the lookup is in flight', async () => {
    let resolveGet: (v: unknown) => void = () => {}
    const client = fakeClient(() => new Promise((res) => { resolveGet = res }))
    const wrapper = mount(CourtNames, {
      props: { facilityId: 'f1', courtIds: [COURT_A], loadingLabel: 'Loading…', client },
    })

    expect(wrapper.text()).toBe('Loading…')

    resolveGet({ data: { facility: { id: 'f1', name: 'R' }, courts: [{ id: COURT_A, name: 'Court 1' }] } })
    await flushPromises()
    expect(wrapper.text()).toBe('Court 1')
  })

  it('re-resolves when the court ids change without a remount', async () => {
    const client = fakeClient(
      withCourts([
        { id: COURT_A, name: 'Court 1' },
        { id: COURT_B, name: 'Court 2' },
      ]),
    )
    const wrapper = mount(CourtNames, { props: { facilityId: 'f1', courtIds: [COURT_A], client } })
    await flushPromises()
    expect(wrapper.text()).toBe('Court 1')

    await wrapper.setProps({ courtIds: [COURT_B] })
    await flushPromises()
    expect(wrapper.text()).toBe('Court 2')
  })

  // The text appears asynchronously with no user action in between, so a
  // screen reader must be told — same reasoning as DisplayName.vue's and
  // VenueName.vue's identical attribute (T10.8/T11.7).
  it('announces asynchronously-arriving text', () => {
    const wrapper = mount(CourtNames, {
      props: { facilityId: 'f1', courtIds: [COURT_A], client: fakeClient(withCourts([])) },
    })

    expect(wrapper.get('[data-testid="court-names"]').attributes('aria-live')).toBe('polite')
  })
})
