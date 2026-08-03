import { ref, type Ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'
import { mapToFacilityDetail, type FacilityDetail } from '../models/facility'

export interface UseFacilityDetailResult {
  facility: Ref<FacilityDetail | null>
  loading: Ref<boolean>
  /** Human-readable message, or `null` when the last load succeeded. */
  error: Ref<string | null>
  load: (facilityId: string) => Promise<void>
}

/**
 * Drives the facility detail view (T7.5 requirement #1): fetches
 * `GetFacility` for one facility, tracking loading/error state. `client`
 * is injectable (defaults to the real `facilitiesClient`) — see
 * `useFacilityList`'s doc comment for why.
 *
 * The mapped `FacilityDetail` never carries camera-link data — see
 * `models/facility.ts`'s file header for why that's enforced in the
 * mapping function, not here.
 */
export function useFacilityDetail(client: FacilitiesClient = facilitiesClient): UseFacilityDetailResult {
  const facility = ref<FacilityDetail | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load(facilityId: string): Promise<void> {
    loading.value = true
    error.value = null
    facility.value = null
    try {
      const { data, error: apiError } = await client.GET('/v1/facilities/{facilityId}', {
        params: { path: { facilityId } },
      })
      if (apiError) {
        error.value = 'Could not load this facility. Please try again.'
        return
      }
      if (!data.facility) {
        error.value = 'Could not load this facility. Please try again.'
        return
      }
      facility.value = mapToFacilityDetail(data.facility)
    } catch {
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { facility, loading, error, load }
}
