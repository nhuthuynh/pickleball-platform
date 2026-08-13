<script setup lang="ts">
// Competition creation + advertise flow (Host) — T9.6
// (docs/process/t9-sprint-plan.md). Implements Flow 5 ("Create competition
// -> [link accounts] -> Publish ad -> Manage roster") against the real
// `CreateCompetition` RPC (T9.4) and the real `share_token` it returns
// (T9.5), mounted at `/competitions/new`.
//
// ONE WIZARD IDIOM, NOT TWO. This deliberately reuses T8.8's
// `GameCreation.vue` patterns — the step-chip nav, the per-step
// `<section data-testid="...-step">` shape, the field-level error mapping
// from domain sentinels back onto the step that owns the field, the
// `defineExpose` for handlers whose guard must be provable independently of
// a disabled button — rather than inventing a second multi-step form
// dialect (UX dossier §1). The facility + courts picker is literally the
// same one: `FacilityListPanel` plus `useFacilityList`/`useFacilityDetail`,
// exactly as GameCreation reuses them, and the entry-fee field is now a
// shared component (`components/payments/EntryFeeInput.vue`, extracted from
// GameCreation by this ticket) rather than a second money input.
//
// THE SESSIONS STEP IS THE EXCEPTION, and the reason this context needs its
// own screen at all: a Game has one (starts_at, ends_at, court_ids) triple,
// whereas a Competition "reserves courts across dates" and therefore has a
// LIST of them (see `CompetitionSession`'s proto doc comment). That step is
// an add/remove row editor with its own validation, and is the one part of
// this file that is not modelled on GameCreation.
//
// Step order — Name, venue & courts -> Sessions -> Capacity, guests &
// format -> Payment & entry fee -> Review & publish -> Advertise.
//
// TWO DELIBERATE OMISSIONS, both stated as one-line notes in the UI rather
// than shipped as controls that control nothing (T8.8's precedent, and
// `GameCreation.vue`'s own matching-omission note):
//
//  1. NO MATCHING CONTROLS. `Match` (T10.3-T10.4) records a result; nothing
//     computes `PlayerRating` or a derived Level from it, and there is no
//     matching algorithm anywhere in this codebase — so an auto-match
//     toggle, level-range slider, or gender-mix selector would control
//     nothing. Brackets, rounds, seeding, and results are likewise out of
//     T9's scope (sprint plan §A4) — so there is no seeding UI either.
//     T10.5 upgraded the note's wording to name precisely what's blocked
//     (ADR-0012's Q1/Q2 — the Player Level formula weighting and whether
//     gender-mix matching is in scope), not merely "not available yet" —
//     see `src/copy/matchingDisclosure.ts`.
//
//  2. NO "CONNECT ACCOUNT" BUTTONS, and no per-platform Connected/Connect
//     state. Flow 5's "link accounts" step is replaced by the honest share
//     composer below. This is ADR-0009 (social-channel integration
//     deferred) applied at the UI layer: no OAuth credential custody
//     exists, nothing can post on a Host's behalf, and the platform is
//     OUTBOUND ONLY — it publishes a link, it never reads the channel that
//     link was posted to (see `GetCompetitionByShareToken`'s proto doc
//     comment). A greyed-out "Connect Facebook" row would advertise an
//     integration that does not exist and is not being built; a one-line
//     note that automatic posting isn't available yet is the truth.
//
// Both omissions are pinned down by ABSENCE assertions in the spec, not
// merely by presence assertions on what is here — the only kind of test
// that catches a well-meaning future re-addition.
import { computed, onMounted, reactive, ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'
import { competitionsClient as defaultCompetitionsClient, type CompetitionsClient } from '../api/competitionsClient'
import type { components as CompetitionsComponents } from '../api/generated/competitions'
import { useBreakpoint } from '../composables/useBreakpoint'
import { useFacilityList } from '../composables/useFacilityList'
import { useFacilityDetail } from '../composables/useFacilityDetail'
import { recordHostEvidence } from '../state/roleEvidence'
import { DEFAULT_CURRENCY_CODE, entryFeeLabel } from '../models/payment'
import {
  competitionFormatLabel,
  competitionDatesLabel,
  draftSessionToWire,
  findOverlappingSessionIndices,
  formatSessionRange,
  mapToCompetitionSummary,
  shareUrlForToken,
  type CompetitionSummary,
  type DraftSession,
} from '../models/competition'
import FacilityListPanel from '../components/discover/FacilityListPanel.vue'
import EntryFeeInput from '../components/payments/EntryFeeInput.vue'
import { MATCHING_BLOCKED_REASON } from '../copy/matchingDisclosure'

// No Identity/Users/Auth context exists yet (same caveat class as
// RoleIndicator.vue's mock account, GameCreation.vue's own MOCK_HOST_ID and
// useHostPayments.ts's copy of it) — the acting Host is a hardcoded
// placeholder id until a real account/session store exists. Deliberately
// the same literal value, so a Competition created here belongs to the same
// mock Host as a Game created in GameCreation.vue.
const MOCK_HOST_ID = 'host-mock-1'

const STEP_ORDER = ['venue', 'sessions', 'details', 'payment', 'review'] as const
type Step = (typeof STEP_ORDER)[number]

const STEP_LABELS: Record<Step, string> = {
  venue: 'Name, venue & courts',
  sessions: 'Sessions',
  details: 'Capacity, guests & format',
  payment: 'Payment & entry fee',
  review: 'Review & publish',
}

type PaymentMethodOption = 'online' | 'cash' | 'either'

// The wire type generated from v1PaymentMethod (proto/pickleball/
// competitions/v1/competitions.proto) — PR #91 review finding: this file
// previously widened every wire constant to `string`, which let a typo
// compile silently past the exact union TypeScript exists to catch here.
type CompetitionPaymentMethodWire = NonNullable<
  CompetitionsComponents['schemas']['v1CreateCompetitionRequest']['paymentMethod']
>

const PAYMENT_METHOD_OPTIONS: { value: PaymentMethodOption; label: string }[] = [
  { value: 'online', label: 'Online' },
  { value: 'cash', label: 'Cash' },
  { value: 'either', label: 'Either' },
]

const PAYMENT_METHOD_WIRE: Record<PaymentMethodOption, CompetitionPaymentMethodWire> = {
  online: 'PAYMENT_METHOD_ONLINE',
  cash: 'PAYMENT_METHOD_CASH',
  either: 'PAYMENT_METHOD_EITHER',
}

// Both real values of `CompetitionFormat`. UNSPECIFIED is deliberately not
// offered: the gRPC adapter maps it to an invalid domain.Format so
// NewCompetition rejects it (ErrInvalidFormat/400) precisely to make the
// client state its choice, so a "not sure yet" option here would only
// produce a guaranteed server rejection.
// The wire type generated from v1CompetitionFormat. UNSPECIFIED is excluded
// here (see the comment above FORMAT_OPTIONS) so this is deliberately
// narrower than the full generated union, not a re-derivation of it.
type CompetitionFormatWire = Exclude<
  NonNullable<CompetitionsComponents['schemas']['v1CreateCompetitionRequest']['format']>,
  'COMPETITION_FORMAT_UNSPECIFIED'
>

const FORMAT_OPTIONS: { value: CompetitionFormatWire; label: string }[] = [
  { value: 'COMPETITION_FORMAT_SINGLES', label: 'Singles' },
  { value: 'COMPETITION_FORMAT_DOUBLES', label: 'Doubles' },
]

const props = withDefaults(
  defineProps<{
    /** Injectable for tests; defaults to the real facilitiesClient. */
    client?: FacilitiesClient
    /** Injectable for tests; defaults to the real competitionsClient. */
    competitionsClient?: CompetitionsClient
  }>(),
  { client: () => facilitiesClient, competitionsClient: () => defaultCompetitionsClient },
)

const { breakpoint } = useBreakpoint()

const { facilities, loading: listLoading, error: listError, nameFilter, search } = useFacilityList(props.client)
const {
  facility: facilityDetail,
  loading: detailLoading,
  load: loadFacilityDetail,
} = useFacilityDetail(props.client)

const currentStep = ref<Step>('venue')
const competition = ref<CompetitionSummary | null>(null)
/** The REAL share token from `CreateCompetitionResponse` — the one moment
 * the caller is provably the Host. Held in memory for this flow only: it is
 * a capability, never persisted, never logged, never put in a page title or
 * a route param (see `GetCompetitionByShareTokenRequest`'s proto doc
 * comment). */
const shareToken = ref('')
const published = ref(false)

function blankSession(): DraftSession {
  return { date: '', startTime: '', endTime: '', courtIds: [] }
}

const form = reactive({
  name: '',
  venueFacilityId: '',
  venueFacilityName: '',
  /** The courts this Competition may use at all — the pool each session row
   * picks from. */
  courtIds: [] as string[],
  sessions: [blankSession()] as DraftSession[],
  capacity: 8,
  guestAllowance: 0,
  format: '' as CompetitionFormatWire | '',
  paymentMethod: '' as PaymentMethodOption | '',
  /** Captured in DOLLARS because that is what a Host thinks and types;
   * converted to the integer minor units the wire expects by
   * `entryFeeCents`. Defaults to '0' — a FREE competition, a real product
   * state — rather than an empty box implying it must be priced first. */
  entryFeeDollars: '0' as string | number,
})

interface FieldErrors {
  name?: string
  venue?: string
  courts?: string
  sessions?: string
  capacity?: string
  guestAllowance?: string
  format?: string
  paymentMethod?: string
  entryFee?: string
}

const fieldErrors = reactive<FieldErrors>({})
const formError = ref('')
const publishing = ref(false)

/** WCAG 4.1.3 Status Messages: the publish confirmation goes through this
 * ARIA live region, never only a visual change of screen. */
const publishStatus = ref('')
/** A second, separate live region for the advertise step's copy/share
 * feedback — separate because a clipboard write is its own event a Host
 * triggers repeatedly, and folding it into the publish announcement would
 * either re-announce the publish or overwrite it. */
const advertiseStatus = ref('')

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
  // A session row's courts are a subset of the venue pool, so changing
  // venue invalidates every row's court selection — cleared rather than
  // left pointing at courts that belong to a different facility.
  for (const session of form.sessions) session.courtIds = []
  fieldErrors.venue = undefined
  fieldErrors.courts = undefined
  await loadFacilityDetail(id)
  form.venueFacilityName = facilityDetail.value?.name ?? ''
}

