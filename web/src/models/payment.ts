// View models for the Payments UI (T8.10, docs/process/t8-sprint-plan.md),
// built from the raw Payments API response (web/src/api/generated/
// payments.d.ts's `components['schemas']`, produced from
// proto/pickleball/payments/v1/payments.proto). Mirrors models/booking.ts's
// shape: a `RawX` type alias per wire message, plus one pure `mapToX`
// function that only ever copies the specific fields it declares.
//
// **Disclosed gap, not silently invented (per this ticket's own process
// instructions):** neither `domain.Game` nor `domain.Registration`
// (internal/socialplay/domain) has ever had a price/fee field — confirmed
// by inspection of internal/socialplay/domain/game.go and registration.go,
// and of every T8.6/T8.7 field added this sprint (PaymentMethod,
// GuestAllowance, GuestCount — no Price/Fee/Amount anywhere). The design
// handoff (docs/design/handoff-2026-08/README.md's Flow 4 "Key fields")
// calls for a real per-Game price, but no ticket before this one ever added
// a backend field to hold one, and adding a priced-Game domain concept is
// out of scope for a Payments-*UI* ticket (it would need its own
// TDD domain+migration+proto cycle, the same class of work T8.6/T8.7 did
// for PaymentMethod/GuestAllowance). `PLACEHOLDER_REGISTRATION_FEE_CENTS`
// is a clearly-labelled, flat stand-in amount used everywhere this ticket
// needs to submit a Money to CreateOnlinePayment/RecordOfflinePayment (both
// require one — it is not an optional field on the wire) — every place it
// is used in the UI shows the word "placeholder" next to it, so this is an
// honest, visible stand-in, not a disguised real price. See this ticket's
// PR description for the full disclosure.
import type { components as PaymentsComponents } from '../api/generated/payments'

export type RawPayment = PaymentsComponents['schemas']['v1Payment']

/** Flat placeholder registration fee (T8.10) — see this file's header
 * comment. $10.00 USD, chosen only because it is an obviously-nominal
 * round number, not a modelled real price. */
export const PLACEHOLDER_REGISTRATION_FEE_CENTS = 1000
export const PLACEHOLDER_CURRENCY_CODE = 'USD'

export interface PaymentSummary {
  id: string
  payableType: string
  payableId: string
  amountCents: number
  currencyCode: string
  method: string
  status: string
  stripeReference: string
  recordedByUserId: string
}

export function mapToPayment(raw: RawPayment): PaymentSummary {
  return {
    id: raw.id ?? '',
    payableType: raw.payableType ?? 'PAYABLE_TYPE_UNSPECIFIED',
    payableId: raw.payableId ?? '',
    amountCents: Number(raw.amount?.amountCents ?? 0),
    currencyCode: raw.amount?.currencyCode ?? '',
    method: raw.method ?? 'PAYMENT_METHOD_UNSPECIFIED',
    status: raw.status ?? 'PAYMENT_STATUS_UNSPECIFIED',
    stripeReference: raw.stripeReference ?? '',
    recordedByUserId: raw.recordedByUserId ?? '',
  }
}

/** Builds the wire `Money` request field. `amount_cents` is an `int64` in
 * the proto, which protojson (and therefore the generated OpenAPI/TS types)
 * represents as a *string* — see models/booking.ts's identical `priceCents`
 * note for the same int64-as-string convention. */
export function toMoneyRequest(cents: number, currencyCode: string = PLACEHOLDER_CURRENCY_CODE): {
  amountCents: string
  currencyCode: string
} {
  return { amountCents: String(cents), currencyCode }
}

/** Formats a cents amount for display (e.g. `1000` -> `"$10.00"`) — mirrors
 * models/booking.ts's `formatPriceCents`, duplicated here rather than
 * imported (models/game.ts's `formatGameRange` sets the same "own copy, no
 * shared formatter module" precedent for this codebase). */
export function formatMoneyCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}
