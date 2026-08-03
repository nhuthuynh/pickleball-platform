<script setup lang="ts">
// Facility detail view (T7.5 requirement #1): description, photos,
// address, and its courts list. Purely presentational, driven entirely by
// props — see FacilityListPanel.vue's header comment for why.
//
// Camera links: `facility` is typed as `FacilityDetail`
// (models/facility.ts), which structurally has no field that could hold
// camera-link data — there is nothing here to accidentally render even if
// a future change to this file tried to. See models/facility.ts's header
// comment and __tests__/FacilityDetailPanel.spec.ts /
// discover/__tests__/DiscoverFacilities.spec.ts for the enforcing tests.
import type { FacilityDetail } from '../../models/facility'

defineProps<{
  facility: FacilityDetail | null
  loading: boolean
  error: string | null
  /** Whether a facility is currently selected at all (vs. no selection yet — iPad/web only; iPhone never renders this component without a selection). */
  hasSelection: boolean
}>()

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <section class="facility-detail" aria-label="Facility details">
    <p v-if="!hasSelection" class="facility-detail__status facility-detail__status--placeholder">
      Select a facility to see its details.
    </p>

    <!-- Loading state (T7.5 requirement #2) -->
    <p v-else-if="loading" class="facility-detail__status facility-detail__status--loading" role="status">
      Loading facility details…
    </p>

    <!-- Error state (API unreachable) -->
    <div v-else-if="error" class="facility-detail__status facility-detail__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="facility-detail__retry" @click="emit('retry')">Try again</button>
    </div>

    <div v-else-if="facility" class="facility-detail__content">
      <h2 class="facility-detail__name">{{ facility.name }}</h2>
      <p class="facility-detail__address">{{ facility.address }}</p>
      <p class="facility-detail__description">{{ facility.description }}</p>

      <div v-if="facility.photoUrls.length > 0" class="facility-detail__photos">
        <img
          v-for="url in facility.photoUrls"
          :key="url"
          :src="url"
          :alt="`Photo of ${facility.name}`"
          class="facility-detail__photo"
        />
      </div>
      <p v-else class="facility-detail__no-photos">No photos yet.</p>

      <h3 class="facility-detail__courts-heading">Courts</h3>
      <!-- Empty state: a facility with zero courts (T7.5 requirement #2) -->
      <p v-if="facility.courts.length === 0" class="facility-detail__status facility-detail__status--empty">
        This facility hasn't listed any courts yet.
      </p>
      <ul v-else class="facility-detail__courts">
        <li v-for="court in facility.courts" :key="court.id" class="facility-detail__court">
          {{ court.name }}
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.facility-detail {
  font-family: var(--font-family-ui);
  color: var(--ink);
}

.facility-detail__status {
  color: var(--ink-soft);
}

.facility-detail__status--error {
  color: var(--ink-warning);
}

.facility-detail__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.facility-detail__name {
  font-size: var(--font-size-2xl);
  margin: 0 0 0.25rem;
  color: var(--court);
}

.facility-detail__address {
  color: var(--ink-soft);
  margin: 0 0 0.75rem;
}

.facility-detail__description {
  margin: 0 0 1rem;
}

.facility-detail__photos {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.facility-detail__photo {
  width: 100%;
  height: 120px;
  object-fit: cover;
  border-radius: var(--radius-md);
}

.facility-detail__no-photos {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
  margin-bottom: 1rem;
}

.facility-detail__courts-heading {
  font-size: var(--font-size-lg);
  margin: 0 0 0.5rem;
}

.facility-detail__courts {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.facility-detail__court {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}
</style>
