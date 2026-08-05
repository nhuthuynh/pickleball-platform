<script setup lang="ts">
// Social Game creation flow (Host/Owner) — T8.8
// (docs/process/t8-sprint-plan.md). Implements Flow 3 (design handoff
// README, `docs/design/handoff-2026-08/README.md`) — "configure and
// publish a social game" — against the real `CreateGame` RPC (T8.7's
// extended request: venue_facility_id, payment_method, guest_allowance),
// replacing T8.1's `/games/new` placeholder route.
//
// Step order — Facility & courts -> date/time & capacity -> payment
// method -> guest allowance -> review & publish — mirrors Flow 3's own
// order MINUS the "Matching" step. **This is a deliberate, documented
// omission, not a truncated form**: there is no Match/PlayerRating
// aggregate or matching algorithm anywhere in this codebase (T8 kickoff
// note's re-scope finding), so an auto-match toggle / level-range slider /
// gender-mix selector would control nothing. The review step below carries
// an honest one-line note instead — same "coming soon note, not a dead
// control" approach T7.4 already used for not-yet-built court pricing.
//
// Facility & courts step reuses T7.5's existing facility-browse pieces
// where the shape actually overlaps: `FacilityListPanel.vue` for the
// facility search/select UI, and `useFacilityList`/`useFacilityDetail` for
// the network calls — rather than duplicating a second facility picker
// from scratch, per this ticket's own instructions. There is no
// player-facing "book a court" behavior wanted here, so
// `FacilityDetailPanel.vue` itself (which mixes in booking/manual-entry
// UI this flow doesn't need) is deliberately not reused — only the two
// composables plus the list panel are.
//
// Error handling (WCAG 3.3.1, same rule T7.4's FacilityOnboarding already
// applies): `applyCreateGameError` maps each of `CreateGame`'s domain
// sentinel error messages (internal/socialplay/domain/errors.go) onto the
// specific step + field it's about, so e.g. a capacity<=0 rejection
// surfaces as text next to the Capacity field on the Date/time/capacity
// step, not a generic toast — and routes the wizard back to that step so
// the message is actually visible.
import { computed, onMounted, reactive, ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'
import { socialplayClient, type SocialPlayClient } from '../api/socialplayClient'
import { useBreakpoint } from '../composables/useBreakpoint'
import { useFacilityList } from '../composables/useFacilityList'
import { useFacilityDetail } from '../composables/useFacilityDetail'
import { recordHostEvidence } from '../state/roleEvidence'
import { DEFAULT_CURRENCY_CODE } from '../models/payment'
import { entryFeeLabel } from '../models/game'
import FacilityListPanel from '../components/discover/FacilityListPanel.vue'

// No Identity/Users/Auth context exists yet (same caveat class as
// RoleIndicator.vue's mock account and FacilityOnboarding.vue's
// MOCK_OWNER_ID) — the acting Host is a hardcoded placeholder id until a
// real account/session store exists.
const MOCK_HOST_ID = 'host-mock-1'

const STEP_ORDER = ['venue', 'schedule', 'payment', 'guests', 'review'] as const
type Step = (typeof STEP_ORDER)[number]

const STEP_LABELS: Record<Step, string> = {
  venue: 'Facility & courts',
  schedule: 'Date, time & capacity',
  payment: 'Payment method',
  guests: 'Guest allowance',
  review: 'Review & publish',
}

type PaymentMethodOption = 'online' | 'cash' | 'either'

const PAYMENT_METHOD_OPTIONS: { value: PaymentMethodOption; label: string }[] = [
  { value: 'online', label: 'Online' },
  { value: 'cash', label: 'Cash' },
  { value: 'either', label: 'Either' },
]

type PaymentMethodWire = 'PAYMENT_METHOD_ONLINE' | 'PAYMENT_METHOD_CASH' | 'PAYMENT_METHOD_EITHER'

const PAYMENT_METHOD_WIRE: Record<PaymentMethodOption, PaymentMethodWire> = {
  online: 'PAYMENT_METHOD_ONLINE',
  cash: 'PAYMENT_METHOD_CASH',
  either: 'PAYMENT_METHOD_EITHER',
}

interface GameDTO {
  id: string
  hostId: string
  venueFacilityId: string
  courtIds: string[]
  startsAt: string
  endsAt: string
  capacity: number
  paymentMethod: string
  guestAllowance: number
  entryFee?: { amountCents?: string; currencyCode?: string }
}

const props = withDefaults(
  defineProps<{
    /** Injectable for tests; defaults to the real facilitiesClient. */
    client?: FacilitiesClient
    /** Injectable for tests; defaults to the real socialplayClient. */
    socialClient?: SocialPlayClient
  }>(),
  { client: () => facilitiesClient, socialClient: () => socialplayClient },
)

const { breakpoint } = useBreakpoint()

const { facilities, loading: listLoading, error: listError, nameFilter, search } = useFacilityList(props.client)
const {
  facility: facilityDetail,
  loading: detailLoading,
  load: loadFacilityDetail,
} = useFacilityDetail(props.client)

const currentStep = ref<Step>('venue')
const game = ref<GameDTO | null>(null)
const published = ref(false)

const form = reactive({
  venueFacilityId: '',
  venueFacilityName: '',
  courtIds: [] as string[],
  startsAt: '',
  endsAt: '',
  capacity: 4,
  paymentMethod: '' as PaymentMethodOption | '',
  guestAllowance: 0,
  // The Host's real entry fee (T9.2), captured in DOLLARS because that is
  // what a Host thinks and types; converted to the integer minor units the
  // wire expects by `entryFeeCents` below. Defaults to '0' — a free Game,
  // a real product state — rather than to an empty box implying the Host
  // must price the Game before they can publish it.
  entryFeeDollars: '0' as string | number,
})

interface FieldErrors {
  venue?: string
  courts?: string
  timeRange?: string
  capacity?: string
  paymentMethod?: string
  guestAllowance?: string
  entryFee?: string
}

const fieldErrors = reactive<FieldErrors>({})
const formError = ref('')

/** WCAG 4.1.3 Status Messages: success confirmations go through this ARIA
 * live region (role="status", template below), never only a toast. */
const statusMessage = ref('')

const publishing = ref(false)

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

async function onSelectFacility(id: string) {
  form.venueFacilityId = id
  form.courtIds = []
  fieldErrors.venue = undefined
  fieldErrors.courts = undefined
  await loadFacilityDetail(id)
  form.venueFacilityName = facilityDetail.value?.name ?? ''
}

function toggleCourt(courtId: string, event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    if (!form.courtIds.includes(courtId)) form.courtIds.push(courtId)
  } else {
    form.courtIds = form.courtIds.filter((id) => id !== courtId)
  }
}

