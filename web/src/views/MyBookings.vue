<script setup lang="ts">
// The player's Bookings tab — T68.3, the read half of issue #353.
//
// ── WHY THIS FILE EXISTS ────────────────────────────────────────────────────
//
// T67 shipped both actor-scoped reads and cut the screen. `/bookings` pointed
// at `ComingSoonView` and rendered literally `Coming soon.`, while `AppNav`
// advertised Bookings as one of five top-level tabs — a tab a user can click
// that answers nothing. `GET /v1/bookings` (T67.1) and
// `GET /v1/registrations` (T67.2) both existed, authenticated, and had no
// consumer anywhere in the product.
//
// This component is the presentational half; the network lives in
// `useMyBookings`, the same split as CourtBookingFlow vs. useCourtBooking.
// Every label comes from `models/actorBookings.ts`'s pure functions, which
// are unit-tested there — nothing is formatted inline in this template.
//
// ── THE TWO SECTIONS ARE INDEPENDENT, VISIBLY ───────────────────────────────
//
// Two bounded contexts, two reads, two sets of loading/error state, rendered
// side by side. Either can fail alone and the other still renders (the
// ticket's requirement 5) — so each section carries its own `role="alert"`
// message and its own "Try again", and there is no screen-level error branch
// that could blank both. A per-section retry calls only that section's read:
// re-fetching the healthy half would discard rows the user is already
// reading.
//
// ── NOTHING IS FILTERED, AND THAT IS LOAD-BEARING ───────────────────────────
//
// There is no `v-if` anywhere below that hides a row, and no `.filter()` in
// the composable. Both protos are explicit that client-side filtering is the
// failure mode:
//
//   - a CANCELLED booking renders, labelled — `ListBookingsForActor` has no
//     status filter precisely so a client cannot hide a cancellation, and
//     the user needs to tell "I cancelled it" from "it was lost".
//   - a HOST-CANCELLED game renders, labelled — `ListGames` filters
//     cancelled Games out and socialplay exposes no `GetGame`, so this read
//     is the ONLY place a player is ever told their game is off.
//   - a booking with an EMPTY facility renders — `ActorBooking.facility_id`'s
//     doc comment: "Empty is a real, expected answer... A client must render
//     such a booking, not hide it — the booking is real." Both Courts seeded
//     by `0002_seed.sql` have no Facility, so this is the common case on a
//     freshly migrated database, not an edge case.
//
// ── NO RAW UUIDS (#342, closed, and kept closed) ────────────────────────────
//
// Venue and court names are resolved by the EXISTING `VenueName.vue` and
// `CourtNames.vue` — the network-calling-child pattern DisplayName.vue
// established — not by a third resolver written here. `CourtNames`' `courtIds`
// prop has always been `readonly string[]`, which is exactly the cardinality
// `RegistrationListing.game_court_ids` needs (repeated: a multi-court Game is
// normal in socialplay), so a booking's single `court_id` is passed as a
// one-element list and a game's list is passed through as-is. One resolver,
// both sections, both cardinalities.
//
// The honest-label props are set explicitly rather than left on their
// defaults, because this screen's empty case means something specific: a
// booking with no facility is not "Not set" by a host's choice, it is a court
// with no venue on record. See `requirement 3`.
import { computed, onMounted } from 'vue'

import VenueName from '../components/facilities/VenueName.vue'
import CourtNames from '../components/facilities/CourtNames.vue'
import { useMyBookings } from '../composables/useMyBookings'
import { authToken } from '../state/authSession'
import type { BookingClient } from '../api/bookingClient'
import type { SocialPlayClient } from '../api/socialplayClient'
import type { FacilitiesClient } from '../api/facilitiesClient'
import {
  amountOwedLabel,
  bookingSourceLabel,
  bookingStatusLabel,
  formatBookingWindow,
  gameStatusLabel,
  guestCountLabel,
  paymentStatusLabel,
  registrationStatusLabel,
} from '../models/actorBookings'

const props = defineProps<{
  /** Injectable for tests; defaults to the real bookingClient. */
  client?: BookingClient
  /** Injectable for tests; defaults to the real socialplayClient. */
  socialplayClient?: SocialPlayClient
  /** Injectable for tests; passed through to VenueName and CourtNames, the
   *  same way GameDetailPanel.vue passes it to the same two components. */
  facilitiesClient?: FacilitiesClient
}>()

const {
  bookings,
  bookingsLoading,
  bookingsError,
  registrations,
  registrationsLoading,
  registrationsError,
  load,
  loadBookings,
  loadRegistrations,
} = useMyBookings(props.client, props.socialplayClient)

