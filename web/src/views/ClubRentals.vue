<script setup lang="ts">
// Club rental request + status screen — T11.6 (docs/process/t11-sprint-plan.md),
// against T11.5's REAL `RequestRecurringHire` and this ticket's new
// actor-scoped `ListRecurringHireTemplatesForActor`, not a mockup.
//
// LAYOUT: reuses FacilityDiscounts.vue's owner-console shape (which itself
// reuses FacilityOnboarding.vue's) per the ticket's instruction — the same
// step nav / `<section class="cr-step">` / `cr-field` + inline
// `cr-field-error` / `role="status"` live-region conventions, class names
// renamed `fd-` -> `cr-`. FacilityDiscounts is the more recent precedent of
// the two and is the one followed.
//
// STATE/NETWORK lives in useClubRentals (request + status list) and
// useActorRoles (the role gate); this component is the presentational half,
// the same split as CourtBookingFlow vs. useCourtBooking. The WCAG
// 3.3.1/3.3.3 validation messages come from models/recurringHireForm.ts's
// pure `validateRecurringHireForm` and are unit-tested there.
//
// ── THE ROLE GATE (ticket instruction #3, sprint plan A6) ────────────────
// The "Request a recurring rental" control must be ABSENT for a logged-in
// actor whose Roles do not include `club`. Three properties make that real
// rather than cosmetic:
//
//  1. It is driven by the actor's REAL Roles, read from Identity's `GetUser`
//     (useActorRoles), not by a client-side guess, a query param, or a
//     self-declared flag.
//  2. It fails CLOSED. While the lookup is in flight, or if it fails, `isClub`
//     is false and the control is not rendered — but the screen does not then
//     claim the actor is not a club (see the three states in useActorRoles).
//  3. Absence means ABSENT, not disabled: `v-if`, so no request tab, no form,
//     and no submit button exist in the DOM at all. A disabled control would
//     still be a control, and the assertion in the spec — which scans for
//     aria-label, <label> association and ARIA/implicit role, not just
//     id/name — would still find it.
//
// This is a rendering decision only. The authorization is the server's:
// RequestRecurringHire resolves the actor's Roles itself and answers
// PermissionDenied regardless of what this client chooses to show (T11.5,
// sprint plan A4 checklist item 2). `useClubRentals.request` surfaces that
// 403 with its own message, so the two are independent.
//
// ── NO FABRICATED DATA (instruction #4) ─────────────────────────────────
// Every status string and every schedule string on this screen comes from
// models/recurringHire.ts, which formats absent fields as "Not specified"
// rather than defaulting them (a `weekday` of 0 MEANS Sunday, so `?? 0` would
// invent a schedule). A rejected request is shown as a terminal state with
// the explicit "this request cannot be approved later; send a new one"
// explanation, and an approved one carries the note that this screen cannot
// know which individual weeks were booked — those per-week outcomes exist
// only in the owner's approval response.
import { ref, reactive, computed, onMounted } from 'vue'
import { useClubRentals } from '../composables/useClubRentals'
import { useActorRoles } from '../composables/useActorRoles'
import { useFacilityList } from '../composables/useFacilityList'
import { useFacilityDetail } from '../composables/useFacilityDetail'
import { useBreakpoint } from '../composables/useBreakpoint'
import {
  WEEKDAY_OPTIONS,
  formatWeekday,
  formatTimeRange,
  formatDate,
  formatRecurringEndCondition,
  formatStatus,
  statusExplanation,
  approvedWeeksAreUnknownNote,
} from '../models/recurringHire'
import { emptyRecurringHireForm, type RecurringHireFormInput } from '../models/recurringHireForm'
import type { BookingClient } from '../api/bookingClient'
import type { IdentityClient } from '../api/identityClient'
import type { FacilitiesClient } from '../api/facilitiesClient'
// The one mock identity every screen shares (useJoinGame's MOCK_PLAYER_ID, via
// useProfile's re-export) — reused rather than adding a fifth placeholder, per
// that constant's own "keeps the gap visible in one place" reasoning. There is
// no session layer yet (HANDOFF.md's Auth cross-cutting item), so today this
// id resolves to no User and the role gate correctly stays closed: this screen
// renders its "we could not confirm a club role" state against a real backend,
// which is the honest outcome, not a bug being routed around.
import { MOCK_PLAYER_ID } from '../composables/useProfile'

const STEP_ORDER = ['new', 'list'] as const
type Step = (typeof STEP_ORDER)[number]

const STEP_LABELS: Record<Step, string> = {
  new: 'Request a recurring rental',
  list: 'My requests',
}

