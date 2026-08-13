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
// control identified any other way, e.g.:
//   - `<select aria-label="Gender-mix preference">` with no `id`/`name`
//     at all — this codebase already ships this exact shape elsewhere
//     (RoleIndicator.vue's role <select>, id/name-less, aria-label-only).
//   - A <label>-associated control, where the identifying text lives on
//     the <label>, not the input.
//   - A non-native ARIA composite widget, e.g.
//     `<div role="radiogroup" aria-label="Gender">` built from <button>
//     children instead of native <input type="radio">.
// A future PR adding any of those three shapes would have kept
// `genderControls` at length 0 under the old check, silently defeating the
// exact regression it exists to catch. This version checks all four
// signals (native id/name, aria-label on any element, label association,
// and ARIA radiogroup/radio role) so the four screens' checks can't drift
// out of sync with each other either — they all import this one function.
import { DOMWrapper } from '@vue/test-utils'

const GENDER_PATTERN = /gender/i

/** The subset of VueWrapper/DOMWrapper this helper actually needs — kept
 * structural (not `VueWrapper<any>`) so it works identically whether
 * called with the root `mount()` wrapper or a sub-tree DOMWrapper. */
export interface GenderScannable {
  findAll(selector: string): DOMWrapper<Element>[]
}

/**
 * Returns every element under `wrapper` that could plausibly BE a
 * gender-mix matching control, across the four shapes described in this
 * file's header comment. Deliberately over-inclusive: a false positive
 * here costs a five-second look at a failing test; a false negative is
 * exactly the silent regression this exists to prevent from shipping.
 */
export function findGenderControls(wrapper: GenderScannable): DOMWrapper<Element>[] {
  const matches = new Set<Element>()

  for (const el of wrapper.findAll('*')) {
    const tag = el.element.tagName.toLowerCase()
    const id = el.attributes('id') ?? ''
    const name = el.attributes('name') ?? ''
    const ariaLabel = el.attributes('aria-label') ?? ''
    const role = el.attributes('role') ?? ''

    // (1) Native form control, identified by its own id/name — the
    // original check, kept as one signal among several now.
    if (
      (tag === 'input' || tag === 'select' || tag === 'textarea') &&
      (GENDER_PATTERN.test(id) || GENDER_PATTERN.test(name))
    ) {
      matches.add(el.element)
      continue
    }

    // (2) ANY element identified only by its own aria-label — the shape
    // an id/name-only check misses entirely (RoleIndicator.vue's own
    // <select> has no id/name driving its meaning, only aria-label).
    if (GENDER_PATTERN.test(ariaLabel)) {
      matches.add(el.element)
      continue
    }

    // (3) A non-native ARIA composite widget (role="radiogroup"/"radio"
    // built from e.g. <button> children, not native <input type=radio>),
    // identified by its own visible text.
    if ((role === 'radiogroup' || role === 'radio') && GENDER_PATTERN.test(el.text())) {
      matches.add(el.element)
      continue
    }
  }

  // (4) A <label> naming "gender", resolved to its associated control via
  // `for="<id>"` or by containing the control as a descendant. If no
  // control can be resolved (malformed markup), the <label> itself is
  // flagged — a gender-labelled element existing at all is the thing
  // being guarded against, not merely a resolvable one.
  for (const label of wrapper.findAll('label')) {
    if (!GENDER_PATTERN.test(label.text())) continue

    let resolvedAny = false
    const forId = label.attributes('for')
    if (forId) {
      const target = wrapper.findAll('[id]').find((el) => el.attributes('id') === forId)
      if (target) {
        matches.add(target.element)
        resolvedAny = true
      }
    }
    for (const nested of label.findAll('input, select, textarea, [role="radiogroup"], [role="radio"]')) {
      matches.add(nested.element)
      resolvedAny = true
    }
    if (!resolvedAny) matches.add(label.element)
  }

  return Array.from(matches).map((el) => new DOMWrapper(el))
}
