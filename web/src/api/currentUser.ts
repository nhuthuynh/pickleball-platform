// Resolving the signed-in player's own user id (#344).
//
// The bearer token carries the identity provider's `sub`, not this
// platform's `users.id`. Before `GET /v1/users/me` existed, the response to
// `POST /v1/users` was the ONLY place that id ever appeared, so
// `rememberUserId`'s localStorage entry was not a cache — it was the only
// copy. A browser that lost it could not get it back: creating the account
// again answers 409 because it exists, and every screen that needs the id
// (Profile above all) dead-ended. Clearing site data locked a player out of
// their own profile.
//
// This module is the recovery path, and it is deliberately tiny: one call,
// three outcomes, no state of its own beyond what authSession already owns.
import { identityClient } from './identityClient'
import { authUserId, recordProfileMissing, rememberUserId } from '../state/authSession'

export type ResolveOutcome =
  /** The id was already known; no request was made. */
  | 'already-known'
  /** Resolved from the server and stored. */
  | 'resolved'
  /** Authenticated, but this subject has no account yet — the
   *  "finish setting up your profile" banner is now showing. */
  | 'no-profile'
  /** Not signed in, or the call failed. Nothing was stored. */
  | 'unavailable'

/**
 * Makes sure `authUserId` holds this player's own user id, fetching it if
 * the browser does not already have it.
 *
 * Safe to call on every sign-in and every app start: it returns
 * `already-known` without a request when the id is present, so the common
 * path costs nothing.
 *
 * A 404 means the caller is authenticated but has no account, which is a
 * different remedy from a 401 — finish signing up rather than sign in
 * again — so it raises #340's profile banner rather than being reported as
 * a failure. That distinction is why the server answers NotFound here
 * instead of an empty 200; see the GetCurrentUser handler.
 */
export async function ensureCurrentUserId(
  client: Pick<typeof identityClient, 'GET'> = identityClient,
): Promise<ResolveOutcome> {
  if (authUserId.value !== null) return 'already-known'

  const { data, response } = await client.GET('/v1/users/me', {})

  const id = data?.user?.id
  if (id !== undefined) {
    rememberUserId(id)
    return 'resolved'
  }

  // 404 is specifically "no account for this verified subject". Anything
  // else (401, a network failure, a 5xx) is not something this function can
  // interpret, and must NOT raise the profile banner — telling a player to
  // finish signing up because the server was briefly unreachable would send
  // them to a screen that cannot help them.
  if (response?.status === 404) {
    recordProfileMissing()
    return 'no-profile'
  }
  return 'unavailable'
}
