<script setup lang="ts">
// Facility onboarding flow (Owner/Host) — T7.4 (docs/process/t7-sprint-plan.md).
// Implements the design review's Flow 1 (docs/design/v1-system-design.md
// §3.1, docs/design/handoff-2026-08/README.md "1. Facility Onboarding")
// against the *real* Facilities API (T7.3), not a mockup.
//
// Step order — name/description/address -> photos -> cameras -> courts —
// matches the reviewed flow exactly. `CreateFacility` only accepts
// name/description/address/photoUrls together in one request (there is no
// separate AddPhoto RPC in T7.3's proto — proto/pickleball/facilities/v1/
// facilities.proto), so the actual `CreateFacility` call fires when the
// user leaves the Photos step (the earliest point every field it needs has
// been collected), not immediately after Details. A field-level error from
// that call (e.g. an empty name) routes the wizard back to the Details step
// so the message renders next to the field it's about — see
// `applyCreateFacilityError` below.
//
// T8.4 CLOSES A PREVIOUSLY-FLAGGED BACKEND GAP: T7.4 originally shipped this
// screen with no API path that could ever set a Facility's
// `cameraConsentAttested` to true, so `submitCameraLink` below always got
// rejected server-side with `domain.ErrCameraConsentRequired` even once the
// checkbox was checked. T8.4 added a dedicated `AttestCameraConsent` RPC
// (proto/pickleball/facilities/v1/facilities.proto) — deliberately separate
// from `AddCameraLink`, per the round-10 design review's requirement that
// consent not be settable at creation time. `submitCameraLink` now calls
// `AttestCameraConsent` first and only proceeds to `AddCameraLink` once that
// succeeds, making the checkbox + form actually work end-to-end for the
// first time.
import { reactive, ref } from 'vue'
import { facilitiesClient, type FacilitiesClient } from '../api/facilitiesClient'
import { useBreakpoint } from '../composables/useBreakpoint'

// No Identity/Users/Auth context exists yet (same caveat class as
// RoleIndicator.vue's mock role data) — the acting owner is a hardcoded
// placeholder id until a real account/session store exists. Also threaded
// through as `actorUserId` on AddCourt/AddCameraLink (T7.7,
// facilities.proto's AddCourtRequest/AddCameraLinkRequest) so those calls'
// caller-claimed identity matches the Facility's ownerId set here on
// CreateFacility — Facility.EnsureOwner 403s on a mismatch/empty actor.
const MOCK_OWNER_ID = 'owner-mock-1'

const STEP_ORDER = ['details', 'photos', 'cameras', 'courts'] as const
type Step = (typeof STEP_ORDER)[number]

const STEP_LABELS: Record<Step, string> = {
  details: 'Details',
  photos: 'Photos',
  cameras: 'Cameras',
  courts: 'Courts',
}

interface FacilityDTO {
  id: string
  ownerId: string
  name: string
  description: string
  address: string
  photoUrls: string[]
  cameraConsentAttested: boolean
  cameraLinks: { url: string; courtId: string }[]
}

interface CourtDTO {
  id: string
  facilityId: string
  name: string
}

const props = withDefaults(
  defineProps<{
    /** Injectable for tests; defaults to the real singleton client. */
    client?: FacilitiesClient
  }>(),
  { client: () => facilitiesClient },
)

const { breakpoint } = useBreakpoint()

const currentStep = ref<Step>('details')
const facility = ref<FacilityDTO | null>(null)
const courts = ref<CourtDTO[]>([])

const form = reactive({
  name: '',
  description: '',
  address: '',
  photoUrls: [] as string[],
  newPhotoUrl: '',
  cameraConsent: false,
  cameraUrl: '',
  newCourtName: '',
})

const fieldErrors = reactive<{ name?: string; address?: string }>({})
const formError = ref('')
const cameraError = ref('')
const courtError = ref('')

/** WCAG 4.1.3 Status Messages: success confirmations go through this ARIA
 * live region (role="status", template below), never only a toast. */
const statusMessage = ref('')

const creating = ref(false)
const addingCamera = ref(false)
const addingCourt = ref(false)

function goToStep(step: Step) {
  // Cameras/Courts operate on an already-created Facility — don't let the
  // step nav jump ahead of that, but Back navigation (Details<->Photos, or
  // from Cameras/Courts once the facility exists) is always allowed and
  // never clears already-entered form data.
  if ((step === 'cameras' || step === 'courts') && !facility.value) return
  currentStep.value = step
}

function addPhotoUrl() {
  const url = form.newPhotoUrl.trim()
  if (!url) return
  form.photoUrls.push(url)
  form.newPhotoUrl = ''
}

