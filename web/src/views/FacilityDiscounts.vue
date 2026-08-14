<script setup lang="ts">
// Facility Owner discount panel — T11.3 (docs/process/t11-sprint-plan.md),
// against T11.2's REAL `CreateDiscountRule`/`ListDiscountRulesForFacility`
// endpoints, not a mockup. Builds the Flow-2 UI the design review's round-10
// mockups scoped ("a discount rule with an end condition") but which had no
// backend to point at until T11.2.
//
// LAYOUT: deliberately reuses FacilityOnboarding.vue's step shape (per the
// ticket's instruction #1: "not a new one-off layout") — the same step nav /
// `<section class="fd-step">` / `fd-field` + inline `fd-field-error` /
// `role="status"` live region conventions, with a two-step split (New
// discount, All discounts) instead of that screen's four. Class names are
// the `fo-` set renamed to `fd-`; the structure, the 44px touch targets, and
// the responsive field grid are unchanged.
//
// STATE/NETWORK lives in useFacilityDiscounts (this component is the
// presentational half), the same split as CourtBookingFlow vs.
// useCourtBooking. In particular the WCAG 3.3.1/3.3.3 validation messages
// come from models/discountForm.ts's pure `validateDiscountForm` and are
// unit-tested there.
//
// HONEST RENDERING (instruction #3): every end-condition string on this
// screen comes from `formatEndCondition`, which never fabricates a date and
// never implies a remaining count — the API has no remaining-count field.
// See models/discount.ts.
import { ref, reactive, onMounted, computed } from 'vue'
import { useFacilityDiscounts } from '../composables/useFacilityDiscounts'
import { formatDiscountAmount, formatEndCondition, formatAppliesTo, type Source } from '../models/discount'
import { emptyDiscountForm, type DiscountFormInput } from '../models/discountForm'
import { useBreakpoint } from '../composables/useBreakpoint'
import type { BookingClient } from '../api/bookingClient'

// No Identity/Users/Auth context exists yet — same caveat class as
// FacilityOnboarding.vue's own MOCK_OWNER_ID, and deliberately the SAME
// value, so a facility created through that screen is owned by the actor
// this one claims to be. `actorUserId` is a caller-supplied claim, not a
// verified identity (booking.proto says so explicitly); the object-level
// check it feeds is real regardless — the server resolves it against the
// Facility's actual owner and 403s on a mismatch.
const MOCK_OWNER_ID = 'owner-mock-1'

const STEP_ORDER = ['new', 'list'] as const
type Step = (typeof STEP_ORDER)[number]

const STEP_LABELS: Record<Step, string> = {
  new: 'New discount',
  list: 'All discounts',
}

const SOURCE_OPTIONS: { value: Source; label: string }[] = [
  { value: 'SOURCE_INDIVIDUAL', label: 'Individual bookings' },
  { value: 'SOURCE_RECURRING_HIRE', label: 'Recurring hire' },
  { value: 'SOURCE_GAME', label: 'Games' },
  { value: 'SOURCE_COMPETITION', label: 'Competitions' },
]

const props = withDefaults(
  defineProps<{
    /** The Facility id, supplied by the router from `/facilities/:facilityId/discounts`. */
    facilityId: string
    /** Injectable for tests; defaults to the real bookingClient. */
    client?: BookingClient
  }>(),
  { client: undefined },
)

const { breakpoint } = useBreakpoint()

const { discounts, loading, error, creating, fieldErrors, formError, statusMessage, load, create } =
  useFacilityDiscounts(props.client)

const currentStep = ref<Step>('new')
const form = reactive<DiscountFormInput>(emptyDiscountForm())

const isPercent = computed(() => form.discountType === 'DISCOUNT_TYPE_PERCENT')
const isFixedAmount = computed(() => form.discountType === 'DISCOUNT_TYPE_FIXED_AMOUNT')

function toggleSource(source: Source, checked: boolean): void {
  if (checked) {
    if (!form.appliesTo.includes(source)) form.appliesTo.push(source)
  } else {
    form.appliesTo = form.appliesTo.filter((s) => s !== source)
  }
}

async function submitDiscount(): Promise<void> {
  // No client-side short-circuit here: `create` itself runs the validation
  // gate (and returns false without calling the API), so this handler can't
  // diverge from it.
  const created = await create(props.facilityId, { ...form, appliesTo: [...form.appliesTo] }, MOCK_OWNER_ID)
  if (!created) return
  Object.assign(form, emptyDiscountForm())
  currentStep.value = 'list'
}

onMounted(() => {
  void load(props.facilityId)
})

// Exposed for tests that exercise the validation gate directly (bypassing
// the form's own submit handler), proving it is a real code path and not
// just a UI affordance — same as FacilityOnboarding.vue's defineExpose.
defineExpose({ submitDiscount, currentStep })
</script>

