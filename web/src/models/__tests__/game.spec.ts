import { describe, it, expect } from 'vitest'
import { entryFeeLabel, mapToGameSummary } from '../game'

// T9.2. The single behaviour that matters most here: a zero fee is a real
// product state — a free Game — and must read as the WORD "Free", never as
// "$0.00" and never as an empty string. NN/g heuristic #2 (match between
// the system and the real world); the retired
// PLACEHOLDER_REGISTRATION_FEE_CENTS could not express "free" at all.
describe('entryFeeLabel (T9.2)', () => {
  it('renders zero as the word "Free", not "$0.00" and not blank', () => {
    expect(entryFeeLabel(0)).toBe('Free')
  })

  it('renders a real price as a formatted amount', () => {
    expect(entryFeeLabel(1000)).toBe('$10.00')
    expect(entryFeeLabel(1250)).toBe('$12.50')
    expect(entryFeeLabel(1)).toBe('$0.01')
  })

  it('never returns an empty label for any input (WCAG 1.4.1 — always text)', () => {
    for (const cents of [-100, 0, 1, 999999]) {
      expect(entryFeeLabel(cents).trim().length).toBeGreaterThan(0)
    }
  })
})

describe('mapToGameSummary entry fee (T9.2)', () => {
  it('reads the int64-as-string amount off the wire', () => {
    const summary = mapToGameSummary({
      game: { id: 'g1', entryFee: { amountCents: '2500', currencyCode: 'USD' } },
      spotsLeft: 3,
    })
    expect(summary.entryFeeCents).toBe(2500)
    expect(summary.entryFeeCurrency).toBe('USD')
  })

  // An older server, or a Game created before T9.2, sends no entry_fee at
  // all — that reads as free, matching the migration's backfill default
  // rather than becoming NaN and rendering as "$NaN".
  it('treats an absent entry_fee as free, never NaN', () => {
    const summary = mapToGameSummary({ game: { id: 'g1' }, spotsLeft: 3 })
    expect(summary.entryFeeCents).toBe(0)
    expect(entryFeeLabel(summary.entryFeeCents)).toBe('Free')
  })
})
