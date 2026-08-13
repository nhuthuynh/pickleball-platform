<script setup lang="ts">
// Discover & Join Games screen (T8.9, docs/process/t8-sprint-plan.md),
// mounted at /games (replacing T8.1's placeholder route). Mirrors
// DiscoverFacilities.vue's list+detail split and responsive layout
// exactly: GamesListPanel/GameDetailPanel are presentational, this
// component owns the fetch state (useGameList) and which Game is
// selected.
//
// Unlike DiscoverFacilities (which fetches a separate GetFacility for
// detail data), the selected Game's detail is derived from the
// already-fetched `games` list — ListGames already returns everything
// GameDetailPanel needs (see that file's header comment) — no second
// network round trip per selection.
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBreakpoint } from '../../composables/useBreakpoint'
import { useGameList } from '../../composables/useGameList'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { IdentityClient } from '../../api/identityClient'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import GamesListPanel from './GamesListPanel.vue'
import GameDetailPanel from './GameDetailPanel.vue'

const props = defineProps<{
  /** Injectable for tests; defaults to the real socialplayClient. */
  client?: SocialPlayClient
  /** Injectable for tests (T10.8); defaults to the real identityClient.
   * Forwarded to GameDetailPanel to resolve the Host's display name. */
  identityClient?: IdentityClient
  /** Injectable for tests (T10.8); defaults to the real facilitiesClient.
   * Forwarded to GameDetailPanel to resolve the venue's name. */
  facilitiesClient?: FacilitiesClient
  /** Injectable for tests; defaults to the ambient `window` (matchMedia). */
  win?: Pick<Window, 'matchMedia'>
}>()

const { breakpoint, isIphone } = useBreakpoint(props.win ?? window)

// Optional: `useRouter()` returns `undefined` when no router plugin is
// installed (rather than throwing) — safe to call unconditionally even in
// component tests that mount this screen standalone, same as every other
// composable call here; `onPayOnline` below only ever runs it if a Player
// actually clicks "Pay online now" (T8.10).
const router = useRouter()

const { games, loading, error, facilityFilter, dateFilter, search } = useGameList(props.client)

const selectedId = ref<string | null>(null)

const selectedGame = computed(() => games.value.find((g) => g.id === selectedId.value) ?? null)

// A stale selection (the selected id no longer resolves to a game in the
// freshest fetch — e.g. it was cancelled/removed) gets its own error
// message rather than silently rendering nothing, per T8.9's detail-view
// error-state requirement.
const detailError = computed(() => {
  if (error.value) return error.value
  if (selectedId.value !== null && !selectedGame.value && !loading.value) {
    return 'Could not find this game. It may have been cancelled or removed.'
  }
  return null
})

function onFacilityFilterInput(value: string) {
  facilityFilter.value = value
}

function onDateFilterInput(value: string) {
  dateFilter.value = value
}

function onSearch() {
  void search()
}

function onSelect(id: string) {
  selectedId.value = id
}

function onDetailRetry() {
  void search()
}

// T8.10: navigates to the checkout route, carrying the Registration id as
// a query param — GameCheckout.vue reads it via `route.query.registrationId`
// (the route's own `:id` path param is the Game id, per router/index.ts).
function onPayOnline(registrationId: string) {
  if (!selectedId.value) return
  void router?.push({ name: 'game-checkout', params: { id: selectedId.value }, query: { registrationId } })
}

onMounted(() => {
  void search()
})
</script>

<template>
  <div class="discover-games-page">
    <section class="discover-games" :data-breakpoint="breakpoint">
      <GamesListPanel
        class="discover-games__list"
        :games="games"
        :loading="loading"
        :error="error"
        :selected-id="selectedId"
        :facility-filter="facilityFilter"
        :date-filter="dateFilter"
        @update:facility-filter="onFacilityFilterInput"
        @update:date-filter="onDateFilterInput"
        @search="onSearch"
        @select="onSelect"
      />
      <GameDetailPanel
        v-if="selectedId !== null || !isIphone"
        class="discover-games__detail"
        :game="selectedGame"
        :loading="loading && selectedId !== null"
        :error="detailError"
        :has-selection="selectedId !== null"
        :client="client"
        :identity-client="identityClient"
        :facilities-client="facilitiesClient"
        @retry="onDetailRetry"
        @pay-online="onPayOnline"
      />
    </section>
  </div>
</template>

<style scoped>
.discover-games-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.discover-games {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* iPad: two-column list+detail, mirroring DiscoverFacilities.vue. */
@media (min-width: 768px) and (max-width: 1180px) {
  .discover-games {
    display: grid;
    grid-template-columns: minmax(260px, 340px) 1fr;
    align-items: start;
    gap: 1.5rem;
  }
}

/* Web: two-column list+detail, wider. */
@media (min-width: 1280px) {
  .discover-games {
    display: grid;
    grid-template-columns: minmax(320px, 1fr) 2fr;
    align-items: start;
    gap: 2rem;
  }
}
</style>
