import { ref, type Ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'

export interface UseFacilityNameResult {
  name: Ref<string | null>
  loading: Ref<boolean>
  /**
   * True once a lookup has been attempted (a non-empty `facilityId` was
   * given) and did not produce a real `Name` — an unknown/deleted Facility,
   * or any other lookup failure (e.g. network error). Deliberately NOT set
   * for an empty `facilityId` — that is an honest "no venue set" state
   * (`GameSummary.venueFacilityId`/`CompetitionSummary.venueFacilityId` can
   * genuinely be unset), not a failed lookup, so callers must tell the two
   * apart rather than collapsing both into one "Unknown venue" label
   * (T10.8 instruction #4's "no fabricated data" rule, applied to the
   * *type* of empty state, not just its wording).
   */
  failed: Ref<boolean>
  load: (facilityId: string) => Promise<void>
}

/**
 * Resolves one `facilities.Facility`'s `Name` by id (T10.8, closes #98) —
 * a lightweight companion to `useFacilityDetail`, which also fetches courts
 * and is heavier than a Games/Competitions display panel needs just to
 * show a venue name. `client` injectable for tests (defaults to the real
 * `facilitiesClient`), same pattern as `useFacilityDetail`/`useFacilityList`.
 */
export function useFacilityName(client: FacilitiesClient = facilitiesClient): UseFacilityNameResult {
  const name = ref<string | null>(null)
  const loading = ref(false)
  const failed = ref(false)

  async function load(facilityId: string): Promise<void> {
    name.value = null
    failed.value = false
    loading.value = false

    // No id was ever set on the parent Game/Competition — a real, honest
    // state (see `failed`'s doc comment), not a lookup to attempt.
    if (!facilityId) {
      return
    }

    loading.value = true
    try {
      const { data, error } = await client.GET('/v1/facilities/{facilityId}', {
        params: { path: { facilityId } },
      })
      if (error || !data?.facility?.name) {
        failed.value = true
        return
      }
      name.value = data.facility.name
    } catch {
      failed.value = true
    } finally {
      loading.value = false
    }
  }

  return { name, loading, failed, load }
}
