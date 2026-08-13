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
 * loading/failed state, request-sequencing guard against a stale response,
 * `client` injectable for tests (defaults to the real `identityClient`).
 *
 * KNOWN DUPLICATION (T10.8 PR review, flagged not fixed): this and
 * `useFacilityName` are near-identical "resolve id -> label" composables
 * over two different clients/response shapes. Left as two copies rather
 * than one generic `useResolvedLabel<TClient, TId>`-shaped composable for
 * now — a genuine future dedupe candidate once a third context needs the
 * same shape (Match/PlayerRating in a later sprint is the likely next
 * candidate), not before, per this codebase's "don't abstract on a guess"
 * convention.
 */
export function useDisplayName(client: IdentityClient = identityClient): UseDisplayNameResult {
  const displayName = ref<string | null>(null)
  const loading = ref(false)
  const failed = ref(false)

  // Request-sequencing guard (T10.8 PR review): DisplayName.vue/VenueName.vue
  // are never remounted when the id they resolve changes (unlike
  // GameJoinPanel.vue/CompetitionEntryPanel.vue, which use `:key="game.id"`/
  // `:key="competition.id"` to force a remount on selection change) — a
  // component instance's `load()` can be called again for a new id while an
  // earlier call for a previous id is still in flight. Without this guard, a
  // slower earlier response landing AFTER a faster later one would silently
  // overwrite the correct, newer state with a stale name belonging to the
  // previously-selected id (e.g. selecting Game A, then quickly Game B: B's
  // fast GetUser resolves first and shows correctly, then A's slow GetUser
  // finally lands and clobbers it with A's host name). `requestId` is bumped
  // on every `load()` call; a response is only applied if its own id still
  // matches the latest at the time it resolves.
  let requestId = 0

  async function load(userId: string): Promise<void> {
    const thisRequestId = ++requestId
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
      // A newer load() call has started since this one began — discard this
      // (now stale) response rather than let it overwrite the newer state.
      if (thisRequestId !== requestId) return
      if (error || !data?.user?.displayName) {
        failed.value = true
        return
      }
      displayName.value = data.user.displayName
    } catch {
      if (thisRequestId !== requestId) return
      failed.value = true
    } finally {
      // Only the latest request gets to clear `loading` — an older
      // request's completion (successful or not) must not mark the newer,
      // still-in-flight request as done.
      if (thisRequestId === requestId) {
        loading.value = false
      }
    }
  }

  return { displayName, loading, failed, load }
}
