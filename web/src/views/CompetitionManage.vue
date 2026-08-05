<script setup lang="ts">
// Competition roster / manage screen (Host) — T9.6
// (docs/process/t9-sprint-plan.md), mounted at
// `/competitions/:id/manage`. The "Manage roster" half of Flow 5: who has
// entered, how many guests they are bringing, whether they have paid, and
// HOW THEY GOT HERE.
//
// Reads `GetCompetition` (T9.4) and `ListEntriesForCompetition` (T9.4's
// Host-facing roster read, which deliberately returns cancelled entries too
// — "who withdrew" and "who never entered" are different answers).
//
// EVERY FIELD ON THIS SCREEN COMES FROM ONE OF THOSE TWO RESPONSES. Three
// consequences worth stating, because each is a thing a reader might
// otherwise expect to find here:
//
//  1. NO ENTRANT NAMES. `CompetitionEntry` carries `player_id` and nothing
//     else about the person — there is no Identity/Users context in this
//     repo (HANDOFF.md's open cross-cutting item), so there is no name to
//     render. The row shows the player identifier, exactly as
//     `HostPayments.vue` (T8.10) already does for Registrations, rather
//     than inventing a display name. When Identity lands, this is the one
//     line that changes.
//
//  2. NO SHARE LINK. `share_token` is returned ONCE, on
//     `CreateCompetitionResponse`, and is deliberately NOT a field on
//     `Competition` so it cannot leak through the unauthenticated
//     `GetCompetition`/`ListCompetitions` reads. T9 has no authenticated
//     re-read and no rotation path (see `GetCompetitionByShareToken`'s
//     KNOWN GAP note — rotation is blocked on real auth, not merely
//     unfinished). So this screen genuinely cannot obtain the link, and
//     says so in one line instead of rendering a plausible-looking URL
//     that would resolve to nothing.
//
//  3. NO "MARK PAID" BUTTON. `payments.PayableType` has exactly three
//     values — BOOKING, REGISTRATION, NO_SHOW_FEE — and none of them is a
//     Competition entry, so `RecordOfflinePayment` cannot be called for
//     one. A "Mark paid" control here could only ever fail. The unpaid
//     amount is still SURFACED, using T8.10's own component
//     (`UnpaidCashAmount.vue`, extracted from HostPayments.vue by this
//     ticket rather than duplicated), with a one-line note about what a
//     Host can't yet do from here — the same "honest note, not a dead
//     control" convention T8.8 used for its omitted matching step.
import { computed, onMounted, ref } from 'vue'
import { competitionsClient as defaultClient, type CompetitionsClient } from '../api/competitionsClient'
import { useBreakpoint } from '../composables/useBreakpoint'
import { entryFeeLabel } from '../models/payment'
import {
  competitionDatesLabel,
  competitionFormatLabel,
  entryPaymentStatusLabel,
  entrySourceLabel,
  entryStatusLabel,
  formatSessionRange,
  mapToCompetitionEntry,
  mapToCompetitionSummary,
  type CompetitionEntrySummary,
  type CompetitionSummary,
} from '../models/competition'
import UnpaidCashAmount from '../components/payments/UnpaidCashAmount.vue'

const props = withDefaults(
  defineProps<{
    /** The Competition id, supplied by the router from `/competitions/:id/manage`. */
    id: string
    /** Injectable for tests; defaults to the real competitionsClient. */
    client?: CompetitionsClient
    /** Injectable for tests; defaults to the ambient `window` (matchMedia) —
     * same pattern as HostPayments.vue's `win` prop. */
    win?: Pick<Window, 'matchMedia'>
  }>(),
  { client: () => defaultClient, win: undefined },
)

const { breakpoint } = useBreakpoint(props.win ?? window)

