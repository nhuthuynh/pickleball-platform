// Player-facing view models for the Discover & Enter Competitions screens
// (T9.7, docs/process/t9-sprint-plan.md), plus the mapping functions that
// build them from the raw Competitions API response
// (web/src/api/generated/competitions.d.ts's `components['schemas']`,
// produced from proto/pickleball/competitions/v1/competitions.proto).
//
// Mirrors models/game.ts's shape: a `RawX` type alias per wire message, plus
// one pure `mapToX` function per view model that only ever copies the
// specific fields it declares (no `...raw` spreads), so a future field added
// to the wire message never leaks into a view unless a mapping function is
// deliberately updated to carry it.
import type { components } from '../api/generated/competitions'
// Reused verbatim from Social Play's view models rather than re-derived:
// PaymentMethod's three values, the "spots left" wording, and the "a zero
// fee is the word Free" rule are one ubiquitous language across contexts
// (CLAUDE.md rule 7), and a second copy of them is exactly how the two
// screens would drift apart. Note this is a reuse of pure *presentation*
// helpers only — nothing here couples the Competitions model to Social
// Play's data, mirroring how the two protos deliberately keep separate Money
// messages while agreeing on what the values mean.
import { paymentMethodLabel, spotsLeftLabel, entryFeeLabel } from './game'

export { paymentMethodLabel, spotsLeftLabel, entryFeeLabel }

export type RawCompetitionListing = components['schemas']['v1CompetitionListing']
export type RawCompetition = components['schemas']['v1Competition']
export type RawCompetitionSession = components['schemas']['v1CompetitionSession']
export type RawCompetitionEntry = components['schemas']['v1CompetitionEntry']

/** How an entrant arrived. The SAME closed vocabulary Social Play's
 * registration source uses, not a third one for the same fact (sprint plan
 * §A3, CLAUDE.md rule 7). The server validates the claimed value rather than
 * inferring it, so the client is the only place the distinction is made. */
export type EntrySource = 'ENTRY_SOURCE_APP' | 'ENTRY_SOURCE_SOCIAL'

/** One sitting of a Competition: a time range plus the courts it reserves
 * for that range. A Competition runs across dates, so this is a LIST on the
 * Competition — the one structural difference from a Game. */
export interface CompetitionSession {
  startsAt: string
  endsAt: string
  courtIds: string[]
}

export interface CompetitionSummary {
  id: string
  hostId: string
  name: string
  venueFacilityId: string
  sessions: CompetitionSession[]
  /** Counts PLACES, not entries — an entrant and each of their guests
   * occupy one place each. */
  capacity: number
  /** The maximum guests a SINGLE entry may bring; 0 means none permitted. */
  guestAllowance: number
  paymentMethod: string
  /** Integer minor units. 0 means FREE — a real, Host-chosen value, never
   * "unset" (see `entryFeeLabel`). */
  entryFeeCents: number
  entryFeeCurrency: string
  format: string
  status: string
  /**
   * Server-computed weighted spots left (capacity minus the sum of
   * `1 + guest_count` across active entries).
   *
   * `null` means **the endpoint this came from does not return it** — not
   * "zero", and not "unknown-so-guess-something". Only `ListCompetitions`
   * carries `spots_left`; `GetCompetition` and `GetCompetitionByShareToken`
   * return a bare `Competition` and deliberately nothing more (their
   * responses are field-for-field identical by design, with a backend
   * regression test pinning that). The UI omits the figure entirely rather
   * than inventing one — see CompetitionDetailPanel.vue.
   */
  spotsLeft: number | null
}

function mapSessions(raw: RawCompetitionSession[] | undefined): CompetitionSession[] {
  return (raw ?? []).map((s) => ({
    startsAt: s.startsAt ?? '',
    endsAt: s.endsAt ?? '',
    courtIds: s.courtIds ?? [],
  }))
}

function mapCompetitionFields(raw: RawCompetition, spotsLeft: number | null): CompetitionSummary {
  return {
    id: raw.id ?? '',
    hostId: raw.hostId ?? '',
    name: raw.name ?? '',
    venueFacilityId: raw.venueFacilityId ?? '',
    sessions: mapSessions(raw.sessions),
    capacity: raw.capacity ?? 0,
    guestAllowance: raw.guestAllowance ?? 0,
    paymentMethod: raw.paymentMethod ?? 'PAYMENT_METHOD_UNSPECIFIED',
    // `amount_cents` is an int64 in the proto, which protojson (and so the
    // generated TS types) represents as a *string* — the same convention
    // models/game.ts and models/payment.ts already document. An absent
    // entry_fee reads as 0 = free, matching the wire default.
    entryFeeCents: Number(raw.entryFee?.amountCents ?? 0),
    entryFeeCurrency: raw.entryFee?.currencyCode ?? '',
    format: raw.format ?? 'COMPETITION_FORMAT_UNSPECIFIED',
    status: raw.status ?? 'COMPETITION_STATUS_UNSPECIFIED',
    spotsLeft,
  }
}

