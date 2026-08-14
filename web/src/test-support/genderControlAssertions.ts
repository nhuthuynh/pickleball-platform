// Shared test helper for the "no gender-mix matching control anywhere"
// absence assertion ADR-0012 (docs/adr/
// 0012-identity-users-and-match-built-rating-and-matching-algorithm-blocked-on-escalated-decisions.md)
// requires GameCreation.vue/CompetitionCreation.vue/CompetitionManage.vue/
// Profile.vue's own tests to prove, not just assert by omission.
//
// T10.5 PR #109 review finding: the first version of this check (kept
// independently, copy-pasted, in all four spec files) only inspected
// native <input>/<select> elements' own `id`/`name` attributes via a
// /gender/i regex. That is real but too narrow — it silently passes a
// control identified any other way (aria-label only, <label>-associated,
// or a non-native ARIA composite widget). The multi-signal scan that
// closed that gap now lives in `semanticControlAssertions.ts`, generalized
// over the pattern it matches, because T11.6 needed the identical check
// for a different concept (the Club rental-request control, which must be
// absent for a non-`club` actor — sprint plan A6). This module is the
// gender-specific binding of that one scan: there is exactly one
// implementation, so the two callers cannot drift apart, and any future
// signal added for one is automatically covered for the other.
import type { DOMWrapper } from '@vue/test-utils'
import { findControlsMatching, type SemanticScannable } from './semanticControlAssertions'

const GENDER_PATTERN = /gender/i

/** Retained under its original name (four spec files import it) — the
 * generalized helper's own structural wrapper type. */
export type GenderScannable = SemanticScannable

/**
 * Returns every element under `wrapper` that could plausibly BE a
 * gender-mix matching control, across every identification shape
 * `findControlsMatching` knows about. Deliberately over-inclusive: a false
 * positive here costs a five-second look at a failing test; a false
 * negative is exactly the silent regression this exists to prevent from
 * shipping.
 */
export function findGenderControls(wrapper: GenderScannable): DOMWrapper<Element>[] {
  return findControlsMatching(wrapper, GENDER_PATTERN)
}
