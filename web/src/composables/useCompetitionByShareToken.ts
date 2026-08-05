import { ref, type Ref } from 'vue'
import { competitionsClient, type CompetitionsClient } from '../api/competitionsClient'
import {
  mapToCompetition,
  mapToCompetitionListing,
  isCancelled,
  type CompetitionSummary,
} from '../models/competition'

export interface UseCompetitionByShareTokenResult {
  competition: Ref<CompetitionSummary | null>
  loading: Ref<boolean>
  /**
   * The token did not resolve. Deliberately a SEPARATE signal from `error`:
   * "this link is invalid or expired" and "we couldn't reach the server" are
   * different facts and need different UI (one offers browsing, the other
   * offers a retry). The backend returns the same NotFound for an unknown
   * token, a malformed token, and a token whose Competition is gone — on
   * purpose, so an unauthenticated caller can't probe which tokens are real
   * — so this is the most the client can honestly distinguish.
   */
  linkInvalid: Ref<boolean>
  /** Human-readable message for any OTHER failure. */
  error: Ref<string | null>
  load: (shareToken: string) => Promise<void>
}

/**
 * Resolves a shareable registration link's token into a Competition
 * (`GetCompetitionByShareToken`, T9.5) for the deep-link landing at
 * `/c/:shareToken`.
 *
 * A CANCELLED Competition resolves normally, with
 * `status: COMPETITION_STATUS_CANCELLED` — verified against the backend, not
 * assumed: the RPC's own doc comment states it, and
 * internal/competitions/adapter/grpcapi/sharelink_test.go covers it. A link
 * already published to a social channel outlives the Competition's scheduled
 * state, so a dead link and a cancelled event must not look the same.
 *
 * The token is a CAPABILITY, not an identifier — never log it, never put it
 * in an analytics event, never render it in a page title.
 */
export function useCompetitionByShareToken(
  client: CompetitionsClient = competitionsClient,
): UseCompetitionByShareTokenResult {
  const competition = ref<CompetitionSummary | null>(null)
  const loading = ref(false)
  const linkInvalid = ref(false)
  const error = ref<string | null>(null)

  /**
   * Best-effort second read for the ONE field the token path cannot supply.
   *
   * `GetCompetitionByShareTokenResponse` is field-for-field identical to
   * `GetCompetitionResponse` by design (a backend regression test enforces
   * it), and neither carries `spots_left` — only the public
   * `ListCompetitions` browse read does. Rather than fabricate a number or
   * drop the urgency signal from the one screen that most needs it, this
   * looks the Competition up in that same public read and uses the real
   * server-computed value if, and only if, it finds it.
   *
   * Failures here are swallowed on purpose: `spotsLeft` simply stays `null`
   * and the detail view omits the row. A landing page must not fail because
   * a supplementary read did.
   */
  async function enrichWithSpotsLeft(loaded: CompetitionSummary): Promise<void> {
    try {
      const query: Record<string, string> = {}
      if (loaded.venueFacilityId) query.venueFacilityId = loaded.venueFacilityId
      const { data, error: apiError } = await client.GET('/v1/competitions', { params: { query } })
      if (apiError || !data) return
      const match = (data.competitions ?? [])
        .map(mapToCompetitionListing)
        .find((c) => c.id === loaded.id)
      if (match) loaded.spotsLeft = match.spotsLeft
    } catch {
      // Deliberately ignored — see this function's doc comment.
    }
  }

  async function load(shareToken: string): Promise<void> {
    competition.value = null
    linkInvalid.value = false
    error.value = null

    // An empty token can only come from a malformed URL. Treat it as an
    // invalid link rather than asking the API about nothing.
    if (!shareToken) {
      linkInvalid.value = true
      return
    }

    loading.value = true
    try {
      const { data, error: apiError, response } = await client.GET(
        '/v1/competitions/by-share-token/{shareToken}',
        { params: { path: { shareToken } } },
      )
      if (response.status === 404) {
        linkInvalid.value = true
        return
      }
      if (apiError || !data?.competition) {
        error.value = 'Could not load this competition. Please try again.'
        return
      }
      const loaded = mapToCompetition(data.competition)
      // A cancelled Competition is excluded from the browse read anyway, and
      // cannot be entered, so there is nothing to enrich and no reason to
      // spend the round trip.
      //
      // Enriched BEFORE being published to the ref, so the landing renders
      // once, with a complete Competition, rather than flashing a detail
      // view that gains a spots-left row a moment later.
      if (!isCancelled(loaded.status)) {
        await enrichWithSpotsLeft(loaded)
      }
      competition.value = loaded
    } catch {
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { competition, loading, linkInvalid, error, load }
}
