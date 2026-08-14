// The reusable "is a control matching THIS concept present anywhere?" scan.
//
// This is `findGenderControls()`'s body, generalized over the pattern it
// matches — not a second, parallel implementation of it. T11.6 needed the
// identical multi-signal check for a different concept (the Club rental
// request control, which must be ABSENT for an actor whose Roles do not
// include `club` — T10 retro finding 5, sprint plan A6), and the sprint plan
// is explicit that this must reuse `findGenderControls()`'s shape rather than
// be "a screen-local reimplementation that only checks one signal type".
// Generalizing the existing helper and having `findGenderControls` delegate to
// it is the only version of that reuse where the two can never drift apart:
// there is exactly one scan, and improving it improves both callers.
//
// Why several signals rather than `id`/`name` (the PR #109 review finding that
// produced the original helper): a control identified any OTHER way silently
// passes an id/name-only check —
//   - `<select aria-label="Gender-mix preference">` with no id/name at all
//     (RoleIndicator.vue already ships exactly this shape);
//   - a `<label>`-associated control, where the identifying text lives on the
//     label, not the input;
//   - a non-native ARIA composite, e.g. `<div role="radiogroup" aria-label=…>`
//     built from `<button>` children.
//
// T11.6 adds a fourth gap the gender-specific version never had to cover,
// because a gender control is a *chooser* while a rental-request control is an
// *action*: a plain `<button>Request a recurring rental</button>` carries no
// id, no name, no aria-label and no <label>, and every signal above would miss
// it. So COMMAND controls (native `<button>`/`<a href>`/`<summary>`, and their
// ARIA equivalents `role="button"`/`role="link"`/`role="menuitem"`/
// `role="tab"`) are matched by their accessible text too. That is a strict
// SUPERSET of the original four signals, so no existing caller loses coverage.
import { DOMWrapper } from '@vue/test-utils'

/** The subset of VueWrapper/DOMWrapper this helper actually needs — kept
 * structural (not `VueWrapper<any>`) so it works identically whether
 * called with the root `mount()` wrapper or a sub-tree DOMWrapper. */
export interface SemanticScannable {
  findAll(selector: string): DOMWrapper<Element>[]
}

/** Elements whose own `id`/`name` identifies them (signal 1). */
const NATIVE_FORM_TAGS = new Set(['input', 'select', 'textarea', 'button'])

/** ARIA roles for a control whose ACCESSIBLE TEXT is what identifies it —
 * both the composite choosers the gender check already covered and the
 * command controls a "must not offer this action" assertion needs. */
const TEXT_IDENTIFIED_ROLES = new Set([
  'radiogroup',
  'radio',
  'button',
  'link',
  'menuitem',
  'menuitemradio',
  'tab',
  'switch',
  'checkbox',
])

/** Native tags carrying an implicit command role, so a plain `<button>` is
 * caught without the author having to spell `role="button"` out. */
const IMPLICIT_COMMAND_TAGS = new Set(['button', 'summary'])

/**
 * Returns every element under `wrapper` that could plausibly BE a control for
 * the concept `pattern` names, across all the identification shapes described
 * in this file's header.
 *
 * Deliberately over-inclusive: a false positive costs a five-second look at a
 * failing test; a false negative is exactly the silent regression this exists
 * to prevent from shipping. Callers assert on the returned list (usually
 * `toHaveLength(0)`); this function itself never asserts, so the same helper
 * serves both the "must be absent" and the "must be present" direction of the
 * same check — which is what lets one test prove the gate really is a gate
 * rather than the control being absent for some unrelated reason.
 */
export function findControlsMatching(
  wrapper: SemanticScannable,
  pattern: RegExp,
): DOMWrapper<Element>[] {
  const matches = new Set<Element>()

  for (const el of wrapper.findAll('*')) {
    const tag = el.element.tagName.toLowerCase()
    const id = el.attributes('id') ?? ''
    const name = el.attributes('name') ?? ''
    const ariaLabel = el.attributes('aria-label') ?? ''
    const role = el.attributes('role') ?? ''

    // (1) Native form control, identified by its own id/name — the original
    // check, kept as one signal among several.
    if (NATIVE_FORM_TAGS.has(tag) && (pattern.test(id) || pattern.test(name))) {
      matches.add(el.element)
      continue
    }

    // (2) ANY element identified only by its own aria-label — the shape an
    // id/name-only check misses entirely.
    if (pattern.test(ariaLabel)) {
      matches.add(el.element)
      continue
    }

    // (3) A control identified by its own accessible TEXT: an explicit ARIA
    // role, or a native tag with an implicit command role (`<button>`,
    // `<summary>`, `<a href>`).
    const isTextIdentifiedControl =
      TEXT_IDENTIFIED_ROLES.has(role) ||
      IMPLICIT_COMMAND_TAGS.has(tag) ||
      (tag === 'a' && el.attributes('href') !== undefined)
    if (isTextIdentifiedControl && pattern.test(el.text())) {
      matches.add(el.element)
      continue
    }
  }

  // (4) A <label> naming the concept, resolved to its associated control via
  // `for="<id>"` or by containing the control as a descendant. If no control
  // can be resolved (malformed markup), the <label> itself is flagged — a
  // matching labelled element existing at all is the thing being guarded
  // against, not merely a resolvable one.
  for (const label of wrapper.findAll('label')) {
    if (!pattern.test(label.text())) continue

    let resolvedAny = false
    const forId = label.attributes('for')
    if (forId) {
      const target = wrapper.findAll('[id]').find((el) => el.attributes('id') === forId)
      if (target) {
        matches.add(target.element)
        resolvedAny = true
      }
    }
    for (const nested of label.findAll(
      'input, select, textarea, button, [role="radiogroup"], [role="radio"]',
    )) {
      matches.add(nested.element)
      resolvedAny = true
    }
    if (!resolvedAny) matches.add(label.element)
  }

  return Array.from(matches).map((el) => new DOMWrapper(el))
}
