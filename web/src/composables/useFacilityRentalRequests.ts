import { ref, type Ref } from 'vue'
import { bookingClient, type BookingClient } from '../api/bookingClient'
import {
  mapToRecurringHireTemplate,
  mapToOccurrence,
  summariseOccurrences,
  summariseApproval,
  type OccurrenceView,
  type RecurringHireTemplateView,
} from '../models/recurringHire'

/** The per-week outcome of one approval, kept keyed by template id so the
 * panel can show the result next to the request it belongs to rather than in
 * a single floating banner. */
export type ApprovalResults = Record<string, OccurrenceView[]>

/** Which of the two one-way transitions is being asked for. */
export type Decision = 'approve' | 'reject'

/** A decision the owner has CHOSEN but not yet CONFIRMED (WCAG 3.3.4). */
export interface PendingDecision {
  templateId: string
  decision: Decision
}

export interface UseFacilityRentalRequestsResult {
  templates: Ref<RecurringHireTemplateView[]>
  loading: Ref<boolean>
  error: Ref<string | null>
  /** Template id currently being approved/rejected, or `null`. */
  deciding: Ref<string | null>
  decisionError: Ref<string | null>
  /** The decision awaiting confirmation, or `null` (WCAG 3.3.4). */
  pendingDecision: Ref<PendingDecision | null>
  /** Stage a decision for confirmation. Writes nothing to the server. */
  requestDecision: (templateId: string, decision: Decision) => void
  /** Abandon the staged decision. Writes nothing to the server. */
  cancelDecision: () => void
  /** Success/result confirmation, announced through an ARIA live region
   * (WCAG 4.1.3). For an approval this always carries the per-week counts. */
  statusMessage: Ref<string>
  /** Per-occurrence results from THIS session's approvals, by template id.
   * Deliberately not seeded from anywhere on load — see `approve`. */
  approvalResults: Ref<ApprovalResults>
  load: (facilityId: string, actorUserId: string) => Promise<void>
  approve: (templateId: string, actorUserId: string) => Promise<boolean>
  reject: (templateId: string, actorUserId: string) => Promise<boolean>
}

/**
 * Drives the Facility Owner's incoming rental-requests panel (T11.6):
 * T11.5's `ListRecurringHireTemplatesForFacility` (owner-only), plus
 * `ApproveRecurringHire`/`RejectRecurringHire`. `client` is injectable
 * (defaults to the real `bookingClient`), same pattern as
 * `useFacilityDiscounts`.
 *
 * **`approvalResults` is only ever written from a real approval response.**
 * T11.5's approval books each week independently and reports BOOKED /
 * SKIPPED_CONFLICT / SKIPPED_ERROR per week; that report exists ONLY in the
 * ApproveRecurringHire response, so it is captured verbatim here and never
 * reconstructed, re-derived from the template's schedule, or assumed to be
 * "all booked" for a template approved in an earlier session. A previously
 * approved template simply has no entry, and the view says so rather than
 * inventing one (T11.6 instructions #2 and #4).
 *
 * **The confirm-step gate (WCAG 3.3.4 Error Prevention) lives here, not just
 * in the UI** — deliberately the same placement `useCourtBooking` uses for
 * the same criterion, so the two screens answer 3.3.4 the same way rather
 * than each inventing a shape. `approve`/`reject` are no-ops returning
 * `false` unless `requestDecision` has staged exactly that decision for
 * exactly that template first, so neither irreversible write can fire
 * without a confirmation regardless of what the calling component does.
 * This matters concretely here: FacilityRentalRequests.vue exposes both
 * methods via `defineExpose`, so a markup-only confirm step would leave the
 * write reachable by a caller that never touches a button. T11.4 models
 * both transitions as one-way and approval generates real Bookings across
 * every implied week, which is what puts them in 3.3.4's scope at all.
 */