// Reactive rather than a one-shot `isSignedIn()` call: signing in from the
// header must change this screen without a reload.
const signedIn = computed(() => authToken.value !== null)

onMounted(() => {
  // Both reads are actor-scoped and take the actor from the token, so with no
  // token there is nothing to ask for — and a 401 per section would be a
  // worse answer than the sign-in explanation below.
  if (signedIn.value) void load()
})
</script>

<template>
  <section class="my-bookings">
    <h1 class="mb-heading">Bookings</h1>

    <!-- Requirement 6: an empty list would say "you have nothing", which is
         false AND unhelpful. Say what is actually true and link to the fix. -->
    <p v-if="!signedIn" data-testid="signed-out-notice" class="mb-notice" role="status">
      Sign in to see the courts you have booked and the games you have joined.
      <RouterLink to="/sign-in">Sign in</RouterLink>
    </p>

    <template v-else>
      <!-- ── Court bookings ────────────────────────────────────────────── -->
      <section
        data-testid="court-bookings-section"
        class="mb-section"
        aria-labelledby="mb-court-bookings-heading"
      >
        <h2 id="mb-court-bookings-heading" class="mb-section__heading">Court bookings</h2>

        <p v-if="bookingsLoading" class="mb-muted" role="status">Loading your court bookings…</p>

        <template v-else-if="bookingsError">
          <p class="mb-error" role="alert">{{ bookingsError }}</p>
          <!-- A real <button>, so it is in the tab order and operable by
               keyboard without a polyfilled role (WCAG 2.1.1). -->
          <button
            type="button"
            data-testid="retry-bookings"
            class="mb-retry"
            @click="loadBookings()"
          >
            Try again
          </button>
        </template>

        <!-- Requirement 4: name the action that would fill this, and link it. -->
        <p v-else-if="bookings.length === 0" class="mb-empty">
          You have not booked a court yet.
          <RouterLink to="/facilities">Find a court to book</RouterLink>.
        </p>

        <ul v-else class="mb-list">
          <li
            v-for="booking in bookings"
            :key="booking.id"
            :data-testid="`booking-row-${booking.id}`"
            class="mb-row"
          >
            <p class="mb-row__when">{{ formatBookingWindow(booking.startsAt, booking.endsAt) }}</p>
            <dl class="mb-row__details">
              <div class="mb-row__detail">
                <dt>Venue</dt>
                <dd>
                  <VenueName
                    :facility-id="booking.facilityId"
                    :client="facilitiesClient"
                    empty-label="No venue on record"
                    failed-label="Venue name unavailable"
                  />
                </dd>
              </div>
              <div class="mb-row__detail">
                <dt>Court</dt>
                <dd>
                  <!-- A Booking has ONE court_id; CourtNames takes a list, so
                       it is passed as a one-element list rather than a second
                       single-id resolver being written for this screen. -->
                  <CourtNames
                    :facility-id="booking.facilityId"
                    :court-ids="[booking.courtId]"
                    :client="facilitiesClient"
                    empty-label="Court name unavailable"
                    failed-label="Court name unavailable"
                  />
                </dd>
              </div>
              <div class="mb-row__detail">
                <dt>Type</dt>
                <dd>{{ bookingSourceLabel(booking.source) }}</dd>
              </div>
              <div class="mb-row__detail">
                <dt>Status</dt>
                <!-- Text, never colour alone (WCAG 1.4.1). -->
                <dd :data-testid="`booking-status-${booking.id}`">
                  {{ bookingStatusLabel(booking.status) }}
                </dd>
              </div>
            </dl>
          </li>
        </ul>
      </section>

      <!-- ── Games ─────────────────────────────────────────────────────── -->
      <section data-testid="games-section" class="mb-section" aria-labelledby="mb-games-heading">
        <h2 id="mb-games-heading" class="mb-section__heading">Games you have joined</h2>

        <p v-if="registrationsLoading" class="mb-muted" role="status">Loading your games…</p>

        <template v-else-if="registrationsError">
          <p class="mb-error" role="alert">{{ registrationsError }}</p>
          <button
            type="button"
            data-testid="retry-registrations"
            class="mb-retry"
            @click="loadRegistrations()"
          >
            Try again
          </button>
        </template>

        <p v-else-if="registrations.length === 0" class="mb-empty">
          You have not joined any games yet.
          <RouterLink to="/games">Browse games to join</RouterLink>.
        </p>

        <ul v-else class="mb-list">
          <li
            v-for="registration in registrations"
            :key="registration.id"
            :data-testid="`registration-row-${registration.id}`"
            class="mb-row"
          >
            <p class="mb-row__when">
              {{ formatBookingWindow(registration.gameStartsAt, registration.gameEndsAt) }}
            </p>
            <dl class="mb-row__details">
              <div class="mb-row__detail">
                <dt>Venue</dt>
                <dd>
                  <VenueName
                    :facility-id="registration.gameVenueFacilityId"
                    :client="facilitiesClient"
                    empty-label="No venue on record"
                    failed-label="Venue name unavailable"
                  />
                </dd>
              </div>
              <div class="mb-row__detail">
                <!-- Plural: game_court_ids is repeated on the wire and a
                     multi-court Game is normal here. -->
                <dt>Courts</dt>
                <dd>
                  <CourtNames
                    :facility-id="registration.gameVenueFacilityId"
                    :court-ids="registration.gameCourtIds"
                    :client="facilitiesClient"
                    empty-label="Court names unavailable"
                    failed-label="Court names unavailable"
                  />
                </dd>
              </div>
              <!-- The GAME's status, kept visually first among the statuses
                   and worded to name the host, because the row below carries
                   the player's OWN status and the two can both read
                   "cancelled" while meaning opposite things. -->
              <div class="mb-row__detail">
                <dt>Game</dt>
                <dd :data-testid="`game-status-${registration.id}`">
                  {{ gameStatusLabel(registration.gameStatus) }}
                </dd>
              </div>
              <div class="mb-row__detail">
                <dt>Your place</dt>
                <dd :data-testid="`registration-status-${registration.id}`">
                  {{ registrationStatusLabel(registration.status) }}
                </dd>
              </div>
              <div v-if="guestCountLabel(registration.guestCount)" class="mb-row__detail">
                <dt>Guests</dt>
                <dd>{{ guestCountLabel(registration.guestCount) }}</dd>
              </div>
              <div class="mb-row__detail">
                <dt>Payment</dt>
                <dd :data-testid="`payment-status-${registration.id}`">
                  {{ paymentStatusLabel(registration.paymentStatus) }}
                </dd>
              </div>
              <!-- Requirement 7: only for unpaid AND non-zero. A zero amount
                   on an unpaid registration is a FREE game, not a $0.00 debt,
                   so amountOwedLabel returns null and this row vanishes. -->
              <div
                v-if="amountOwedLabel(registration.paymentStatus, registration.amountOwedCents)"
                class="mb-row__detail"
              >
                <dt>Amount owed</dt>
                <dd :data-testid="`amount-owed-${registration.id}`">
                  {{ amountOwedLabel(registration.paymentStatus, registration.amountOwedCents) }}
                </dd>
              </div>
            </dl>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>

