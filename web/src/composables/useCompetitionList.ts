import { ref, type Ref } from 'vue'
import { competitionsClient, type CompetitionsClient } from '../api/competitionsClient'
import { mapToCompetitionListing, type CompetitionSummary } from '../models/competition'

export interface UseCompetitionListResult {
  competitions: Ref<CompetitionSummary[]>
  loading: Ref<boolean>
  /** Human-readable message, or `null` when the last search succeeded. */
  error: Ref<string | null>
  facilityFilter: Ref<string>
  dateFilter: Ref<string>
  /**
   * Runs `ListCompetitions` with the current `facilityFilter`/`dateFilter`.
   * An empty facility filter returns Competitions at every facility;
   * `dateFilter`, when set, becomes a `[00:00, 24:00)` local-day window sent
   * as `startsAfter`/`startsBefore`. Both empty returns every scheduled
   * Competition — the same "empty means unfiltered" convention
   * `useGameList` and `ListFacilitiesRequest.name_filter` follow.
   *
   * The date window filters on a Competition's EARLIEST session start, since
   * that is what the backend both filters and orders by
   * (ListCompetitionsRequest's doc comment).
   *
   * Only SCHEDULED Competitions ever come back — the backend excludes
   * cancelled ones from this browse read, because a cancelled Competition
   * cannot be entered. A cancelled Competition is still reachable, and still
   * renders honestly, via its share link (T9.5).
   */
  search: () => Promise<void>
}

/**
 * Drives the competitions list/search view (T9.7 requirement #1). Mirrors
 * `useGameList` exactly — including its loading/error handling and the
 * distinction between a non-2xx response and a thrown fetch.
 */
export function useCompetitionList(
  client: CompetitionsClient = competitionsClient,
): UseCompetitionListResult {
  const competitions = ref<CompetitionSummary[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const facilityFilter = ref('')
  const dateFilter = ref('')

  async function search(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const query: Record<string, string> = {}
      if (facilityFilter.value) {
        query.venueFacilityId = facilityFilter.value
      }
      if (dateFilter.value) {
        const dayStart = new Date(`${dateFilter.value}T00:00:00`)
        const dayEnd = new Date(dayStart.getTime() + 24 * 60 * 60 * 1000)
        query.startsAfter = dayStart.toISOString()
        query.startsBefore = dayEnd.toISOString()
      }

      const { data, error: apiError } = await client.GET('/v1/competitions', {
        params: { query },
      })
      if (apiError) {
        competitions.value = []
        error.value = 'Could not load competitions. Please try again.'
        return
      }
      competitions.value = (data.competitions ?? []).map(mapToCompetitionListing)
    } catch {
      // Network failure / API unreachable (openapi-fetch propagates a thrown
      // fetch() rejection rather than populating `error` for this case — see
      // api/__tests__/client.spec.ts's non-2xx-vs-thrown note).
      competitions.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { competitions, loading, error, facilityFilter, dateFilter, search }
}