const canProceedFromVenue = computed(() => form.venueFacilityId !== '' && form.courtIds.length > 0)
const canProceedFromSchedule = computed(
  () => form.startsAt !== '' && form.endsAt !== '' && form.capacity > 0,
)

function decrementGuestAllowance() {
  if (form.guestAllowance > 0) form.guestAllowance -= 1
}

function incrementGuestAllowance() {
  form.guestAllowance += 1
}

/**
 * The typed fee in integer minor units (T9.2). Rounds rather than
 * truncates so a Host typing "12.005" cannot silently lose a cent, and
 * returns NaN for input that isn't a number at all so the caller can tell
 * "not a valid amount" apart from "zero".
 */
const entryFeeCents = computed(() => {
  // Coerce: a number <input> hands back a string in the browser but a
  // number when set programmatically (and by @vue/test-utils' setValue).
  const raw = String(form.entryFeeDollars ?? '').trim()
  if (raw === '') return 0
  const parsed = Number(raw)
  if (Number.isNaN(parsed)) return Number.NaN
  return Math.round(parsed * 100)
})

/** Client-side mirror of domain.Money.Validate's negative-amount rule. The
 * server is still authoritative (ErrInvalidMoney -> 400, mapped by
 * applyCreateGameError below); this only gets the message in front of the
 * Host sooner. */
const entryFeeValid = computed(() => !Number.isNaN(entryFeeCents.value) && entryFeeCents.value >= 0)

/**
 * The fee error actually shown next to the field. Two sources, one message
 * slot: the LIVE client-side check (so a Host who types a negative amount
 * is told why immediately — leaving them staring at a disabled "Next" with
 * no explanation would fail WCAG 3.3.1 just as surely as showing nothing
 * at all), and the server's own ErrInvalidMoney rejection routed here by
 * applyCreateGameError.
 */
const entryFeeError = computed(() => {
  if (!entryFeeValid.value) return "Entry fee can't be negative."
  return fieldErrors.entryFee ?? ''
})

function goToStep(step: Step) {
  currentStep.value = step
}

