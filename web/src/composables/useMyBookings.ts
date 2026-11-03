import { ref, type Ref } from 'vue'
import { bookingClient, type BookingClient } from '../api/bookingClient'
import { socialplayClient, type SocialPlayClient } from '../api/socialplayClient'
import {
  mapToActorBooking,
  mapToActorRegistration,
  type ActorBookingView,
  type ActorRegistrationView,
} from '../models/actorBookings'

export interface UseMyBookingsResult {
  bookings: Ref<ActorBookingView[]>
  bookingsLoading: Ref<boolean>
  /** Human-readable message when the BOOKINGS read failed. Independent of
   *  `registrationsError` on purpose — see this file's header. */
  bookingsError: Ref<string | null>
  registrations: Ref<ActorRegistrationView[]>
  registrationsLoading: Ref<boolean>
  registrationsError: Ref<string | null>
  /** Runs both reads. Never rejects: each half reports its own failure. */
  load: () => Promise<void>
  /** Re-runs ONE read, for a per-section "Try again". Exposed because the
   *  errors are per-section: retrying the half that already succeeded would
   *  discard good rows to re-request them. Neither rejects. */
  loadBookings: () => Promise<void>
  loadRegistrations: () => Promise<void>
}

/**
 * Drives the player's Bookings tab (T68.3, the read half of #353) from the
 * two actor-scoped reads T67 shipped and nothing displayed:
 *
 *   - `GET /v1/bookings`      — booking.ListBookingsForActor      (T67.1)
 *   - `GET /v1/registrations` — socialplay.ListRegistrationsForActor (T67.2)
 *
 * **Both request messages are empty, and that is the contract.** The acting
 * user comes from the verified bearer token's principal and from nowhere
 * else; neither proto has an actor field for a caller to populate, not even
 * a deprecated one. So `load()` takes no arguments — there is deliberately
 * nothing to pass it, and a signature that accepted an actor id would be
 * this client inventing a field the API refuses to read. Contrast
 * `useClubRentals.load(actorUserId)`, whose RPC predates T12.7's
 * authentication and still carries such a field.
 *
 * ── TWO READS, TWO SETS OF STATE, AND WHY IT IS NOT ONE ─────────────────────
 *
 * These are two different bounded contexts over two different typed clients,
 * and either can fail while the other succeeds. The ticket's requirement 5
 * is that one failing read must not blank the screen, so there is no shared
 * `loading`/`error` pair here: each section owns its own, and the component
 * renders them independently.
 *
 * **`Promise.all` would be wrong, and would be wrong quietly.** It rejects
 * on the first failure, so a single 500 from one context would abandon the
 * other read mid-flight, leave its `loading` stuck `true` forever, and
 * propagate a rejection out of `load()` that the caller has no sensible
 * place to handle. Each half is therefore wrapped so it RESOLVES on failure,
 * and the two are run concurrently with `Promise.all` over those already-safe
 * halves — concurrency without coupled failure. (`allSettled` would work too;
 * this shape keeps the error handling beside the read it belongs to, rather
 * than in a result-inspecting loop that has to re-derive which was which.)
 *
 * `client`s are injectable, defaulting to the real ones — the same pattern
 * as `useClubRentals`/`useCourtBooking`/`useFacilityList`.
 *
 * ── NOTHING IS FILTERED ─────────────────────────────────────────────────────
 *
 * Every row either read returns is mapped and kept: cancelled bookings,
 * host-cancelled games, rows with no facility. Both protos are explicit that
 * client-side filtering is the failure mode here — see models/actorBookings.ts's
 * header for the quotes. There is no `.filter()` in this file, and that is a
 * property to preserve, not an omission to tidy up.
 */
export function useMyBookings(
  booking: BookingClient = bookingClient,
  socialplay: SocialPlayClient = socialplayClient,
): UseMyBookingsResult {
  const bookings = ref<ActorBookingView[]>([])
  const bookingsLoading = ref(false)
  const bookingsError = ref<string | null>(null)

  const registrations = ref<ActorRegistrationView[]>([])
  const registrationsLoading = ref(false)
  const registrationsError = ref<string | null>(null)

  // A 401 means "sign in again", not "the server is broken" — different
  // remedy, so a different sentence. The client middleware has already
  // cleared the stale token by the time this runs (see api/client.ts); all
  // this has to do is not mislabel it. Kept as one helper so the two reads
  // cannot drift into saying different things about the same status.
  function messageFor(status: number | undefined, subject: string): string {
    if (status === 401) {
      return `Your session has expired. Sign in again to see ${subject}.`
    }
    return `Could not load ${subject}. Please try again.`
  }

  async function loadBookings(): Promise<void> {
    bookingsLoading.value = true
    bookingsError.value = null
    try {
      // `{}` rather than no second argument: ListBookingsForActorRequest is
      // empty, so there are no params to send, but openapi-fetch's typing
      // still expects the options object.
      const { data, error, response } = await booking.GET('/v1/bookings', {})
      if (error || !data) {
        bookings.value = []
        bookingsError.value = messageFor(response?.status, 'your court bookings')
        return
      }
      // An absent `bookings` is an empty list, not a failure — a player who
      // has never booked a court is a normal, valid state.
      bookings.value = (data.bookings ?? []).map(mapToActorBooking)
    } catch {
      // openapi-fetch propagates a thrown fetch() rejection rather than
      // populating `error` for this case — see api/__tests__/client.spec.ts.
      bookings.value = []
      bookingsError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      bookingsLoading.value = false
    }
  }

  async function loadRegistrations(): Promise<void> {
    registrationsLoading.value = true
    registrationsError.value = null
    try {
      const { data, error, response } = await socialplay.GET('/v1/registrations', {})
      if (error || !data) {
        registrations.value = []
        registrationsError.value = messageFor(response?.status, 'the games you have joined')
        return
      }
      registrations.value = (data.registrations ?? []).map(mapToActorRegistration)
    } catch {
      registrations.value = []
      registrationsError.value =
        'Could not reach the server. Check your connection and try again.'
    } finally {
      registrationsLoading.value = false
    }
  }

  async function load(): Promise<void> {
    // Both halves resolve rather than reject (above), so this `Promise.all`
    // runs them concurrently WITHOUT coupling their failures. See the header.
    await Promise.all([loadBookings(), loadRegistrations()])
  }

  return {
    bookings,
    bookingsLoading,
    bookingsError,
    registrations,
    registrationsLoading,
    registrationsError,
    load,
    loadBookings,
    loadRegistrations,
  }
}
