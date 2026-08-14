<script setup lang="ts">
// Facility Owner incoming rental-requests panel — T11.6
// (docs/process/t11-sprint-plan.md), against T11.5's REAL
// `ListRecurringHireTemplatesForFacility`/`ApproveRecurringHire`/
// `RejectRecurringHire` endpoints, not a mockup.
//
// LAYOUT: reuses FacilityDiscounts.vue's owner-console patterns (which reuse
// FacilityOnboarding.vue's) per the ticket's instruction — same `role="status"`
// live region, same `frr-`-renamed field/error/list conventions, same 44px
// touch targets. FacilityDiscounts is the more recent of the two precedents
// and is the one followed. Single-section rather than stepped: this screen has
// one job, a queue.
//
// ── PER-OCCURRENCE VISIBILITY (instruction #2) ──────────────────────────
// T11.5's approval books each implied week INDEPENDENTLY: a week whose court
// is already booked is skipped, and the template still becomes `approved`
// regardless. So "Approved" on its own tells an owner nothing about what they
// just did. This screen therefore always renders, for an approval made in this
// session:
//   - a counted summary (booked / skipped-conflict / skipped-error out of the
//     total), also announced through the live region; and
//   - the full per-week list, each row carrying its own outcome and, for a
//     skipped week, the reason the server gave.
// A bare "approved" is never shown.
//
// ── NO FABRICATED DATA (instruction #4) ─────────────────────────────────
// Those per-week outcomes exist ONLY in the ApproveRecurringHire response —
// nothing replays them later. A template that was already approved before this
// screen loaded therefore has no result list, and the screen SAYS that rather
// than reconstructing one from the schedule (which would be a guess presented
// as a record). Likewise a rejected request is terminal: no Approve control is
// offered for it, and the copy states a new request would be required.
import { onMounted } from 'vue'
import { useFacilityRentalRequests } from '../composables/useFacilityRentalRequests'
import { useBreakpoint } from '../composables/useBreakpoint'
import {
  formatWeekday,
  formatTimeRange,
  formatDate,
  formatDateTime,
  formatRecurringEndCondition,
  formatStatus,
  statusExplanation,
  formatOccurrenceOutcome,
  summariseOccurrences,
  summariseApproval,
  isDecided,
  type RecurringHireTemplateView,
} from '../models/recurringHire'
import type { BookingClient } from '../api/bookingClient'

// Same caveat class and deliberately the SAME value as FacilityOnboarding.vue/
// FacilityDiscounts.vue's MOCK_OWNER_ID, so a facility created through those
// screens is owned by the actor this one claims to be. `actorUserId` is a
// caller-supplied claim, not a verified identity (booking.proto says so
// explicitly); the object-level check it feeds is real regardless — the server
// resolves it against the Facility's actual owner and 403s on a mismatch, and
// this list is owner-only for that reason.
const MOCK_OWNER_ID = 'owner-mock-1'

const props = withDefaults(
  defineProps<{
    /** The Facility id, supplied by the router from
     * `/facilities/:facilityId/rental-requests`. */
    facilityId: string
    /** Injectable for tests; defaults to the real bookingClient. */
    client?: BookingClient
  }>(),
  { client: undefined },
)

const { breakpoint } = useBreakpoint()

const {
  templates,
  loading,
  error,
  deciding,
  decisionError,
  statusMessage,
  approvalResults,
  load,
  approve,
  reject,
} = useFacilityRentalRequests(props.client)

/** A decision is offered only while the request is still open. T11.4 models
 * approve/reject as one-way transitions, so an already-answered request cannot
 * be answered again — offering the buttons anyway would advertise an action
 * the server would refuse with FailedPrecondition. */
function canDecide(template: RecurringHireTemplateView): boolean {
  return !isDecided(template.status)
}

function occurrencesFor(templateId: string) {
  return approvalResults.value[templateId]
}

function summaryFor(templateId: string): string {
  const occurrences = approvalResults.value[templateId] ?? []
  return summariseApproval(summariseOccurrences(occurrences))
}

onMounted(() => {
  void load(props.facilityId, MOCK_OWNER_ID)
})

// Exposed so tests can drive the decisions directly, proving they are real
// code paths rather than only UI affordances — same as FacilityDiscounts.vue.
defineExpose({ approve, reject })
</script>