const props = withDefaults(
  defineProps<{
    /** Injectable for tests; default to the real clients. */
    client?: BookingClient
    identity?: IdentityClient
    facilities?: FacilitiesClient
    /** The current actor. Defaults to the shared mock identity — see the
     * import note above. */
    actorUserId?: string
  }>(),
  { client: undefined, identity: undefined, facilities: undefined, actorUserId: MOCK_PLAYER_ID },
)

const { breakpoint } = useBreakpoint()

const {
  templates,
  loading,
  error,
  requesting,
  fieldErrors,
  formError,
  statusMessage,
  load,
  request,
} = useClubRentals(props.client)

const {
  isClub,
  resolved: rolesResolved,
  loading: rolesLoading,
  error: rolesError,
  load: loadActorRoles,
} = useActorRoles(props.identity)

const { facilities: facilityOptions, search: searchFacilities } = useFacilityList(props.facilities)
const { facility: selectedFacility, load: loadFacility } = useFacilityDetail(props.facilities)

const currentStep = ref<Step>('list')
const form = reactive<RecurringHireFormInput>(emptyRecurringHireForm())
const selectedFacilityId = ref('')

const courtOptions = computed(() => selectedFacility.value?.courts ?? [])

/** The single gate. Everything the request flow renders — the tab, the form,
 * the submit button — is behind this one computed, so no two of them can
 * disagree about whether the actor may request a rental. It is `isClub`,
 * which is false unless a real `GetUser` said otherwise. */
const canRequest = computed(() => isClub.value)

/** The steps actually offered. */
const visibleSteps = computed<Step[]>(() => (canRequest.value ? [...STEP_ORDER] : ['list']))

async function selectFacility(facilityId: string): Promise<void> {
  selectedFacilityId.value = facilityId
  // Changing facility invalidates any court already chosen — clearing it is
  // what stops a court from another facility being submitted silently.
  form.courtId = ''
  if (facilityId) await loadFacility(facilityId)
}

async function submitRequest(): Promise<void> {
  // No client-side short-circuit here: `request` itself runs the validation
  // gate (and returns false without calling the API), so this handler cannot
  // diverge from it.
  const created = await request({ ...form }, props.actorUserId)
  if (!created) return
  Object.assign(form, emptyRecurringHireForm())
  selectedFacilityId.value = ''
  currentStep.value = 'list'
}

onMounted(async () => {
  await loadActorRoles(props.actorUserId)
  void load(props.actorUserId)
  void searchFacilities('')
})

// Exposed for tests that exercise the validation gate directly (bypassing the
// form's own submit handler), proving it is a real code path and not just a
// UI affordance — same as FacilityDiscounts.vue/FacilityOnboarding.vue.
defineExpose({ submitRequest, currentStep, selectFacility })
</script>

