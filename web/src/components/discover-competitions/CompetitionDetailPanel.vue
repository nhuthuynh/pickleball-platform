<script setup lang="ts">
// Competition detail view (T9.7 requirement #1): host, venue, EVERY
// session's date/time/courts, format, spots left, entry fee (or "Free"), and
// payment method. Purely presentational, driven by the already-loaded
// `competition` prop — mirrors GameDetailPanel.vue's split, with the entry
// flow itself (network-calling) delegated to CompetitionEntryPanel.vue.
//
// Reused by BOTH arrival paths — the in-app browse screen
// (DiscoverCompetitions.vue) and the deep-link landing
// (views/CompetitionLanding.vue) — which is why it takes `source` as a prop
// rather than deciding attribution itself, and why it stays router-free: the
// landing owns its own "browse instead" navigation.
import CompetitionEntryPanel from './CompetitionEntryPanel.vue'
import type { CompetitionSummary, EntrySource } from '../../models/competition'
import {
  competitionFormatLabel,
  entryFeeLabel,
  formatSessionRangeFromSession,
  isCancelled,
  paymentMethodLabel,
  sessionCourtsLabel,
  spotsLeftLabel,
} from '../../models/competition'
import type { CompetitionsClient } from '../../api/competitionsClient'

const props = defineProps<{
  competition: CompetitionSummary | null
  loading: boolean
  error: string | null
  /** Whether a competition is currently selected at all (vs. no selection
   * yet — iPad/web only; iPhone never renders this without a selection) —
   * mirrors GameDetailPanel.vue's identical prop. The landing always passes
   * true. */
  hasSelection: boolean
  source: EntrySource
  /** Injectable for tests; defaults to the real competitionsClient. Passed
   * through to CompetitionEntryPanel. */
  client?: CompetitionsClient
}>()

const emit = defineEmits<{ retry: [] }>()

/** A cancelled Competition cannot be entered, so no entry form is rendered
 * for one — not a disabled button, which would leave a visitor guessing why
 * it doesn't work (NN/g heuristic #9). The landing adds a route onward. */
function cancelled(): boolean {
  return props.competition !== null && isCancelled(props.competition.status)
}
</script>

