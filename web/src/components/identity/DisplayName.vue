<script setup lang="ts">
// Resolves a `hostId`/`playerId`/etc. into a real `identity.User.DisplayName`
// (T10.8, closes #98). Small, network-calling, and reusable — the same
// pattern GameJoinPanel.vue/CompetitionEntryPanel.vue already established
// for a network-calling child inside an otherwise-presentational parent
// (see those files' header comments), applied here instead of making
// GameDetailPanel.vue/CompetitionDetailPanel.vue/CompetitionManage.vue do
// their own Identity fetches and stop being purely presentational.
//
// Used for BOTH Games and Competitions, and for both a single host field
// and a roster of players — one component, not a per-screen copy, per
// CLAUDE.md rule 7 (one ubiquitous language) and T10.8 instruction #3
// ("apply consistently across both Games and Competitions").
//
// No fabricated data (T10.8 instruction #4): an empty `userId`, an
// unknown/deleted User, or any other lookup failure all render `fallback`
// — never a placeholder that could be mistaken for a real name. A
// `DisplayName` is required non-empty at construction (T10.1), so this
// path is only ever reached by a genuine lookup failure, not "the User
// exists but chose not to set one."
import { watch } from 'vue'
import { useDisplayName } from '../../composables/useDisplayName'
import { identityClient, type IdentityClient } from '../../api/identityClient'

const props = withDefaults(
  defineProps<{
    userId: string
    /** Shown while the lookup is in flight. */
    loadingLabel?: string
    /** Shown for an empty id or any lookup failure — callers pass a
     * context-specific word ("Unknown host", "Unknown player") rather than
     * a generic one, so the empty state still means something. */
    fallback?: string
    /** Injectable for tests; defaults to the real identityClient. */
    client?: IdentityClient
  }>(),
  { loadingLabel: 'Loading…', fallback: 'Unknown user', client: () => identityClient },
)

const { displayName, loading, failed, load } = useDisplayName(props.client)

watch(
  () => props.userId,
  (id) => {
    void load(id)
  },
  { immediate: true },
)
</script>

<template>
  <span class="display-name" data-testid="display-name">
    <template v-if="loading">{{ loadingLabel }}</template>
    <template v-else-if="failed || !displayName">{{ fallback }}</template>
    <template v-else>{{ displayName }}</template>
  </span>
</template>
