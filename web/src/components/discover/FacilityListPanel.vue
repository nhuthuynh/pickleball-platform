<script setup lang="ts">
// Facility list/search view (T7.5 requirement #1). Purely presentational —
// all fetching/state lives in useFacilityList, owned by the parent
// (DiscoverFacilities.vue) — so this component is trivial to unit test
// with plain props.
import type { FacilitySummary } from '../../models/facility'

defineProps<{
  facilities: FacilitySummary[]
  loading: boolean
  error: string | null
  selectedId: string | null
  nameFilter: string
}>()

const emit = defineEmits<{
  'update:nameFilter': [value: string]
  search: []
  select: [id: string]
}>()

function onInput(event: Event) {
  emit('update:nameFilter', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <section class="facility-list" aria-label="Facility search results">
    <form class="facility-list__search" role="search" @submit.prevent="emit('search')">
      <label for="facility-search-input" class="facility-list__search-label">Search facilities by name</label>
      <div class="facility-list__search-row">
        <input
          id="facility-search-input"
          class="facility-list__search-input"
          type="search"
          placeholder="e.g. Riverside Courts"
          :value="nameFilter"
          @input="onInput"
        />
        <button type="submit" class="facility-list__search-submit">Search</button>
      </div>
    </form>

    <!-- Loading state (T7.5 requirement #2) -->
    <p v-if="loading" class="facility-list__status facility-list__status--loading" role="status">
      Loading facilities…
    </p>

    <!-- Error state (API unreachable) -->
    <div v-else-if="error" class="facility-list__status facility-list__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="facility-list__retry" @click="emit('search')">Try again</button>
    </div>

    <!-- Empty state (no facilities yet, or none match the search) -->
    <p v-else-if="facilities.length === 0" class="facility-list__status facility-list__status--empty">
      {{ nameFilter ? `No facilities match "${nameFilter}".` : 'No facilities yet. Check back soon!' }}
    </p>

    <ul v-else class="facility-list__items">
      <li v-for="f in facilities" :key="f.id">
        <button
          type="button"
          class="facility-list__item"
          :class="{ 'facility-list__item--selected': f.id === selectedId }"
          :aria-current="f.id === selectedId ? 'true' : undefined"
          @click="emit('select', f.id)"
        >
          <span class="facility-list__item-name">{{ f.name }}</span>
          <span class="facility-list__item-address">{{ f.address }}</span>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.facility-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  font-family: var(--font-family-ui);
}

.facility-list__search-label {
  display: block;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  margin-bottom: 0.25rem;
}

.facility-list__search-row {
  display: flex;
  gap: 0.5rem;
}

.facility-list__search-input {
  flex: 1;
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
}

.facility-list__search-submit,
.facility-list__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.facility-list__status {
  color: var(--ink-soft);
  font-size: var(--font-size-base);
}

.facility-list__status--error {
  color: var(--ink-warning);
}

.facility-list__items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.facility-list__item {
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

.facility-list__item--selected {
  border-color: var(--court);
  outline: 2px solid var(--court);
  outline-offset: -2px;
}

.facility-list__item-name {
  font-weight: 600;
}

.facility-list__item-address {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

/* Web (>=1280px): multi-column grid of facility cards, per the external
   handoff's Platform Notes ("multi-column dashboards") and T7.5's
   non-functional requirement. iPhone/iPad keep the single-column list
   (paired with the detail panel per DiscoverFacilities.vue's own layout). */
@media (min-width: 1280px) {
  .facility-list__items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 0.75rem;
  }
}
</style>