<template>
  <section class="competition-detail" aria-label="Competition details">
    <p v-if="!hasSelection" class="competition-detail__status competition-detail__status--placeholder">
      Select a competition to see its details.
    </p>

    <!-- Loading -->
    <p v-else-if="loading" class="competition-detail__status competition-detail__status--loading" role="status">
      Loading competition details…
    </p>

    <!-- Error: API unreachable, or a stale selection that no longer resolves
         to a real Competition. -->
    <div v-else-if="error" class="competition-detail__status competition-detail__status--error" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="competition-detail__retry" @click="emit('retry')">Try again</button>
    </div>

    <div v-else-if="competition" class="competition-detail__content">
      <h2 class="competition-detail__heading">{{ competition.name || 'Untitled competition' }}</h2>

      <p v-if="cancelled()" class="competition-detail__cancelled-note">
        This competition has been cancelled, so entries are closed.
      </p>

      <dl class="competition-detail__fields">
        <div class="competition-detail__field">
          <dt>Host</dt>
          <dd>{{ competition.hostId }}</dd>
        </div>
        <div class="competition-detail__field">
          <dt>Venue</dt>
          <dd>{{ competition.venueFacilityId || 'No venue set' }}</dd>
        </div>
        <div class="competition-detail__field">
          <!-- Descriptive only: CompetitionFormat enforces nothing
               server-side (see competitions.proto's enum comment). -->
          <dt>Format</dt>
          <dd>{{ competitionFormatLabel(competition.format) }}</dd>
        </div>
        <!-- Spots left is rendered ONLY when the API actually supplied it.
             GetCompetition and GetCompetitionByShareToken return a bare
             Competition with no spots_left field, so `null` here means "this
             endpoint doesn't tell us" — and an omitted row is honest where an
             invented number would not be. -->
        <div v-if="competition.spotsLeft !== null" class="competition-detail__field" data-testid="competition-detail-spots">
          <!-- WCAG 1.4.1: urgency is words, never a coloured number. -->
          <dt>Spots left</dt>
          <dd>{{ spotsLeftLabel(competition.spotsLeft) }}</dd>
        </div>
        <div class="competition-detail__field">
          <!-- WCAG 1.4.1: "Cash at facility" / "Online" / "Online or cash"
               is text, never a colour-only cue. -->
          <dt>Payment</dt>
          <dd>{{ paymentMethodLabel(competition.paymentMethod) }}</dd>
        </div>
        <div class="competition-detail__field">
          <!-- A free competition reads as the word "Free", never "$0.00" and
               never a blank — a zero fee is a real state a Host chose. -->
          <dt>Entry fee</dt>
          <dd data-testid="competition-detail-entry-fee">{{ entryFeeLabel(competition.entryFeeCents) }}</dd>
        </div>
      </dl>

      <!-- Every session, each with its own date/time and courts. A
           Competition runs across dates — this list IS the difference
           between a Competition and a Game. -->
      <section class="competition-detail__sessions-block" aria-label="Sessions">
        <h3 class="competition-detail__subheading">
          Sessions ({{ competition.sessions.length }})
        </h3>
        <ol v-if="competition.sessions.length > 0" class="competition-detail__sessions">
          <li v-for="(session, index) in competition.sessions" :key="index" class="competition-detail__session">
            <span class="competition-detail__session-time">{{ formatSessionRangeFromSession(session) }}</span>
            <span class="competition-detail__session-courts">
              {{ session.courtIds.length === 1 ? 'Court' : 'Courts' }}: {{ sessionCourtsLabel(session) }}
            </span>
          </li>
        </ol>
        <p v-else class="competition-detail__status">No sessions listed for this competition.</p>
      </section>

      <CompetitionEntryPanel
        v-if="!cancelled()"
        :key="competition.id"
        :competition="competition"
        :source="source"
        :client="client"
      />
    </div>
  </section>
</template>

<style scoped>
.competition-detail {
  font-family: var(--font-family-ui);
  color: var(--ink);
}

.competition-detail__status {
  color: var(--ink-soft);
}

.competition-detail__status--error {
  color: var(--ink-warning);
}

.competition-detail__retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.competition-detail__retry:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 2px;
}

.competition-detail__heading {
  font-size: var(--font-size-2xl);
  margin: 0 0 0.5rem;
  color: var(--court);
}

.competition-detail__cancelled-note {
  margin: 0 0 0.75rem;
  padding: 0.6rem 0.75rem;
  border-radius: var(--radius-sm);
  background: var(--pill-critical-bg);
  color: var(--ink);
  font-weight: 600;
}

.competition-detail__fields {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.competition-detail__field {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.4rem 0;
  border-bottom: 1px solid var(--hs-border);
}

.competition-detail__field dt {
  color: var(--ink-soft);
}

.competition-detail__field dd {
  margin: 0;
  font-weight: 600;
  text-align: right;
}

.competition-detail__sessions-block {
  margin-top: 1rem;
}

.competition-detail__subheading {
  font-size: var(--font-size-base);
  margin: 0 0 0.5rem;
}

.competition-detail__sessions {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.competition-detail__session {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
}

.competition-detail__session-time {
  font-weight: 600;
}

.competition-detail__session-courts {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

/* iPad/web: more breathing room, and sessions side by side once there is
   room for them — widened, not restructured, mirroring
   GameDetailPanel.vue's approach at these breakpoints. */
@media (min-width: 768px) {
  .competition-detail__field {
    padding: 0.5rem 0;
  }
}

@media (min-width: 1280px) {
  .competition-detail__sessions {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  }
}
</style>
