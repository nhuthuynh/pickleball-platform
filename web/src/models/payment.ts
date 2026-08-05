// View models for the Payments UI (T8.10, docs/process/t8-sprint-plan.md),
// built from the raw Payments API response (web/src/api/generated/
// payments.d.ts's `components['schemas']`, produced from
// proto/pickleball/payments/v1/payments.proto). Mirrors models/booking.ts's
// shape: a `RawX` type alias per wire message, plus one pure `mapToX`
// function that only ever copies the specific fields it declares.
//
// T9.2 note: this file previously exported
// `PLACEHOLDER_REGISTRATION_FEE_CENTS`, a flat $10.00 stand-in used
// wherever the UI had to submit a Money, because neither `domain.Game` nor
// `domain.Registration` had a price field at all. That field now exists
// (`domain.Game.EntryFee`, T9.2), so the placeholder — and every "flat
// placeholder rate" label it drove — has been deleted. Amounts now come
// from the Game the player is actually paying for; see
// `models/game.ts`'s `entryFeeCents` / `entryFeeLabel`.
import type { components as PaymentsComponents } from '../api/generated/payments'

export type RawPayment = PaymentsComponents['schemas']['v1Payment']

/**
 * The launch market's currency code (ADR-0005). Not a placeholder: v1 is
 * explicitly a single-currency product, and this is that one real currency
 * — the same constant the `pricing_rules`, `payments`, and (T9.2) `games`
 * currency columns carry. It is the fallback used only when an amount
 * carries no currency of its own (a free Game, which has no currency to
 * name — see domain.Money.Validate).
 *
 * Renamed from `PLACEHOLDER_CURRENCY_CODE` in T9.2: the name was collateral
 * from the retired fee placeholder, and calling the real launch currency a
 * "placeholder" was actively misleading.
 */
export const DEFAULT_CURRENCY_CODE = 'USD'

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
export function toMoneyRequest(cents: number, currencyCode: string = DEFAULT_CURRENCY_CODE): {
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
