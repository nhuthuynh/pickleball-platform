import { ref, type Ref } from 'vue'
import { identityClient, type IdentityClient } from '../api/identityClient'

export interface UseDisplayNameResult {
  displayName: Ref<string | null>
  loading: Ref<boolean>
  /**
   * True once a lookup has been attempted and did not produce a real
   * `DisplayName` — an empty `userId`, an unknown/deleted User (`GetUser`
   * returns `NotFound`), or any other lookup failure (e.g. network error).
   * T10.8 instruction #4: the caller must degrade to a named empty state
   * ("Unknown host") on this, never a placeholder implying real data was
   * found. A User's `DisplayName` is required non-empty at construction
   * (T10.1), so this flag is never set for a User that genuinely exists —
   * it only ever means the lookup itself didn't resolve.
   */
  failed: Ref<boolean>
  load: (userId: string) => Promise<void>
}

/**
 * Resolves one `identity.User`'s `DisplayName` by id (T10.8, closes #98).
 * Mirrors `useFacilityName`'s shape exactly — one lookup, tracked
 * loading/failed state, `client` injectable for tests (defaults to the
 * real `identityClient`).
 */
export function useDisplayName(client: IdentityClient = identityClient): UseDisplayNameResult {
  const displayName = ref<string | null>(null)
  const loading = ref(false)
  const failed = ref(false)

  async function load(userId: string): Promise<void> {
    displayName.value = null
    failed.value = false
    loading.value = false

    // An empty id is a caller bug, not a reachable User — no request to
    // make, and no real name to show.
    if (!userId) {
      failed.value = true
      return
    }

    loading.value = true
    try {
      const { data, error } = await client.GET('/v1/users/{userId}', {
        params: { path: { userId } },
      })
      if (error || !data?.user?.displayName) {
        failed.value = true
        return
      }
      displayName.value = data.user.displayName
    } catch {
      failed.value = true
    } finally {
      loading.value = false
    }
  }

  return { displayName, loading, failed, load }
}
