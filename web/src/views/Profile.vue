<script setup lang="ts">
// Profile screen (T10.5, docs/process/t10-sprint-plan.md), mounted at
// `/profile` (replacing the `ComingSoonView` placeholder T8.1 wired there —
// see router/index.ts's own note on why that route existed with nothing
// behind it yet).
//
// MINIMAL BY DESIGN, per this ticket's own scope:
//  - `DisplayName` is READ-ONLY here. No rename flow — out of scope for
//    this ticket (T10.5 instructions #1).
//  - `SelfReportedStartingLevel` is the one editable field, via the real
//    `UpdateSelfReportedLevel` RPC (T10.2) — a plain 1..5 self-assessment
//    <select>, not a range/slider control.
//  - NO MATCHING UI OF ANY KIND. No auto-match toggle, no level-RANGE
//    slider (this screen sets one value, not a range), no gender-mix
//    selector, no "find a match" button — there is no Match/PlayerRating
//    computation or matching algorithm anywhere in this codebase to attach
//    one to (ADR-0012). The note below states precisely why, reusing
//    `src/copy/matchingDisclosure.ts` so this screen's wording can't drift
//    from GameCreation.vue/CompetitionCreation.vue/CompetitionManage.vue's
//    own copies of the same explanation.
//
// NO SIGN-UP FLOW. This ticket builds no `CreateUser` UI — see
// `useProfile.ts`'s header comment for why `MOCK_PLAYER_ID` (the same
// placeholder `useJoinGame.ts`/`useEnterCompetition.ts` already use, not a
// second one) resolves to the honest "no profile yet" state against the
// real backend today, and why that is correct given this ticket's scope
// rather than a bug.
//
// NO FABRICATED DATA (T10.5 instructions #4): an unset (or, defensively,
// out-of-range) starting level renders as "Not set", never a default value
// like "Level 1" that would read as a real, player-made choice — see
// `models/identity.ts`'s `mapToUserProfile`.
import { computed, onMounted, ref } from 'vue'
import { useProfile, MOCK_PLAYER_ID } from '../composables/useProfile'
import type { IdentityClient } from '../api/identityClient'
import { MIN_SELF_REPORTED_LEVEL, MAX_SELF_REPORTED_LEVEL } from '../models/identityLevel'
import { MATCHING_BLOCKED_REASON } from '../copy/matchingDisclosure'

const props = defineProps<{
  /** Injectable for tests; defaults to the real identityClient. */
  client?: IdentityClient
}>()

const { profile, loading, error, notFound, saving, saveError, load, updateLevel } = useProfile(props.client)

const LEVEL_OPTIONS = Array.from(
  { length: MAX_SELF_REPORTED_LEVEL - MIN_SELF_REPORTED_LEVEL + 1 },
  (_, i) => MIN_SELF_REPORTED_LEVEL + i,
)

/** The <select>'s own bound value — '' means "no selection made in the
 * form yet" (distinct from the profile's actual saved level, which may
 * itself be unset — see selectedLevel's initialiser below). */
const selectedLevel = ref<number | ''>('')

/** WCAG 4.1.3 Status Messages: a save confirmation is announced here,
 * independent of any visual-only styling change. */
const statusMessage = ref('')

function syncSelectFromProfile(): void {
  selectedLevel.value = profile.value?.selfReportedStartingLevel ?? ''
}

async function refresh(): Promise<void> {
  await load(MOCK_PLAYER_ID)
  syncSelectFromProfile()
}

const canSave = computed(() => selectedLevel.value !== '')

async function onSave(): Promise<void> {
  if (selectedLevel.value === '') return
  statusMessage.value = ''
  await updateLevel(MOCK_PLAYER_ID, MOCK_PLAYER_ID, selectedLevel.value)
  if (!saveError.value) {
    statusMessage.value = 'Starting level saved.'
    syncSelectFromProfile()
  }
}

onMounted(() => {
  void refresh()
})

defineExpose({ onSave, refresh })
</script>

<template>
  <section class="profile" aria-label="Profile">
    <h1 class="profile__heading">Profile</h1>

    <p v-if="loading" class="profile__status" role="status">Loading your profile…</p>

    <div v-else-if="error" class="profile__status profile__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="profile__retry" @click="refresh">Try again</button>
    </div>

    <p v-else-if="notFound" class="profile__empty" data-testid="profile-not-found">
      We don't have a profile for this account yet.
    </p>

    <template v-else-if="profile">
      <dl class="profile__fields">
        <dt>Display name</dt>
        <dd data-testid="display-name">{{ profile.displayName }}</dd>

        <dt>Self-reported starting level</dt>
        <dd v-if="profile.selfReportedStartingLevel !== null" data-testid="level-value">
          Level {{ profile.selfReportedStartingLevel }}
        </dd>
        <dd v-else data-testid="level-empty-state" class="profile__level-current">Not set</dd>
      </dl>

      <form class="profile__level-form" @submit.prevent="onSave">
        <label class="profile__level-label" for="self-reported-level">Update your starting level</label>
        <p class="profile__level-hint">
          Your own 1-5 assessment of your starting level, used to seed matching once it exists.
          {{ MATCHING_BLOCKED_REASON }}
        </p>
        <select id="self-reported-level" v-model.number="selectedLevel" class="profile__level-select">
          <option value="" disabled>Choose a level</option>
          <option v-for="level in LEVEL_OPTIONS" :key="level" :value="level">Level {{ level }}</option>
        </select>

        <div class="profile__actions">
          <button type="submit" data-testid="save-level" :disabled="!canSave || saving">
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </div>

        <p class="profile__status" role="status">{{ statusMessage }}</p>
        <p v-if="saveError" class="profile__status profile__status--error" role="alert">{{ saveError }}</p>
      </form>
    </template>
  </section>
</template>

<style scoped>
.profile {
  font-family: var(--font-family-ui);
  color: var(--ink);
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  max-width: 560px;
}

.profile__heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.profile__status {
  min-height: 1.25rem;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
  margin: 0;
}

.profile__status--error {
  color: var(--ink-warning);
}

.profile__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.profile__empty {
  color: var(--ink-soft);
}

.profile__fields {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.35rem 1rem;
  margin: 0;
}

.profile__fields dt {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.profile__fields dd {
  margin: 0;
  font-weight: 600;
}

.profile__level-form {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-lg);
  background: var(--paper-raised);
  padding: 1.25rem;
}

.profile__level-label {
  font-weight: 600;
}

.profile__level-hint {
  margin: 0;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.profile__level-current {
  margin: 0;
  color: var(--ink-soft);
  font-style: italic;
}

.profile__level-select {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
  max-width: 220px;
}

.profile__actions {
  display: flex;
}

.profile__actions button {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--court);
  background: var(--court);
  color: var(--paper);
  cursor: pointer;
}

.profile__actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
