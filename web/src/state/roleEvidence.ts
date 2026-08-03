// Shared, session-scoped "role evidence" state (T8.8,
// docs/process/t8-sprint-plan.md's kickoff note decision #1: "T8 extends
// the existing RoleIndicator mechanism to recognize 'Host' once the
// signed-in browser session has created at least one Game — no new
// mechanism, just a second role fed into the pattern T7.1 already built
// for exactly this").
//
// Before this ticket, RoleIndicator.vue's MOCK_AVAILABLE_ROLES hardcoded
// Host as always present ("a Player who has also hosted a game"). That
// hardcoding is what this module replaces: "Host" evidence now means one
// concrete, real thing — this browser session successfully called
// CreateGame at least once. `src/views/GameCreation.vue` calls
// `recordHostEvidence()` only after a real `CreateGame` response, never
// speculatively (e.g. never on merely opening the form or a failed
// submission).
//
// Reactive (a Vue `ref`), not just a localStorage read-on-mount, so
// RoleIndicator — already mounted in App.vue's persistent header chrome —
// picks up new evidence immediately within the same SPA session, without
// needing a reload. Also persisted to localStorage so the evidence still
// holds after a reload/new tab within the same browser — "for this browser
// session" per the kickoff note, not merely "for this page load".
//
// Same caveat as RoleIndicator.vue's own header comment and every
// ActorUserID call site in this codebase: this is client-side UI state
// only, never a server-verified permission, until the JWT/Auth0
// cross-cutting item lands. Do not use this module to gate anything that
// actually needs authorization.
import { ref } from 'vue'

const HOST_EVIDENCE_KEY = 'pickleball.roleEvidence.host'

function readStoredHostEvidence(): boolean {
  try {
    return typeof localStorage !== 'undefined' && localStorage.getItem(HOST_EVIDENCE_KEY) === 'true'
  } catch {
    // Storage disabled (private browsing, hardened settings, etc.) — treat
    // as "no evidence yet" rather than let this module throw.
    return false
  }
}

/** Whether this browser session has Host evidence yet. Reactive — read it
 * inside a `computed`/template to react to `recordHostEvidence()` calls
 * made elsewhere without a remount. */
export const hasHostEvidence = ref(readStoredHostEvidence())

/**
 * Records that this browser session now has Host evidence. Idempotent:
 * calling it more than once (e.g. a Host publishes a second Game) is a
 * harmless no-op past the first call.
 */
export function recordHostEvidence(): void {
  hasHostEvidence.value = true
  try {
    localStorage.setItem(HOST_EVIDENCE_KEY, 'true')
  } catch {
    // Best-effort persistence only — the reactive ref above already holds
    // for the rest of this session even if storage itself is unavailable.
  }
}

/** Test-only: resets both the reactive ref and localStorage so specs don't
 * leak evidence across each other (this module's state is otherwise
 * module-scoped and shared for the lifetime of the page, by design). */
export function __resetHostEvidenceForTests(): void {
  hasHostEvidence.value = false
  try {
    localStorage.removeItem(HOST_EVIDENCE_KEY)
  } catch {
    // ignore
  }
}