/** From `ListCompetitions` — the one read that carries the server-computed
 * weighted `spots_left`. */
export function mapToCompetitionListing(raw: RawCompetitionListing): CompetitionSummary {
  return mapCompetitionFields(raw.competition ?? {}, raw.spotsLeft ?? 0)
}

/** From `GetCompetition` / `GetCompetitionByShareToken` — a bare
 * Competition, so `spotsLeft` stays `null`. See CompetitionSummary. */
export function mapToCompetition(raw: RawCompetition): CompetitionSummary {
  return mapCompetitionFields(raw, null)
}

/** A confirmed `CompetitionEntry`, as returned by `EnterCompetition`. */
export interface ConfirmedEntry {
  id: string
  competitionId: string
  playerId: string
  guestCount: number
  source: string
  paymentStatus: string
  status: string
}

export function mapToCompetitionEntry(raw: RawCompetitionEntry): ConfirmedEntry {
  return {
    id: raw.id ?? '',
    competitionId: raw.competitionId ?? '',
    playerId: raw.playerId ?? '',
    guestCount: raw.guestCount ?? 0,
    source: raw.source ?? 'ENTRY_SOURCE_UNSPECIFIED',
    paymentStatus: raw.paymentStatus ?? 'PAYMENT_STATUS_UNSPECIFIED',
    status: raw.status ?? 'ENTRY_STATUS_UNSPECIFIED',
  }
}

/**
 * The competition-format label.
 *
 * DESCRIPTIVE ONLY — `CompetitionFormat` enforces nothing server-side (see
 * the enum's doc comment in competitions.proto: an entrant may enter a
 * doubles Competition alone and the server accepts it). It is displayed as a
 * label and never treated as a rule the UI can rely on.
 *
 * WCAG 1.4.1 (Use of Color): every branch returns text, including the
 * unrecognized-value fallback, so the format is never conveyed by styling.
 */
export function competitionFormatLabel(format: string): string {
  switch (format) {
    case 'COMPETITION_FORMAT_SINGLES':
      return 'Singles'
    case 'COMPETITION_FORMAT_DOUBLES':
      return 'Doubles'
    default:
      return 'Format not set'
  }
}

/** Whether this Competition is cancelled. A cancelled Competition is a real,
 * displayable state — a share link to one still resolves (T9.5) and must
 * render as "this was cancelled", never as a broken link. */
export function isCancelled(status: string): boolean {
  return status === 'COMPETITION_STATUS_CANCELLED'
}

/**
 * The EARLIEST session start across a Competition's sessions.
 *
 * This is the single definition of when a Competition "starts" that the
 * backend already uses for both `starts_after`/`starts_before` filtering and
 * result ordering (ListCompetitionsRequest's doc comment). A Competition
 * runs across dates, so the list row has to agree with that definition
 * rather than picking whichever session happens to be first in array order.
 */
export function earliestSessionStart(sessions: CompetitionSession[]): string {
  let earliest = ''
  for (const s of sessions) {
    if (!s.startsAt) continue
    if (!earliest || new Date(s.startsAt).getTime() < new Date(earliest).getTime()) {
      earliest = s.startsAt
    }
  }
  return earliest
}

/** Formats one ISO instant as a date, mirroring models/game.ts's
 * `Intl.DateTimeFormat` approach. */
export function formatCompetitionDate(startsAt: string): string {
  if (!startsAt) return 'Date not set'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(startsAt))
}

/**
 * Formats one session's date and time range for display.
 *
 * An absent range renders as the words "Time not set" rather than
 * `Intl`'s "Invalid Date" — a missing value is stated, never leaked as a
 * formatter artifact.
 */
export function formatSessionRange(session: CompetitionSession): string {
  if (!session.startsAt || !session.endsAt) return 'Time not set'
  const start = new Date(session.startsAt)
  const end = new Date(session.endsAt)
  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
  return `${dateFmt.format(start)}, ${timeFmt.format(start)}–${timeFmt.format(end)}`
}

/** The courts one session reserves, as text. */
export function sessionCourtsLabel(session: CompetitionSession): string {
  return session.courtIds.length > 0 ? session.courtIds.join(', ') : 'Courts not set'
}
