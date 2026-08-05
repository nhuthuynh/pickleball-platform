// Host-facing view models for the Competitions screens (T9.6,
// docs/process/t9-sprint-plan.md), built from the raw Competitions API
// response (web/src/api/generated/competitions.d.ts's
// `components['schemas']`, produced from
// proto/pickleball/competitions/v1/competitions.proto).
//
// Mirrors models/game.ts / models/facility.ts / models/booking.ts's shape: a
// `RawX` type alias per wire message, plus one pure `mapToX` function per
// view model that only ever copies the specific fields it declares (no
// `...raw` spreads), so a future field added to the wire message never
// leaks into a view unless a mapping function is deliberately updated to
// carry it.
//
// Every label function here returns TEXT on every branch, including its
// fallback — WCAG 1.4.1 (Use of Color): payment status, entry source, and
// format are never conveyed by styling alone anywhere in the Competitions
// UI. Same rule models/game.ts's `paymentMethodLabel`/`spotsLeftLabel`
// already apply for Social Play.
import type { components } from '../api/generated/competitions'

export type RawCompetition = components['schemas']['v1Competition']
export type RawCompetitionEntry = components['schemas']['v1CompetitionEntry']
export type RawCompetitionSession = components['schemas']['v1CompetitionSession']

/** One sitting of a Competition, as it comes off the wire. A value type
 * inside the Competition aggregate — deliberately no id, mirroring
 * `CompetitionSession`'s own proto doc comment. */
export interface CompetitionSessionSummary {
  startsAt: string
  endsAt: string
  courtIds: string[]
}

export interface CompetitionSummary {
  id: string
  hostId: string
  name: string
  venueFacilityId: string
  sessions: CompetitionSessionSummary[]
  capacity: number
  guestAllowance: number
  paymentMethod: string
  /** Integer minor units. 0 means the Competition is FREE — a real,
   * Host-chosen value, never "unset" (see `Money`'s proto doc comment and
   * `entryFeeLabel`). */
  entryFeeCents: number
  entryFeeCurrency: string
  format: string
  status: string
}

export function mapToCompetitionSession(raw: RawCompetitionSession): CompetitionSessionSummary {
  return {
    startsAt: raw.startsAt ?? '',
    endsAt: raw.endsAt ?? '',
    courtIds: raw.courtIds ?? [],
  }
}

export function mapToCompetitionSummary(raw: RawCompetition): CompetitionSummary {
  return {
    id: raw.id ?? '',
    hostId: raw.hostId ?? '',
    name: raw.name ?? '',
    venueFacilityId: raw.venueFacilityId ?? '',
    sessions: (raw.sessions ?? []).map(mapToCompetitionSession),
    capacity: raw.capacity ?? 0,
    guestAllowance: raw.guestAllowance ?? 0,
    paymentMethod: raw.paymentMethod ?? 'PAYMENT_METHOD_UNSPECIFIED',
    // `amountCents` is an int64 in the proto, which protojson (and so the
    // generated TS types) represents as a *string* — same convention
    // models/payment.ts's `toMoneyRequest` documents.
    entryFeeCents: Number(raw.entryFee?.amountCents ?? 0),
    entryFeeCurrency: raw.entryFee?.currencyCode ?? '',
    format: raw.format ?? 'COMPETITION_FORMAT_UNSPECIFIED',
    status: raw.status ?? 'COMPETITION_STATUS_UNSPECIFIED',
  }
}

export interface CompetitionEntrySummary {
  id: string
  competitionId: string
  playerId: string
  guestCount: number
  source: string
  paymentStatus: string
  status: string
}