function clearFieldErrors() {
  fieldErrors.venue = undefined
  fieldErrors.courts = undefined
  fieldErrors.timeRange = undefined
  fieldErrors.capacity = undefined
  fieldErrors.paymentMethod = undefined
  fieldErrors.guestAllowance = undefined
  fieldErrors.entryFee = undefined
  formError.value = ''
}

// Maps CreateGame's domain sentinel error messages
// (internal/socialplay/domain/errors.go) onto the specific step + field
// they're about (WCAG 3.3.1) — mirrors FacilityOnboarding.vue's
// applyCreateFacilityError, but matched by substring rather than a
// trailing "<Field>" segment, since socialplay's domain errors are plain
// sentinel strings (no structured FieldError type on this context, unlike
// facilities.domain.FieldError).
function applyCreateGameError(error: unknown) {
  const message = (error as { message?: string } | undefined)?.message ?? ''

  if (message.includes('capacity must be greater than zero')) {
    fieldErrors.capacity = 'Capacity must be greater than 0.'
    currentStep.value = 'schedule'
  } else if (message.includes('invalid time range')) {
    fieldErrors.timeRange = 'End time must be after the start time.'
    currentStep.value = 'schedule'
  } else if (message.includes('court unavailable for the requested time range')) {
    fieldErrors.timeRange = 'A selected court is unavailable at that time — pick a different time or court.'
    currentStep.value = 'schedule'
  } else if (message.includes('at least one court id is required')) {
    fieldErrors.courts = 'Select at least one court.'
    currentStep.value = 'venue'
  } else if (message.includes('venue facility not found')) {
    fieldErrors.venue = 'This facility could not be found. Pick a different one.'
    currentStep.value = 'venue'
  } else if (message.includes('invalid payment method')) {
    fieldErrors.paymentMethod = 'Choose a payment method.'
    currentStep.value = 'payment'
  } else if (message.includes('invalid money amount')) {
    fieldErrors.entryFee = "Entry fee can't be negative."
    currentStep.value = 'payment'
  } else if (message.includes('guest allowance must not be negative')) {
    fieldErrors.guestAllowance = 'Guest allowance cannot be negative.'
    currentStep.value = 'guests'
  } else {
    formError.value = message || 'Could not publish the game. Please try again.'
    currentStep.value = 'review'
  }
}

async function publishGame() {
  clearFieldErrors()
  if (!entryFeeValid.value) {
    fieldErrors.entryFee = "Entry fee can't be negative."
    currentStep.value = 'payment'
    return
  }
  publishing.value = true
  try {
    const startsAt = form.startsAt
    const endsAt = form.endsAt
    const paymentMethod = form.paymentMethod

    const { data, error } = await props.socialClient.POST('/v1/games', {
      body: {
        hostId: MOCK_HOST_ID,
        venueFacilityId: form.venueFacilityId,
        courtIds: form.courtIds,
        startsAt: startsAt ? new Date(startsAt).toISOString() : undefined,
        endsAt: endsAt ? new Date(endsAt).toISOString() : undefined,
        capacity: form.capacity,
        paymentMethod: paymentMethod ? PAYMENT_METHOD_WIRE[paymentMethod] : undefined,
        guestAllowance: form.guestAllowance,
        // int64 on the wire is protojson-encoded as a string — same
        // convention models/payment.ts's toMoneyRequest documents. A free
        // Game sends an explicit 0 with the launch currency, never an
        // omitted field: "free" is a value the Host chose.
        entryFee: {
          amountCents: String(entryFeeCents.value),
          currencyCode: DEFAULT_CURRENCY_CODE,
        },
      },
    })

    if (error || !data?.game) {
      applyCreateGameError(error)
      return
    }

    game.value = data.game as GameDTO
    statusMessage.value =
      'Game published. Automated matching isn’t available yet — players join directly.'
    // T8.8 (kickoff note decision #1): a successful CreateGame is real
    // evidence this session has hosted a game, so RoleIndicator can now
    // list "Host" — see src/state/roleEvidence.ts's header comment for why
    // this only ever fires on an actual successful response, never
    // speculatively.
    recordHostEvidence()
    published.value = true
  } catch {
    formError.value = 'Something went wrong publishing the game. Please try again.'
    currentStep.value = 'review'
  } finally {
    publishing.value = false
  }
}

