<script setup lang="ts">
// Quote-and-book flow (T7.6, docs/process/t7-sprint-plan.md): a date/time
// picker that calls GetQuote, a review/confirm step (WCAG 3.3.4 Error
// Prevention — the only error-prevention mechanism this flow has since T7
// has no payment step yet), and a booking-success confirmation exposed via
// an ARIA live region (WCAG 4.1.3). All state/network logic lives in
// useCourtBooking (this component is the presentational half), same split
// as FacilityListPanel/FacilityDetailPanel vs. useFacilityList/useFacilityDetail.
//
// `courtId` is accepted directly as a prop rather than resolved from a
// Facility's courts list — the merged Facilities API has no courts-listing
// endpoint yet (see web/README.md's "Known gap" note and
// FacilityDetailPanel.vue's booking-entry section), so the caller supplies
// a courtId however it currently can (a manually-entered id today, a real
// court selection once that gap is closed — this component doesn't care
// which).
import { ref, computed } from 'vue'
import { useCourtBooking } from '../../composables/useCourtBooking'
import { formatPriceCents, type BookingSlot } from '../../models/booking'
import type { BookingClient } from '../../api/bookingClient'

const props = defineProps<{
  courtId: string
  courtName?: string
  /** Injectable for tests; defaults to the real bookingClient. */
  client?: BookingClient
}>()

const emit = defineEmits<{
  close: []
}>()

const {
  step,
  quote,
  quoteLoading,
  quoteError,
  bookingLoading,
  bookingError,
  conflict,
  confirmedBooking,
  quoteRefreshedNotice,
  getQuote,
  confirmBooking,
  backToForm,
  reset,
} = useCourtBooking(props.client)

const date = ref('')
const startTime = ref('')
const durationMinutes = ref(60)

/** The slot last sent to GetQuote, for display on the review step — kept
 * here (not in the composable) since it's purely presentational: the
 * composable only needs it internally for the stale-quote re-fetch and
 * conflict-suggestion search. */
const reviewRange = ref<{ startsAt: string; endsAt: string } | null>(null)

const courtLabel = computed(() => props.courtName ?? `Court ${props.courtId}`)
const canGetQuote = computed(() => date.value !== '' && startTime.value !== '')

function rangeFromForm(): { startsAt: string; endsAt: string } | null {
  if (!date.value || !startTime.value) return null
  const start = new Date(`${date.value}T${startTime.value}`)
  if (Number.isNaN(start.getTime())) return null
  const end = new Date(start.getTime() + durationMinutes.value * 60_000)
  return { startsAt: start.toISOString(), endsAt: end.toISOString() }
}

function onGetQuote(): void {
  const range = rangeFromForm()
  if (!range) return
  reviewRange.value = range
  void getQuote(props.courtId, range.startsAt, range.endsAt)
}

function onPickSuggestion(slot: BookingSlot): void {
  reviewRange.value = slot
  void getQuote(props.courtId, slot.startsAt, slot.endsAt)
}

function onBookAnother(): void {
  reset()
  reviewRange.value = null
  date.value = ''
  startTime.value = ''
}

function formatRange(startsAt: string, endsAt: string): string {
  const start = new Date(startsAt)
  const end = new Date(endsAt)
  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
  return `${dateFmt.format(start)}, ${timeFmt.format(start)}–${timeFmt.format(end)}`
}
</script>