function toggleVenueCourt(courtId: string, event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    if (!form.courtIds.includes(courtId)) form.courtIds.push(courtId)
  } else {
    form.courtIds = form.courtIds.filter((id) => id !== courtId)
    // Dropping a court from the pool must drop it from every row that used
    // it, or the form would submit a court the Host believes they removed.
    for (const session of form.sessions) {
      session.courtIds = session.courtIds.filter((id) => id !== courtId)
    }
  }
}

/** The courts a session row may choose from: the venue pool, resolved to
 * the real `GetFacilityResponse.courts` entries so rows show names, not
 * ids. */
const sessionCourtOptions = computed(() =>
  (facilityDetail.value?.courts ?? []).filter((court) => form.courtIds.includes(court.id)),
)

function toggleSessionCourt(index: number, courtId: string, event: Event) {
  const session = form.sessions[index]
  if (!session) return
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    if (!session.courtIds.includes(courtId)) session.courtIds.push(courtId)
  } else {
    session.courtIds = session.courtIds.filter((id) => id !== courtId)
  }
}

function addSession() {
  form.sessions.push(blankSession())
}

/**
 * Removes session row `index`, except when it is the only one left: a
 * Competition with no sittings is rejected by the domain
 * (`ErrEmptySessions`), so the floor is enforced here as a real code path
 * rather than only by withholding the button — the spec calls this
 * directly to prove it.
 */
