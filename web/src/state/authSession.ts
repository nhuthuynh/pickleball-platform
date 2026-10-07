// Session-scoped bearer-token state — T65.1, the client half of DECISION D1.
//
// # What this fixes, and why it was invisible
//
// `cmd/server` has enforced authentication on 31 RPCs across all six bounded
// contexts since T55.1 (`internal/platform/auth/require.go`, composed in
// `cmd/server/main.go`). Until this module existed the Vue client could not
// send a token at all: `web/src/api/client.ts` passed only `baseUrl` and
// `fetch` to `openapi-fetch`, and `grep -rn "Authorization" web/src` returned
// exactly one hit — a comment. So **every write path in the shipped UI
// returned `Unauthenticated`**: twelve call sites covering booking a court,
// joining a game, paying, entering a competition, creating a facility, adding
// a court, creating a game or a competition, setting a discount, requesting
// and approving a club rental, and editing a profile.
//
// Owed since T55 (2026-09-04) and recorded only in `HANDOFF.md`'s
// "Cross-cutting / later", which no ceremony step reads
// (`grep -n "Cross-cutting" docs/process/sprint-process.md` → 0 matches). No
// gate caught it because all 717 web tests mount a fixture client and `web/`
// has no end-to-end tooling — `CLAUDE.md`'s "an in-memory fake is more
// permissive than Postgres" lesson, with the client as the fake and the real
// gateway as the thing it was more permissive than.
//
// # Scope, stated narrowly
//
// This is the **dev-fixture token path only**, which is what `dev/auth/`
// exists for. There is no redirect flow, no remote JWKS `KeySource`, and no
// real identity provider — those are #145 and #137 and are deliberately out of
// scope (ADR-0013: no coding session here can provision a tenant). A human
// mints a token with `make dev-token` and pastes it into the dev sign-in view.
//
// Reactive (a `ref`) plus `localStorage`, mirroring `state/roleEvidence.ts`
// exactly rather than inventing a second convention: a token must survive a
// reload within the same browser, and the header chrome must notice a sign-in
// without one.
import { ref } from 'vue'

const TOKEN_KEY = 'pickleball.authSession.token'
const USER_ID_KEY = 'pickleball.authSession.userId'

function read(key: string): string | null {
  // Same defensive shape as roleEvidence: a private window, cleared site data
  // or a storage-blocked context must not break rendering.
  try {
    const raw = window.localStorage.getItem(key)
    return raw !== null && raw !== '' ? raw : null
  } catch {
    return null
  }
}

function write(key: string, value: string | null): void {
  try {
    if (value === null) window.localStorage.removeItem(key)
    else window.localStorage.setItem(key, value)
  } catch {
    // Non-fatal: the session still works for this page load.
  }
}

/** The bearer token, or null when nobody is signed in. */
export const authToken = ref<string | null>(read(TOKEN_KEY))

/**
 * The acting user's `identity_users.id`, when known.
 *
 * **Why this is separate from the token, and why it is not derived from it.**
 * The server resolves a token's `sub` claim to a `User.ID` itself (ADR-0014's
 * resolution seam), and **there is no RPC that tells a client its own user
 * id** — Identity exposes `CreateUser`, `GetUser` and
 * `UpdateSelfReportedLevel`, none of them authenticated, and no `GetMe`. So a
 * client that needs a user id for a *domain* field (which player is
 * registering, which owner owns a facility) cannot obtain it from the token.
 *
 * Until a `GetMe` exists this is captured alongside the token in the dev
 * sign-in view. That is honest for a dev-only path and is tracked as a gap
 * rather than hidden: see the issue T65.1 filed.
 */
export const authUserId = ref<string | null>(read(USER_ID_KEY))

/** True when a token is present. Not a permission — the server decides. */
export function isSignedIn(): boolean {
  return authToken.value !== null
}

/** Record a token (and optionally the acting user id) for this browser. */
export function signIn(token: string, userId?: string | null): void {
  const trimmed = token.trim()
  if (trimmed === '') return
  authToken.value = trimmed
  write(TOKEN_KEY, trimmed)
  if (userId !== undefined) {
    const id = userId === null || userId.trim() === '' ? null : userId.trim()
    authUserId.value = id
    write(USER_ID_KEY, id)
  }
}

/** Forget the token and user id. Called by sign-out and on a 401. */
export function signOut(): void {
  authToken.value = null
  authUserId.value = null
  write(TOKEN_KEY, null)
  write(USER_ID_KEY, null)
}

/**
 * Set when any API call comes back 401, so the UI can say "sign in required"
 * once rather than each composable inventing its own message.
 *
 * Deliberately a separate signal from `authToken === null`: "you were never
 * signed in" and "your token was rejected" are different things to tell
 * someone, and the second is the one that needs explaining.
 */
export const authRejected = ref(false)

/** Called by the client middleware on a 401. Clears the stale token. */
export function recordAuthRejected(): void {
  authRejected.value = true
  signOut()
}

/** Cleared when the user acts on the prompt (opens sign-in, dismisses it). */
export function clearAuthRejected(): void {
  authRejected.value = false
}

/**
 * The acting user's id for a **domain** field — which player is registering,
 * which owner owns a facility — falling back to the caller's placeholder when
 * nobody is signed in.
 *
 * **Why a fallback rather than a hard requirement.** Two different things use
 * a user id in this client, and conflating them is what made the old
 * placeholders look harmless:
 *
 *   - the **actor** making the call, which the server now takes from the
 *     verified token and never from the wire (T12.8; `actor_user_id` is
 *     `[deprecated = true]` in the protos). Those fields are being removed.
 *   - a **domain** identifier naming a subject of the operation. The server
 *     cannot infer these, and until a `GetMe` RPC exists the client only knows
 *     one if the dev sign-in captured it.
 *
 * So this helper exists for the second kind: signed in, use the real id;
 * otherwise keep the placeholder the screen already used, so a browse-only
 * session behaves exactly as it did before. Every remaining `MOCK_*` constant
 * is reachable from a call to this function, which is what makes them
 * deletable in one step once a `GetMe` lands.
 */
export function actingUserId(fallback: string): string {
  return authUserId.value ?? fallback
}
