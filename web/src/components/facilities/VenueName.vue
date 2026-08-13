<script setup lang="ts">
// Resolves a `venueFacilityId` into a real `facilities.Facility.Name`
// (T10.8, closes #98) — client-side composition against the existing
// `GetFacility` read (see this ticket's PR description for why: no new
// server-side field, `GetFacility` already returns `Name`, T8.2). Small,
// network-calling, and reusable across Games and Competitions — mirrors
// DisplayName.vue's identical role for the Identity half of this same gap;
// see that file's header comment for the network-calling-child-of-a-
// presentational-parent pattern both follow.
//
// Two DIFFERENT empty states, not one collapsed into the other (T10.8
// instruction #4's "no fabricated data" rule, applied to which honest
// message is shown, not just to withholding a fake name):
//   - no id was ever set on the Game/Competition -> `emptyLabel` (a real,
//     Host-chosen state — the venue genuinely isn't decided yet).
//   - an id was set but the lookup failed (unknown/deleted Facility,
//     network error) -> `failedLabel` (a lookup problem, a different fact
//     than "no venue").
import { watch } from 'vue'
import { useFacilityName } from '../../composables/useFacilityName'
import { facilitiesClient, type FacilitiesClient } from '../../api/facilitiesClient'

const props = withDefaults(
  defineProps<{
    facilityId: string
    /** Shown while the lookup is in flight. */
    loadingLabel?: string
    /** Shown when `facilityId` is empty — see file header. */
    emptyLabel?: string
    /** Shown when a real `facilityId` was set but the lookup failed — see
     * file header. */
    failedLabel?: string
    /** Injectable for tests; defaults to the real facilitiesClient. */
    client?: FacilitiesClient
  }>(),
  {
    loadingLabel: 'Loading…',
    emptyLabel: 'Not set',
    failedLabel: 'Unknown venue',
    client: () => facilitiesClient,
  },
)

const { name, loading, failed, load } = useFacilityName(props.client)

watch(
  () => props.facilityId,
  (id) => {
    void load(id)
  },
  { immediate: true },
)
</script>

<template>
  <!-- aria-live (T10.8 PR review): same reasoning as DisplayName.vue's
       identical addition — this text changes asynchronously with no user
       action in between, mirroring GameJoinPanel.vue's guest-count span. -->
  <span class="venue-name" data-testid="venue-name" aria-live="polite">
    <template v-if="!facilityId">{{ emptyLabel }}</template>
    <template v-else-if="loading">{{ loadingLabel }}</template>
    <template v-else-if="failed || !name">{{ failedLabel }}</template>
    <template v-else>{{ name }}</template>
  </span>
</template>
