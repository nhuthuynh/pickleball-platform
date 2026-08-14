import { ref, computed, type Ref, type ComputedRef } from 'vue'
import { identityClient, type IdentityClient } from '../api/identityClient'
import { mapToUserRoles, hasClubRole, type Role } from '../models/identity'

/**
 * Resolves the CURRENT actor's real Roles from the Identity/Users context
 * (T11.6), so a screen can decide which controls to render for them.
 *
 * It reuses the exact mechanism `useProfile` (T10.5) already established —
 * `identityClient.GET('/v1/users/{userId}')`, the one `GetUser` read, with an
 * injectable client — rather than introducing a second auth-context
 * mechanism. There is still no session/JWT layer (HANDOFF.md's Auth
 * cross-cutting item); the actor id comes from the same single mock identity
 * every other screen uses, kept in one place on purpose.
 *
 * **Three states, deliberately distinguished, because collapsing them is how
 * a screen ends up lying:**
 *   - `resolved` false: we have not been told anything yet (still loading, or
 *     the lookup failed). `isClub` is false — the gate fails CLOSED — but the
 *     screen must NOT say "you are not a club", because that is not what we
 *     learned.
 *   - `resolved` true, `isClub` false: we really did read this user's Roles
 *     and `club` is not among them. That claim can be made in words.
 *   - `resolved` true, `isClub` true: show the Club controls.
 *
 * `error` carries the reason a lookup failed; `notFound` marks "no such user",
 * which today is the ordinary outcome for the mock id (see `useProfile`'s own
 * note on why) and is not an error to shout about.
 */
export interface UseActorRolesResult {
  roles: Ref<Role[]>
  /** True only when a real `GetUser` answer was received — success OR a clean
   * "no such user". False while loading and after a failed lookup. */
  resolved: Ref<boolean>
  loading: Ref<boolean>
  error: Ref<string | null>
  notFound: Ref<boolean>
  /** Fails closed: false unless the actor's real Roles include `club`. */
  isClub: ComputedRef<boolean>
  load: (userId: string) => Promise<void>
}

export function useActorRoles(client: IdentityClient = identityClient): UseActorRolesResult {
  const roles = ref<Role[]>([])
  const resolved = ref(false)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const notFound = ref(false)

  const isClub = computed(() => resolved.value && hasClubRole(roles.value))

  async function load(userId: string): Promise<void> {
    loading.value = true
    error.value = null
    notFound.value = false
    resolved.value = false
    roles.value = []
    try {
      const { data, error: apiError, response } = await client.GET('/v1/users/{userId}', {
        params: { path: { userId } },
      })
      if (response?.status === 404) {
        // A real answer: this id is nobody. Roles stay empty, and `resolved`
        // is true because we genuinely know there is no Club role to find.
        notFound.value = true
        resolved.value = true
        return
      }
      if (apiError || !data?.user) {
        error.value = 'Could not check which roles your account holds. Please try again.'
        return
      }
      roles.value = mapToUserRoles(data.user)
      resolved.value = true
    } catch {
      error.value = 'Could not reach the server to check your account’s roles. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  return { roles, resolved, loading, error, notFound, isClub, load }
}
