<script setup lang="ts">
// Competitions list/search view (T9.7 requirement #1). Purely
// presentational — all fetching/state lives in useCompetitionList, owned by
// the parent (DiscoverCompetitions.vue) — mirrors GamesListPanel.vue's split
// exactly, including its loading/error/empty idiom.
//
// The loading idiom here is T8.9's: a `role="status"` text line, not a
// spinner and not a shimmering skeleton. T8.9 established exactly one
// loading treatment across the Discover surfaces, and the ticket's own
// instruction is to match it rather than introduce a third idiom — a
// skeleton on Competitions alone would make the two sibling browse screens
// visibly inconsistent. See the PR description.
import type { CompetitionSummary } from '../../models/competition'
import { earliestSessionStart, formatCompetitionDate, spotsLeftLabel } from '../../models/competition'

defineProps<{
  competitions: CompetitionSummary[]
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

function sessionCountLabel(competition: CompetitionSummary): string {
  const n = competition.sessions.length
  return n === 1 ? '1 session' : `${n} sessions`
}
</script>

<template>
  <section class="competition-list" aria-label="Competition search results">
    <form class="competition-list__search" role="search" @submit.prevent="emit('search')">
      <div class="competition-list__search-field">
        <label for="competition-facility-filter">Facility ID</label>
        <input
          id="competition-facility-filter"
          class="competition-list__search-input"
          type="text"
          placeholder="Filter by facility ID"
          :value="facilityFilter"
          @input="onFacilityInput"
        />
      </div>
      <div class="competition-list__search-field">
        <label for="competition-date-filter">Date</label>
        <input
          id="competition-date-filter"
          class="competition-list__search-input"
          type="date"
          :value="dateFilter"
          @input="onDateInput"
        />
      </div>
      <button type="submit" class="competition-list__search-submit">Search</button>
    </form>

    <!-- Loading -->
    <p v-if="loading" class="competition-list__status competition-list__status--loading" role="status">
      Loading competitions…
    </p>

    <!-- Error (API unreachable / non-2xx) -->
    <div v-else-if="error" class="competition-list__status competition-list__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="competition-list__retry" @click="emit('search')">Try again</button>
    </div>

    <!-- Empty. "Nothing scheduled" and "nothing matches your filters" are
         different facts and get different words — a single generic empty
         string would leave a player unsure whether to clear the filters. -->
    <p v-else-if="competitions.length === 0" class="competition-list__status competition-list__status--empty">
      {{
        facilityFilter || dateFilter
          ? 'No competitions match these filters. Try a different facility or date.'
          : 'No competitions scheduled yet. Check back soon!'
      }}
    </p>

    <ul v-else class="competition-list__items">
      <li v-for="c in competitions" :key="c.id">
        <button
          type="button"
          class="competition-list__item"
          :class="{ 'competition-list__item--selected': c.id === selectedId }"
          :aria-current="c.id === selectedId ? 'true' : undefined"
          @click="emit('select', c.id)"
        >
          <span class="competition-list__item-name">{{ c.name || 'Untitled competition' }}</span>
          <span class="competition-list__item-time">
            {{ formatCompetitionDate(earliestSessionStart(c.sessions)) }} · {{ sessionCountLabel(c) }}
          </span>
          <!-- WCAG 1.4.1: spots-left urgency is words, never a coloured
               number on its own. -->
          <span class="competition-list__item-spots">{{ spotsLeftLabel(c.spotsLeft ?? 0) }}</span>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.competition-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  font-family: var(--font-family-ui);
}

.competition-list__search {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: end;
}

.competition-list__search-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  flex: 1;
  min-width: 140px;
}

.competition-list__search-input {
  flex: 1;
  font: inherit;
  /* >=44px touch target on iPad/iPhone. */
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
}

.competition-list__search-submit,
.competition-list__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.competition-list__status {
  color: var(--ink-soft);
  font-size: var(--font-size-base);
}

.competition-list__status--error {
  color: var(--ink-warning);
}

.competition-list__items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.competition-list__item {
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

.competition-list__item--selected {
  border-color: var(--court);
  outline: 2px solid var(--court);
  outline-offset: -2px;
}

/* WCAG 2.4.7: a visible focus indicator that does not rely on the UA
   default, which the selected-state outline above would otherwise mask. */
.competition-list__item:focus-visible,
.competition-list__search-input:focus-visible,
.competition-list__search-submit:focus-visible,
.competition-list__retry:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 2px;
}

.competition-list__item-name {
  font-weight: 600;
}

.competition-list__item-time,
.competition-list__item-spots {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

/* Web (>=1280px): multi-column grid, mirroring GamesListPanel.vue's
   identical breakpoint treatment. */
@media (min-width: 1280px) {
  .competition-list__items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 0.75rem;
  }
}
</style>