function removeSession(index: number) {
  if (form.sessions.length <= 1) return
  form.sessions.splice(index, 1)
}

/**
 * Rows that double-book a court, recomputed on every keystroke — this is
 * what makes the check fire AT ENTRY, next to the offending row, instead of
 * after a round trip that would discard the Host's whole form (NN/g
 * heuristic #5).
 *
 * A UX NICETY MIRRORING T9.1's DOMAIN RULE, NOT A REPLACEMENT FOR IT. The
 * server's `domain.ErrOverlappingSessions` remains authoritative — it is
 * the only check that sees the Competition as actually constructed, and it
 * is still mapped back onto this step by `applyCreateError` below. See
 * `findOverlappingSessionIndices`' doc comment.
 */
const overlappingSessionIndices = computed(() => findOverlappingSessionIndices(form.sessions))

function sessionRowError(index: number): string {
  if (!overlappingSessionIndices.value.has(index)) return ''
  return 'This sitting overlaps another one on the same court. Change its time, date, or court.'
}

/** Rows complete enough to send. An incomplete row is not an error the Host
 * needs shouting about while they're still typing it — it just isn't ready,
 * and `canProceedFromSessions` is what withholds "Next". */
const wireSessions = computed(() =>
  form.sessions.map(draftSessionToWire).filter((s): s is NonNullable<typeof s> => s !== null),
)

const canProceedFromVenue = computed(
  () => form.name.trim() !== '' && form.venueFacilityId !== '' && form.courtIds.length > 0,
)

const canProceedFromSessions = computed(
  () =>
    overlappingSessionIndices.value.size === 0 &&
    form.sessions.length > 0 &&
    // Every row must be complete: a half-filled row would be silently
    // dropped from the payload, publishing a Competition missing a sitting
    // the Host thought they had entered.
    form.sessions.every(
      (s) => s.date !== '' && s.startTime !== '' && s.endTime !== '' && s.courtIds.length > 0,
    ) &&
    // Mirrors domain.ErrInvalidTimeRange for each row.
    form.sessions.every((s) => s.startTime < s.endTime),
)

const canProceedFromDetails = computed(
  () => form.capacity > 0 && form.guestAllowance >= 0 && form.format !== '',
)

function decrementGuestAllowance() {
  if (form.guestAllowance > 0) form.guestAllowance -= 1
}

function incrementGuestAllowance() {
  form.guestAllowance += 1
}

const entryFeeCents = computed(() => {
  const raw = String(form.entryFeeDollars ?? '').trim()
  if (raw === '') return 0
  const parsed = Number(raw)
  if (Number.isNaN(parsed)) return Number.NaN
  return Math.round(parsed * 100)
})

const entryFeeValid = computed(() => !Number.isNaN(entryFeeCents.value) && entryFeeCents.value >= 0)

