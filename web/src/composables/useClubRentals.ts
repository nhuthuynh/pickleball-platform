import { ref, type Ref } from 'vue'
import { bookingClient, type BookingClient } from '../api/bookingClient'
import { mapToRecurringHireTemplate, type RecurringHireTemplateView } from '../models/recurringHire'
import {
  validateRecurringHireForm,
  toRequestRecurringHireBody,
  type RecurringHireFormInput,
  type RecurringHireFormErrors,
} from '../models/recurringHireForm'

export interface UseClubRentalsResult {
  templates: Ref<RecurringHireTemplateView[]>
  loading: Ref<boolean>
  /** Human-readable message when the LIST call failed. */
  error: Ref<string | null>
  requesting: Ref<boolean>
  /** Per-field validation/identification messages (WCAG 3.3.1/3.3.3). */
  fieldErrors: Ref<RecurringHireFormErrors>
  /** Form-level message for a failure that isn't about one specific field. */
  formError: Ref<string | null>
  /** Success confirmation, rendered through an ARIA live region (WCAG 4.1.3). */
  statusMessage: Ref<string>
  load: (actorUserId: string) => Promise<void>
  /** Returns true only if the template was actually created server-side. */
  request: (input: RecurringHireFormInput, actorUserId: string) => Promise<boolean>
}

/**
 * Drives the Club's rental screen (T11.6): T11.5's `RequestRecurringHire` for
 * the write, and this ticket's new actor-scoped
 * `ListRecurringHireTemplatesForActor` for the status view. `client` is
 * injectable (defaults to the real `bookingClient`), same pattern as
 * `useFacilityDiscounts`/`useCourtBooking`.
 *
 * **Validation is enforced here, in code, not only by a disabled submit
 * button** — `request` returns before calling the API whenever
 * `validateRecurringHireForm` rejects the input, so a programmatic call cannot
 * bypass the gate either. Same discipline as `useFacilityDiscounts.create`.
 *
 * **The list is never filtered by status.** Every template the actor requested
 * comes back and is shown, including rejected ones: dropping a decided request
 * from the list is exactly how a screen ends up implying an answered request
 * is still pending (T11.6 instruction #4).
 */
export function useClubRentals(client: BookingClient = bookingClient): UseClubRentalsResult {
  const templates = ref<RecurringHireTemplateView[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const requesting = ref(false)
  const fieldErrors = ref<RecurringHireFormErrors>({})
  const formError = ref<string | null>(null)
  const statusMessage = ref('')

  async function load(actorUserId: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await client.GET('/v1/recurring-hire-templates', {
        params: { query: { actorUserId } },
      })
      if (apiError || !data) {
        templates.value = []
        error.value = 'Could not load your rental requests. Please try again.'
        return
      }
      // An absent `templates` is an empty list, not a failure — a club that
      // has never requested a slot is a normal, valid state.
      templates.value = (data.templates ?? []).map(mapToRecurringHireTemplate)
    } catch {
      // openapi-fetch propagates a thrown fetch() rejection rather than
      // populating `error` for this case — see api/__tests__/client.spec.ts.
      templates.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  async function request(input: RecurringHireFormInput, actorUserId: string): Promise<boolean> {
    formError.value = null

    const errors = validateRecurringHireForm(input)
    fieldErrors.value = errors
    if (Object.keys(errors).length > 0) {
      // Real code gate, not just a disabled button — nothing is sent.
      return false
    }

    requesting.value = true
    try {
      const { data, error: apiError, response } = await client.POST('/v1/recurring-hire-templates', {
        body: toRequestRecurringHireBody(input, actorUserId),
      })

      // The server resolves the actor's `club` role from Identity and answers
      // PermissionDenied (403) for anyone else. The client's own role check
      // decides what to render; THIS is the check that decides what happens,
      // so its failure gets its own message rather than a generic one.
      if (response?.status === 403) {
        formError.value =
          'Your account is not registered as a club, so it cannot request a recurring rental. Contact the facility if you think that is wrong.'
        return false
      }
      if (response?.status === 404) {
        formError.value =
          'That court could not be found, or it is not attached to a facility that can approve rentals. Pick a different court.'
        return false
      }

      if (apiError || !data?.template) {
        formError.value = 'Could not send this request. Check the details above and try again.'
        return false
      }

      const created = mapToRecurringHireTemplate(data.template)
      templates.value = [...templates.value, created]
      // Deliberately not "Court booked": nothing is booked until the owner
      // approves, and T11.4/T11.5 keep the template distinct from the
      // Bookings an approval later generates.
      statusMessage.value = 'Request sent to the facility owner. No courts are booked until they approve it.'
      return true
    } catch {
      formError.value = 'Could not reach the server. Check your connection and try again.'
      return false
    } finally {
      requesting.value = false
    }
  }

  return { templates, loading, error, requesting, fieldErrors, formError, statusMessage, load, request }
}
