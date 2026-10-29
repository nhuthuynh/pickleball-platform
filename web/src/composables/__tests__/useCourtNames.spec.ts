// #342: a Game's courts resolve to names, not raw uuids.
import { describe, it, expect, vi } from 'vitest'

import { useCourtNames } from '../useCourtNames'
import type { FacilitiesClient } from '../../api/facilitiesClient'

const FACILITY = 'fac-1'
const COURT_A = 'caa9daca-7efd-4d5c-a9e3-705e34e3f874'
const COURT_B = '89195969-6bb3-433a-9cd5-77381a8fa053'

function fakeClient(result: unknown, spy = vi.fn()): FacilitiesClient {
  spy.mockResolvedValue(result)
  return { GET: spy, POST: vi.fn() } as unknown as FacilitiesClient
}

function facilityWithCourts(courts: Array<{ id?: string; name?: string }>) {
  return { data: { facility: { id: FACILITY, name: 'Riverside Courts' }, courts }, error: undefined }
}

describe('useCourtNames', () => {
  it('resolves ids to names, in the order asked for', async () => {
    const { names, failed, load } = useCourtNames(
      fakeClient(
        facilityWithCourts([
          { id: COURT_B, name: 'Court 2' },
          { id: COURT_A, name: 'Court 1' },
        ]),
      ),
    )

    await load(FACILITY, [COURT_A, COURT_B])

    // Asked for A then B; the response listed B first. Order follows the
    // Game's courtIds, not the Facility's courts.
    expect(names.value).toEqual(['Court 1', 'Court 2'])
    expect(failed.value).toBe(false)
  })

  it('reads the courts from the facility response, with one request', async () => {
    const spy = vi.fn()
    const { load } = useCourtNames(fakeClient(facilityWithCourts([{ id: COURT_A, name: 'Court 1' }]), spy))

    await load(FACILITY, [COURT_A])

    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy).toHaveBeenCalledWith('/v1/facilities/{facilityId}', {
      params: { path: { facilityId: FACILITY } },
    })
  })

  // Partial resolution is normal: a court can be removed from a Facility
  // while a Game still references it. Showing fewer courts than the Game
  // has would be a worse lie than an unresolved id.
  it('keeps an unresolvable id rather than dropping it', async () => {
    const { names, failed, load } = useCourtNames(
      fakeClient(facilityWithCourts([{ id: COURT_A, name: 'Court 1' }])),
    )

    await load(FACILITY, [COURT_A, 'gone-from-this-facility'])

    expect(names.value).toEqual(['Court 1', 'gone-from-this-facility'])
    expect(failed.value).toBe(false)
  })

  // If NOTHING resolved, the lookup told us nothing — reporting that as
  // success would render a row of raw uuids that looks like an answer,
  // which is the defect this composable exists to remove.
  it('fails when no id resolved at all', async () => {
    const { names, failed, load } = useCourtNames(fakeClient(facilityWithCourts([])))

    await load(FACILITY, [COURT_A, COURT_B])

    expect(failed.value).toBe(true)
    expect(names.value).toBeNull()
  })

  it('fails on a lookup error', async () => {
    const { names, failed, load } = useCourtNames(
      fakeClient({ data: undefined, error: { message: 'boom' } }),
    )

    await load(FACILITY, [COURT_A])

    expect(failed.value).toBe(true)
    expect(names.value).toBeNull()
  })

  it('fails when the client throws', async () => {
    const spy = vi.fn().mockRejectedValue(new Error('network'))
    const { failed, load } = useCourtNames({ GET: spy, POST: vi.fn() } as unknown as FacilitiesClient)

    await load(FACILITY, [COURT_A])

    expect(failed.value).toBe(true)
  })

  // An honest "no court set", distinct from a failed lookup — the same
  // distinction useFacilityName draws. Neither case should make a request.
  it.each([
    ['no court ids', FACILITY, [] as string[]],
    ['no facility id', '', [COURT_A]],
  ])('does not attempt a lookup or report failure with %s', async (_label, facilityId, courtIds) => {
    const spy = vi.fn()
    const { names, failed, load } = useCourtNames(fakeClient(facilityWithCourts([]), spy))

    await load(facilityId, courtIds)

    expect(spy).not.toHaveBeenCalled()
    expect(failed.value).toBe(false)
    expect(names.value).toBeNull()
  })

  it('ignores a court with no name rather than labelling it empty', async () => {
    const { names, load } = useCourtNames(
      fakeClient(
        facilityWithCourts([
          { id: COURT_A, name: '' },
          { id: COURT_B, name: 'Court 2' },
        ]),
      ),
    )

    await load(FACILITY, [COURT_A, COURT_B])

    expect(names.value).toEqual([COURT_A, 'Court 2'])
  })

  // Same guard as useFacilityName/useDisplayName: the component is not
  // remounted when the ids change, so a slower earlier call must not
  // clobber a newer one.
  it('discards a stale response', async () => {
    let resolveFirst: ((v: unknown) => void) | undefined
    const spy = vi
      .fn()
      .mockImplementationOnce(() => new Promise((res) => { resolveFirst = res }))
      .mockResolvedValueOnce(facilityWithCourts([{ id: COURT_B, name: 'Court 2' }]))
    const { names, load } = useCourtNames({ GET: spy, POST: vi.fn() } as unknown as FacilitiesClient)

    const first = load(FACILITY, [COURT_A])
    await load(FACILITY, [COURT_B])
    resolveFirst?.(facilityWithCourts([{ id: COURT_A, name: 'Court 1' }]))
    await first

    expect(names.value).toEqual(['Court 2'])
  })
})