<template>
  <div class="club-rentals" :data-breakpoint="breakpoint">
    <h1 class="cr-heading">Recurring rentals</h1>

    <!-- WCAG 4.1.3 Status Messages: confirmations are announced through this
         live region, never only as a visual change. -->
    <p class="cr-status" role="status">{{ statusMessage }}</p>

    <nav v-if="visibleSteps.length > 1" class="cr-steps" aria-label="Rental sections">
      <button
        v-for="step in visibleSteps"
        :key="step"
        type="button"
        class="cr-steps__item"
        :class="{ 'cr-steps__item--active': currentStep === step }"
        :aria-current="currentStep === step ? 'step' : undefined"
        @click="currentStep = step"
      >
        {{ STEP_LABELS[step] }}
      </button>
    </nav>

    <!-- The honest account of WHY the request control is missing. Three
         distinct states, because "we could not check" and "we checked, and
         you are not a club" are different facts and only the second one may
         be stated as such. -->
    <section v-if="!canRequest" data-testid="no-club-role-notice" class="cr-notice">
      <p v-if="rolesLoading" role="status">Checking what your account can do…</p>
      <p v-else-if="rolesError" class="cr-field-error" role="alert">
        {{ rolesError }} Until that check succeeds, this screen cannot offer to send a rental
        request.
      </p>
      <p v-else-if="rolesResolved">
        Recurring rentals are booked by club accounts. This account is not registered as a club,
        so it cannot send a rental request. Your existing requests, if any, are listed below.
      </p>
      <p v-else role="status">Checking what your account can do…</p>
    </section>

    <!-- Step 1: request. Absent entirely — not disabled — unless the actor's
         real Roles include `club`. -->
    <section
      v-if="currentStep === 'new' && canRequest"
      data-testid="new-rental-step"
      class="cr-step"
    >
      <h2>Request a recurring rental</h2>
      <form class="cr-fields" @submit.prevent="submitRequest">
        <label class="cr-field">
          <span>Facility</span>
          <select
            id="rental-facility"
            :value="selectedFacilityId"
            @change="selectFacility(($event.target as HTMLSelectElement).value)"
          >
            <option value="">Choose a facility</option>
            <option v-for="option in facilityOptions" :key="option.id" :value="option.id">
              {{ option.name }}
            </option>
          </select>
        </label>

        <label class="cr-field">
          <span>Court</span>
          <select
            id="rental-court"
            v-model="form.courtId"
            :aria-invalid="fieldErrors.courtId ? 'true' : undefined"
            :aria-describedby="fieldErrors.courtId ? 'rental-court-error' : undefined"
          >
            <option value="">
              {{ selectedFacilityId ? 'Choose a court' : 'Choose a facility first' }}
            </option>
            <option v-for="court in courtOptions" :key="court.id" :value="court.id">
              {{ court.name }}
            </option>
          </select>
          <p v-if="fieldErrors.courtId" id="rental-court-error" class="cr-field-error" role="alert">
            {{ fieldErrors.courtId }}
          </p>
        </label>

        <label class="cr-field">
          <span>Repeats on</span>
          <select
            id="rental-weekday"
            v-model="form.weekday"
            :aria-invalid="fieldErrors.weekday ? 'true' : undefined"
            :aria-describedby="fieldErrors.weekday ? 'rental-weekday-error' : undefined"
          >
            <option value="">Choose a day</option>
            <option v-for="option in WEEKDAY_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
          <p
            v-if="fieldErrors.weekday"
            id="rental-weekday-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.weekday }}
          </p>
        </label>

        <label class="cr-field">
          <span>Starts at (time)</span>
          <input
            id="rental-start-time"
            v-model="form.startTime"
            type="time"
            :aria-invalid="fieldErrors.startTime ? 'true' : undefined"
            :aria-describedby="fieldErrors.startTime ? 'rental-start-time-error' : undefined"
          />
          <p
            v-if="fieldErrors.startTime"
            id="rental-start-time-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.startTime }}
          </p>
        </label>

        <label class="cr-field">
          <span>Ends at (time)</span>
          <input
            id="rental-end-time"
            v-model="form.endTime"
            type="time"
            :aria-invalid="fieldErrors.endTime ? 'true' : undefined"
            :aria-describedby="fieldErrors.endTime ? 'rental-end-time-error' : undefined"
          />
          <p
            v-if="fieldErrors.endTime"
            id="rental-end-time-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.endTime }}
          </p>
        </label>

        <label class="cr-field">
          <span>First date</span>
          <input
            id="rental-starts-at"
            v-model="form.startsAt"
            type="date"
            aria-describedby="rental-starts-at-hint"
            :aria-invalid="fieldErrors.startsAt ? 'true' : undefined"
          />
          <!-- States the server's own rule rather than computing a first
               session date here: the backend anchors generation to the first
               matching weekday on or after this date, and a client-side guess
               at that date would be exactly the kind of derived-looking fact
               that can be wrong. -->
          <p id="rental-starts-at-hint" class="cr-hint">
            Sessions run on the first chosen weekday on or after this date.
          </p>
          <p
            v-if="fieldErrors.startsAt"
            id="rental-starts-at-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.startsAt }}
          </p>
        </label>

        <fieldset class="cr-fieldset">
          <legend>How long it runs</legend>
          <label class="cr-radio">
            <input
              v-model="form.endKind"
              type="radio"
              value="RECURRING_HIRE_END_CONDITION_KIND_NO_END"
            />
            <span>Keep going with no end date</span>
          </label>
          <label class="cr-radio">
            <input
              v-model="form.endKind"
              type="radio"
              value="RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE"
            />
            <span>Stop on a date</span>
          </label>
          <label class="cr-radio">
            <input
              v-model="form.endKind"
              type="radio"
              value="RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES"
            />
            <span>Run for a number of weeks</span>
          </label>
          <!-- The horizon is the server's, and it is real: an open-ended
               request still only generates a bounded set of bookings on
               approval. Saying "forever" would be a promise the backend does
               not make. -->
          <p class="cr-hint">
            An open-ended rental is still only booked up to the facility’s booking horizon at the
            time it is approved.
          </p>
        </fieldset>

        <label
          v-if="form.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE'"
          class="cr-field"
        >
          <span>Stops on</span>
          <input
            id="rental-end-date"
            v-model="form.endDate"
            type="date"
            :aria-invalid="fieldErrors.endDate ? 'true' : undefined"
            :aria-describedby="fieldErrors.endDate ? 'rental-end-date-error' : undefined"
          />
          <p
            v-if="fieldErrors.endDate"
            id="rental-end-date-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.endDate }}
          </p>
        </label>

        <label
          v-if="form.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES'"
          class="cr-field"
        >
          <span>Number of weeks</span>
          <input
            id="rental-occurrences"
            v-model="form.occurrences"
            type="number"
            min="1"
            step="1"
            inputmode="numeric"
            :aria-invalid="fieldErrors.occurrences ? 'true' : undefined"
            :aria-describedby="fieldErrors.occurrences ? 'rental-occurrences-error' : undefined"
          />
          <p
            v-if="fieldErrors.occurrences"
            id="rental-occurrences-error"
            class="cr-field-error"
            role="alert"
          >
            {{ fieldErrors.occurrences }}
          </p>
        </label>

        <p v-if="formError" class="cr-field-error" role="alert">{{ formError }}</p>

        <div class="cr-actions">
          <button type="submit" data-testid="request-rental-button" :disabled="requesting">
            {{ requesting ? 'Sending…' : 'Send rental request' }}
          </button>
        </div>
      </form>
    </section>

    <!-- Step 2: status view. Shown to every actor, club or not: the backend's
         actor-scoped read is deliberately not role-gated, so a club that later
         lost the role can still read back its own history, including the
         rejections. -->
    <section v-else data-testid="rental-list-step" class="cr-step">
      <h2>My requests</h2>

      <p v-if="loading" role="status">Loading your rental requests…</p>
      <p v-else-if="error" class="cr-field-error" role="alert">{{ error }}</p>
      <p v-else-if="templates.length === 0" class="cr-empty">
        You have not requested any recurring rentals yet.
      </p>

      <ul v-else class="cr-list">
        <li v-for="template in templates" :key="template.id" class="cr-request">
          <p class="cr-request__schedule">
            {{ formatWeekday(template.weekday) }},
            {{ formatTimeRange(template.startMinute, template.endMinute) }}
          </p>
          <dl class="cr-request__details">
            <div class="cr-request__row">
              <dt>Status</dt>
              <!-- Text, not colour alone (WCAG 1.4.1). -->
              <dd :data-testid="`rental-status-${template.id}`">{{ formatStatus(template.status) }}</dd>
            </div>
            <div class="cr-request__row">
              <dt>From</dt>
              <dd>{{ formatDate(template.startsAt) || 'Not specified' }}</dd>
            </div>
            <div class="cr-request__row">
              <dt>Runs</dt>
              <dd>{{ formatRecurringEndCondition(template.endCondition) }}</dd>
            </div>
          </dl>
          <p class="cr-request__explanation">{{ statusExplanation(template.status) }}</p>
          <!-- An approved template does not mean every week was booked, and
               this screen has no way to learn which were: those outcomes are
               only in the owner's approval response. Say so. -->
          <p
            v-if="template.status === 'RECURRING_HIRE_STATUS_APPROVED'"
            class="cr-request__explanation"
          >
            {{ approvedWeeksAreUnknownNote }}
          </p>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