function removePhotoUrl(index: number) {
  form.photoUrls.splice(index, 1)
}

function clearFieldErrors() {
  fieldErrors.name = undefined
  fieldErrors.address = undefined
  formError.value = ''
}

// Maps the trailing "<Field>" segment of domain.FieldError's message
// ("<sentinel text>: <Field>", e.g. "facilities: required facility field is
// empty: Name" — see internal/facilities/domain/errors.go's FieldError.Error)
// onto the input it belongs next to. Anything unrecognized (including a
// theoretical OwnerID error, since that field isn't user-facing on this
// screen) falls back to `formError` — still rendered as inline text next to
// the relevant step, never a toast.
const FIELD_MESSAGE_BY_KEY: Record<'name' | 'address', string> = {
  name: 'Facility name is required.',
  address: 'Address is required.',
}

function applyCreateFacilityError(error: unknown) {
  const message = (error as { message?: string } | undefined)?.message ?? ''
  const trailingField = message.split(':').pop()?.trim()
  const key =
    trailingField === 'Name' ? 'name' : trailingField === 'Address' ? 'address' : undefined

  if (key) {
    fieldErrors[key] = FIELD_MESSAGE_BY_KEY[key]
  } else {
    formError.value = message || 'Could not create the facility. Please try again.'
  }
}

async function submitFacility() {
  clearFieldErrors()
  creating.value = true
  try {
    const { data, error } = await props.client.POST('/v1/facilities', {
      body: {
        ownerId: MOCK_OWNER_ID,
        name: form.name,
        description: form.description,
        address: form.address,
        photoUrls: form.photoUrls,
      },
    })

    if (error || !data?.facility) {
      applyCreateFacilityError(error)
      currentStep.value = 'details'
      return
    }

    facility.value = data.facility as FacilityDTO
    statusMessage.value = `Facility "${facility.value.name}" created.`
    currentStep.value = 'cameras'
  } catch {
    formError.value = 'Something went wrong creating the facility. Please try again.'
    currentStep.value = 'details'
  } finally {
    creating.value = false
  }
}

/**
 * Adds the entered camera link. **The consent gate is enforced here, in
 * code, not just via the submit button's `disabled` attribute** — this
 * function returns before calling the API whenever `form.cameraConsent` is
 * false, so even a caller that bypasses the disabled button (e.g. a
 * programmatic call) cannot submit a camera link while consent is
 * unchecked. This is the T7.4 / round-10 §2b acceptance criterion.
 *
 * T8.4: once past that client-side gate, this now calls the real
 * `AttestCameraConsent` RPC first, and only proceeds to `AddCameraLink` once
 * that succeeds — attesting consent and adding a link are two explicit
 * server-side steps (matching `AddCameraLinkRequest`'s own doc comment on
 * why consent isn't a field on that request), not one combined call. A
 * failure attesting consent surfaces inline and never reaches the
 * `AddCameraLink` call.
 */
async function submitCameraLink() {
  cameraError.value = ''

  if (!form.cameraConsent) {
    cameraError.value = 'Check the consent box above before adding a camera link.'
    return
  }
  if (!facility.value) return
  const url = form.cameraUrl.trim()
  if (!url) {
    cameraError.value = 'Enter a camera link URL.'
    return
  }

  addingCamera.value = true
  try {
    const attestResult = await props.client.POST(
      '/v1/facilities/{facilityId}/attestCameraConsent',
      {
        params: { path: { facilityId: facility.value.id } },
        body: { actorUserId: MOCK_OWNER_ID },
      },
    )

    if (attestResult.error || !attestResult.data?.facility) {
      cameraError.value =
        (attestResult.error as { message?: string } | undefined)?.message ??
        'Could not confirm camera consent.'
      return
    }

    facility.value = attestResult.data.facility as FacilityDTO

    const { data, error } = await props.client.POST('/v1/facilities/{facilityId}/cameraLinks', {
      params: { path: { facilityId: facility.value.id } },
      body: { url, actorUserId: MOCK_OWNER_ID },
    })

    if (error || !data?.facility) {
      cameraError.value =
        (error as { message?: string } | undefined)?.message ?? 'Could not add camera link.'
      return
    }

    facility.value = data.facility as FacilityDTO
    form.cameraUrl = ''
    statusMessage.value = 'Camera link added.'
  } finally {
    addingCamera.value = false
  }
}

