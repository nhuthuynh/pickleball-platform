// Shared axe-core test helper (T11.7, docs/process/t11-sprint-plan.md — WCAG
// 2.2 AA audit). One shared `expectNoA11yViolations` rather than each spec
// file configuring axe independently, the same "shared helper, not a
// per-screen reimplementation" discipline this project already applies to
// `findGenderControls` (test-support/genderControlAssertions.ts).
//
// jsdom caveat, stated once here rather than re-derived per call site: jsdom
// has no real layout/paint engine, so axe-core rules that depend on computed
// visual rendering are unreliable in this environment:
//   - `color-contrast` needs actual rendered pixel colors; jsdom cannot
//     produce them, so this rule both false-passes (nothing looks like it has
//     "no color" to check) and would false-fail unpredictably if enabled. The
//     contrast checks that actually matter (WCAG 1.4.3) are covered instead
//     by computing the real ratios against this project's own contrast-
//     verified design tokens (src/styles/tokens.css, ported from
//     docs/design/v1-review-round-10-final.md) — see the T11.7 PR
//     description for the worked ratios — not by axe in jsdom.
//   - `region`/`landmark-*` rules assume a full page landmark structure;
//     component-level mounts (a single .vue file mounted standalone, not the
//     whole App shell) legitimately have none, so these are only run against
//     full-page (App.vue + router) mounts, never per-component ones.
// Every other rule (labels, roles, aria-*, live-region shape, form
// association, heading order, etc.) works the same in jsdom as a real DOM,
// since it only inspects markup/attributes, not paint.
import axe, { type RunOptions } from 'axe-core'

/** Rules that need real layout/paint and are unreliable under jsdom — see
 * file header. Disabled globally rather than per call site so no spec can
 * forget to disable them and get a misleading pass/fail. */
const JSDOM_UNSAFE_RULES = ['color-contrast', 'region', 'landmark-one-main', 'page-has-heading-one']

const DEFAULT_OPTIONS: RunOptions = {
  rules: Object.fromEntries(JSDOM_UNSAFE_RULES.map((id) => [id, { enabled: false }])),
}

/**
 * Runs axe-core against `node` and throws (failing the test) if it finds any
 * violation, with a readable summary — which rule, which elements, and axe's
 * own fix suggestion — rather than a bare boolean, so a failure is
 * actionable from the test output alone.
 */
export async function expectNoA11yViolations(node: Element | Document, options?: RunOptions): Promise<void> {
  const results = await axe.run(node, { ...DEFAULT_OPTIONS, ...options })

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
