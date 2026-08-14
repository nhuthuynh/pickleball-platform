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
import { onMounted, nextTick, ref, computed } from 'vue'
import { useFacilityRentalRequests, type Decision } from '../composables/useFacilityRentalRequests'
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
  pendingDecision,
  requestDecision,
  cancelDecision,
  load,
  approve,
  reject,
} = useFacilityRentalRequests(props.client)

// ── CONFIRM BEFORE COMMIT (WCAG 3.3.4 Error Prevention) — T12.5 ──────────
// T11.6 shipped Approve/Reject as single-click, one-way, no-undo controls.
// Approving generates real Bookings across every implied week and T11.4
// models both transitions as terminal (this file's header, above), which is
// exactly 3.3.4's Legal/Financial/Data scope. The flow that already answers
// this criterion in this codebase is CourtBookingFlow.vue's REVIEW/CONFIRM
// STEP, so this is the same shape rather than a second convention: a review
// panel stating what is about to happen, an explicit confirm control, and a
// way back out — with the GATE itself in the composable (see
// useFacilityRentalRequests), not only in this markup.
const pendingPanel = ref<HTMLElement | null>(null)
const triggerButtons: Record<string, HTMLElement | null> = {}

function registerTrigger(key: string, el: unknown): void {
  triggerButtons[key] = (el as HTMLElement | null) ?? null
}

function isAwaitingConfirmation(templateId: string): boolean {
  return pendingDecision.value?.templateId === templateId
}

/** Everything the confirm step renders, resolved in one place so the
 * template never has to re-derive which decision is staged (and so the
 * control for the decision that was NOT chosen is never in the DOM at all —
 * hidden-but-present would still be reachable by a screen reader and by any
 * "is this control offered?" scan). */
const activeConfirm = computed(() => {
  const pending = pendingDecision.value
  if (!pending) return null
  const template = templates.value.find((t) => t.id === pending.templateId)
  if (!template) return null
  return {
    template,
    decision: pending.decision,
    heading: CONFIRM_HEADINGS[pending.decision],
    consequence: CONFIRM_CONSEQUENCES[pending.decision],
    confirmLabel: CONFIRM_LABELS[pending.decision],
  }
})

/** Commits whichever decision is staged. Routing both through one handler
 * keeps the component from being a second place that decides which write to
 * fire — the composable's gate remains the only thing that can authorise
 * either one. */
function commitPending(): void {
  const pending = pendingDecision.value
  if (!pending) return
  const run = pending.decision === 'approve' ? approve : reject
  void run(pending.templateId, MOCK_OWNER_ID)
}

/** Keyboard (WCAG 2.4.3 Focus Order): choosing a decision REPLACES the
 * button that has focus, so without this focus would fall to <body> and a
 * keyboard-only owner would have to re-traverse the queue to reach the
 * confirmation they just asked for. Same technique CompetitionLanding.vue
 * uses for its heading: `tabindex="-1"` so the panel is programmatically
 * focusable without becoming a tab stop of its own. */
async function chooseDecision(templateId: string, decision: Decision): Promise<void> {
  requestDecision(templateId, decision)
  await nextTick()
  pendingPanel.value?.focus()
}

/** And the return journey: backing out puts focus back on the control the
 * owner left, not at the top of the document. */
async function backOut(templateId: string, decision: Decision): Promise<void> {
  cancelDecision()
  await nextTick()
  triggerButtons[`${decision}-${templateId}`]?.focus()
}

const CONFIRM_HEADINGS: Record<Decision, string> = {
  approve: 'Confirm this approval',
  reject: 'Confirm this rejection',
}

/** The consequence, in text — the thing an owner is actually being asked to
 * check. Deliberately does NOT claim how many weeks will be booked: the
 * per-week outcome exists only in the approval response (see this file's
 * header), so a count here would be a guess presented as a fact. */
const CONFIRM_CONSEQUENCES: Record<Decision, string> = {
  approve:
    'Approving creates a real booking for every week this request covers, and cannot be undone from ' +
    'this screen — any week you did not want would have to be cancelled one at a time.',
  reject:
    'Rejecting books no courts and closes this request for good. It cannot be undone — the club would ' +
    'have to send a new request.',
}

