<script setup lang="ts">
// Deep-link landing for a Competition's shareable registration link (T9.7),
// mounted at `/c/:shareToken`. This is the path a link a Host posted outside
// the app lands on, and it is the distinguishing surface of this ticket: the
// visitor may be brand new, arriving with NO prior navigation state, no
// selection, and no idea what this product is.
//
// Three consequences shape this file:
//
// 1. ATTRIBUTION. An entry started here sends `source: ENTRY_SOURCE_SOCIAL`;
//    one started from /competitions sends `ENTRY_SOURCE_APP`. The server
//    validates the claimed value against a closed enum rather than inferring
//    it, so this component is the only place that fact can be recorded. A
//    test asserts the two paths differ on the wire
//    (components/discover-competitions/__tests__/entrySourceAttribution.spec.ts).
//
// 2. EVERY OUTCOME IS A DESIGNED SCREEN. Loading, unknown/expired token,
//    cancelled competition, server error, and the happy path each render
//    real UI — never a blank page, and never a bare 404. In particular a
//    CANCELLED competition still resolves (verified against
//    GetCompetitionByShareToken's contract and its backend test, not
//    assumed): a link published to a social channel outlives the
//    Competition's scheduled state, and "this event was cancelled" and "this
//    link is broken" are different facts that deserve different answers.
//
// 3. KEYBOARD FIRST (WCAG 2.1.1 / 2.4.7). Focus is moved to this screen's
//    own heading on arrival, because there is no prior screen to have set it
//    — every other screen in this app can assume a nav interaction put focus
//    somewhere sensible; this one cannot. Every interactive control here has
//    an explicit visible focus indicator.
import { onMounted, ref, computed, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCompetitionByShareToken } from '../composables/useCompetitionByShareToken'
import { isCancelled } from '../models/competition'
import CompetitionDetailPanel from '../components/discover-competitions/CompetitionDetailPanel.vue'
import type { CompetitionsClient } from '../api/competitionsClient'
import type { IdentityClient } from '../api/identityClient'
import type { FacilitiesClient } from '../api/facilitiesClient'

const props = defineProps<{
  /** Injectable for tests; defaults to the real competitionsClient. */
  client?: CompetitionsClient
  /** Injectable for tests (T10.8, closes #98); defaults to the real
   * identityClient. Forwarded to CompetitionDetailPanel — this landing is
   * exactly the audience T10.8's story names ("a stranger arriving from a
   * social link with no other context"), so the Host name matters here
   * more than anywhere else in the app. */
  identityClient?: IdentityClient
  /** Injectable for tests (T10.8); defaults to the real facilitiesClient.
   * Forwarded to CompetitionDetailPanel to resolve the venue's name. */
  facilitiesClient?: FacilitiesClient
  /** From the `/c/:shareToken` route. Injectable so this view mounts in a
   * test without a router. */
  shareToken?: string
}>()

// `useRoute()` returns undefined without a router plugin installed, so the
// prop is the source of truth in tests and the route supplies it in the app.
const route = useRoute()
// `useRouter()` is likewise undefined in a standalone component test — same
// pattern DiscoverCompetitions.vue/DiscoverGames.vue use. Needed here for
// T10.6's checkout navigation (closes #96): `onPayOnline` below.
const router = useRouter()

const token = computed(() => props.shareToken ?? String(route?.params?.shareToken ?? ''))

const { competition, loading, linkInvalid, error, load } = useCompetitionByShareToken(props.client)

const cancelled = computed(() => competition.value !== null && isCancelled(competition.value.status))

const heading = useTemplateRef<HTMLElement>('heading')

const headingText = computed(() => {
  if (loading.value) return 'Loading this competition…'
  if (linkInvalid.value) return 'This link no longer works'
  if (error.value) return 'We could not load this competition'
  if (cancelled.value) return 'This competition was cancelled'
  return 'You have been invited to a competition'
})

async function resolveToken(): Promise<void> {
  await load(token.value)
  // Focus this screen's own heading once there is something to read. It
  // carries tabindex="-1" so it is programmatically focusable without
  // becoming a tab stop of its own.
  heading.value?.focus()
}

onMounted(() => {
  void resolveToken()
})

// T10.6 (closes #96): mirrors DiscoverCompetitions.vue's identical
// `onPayOnline` — navigates to the checkout route, carrying the
// CompetitionEntry id as a query param. This landing knows the Competition
// id from the already-loaded `competition`, since (unlike
// DiscoverCompetitions.vue) there is no separate "selected id" state here —
// the whole screen is about the one Competition the share link named.
function onPayOnline(entryId: string): void {
  if (!competition.value) return
  void router?.push({ name: 'competition-checkout', params: { id: competition.value.id }, query: { entryId } })
}
</script>