export function useFacilityRentalRequests(
  client: BookingClient = bookingClient,
): UseFacilityRentalRequestsResult {
  const templates = ref<RecurringHireTemplateView[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const deciding = ref<string | null>(null)
  const decisionError = ref<string | null>(null)
  const statusMessage = ref('')
  const approvalResults = ref<ApprovalResults>({})
  const pendingDecision = ref<PendingDecision | null>(null)

  function requestDecision(templateId: string, decision: Decision): void {
    // Staging a decision clears the previous attempt's error: the owner is
    // being asked a fresh question, so a stale failure alongside it would
    // read as if it applied to this one.
    decisionError.value = null
    pendingDecision.value = { templateId, decision }
  }

  function cancelDecision(): void {
    pendingDecision.value = null
  }

  /** The 3.3.4 gate itself. `true` only when THIS decision on THIS template
   * is the one the owner confirmed — not merely "some decision is staged". */
  function isConfirmed(templateId: string, decision: Decision): boolean {
    const pending = pendingDecision.value
    return pending !== null && pending.templateId === templateId && pending.decision === decision
  }

  async function load(facilityId: string, actorUserId: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError, response } = await client.GET(
        '/v1/facilities/{facilityId}/recurring-hire-templates',
        { params: { path: { facilityId }, query: { actorUserId } } },
      )
      // This read is owner-only (unlike ListDiscountRulesForFacility), so a
      // 403 means something specific and actionable — say that.
      if (response?.status === 403) {
        templates.value = []
        error.value =
          'Only the owner of this facility can see its rental requests. Check you are signed in as the owner.'
        return
      }
      if (apiError || !data) {
        templates.value = []
        error.value = 'Could not load rental requests for this facility. Please try again.'
        return
      }
      templates.value = (data.templates ?? []).map(mapToRecurringHireTemplate)
    } catch {
      templates.value = []
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  function replaceTemplate(updated: RecurringHireTemplateView): void {
    templates.value = templates.value.map((t) => (t.id === updated.id ? updated : t))
  }

  /** Both decisions share their failure handling; only the path and the
   * success wording differ. Kept together so the two cannot drift apart on
   * the 403/404 cases, which mean the same thing for either. */
  function decisionFailureMessage(status: number | undefined): string | null {
    if (status === 403) {
      return 'Only the owner of the facility that owns this court can answer this request.'
    }
    if (status === 404) {
      return 'This request no longer exists. Reload the list to see the current requests.'
    }
    if (status === 400) {
      return 'This request has already been answered and cannot be answered again. Reload the list to see its current status.'
    }
    return null
  }

  async function approve(templateId: string, actorUserId: string): Promise<boolean> {
    // Confirm-step gate (WCAG 3.3.4): never generate a season of Bookings
    // for a decision the owner has not confirmed.
    if (!isConfirmed(templateId, 'approve')) return false

    deciding.value = templateId
    decisionError.value = null
    try {
      const { data, error: apiError, response } = await client.POST(
        '/v1/recurring-hire-templates/{templateId}:approve',
        { params: { path: { templateId } }, body: { actorUserId } },
      )

      const failure = decisionFailureMessage(response?.status)
      if (failure) {
        decisionError.value = failure
        return false
      }
      if (apiError || !data?.template) {
        decisionError.value = 'Could not approve this request. Please try again.'
        return false
      }

      replaceTemplate(mapToRecurringHireTemplate(data.template))

      // The occurrences list is the ONLY place the per-week outcome exists.
      // An approval response with no occurrences is recorded as an empty list
      // (a real, reportable fact: no weeks were implied) rather than dropped.
      const occurrences = (data.occurrences ?? []).map(mapToOccurrence)
      approvalResults.value = { ...approvalResults.value, [templateId]: occurrences }
      statusMessage.value = summariseApproval(summariseOccurrences(occurrences))
      return true
    } catch {
      decisionError.value = 'Could not reach the server. Check your connection and try again.'
      return false
    } finally {
      deciding.value = null
      // The confirmation is spent either way. On success the request is now
      // decided and offers no controls at all; on failure the owner is
      // returned to the queue with the reason, and must choose again rather
      // than find a still-armed confirmation waiting.
      pendingDecision.value = null
    }
  }

  async function reject(templateId: string, actorUserId: string): Promise<boolean> {
    // Confirm-step gate (WCAG 3.3.4): rejection is terminal too — the club
    // would have to send a new request — so it is confirmed the same way.
    if (!isConfirmed(templateId, 'reject')) return false

    deciding.value = templateId
    decisionError.value = null
    try {
      const { data, error: apiError, response } = await client.POST(
        '/v1/recurring-hire-templates/{templateId}:reject',
        { params: { path: { templateId } }, body: { actorUserId } },
      )

      const failure = decisionFailureMessage(response?.status)
      if (failure) {
        decisionError.value = failure
        return false
      }
      if (apiError || !data?.template) {
        decisionError.value = 'Could not reject this request. Please try again.'
        return false
      }

      replaceTemplate(mapToRecurringHireTemplate(data.template))
      // States the finality, because it is real: T11.4 models reject as a
      // one-way transition, so this request cannot later be approved.
      statusMessage.value =
        'Request rejected. No courts were booked, and this request is now closed — the club would have to send a new one.'
      return true
    } catch {
      decisionError.value = 'Could not reach the server. Check your connection and try again.'
      return false
    } finally {
      deciding.value = null
      // The confirmation is spent either way. On success the request is now
      // decided and offers no controls at all; on failure the owner is
      // returned to the queue with the reason, and must choose again rather
      // than find a still-armed confirmation waiting.
      pendingDecision.value = null
    }
  }

  return {
    templates,
    loading,
    error,
    deciding,
    decisionError,
    statusMessage,
    approvalResults,
    pendingDecision,
    requestDecision,
    cancelDecision,
    load,
    approve,
    reject,
  }
}