function resetForNewGame() {
  form.venueFacilityId = ''
  form.venueFacilityName = ''
  form.courtIds = []
  form.startsAt = ''
  form.endsAt = ''
  form.capacity = 4
  form.paymentMethod = ''
  form.guestAllowance = 0
  form.entryFeeDollars = '0'
  game.value = null
  published.value = false
  statusMessage.value = ''
  clearFieldErrors()
  currentStep.value = 'venue'
}

const paymentMethodLabel = computed(
  () => PAYMENT_METHOD_OPTIONS.find((o) => o.value === form.paymentMethod)?.label ?? 'Not selected',
)

onMounted(() => {
  void search()
})

// Exposed for tests that need to exercise handlers directly (bypassing a
// disabled button's DOM state, the same "prove the gate is a real code
// path" reasoning FacilityOnboarding.vue's defineExpose already
// documents) — decrementGuestAllowance in particular, to prove the
// guest-allowance stepper's minimum-of-0 floor is enforced in the handler
// itself, not just via the decrement button's `disabled` attribute.
defineExpose({ publishGame, currentStep, decrementGuestAllowance, incrementGuestAllowance })
</script>

<template>
  <div class="game-creation" :data-breakpoint="breakpoint">
    <!-- WCAG 4.1.3 Status Messages: announced to assistive tech whenever
         statusMessage changes, independent of any visual toast. -->
    <p class="gc-status" role="status">{{ statusMessage }}</p>

    <section v-if="published" data-testid="published-step" class="gc-step">
      <h2>Game published</h2>
      <p>Players can join directly — automated matching isn't available yet.</p>
      <dl class="gc-review-summary">
        <dt>Game ID</dt>
        <dd>{{ game?.id }}</dd>
      </dl>
      <div class="gc-actions">
        <button type="button" data-testid="create-another" @click="resetForNewGame">
          Create another game
        </button>
      </div>
    </section>

    <template v-else>
      <nav class="gc-steps" aria-label="Create a game steps">
        <button
          v-for="step in STEP_ORDER"
          :key="step"
          type="button"
          class="gc-steps__item"
          :class="{ 'gc-steps__item--active': currentStep === step }"
          :aria-current="currentStep === step ? 'step' : undefined"
          @click="goToStep(step)"
        >
          {{ STEP_LABELS[step] }}
        </button>
      </nav>

      <!-- Step 1: Facility & courts -->
      <section v-if="currentStep === 'venue'" data-testid="venue-step" class="gc-step">
        <h2>Facility & courts</h2>
        <FacilityListPanel
          :facilities="facilities"
          :loading="listLoading"
          :error="listError"
          :selected-id="form.venueFacilityId || null"
          :name-filter="nameFilter"
          @update:name-filter="onNameFilterInput"
          @search="onSearchSubmit"
          @select="onSelectFacility"
        />
        <p v-if="fieldErrors.venue" id="venue-error" class="gc-field-error" role="alert">
          {{ fieldErrors.venue }}
        </p>

        <div v-if="form.venueFacilityId" class="gc-courts">
          <h3>Courts</h3>
          <p v-if="detailLoading" role="status">Loading courts…</p>
          <p v-else-if="(facilityDetail?.courts.length ?? 0) === 0" class="gc-hint">
            This facility hasn't listed any courts yet.
          </p>
          <ul v-else class="gc-court-list">
            <li v-for="court in facilityDetail?.courts ?? []" :key="court.id">
              <label class="gc-court-option">
                <input
                  type="checkbox"
                  :value="court.id"
                  :checked="form.courtIds.includes(court.id)"
                  @change="toggleCourt(court.id, $event)"
                />
                <span>{{ court.name }}</span>
              </label>
            </li>
          </ul>
          <p v-if="fieldErrors.courts" id="courts-error" class="gc-field-error" role="alert">
            {{ fieldErrors.courts }}
          </p>
        </div>

        <div class="gc-actions">
          <button
            type="button"
            data-testid="venue-next"
            :disabled="!canProceedFromVenue"
            @click="goToStep('schedule')"
          >
            Next: Date & time
          </button>
        </div>
      </section>

      <!-- Step 2: Date, time & capacity -->
      <section v-else-if="currentStep === 'schedule'" data-testid="schedule-step" class="gc-step">
        <h2>Date, time & capacity</h2>
        <div class="gc-fields">
          <label class="gc-field">
            <span>Starts at</span>
            <input id="game-starts-at" v-model="form.startsAt" type="datetime-local" />
          </label>

          <label class="gc-field">
            <span>Ends at</span>
            <input id="game-ends-at" v-model="form.endsAt" type="datetime-local" />
          </label>

          <label class="gc-field">
            <span>Capacity</span>
            <input id="game-capacity" v-model.number="form.capacity" type="number" min="1" />
          </label>
        </div>
        <p v-if="fieldErrors.timeRange" id="time-range-error" class="gc-field-error" role="alert">
          {{ fieldErrors.timeRange }}
        </p>
        <p v-if="fieldErrors.capacity" id="capacity-error" class="gc-field-error" role="alert">
          {{ fieldErrors.capacity }}
        </p>

        <div class="gc-actions">
          <button type="button" data-testid="schedule-back" @click="goToStep('venue')">Back</button>
          <button
            type="button"
            data-testid="schedule-next"
            :disabled="!canProceedFromSchedule"
            @click="goToStep('payment')"
          >
            Next: Payment method
          </button>
        </div>
      </section>

      <!-- Step 3: Payment method (3-way: Online / Cash / Either) -->
      <section v-else-if="currentStep === 'payment'" data-testid="payment-step" class="gc-step">
        <h2>Payment method</h2>
        <fieldset class="gc-payment-options">
          <legend>How will players pay?</legend>
          <label v-for="option in PAYMENT_METHOD_OPTIONS" :key="option.value" class="gc-radio-option">
            <input
              v-model="form.paymentMethod"
              type="radio"
              name="payment-method"
              :value="option.value"
            />
            <span>{{ option.label }}</span>
          </label>
        </fieldset>
        <p v-if="fieldErrors.paymentMethod" id="payment-method-error" class="gc-field-error" role="alert">
          {{ fieldErrors.paymentMethod }}
        </p>

        <!-- Entry fee (T9.2), replacing T8.10's flat placeholder rate.
             Leaving it at 0 publishes a FREE game — a real choice, spelled
             out in the hint text so it never reads as an unfinished
             field. -->
        <div class="gc-field">
          <label for="entry-fee">Entry fee per player</label>
          <input
            id="entry-fee"
            v-model="form.entryFeeDollars"
            data-testid="entry-fee-input"
            type="number"
            min="0"
            step="0.01"
            inputmode="decimal"
            :aria-invalid="entryFeeError ? 'true' : undefined"
            :aria-describedby="entryFeeError ? 'entry-fee-error' : 'entry-fee-hint'"
          />
          <!-- WCAG 1.4.1: the "this game is free" state is stated in text,
               not implied by an empty or differently-styled field. -->
          <p id="entry-fee-hint" class="gc-hint">
            Leave at 0 to make this game free.<template v-if="entryFeeValid">
              Players will see “{{ entryFeeLabel(entryFeeCents) }}”.</template>
          </p>
          <!-- WCAG 3.3.1 Error Identification: the validation error is
               visible text next to the field, never color alone. -->
          <p v-if="entryFeeError" id="entry-fee-error" class="gc-field-error" role="alert">
            {{ entryFeeError }}
          </p>
        </div>

        <div class="gc-actions">
          <button type="button" data-testid="payment-back" @click="goToStep('schedule')">Back</button>
          <button
            type="button"
            data-testid="payment-next"
            :disabled="!form.paymentMethod || !entryFeeValid"
            @click="goToStep('guests')"
          >
            Next: Guest allowance
          </button>
        </div>
      </section>

      <!-- Step 4: Guest allowance (stepper, minimum 0) -->
      <section v-else-if="currentStep === 'guests'" data-testid="guests-step" class="gc-step">
        <h2>Guest allowance</h2>
        <div class="gc-stepper" role="group" aria-labelledby="guest-allowance-label">
          <span id="guest-allowance-label">Guests allowed per registration</span>
          <button
            type="button"
            data-testid="guest-allowance-decrement"
            aria-label="Decrease guest allowance"
            :disabled="form.guestAllowance <= 0"
            @click="decrementGuestAllowance"
          >
            −
          </button>
          <output data-testid="guest-allowance-value">{{ form.guestAllowance }}</output>
          <button
            type="button"
            data-testid="guest-allowance-increment"
            aria-label="Increase guest allowance"
            @click="incrementGuestAllowance"
          >
            +
          </button>
        </div>
        <p v-if="fieldErrors.guestAllowance" id="guest-allowance-error" class="gc-field-error" role="alert">
          {{ fieldErrors.guestAllowance }}
        </p>

        <div class="gc-actions">
          <button type="button" data-testid="guests-back" @click="goToStep('payment')">Back</button>
          <button type="button" data-testid="guests-next" @click="goToStep('review')">
            Next: Review & publish
          </button>
        </div>
      </section>

      <!-- Step 5: Review & publish. NOTE: Flow 3's "Matching & publish"
           step is deliberately just "Publish" here — see file header. -->
      <section v-else data-testid="review-step" class="gc-step">
        <h2>Review & publish</h2>
        <dl class="gc-review-summary">
          <dt>Facility</dt>
          <dd>{{ form.venueFacilityName || form.venueFacilityId }}</dd>
          <dt>Courts</dt>
          <dd>{{ form.courtIds.length }} selected</dd>
          <dt>Starts</dt>
          <dd>{{ form.startsAt }}</dd>
          <dt>Ends</dt>
          <dd>{{ form.endsAt }}</dd>
          <dt>Capacity</dt>
          <dd>{{ form.capacity }}</dd>
          <dt>Payment method</dt>
          <dd>{{ paymentMethodLabel }}</dd>
          <dt>Guest allowance</dt>
          <dd>{{ form.guestAllowance }}</dd>
          <dt>Entry fee</dt>
          <dd data-testid="review-entry-fee">{{ entryFeeValid ? entryFeeLabel(entryFeeCents) : '—' }}</dd>
        </dl>

        <!-- T8.8: "Matching & publish" -> just "Publish". No auto-match
             toggle, level-range slider, or gender-mix selector anywhere in
             this form (there is no Match/PlayerRating backend to attach
             one to) — an honest note instead of a dead control. -->
        <p class="gc-matching-note" data-testid="matching-note">
          Automated matching isn't available yet — players join directly.
        </p>

        <p v-if="formError" class="gc-field-error" role="alert">{{ formError }}</p>

        <div class="gc-actions">
          <button type="button" data-testid="review-back" @click="goToStep('guests')">Back</button>
          <button
            type="button"
            data-testid="publish-button"
            :disabled="publishing"
            @click="publishGame"
          >
            {{ publishing ? 'Publishing…' : 'Publish game' }}
          </button>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.game-creation {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 720px;
}