function goToStep(step: Step) {
  currentStep.value = step
}

function clearFieldErrors() {
  fieldErrors.name = undefined
  fieldErrors.venue = undefined
  fieldErrors.courts = undefined
  fieldErrors.sessions = undefined
  fieldErrors.capacity = undefined
  fieldErrors.guestAllowance = undefined
  fieldErrors.format = undefined
  fieldErrors.paymentMethod = undefined
  fieldErrors.entryFee = undefined
  formError.value = ''
}

/**
 * Maps `CreateCompetition`'s domain sentinel messages
 * (internal/competitions/domain/errors.go) onto the specific step + field
 * they're about (WCAG 3.3.1), and routes the wizard back to that step so
 * the message is actually visible — the same approach
 * `GameCreation.vue`'s `applyCreateGameError` takes, matched by substring
 * because Competitions' domain errors are plain sentinel strings.
 *
 * Note that the overlap and court-unavailable rejections both land on the
 * sessions step: that is where the Host chose the times and courts, and it
 * is the concrete demonstration that the client-side overlap check is an
 * early warning rather than a substitute for the server's ruling.
 */
function applyCreateError(error: unknown) {
  const message = (error as { message?: string } | undefined)?.message ?? ''

  if (message.includes('sessions overlap on the same court')) {
    fieldErrors.sessions = 'Two of these sittings book the same court at overlapping times.'
    currentStep.value = 'sessions'
  } else if (message.includes('court is unavailable for the requested time')) {
    fieldErrors.sessions =
      'A court is unavailable at one of these times — it is already booked. Change the time or the court.'
    currentStep.value = 'sessions'
  } else if (message.includes('at least one session is required')) {
    fieldErrors.sessions = 'Add at least one sitting.'
    currentStep.value = 'sessions'
  } else if (message.includes('at least one court id is required')) {
    fieldErrors.sessions = 'Every sitting needs at least one court.'
    currentStep.value = 'sessions'
  } else if (message.includes('invalid time range')) {
    fieldErrors.sessions = 'A sitting ends before it starts — check its times.'
    currentStep.value = 'sessions'
  } else if (message.includes('capacity must be greater than zero')) {
    fieldErrors.capacity = 'Capacity must be greater than 0.'
    currentStep.value = 'details'
  } else if (message.includes('guest allowance must not be negative')) {
    fieldErrors.guestAllowance = 'Guest allowance cannot be negative.'
    currentStep.value = 'details'
  } else if (message.includes('invalid format')) {
    fieldErrors.format = 'Choose singles or doubles.'
    currentStep.value = 'details'
  } else if (message.includes('facility not found')) {
    fieldErrors.venue = 'This facility could not be found. Pick a different one.'
    currentStep.value = 'venue'
  } else if (message.includes('invalid payment method')) {
    fieldErrors.paymentMethod = 'Choose a payment method.'
    currentStep.value = 'payment'
  } else if (message.includes('invalid money value')) {
    fieldErrors.entryFee = "Entry fee can't be negative."
    currentStep.value = 'payment'
  } else {
    formError.value = message || 'Could not publish the competition. Please try again.'
    currentStep.value = 'review'
  }
}

async function publishCompetition() {
  clearFieldErrors()
  if (!entryFeeValid.value) {
    fieldErrors.entryFee = "Entry fee can't be negative."
    currentStep.value = 'payment'
    return
  }
  publishing.value = true
  try {
    const { data, error } = await props.competitionsClient.POST('/v1/competitions', {
      body: {
        hostId: MOCK_HOST_ID,
        name: form.name.trim(),
        venueFacilityId: form.venueFacilityId,
        sessions: wireSessions.value,
        capacity: form.capacity,
        guestAllowance: form.guestAllowance,
        paymentMethod: form.paymentMethod ? PAYMENT_METHOD_WIRE[form.paymentMethod] : undefined,
        format: form.format || undefined,
        // int64 on the wire is protojson-encoded as a string. A FREE
        // competition sends an explicit 0 with the launch currency, never
        // an omitted field: "free" is a value the Host chose.
        entryFee: {
          amountCents: String(entryFeeCents.value),
          currencyCode: DEFAULT_CURRENCY_CODE,
        },
      },
    })

    if (error || !data?.competition) {
      applyCreateError(error)
      return
    }

    competition.value = mapToCompetitionSummary(data.competition)
    // The token exists only on this response. If a server ever omitted it,
    // `shareUrlForToken` yields '' and the promo renders without a link
    // rather than with a broken one.
    shareToken.value = data.shareToken ?? ''
    publishStatus.value = `${competition.value.name} published. Automated matching isn't available yet — players enter directly. ${MATCHING_BLOCKED_REASON}`
    // Kickoff note decision #1: a successful CreateCompetition is real
    // evidence this session has hosted, so RoleIndicator can list "Host".
    // This is the EXISTING T8.8 mechanism (state/roleEvidence.ts) gaining a
    // second real signal — deliberately not a second, parallel mechanism.
    recordHostEvidence()
    published.value = true
  } catch {
    formError.value = 'Something went wrong publishing the competition. Please try again.'
    currentStep.value = 'review'
  } finally {
    publishing.value = false
  }
}

