# ADR-0009: Owned-channel messaging and social-account OAuth custody wait for real authentication

## Status
Accepted as a deferred, tracked decision with a named trigger condition
(not implemented). Produced by T9.8, a spike whose only artifact is this
ADR — no vendor account, no credentials, no adapter package, no proto was
created by it.

**This ADR does not reopen a locked decision, and is not a new position on
one.** `docs/design/v1-system-design.md` §4 and
`docs/design/v1-external-reference-reconciliation.md`'s "One direct,
load-bearing conflict" section already settled *what* this platform builds
for social-driven registration: in-app RSVP through a shareable link, plus
— optionally — a bot on **a channel the platform controls** (a WhatsApp
Business number, a Zalo Official Account, a Telegram/Discord group) parsing
a constrained format such as `IN 2`. Scraping or monitoring public replies
on WhatsApp/Facebook/X/Instagram is not being built, and nothing here
changes that. What this ADR adds is the one thing those documents left
open: ***when***. It extends the locked position with a timing decision and
a trigger condition; it contradicts none of it.

## Context

### What T7 already established (cited, not re-derived)
`docs/process/t7-sprint-plan.md`'s "Social platform integration: WhatsApp
vs. Zalo, researched" section is the research of record. T8's ceremony
re-checked it for staleness and found it **not stale** — its findings are
facts about the platforms, not project state. Summarized here only enough
to make this ADR readable; read that section for the substance:

- The owned-channel-bot pattern is realistic on **both** platforms — each
  lets a business register an official presence it controls, receive
  inbound messages via webhook, and reply programmatically, with no
  scraping involved.
- They differ enough to matter for scoping: WhatsApp Cloud API is
  **message-based** opt-in (a user messaging the number opens the window),
  Zalo OA is **follow-based** (a user must follow the OA before it can
  message them at all); Zalo OA additionally has tiered trial-vs-verified
  account levels gating rate limits and messaging permissions, a heavier
  bring-up cost for a first integration.
- **Market scope is the deciding variable and it is unanswered.** Zalo is
  Vietnam-specific, WhatsApp is global. T7 escalated this to PM/PO as a
  product/go-to-market question and it has not come back. See the Decision
  section — this ADR does not answer it either.
- T7's own recommendation, carried into T8's roadmap and re-affirmed here:
  spike first, produce a `port.MessagingChannel` shape, and prototype
  against **one** platform, not both.

### What T9's ceremony added: the credential-custody argument
T9's plan §A1 ("The social-account-linking split") splits T8's roadmap line
in two. The shareable-registration-link half ships this sprint as **T9.5**
(*Shareable registration links: token-addressed public read + in-app RSVP
with source attribution*), joined by T9.6 (client-side promo composition)
and T9.7 (entry via link). The OAuth half — the platform posting *on a
host's behalf* — is what this ADR defers, and the reason is a
credential-custody argument raised by PE and independently seconded by QA:

> Storing a third-party OAuth access/refresh token keyed to a claimed,
> unverified `actor_user_id` is a categorically worse failure than the
> object-level-authorization caveat this project already carries.

`HANDOFF.md`'s Cross-cutting section records that caveat **three times**,
with the same recorded conclusion each time — it proves the *object-level*
check given a claimed actor, it does not and cannot prove that identity
itself until real auth lands:

1. **T5.5, Social Play** — `actor_player_id` vs. Registration/Game
   ownership (`internal/socialplay/adapter/grpcapi/authz_regression_test.go`).
2. **T6.3/T6.7, later closed by T8.5, Payments** — `actor_user_id` vs.
   Booking / Game-Host / Game-Admin ownership facts
   (`internal/payments/adapter/grpcapi/authz_regression_test.go`,
   `app.authorizeOfflineRecording` / `domain.ErrNotPaymentRecorder`).
3. **T7.7, Facilities** — `actor_user_id` vs. `Facility.OwnerID`.