const competition = ref<CompetitionSummary | null>(null)
const entries = ref<CompetitionEntrySummary[]>([])
const loading = ref(false)
const error = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const [competitionResult, entriesResult] = await Promise.all([
      props.client.GET('/v1/competitions/{competitionId}', {
        params: { path: { competitionId: props.id } },
      }),
      props.client.GET('/v1/competitions/{competitionId}/entries', {
        params: { path: { competitionId: props.id } },
      }),
    ])

    if (competitionResult.error || !competitionResult.data?.competition) {
      error.value = 'Could not load this competition. Please try again.'
      return
    }
    competition.value = mapToCompetitionSummary(competitionResult.data.competition)

    if (entriesResult.error || !entriesResult.data) {
      // The Competition loaded but its roster didn't — say which half
      // failed rather than blanking the whole screen.
      error.value = 'Could not load the entries for this competition. Please try again.'
      return
    }
    entries.value = (entriesResult.data.entries ?? []).map(mapToCompetitionEntry)
  } catch {
    error.value = 'Could not reach the server. Check your connection and try again.'
  } finally {
    loading.value = false
  }
}

/** Whether this Competition takes cash at all — the same proxy
 * `useHostPayments.load` uses for Games, and for the same reason: an entry
 * has a PaymentStatus but no per-entry method, so the Competition's own
 * declared PaymentMethod is the only fact available. */
const acceptsCash = computed(
  () =>
    competition.value?.paymentMethod === 'PAYMENT_METHOD_CASH' ||
    competition.value?.paymentMethod === 'PAYMENT_METHOD_EITHER',
)

/** A free Competition has nothing to collect, so an unpaid entry on one
 * owes nothing — the same correctness filter `useHostPayments` applies. */
function owesCash(entry: CompetitionEntrySummary): boolean {
  return (
    acceptsCash.value &&
    entry.paymentStatus === 'PAYMENT_STATUS_UNPAID' &&
    (competition.value?.entryFeeCents ?? 0) > 0
  )
}

function guestLabel(guestCount: number): string {
  if (guestCount <= 0) return ''
  return guestCount === 1 ? '1 guest' : `${guestCount} guests`
}

