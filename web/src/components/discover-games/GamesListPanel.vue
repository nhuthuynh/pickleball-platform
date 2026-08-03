<script setup lang="ts">
// Games list/search view (T8.9 requirement #1). Purely presentational —
// all fetching/state lives in useGameList, owned by the parent
// (DiscoverGames.vue) — mirrors FacilityListPanel.vue's split.
import type { GameSummary } from '../../models/game'
import { formatGameRange, spotsLeftLabel } from '../../models/game'

defineProps<{
  games: GameSummary[]
  loading: boolean
  error: string | null
  selectedId: string | null
  facilityFilter: string
  dateFilter: string
}>()

const emit = defineEmits<{
  'update:facilityFilter': [value: string]
  'update:dateFilter': [value: string]
  search: []
  select: [id: string]
}>()

function onFacilityInput(event: Event): void {
  emit('update:facilityFilter', (event.target as HTMLInputElement).value)
}

function onDateInput(event: Event): void {
  emit('update:dateFilter', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <section class="game-list" aria-label="Game search results">
    <form class="game-list__search" role="search" @submit.prevent="emit('search')">
      <div class="game-list__search-field">
        <label for="game-facility-filter">Facility ID</label>
        <input
          id="game-facility-filter"
          class="game-list__search-input"
          type="text"
          placeholder="Filter by facility ID"
          :value="facilityFilter"
          @input="onFacilityInput"
        />
      </div>
      <div class="game-list__search-field">
        <label for="game-date-filter">Date</label>
        <input
          id="game-date-filter"
          class="game-list__search-input"
          type="date"
          :value="dateFilter"
          @input="onDateInput"
        />
      </div>
      <button type="submit" class="game-list__search-submit">Search</button>
    </form>

    <!-- Loading state (T8.9 requirement #3) -->
    <p v-if="loading" class="game-list__status game-list__status--loading" role="status">Loading games…</p>

    <!-- Error state (API unreachable) -->
    <div v-else-if="error" class="game-list__status game-list__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="game-list__retry" @click="emit('search')">Try again</button>
    </div>

    <!-- Empty state (no games yet, or none match the filters — including
         "a facility with zero games", T8.9 requirement #3) -->
    <p v-else-if="games.length === 0" class="game-list__status game-list__status--empty">
      {{
        facilityFilter || dateFilter
          ? 'No games match these filters.'
          : 'No games yet. Check back soon!'
      }}
    </p>

    <ul v-else class="game-list__items">
      <li v-for="g in games" :key="g.id">
        <button
          type="button"
          class="game-list__item"
          :class="{ 'game-list__item--selected': g.id === selectedId }"
          :aria-current="g.id === selectedId ? 'true' : undefined"
          @click="emit('select', g.id)"
        >
          <span class="game-list__item-time">{{ formatGameRange(g.startsAt, g.endsAt) }}</span>
          <!-- WCAG 1.4.1: spots-left is always rendered as text, never a
               color-only cue -->
          <span class="game-list__item-spots">{{ spotsLeftLabel(g.spotsLeft) }}</span>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.game-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  font-family: var(--font-family-ui);
}

.game-list__search {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: end;
}

.game-list__search-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  flex: 1;
  min-width: 140px;
}

.game-list__search-input {
  flex: 1;
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
}

.game-list__search-submit,
.game-list__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.game-list__status {
  color: var(--ink-soft);
  font-size: var(--font-size-base);
}

.game-list__status--error {
  color: var(--ink-warning);
}

.game-list__items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.game-list__item {
  width: 100%;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font: inherit;
  min-height: 44px;
  padding: 0.6rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper-raised);
  color: var(--ink);
  cursor: pointer;
}

.game-list__item--selected {
  border-color: var(--court);
  outline: 2px solid var(--court);
  outline-offset: -2px;
}

.game-list__item-time {
  font-weight: 600;
}

.game-list__item-spots {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

/* Web (>=1280px): multi-column grid, mirroring FacilityListPanel.vue's
   identical breakpoint treatment. */
@media (min-width: 1280px) {
  .game-list__items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 0.75rem;
  }
}
</style>