Every one of those three bounds the blast radius to **this platform's own
data**: a bad actor claiming someone else's ID could cancel a registration,
record a payment, or edit a court. An OAuth token store changes the blast
radius **in kind, not in degree**. The same unverified claim would hand the
caller a live credential that posts to a real person's WhatsApp Business
number, Zalo OA, Facebook Group, X account, or Instagram — on a system with
no authentication, no encryption-at-rest story, and no token-revocation
path. It is the first time the caveat would guard *someone else's property,
outside this system*. That is why this is not "the same caveat a fourth
time" and must not be filed as one.

PE classifies it as a **one-way door**: once real hosts have connected real
accounts, a token store built without auth cannot be un-shipped, and a leak
cannot be un-leaked. The correct response to a one-way door with an
unresolved prerequisite is not to walk through it yet.

## Decision

**1. No OAuth token storage and no inbound messaging integration until real
authentication exists.** No `social_account_links` table, no token column
(encrypted or otherwise), no OAuth callback endpoint, no webhook receiver,
no vendor account, no API key. Not in T9, and not in any sprint before the
trigger condition below is met.

**2. Shareable registration links (T9.5) are the shipped mechanism for
social-driven registration in the meantime.** This is not a consolation
prize: per §A1's decomposition, four of Flow 5's five capabilities —
auto-formatted promo copy, a shareable link that lands in-app, link-driven
registration with guest count and channel attribution, and a roster showing
source channel — need no credential at all. Only "the platform posts on the
host's behalf without the host pasting" does, and only that is deferred
here. The UX must say so honestly rather than draw a dead control: no
"Connect account" button that opens no OAuth flow (§A1's Designer ruling,
agreed by PM).

**3. The port shape is designed on paper, and the package is deliberately
not created.** When this is eventually built, it goes behind
`port.MessagingChannel` — an outbound anti-corruption layer in exactly the
pattern `internal/payments/port.PaymentProcessor` (T6.2) already
established for Stripe: the port is shaped around the **vendor-agnostic
concept**, expressed in the owning context's own primitives, never a vendor
SDK type, so that swapping or adding an adapter is an adapter-only change
and app/domain never move. Per CLAUDE.md rule 5, implementations translate
infra errors into the context's own sentinel domain errors — upper layers
never see a WhatsApp or Zalo error type. As a **sketch only**, the shape
T7 recommended (`SendMessage` / `ParseInboundReply`) would read roughly:

```go
// SKETCH ONLY — ADR-0009 explicitly does NOT create this package or file.
// Reproduced here so the eventual implementer starts from a shape, not a
// blank page. Home package would be internal/<context>/port once the
// owning context is decided (see Consequences).

// MessagingChannel is the outbound anti-corruption layer for an
// owned messaging channel — a WhatsApp Business number or a Zalo
// Official Account the platform itself controls. It is NOT an interface
// over public-reply scraping; the only inbound text it ever sees is a
// message a user deliberately sent to the platform's own channel.
type MessagingChannel interface {
    // SendMessage delivers body to a channel-scoped recipient handle
    // (opaque to domain/app — a phone number or OA user id belongs to
    // the adapter, not to us) and returns the provider's message
    // reference for later correlation.
    SendMessage(ctx context.Context, recipient string, body string) (messageRef string, err error)

    // ParseInboundReply turns one raw inbound message from the owned
    // channel into this context's own primitives — e.g. the constrained
    // `IN 2` format (v1-system-design.md §4) meaning "me plus one
    // guest". Returns a domain sentinel error, never a provider error,
    // when the text doesn't match the constrained grammar.
    ParseInboundReply(ctx context.Context, raw InboundMessage) (ParsedReply, error)
}
```

Naming it now and building it later is the point: per the
`PaymentProcessor` precedent's own comment, treat a port as a promise from
day one — but a promise costs nothing to write down and a package costs
maintenance from the day it exists.