/** The origin the share link is built against — the app's own, injectable
 * only through the ambient `window` (no configuration knob: a link to
 * anywhere other than this deployment would not resolve). */
const shareUrl = computed(() =>
  shareUrlForToken(shareToken.value, typeof window !== 'undefined' ? window.location.origin : ''),
)

/**
 * The ready-to-post promo. EVERY line comes from the real
 * `CreateCompetition` response or the values the Host actually submitted —
 * there is no invented venue blurb, no fabricated "join 200 players
 * nearby", and no placeholder URL. Anything with no backend home simply
 * isn't in it.
 */
const promoText = computed(() => {
  const c = competition.value
  if (!c) return ''
  const lines = [
    `${c.name} — ${competitionFormatLabel(c.format)}`,
    competitionDatesLabel(c.sessions),
    form.venueFacilityName || c.venueFacilityId,
    `Entry: ${entryFeeLabel(c.entryFeeCents)}`,
    `${c.capacity} spots`,
  ].filter((line) => line !== '')

  if (shareUrl.value) lines.push(`Enter here: ${shareUrl.value}`)
  return lines.join('\n')
})

/** Whether this browser actually has the Web Share API. Resolved once, at
 * setup: where it's absent the button is not rendered at all, rather than
 * rendered disabled — a control that can never do anything is exactly the
 * dead affordance this ticket is about avoiding. */
const canWebShare = ref(
  typeof navigator !== 'undefined' && typeof (navigator as Navigator).share === 'function',
)