async function addCourt() {
  courtError.value = ''
  if (!facility.value) return
  const name = form.newCourtName.trim()
  if (!name) return

  addingCourt.value = true
  try {
    const { data, error } = await props.client.POST('/v1/facilities/{facilityId}/courts', {
      params: { path: { facilityId: facility.value.id } },
      body: { name, actorUserId: MOCK_OWNER_ID },
    })

    if (error || !data?.court) {
      courtError.value =
        (error as { message?: string } | undefined)?.message ?? 'Could not add court.'
      return
    }

    courts.value.push(data.court as CourtDTO)
    form.newCourtName = ''
    statusMessage.value = `Court "${data.court.name}" added.`
  } finally {
    addingCourt.value = false
  }
}

// Exposed for tests that need to exercise the consent gate directly
// (bypassing the disabled button), proving the gate is a real code path and
// not just a UI affordance — see FacilityOnboarding.spec.ts.
defineExpose({ submitCameraLink, currentStep })
</script>

<template>
  <div class="facility-onboarding" :data-breakpoint="breakpoint">
    <!-- WCAG 4.1.3 Status Messages: announced to assistive tech whenever
         statusMessage changes, independent of any visual toast. -->
    <p class="fo-status" role="status">{{ statusMessage }}</p>

    <nav class="fo-steps" aria-label="Facility onboarding steps">
      <button
        v-for="step in STEP_ORDER"
        :key="step"
        type="button"
        class="fo-steps__item"
        :class="{ 'fo-steps__item--active': currentStep === step }"
        :disabled="(step === 'cameras' || step === 'courts') && !facility"
        :aria-current="currentStep === step ? 'step' : undefined"
        @click="goToStep(step)"
      >
        {{ STEP_LABELS[step] }}
      </button>
    </nav>

    <!-- Step 1: Details -->
    <section v-if="currentStep === 'details'" data-testid="details-step" class="fo-step">
      <h2>Facility details</h2>
      <div class="fo-fields">
        <label class="fo-field">
          <span>Name</span>
          <input
            id="facility-name"
            v-model="form.name"
            type="text"
            :aria-invalid="fieldErrors.name ? 'true' : undefined"
            :aria-describedby="fieldErrors.name ? 'name-error' : undefined"
          />
          <p v-if="fieldErrors.name" id="name-error" class="fo-field-error" role="alert">
            {{ fieldErrors.name }}
          </p>
        </label>

        <label class="fo-field">
          <span>Description</span>
          <textarea id="facility-description" v-model="form.description" rows="3"></textarea>
        </label>

        <label class="fo-field">
          <span>Address</span>
          <input
            id="facility-address"
            v-model="form.address"
            type="text"
            :aria-invalid="fieldErrors.address ? 'true' : undefined"
            :aria-describedby="fieldErrors.address ? 'address-error' : undefined"
          />
          <p v-if="fieldErrors.address" id="address-error" class="fo-field-error" role="alert">
            {{ fieldErrors.address }}
          </p>
        </label>
      </div>

      <p v-if="formError" class="fo-field-error" role="alert">{{ formError }}</p>

      <div class="fo-actions">
        <button type="button" data-testid="details-next" @click="goToStep('photos')">
          Next: Photos
        </button>
      </div>
    </section>

    <!-- Step 2: Photos -->
    <section v-else-if="currentStep === 'photos'" data-testid="photos-step" class="fo-step">
      <h2>Photos</h2>
      <div class="fo-add-row">
        <label class="fo-field">
          <span>Photo URL</span>
          <input id="facility-photo-url" v-model="form.newPhotoUrl" type="url" />
        </label>
        <button type="button" @click="addPhotoUrl">Add photo</button>
      </div>

      <ul class="fo-list">
        <li v-for="(url, index) in form.photoUrls" :key="url + index">
          {{ url }}
          <button type="button" @click="removePhotoUrl(index)">Remove</button>
        </li>
      </ul>

      <div class="fo-actions">
        <button type="button" data-testid="photos-back" @click="goToStep('details')">Back</button>
        <button
          type="button"
          data-testid="photos-next"
          :disabled="creating"
          @click="submitFacility"
        >
          {{ creating ? 'Creating facility…' : 'Next: Cameras' }}
        </button>
      </div>
    </section>

    <!-- Step 3: Cameras -->
    <section v-else-if="currentStep === 'cameras'" data-testid="cameras-step" class="fo-step">
      <h2>Security cameras (optional)</h2>

      <!-- The checkbox's own click target is the whole label, not an 18px
           box — round-10 §2b's secondary, non-blocking finding, addressed
           here so this ticket doesn't reintroduce the inconsistency. -->
      <label class="fo-consent-label" for="camera-consent-checkbox">
        <input id="camera-consent-checkbox" v-model="form.cameraConsent" type="checkbox" />
        <span>
          I confirm required signage/consent for any security cameras linked here is in place.
        </span>
      </label>

      <label class="fo-field">
        <span>Camera link URL</span>
        <input id="camera-url-input" v-model="form.cameraUrl" type="url" />
      </label>

      <div class="fo-actions">
        <button
          type="button"
          data-testid="add-camera-button"
          :disabled="!form.cameraConsent || !form.cameraUrl.trim() || addingCamera"
          @click="submitCameraLink"
        >
          {{ addingCamera ? 'Adding…' : 'Add camera link' }}
        </button>
      </div>
      <p v-if="cameraError" class="fo-field-error" role="alert">{{ cameraError }}</p>

      <ul v-if="facility?.cameraLinks.length" class="fo-list">
        <li v-for="link in facility.cameraLinks" :key="link.url">{{ link.url }}</li>
      </ul>

      <div class="fo-actions">
        <button type="button" data-testid="cameras-back" @click="goToStep('photos')">Back</button>
        <button type="button" data-testid="cameras-next" @click="goToStep('courts')">
          Next: Courts
        </button>
      </div>
    </section>

    <!-- Step 4: Courts -->
    <section v-else data-testid="courts-step" class="fo-step">
      <h2>Courts</h2>
      <div class="fo-add-row">
        <label class="fo-field">
          <span>Court name</span>
          <input id="court-name-input" v-model="form.newCourtName" type="text" />
        </label>
        <button
          type="button"
          data-testid="add-court-button"
          :disabled="!form.newCourtName.trim() || addingCourt"
          @click="addCourt"
        >
          {{ addingCourt ? 'Adding…' : 'Add court' }}
        </button>
      </div>
      <p v-if="courtError" class="fo-field-error" role="alert">{{ courtError }}</p>

      <ul class="fo-list fo-list--courts">
        <li v-for="court in courts" :key="court.id" class="fo-court">
          <span class="fo-court__name">{{ court.name }}</span>
          <!-- Always "Needs pricing" for a freshly-created court — honest,
               since there's no pricing write path yet (HANDOFF.md T1 gap),
               not a bug. No dead "set price" button. -->
          <span class="fo-badge fo-badge--needs-pricing">Needs pricing</span>
          <p class="fo-court__hint">Pricing setup coming soon.</p>
        </li>
      </ul>

      <div class="fo-actions">
        <button type="button" data-testid="courts-back" @click="goToStep('cameras')">Back</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.facility-onboarding {
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
.fo-status {
  min-height: 1.25rem;
  font-size: var(--font-size-sm);
  color: var(--ink-success);
  font-weight: 600;
}

.fo-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.fo-steps__item {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-pill);
  border: 1px solid var(--hs-border);
  background: var(--paper-raised);
  color: var(--ink-soft);
  cursor: pointer;
}