/* Mirrors FacilityDiscounts.vue's `fd-` styles (themselves FacilityOnboarding's
   `fo-` set): same tokens, same 44px touch targets, same responsive field
   grid — renamed to `cr-`. */
.club-rentals {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 720px;
}

.cr-heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.cr-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
}

.cr-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.cr-steps__item {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink-soft);
  cursor: pointer;
}

.cr-steps__item--active {
  background: var(--court);
  color: var(--paper);
  border-color: var(--court);
}

.cr-step,
.cr-notice {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cr-notice p {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.cr-fields {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.cr-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.cr-field input,
.cr-field select {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

/* Error text: colour is REINFORCEMENT only. The message itself is the
   identification (WCAG 3.3.1) and it carries the fix (3.3.3), so nothing here
   depends on the colour being perceived (1.4.1). */
.cr-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.cr-hint {
  color: var(--ink-soft);
  font-size: var(--font-size-xs);
  margin: 0;
}

.cr-fieldset {
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  padding: 0.75rem 1rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
}

.cr-fieldset legend {
  font-size: var(--font-size-sm);
  font-weight: 600;
  padding: 0 0.35rem;
}

.cr-radio {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-height: 44px;
  font-size: var(--font-size-sm);
  cursor: pointer;
}

.cr-radio input {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.cr-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cr-actions button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.cr-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.cr-empty {
  margin: 0;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.cr-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.cr-request {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem;
  background: var(--paper);
  border-radius: var(--radius-sm);
}

.cr-request__schedule {
  font-weight: 600;
  margin: 0;
}

.cr-request__details {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.cr-request__row {
  display: flex;
  gap: 0.5rem;
}

.cr-request__row dt {
  font-weight: 600;
}

.cr-request__row dd {
  margin: 0;
}

.cr-request__explanation {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

/* Two-column field grid on iPad (768-1180px), matching FacilityDiscounts. */
@media (min-width: 768px) {
  .cr-fields {
    grid-template-columns: repeat(2, 1fr);
  }

  .cr-fields .cr-fieldset {
    grid-column: 1 / -1;
  }
}

@media (min-width: 1280px) {
  .club-rentals {
    max-width: 960px;
  }
}
</style>