<template>
  <section class="court-booking" aria-label="Book this court">
    <div class="court-booking__header">
      <h3 class="court-booking__heading">Book {{ courtLabel }}</h3>
      <button type="button" class="court-booking__close" @click="emit('close')">Close</button>
    </div>

    <!-- FORM STEP: date/time picker, calls GetQuote -->
    <form v-if="step === 'form'" class="court-booking__form" @submit.prevent="onGetQuote">
      <div class="court-booking__field">
        <label :for="`booking-date-${courtId}`">Date</label>
        <input :id="`booking-date-${courtId}`" v-model="date" type="date" required />
      </div>
      <div class="court-booking__field">
        <label :for="`booking-time-${courtId}`">Start time</label>
        <input :id="`booking-time-${courtId}`" v-model="startTime" type="time" required />
      </div>
      <div class="court-booking__field">
        <label :for="`booking-duration-${courtId}`">Duration</label>
        <select :id="`booking-duration-${courtId}`" v-model="durationMinutes">
          <option :value="30">30 minutes</option>
          <option :value="60">60 minutes</option>
          <option :value="90">90 minutes</option>
          <option :value="120">120 minutes</option>
        </select>
      </div>

      <p v-if="quoteError" class="court-booking__status court-booking__status--error" role="alert">
        {{ quoteError }}
      </p>

      <button type="submit" class="court-booking__primary" :disabled="!canGetQuote || quoteLoading">
        {{ quoteLoading ? 'Getting quote…' : 'Get quote' }}
      </button>
    </form>

    <!-- REVIEW/CONFIRM STEP (WCAG 3.3.4 Error Prevention) -->
    <div v-else-if="step === 'review' && quote && reviewRange" class="court-booking__review">
      <h4 class="court-booking__review-heading">Review your booking</h4>
      <dl class="court-booking__summary">
        <div class="court-booking__summary-row">
          <dt>Court</dt>
          <dd>{{ courtLabel }}</dd>
        </div>
        <div class="court-booking__summary-row">
          <dt>Time</dt>
          <dd>{{ formatRange(reviewRange.startsAt, reviewRange.endsAt) }}</dd>
        </div>
        <div class="court-booking__summary-row">
          <dt>Price</dt>
          <dd>{{ formatPriceCents(quote.priceCents) }} <span class="court-booking__band">({{ quote.band }})</span></dd>
        </div>
      </dl>

      <p v-if="quoteRefreshedNotice" class="court-booking__status court-booking__status--notice" role="status">
        {{ quoteRefreshedNotice }}
      </p>

      <!-- Double-booking conflict (WCAG 3.3.3 Error Suggestion) -->
      <div v-if="conflict" class="court-booking__conflict" role="alert">
        <p class="court-booking__conflict-message">{{ conflict.message }}</p>
        <p v-if="conflict.suggestionsLoading">Looking for the next available time…</p>
        <template v-else-if="conflict.suggestions.length > 0">
          <p>Here are the next available times for this court:</p>
          <ul class="court-booking__suggestions">
            <li v-for="slot in conflict.suggestions" :key="slot.startsAt">
              <button type="button" class="court-booking__suggestion" @click="onPickSuggestion(slot)">
                {{ formatRange(slot.startsAt, slot.endsAt) }}
              </button>
            </li>
          </ul>
        </template>
        <p v-else>No other times were found nearby — try a different day.</p>
      </div>

      <p v-if="bookingError" class="court-booking__status court-booking__status--error" role="alert">
        {{ bookingError }}
      </p>

      <div class="court-booking__actions">
        <button type="button" class="court-booking__secondary" :disabled="bookingLoading" @click="backToForm">
          Change time
        </button>
        <button type="button" class="court-booking__primary" :disabled="bookingLoading" @click="confirmBooking">
          {{ bookingLoading ? 'Booking…' : 'Confirm booking' }}
        </button>
      </div>
    </div>

    <!-- SUCCESS STEP: ARIA live region (WCAG 4.1.3) -->
    <div v-else-if="step === 'success' && confirmedBooking" class="court-booking__success" role="status" aria-live="polite">
      <p>
        Booking confirmed for {{ courtLabel }}, {{ formatRange(confirmedBooking.startsAt, confirmedBooking.endsAt) }}.
        Reference: {{ confirmedBooking.id }}.
      </p>
      <button type="button" class="court-booking__secondary" @click="onBookAnother">Book another time</button>
    </div>
  </section>
</template>

<style scoped>
.court-booking {
  font-family: var(--font-family-ui);
  color: var(--ink);
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.court-booking__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.court-booking__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.court-booking__close {
  font: inherit;
  min-height: 44px;
  min-width: 44px;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink);
  cursor: pointer;
}

.court-booking__form,
.court-booking__review {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.court-booking__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.court-booking__field label {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.court-booking__field input,
.court-booking__field select {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
}

.court-booking__status {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.court-booking__status--error {
  color: var(--ink-warning);
}

.court-booking__status--notice {
  color: var(--ink-warning);
}

.court-booking__primary,
.court-booking__secondary {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.court-booking__primary {
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper-raised);
}

.court-booking__primary:disabled,
.court-booking__secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.court-booking__secondary {
  border: 1px solid var(--hs-border);
  background: transparent;
  color: var(--ink);
}

.court-booking__review-heading {
  font-size: var(--font-size-base);
  margin: 0;
}

.court-booking__summary {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.court-booking__summary-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.court-booking__summary-row dt {
  color: var(--ink-soft);
}

.court-booking__summary-row dd {
  margin: 0;
  font-weight: 600;
}

.court-booking__band {
  font-weight: 400;
  color: var(--ink-soft);
}

.court-booking__conflict {
  border: 1px solid var(--pill-critical-bg);
  background: var(--pill-critical-bg);
  color: var(--ink);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
}

.court-booking__conflict-message {
  font-weight: 600;
  margin: 0 0 0.5rem;
}

.court-booking__suggestions {
  list-style: none;
  margin: 0.5rem 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.court-booking__suggestion {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--court);
  cursor: pointer;
}

.court-booking__actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.court-booking__success {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  color: var(--ink-success);
}

/* Responsive: iPhone keeps the stacked single-column form; iPad/web widen
   the field rows and put the confirm actions side by side with more
   breathing room, matching the rest of this screen's breakpoint handling
   (src/composables/useBreakpoint.ts / src/styles/breakpoints.css). */
@media (min-width: 768px) {
  .court-booking__form {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    align-items: end;
    gap: 1rem;
  }

  .court-booking__form .court-booking__status,
  .court-booking__form button[type='submit'] {
    grid-column: 1 / -1;
  }
}
</style>