async function copyPromo() {
  advertiseStatus.value = ''
  try {
    if (!navigator?.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(promoText.value)
    // WCAG 4.1.3 + NN/g heuristic #1: a clipboard write with no feedback is
    // invisible — the confirmation is announced, not merely implied.
    advertiseStatus.value = 'Copied. Paste it wherever you want to share this competition.'
  } catch {
    // Never claim a success that didn't happen: the same live region says
    // so, and the promo text is selectable on screen as the fallback.
    advertiseStatus.value = 'Could not copy automatically — select the text above and copy it.'
  }
}

async function sharePromo() {
  try {
    await (navigator as Navigator).share({
      title: competition.value?.name ?? 'Competition',
      text: promoText.value,
      url: shareUrl.value,
    })
    advertiseStatus.value = 'Shared.'
  } catch {
    // A cancelled share sheet is a normal, deliberate user action — not an
    // error to shout about, so the region is simply cleared.
    advertiseStatus.value = ''
  }
}

const paymentMethodLabel = computed(
  () => PAYMENT_METHOD_OPTIONS.find((o) => o.value === form.paymentMethod)?.label ?? 'Not selected',
)

onMounted(() => {
  void search()
})

// Exposed for tests that need to exercise handlers directly, bypassing a
// withheld/disabled control — the same "prove the gate is a real code path"
// reasoning GameCreation.vue's own defineExpose documents.
defineExpose({ publishCompetition, currentStep, removeSession, addSession, decrementGuestAllowance })
</script>

<template>
  <div class="competition-creation" :data-breakpoint="breakpoint">
    <!-- WCAG 4.1.3: announced whenever publishStatus changes. -->
    <p class="cc-status" role="status" data-testid="publish-status">{{ publishStatus }}</p>

    <!-- Step 6: Advertise. Replaces Flow 5's "link accounts" step — see the
         file header for why there is no account-linking UI here. -->
    <section v-if="published" data-testid="advertise-step" class="cc-step">
      <h2>{{ competition?.name }} is published</h2>
      <p class="cc-hint">Here's a ready-to-post promo with its registration link.</p>

      <pre class="cc-promo" data-testid="promo-text">{{ promoText }}</pre>

      <div class="cc-actions">
        <button type="button" data-testid="copy-promo" @click="copyPromo">Copy promo</button>
        <button v-if="canWebShare" type="button" data-testid="share-promo" @click="sharePromo">
          Share…
        </button>
      </div>

      <!-- WCAG 4.1.3 Status Messages: the copy result is announced, not
           left as an invisible clipboard side effect. -->
      <p class="cc-status" role="status" data-testid="advertise-status">{{ advertiseStatus }}</p>

      <!-- ADR-0009 at the UI layer. An honest one-line note, NOT a row of
           greyed-out "Connect Facebook / Connect WhatsApp" buttons: no
           OAuth credential custody exists and nothing can post on a Host's
           behalf, so a control here would advertise an integration that
           isn't being built. -->
      <p class="cc-note" data-testid="auto-post-note">
        Posting this automatically to your accounts isn't available yet — copy the promo and
        paste it wherever you like.
      </p>

      <p class="cc-note" data-testid="matching-note">
        Automated matching isn't available yet — players enter directly. {{ MATCHING_BLOCKED_REASON }}
      </p>

      <div class="cc-actions">
        <RouterLink
          v-if="competition"
          class="cc-link-button"
          :to="`/competitions/${competition.id}/manage`"
          data-testid="manage-link"
        >
          Manage roster
        </RouterLink>
      </div>
    </section>

    <template v-else>
      <nav class="cc-steps" aria-label="Create a competition steps">
        <button
          v-for="step in STEP_ORDER"
          :key="step"
          type="button"
          class="cc-steps__item"
          :class="{ 'cc-steps__item--active': currentStep === step }"
          :aria-current="currentStep === step ? 'step' : undefined"
          @click="goToStep(step)"
        >
          {{ STEP_LABELS[step] }}
        </button>
      </nav>

      <!-- Step 1: Name, venue & courts -->
      <section v-if="currentStep === 'venue'" data-testid="venue-step" class="cc-step">
        <h2>Name, venue & courts</h2>

        <div class="cc-field">
          <label for="competition-name">Competition name</label>
          <input
            id="competition-name"
            v-model="form.name"
            data-testid="competition-name"
            type="text"
            :aria-describedby="fieldErrors.name ? 'competition-name-error' : undefined"
          />
          <p v-if="fieldErrors.name" id="competition-name-error" class="cc-field-error" role="alert">
            {{ fieldErrors.name }}
          </p>
        </div>

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
        <p v-if="fieldErrors.venue" id="venue-error" class="cc-field-error" role="alert">
          {{ fieldErrors.venue }}
        </p>

        <div v-if="form.venueFacilityId" class="cc-courts">
          <h3>Courts this competition can use</h3>
          <p v-if="detailLoading" role="status">Loading courts…</p>
          <p v-else-if="(facilityDetail?.courts.length ?? 0) === 0" class="cc-hint">
            This facility hasn't listed any courts yet.
          </p>
          <ul v-else class="cc-court-list">
            <li v-for="court in facilityDetail?.courts ?? []" :key="court.id">
              <label class="cc-court-option" data-testid="venue-court-option">
                <input
                  type="checkbox"
                  :value="court.id"
                  :checked="form.courtIds.includes(court.id)"
                  @change="toggleVenueCourt(court.id, $event)"
                />
                <span>{{ court.name }}</span>
              </label>
            </li>
          </ul>
          <p v-if="fieldErrors.courts" id="courts-error" class="cc-field-error" role="alert">
            {{ fieldErrors.courts }}
          </p>
        </div>

        <div class="cc-actions">
          <button
            type="button"
            data-testid="venue-next"
            :disabled="!canProceedFromVenue"
            @click="goToStep('sessions')"
          >
            Next: Sessions
          </button>
        </div>
      </section>

      <!-- Step 2: Sessions. THE step that is not a copy of GameCreation —
           a Competition spans dates, so this is a row editor, not one
           range. -->
      <section v-else-if="currentStep === 'sessions'" data-testid="sessions-step" class="cc-step">
        <h2>Sessions</h2>
        <p class="cc-hint">
          Add every sitting this competition runs. Each one reserves its courts for that time.
        </p>

        <ul class="cc-session-list">
          <li
            v-for="(session, index) in form.sessions"
            :key="index"
            class="cc-session-row"
            data-testid="session-row"
          >
            <div class="cc-session-fields">
              <label class="cc-field">
                <span>Date</span>
                <input
                  v-model="session.date"
                  :data-testid="`session-date-${index}`"
                  type="date"
                  :aria-invalid="sessionRowError(index) ? 'true' : undefined"
                  :aria-describedby="sessionRowError(index) ? `session-error-${index}` : undefined"
                />
              </label>
              <label class="cc-field">
                <span>Starts</span>
                <input
                  v-model="session.startTime"
                  :data-testid="`session-start-${index}`"
                  type="time"
                />
              </label>
              <label class="cc-field">
                <span>Ends</span>
                <input v-model="session.endTime" :data-testid="`session-end-${index}`" type="time" />
              </label>
            </div>

            <fieldset class="cc-session-courts" :data-testid="`session-court-${index}`">
              <legend>Courts</legend>
              <p v-if="sessionCourtOptions.length === 0" class="cc-hint">
                Go back and pick at least one court for this competition.
              </p>
              <label v-for="court in sessionCourtOptions" :key="court.id" class="cc-court-option">
                <input
                  type="checkbox"
                  :value="court.id"
                  :checked="session.courtIds.includes(court.id)"
                  @change="toggleSessionCourt(index, court.id, $event)"
                />
                <span>{{ court.name }}</span>
              </label>
            </fieldset>

            <!-- WCAG 3.3.1: the conflict is named in text, beside the row
                 that causes it, as soon as it is typed — not as a
                 form-level banner and not after a submit. -->
            <p
              v-if="sessionRowError(index)"
              :id="`session-error-${index}`"
              :data-testid="`session-error-${index}`"
              class="cc-field-error"
              role="alert"
            >
              {{ sessionRowError(index) }}
            </p>

            <button
              v-if="form.sessions.length > 1"
              type="button"
              class="cc-remove-session"
              :data-testid="`remove-session-${index}`"
              @click="removeSession(index)"
            >
              Remove this session
            </button>
          </li>
        </ul>

        <div class="cc-actions">
          <button type="button" data-testid="add-session" @click="addSession">
            Add another session
          </button>
        </div>

        <p v-if="fieldErrors.sessions" data-testid="sessions-form-error" class="cc-field-error" role="alert">
          {{ fieldErrors.sessions }}
        </p>

        <div class="cc-actions">
          <button type="button" data-testid="sessions-back" @click="goToStep('venue')">Back</button>
          <button
            type="button"
            data-testid="sessions-next"
            :disabled="!canProceedFromSessions"
            @click="goToStep('details')"
          >
            Next: Capacity & format
          </button>
        </div>
      </section>

      <!-- Step 3: Capacity, guest allowance & format -->
      <section v-else-if="currentStep === 'details'" data-testid="details-step" class="cc-step">
        <h2>Capacity, guests & format</h2>

        <div class="cc-field">
          <label for="competition-capacity">Capacity (total places)</label>
          <input
            id="competition-capacity"
            v-model.number="form.capacity"
            data-testid="capacity-input"
            type="number"
            min="1"
            :aria-describedby="fieldErrors.capacity ? 'capacity-error' : 'capacity-hint'"
          />
          <!-- Stated because the domain counts PLACES, not entrants: an
               entrant and each of their guests occupy one each. -->
          <p id="capacity-hint" class="cc-hint">
            Each entrant and each of their guests takes one place.
          </p>
          <p v-if="fieldErrors.capacity" id="capacity-error" class="cc-field-error" role="alert">
            {{ fieldErrors.capacity }}
          </p>
        </div>

        <div class="cc-stepper" role="group" aria-labelledby="guest-allowance-label">
          <span id="guest-allowance-label">Guests allowed per entry</span>
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
        <p v-if="fieldErrors.guestAllowance" id="guest-allowance-error" class="cc-field-error" role="alert">
          {{ fieldErrors.guestAllowance }}
        </p>

        <fieldset class="cc-radio-group">
          <legend>Format</legend>
          <label v-for="option in FORMAT_OPTIONS" :key="option.value" class="cc-radio-option">
            <input v-model="form.format" type="radio" name="format" :value="option.value" />
            <span>{{ option.label }}</span>
          </label>
        </fieldset>
        <!-- Honest about what this field is: descriptive, enforcing
             nothing (see domain.Format's doc comment). Saying so here is
             what stops it reading as a promise of pairing or brackets. -->
        <p class="cc-hint">Shown to players as a label. Entries aren't paired or scheduled by it.</p>
        <p v-if="fieldErrors.format" id="format-error" class="cc-field-error" role="alert">
          {{ fieldErrors.format }}
        </p>

        <div class="cc-actions">
          <button type="button" data-testid="details-back" @click="goToStep('sessions')">Back</button>
          <button
            type="button"
            data-testid="details-next"
            :disabled="!canProceedFromDetails"
            @click="goToStep('payment')"
          >
            Next: Payment
          </button>
        </div>
      </section>

      <!-- Step 4: Payment method & entry fee -->
      <section v-else-if="currentStep === 'payment'" data-testid="payment-step" class="cc-step">
        <h2>Payment & entry fee</h2>
        <fieldset class="cc-radio-group">
          <legend>How will entrants pay?</legend>
          <label v-for="option in PAYMENT_METHOD_OPTIONS" :key="option.value" class="cc-radio-option">
            <input
              v-model="form.paymentMethod"
              type="radio"
              name="payment-method"
              :value="option.value"
            />
            <span>{{ option.label }}</span>
          </label>
        </fieldset>
        <p v-if="fieldErrors.paymentMethod" id="payment-method-error" class="cc-field-error" role="alert">
          {{ fieldErrors.paymentMethod }}
        </p>

        <!-- T9.2's fee input, shared with GameCreation.vue. -->
        <EntryFeeInput
          v-model="form.entryFeeDollars"
          subject="competition"
          :server-error="fieldErrors.entryFee ?? ''"
        />

        <div class="cc-actions">
          <button type="button" data-testid="payment-back" @click="goToStep('details')">Back</button>
          <button
            type="button"
            data-testid="payment-next"
            :disabled="!form.paymentMethod || !entryFeeValid"
            @click="goToStep('review')"
          >
            Next: Review
          </button>
        </div>
      </section>

      <!-- Step 5: Review & publish -->
      <section v-else data-testid="review-step" class="cc-step">
        <h2>Review & publish</h2>
        <dl class="cc-review-summary">
          <dt>Name</dt>
          <dd>{{ form.name }}</dd>
          <dt>Facility</dt>
          <dd>{{ form.venueFacilityName || form.venueFacilityId }}</dd>
          <dt>Sessions</dt>
          <dd>
            <ul class="cc-review-sessions">
              <li v-for="(session, index) in wireSessions" :key="index">
                {{ formatSessionRange(session.startsAt, session.endsAt) }} ·
                {{ session.courtIds.length }} court{{ session.courtIds.length === 1 ? '' : 's' }}
              </li>
            </ul>
          </dd>
          <dt>Capacity</dt>
          <dd>{{ form.capacity }}</dd>
          <dt>Guests per entry</dt>
          <dd>{{ form.guestAllowance }}</dd>
          <dt>Format</dt>
          <dd>{{ competitionFormatLabel(form.format) }}</dd>
          <dt>Payment</dt>
          <dd>{{ paymentMethodLabel }}</dd>
          <dt>Entry fee</dt>
          <dd data-testid="review-entry-fee">
            {{ entryFeeValid ? entryFeeLabel(entryFeeCents) : '—' }}
          </dd>
        </dl>

        <p class="cc-note" data-testid="matching-note">
          Automated matching isn't available yet — players enter directly. {{ MATCHING_BLOCKED_REASON }}
        </p>

        <p v-if="formError" class="cc-field-error" role="alert">{{ formError }}</p>

        <div class="cc-actions">
          <button type="button" data-testid="review-back" @click="goToStep('payment')">Back</button>
          <button
            type="button"
            data-testid="publish-button"
            :disabled="publishing"
            @click="publishCompetition"
          >
            {{ publishing ? 'Publishing…' : 'Publish competition' }}
          </button>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.competition-creation {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  padding: 1.5rem 1rem;
  /* iPhone <600px: a single stacked column. */
  max-width: 720px;
}

/* Live regions: always in the layout so they can be announced; empty text
   collapses visually without display:none (which would strip them from the
   accessibility tree in some AT/browser combinations). */
.cc-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
  margin: 0;
}

.cc-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.cc-steps__item {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink-soft);
  cursor: pointer;
}

