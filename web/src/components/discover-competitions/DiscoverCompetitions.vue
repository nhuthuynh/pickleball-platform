<script setup lang="ts">
// Discover & Enter Competitions screen (T9.7), mounted at /competitions and
// /competitions/:id. Mirrors DiscoverGames.vue's list+detail split and
// responsive layout exactly: CompetitionsListPanel/CompetitionDetailPanel
// are presentational, this component owns the fetch state
// (useCompetitionList) and which Competition is selected.
//
// As in DiscoverGames, the selected Competition's detail is derived from the
// already-fetched list rather than a second round trip — and here that is
// not merely an optimisation: `ListCompetitions` is the ONLY read that
// carries the server-computed weighted `spots_left`, which the detail view
// needs. `GetCompetition` returns a bare Competition without it.
//
// `competitionId` arrives as a route prop (router/index.ts maps `:id` onto
// it) so this component needs no router to be mounted in a test, matching
// the injectable-client convention every other screen here follows.
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBreakpoint } from '../../composables/useBreakpoint'
import { useCompetitionList } from '../../composables/useCompetitionList'
import type { CompetitionsClient } from '../../api/competitionsClient'
import type { IdentityClient } from '../../api/identityClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import CompetitionsListPanel from './CompetitionsListPanel.vue'
import CompetitionDetailPanel from './CompetitionDetailPanel.vue'

const props = defineProps<{
  /** Injectable for tests; defaults to the real competitionsClient. */
  client?: CompetitionsClient
  /** Injectable for tests (T10.8); defaults to the real identityClient.
   * Forwarded to CompetitionDetailPanel to resolve the Host's display name. */
  identityClient?: IdentityClient
  /** Injectable for tests (T10.8); defaults to the real facilitiesClient.
   * Forwarded to CompetitionDetailPanel to resolve the venue's name. */
  facilitiesClient?: FacilitiesClient
  /** Injectable for tests; defaults to the ambient `window` (matchMedia). */
  win?: Pick<Window, 'matchMedia'>
  /** From the `/competitions/:id` route, so a link to one competition opens
   * straight onto it instead of an unselected list. */
  competitionId?: string
}>()

const { breakpoint, isIphone } = useBreakpoint(props.win ?? window)

// `useRouter()` returns undefined when no router plugin is installed (rather
// than throwing), so this is safe to call in a standalone component test —
// the same pattern DiscoverGames.vue uses.
const router = useRouter()

const { competitions, loading, error, facilityFilter, dateFilter, search } = useCompetitionList(props.client)

const selectedId = ref<string | null>(props.competitionId ?? null)

watch(
  () => props.competitionId,
  (id) => {
    selectedId.value = id ?? null
  },
)

const selectedCompetition = computed(
  () => competitions.value.find((c) => c.id === selectedId.value) ?? null,
)

// A stale selection (the id no longer resolves in the freshest fetch — it
// was cancelled, or the link is old) gets its own message rather than
// silently rendering nothing. Note that cancelled Competitions are excluded
// from ListCompetitions by the backend, so "cancelled" is a real reason a
// /competitions/:id link can land here.
const detailError = computed(() => {
  if (error.value) return error.value
  if (selectedId.value !== null && !selectedCompetition.value && !loading.value) {
    return 'Could not find this competition. It may have been cancelled, or it may no longer be listed.'
  }
  return null
})

function onSelect(id: string): void {
  selectedId.value = id
  // Keep the URL addressable, so a player can share or bookmark what they're
  // looking at. Harmless when no router is installed (component tests).
  void router?.push({ name: 'competition-detail', params: { id } })
}

// T10.6 (closes #96): navigates to the checkout route, carrying the
// CompetitionEntry id as a query param — CompetitionCheckout.vue reads it
// via `route.query.entryId` (the route's own `:id` path param is the
// Competition id, per router/index.ts). Mirrors DiscoverGames.vue's
// `onPayOnline` exactly.
function onPayOnline(entryId: string): void {
  if (!selectedId.value) return
  void router?.push({ name: 'competition-checkout', params: { id: selectedId.value }, query: { entryId } })
}

onMounted(() => {
  void search()
})
</script>

<template>
  <div class="discover-competitions-page">
    <!-- T11.7 fix (WCAG 2.4.6/2.4.10, axe page-has-heading-one): this screen
         had no page-level heading at all — same static text for both
         `/competitions` and `/competitions/:id`, since a selected
         competition doesn't change what the SCREEN itself is. -->
    <h1 class="discover-competitions__heading">Competitions</h1>
    <section class="discover-competitions" :data-breakpoint="breakpoint">
      <CompetitionsListPanel
        class="discover-competitions__list"
        :competitions="competitions"
        :loading="loading"
        :error="error"
        :selected-id="selectedId"
        :facility-filter="facilityFilter"
        :date-filter="dateFilter"
        @update:facility-filter="(v: string) => (facilityFilter = v)"
        @update:date-filter="(v: string) => (dateFilter = v)"
        @search="search"
        @select="onSelect"
      />
      <CompetitionDetailPanel
        v-if="selectedId !== null || !isIphone"
        class="discover-competitions__detail"
        :competition="selectedCompetition"
        :loading="loading && selectedId !== null"
        :error="detailError"
        :has-selection="selectedId !== null"
        source="ENTRY_SOURCE_APP"
        :client="client"
        :identity-client="identityClient"
        :facilities-client="facilitiesClient"
        @retry="search"
        @pay-online="onPayOnline"
      />
    </section>
  </div>
</template>

<style scoped>
.discover-competitions-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.discover-competitions__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

/* iPhone (default): a single stacked column. */
.discover-competitions {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* iPad: two-column list+detail, mirroring DiscoverGames.vue. */
@media (min-width: 768px) and (max-width: 1180px) {
  .discover-competitions {
    display: grid;
    grid-template-columns: minmax(260px, 340px) 1fr;
    align-items: start;
    gap: 1.5rem;
  }
}

/* Web: two-column list+detail, wider. */
@media (min-width: 1280px) {
  .discover-competitions {
    display: grid;
    grid-template-columns: minmax(320px, 1fr) 2fr;
    align-items: start;
    gap: 2rem;
  }
}
</style>