/* Status live region: small, always in the layout (so it can receive focus
   context / be announced), not a floating toast. Empty text collapses
   visually without display:none (which would strip it from the
   accessibility tree in some AT/browser combinations). */
.gc-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
}

.gc-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.gc-steps__item {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink-soft);
  cursor: pointer;
}

.gc-steps__item--active {
  background: var(--court);
  color: var(--paper);
  border-color: var(--court);
}

.gc-step {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.gc-fields {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.gc-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.gc-field input {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

.gc-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.gc-hint {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
  margin: 0;
}

.gc-courts {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.gc-court-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.gc-court-option {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper);
  cursor: pointer;
}

.gc-court-option input[type='checkbox'] {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.gc-payment-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border: none;
  padding: 0;
  margin: 0;
}

.gc-payment-options legend {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  padding: 0;
  margin-bottom: 0.35rem;
}

.gc-radio-option {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-height: 44px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper);
  cursor: pointer;
}

.gc-radio-option input[type='radio'] {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.gc-stepper {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.gc-stepper span {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.gc-stepper button {
  font: inherit;
  min-width: 44px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
  font-size: var(--font-size-lg);
  line-height: 1;
}

.gc-stepper button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.gc-stepper output {
  min-width: 2.5rem;
  text-align: center;
  font-weight: 600;
  font-size: var(--font-size-lg);
}

.gc-review-summary {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.35rem 1rem;
  margin: 0;
}

.gc-review-summary dt {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.gc-review-summary dd {
  margin: 0;
  font-weight: 600;
}

.gc-matching-note {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  background: var(--paper);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  padding: 0.65rem 0.85rem;
  margin: 0;
}

.gc-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.gc-actions button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.gc-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Two-column field grid on iPad (768-1180px). */
@media (min-width: 768px) {
  .gc-fields {
    grid-template-columns: repeat(2, 1fr);
  }

  .gc-fields .gc-field:first-child {
    grid-column: 1 / -1;
  }
}

/* Multi-column layout on web (>=1280px). */
@media (min-width: 1280px) {
  .game-creation {
    max-width: 960px;
  }

  .gc-fields {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
