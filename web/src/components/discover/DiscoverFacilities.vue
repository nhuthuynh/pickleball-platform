<script setup lang="ts">
// Discover & browse facilities/courts screen (T7.5,
// docs/process/t7-sprint-plan.md). Player-facing, read-only: a facility
// list/search view (ListFacilities) plus a facility detail view
// (GetFacility) showing description, photos, address, and courts.
//
// Camera links are never shown here — see models/facility.ts's file
// header for how that's enforced (the view models this screen renders
// structurally have no field that could hold camera-link data), and
// __tests__/DiscoverFacilities.spec.ts for the end-to-end regression test
// using a mocked API response that includes camera-link data.
//
// Responsive layout (T7.5 non-functional requirement, matching the
// external handoff's Platform Notes / src/styles/breakpoints.css):
//   - iPhone (<600px): single-column, stacked. The detail panel is only
//     rendered once a facility is selected (a persistently-visible empty
//     "select a facility" placeholder column doesn't make sense in a
//     single, already-scrollable column).
//   - iPad (768-1180px) / web (>=1280px): two-column list+detail, always
//     both visible (detail shows a placeholder before any selection).
//     Web additionally renders the facility list itself as a multi-column
//     grid of cards — see FacilityListPanel.vue's own `@media` block.
import { ref, onMounted } from 'vue'
import { useBreakpoint } from '../../composables/useBreakpoint'
import { useFacilityList } from '../../composables/useFacilityList'
import { useFacilityDetail } from '../../composables/useFacilityDetail'
import type { FacilitiesClient } from '../../api/facilitiesClient'
import type { BookingClient } from '../../api/bookingClient'
import FacilityListPanel from './FacilityListPanel.vue'
import FacilityDetailPanel from './FacilityDetailPanel.vue'
import CourtBookingFlow from '../booking/CourtBookingFlow.vue'

const props = defineProps<{
  /** Injectable for tests; defaults to the real facilitiesClient. */
  client?: FacilitiesClient
  /** Injectable for tests; defaults to the real bookingClient. Passed
   * through to CourtBookingFlow (T7.6). */
  bookingClient?: BookingClient
  /** Injectable for tests; defaults to the ambient `window` (matchMedia). */
  win?: Pick<Window, 'matchMedia'>
}>()

const { breakpoint, isIphone } = useBreakpoint(props.win ?? window)

const { facilities, loading: listLoading, error: listError, nameFilter, search } = useFacilityList(props.client)
const { facility, loading: detailLoading, error: detailError, load } = useFacilityDetail(props.client)

const selectedId = ref<string | null>(null)

let debounceTimer: ReturnType<typeof setTimeout> | undefined

function onNameFilterInput(value: string) {
  nameFilter.value = value
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    void search()
  }, 300)
}

function onSearchSubmit() {
  if (debounceTimer) clearTimeout(debounceTimer)
  void search()
}

function onSelect(id: string) {
  selectedId.value = id
  void load(id)
}

function onDetailRetry() {
  if (selectedId.value) void load(selectedId.value)
}

// "Book a court" entry point (T7.6): FacilityDetailPanel only emits a
// courtId (+ optional courtName); this screen owns whether/which booking
// flow is shown, same split as everywhere else on this screen (panels are
// presentational, DiscoverFacilities owns the state).
const bookingCourtId = ref<string | null>(null)
const bookingCourtName = ref<string | undefined>(undefined)

function onBookCourt(courtId: string, courtName?: string) {
  bookingCourtId.value = courtId
  bookingCourtName.value = courtName
}

function onCloseBooking() {
  bookingCourtId.value = null
  bookingCourtName.value = undefined
}

onMounted(() => {
  void search()
})
</script>

<template>
  <div class="discover-page">
    <section class="discover" :data-breakpoint="breakpoint">
      <FacilityListPanel
        class="discover__list"
        :facilities="facilities"
        :loading="listLoading"
        :error="listError"
        :selected-id="selectedId"
        :name-filter="nameFilter"
        @update:name-filter="onNameFilterInput"
        @search="onSearchSubmit"
        @select="onSelect"
      />
      <FacilityDetailPanel
        v-if="selectedId !== null || !isIphone"
        class="discover__detail"
        :facility="facility"
        :loading="detailLoading"
        :error="detailError"
        :has-selection="selectedId !== null"
        @retry="onDetailRetry"
        @book-court="onBookCourt"
      />
    </section>

    <!-- Quote-and-book flow (T7.6). Rendered below the browse layout rather
         than replacing it — no routing library exists yet (T7.1's own
         scope note), so this is the simplest way to keep "browse" and
         "book" both reachable without a real navigation stack. -->
    <CourtBookingFlow
      v-if="bookingCourtId"
      class="discover__booking"
      :court-id="bookingCourtId"
      :court-name="bookingCourtName"
      :client="bookingClient"
      @close="onCloseBooking"
    />
  </div>
</template>

<style scoped>
.discover-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.discover {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* iPad: two-column list+detail. */
@media (min-width: 768px) and (max-width: 1180px) {
  .discover {
    display: grid;
    grid-template-columns: minmax(260px, 340px) 1fr;
    align-items: start;
    gap: 1.5rem;
  }
}

/* Web: two-column list+detail, wider — the list panel's own contents
   additionally become a multi-column grid at this breakpoint (see
   FacilityListPanel.vue). */
@media (min-width: 1280px) {
  .discover {
    display: grid;
    grid-template-columns: minmax(320px, 1fr) 2fr;
    align-items: start;
    gap: 2rem;
  }
}
</style>
