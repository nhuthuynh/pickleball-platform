// Mirrors internal/identity/domain/level.go's `SelfReportedStartingLevel`
// bounds exactly (1..5) — kept as the single source of truth on this side
// so `identity.ts`'s mapping and `Profile.vue`'s edit control can't drift
// apart from each other or from the domain.
export const MIN_SELF_REPORTED_LEVEL = 1
export const MAX_SELF_REPORTED_LEVEL = 5