**4. Prototype ONE platform, not both — and which one is still open,
addressed to the user.** The tradeoff is stated once, above, from T7's
research; it is not re-derived here. The choice is gated on a
product/market question this team cannot answer:

> **OPEN QUESTION FOR PM/PO — unanswered since T7, restated here so it is
> discoverable from the ADR index rather than only from a sprint plan:**
> is this platform's v1 user base **Vietnam-concentrated** or **global**?
> Zalo OA only pays for its heavier verification and follow-based opt-in
> cost if the answer is Vietnam-concentrated; WhatsApp Cloud API is the
> default otherwise. This ADR deliberately does **not** pick for you.

Whoever picks up the eventual build ticket must get an answer to that
question first, and must still prototype exactly one platform.

**5. Trigger condition — when this ADR is revisited.** This is revisited in
**the sprint that lands real authentication**, currently recommended as
**T10** (T9 plan §A5's roadmap update: Identity/Users first, because one
context unblocks four open items — auto-matching per ADR-0010, this ADR's
deferred OAuth work, and three sprints of `actor_user_id` caveats).
"Recommended," not decided — T10's own ceremony owns its scope. Revisiting
is *permitted* only once all three of the following actually exist, not
merely planned:

1. **Verified identity.** A real Identity/Users context with authenticated
   sessions — a server-verified subject, not a request-supplied
   `actor_user_id`. Until this exists, every one of the three caveats above
   still stands and a token would be bound to a claim, not a person.
2. **An encrypted token-at-rest story.** A written, reviewed decision on
   where access/refresh tokens live, how they are encrypted at rest, who
   holds the key, and how the key rotates — the same "decide it, don't
   inherit it by default" bar `docs/requirements/research-security-compliance.md`
   applies elsewhere. An ADR of its own, most likely.
3. **A revocation path.** A concrete answer to "a host says revoke, or a
   token leaks — what happens?": host-initiated disconnect,
   provider-initiated invalidation handling, and operator-initiated bulk
   revocation, all before the first real token is stored.

If any of the three is missing when the topic is raised again, the answer
is still no, and this ADR is the reason to point at.

## Consequences

**Pros.** The one-way door stays shut until its prerequisite is met — no
credential belonging to a third party is held by a system that cannot yet
say who is asking. The T9 user-visible value of Flow 5 ships anyway (T9.5 /
T9.6 / T9.7) with zero credential custody, so the deferral costs a
convenience (host pastes the promo themselves) rather than a capability.
The port shape is written down, so the eventual implementer inherits a
decision rather than a blank page and does not re-open the WhatsApp-vs-Zalo
research a fourth time. The still-open market-scope question is now
discoverable from the ADR index instead of living only in a T7 sprint plan
section.

**Cons.** Hosts must copy/paste their own promo posts for at least one more
sprint, and the "connected accounts" area of the design handoff's screen 5
stays un-built with an honest note in its place — a real UX gap, recorded
rather than smoothed over. Automated posting is pushed to T11+ per §A5,
which means any go-to-market plan depending on it needs to know that now.

**Alternative considered and rejected: build the token store now with
encryption, and bolt verified identity on later.** Rejected because
encryption-at-rest does not address the actual failure mode here — the
threat is not a stolen database, it is an unauthenticated caller
successfully claiming to be a host and being handed that host's live
credential by design. Encryption protects against the wrong adversary, and
shipping it would make the one-way door *look* addressed while leaving it
open.

**Alternative considered and rejected: prototype both WhatsApp and Zalo to
"keep options open."** Rejected on T7's own finding — the follow-based vs.
message-based opt-in difference means the two are not one uniform feature,
so building both doubles the integration surface (and the vendor
verification cost) to avoid making one product decision that PM/PO can
simply make. Ask the question instead.

**Related:** ADR-0010 (auto-matching deferred, T9.9) shares this ADR's
trigger — both are waiting on the same Identity/Users context. If T10 lands
that context, both should be revisited in the same sprint, not
independently.