.cc-steps__item--active {
  background: var(--court);
  color: var(--paper);
  border-color: var(--court);
}

.cc-step {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cc-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.cc-field input {
  font: inherit;
  /* >=44px touch targets on iPad/iPhone (Apple HIG). */
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

.cc-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.cc-hint {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
  margin: 0;
}

.cc-note {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  background: var(--paper);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  padding: 0.65rem 0.85rem;
  margin: 0;
}

.cc-courts,
.cc-court-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.cc-court-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.cc-court-option {
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

.cc-court-option input[type='checkbox'] {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.cc-session-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cc-session-row {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.85rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper);
}

.cc-session-fields {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.75rem;
}

.cc-session-courts {
  border: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.cc-session-courts legend {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  padding: 0;
  margin-bottom: 0.35rem;
}

.cc-remove-session {
  font: inherit;
  min-height: 44px;
  align-self: flex-start;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink);
  cursor: pointer;
}

.cc-radio-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border: none;
  padding: 0;
  margin: 0;
}

.cc-radio-group legend {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  padding: 0;
  margin-bottom: 0.35rem;
}

.cc-radio-option {
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

.cc-radio-option input[type='radio'] {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  accent-color: var(--court);
}

.cc-stepper {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.cc-stepper span {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.cc-stepper button {
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

.cc-stepper button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.cc-stepper output {
  min-width: 2.5rem;
  text-align: center;
  font-weight: 600;
  font-size: var(--font-size-lg);
}

.cc-review-summary {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.35rem 1rem;
  margin: 0;
}

.cc-review-summary dt {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.cc-review-summary dd {
  margin: 0;
  font-weight: 600;
}

.cc-review-sessions {
  list-style: none;
  margin: 0;
  padding: 0;
}

.cc-promo {
  margin: 0;
  padding: 0.85rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper);
  color: var(--ink);
  font-family: var(--font-family-ui);
  font-size: var(--font-size-sm);
  /* The promo is user-selectable text, so "copy it yourself" is a real
     fallback when the clipboard API is unavailable or refused. */
  white-space: pre-wrap;
  word-break: break-word;
}

.cc-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cc-actions button,
.cc-link-button {
  font: inherit;
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
  text-decoration: none;
}

.cc-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* iPad (768-1180px): the session row's date/start/end sit side by side, and
   the form gets a two-column feel rather than one long stack. */
@media (min-width: 768px) {
  .cc-session-fields {
    grid-template-columns: repeat(3, 1fr);
  }

  .cc-session-courts {
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
  }
}

/* Web (>=1280px): wider measure alongside the persistent sidebar. */
@media (min-width: 1280px) {
  .competition-creation {
    max-width: 960px;
  }
}
</style>