<template>
  <div class="facility-discounts" :data-breakpoint="breakpoint">
    <h1 class="fd-heading">Discounts</h1>

    <!-- WCAG 4.1.3 Status Messages: success confirmations are announced via
         this live region, never only a visual change. -->
    <p class="fd-status" role="status">{{ statusMessage }}</p>

    <nav class="fd-steps" aria-label="Discount sections">
      <button
        v-for="step in STEP_ORDER"
        :key="step"
        type="button"
        class="fd-steps__item"
        :class="{ 'fd-steps__item--active': currentStep === step }"
        :aria-current="currentStep === step ? 'step' : undefined"
        @click="currentStep = step"
      >
        {{ STEP_LABELS[step] }}
      </button>
    </nav>

    <!-- Step 1: create -->
    <section v-if="currentStep === 'new'" data-testid="new-discount-step" class="fd-step">
      <h2>New discount</h2>
      <form class="fd-fields" @submit.prevent="submitDiscount">
        <fieldset class="fd-fieldset">
          <legend>Discount type</legend>
          <label class="fd-radio">
            <input v-model="form.discountType" type="radio" value="DISCOUNT_TYPE_PERCENT" />
            <span>Percentage off</span>
          </label>
          <label class="fd-radio">
            <input v-model="form.discountType" type="radio" value="DISCOUNT_TYPE_FIXED_AMOUNT" />
            <span>Fixed amount off</span>
          </label>
        </fieldset>

        <!-- Only the amount field matching the chosen type is shown, because
             only that one is meaningful server-side (booking.proto). -->
        <label v-if="isPercent" class="fd-field">
          <span>Percentage off</span>
          <input
            id="discount-percent"
            v-model="form.percent"
            type="number"
            min="0.01"
            max="100"
            step="0.01"
            inputmode="decimal"
            :aria-invalid="fieldErrors.percent ? 'true' : undefined"
            :aria-describedby="fieldErrors.percent ? 'discount-percent-error' : undefined"
          />
          <p v-if="fieldErrors.percent" id="discount-percent-error" class="fd-field-error" role="alert">
            {{ fieldErrors.percent }}
          </p>
        </label>

        <template v-if="isFixedAmount">
          <label class="fd-field">
            <span>Amount off</span>
            <input
              id="discount-amount"
              v-model="form.amount"
              type="number"
              min="0.01"
              step="0.01"
              inputmode="decimal"
              :aria-invalid="fieldErrors.amount ? 'true' : undefined"
              :aria-describedby="fieldErrors.amount ? 'discount-amount-error' : undefined"
            />
            <p v-if="fieldErrors.amount" id="discount-amount-error" class="fd-field-error" role="alert">
              {{ fieldErrors.amount }}
            </p>
          </label>

          <label class="fd-field">
            <span>Currency</span>
            <input
              id="discount-currency"
              v-model="form.currency"
              type="text"
              maxlength="3"
              autocomplete="transaction-currency"
              :aria-invalid="fieldErrors.currency ? 'true' : undefined"
              :aria-describedby="fieldErrors.currency ? 'discount-currency-error' : undefined"
            />
            <p v-if="fieldErrors.currency" id="discount-currency-error" class="fd-field-error" role="alert">
              {{ fieldErrors.currency }}
            </p>
          </label>
        </template>

        <fieldset
          class="fd-fieldset"
          :aria-invalid="fieldErrors.appliesTo ? 'true' : undefined"
          :aria-describedby="fieldErrors.appliesTo ? 'discount-applies-to-error' : undefined"
        >
          <legend>Applies to</legend>
          <label v-for="option in SOURCE_OPTIONS" :key="option.value" class="fd-checkbox">
            <input
              type="checkbox"
              :value="option.value"
              :checked="form.appliesTo.includes(option.value)"
              @change="toggleSource(option.value, ($event.target as HTMLInputElement).checked)"
            />
            <span>{{ option.label }}</span>
          </label>
          <p v-if="fieldErrors.appliesTo" id="discount-applies-to-error" class="fd-field-error" role="alert">
            {{ fieldErrors.appliesTo }}
          </p>
        </fieldset>

        <label class="fd-field">
          <span>Starts on</span>
          <input
            id="discount-starts-at"
            v-model="form.startsAt"
            type="date"
            :aria-invalid="fieldErrors.startsAt ? 'true' : undefined"
            :aria-describedby="fieldErrors.startsAt ? 'discount-starts-at-error' : undefined"
          />
          <p v-if="fieldErrors.startsAt" id="discount-starts-at-error" class="fd-field-error" role="alert">
            {{ fieldErrors.startsAt }}
          </p>
        </label>

        <fieldset
          class="fd-fieldset"
          :aria-invalid="fieldErrors.endKind ? 'true' : undefined"
          :aria-describedby="fieldErrors.endKind ? 'discount-end-kind-error' : undefined"
        >
          <legend>When it ends</legend>
          <label class="fd-radio">
            <input v-model="form.endKind" type="radio" value="END_CONDITION_KIND_NO_END" />
            <span>No end date</span>
          </label>
          <label class="fd-radio">
            <input v-model="form.endKind" type="radio" value="END_CONDITION_KIND_END_AFTER_DATE" />
            <span>Ends on a date</span>
          </label>
          <label class="fd-radio">
            <input v-model="form.endKind" type="radio" value="END_CONDITION_KIND_END_AFTER_OCCURRENCES" />
            <span>Ends after a number of uses</span>
          </label>
          <p v-if="fieldErrors.endKind" id="discount-end-kind-error" class="fd-field-error" role="alert">
            {{ fieldErrors.endKind }}
          </p>
        </fieldset>

        <label v-if="form.endKind === 'END_CONDITION_KIND_END_AFTER_DATE'" class="fd-field">
          <span>Ends on</span>
          <input
            id="discount-end-date"
            v-model="form.endDate"
            type="date"
            :aria-invalid="fieldErrors.endDate ? 'true' : undefined"
            :aria-describedby="fieldErrors.endDate ? 'discount-end-date-error' : undefined"
          />
          <p v-if="fieldErrors.endDate" id="discount-end-date-error" class="fd-field-error" role="alert">
            {{ fieldErrors.endDate }}
          </p>
        </label>

        <label v-if="form.endKind === 'END_CONDITION_KIND_END_AFTER_OCCURRENCES'" class="fd-field">
          <!-- "Total uses", not "uses remaining": this is the total the rule
               is configured with. No read path returns a live remaining
               count (T11.3 instruction #3) — see models/discount.ts. -->
          <span>Total uses allowed</span>
          <input
            id="discount-occurrences"
            v-model="form.occurrences"
            type="number"
            min="1"
            step="1"
            inputmode="numeric"
            :aria-invalid="fieldErrors.occurrences ? 'true' : undefined"
            :aria-describedby="fieldErrors.occurrences ? 'discount-occurrences-error' : undefined"
          />
          <p v-if="fieldErrors.occurrences" id="discount-occurrences-error" class="fd-field-error" role="alert">
            {{ fieldErrors.occurrences }}
          </p>
        </label>

        <p v-if="formError" class="fd-field-error" role="alert">{{ formError }}</p>

        <div class="fd-actions">
          <button type="submit" data-testid="create-discount-button" :disabled="creating">
            {{ creating ? 'Creating…' : 'Create discount' }}
          </button>
        </div>
      </form>
    </section>

    <!-- Step 2: list -->
    <section v-else data-testid="discount-list-step" class="fd-step">
      <h2>All discounts</h2>

      <p v-if="loading" role="status">Loading discounts…</p>
      <p v-else-if="error" class="fd-field-error" role="alert">{{ error }}</p>
      <p v-else-if="discounts.length === 0" class="fd-empty">
        No discounts yet. Create one from the “New discount” tab.
      </p>

      <ul v-else class="fd-list">
        <li v-for="rule in discounts" :key="rule.id" class="fd-rule">
          <span class="fd-rule__amount">{{ formatDiscountAmount(rule) }}</span>
          <dl class="fd-rule__details">
            <div class="fd-rule__row">
              <dt>Applies to</dt>
              <dd>{{ formatAppliesTo(rule.appliesTo) }}</dd>
            </div>
            <div class="fd-rule__row">
              <dt>Ends</dt>
              <!-- Never a fabricated date, never a remaining count. -->
              <dd>{{ formatEndCondition(rule.endCondition) }}</dd>
            </div>
          </dl>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
/* Mirrors FacilityOnboarding.vue's `fo-` styles (same tokens, same 44px
   touch targets, same responsive field grid) — renamed to `fd-`. */
.facility-discounts {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 720px;
}

.fd-heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.fd-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
}