<template>
  <section class="competition-landing" aria-label="Competition invitation">
    <!-- `<section>`, not `<main>` (T11.7 fix — WCAG 1.3.1/landmark
         structure): this view always mounts inside App.vue's own <main
         class="app-shell__main">, so a second <main> here was a NESTED,
         duplicate landmark — invalid regardless of how this screen is
         reached, and specifically flagged by axe-core's
         landmark-main-is-top-level/landmark-no-duplicate-main/
         landmark-unique rules (src/__tests__/accessibility.spec.ts's route
         sweep). The "no prior nav context" reasoning in this file's header
         comment #3 still holds and still matters for focus handling — it
         just doesn't require a second page-level landmark, since the app
         shell already provides the one <main> a page needs; `aria-label`
         above gives this section its own accessible name instead, the same
         `aria-label="<screen>"` convention every other top-level screen
         root already uses (see e.g. GameCheckout.vue/
         CompetitionCheckout.vue). (Kept as the section's first CHILD, not a
         sibling before it — a comment placed before the root element turns
         a single-root Vue 3 template into a multi-root fragment, which
         silently breaks anything that assumes one root DOM node, e.g.
         `wrapper.element` in this file's own component tests.) -->
    <h1 ref="heading" class="competition-landing__heading" tabindex="-1">{{ headingText }}</h1>

    <!-- LOADING: a designed state, matching the text-status loading idiom
         T8.9 established across the Discover surfaces (see
         CompetitionsListPanel.vue's header comment). -->
    <p v-if="loading" class="competition-landing__status" role="status">
      Loading this competition…
    </p>

    <!-- UNKNOWN / EXPIRED TOKEN. The backend deliberately answers the same
         NotFound for an unknown token, a malformed one, and one whose
         Competition is gone — so no caller can probe which tokens are real.
         The honest thing to say is therefore "invalid or expired", plus a
         way onward. -->
    <div v-else-if="linkInvalid" class="competition-landing__panel competition-landing__panel--invalid" role="alert">
      <p class="competition-landing__panel-message">
        This invitation link is invalid or has expired.
      </p>
      <p>
        Links can stop working if they were mistyped or if the competition was removed. You can still
        browse everything that's open right now.
      </p>
      <RouterLink class="competition-landing__cta" to="/competitions">Browse competitions</RouterLink>
    </div>

    <!-- SERVER ERROR — distinct from a bad link, and retryable. -->
    <div v-else-if="error" class="competition-landing__panel competition-landing__panel--error" role="alert">
      <p class="competition-landing__panel-message">{{ error }}</p>
      <button type="button" class="competition-landing__retry" @click="resolveToken">Try again</button>
      <RouterLink class="competition-landing__cta" to="/competitions">Browse competitions</RouterLink>
    </div>

    <template v-else-if="competition">
      <!-- CANCELLED: honest, named, and routed onward — not a 404, and not a
           silently disabled Enter button. The full detail still renders below
           so the visitor can see WHICH competition this was. -->
      <div v-if="cancelled" class="competition-landing__cancelled" role="status">
        <p class="competition-landing__panel-message">
          The host cancelled this competition, so entries are closed.
        </p>
        <RouterLink class="competition-landing__cta" to="/competitions">Browse other competitions</RouterLink>
      </div>

      <CompetitionDetailPanel
        class="competition-landing__detail"
        :competition="competition"
        :loading="false"
        :error="null"
        :has-selection="true"
        source="ENTRY_SOURCE_SOCIAL"
        :client="client"
        :identity-client="identityClient"
        :facilities-client="facilitiesClient"
        @pay-online="onPayOnline"
      />

      <p v-if="!cancelled" class="competition-landing__footer">
        <RouterLink class="competition-landing__link" to="/competitions">
          See all competitions
        </RouterLink>
      </p>
    </template>
  </section>
</template>

<style scoped>
.competition-landing {
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 560px;
  margin: 0 auto;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.competition-landing__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

/* WCAG 2.4.7: the heading takes focus on arrival, so it must show it. */
.competition-landing__heading:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 4px;
}

.competition-landing__status {
  color: var(--ink-soft);
  margin: 0;
}

.competition-landing__panel,
.competition-landing__cancelled {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  align-items: flex-start;
  padding: 1rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper-raised);
}

.competition-landing__panel--invalid,
.competition-landing__cancelled {
  background: var(--pill-critical-bg);
}

.competition-landing__panel--error {
  color: var(--ink-warning);
}

.competition-landing__panel-message {
  margin: 0;
  font-weight: 600;
}

.competition-landing__cta,
.competition-landing__retry {
  font: inherit;
  /* >=44px touch target on iPad/iPhone — links included, since on this
     screen the link IS the primary action. */
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  text-decoration: none;
  cursor: pointer;
}

.competition-landing__link {
  color: var(--court);
  min-height: 44px;
  display: inline-flex;
  align-items: center;
}

/* WCAG 2.4.7 on every interactive control on this screen — it is the one
   screen a keyboard user may reach with no prior navigation state. */
.competition-landing__cta:focus-visible,
.competition-landing__retry:focus-visible,
.competition-landing__link:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 3px;
}

.competition-landing__footer {
  margin: 0;
}

/* iPad/web: wider, not restructured — mirroring GameCheckout.vue's identical
   treatment of a single-column focused screen. */
@media (min-width: 768px) {
  .competition-landing {
    max-width: 720px;
    padding: 2rem 1.5rem;
  }
}
</style>