const CONFIRM_LABELS: Record<Decision, string> = {
  approve: 'Yes, approve this request',
  reject: 'Yes, reject this request',
}

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
// `requestDecision`/`cancelDecision` join them so a test can stage a decision
// without the markup too. Note this exposure is exactly why the 3.3.4 gate
// had to go in the composable: `approve`/`reject` are reachable here without
// any button being pressed, so a markup-only confirm step would not be one.
defineExpose({ approve, reject, requestDecision, cancelDecision })
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

          <!-- CHOOSE. Neither control commits anything: each only stages a
               decision for the confirm step below (WCAG 3.3.4). They are
               replaced by that step for the request being decided, while
               every OTHER request in the queue stays answerable. -->
          <div v-if="canDecide(template) && !isAwaitingConfirmation(template.id)" class="frr-actions">
            <button
              :ref="(el) => registerTrigger(`approve-${template.id}`, el)"
              type="button"
              :data-testid="`approve-${template.id}`"
              :disabled="deciding === template.id"
              @click="chooseDecision(template.id, 'approve')"
            >
              Approve request
            </button>
            <button
              :ref="(el) => registerTrigger(`reject-${template.id}`, el)"
              type="button"
              class="frr-actions__secondary"
              :data-testid="`reject-${template.id}`"
              :disabled="deciding === template.id"
              @click="chooseDecision(template.id, 'reject')"
            >
              Reject request
            </button>
          </div>

          <!-- REVIEW/CONFIRM STEP (WCAG 3.3.4 Error Prevention) — the same
               shape CourtBookingFlow.vue uses for this criterion: a heading,
               a <dl> restating exactly what is about to happen, the
               consequence in words, then an explicit confirm and an explicit
               way back out. `tabindex="-1"` so focus can be moved here
               without adding a tab stop of its own. -->
          <template v-if="activeConfirm && activeConfirm.template.id === template.id">
            <div
              :ref="(el) => (pendingPanel = el as HTMLElement | null)"
              class="frr-confirm"
              tabindex="-1"
              role="group"
              :aria-labelledby="`confirm-heading-${template.id}`"
              :data-testid="`confirm-step-${activeConfirm.decision}-${template.id}`"
            >
              <h3 :id="`confirm-heading-${template.id}`" class="frr-confirm__heading">
                {{ activeConfirm.heading }}
              </h3>

              <dl class="frr-confirm__summary">
                <div class="frr-request__row">
                  <dt>Slot</dt>
                  <dd>
                    {{ formatWeekday(template.weekday) }},
                    {{ formatTimeRange(template.startMinute, template.endMinute) }}
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

              <p class="frr-confirm__consequence">{{ activeConfirm.consequence }}</p>

              <div class="frr-actions">
                <button
                  type="button"
                  :data-testid="`confirm-${activeConfirm.decision}-${template.id}`"
                  :disabled="deciding === template.id"
                  @click="commitPending()"
                >
                  {{ deciding === template.id ? 'Working…' : activeConfirm.confirmLabel }}
                </button>
                <button
                  type="button"
                  class="frr-actions__secondary"
                  :data-testid="`cancel-${activeConfirm.decision}-${template.id}`"
                  :disabled="deciding === template.id"
                  @click="backOut(template.id, activeConfirm.decision)"
                >
                  No, keep this request open
                </button>
              </div>
            </div>
          </template>

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

/* WCAG 2.4.7 Focus Visible / 2.4.11 Focus Appearance. T12.5's keyboard pass:
   this screen shipped relying on the user agent's default focus ring, which
   is not guaranteed to have adequate contrast against the `--court`-filled
   primary button. Explicit indicator, matching the convention
   CompetitionLanding.vue / CompetitionEntryPanel.vue already established
   (3px solid --court, offset 2px) rather than a new one. */
.frr-actions button:focus-visible,
.frr-confirm:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 2px;
}

/* The confirm step is visually set apart from the request it belongs to, but
   nothing here depends on that: the heading, the <dl> and the consequence
   sentence carry the meaning in TEXT (WCAG 1.4.1). */
.frr-confirm {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  border: 2px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}

.frr-confirm__heading {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--court);
}

.frr-confirm__summary {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.frr-confirm__consequence {
  margin: 0;
  font-size: var(--font-size-sm);
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
