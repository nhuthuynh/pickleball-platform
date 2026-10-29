<script setup lang="ts">
// Resolves a Game's `courtIds` into real `facilities.Court.Name`s (#342) —
// the third member of the DisplayName.vue/VenueName.vue family, following
// the same network-calling-child-of-a-presentational-parent pattern; see
// DisplayName.vue's header for why that pattern exists.
//
// It is here because the game detail card resolved a host id to a name and
// a facility id to a name, then printed the court's raw uuid beside them.
//
// Three DIFFERENT empty states, not collapsed into one (T10.8 instruction
// #4's no-fabricated-data rule, applied to which honest message is shown):
//   - the Game has no courts, or no facility to resolve them against ->
//     `emptyLabel`. A real state, not a lookup problem.
//   - a lookup was attempted and told us nothing -> `failedLabel`.
//   - a lookup partly succeeded -> the names it found, with any
//     unresolvable id left AS the id. See useCourtNames for why an
//     unresolved court keeps its id rather than disappearing.
import { watch } from 'vue'
import { useCourtNames } from '../../composables/useCourtNames'
import { facilitiesClient, type FacilitiesClient } from '../../api/facilitiesClient'

const props = withDefaults(
  defineProps<{
    /** The Facility the courts belong to — GetFacility is what carries
     *  their names (T8.2). */
    facilityId: string
    courtIds: readonly string[]
    /** Shown while the lookup is in flight. */
    loadingLabel?: string
    /** Shown when there is nothing to resolve — see file header. */
    emptyLabel?: string
    /** Shown when a lookup was attempted and resolved nothing. */
    failedLabel?: string
    /** Injectable for tests; defaults to the real facilitiesClient. */
    client?: FacilitiesClient
  }>(),
  {
    loadingLabel: 'Loading…',
    emptyLabel: 'Not set',
    failedLabel: 'Unknown court',
    client: () => facilitiesClient,
  },
)

const { names, loading, failed, load } = useCourtNames(props.client)

watch(
  // Both inputs matter: a card can switch to another Game with the same
  // facility and different courts, or the same courts at another facility.
  () => [props.facilityId, props.courtIds.join(',')] as const,
  () => {
    void load(props.facilityId, props.courtIds)
  },
  { immediate: true },
)
</script>

<template>
  <!-- aria-live: this text changes asynchronously with no user action in
       between, the same reasoning as DisplayName.vue's and VenueName.vue's
       identical attribute. -->
  <span class="court-names" data-testid="court-names" aria-live="polite">
    <template v-if="!facilityId || courtIds.length === 0">{{ emptyLabel }}</template>
    <template v-else-if="loading">{{ loadingLabel }}</template>
    <template v-else-if="failed || names === null">{{ failedLabel }}</template>
    <template v-else>{{ names.join(', ') }}</template>
  </span>
</template>