<template>
  <div class="facility-rental-requests" :data-breakpoint="breakpoint">
    <h1 class="frr-heading">Rental requests</h1>

    <!-- WCAG 4.1.3 Status Messages. For an approval this carries the per-week
         counts, not a bare "Approved". -->
    <p class="frr-status" role="status">{{ statusMessage }}</p>

    <section class="frr-step">
      <h2>Incoming requests</h2>

      <p v-if="loading" role="status">Loading rental requests…</p>
      <p v-else-if="error" class="frr-field-error" role="alert">{{ error }}</p>
      <p v-else-if="templates.length === 0" class="frr-empty">
        No club has requested a recurring rental at this facility yet.
      </p>

      <ul v-else class="frr-list">
        <li v-for="template in templates" :key="template.id" class="frr-request">
          <p class="frr-request__schedule">
            {{ formatWeekday(template.weekday) }},
            {{ formatTimeRange(template.startMinute, template.endMinute) }}
          </p>

          <dl class="frr-request__details">
            <div class="frr-request__row">
              <dt>Status</dt>
              <dd :data-testid="`request-status-${template.id}`">
                {{ formatStatus(template.status) }}
              </dd>
            </div>
            <div class="frr-request__row">
              <dt>From</dt>
              <dd>{{ formatDate(template.startsAt) || 'Not specified' }}</dd>
            </div>
            <div class="frr-request__row">
              <dt>Runs</dt>
              <dd>{{ formatRecurringEndCondition(template.endCondition) }}</dd>
            </div>
          </dl>

          <p class="frr-request__explanation">{{ statusExplanation(template.status) }}</p>

          <div v-if="canDecide(template)" class="frr-actions">
            <button
              type="button"
              :data-testid="`approve-${template.id}`"
              :disabled="deciding === template.id"
              @click="approve(template.id, MOCK_OWNER_ID)"
            >
              {{ deciding === template.id ? 'Working…' : 'Approve request' }}
            </button>
            <button
              type="button"
              class="frr-actions__secondary"
              :data-testid="`reject-${template.id}`"
              :disabled="deciding === template.id"
              @click="reject(template.id, MOCK_OWNER_ID)"
            >
              Reject request
            </button>
          </div>

          <!-- The per-week result of an approval made in THIS session. -->
          <div
            v-if="occurrencesFor(template.id)"
            class="frr-occurrences"
            :data-testid="`occurrences-${template.id}`"
          >
            <p class="frr-occurrences__summary">{{ summaryFor(template.id) }}</p>
            <ul class="frr-occurrences__list">
              <li
                v-for="occurrence in occurrencesFor(template.id)"
                :key="`${occurrence.startsAt}-${occurrence.outcome}`"
                class="frr-occurrence"
              >
                <span class="frr-occurrence__when">{{ formatDateTime(occurrence.startsAt) }}</span>
                <!-- Outcome in TEXT, never colour alone (WCAG 1.4.1), and the
                     server's own reason where it gave one. -->
                <span class="frr-occurrence__outcome">
                  {{ formatOccurrenceOutcome(occurrence.outcome) }}
                </span>
                <span v-if="occurrence.reason" class="frr-occurrence__reason">
                  {{ occurrence.reason }}
                </span>
              </li>
            </ul>
          </div>

          <!-- An approval this screen did not perform. The per-week outcomes
               are not stored anywhere readable, so nothing is claimed about
               which weeks were booked. -->
          <p
            v-else-if="template.status === 'RECURRING_HIRE_STATUS_APPROVED'"
            class="frr-request__explanation"
            :data-testid="`no-occurrence-record-${template.id}`"
          >
            This request was approved before this screen was opened. The week-by-week result is
            only reported at the moment of approval and is not stored, so it cannot be shown here —
            check the court’s schedule for the bookings that exist.
          </p>
        </li>
      </ul>

      <p v-if="decisionError" class="frr-field-error" role="alert">{{ decisionError }}</p>
    </section>
  </div>
</template>

<style scoped>
/* Mirrors FacilityDiscounts.vue's `fd-` styles (themselves FacilityOnboarding's
   `fo-` set): same tokens, same 44px touch targets — renamed to `frr-`. */
.facility-rental-requests {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 720px;
}

.frr-heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.frr-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
}

.frr-step {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Error text: colour is REINFORCEMENT only — the message is the
   identification (WCAG 3.3.1) and carries the fix (3.3.3). */
.frr-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.frr-empty {
  margin: 0;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.frr-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.frr-request {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--paper);
  border-radius: var(--radius-sm);
}

.frr-request__schedule {
  font-weight: 600;
  margin: 0;
}

.frr-request__details {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.frr-request__row {
  display: flex;
  gap: 0.5rem;
}

.frr-request__row dt {
  font-weight: 600;
}

.frr-request__row dd {
  margin: 0;
}

.frr-request__explanation {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.frr-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.frr-actions button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.frr-actions__secondary {
  background: var(--paper-raised);
  color: var(--court);
}

.frr-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.frr-occurrences {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.6rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}

.frr-occurrences__summary {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: 600;
}

.frr-occurrences__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.frr-occurrence {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.frr-occurrence__when {
  font-variant-numeric: tabular-nums;
}

.frr-occurrence__outcome {
  font-weight: 600;
}

@media (min-width: 1280px) {
  .facility-rental-requests {
    max-width: 960px;
  }
}
</style>
