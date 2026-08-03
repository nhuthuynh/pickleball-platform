import { ref, type Ref } from 'vue'
import { socialplayClient, type SocialPlayClient } from '../api/socialplayClient'
import { mapToGameSummary, type GameSummary } from '../models/game'

export interface UseGameListResult {
  games: Ref<GameSummary[]>
  loading: Ref<boolean>
  /** Human-readable message, or `null` when the last search succeeded. */
  error: Ref<string | null>
  facilityFilter: Ref<string>
  dateFilter: Ref<string>
  /**
   * Runs `ListGames` with the current `facilityFilter`/`dateFilter` (a
   * facility ID and/or a single calendar date, T8.9's facility/date
   * filter). An empty facility filter returns Games at every facility;
   * `dateFilter`, when set, is converted to a `[00:00, 24:00)` local-day
   * window sent as `startsAfter`/`startsBefore` — an empty date filter
   * returns Games starting at any time. Both empty returns every scheduled
   * Game, mirroring `ListFacilitiesRequest.name_filter`'s "empty means
   * unfiltered" convention.
   */
  search: () => Promise<void>
}

/**
 * Drives the games list/search view (T8.9 requirement #1): fetches
 * `ListGames`, tracking loading/error state for the empty/loading/error UI
 * states. `client` is injectable (defaults to the real `socialplayClient`)
 * — same pattern as `useFacilityList`.
 */
export function useGameList(client: SocialPlayClient = socialplayClient): UseGameListResult {
  const games = ref<GameSummary[]>([])
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

      const { data, error: apiError } = await client.GET('/v1/games', {
        params: { query },
      })
      if (apiError) {
        games.value = []
        error.value = 'Could not load games. Please try again.'
        return
      }
      games.value = (data.games ?? []).map(mapToGameSummary)
    } catch {
      // Network failure / API unreachable (openapi-fetch propagates a
      // thrown fetch() rejection rather than populating `error` for this
      // case — see api/__tests__/client.spec.ts's non-2xx-vs-thrown note).
      games.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { games, loading, error, facilityFilter, dateFilter, search }
}