.fo-steps__item--active {
  background: var(--court);
  color: var(--paper);
  border-color: var(--court);
}

.fo-steps__item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.fo-step {
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.fo-fields {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.fo-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.fo-field input,
.fo-field textarea {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

.fo-field textarea {
  min-height: 5.5rem;
}

.fo-field-error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}

.fo-add-row {
  display: flex;
  align-items: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.fo-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.fo-actions button,
.fo-add-row button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.fo-actions button:disabled,
.fo-add-row button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.fo-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.fo-list li {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: var(--paper);
  border-radius: var(--radius-sm);
}

/* Consent checkbox: the whole label is the hit target (round-10 §2b), not a
   bare 18px box — generously padded, min 44px tall for touch. */
.fo-consent-label {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  min-height: 44px;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper);
  cursor: pointer;
}

.fo-consent-label input[type='checkbox'] {
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  margin-top: 2px;
  accent-color: var(--court);
}

.fo-court {
  flex-direction: column;
  align-items: flex-start !important;
  gap: 0.25rem !important;
}

.fo-court__name {
  font-weight: 600;
}

.fo-court__hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.fo-badge {
  border-radius: var(--radius-pill);
  padding: 0.15rem 0.75rem;
  font-size: var(--font-size-xs);
  font-weight: 600;
}

.fo-badge--needs-pricing {
  background: var(--pill-warning-bg);
  color: var(--pill-fg);
}

/* Two-column field grid on iPad (768-1180px). */
@media (min-width: 768px) {
  .fo-fields {
    grid-template-columns: repeat(2, 1fr);
  }

  .fo-fields .fo-field:first-child {
    grid-column: 1 / -1;
  }
}

/* Multi-column layout on web (>=1280px). */
@media (min-width: 1280px) {
  .facility-onboarding {
    max-width: 960px;
  }

  .fo-fields {
    grid-template-columns: repeat(3, 1fr);
  }

  .fo-fields .fo-field:nth-child(2) {
    grid-column: 1 / -1;
  }
}
</style>
