// Shared copy for the "automated matching isn't available yet" disclosure
// T8.8 (GameCreation.vue), T9.6 (CompetitionCreation.vue), and T9.7
// (CompetitionManage.vue) each independently introduced as a one-line note
// — see those files' own header comments for why the note exists instead
// of a dead matching control.
//
// T10.5 upgrades the wording per ADR-0012 (docs/adr/
// 0012-identity-users-and-match-built-rating-and-matching-algorithm-blocked-on-escalated-decisions.md)
// and centralises it here so the three screens carry the identical
// explanation rather than three independently-edited copies of the same
// sentence drifting apart. Each screen keeps its own original lead clause
// (so existing "players join directly" / "players enter directly" copy,
// and the regression tests asserting it, are undisturbed) and appends this
// constant for the upgraded, precise detail.
//
// Before this ticket, the note said only "isn't available yet" — true but
// no longer precise once Identity/Users is real (T10.1-T10.2): a reader
// could not tell whether "not available yet" meant "unstarted" or
// "one field away from done". This text names both halves of what changed:
//   - what NOW EXISTS: a real Identity/Users profile with a self-reported
//     starting level (T10.1-T10.2, this ticket's own Profile screen).
//   - what's STILL BLOCKED, and on what, specifically: not "matching
//     generally", but two named, escalated product/legal decisions only
//     the platform owner can make (ADR-0012's Q1/Q2) — deliberately not
//     phrased as a timeline the engineering team controls ("coming soon",
//     "next sprint"), since it isn't one.
export const MATCHING_BLOCKED_REASON =
  'Identity now exists, with a real profile and a self-reported starting level. ' +
  "What's still on hold is two decisions only the platform owner can make: how the " +
  'Player Level formula is weighted, and whether gender-mix matching is in scope.'
