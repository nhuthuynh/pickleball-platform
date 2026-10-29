import { ref, type Ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'

export interface UseCourtNamesResult {
  /** One label per requested court id, in the order given. Null until a
   *  lookup has produced an answer. */
  names: Ref<string[] | null>
  loading: Ref<boolean>
  /**
   * True once a lookup was attempted (a non-empty facility id and at least
   * one court id) and did not produce names — an unknown/deleted Facility,
   * or any other lookup failure. Deliberately NOT set when there are no
   * court ids: that is an honest "no court set" state, not a failed lookup,
   * the same distinction useFacilityName draws between `emptyLabel` and
   * `failedLabel` (T10.8 instruction #4's no-fabricated-data rule applied
   * to the TYPE of empty state).
   */
  failed: Ref<boolean>
  load: (facilityId: string, courtIds: readonly string[]) => Promise<void>
}

/**
 * Resolves a Game's `courtIds` into `facilities.Court.Name`s (#342).
 *
 * The game detail card resolved a host id to a display name and a facility
 * id to a facility name, and then printed the court's raw uuid beside them
 * — the one place in the whole journey where the product leaked its own
 * primary keys at a player. Found by the first end-to-end run in a browser.
 *
 * Client-side composition against the EXISTING `GetFacility` read, with no
 * server change, for a reason rather than by habit: that response already
 * carries every Court belonging to the Facility (T8.2 added `courts` to
 * GetFacilityResponse precisely so a client would not need a second call),
 * and `/v1/facilities/{facilityId}/courts` is POST-only — AddCourt — so
 * there is no lighter read to use.
 *
 * KNOWN COST, stated rather than hidden: a card showing both a venue name
 * and court names now makes two GetFacility calls, because VenueName.vue
 * resolves the same facility independently. That is the price of the
 * network-calling-child pattern DisplayName/VenueName established, and it
 * is cheap at this size. The point at which it stops being cheap is the one
 * #342 names: if this panel ever needs a third lookup, the read model
 * should return the names and the client should stop round-tripping.
 *
 * Request-sequencing guard against a stale response, same as
 * useFacilityName and useDisplayName — see useDisplayName.ts for the full
 * scenario.
 */
export function useCourtNames(client: FacilitiesClient = facilitiesClient): UseCourtNamesResult {
  const names = ref<string[] | null>(null)
  const loading = ref(false)
  const failed = ref(false)

  let requestId = 0

  async function load(facilityId: string, courtIds: readonly string[]): Promise<void> {
    const thisRequestId = ++requestId
    names.value = null
    failed.value = false
    loading.value = false

    // Nothing to resolve. A Game with no courts, or no facility to resolve
    // them against, is an honest empty state and not a lookup to attempt.
    if (!facilityId || courtIds.length === 0) {
      return
    }

    loading.value = true
    try {
      const { data, error } = await client.GET('/v1/facilities/{facilityId}', {
        params: { path: { facilityId } },
      })
      if (thisRequestId !== requestId) return
      if (error || !data?.courts) {
        failed.value = true
        return
      }

      const byId = new Map<string, string>()
      for (const court of data.courts) {
        if (court.id !== undefined && court.name !== undefined && court.name !== '') {
          byId.set(court.id, court.name)
        }
      }

      // A court id this Facility does not account for keeps its id rather
      // than being dropped or replaced with a guess: the card would
      // otherwise show fewer courts than the Game has, which is a worse
      // lie than an unresolved id. Partial resolution is normal — a court
      // can be removed from a Facility while a Game still references it.
      const resolved = courtIds.map((id) => byId.get(id) ?? id)

      // Every id came back unresolved, so the lookup told us nothing. Report
      // it as a failure rather than rendering a row of raw uuids that looks
      // like a successful answer.
      if (resolved.every((label, i) => label === courtIds[i])) {
        failed.value = true
        return
      }
      names.value = resolved
    } catch {
      if (thisRequestId !== requestId) return
      failed.value = true
    } finally {
      if (thisRequestId === requestId) {
        loading.value = false
      }
    }
  }

  return { names, loading, failed, load }
}