export function mapToCompetitionEntry(raw: RawCompetitionEntry): CompetitionEntrySummary {
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
 * The Competition's format, as a display label.
 *
 * DESCRIPTIVE ONLY, exactly as `CompetitionFormat`'s proto doc comment
 * insists: it enforces nothing, and the UI must never present it as a rule
 * the backend upholds (no partner-pairing field, no bracket, no seeding —
 * none of that exists in T9).
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

/**
 * How an entrant arrived, as a display label.
 *
 * EXACTLY TWO real values, because `EntrySource` has exactly two: an entry
 * happened in the app, or it happened via a shareable registration link the
 * Host published (T9.5). The label deliberately does NOT name a third-party
 * platform.
 *
 * `docs/design/v1-external-reference-reconciliation.md` is explicit about
 * this: the design attachment's wireframe labels this list "via WhatsApp
 * reply" / "via Facebook reply", drawing reply-scraping that is not built
 * and — per ADR-0009 — is not being built. Copying that phrasing would
 * claim a fact the backend does not have (nothing anywhere records which
 * channel a link was pasted into) and would advertise an integration that
 * does not exist. "Via shared link" is the true, complete statement of what
 * `ENTRY_SOURCE_SOCIAL` means.
 */
export function entrySourceLabel(source: string): string {
  switch (source) {
    case 'ENTRY_SOURCE_APP':
      return 'In app'
    case 'ENTRY_SOURCE_SOCIAL':
      return 'Via shared link'
    default:
      return 'Entry source not set'
  }
}

/**
 * An entry's payment status, as a display label.
 *
 * "Unpaid" and "Refunded" stay distinct because the domain keeps them
 * distinct — "never paid" and "paid, then refunded" are different facts a
 * Host reconciling money must be able to tell apart (see `PaymentStatus`'s
 * proto doc comment).
 */
export function entryPaymentStatusLabel(status: string): string {
  switch (status) {
    case 'PAYMENT_STATUS_UNPAID':
      return 'Unpaid'
    case 'PAYMENT_STATUS_PAID':
      return 'Paid'
    case 'PAYMENT_STATUS_REFUNDED':
      return 'Refunded'
    default:
      return 'Payment status not set'
  }
}

/** An entry's lifecycle status, as a display label. The roster read returns
 * cancelled entries deliberately ("who withdrew" and "who never entered"
 * are different answers — see `ListEntriesForCompetitionRequest`), so this
 * label has to be able to say so. */
export function entryStatusLabel(status: string): string {
  switch (status) {
    case 'ENTRY_STATUS_ENTERED':
      return 'Entered'
    case 'ENTRY_STATUS_CANCELLED':
      return 'Withdrawn'
    default:
      return 'Entry status not set'
  }
}

/** Formats one session's date + time range for display — Competitions' own
 * copy of the `Intl.DateTimeFormat` approach `models/game.ts`'s
 * `formatGameRange` uses (that file's own comment establishes the
 * "own copy, no shared formatter module" precedent for this codebase). */
export function formatSessionRange(startsAt: string, endsAt: string): string {
  if (!startsAt || !endsAt) return ''
  const start = new Date(startsAt)
  const end = new Date(endsAt)
  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const timeFmt = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
  return `${dateFmt.format(start)}, ${timeFmt.format(start)}–${timeFmt.format(end)}`
}

/**
 * The span a Competition covers, from its earliest session start to its
 * latest session end — the "dates" line of the promo (a Competition runs
 * across dates, which is the whole structural difference from a Game).
 *
 * Returns an empty string when there are no sessions: with nothing on the
 * wire to derive a date from, the honest output is nothing at all, not a
 * fabricated placeholder.
 */
export function competitionDatesLabel(sessions: CompetitionSessionSummary[]): string {
  const starts = sessions.map((s) => new Date(s.startsAt).getTime()).filter((t) => !Number.isNaN(t))
  const ends = sessions.map((s) => new Date(s.endsAt).getTime()).filter((t) => !Number.isNaN(t))
  if (starts.length === 0 || ends.length === 0) return ''

  const dateFmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  const first = dateFmt.format(new Date(Math.min(...starts)))
  const last = dateFmt.format(new Date(Math.max(...ends)))
  return first === last ? first : `${first} – ${last}`
}

/**
 * One row of the create form's sessions step, in the shape the Host types
 * it: a calendar date plus a start and end time, and the courts that
 * sitting reserves.
 *
 * Deliberately date + time rather than two `datetime-local` inputs (which
 * is what a Game's single range uses): a Competition's rows are read down a
 * column as "which days is this on", and repeating the date inside both
 * bounds of every row makes that scan harder for no gain.
 */
export interface DraftSession {
  date: string
  startTime: string
  endTime: string
  courtIds: string[]
}

/** The instants a draft row denotes, or null when the row is not yet
 * complete enough to denote any (so an incomplete row never becomes an
 * `Invalid Date` on the wire or a phantom overlap in the check below). */
export function draftSessionRange(session: DraftSession): { start: number; end: number } | null {
  if (!session.date || !session.startTime || !session.endTime) return null
  const start = new Date(`${session.date}T${session.startTime}`).getTime()
  const end = new Date(`${session.date}T${session.endTime}`).getTime()
  if (Number.isNaN(start) || Number.isNaN(end)) return null
  return { start, end }
}

/** The wire `CompetitionSession` for a draft row, or null if incomplete. */
export function draftSessionToWire(
  session: DraftSession,
): { startsAt: string; endsAt: string; courtIds: string[] } | null {
  const range = draftSessionRange(session)
  if (!range) return null
  return {
    startsAt: new Date(range.start).toISOString(),
    endsAt: new Date(range.end).toISOString(),
    courtIds: [...session.courtIds],
  }
}

/**
 * Which draft rows double-book a court against another row (or themselves).
 *
 * A CLIENT-SIDE MIRROR of `internal/competitions/domain`'s
 * `ensureNoSessionOverlap` (T9.1), reproducing its rule exactly:
 *   - only reservations sharing a court can conflict;
 *   - ranges are half-open [start, end), so back-to-back sittings on one
 *     court are fine;
 *   - a court listed twice inside a single row conflicts with itself.
 *
 * THE SERVER CHECK REMAINS AUTHORITATIVE. This exists only so the Host is
 * told at entry, next to the offending row, rather than after a round trip
 * that discards their whole form (NN/g heuristic #5 — error prevention
 * beats error correction). It is not a substitute for
 * `domain.ErrOverlappingSessions`, and `CompetitionCreation.vue` still maps
 * that server rejection onto the same rows.
 *
 * Returns the indices of every row involved, so the message can sit beside
 * each one rather than as a single form-level banner that doesn't say which
 * rows are at fault.
 */
export function findOverlappingSessionIndices(sessions: DraftSession[]): Set<number> {
  const offending = new Set<number>()
  // Same plain O(n²) walk over (session, court) reservations the domain
  // uses, and for the same reason: a handful of rows, run on keystroke,
  // clarity worth more than imaginary speed.
  const seen: { index: number; courtId: string; start: number; end: number }[] = []

  sessions.forEach((session, index) => {
    const range = draftSessionRange(session)
    if (!range) return
    for (const courtId of session.courtIds) {
      for (const prev of seen) {
        if (prev.courtId !== courtId) continue
        // Half-open overlap: [a.start, a.end) ∩ [b.start, b.end) ≠ ∅.
        if (prev.start < range.end && range.start < prev.end) {
          offending.add(prev.index)
          offending.add(index)
        }
      }
      seen.push({ index, courtId, start: range.start, end: range.end })
    }
  })

  return offending
}

/**
 * The URL a Host actually posts, built from the REAL `share_token`
 * `CreateCompetition` returned.
 *
 * `/c/:shareToken` is T9.7's deep-link landing route — the one that calls
 * `GetCompetitionByShareToken`. Empty token in, empty string out: with no
 * token there is no link, and rendering a copyable URL that resolves to
 * nothing would be worse than rendering none.
 *
 * The token is a CAPABILITY, not an identifier (see
 * `GetCompetitionByShareTokenRequest`'s proto doc comment) — so it appears
 * only in this URL, and is never logged or put in a page title.
 */
export function shareUrlForToken(shareToken: string, origin: string): string {
  if (!shareToken) return ''
  return `${origin.replace(/\/$/, '')}/c/${shareToken}`
}
