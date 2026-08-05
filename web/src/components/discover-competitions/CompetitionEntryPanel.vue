<script setup lang="ts">
// Enter-a-competition flow (T9.7 requirement #1): a guest-count stepper
// bounded by the Competition's GuestAllowance, then `EnterCompetition`
// carrying the `source` attribution for how this entrant arrived.
//
// Owns its own network call (like GameJoinPanel.vue owns useJoinGame) rather
// than being purely presentational, since the entry flow is the
// network-calling half of these screens — CompetitionDetailPanel.vue, this
// component's parent, stays presentational for the read-only fields around
// it.
//
// Keyed by `competition.id` at the call site so switching the selected
// Competition always mounts a fresh instance — no stale guest count or
// leftover success/error state from a previously viewed Competition.
//
// TWO THINGS THAT ARE DELIBERATELY ABSENT, so neither reads as an oversight:
//
// 1. NO WAITLIST. Social Play's waitlist is Game-scoped (T6.6 / ADR-0006), a
//    different bounded context; T9 gives Competitions none. A full
//    Competition therefore suggests browsing, not a queue — offering a
//    waitlist here would be a promise nothing in this system keeps.
//
// 2. NO ONLINE CHECKOUT HOP. T8.10's cash path is reused verbatim below. Its
//    ONLINE path is not, and could not honestly be: the Payments context has
//    no payable type for a Competition entry, and confirming a
//    PAYABLE_TYPE_REGISTRATION Payment reconciles the paid status back into
//    SOCIAL PLAY's registrations (internal/payments/app/service.go's
//    reconcileRegistrationPaymentStatus). Sending a Competition entry id
//    down that path would write one context's id into another's table and
//    fail at confirm time — after the Payment had already been persisted as
//    paid. Competitions↔Payments wiring is explicitly not a T9 ticket
//    (competitions.proto's PaymentStatus doc comment). No second checkout
//    was built; the position is simply stated to the entrant instead.
import { computed, ref } from 'vue'
import { useEnterCompetition, MOCK_PLAYER_ID } from '../../composables/useEnterCompetition'
import { entryFeeLabel, type CompetitionSummary, type EntrySource } from '../../models/competition'
import type { CompetitionsClient } from '../../api/competitionsClient'

const props = defineProps<{
  competition: CompetitionSummary
  /** How this entrant arrived — `ENTRY_SOURCE_SOCIAL` from the deep-link
   * landing at `/c/:shareToken`, `ENTRY_SOURCE_APP` from `/competitions`.
   * Supplied by the screen, since only the screen knows the arrival path. */
  source: EntrySource
  /** Injectable for tests; defaults to the real competitionsClient. */
  client?: CompetitionsClient
}>()

const {
  guestCount,
  entering,
  enterError,
  competitionFull,
  alreadyEntered,
  confirmedEntry,
  incrementGuests,
  decrementGuests,
  enter,
} = useEnterCompetition(props.competition.id, props.source, props.client)

/**
 * The Competition's last-known spots_left is already 0, so skip straight to
 * the "full" state rather than rendering a form that would predictably fail.
 *
 * A UX nicety only, with the same caveat as the GuestAllowance-bounded
 * stepper: spots_left is a snapshot and can be stale in EITHER direction. A
 * genuine race is still handled by the same 409 path this pre-seeds, not
 * bypassed by it. `null` (the endpoint didn't supply a figure) is NOT
 * treated as zero — an unknown is not a "no".
 */
const startsFull = computed(() => props.competition.spotsLeft !== null && props.competition.spotsLeft <= 0)

const showFullState = computed(() => competitionFull.value || startsFull.value)

const canIncrement = computed(() => guestCount.value < props.competition.guestAllowance)
const canDecrement = computed(() => guestCount.value > 0)

const isFree = computed(() => props.competition.entryFeeCents <= 0)

// PAYMENT_METHOD_UNSPECIFIED (an old/unaware Competition, or a field left
// unset) folds into 'either' — the least restrictive option — mirroring
// internal/competitions/adapter/grpcapi.fromProtoPaymentMethod's identical
// "closest thing to unspecified" resolution server-side.
const paymentMode = computed<'online' | 'cash' | 'either'>(() => {
  if (props.competition.paymentMethod === 'PAYMENT_METHOD_ONLINE') return 'online'
  if (props.competition.paymentMethod === 'PAYMENT_METHOD_CASH') return 'cash'
  return 'either'
})

/** Set once the entrant picks "Pay cash at facility" on an 'either'
 * Competition — no network call, exactly as T8.10 established. */
const cashChosen = ref(false)

const showPendingCash = computed(() => paymentMode.value === 'cash' || cashChosen.value)

function onIncrement(): void {
  incrementGuests(props.competition.guestAllowance)
}

function onEnter(): void {
  void enter(MOCK_PLAYER_ID)
}
</script>

