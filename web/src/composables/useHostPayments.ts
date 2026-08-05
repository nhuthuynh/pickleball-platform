import { ref, type Ref } from 'vue'
import { socialplayClient, type SocialPlayClient } from '../api/socialplayClient'
import { paymentsClient, type PaymentsClient } from '../api/paymentsClient'
import { mapToGameSummary, formatGameRange } from '../models/game'
import { DEFAULT_CURRENCY_CODE, toMoneyRequest } from '../models/payment'

// No Identity/Users/Auth context exists yet (same caveat class as
// FacilityOnboarding.vue's MOCK_OWNER_ID) — RecordOfflinePaymentRequest.
// actor_user_id is necessarily a request field, not a verified identity
// yet, the same known gap that field's own proto doc comment documents.
// Deliberately the same literal value as GameCreation.vue's own (private,
// unexported) `MOCK_HOST_ID` — not imported from there because a
// `<script setup>` SFC has no exports another module can consume without
// adding a second, riskier plain `<script>` block to an already-merged,
// tested file (T8.8) — so this is a duplicated constant, not a shared one,
// and the two must be kept equal by hand if either ever changes. Exists so
// a Game this browser session created in GameCreation.vue (hostId:
// MOCK_HOST_ID) shows up as this Host's own Game here.
export const MOCK_HOST_ID = 'host-mock-1'

/** One pending (unpaid, cash-eligible) Registration a Host still needs to
 * settle (T8.10's Host pending-cash-payments dashboard). "cash-eligible"
 * means the *Game* accepts cash (PaymentMethod cash or either) — Social
 * Play's Registration has no per-registration "method" field of its own
 * (only PaymentStatus); see this file's `load` doc comment for why that's
 * the correct proxy, not a guess. */
export interface PendingCashPayment {
  registrationId: string
  gameId: string
  gameHostId: string
  playerId: string
  guestCount: number
  /** Formatted Game date/time range, for display. */
  gameLabel: string
  /** The owning Game's real entry fee (T9.2) — the amount actually
   * recorded by `markPaid`, replacing T8.10's flat placeholder. Always
   * > 0 here: free games are filtered out of this list entirely (see
   * `load`). */
  entryFeeCents: number
  entryFeeCurrency: string
}

export interface UseHostPaymentsResult {
  pending: Ref<PendingCashPayment[]>
  loading: Ref<boolean>
  error: Ref<string | null>
  /** The registrationId currently being marked paid, or null — drives a
   * per-row "Marking…" disabled state without a single shared spinner
   * blocking every row. */
  markingPaidId: Ref<string | null>
  markPaidError: Ref<string | null>
  /** Fetches this Host's Games (`ListGames`, filtered client-side to
   * `hostId`), then each cash-eligible Game's Registrations
   * (`ListRegistrationsForGame`, T8.10), keeping only PaymentStatus ==
   * unpaid ones.
   *
   * "method == offline" (this ticket's requirement #3) is derived from the
   * *Game's* declared PaymentMethod (cash or either) rather than a
   * per-Payment method, because an unpaid, cash-eligible Registration this
   * dashboard is meant to surface has, by requirement #2, no Payment row
   * at all yet (the Player's cash choice never calls CreateOnlinePayment
   * client-side) — there is nothing else to filter on.
   */
  load: (hostId: string) => Promise<void>
  /** Calls `RecordOfflinePayment` for `entry` and removes it from `pending`
   * on success (T8.10's "Mark paid" action). Uses
   * Records the Game's REAL entry fee (T9.2), carried on each entry from
   * the Game it belongs to — no flat placeholder amount is involved
   * anywhere in this flow any more. */
  markPaid: (entry: PendingCashPayment, actorUserId: string) => Promise<void>
}

export function useHostPayments(
  client: SocialPlayClient = socialplayClient,
  payments: PaymentsClient = paymentsClient,
): UseHostPaymentsResult {
  const pending = ref<PendingCashPayment[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const markingPaidId = ref<string | null>(null)
  const markPaidError = ref<string | null>(null)

  async function load(hostId: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const { data, error: listError } = await client.GET('/v1/games', {})
      if (listError || !data) {
        error.value = 'Could not load your games. Please try again.'
        return
      }

      const hostGames = (data.games ?? [])
        .map(mapToGameSummary)
        .filter(
          (g) =>
            g.hostId === hostId &&
            (g.paymentMethod === 'PAYMENT_METHOD_CASH' || g.paymentMethod === 'PAYMENT_METHOD_EITHER') &&
            // A FREE game (T9.2) has nothing to collect, so it has no place
            // on a "cash still owed" dashboard. This is a correctness
            // requirement, not a cosmetic filter: RecordOfflinePayment
            // rejects a zero amount (payments domain.NewPayment's
            // `amount.Cents <= 0` -> ErrInvalidAmount), so listing a free
            // game's registrations here would render a "Mark paid" button
            // that could only ever fail.
            g.entryFeeCents > 0,
        )

      const results: PendingCashPayment[] = []
      for (const game of hostGames) {
        const { data: regData, error: regError } = await client.GET('/v1/games/{gameId}/registrations', {
          params: { path: { gameId: game.id } },
        })
        if (regError || !regData) continue
        for (const raw of regData.registrations ?? []) {
          if (raw.paymentStatus !== 'PAYMENT_STATUS_UNPAID') continue
          results.push({
            registrationId: raw.id ?? '',
            gameId: game.id,
            gameHostId: game.hostId,
            playerId: raw.playerId ?? '',
            guestCount: raw.guestCount ?? 0,
            gameLabel: formatGameRange(game.startsAt, game.endsAt),
            entryFeeCents: game.entryFeeCents,
            entryFeeCurrency: game.entryFeeCurrency,
          })
        }
      }
      pending.value = results
    } catch {
      error.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      loading.value = false
    }
  }

  async function markPaid(entry: PendingCashPayment, actorUserId: string): Promise<void> {
    markingPaidId.value = entry.registrationId
    markPaidError.value = null
    try {
      const { data, error: recordError } = await payments.POST('/v1/payments:recordOffline', {
        body: {
          payableType: 'PAYABLE_TYPE_REGISTRATION',
          payableId: entry.registrationId,
          amount: toMoneyRequest(entry.entryFeeCents, entry.entryFeeCurrency || DEFAULT_CURRENCY_CODE),
          actorUserId,
          gameHostId: entry.gameHostId,
        },
      })
      if (recordError || !data?.payment) {
        markPaidError.value = 'Could not record this payment. Please try again.'
        return
      }
      pending.value = pending.value.filter((p) => p.registrationId !== entry.registrationId)
    } catch {
      markPaidError.value = 'Could not reach the server. Check your connection and try again.'
    } finally {
      markingPaidId.value = null
    }
  }

  return { pending, loading, error, markingPaidId, markPaidError, load, markPaid }
}