.fd-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.fd-steps__item {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink-soft);
  cursor: pointer;
}

.fd-steps__item--active {
  background: var(--court);
  color: var(--paper);
  border-color: var(--court);
}

.fd-step {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.fd-fields {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.fd-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.fd-field input {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

/* Error text: colour is REINFORCEMENT only. The message itself is the
   identification (WCAG 3.3.1) and it carries the fix (3.3.3), so nothing
   here depends on the colour being perceived (1.4.1). */
.fd-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.fd-fieldset {
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  padding: 0.75rem 1rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
}

.fd-fieldset legend {
  font-size: var(--font-size-sm);
  font-weight: 600;
  padding: 0 0.35rem;
}

.fd-radio,
.fd-checkbox {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-height: 44px;
  font-size: var(--font-size-sm);
  cursor: pointer;
}

.fd-radio input,
.fd-checkbox input {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.fd-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.fd-actions button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.fd-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.fd-empty {
  margin: 0;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.fd-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.fd-rule {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem;
  background: var(--paper);
  border-radius: var(--radius-sm);
}

.fd-rule__amount {
  font-weight: 600;
}

.fd-rule__details {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.fd-rule__row {
  display: flex;
  gap: 0.5rem;
}

.fd-rule__row dt {
  font-weight: 600;
}

.fd-rule__row dd {
  margin: 0;
}

/* Two-column field grid on iPad (768-1180px), matching FacilityOnboarding. */
@media (min-width: 768px) {
  .fd-fields {
    grid-template-columns: repeat(2, 1fr);
  }

  .fd-fields .fd-fieldset {
    grid-column: 1 / -1;
  }
}

@media (min-width: 1280px) {
  .facility-discounts {
    max-width: 960px;
  }
}
</style>
