// Player-facing view model for the Profile screen (T10.5,
// docs/process/t10-sprint-plan.md), plus the mapping function that builds
// it from the raw Identity/Users API response
// (web/src/api/generated/identity.d.ts's `components['schemas']['v1User']`,
// produced from proto/pickleball/identity/v1/identity.proto).
//
// Explicitly not modelled here, per ADR-0012
// (docs/adr/0012-identity-users-and-match-built-rating-and-matching-algorithm-blocked-on-escalated-decisions.md):
// no PlayerRating/derived-Level field, no Gender field, no matching-mode
// flag — `v1User` itself carries none of these (see identity.proto's own
// doc comment), so there is nothing to deliberately omit here beyond
// stating why, but a future reader adding one to `UserProfile` should read
// that ADR first.
import type { components } from '../api/generated/identity'
import { MIN_SELF_REPORTED_LEVEL, MAX_SELF_REPORTED_LEVEL } from './identityLevel'

export type RawUser = components['schemas']['v1User']

/**
 * A Role exactly as the Identity/Users context defines it — the same closed
 * enum `internal/identity/domain` and `identity.proto` carry, not a
 * client-side re-listing of it (CLAUDE.md rule 7: one ubiquitous language).
 * Added T11.6, whose Club rental screen has to know whether the current actor
 * really holds `club`.
 */
export type Role = components['schemas']['v1Role']

/** The role that gates requesting a recurring hire (T11.5's server-side
 * check, mirrored here only to decide what to SHOW — see `hasClubRole`). */
export const CLUB_ROLE: Role = 'ROLE_CLUB'

/**
 * The roles a User actually holds, from `GetUser`.
 *
 * An absent `roles` array maps to `[]`, never to a guessed default — "we did
 * not learn any roles" and "this user holds no roles" both mean the same
 * thing to a caller, and both must fail CLOSED at every call site.
 */
export function mapToUserRoles(raw: RawUser): Role[] {
  return raw.roles ?? []
}

/**
 * Whether the actor holds the Club role.
 *
 * **This is a rendering decision, never an authorization one.** The real check
 * lives in the backend: `RequestRecurringHire` resolves the actor's Roles
 * server-side and answers PermissionDenied for a non-Club (T11.5, sprint plan
 * A4 checklist item 2). Hiding the control here means a non-Club is not shown
 * an action that would fail; it is not what makes the action safe, and a
 * client that lied about this would still be refused.
 */
export function hasClubRole(roles: Role[]): boolean {
  return roles.includes(CLUB_ROLE)
}

/** The Profile screen's data: a read-only display name plus the editable
 * self-reported starting level. */
export interface UserProfile {
  id: string
  displayName: string
  /** `null` when unset (or, defensively, out of the domain's valid 1..5
   * range — see `identityLevel.ts`'s header) — the honest "no level chosen
   * yet" state (T10.5 instructions #4), never a fabricated default like
   * `1` that would read as a real, player-made choice. */
  selfReportedStartingLevel: number | null
}

/**
 * Maps a raw API `User` to the Profile view model. `raw.selfReportedStartingLevel`
 * is only trusted onto `selfReportedStartingLevel` when it falls within the
 * domain's own valid 1..5 range (`identityLevel.ts`) — the wire's zero
 * value (an omitted/never-set proto field) and any other out-of-range value
 * both map to `null` rather than being displayed as if the player had made
 * a real choice. `domain.NewUser` never actually persists an out-of-range
 * value (T10.1), so this is defense in depth against a partial/malformed
 * response, not a state this codebase's own write path can produce.
 */
export function mapToUserProfile(raw: RawUser): UserProfile {
  const level = raw.selfReportedStartingLevel
  const validLevel =
    typeof level === 'number' && level >= MIN_SELF_REPORTED_LEVEL && level <= MAX_SELF_REPORTED_LEVEL
      ? level
      : null
  return {
    id: raw.id ?? '',
    displayName: raw.displayName ?? '',
    selfReportedStartingLevel: validLevel,
  }
}
