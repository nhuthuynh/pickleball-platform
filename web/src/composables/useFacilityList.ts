import { ref, type Ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'
import { mapToFacilitySummary, type FacilitySummary } from '../models/facility'

export interface UseFacilityListResult {
  facilities: Ref<FacilitySummary[]>
  loading: Ref<boolean>
  /** Human-readable message, or `null` when the last search succeeded. */
  error: Ref<string | null>
  nameFilter: Ref<string>
  /**
   * Runs `ListFacilities` with the given name filter (or the current
   * `nameFilter.value` when omitted — used for retry buttons). An empty
   * filter returns every Facility, per T7.3's `ListFacilitiesRequest` doc
   * comment.
   */
  search: (filter?: string) => Promise<void>
}

/**
 * Drives the facility list/search view (T7.5 requirement #1): fetches
 * `ListFacilities`, tracking loading/error state for the empty/loading/
 * error UI states (T7.5 requirement #2). `client` is injectable (defaults
 * to the real `facilitiesClient`) so component tests can supply a mock
 * without touching the network — same pattern as `client.ts`'s own
 * injectable `fetch`.
 */
export function useFacilityList(client: FacilitiesClient = facilitiesClient): UseFacilityListResult {
  const facilities = ref<FacilitySummary[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const nameFilter = ref('')

  async function search(filter?: string): Promise<void> {
    if (filter !== undefined) {
      nameFilter.value = filter
    }
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await client.GET('/v1/facilities', {
        params: { query: nameFilter.value ? { nameFilter: nameFilter.value } : {} },
      })
      if (apiError) {
        facilities.value = []
        error.value = 'Could not load facilities. Please try again.'
        return
      }
      facilities.value = (data.facilities ?? []).map(mapToFacilitySummary)
    } catch {
      // Network failure / API unreachable (openapi-fetch propagates a
      // thrown fetch() rejection rather than populating `error` for this
      // case — see api/__tests__/client.spec.ts's non-2xx-vs-thrown note).
      facilities.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { facilities, loading, error, nameFilter, search }
}
