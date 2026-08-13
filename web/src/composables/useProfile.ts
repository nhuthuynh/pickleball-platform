import { ref, type Ref } from 'vue'
import { identityClient, type IdentityClient } from '../api/identityClient'
import { mapToUserProfile, type UserProfile } from '../models/identity'

/**
 * There is no JWT/session layer yet (HANDOFF.md cross-cutting Auth
 * backlog) — same known gap `useJoinGame.ts`'s `MOCK_PLAYER_ID` documents
 * for Social Play, reused here rather than a second placeholder, per that
 * file's own "one mock identity ... keeps the gap visible in one place
 * instead of two" reasoning (`useEnterCompetition.ts` already does the
 * same for Competitions).
 *
 * Identity/Users' own malformed-ID boundary guard (T10.2 item 4,
 * `internal/identity/app/service.go`'s `uuidShape`) means a non-UUID-shaped
 * id like this one is answered with the identical `NotFound` an unknown
 * *well-formed* id would get — so against the real backend `load` below
 * always resolves to the "no profile yet" state today. That is honest, not
 * a bug to route around: this ticket builds no sign-up/CreateUser flow (see
 * Profile.vue's header comment), so no User row exists for this session
 * regardless of the id's shape. Swap this for a real session-derived id the
 * moment one exists — tracked as the same follow-up `MOCK_PLAYER_ID`/
 * `MOCK_HOST_ID`/`MOCK_OWNER_ID` already are.
 */
export { MOCK_PLAYER_ID } from './useJoinGame'

export interface UseProfileResult {
  profile: Ref<UserProfile | null>
  loading: Ref<boolean>
  /** Human-readable message, or `null` when the last load succeeded. Also
   * `null` on a plain "no profile exists yet" result — that is not an
   * error, see `notFound`. */
  error: Ref<string | null>
  /** Set when `GetUser` resolves NotFound — the honest "no profile yet"
   * state (T10.5 instructions #4), kept distinct from `error` so the view
   * can render a specific, calm explanation rather than a generic failure
   * message. */
  notFound: Ref<boolean>
  saving: Ref<boolean>
  /** Human-readable message for the most recent `UpdateSelfReportedLevel`
   * failure. `null` once a save succeeds or before one is attempted. */
  saveError: Ref<string | null>
  load: (userId: string) => Promise<void>
  updateLevel: (userId: string, actorUserId: string, level: number) => Promise<void>
}

/**
 * Drives the Profile screen (T10.5): `GetUser` for the read side,
 * `UpdateSelfReportedLevel` for the one editable field. `client` is
 * injectable (defaults to the real `identityClient`), same pattern as
 * `useFacilityDetail`/`useEnterCompetition`.
 */
export function useProfile(client: IdentityClient = identityClient): UseProfileResult {
  const profile = ref<UserProfile | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const notFound = ref(false)
  const saving = ref(false)
  const saveError = ref<string | null>(null)

  async function load(userId: string): Promise<void> {
    loading.value = true
    error.value = null
    notFound.value = false
    profile.value = null
    try {
      const { data, error: apiError, response } = await client.GET('/v1/users/{userId}', {
        params: { path: { userId } },
      })
      if (response.status === 404) {
        notFound.value = true
        return
      }
      if (apiError || !data?.user) {
        error.value = 'Could not load your profile. Please try again.'
        return
      }
      profile.value = mapToUserProfile(data.user)
    } catch {
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  async function updateLevel(userId: string, actorUserId: string, level: number): Promise<void> {
    saving.value = true
    saveError.value = null
    try {
      const { data, error: apiError, response } = await client.POST(
        '/v1/users/{userId}/selfReportedLevel',
        {
          params: { path: { userId } },
          body: { actorUserId, selfReportedStartingLevel: level },
        },
      )
      if (response.status === 403) {
        saveError.value = "This isn't your profile to edit."
        return
      }
      if (apiError || !data?.user) {
        saveError.value = 'Could not save your starting level. Please try again.'
        return
      }
      profile.value = mapToUserProfile(data.user)
    } catch {
      saveError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      saving.value = false
    }
  }

  return { profile, loading, error, notFound, saving, saveError, load, updateLevel }
}