<template>
  <section class="competition-entry" aria-label="Enter this competition">
    <!-- SUCCESS (ARIA live region, WCAG 4.1.3) -->
    <div v-if="confirmedEntry" class="competition-entry__success" role="status" aria-live="polite">
      <p>
        You're in! Entered with {{ confirmedEntry.guestCount }}
        {{ confirmedEntry.guestCount === 1 ? 'guest' : 'guests' }}.
      </p>

      <!-- A free Competition owes nothing, so no payment step is offered at
           all — a zero fee is a real product state a Host chose. -->
      <p v-if="isFree" class="competition-entry__payment-status" data-testid="entry-free-notice">
        This competition is free — there's nothing to pay.
      </p>

      <!-- T8.10's cash path, unchanged: no Payment is created client-side,
           the amount owed is named in words. WCAG 1.4.1 — "pending (cash at
           facility)" is text, never a colour. -->
      <p v-else-if="showPendingCash" class="competition-entry__payment-status">
        Payment: {{ entryFeeLabel(competition.entryFeeCents) }} pending (cash at facility)
      </p>

      <template v-else>
        <p class="competition-entry__payment-status">
          Entry fee: {{ entryFeeLabel(competition.entryFeeCents) }}
        </p>
        <!-- See this file's header comment #2 for why there is no
             "Pay online now" button here. -->
        <p class="competition-entry__payment-status" data-testid="entry-online-unavailable">
          Paying online for a competition is not available yet — the host will confirm how to pay.
        </p>
        <div v-if="paymentMode === 'either'" class="competition-entry__payment-choice">
          <button type="button" class="competition-entry__secondary competition-entry__pay-cash" @click="cashChosen = true">
            Pay cash at facility
          </button>
        </div>
      </template>
    </div>

    <!-- FULL: a specific, actionable message (WCAG 3.3.3 Error Suggestion).
         No waitlist is offered — see this file's header comment #1. -->
    <div v-else-if="showFullState" class="competition-entry__conflict" role="alert">
      <p class="competition-entry__conflict-message">
        {{ enterError ?? 'This competition is full.' }}
      </p>
      <!-- The suggested next step is browsing, not a queue. Naming the
           absent waitlist would only raise a question the product has no
           answer to — see this file's header comment #1. -->
      <p class="competition-entry__conflict-hint">
        Browse the other competitions to find one you can still enter.
      </p>
    </div>

    <!-- ALREADY ENTERED: also a 409, but a different fact and a different
         next step, so it gets its own message rather than "just filled up". -->
    <div v-else-if="alreadyEntered" class="competition-entry__conflict" role="alert">
      <p class="competition-entry__conflict-message">{{ enterError }}</p>
    </div>

    <!-- ENTRY FORM -->
    <form v-else class="competition-entry__form" @submit.prevent="onEnter">
      <div class="competition-entry__stepper">
        <span class="competition-entry__stepper-label" id="competition-guest-stepper-label">Guests</span>
        <div
          class="competition-entry__stepper-controls"
          role="group"
          aria-labelledby="competition-guest-stepper-label"
        >
          <button
            type="button"
            class="competition-entry__stepper-button"
            aria-label="Remove a guest"
            :disabled="!canDecrement"
            @click="decrementGuests"
          >
            −
          </button>
          <span class="competition-entry__stepper-count" aria-live="polite">{{ guestCount }}</span>
          <button
            type="button"
            class="competition-entry__stepper-button"
            aria-label="Add a guest"
            :disabled="!canIncrement"
            @click="onIncrement"
          >
            +
          </button>
        </div>
        <span class="competition-entry__stepper-hint">
          Up to {{ competition.guestAllowance }} guests allowed
        </span>
      </div>

      <!-- The entrant sees the real price BEFORE entering, never a
           placeholder discovered later. A free competition says "Free" in
           words (WCAG 1.4.1). -->
      <p class="competition-entry__fee" data-testid="entry-fee">
        Entry fee: <strong>{{ entryFeeLabel(competition.entryFeeCents) }}</strong>
      </p>

      <p v-if="enterError" class="competition-entry__status competition-entry__status--error" role="alert">
        {{ enterError }}
      </p>

      <button type="submit" class="competition-entry__primary" :disabled="entering">
        {{ entering ? 'Entering…' : 'Enter competition' }}
      </button>
    </form>
  </section>
</template>

<style scoped>
.competition-entry {
  font-family: var(--font-family-ui);
  color: var(--ink);
  border-top: 1px solid var(--hs-border);
  margin-top: 1rem;
  padding-top: 1rem;
}

.competition-entry__form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.competition-entry__stepper {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.competition-entry__stepper-label {
  font-size: var(--font-size-sm);
  color: var(--ink-soft);
}

.competition-entry__stepper-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.competition-entry__stepper-button {
  font: inherit;
  /* >=44px touch target on iPad/iPhone. */
  min-height: 44px;
  min-width: 44px;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--court);
  cursor: pointer;
  font-size: var(--font-size-lg);
  line-height: 1;
}

.competition-entry__stepper-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.competition-entry__stepper-count {
  min-width: 2ch;
  text-align: center;
  font-weight: 600;
}

.competition-entry__stepper-hint {
  font-size: var(--font-size-xs);
  color: var(--ink-soft);
}

.competition-entry__status {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
}

.competition-entry__status--error {
  color: var(--ink-warning);
}

.competition-entry__primary {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--court);
  border-radius: var(--radius-sm);
  background: var(--court);
  color: var(--paper-raised);
  cursor: pointer;
  align-self: flex-start;
}

.competition-entry__primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.competition-entry__secondary {
  font: inherit;
  min-height: 44px;
  padding: 0.5rem 1rem;
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink);
  cursor: pointer;
}

/* WCAG 2.4.7 — a visible focus indicator on every interactive control. */
.competition-entry__primary:focus-visible,
.competition-entry__secondary:focus-visible,
.competition-entry__stepper-button:focus-visible {
  outline: 3px solid var(--court);
  outline-offset: 2px;
}

.competition-entry__conflict {
  border: 1px solid var(--pill-critical-bg);
  background: var(--pill-critical-bg);
  color: var(--ink);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.competition-entry__conflict-message {
  font-weight: 600;
  margin: 0;
}

.competition-entry__conflict-hint {
  margin: 0;
  font-size: var(--font-size-sm);
}

.competition-entry__success {
  color: var(--ink-success);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.competition-entry__payment-status {
  margin: 0;
  color: var(--ink-soft);
}

.competition-entry__fee {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--ink);
}

.competition-entry__payment-choice {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}
</style>
