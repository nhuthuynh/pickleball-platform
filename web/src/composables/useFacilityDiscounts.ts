import { ref, type Ref } from 'vue'
import { bookingClient, type BookingClient } from '../api/bookingClient'
import { mapToDiscountRule, type DiscountRule } from '../models/discount'
import {
  validateDiscountForm,
  toCreateDiscountRuleBody,
  type DiscountFormInput,
  type DiscountFormErrors,
} from '../models/discountForm'

export interface UseFacilityDiscountsResult {
  discounts: Ref<DiscountRule[]>
  loading: Ref<boolean>
  /** Human-readable message when the LIST call failed. */
  error: Ref<string | null>
  creating: Ref<boolean>
  /** Per-field validation/identification messages (WCAG 3.3.1/3.3.3). */
  fieldErrors: Ref<DiscountFormErrors>
  /** Form-level message for a failure that isn't about one specific field. */
  formError: Ref<string | null>
  /** Success confirmation, rendered through an ARIA live region (WCAG 4.1.3). */
  statusMessage: Ref<string>
  load: (facilityId: string) => Promise<void>
  /** Returns true only if the rule was actually created server-side. */
  create: (facilityId: string, input: DiscountFormInput, actorUserId: string) => Promise<boolean>
}

/**
 * Drives the Facility Owner's discount panel (T11.3): T11.2's
 * `ListDiscountRulesForFacility` and `CreateDiscountRule`. `client` is
 * injectable (defaults to the real `bookingClient`), same pattern as
 * `useCourtBooking`/`useFacilityList`.
 *
 * **Validation is enforced here, in code, not only by a disabled submit
 * button** — `create` returns before calling the API whenever
 * `validateDiscountForm` rejects the input, so a programmatic call can't
 * bypass the gate either. Same discipline as `FacilityOnboarding.vue`'s
 * camera-consent gate.
 */
export function useFacilityDiscounts(client: BookingClient = bookingClient): UseFacilityDiscountsResult {
  const discounts = ref<DiscountRule[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const creating = ref(false)
  const fieldErrors = ref<DiscountFormErrors>({})
  const formError = ref<string | null>(null)
  const statusMessage = ref('')

  async function load(facilityId: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await client.GET('/v1/facilities/{facilityId}/discount-rules', {
        params: { path: { facilityId } },
      })
      if (apiError || !data) {
        discounts.value = []
        error.value = 'Could not load discounts for this facility. Please try again.'
        return
      }
      // An absent `discountRules` is an empty list, not a failure — a
      // facility with no discounts configured is a normal, valid state.
      discounts.value = (data.discountRules ?? []).map(mapToDiscountRule)
    } catch {
      // openapi-fetch propagates a thrown fetch() rejection rather than
      // populating `error` for this case — see api/__tests__/client.spec.ts.
      discounts.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  async function create(
    facilityId: string,
    input: DiscountFormInput,
    actorUserId: string,
  ): Promise<boolean> {
    formError.value = null

    const errors = validateDiscountForm(input)
    fieldErrors.value = errors
    if (Object.keys(errors).length > 0) {
      // Real code gate, not just a disabled button — nothing is sent.
      return false
    }

    creating.value = true
    try {
      const { data, error: apiError, response } = await client.POST(
        '/v1/facilities/{facilityId}/discount-rules',
        {
          params: { path: { facilityId } },
          body: toCreateDiscountRuleBody(input, actorUserId),
        },
      )

      // `actorUserId` is checked against the Facility's real OwnerID
      // server-side (Facility.EnsureOwner), so a 403 here means "you are not
      // the owner" specifically — say that, rather than folding it into a
      // generic failure the Owner can't act on.
      if (response?.status === 403) {
        formError.value =
          'Only the owner of this facility can create a discount for it. Check you are signed in as the owner.'
        return false
      }

      if (apiError || !data?.discountRule) {
        formError.value = 'Could not create this discount. Check the details above and try again.'
        return false
      }

      const created = mapToDiscountRule(data.discountRule)
      discounts.value = [...discounts.value, created]
      statusMessage.value = 'Discount created.'
      return true
    } catch {
      formError.value = 'Could not reach the server. Check your connection and try again.'
      return false
    } finally {
      creating.value = false
    }
  }

  return { discounts, loading, error, creating, fieldErrors, formError, statusMessage, load, create }
}