const isCancelled = computed(
  () => competition.value?.status === 'COMPETITION_STATUS_CANCELLED',
)

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="competition-manage" :data-breakpoint="breakpoint" aria-label="Manage competition">
    <p v-if="loading" class="cm-status" role="status">Loading this competition…</p>

    <div v-else-if="error" class="cm-error" role="alert" data-testid="manage-error">
      <p>{{ error }}</p>
      <button type="button" class="cm-retry" @click="load">Try again</button>
    </div>

    <template v-else-if="competition">
      <header class="cm-header" data-testid="competition-summary">
        <h1 class="cm-heading">{{ competition.name }}</h1>

        <!-- A cancelled Competition is stated in words, not implied by a
             greyed-out card (WCAG 1.4.1). -->
        <p v-if="isCancelled" class="cm-note" data-testid="cancelled-notice">
          This competition has been cancelled. Its registration link still resolves for anyone
          who has it, and shows the cancelled state.
        </p>

        <dl class="cm-summary">
          <dt>Format</dt>
          <dd>{{ competitionFormatLabel(competition.format) }}</dd>
          <dt>Dates</dt>
          <dd>{{ competitionDatesLabel(competition.sessions) }}</dd>
          <dt>Capacity</dt>
          <dd>{{ competition.capacity }} places</dd>
          <dt>Guests per entry</dt>
          <dd>{{ competition.guestAllowance }}</dd>
          <dt>Entry fee</dt>
          <!-- 0 renders as the word "Free" — a real Host-chosen state. -->
          <dd>{{ entryFeeLabel(competition.entryFeeCents) }}</dd>
        </dl>

        <h2 class="cm-subheading">Sessions</h2>
        <ul class="cm-sessions">
          <li
            v-for="(session, index) in competition.sessions"
            :key="index"
            data-testid="session-item"
          >
            {{ formatSessionRange(session.startsAt, session.endsAt) }} ·
            {{ session.courtIds.length }} court{{ session.courtIds.length === 1 ? '' : 's' }}
          </li>
        </ul>
      </header>

      <section class="cm-roster" aria-labelledby="roster-heading">
        <h2 id="roster-heading" class="cm-subheading">Entries</h2>

        <p v-if="entries.length === 0" class="cm-empty" data-testid="roster-empty">
          No entries yet. Share this competition's registration link to get the first one.
        </p>

        <ul v-else class="cm-roster-list">
          <li v-for="entry in entries" :key="entry.id" class="cm-roster-row" data-testid="roster-row">
            <div class="cm-roster-details">
              <!-- No Identity context exists, so there is no name to show —
                   the player identifier, exactly as HostPayments.vue does.
                   See this file's header. -->
              <p class="cm-roster-player">
                Player {{ entry.playerId }}
                <span v-if="guestLabel(entry.guestCount)"> + {{ guestLabel(entry.guestCount) }}</span>
              </p>

              <!-- WCAG 1.4.1: payment status, entry source, and entry
                   status are each TEXT. None of the three is conveyed by a
                   colour, a dot, or a row tint. -->
              <p class="cm-roster-facts">
                <span data-testid="entry-payment-status">{{
                  entryPaymentStatusLabel(entry.paymentStatus)
                }}</span>
                ·
                <span data-testid="entry-source">{{ entrySourceLabel(entry.source) }}</span>
                <template v-if="entry.status !== 'ENTRY_STATUS_ENTERED'">
                  · <span data-testid="entry-status">{{ entryStatusLabel(entry.status) }}</span>
                </template>
              </p>

              <!-- T8.10's own component, reused rather than duplicated. -->
              <UnpaidCashAmount v-if="owesCash(entry)" :amount-cents="competition.entryFeeCents" />
            </div>
          </li>
        </ul>

        <!-- Honest note, not a dead "Mark paid" button — see header note 3. -->
        <p v-if="acceptsCash" class="cm-note" data-testid="cash-recording-note">
          Recording a cash payment against a competition entry isn't available yet — payments
          can only be recorded for game registrations and bookings today.
        </p>
      </section>

      <!-- See header note 2: the token is not on this response and cannot
           be re-read, so this states the fact instead of inventing a URL. -->
      <p class="cm-note" data-testid="share-link-note">
        This competition's registration link is shown once, when it's created. It can't be
        retrieved again yet.
      </p>

      <p class="cm-note" data-testid="matching-note">
        Automated matching isn't available yet — players enter directly.
      </p>
    </template>
  </section>
</template>

<style scoped>
.competition-manage {
  font-family: var(--font-family-ui);
  color: var(--ink);
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  /* iPhone <600px: one stacked column. */
  max-width: 720px;
}

.cm-heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.cm-subheading {
  font-size: var(--font-size-md, 1rem);
  margin: 0;
}

.cm-header,
.cm-roster {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.cm-status {
  color: var(--ink-soft);
  margin: 0;
}

.cm-error {
  color: var(--ink-warning);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.cm-retry {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
}

.cm-summary {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.35rem 1rem;
  margin: 0;
}

.cm-summary dt {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.cm-summary dd {
  margin: 0;
  font-weight: 600;
}

.cm-sessions {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.cm-empty {
  color: var(--ink-soft);
  margin: 0;
}

.cm-roster-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.cm-roster-row {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-md);
  background: var(--paper-raised);
}

.cm-roster-details p {
  margin: 0;
}

.cm-roster-player {
  font-weight: 600;
}

.cm-roster-facts {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.cm-note {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
  background: var(--paper);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  padding: 0.65rem 0.85rem;
  margin: 0;
}

/* iPad (768-1180px) and up: the roster becomes a two-column read, with the
   summary and the list side by side rather than stacked. */
@media (min-width: 768px) {
  .cm-roster-row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

@media (min-width: 1280px) {
  .competition-manage {
    max-width: 960px;
  }
}
</style>