<style scoped>
/* Mirrors ClubRentals.vue's `cr-` list/row set (itself FacilityDiscounts'
   `fd-`): same tokens, same 44px touch targets. Renamed `mb-`. */
.my-bookings {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: var(--font-family-ui);
  color: var(--ink);
  max-width: 720px;
}

.mb-heading {
  font-size: var(--font-size-lg);
  margin: 0;
  color: var(--court);
}

.mb-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.mb-section__heading {
  font-size: var(--font-size-md);
  margin: 0;
}

.mb-notice,
.mb-empty,
.mb-muted {
  margin: 0;
  color: var(--ink-soft);
}

.mb-error {
  margin: 0;
  color: var(--ink-danger, #b3261e);
  font-weight: 600;
}

.mb-retry {
  align-self: flex-start;
  min-height: 44px;
  padding: 0 1rem;
  border: 1px solid var(--court);
  border-radius: 0.5rem;
  background: transparent;
  color: var(--court);
  font: inherit;
  cursor: pointer;
}

.mb-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.mb-row {
  border: 1px solid var(--line, rgba(0, 0, 0, 0.12));
  border-radius: 0.5rem;
  padding: 0.75rem 1rem;
}

.mb-row__when {
  margin: 0 0 0.5rem;
  font-weight: 600;
}

.mb-row__details {
  margin: 0;
  display: grid;
  grid-template-columns: minmax(7rem, auto) 1fr;
  gap: 0.25rem 1rem;
}

.mb-row__detail {
  display: contents;
}

.mb-row__details dt {
  color: var(--ink-soft);
}

.mb-row__details dd {
  margin: 0;
}

/* WCAG 2.4.7 — an outline, declared here rather than inherited from the user
   agent, whose ring is not guaranteed to contrast against a tinted panel.
   Same convention as CompetitionLanding.vue/ClubRentals.vue. */
.mb-retry:focus-visible,
.my-bookings a:focus-visible {
  outline: 2px solid var(--court);
  outline-offset: 2px;
}

@media (max-width: 480px) {
  .mb-row__details {
    grid-template-columns: 1fr;
  }

  .mb-row__detail {
    display: block;
    margin-bottom: 0.25rem;
  }
}
</style>
