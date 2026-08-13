// Shared axe-core test helper (T11.7, docs/process/t11-sprint-plan.md — WCAG
// 2.2 AA audit). One shared `expectNoA11yViolations` rather than each spec
// file configuring axe independently, the same "shared helper, not a
// per-screen reimplementation" discipline this project already applies to
// `findGenderControls` (test-support/genderControlAssertions.ts).
//
// Every rule below is disabled for a SPECIFIC, EMPIRICALLY-CONFIRMED reason
// (checked by running axe-core 4.13.0 against this app's own real markup —
// see the reasoning inline — not assumed from a rule's name or tag):
//
// - `color-contrast`: needs real rendered pixel colors, which jsdom (no
//   layout/paint engine) cannot produce. The contrast checks that actually
//   matter (WCAG 1.4.3) are covered instead by computing the real ratios
//   against this project's own contrast-verified design tokens
//   (src/styles/tokens.css, ported from docs/design/v1-review-round-10-
//   final.md) — see the T11.7 PR description for the worked ratios — not by
//   axe in jsdom.
// - `page-has-heading-one` / `landmark-one-main`: BOTH use axe's
//   `has-descendant-evaluate` strategy, which resolves to `incomplete`
//   ("needs manual review" — neither a pass nor a violation) under jsdom
//   REGARDLESS of the actual page content. Confirmed directly, not assumed:
//   run against a route with zero `<h1>` elements AND against a route with
//   exactly one real `<h1>`, both landed in `results.incomplete`, never in
//   `results.passes` or `results.violations`. Leaving these "enabled" would
//   silently provide ZERO protection in this test suite (an `incomplete`
//   result never fails `expectNoA11yViolations`, which only inspects
//   `violations`) — worse than disabling them outright, because it would
//   look covered without being covered. The real requirement each expresses
//   (exactly one top-level heading; exactly one <main> landmark) is instead
//   enforced directly via plain DOM queries in
//   accessibility.spec.ts's `assertPageStructure` — a mechanism proven
//   reliable in this environment, unlike axe's evaluator for these two
//   specific rules.
// - `region`: by contrast, THIS rule resolves normally (a real pass/
//   violation, not `incomplete`) when axe is scoped against the full
//   `document` — confirmed by running it against a real route and getting
//   `results.passes` back. It stays enabled by default for exactly that
//   reason. It only gets explicitly turned off via
//   `COMPONENT_MOUNT_OPTIONS`, for call sites that scope axe to a single
//   mounted component rather than `document` (flagshipFlows.spec.ts) — a
//   standalone component has no surrounding App-shell landmarks by
//   construction, so the rule isn't asking a meaningful question there. (In
//   practice `region`'s `body *` selector is already inapplicable when axe
//   is scoped to a non-document element instead of `document`, so this is
//   currently a documented safeguard more than an active override — kept
//   explicit so it stays correct if a future change re-scopes a
//   component-only spec to run against `document` instead.)
//
// Every other rule (labels, roles, aria-*, live-region shape, form
// association, most landmark/heading-order checks, etc.) resolves reliably
// under jsdom, confirmed the same way: `landmark-main-is-top-level`/
// `landmark-no-duplicate-main`/`landmark-unique` are what actually caught
// this PR's own CompetitionLanding.vue nested-<main> bug as a real
// `violations` entry, not an `incomplete` one.
import axe, { type RunOptions } from 'axe-core'

const DEFAULT_OPTIONS: RunOptions = {
  rules: {
    'color-contrast': { enabled: false },
    'page-has-heading-one': { enabled: false },
    'landmark-one-main': { enabled: false },
  },
}

/** Opt-in override for a STANDALONE component mount (no surrounding App
 * shell) — see file header for exactly which rule this disables and why.
 * Pass as `options` to `expectNoA11yViolations` at component-only call
 * sites; full-page mounts should pass nothing and get the (stricter)
 * default. */
export const COMPONENT_MOUNT_OPTIONS: RunOptions = {
  rules: { region: { enabled: false } },
}

/**
 * Runs axe-core against `node` and throws (failing the test) if it finds any
 * violation, with a readable summary — which rule, which elements, and axe's
 * own fix suggestion — rather than a bare boolean, so a failure is
 * actionable from the test output alone.
 */
export async function expectNoA11yViolations(node: Element | Document, options?: RunOptions): Promise<void> {
  // A plain `{ ...DEFAULT_OPTIONS, ...options }` spread would let a caller's
  // `options.rules` silently REPLACE (not merge with) DEFAULT_OPTIONS.rules,
  // dropping the color-contrast/page-has-heading-one/landmark-one-main
  // disables the moment any caller passes its own rule overrides (exactly
  // what COMPONENT_MOUNT_OPTIONS does) — merge the `rules` maps explicitly
  // instead.
  const results = await axe.run(node, {
    ...DEFAULT_OPTIONS,
    ...options,
    rules: { ...DEFAULT_OPTIONS.rules, ...options?.rules },
  })

  if (results.violations.length > 0) {
    const summary = results.violations
      .map((violation) => {
        const targets = violation.nodes.map((n) => `    - ${n.target.join(' ')}: ${n.failureSummary}`).join('\n')
        return `[${violation.id}] ${violation.help} (${violation.helpUrl})\n${targets}`
      })
      .join('\n\n')
    throw new Error(`axe-core found ${results.violations.length} accessibility violation(s):\n\n${summary}`)
  }
}

/**
 * Direct, jsdom-reliable stand-in for `page-has-heading-one`/
 * `landmark-one-main` — see this file's header comment for why those two
 * axe rules can't be trusted under jsdom and this exists instead. Returns
 * plain counts (rather than asserting itself, matching
 * `findGenderControls`'s own "helper returns data, the caller asserts on
 * it" split) for the real page's top-level headings and `<main>` landmarks,
 * so the caller can assert `=== 1` for each with its own, readable failure
 * message.
 */
export function pageStructureCounts(): { headingCount: number; mainCount: number } {
  return {
    headingCount: document.querySelectorAll('h1:not([role]), [role="heading"][aria-level="1"]').length,
    mainCount: document.querySelectorAll('main:not([role]), [role="main"]').length,
  }
}
