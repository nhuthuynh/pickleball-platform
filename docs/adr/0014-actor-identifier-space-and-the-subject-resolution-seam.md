# ADR-0014: Actor identifier space — verified subjects are translated to `User.ID` at the grpcapi boundary, not stored

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** T13.2 (`docs/process/t13-sprint-plan.md`)
- **Closes:** #146 (remaining half), #152
- **Relates to:** ADR-0013 (verified caller identity as a platform capability
  — this ADR answers the question ADR-0013 left open the moment it landed),
  ADR-0012 (`identity_users` and the split between claimed and real actors),
  `db/migrations/0019_identity_subject.sql` (which created the two identifier
  spaces this ADR routes between)
- **Consumed by:** T13.3 (Facilities' owner seam, #154), T13.6 (roster read
  authz), T13.7 (`ConfirmOnlinePayment` owner check), T13.8 (`ServiceOptions`)

## Context

Since T12.9 this platform has **two** identifier spaces for one person, and
`db/migrations/0019_identity_subject.sql` explains at length why they must stay
two:

| | `identity_users.subject` | `identity_users.id` |
|---|---|---|
| Owned by | the identity provider | this backend |
| Shape | an arbitrary provider string — `auth0\|abc123`, `google-oauth2\|10769150350006150715`, an opaque pairwise id | a uuid this backend mints |
| Stable under IdP change | no | yes |
| Referenced by other contexts' FKs | never | always |

T12.7/T12.8 then migrated every context's gRPC handler to resolve the actor
from the verified principal. All six contexts converged on the identical
funnel, verified by grep this ticket rather than assumed:

```go
func actor(ctx context.Context) (string, error) {
	return auth.RequireSubject(ctx)   // returns the SUBJECT
}
```

`booking:39`, `facilities:38`, `competitions:48`, `payments:50`,
`socialplay:56`; Identity's handler inlines `auth.PrincipalFromContext` at
`:87` and `:132` instead.

Nothing ever ruled on what happens next. Each context's actor value went from
"a string the caller claimed" to "a subject the IdP verified" — but the columns
those values are compared against and written to did not move, and nobody
decided whether they should. Two live defects are the direct result:

- **#146 / #152 — `RequestRecurringHire` returns an error to every caller
  alive.** `app.Service.RequestRecurringHire` opens with
  `uuidShape.MatchString(in.ActorUserID)`, which cannot match a subject, so it
  returns `ErrUserNotFound` *before* `port.IdentityLookup` is consulted at all.
- **#154 — `CreateFacility` would panic.** It mints `owner_id` from the
  subject; `facilities.owner_id` is `uuid NOT NULL`, written through a
  `mustUUID()` helper that panics on a non-uuid.

These are not two bugs. They are one unmade decision, surfacing twice.

## Decisions

### 1. The ruling: **translate**, do not widen

> A verified `auth.Principal.Subject` is translated to the caller's
> `identity_users.id` (uuid) **at the grpcapi boundary**. Actor columns keep
> the types they have. No column widens to hold a subject, and no subject is
> ever persisted outside `identity_users.subject`.

The alternative — widening actor columns to `text` and storing subjects — is
rejected in §7 below.

### 2. The invariant this creates, stated so it can be checked

> **Below a context's grpcapi boundary, an actor value is always a
> `User.ID` uuid. A subject exists only in `internal/platform/auth`, in
> `identity_users.subject`, and inside each context's `actor()` funnel.**

That single sentence is what the three downstream tickets need, and it is
checkable by reading one function per context. It also restores the
`uuidShape` guards in every app layer to being *honest* malformed-input checks
rather than the stale every-caller-rejecting checks #152 found: after this ADR
an actor reaching `app` genuinely is a uuid, so a non-uuid one genuinely is a
programming error.

### 3. The seam's shape — the pattern T13.3 and everything after it copies

Three pieces, in Booking (`internal/booking`), which is the reference
implementation:

1. **The port gains a resolution method.** `port.IdentityLookup` grows
   `UserIDBySubject(ctx, subject string) (string, error)`. It returns the
   **id string only, never the `User`** — preserving the reason that port
   documented for exposing no `User`-returning method: an aggregate handed
   across a context boundary invites the app layer to start reading another
   context's model. A uuid is not an aggregate.
2. **The app layer exposes the seam as one named method.**
   `app.Service.ResolveActorUserID(ctx, subject) (string, error)` delegates to
   the port the service already holds. It is the *only* method in
   `internal/booking/app` whose parameter is a subject, and it says so in its
   name, so the identifier space is explicit at exactly one point instead of
   being ambiguous at every point.
3. **The handler's `actor()` becomes the funnel.** `actor(ctx)` stops being a
   package function returning `auth.RequireSubject(ctx)` and becomes a
   `*Handler` method that resolves before returning:

   ```go
   func (h *Handler) actor(ctx context.Context) (string, error) {
       subject, err := auth.RequireSubject(ctx)   // Unauthenticated if absent
       if err != nil {
           return "", err
       }
       return h.resolveActor(ctx, subject)        // PermissionDenied if unknown
   }
   ```

**Why the funnel is the important part.** Every actor-taking RPC in a context
already goes through `actor(ctx)` — six call sites in Booking, re-derived by
grep this ticket rather than trusted from #152's list. Putting the translation
there fixes all of them at once and makes forgetting structurally hard: a
handler cannot obtain an actor any other way, and a new RPC that wants one
must call the funnel. The rejected alternative — resolving inside each app
method — has to be remembered six times in Booking alone and again in every
context, which is the exact failure mode that produced #146 and #154.

**Why the boundary and not the app layer, confirmed rather than assumed**
(T13.2 instruction 7). #152 suspected the handler boundary was right because
it matches A11 Ruling 3's precedent. It is right, and there is a stronger
reason than consistency: **the precedent already exists in shipped code.**
`internal/identity/adapter/grpcapi/handler.go`'s `UpdateSelfReportedLevel`
(T12.9, `:137`) resolves `principal.Subject` through
`Service.UserBySubject` and passes `actor.ID` down, with a doc comment saying
"keeping the translation at this boundary is what lets app and domain keep
their existing signatures and stay free of any `internal/platform/auth`
import". This ADR generalises a decision one context already made, rather than
inventing one.

### 4. Why "just delete the guard" is wrong — recorded, because it is the obvious move

Deleting `RequestRecurringHire`'s `uuidShape` guard makes the symptom
(`ErrUserNotFound` for everyone) disappear and makes the system **worse**:

1. The raw subject flows into `domain.NewRecurringHireTemplate` as
   `RequestedByUserID` — which validates only non-emptiness, so it passes.
2. It reaches `internal/booking/adapter/postgres.Repository.Create`, which
   converts it with `mustUUID()`. That helper **panics** on a non-uuid, and
   `grpc` installs no `recover()` of its own beyond
   `internal/platform/grpcrecovery`.
3. Had it survived, `recurring_hire_templates.requested_by_user_id` is
   `uuid NOT NULL REFERENCES identity_users (id)` — an FK violation.

So the trade is: a clean `PermissionDenied` becomes a `500` (or a downed
process) plus a constraint violation, and the actor fact is corrupted in the
one direction that is hardest to migrate back out of. A guard that rejects
every caller is a loud bug; a guard deleted without a translation is a quiet
one. **The guard is not the defect — the missing translation is.** The guards
stay, and this ADR is what makes them true again.

### 5. Per-context ruling — all six, including those needing no code change

Every actor column was re-verified against `db/migrations/` this ticket, not
carried over from the sprint plan.

| Context | Actor fact | Column type | Written from | Conformant today? | Ruling |
|---|---|---|---|---|---|
| **Identity** | `identity_users.id` / `.subject` | `uuid` / `text UNIQUE` | server-minted / IdP | ✅ yes | It owns both spaces and is the only place they may coexist. `UserBySubject` is the sole translation primitive. |
| **Booking** | `recurring_hire_templates.requested_by_user_id` | `uuid NOT NULL REFERENCES identity_users (id)` | `actor(ctx)` | ✅ **as of this ticket** | Translate at the boundary. This ADR's reference implementation. |
| **Facilities** | `facilities.owner_id` | `uuid NOT NULL` (no FK — no target existed when `0010` landed) | `actor(ctx)` | ❌ **broken — #154** | Translate at the boundary, same shape as Booking. **T13.3.** No migration: `0020` stays unclaimed. |
| **Social Play** | `games.host_id`, `registrations.player_id`, `waitlist_entries.player_id` | `text NOT NULL` | `actor(ctx)` | ⚠️ **non-conformant but self-consistent** | See §5a. **Do not resolve in T13.6.** |
| **Competitions** | `competitions.host_id`, `player_id` | `text NOT NULL` | `actor(ctx)` | ⚠️ same as Social Play | Same as Social Play. **Do not resolve in T13.6.** |
| **Payments** | `payments.recorded_by_user_id` | `text` (nullable) | `actor(ctx)` | ⚠️ same as Social Play | Same. **Do not resolve in T13.7.** |

#### 5a. The three text-column contexts: the answer T13.6 and T13.7 must not get wrong

Social Play, Competitions and Payments store actor facts as **plain text**, for
a reason their own migrations state: they were written before an Identity
context existed (`0005_socialplay.sql:4`, `0014_competitions.sql:18`,
`0005_payments.sql:11`, all quoted verbatim from the files this ticket read).

Since T12.8 those columns are written from `actor(ctx)` — so they hold
**subjects**. Their ownership checks also read the actor from `actor(ctx)`. Both
sides of every comparison therefore come from the same space, and the checks
are correct *today* by that coincidence.

> **Ruling for T13.6 and T13.7: compare the value `actor(ctx)` returns against
> the stored value, unchanged. Do NOT add a resolution step to these three
> contexts in T13.** They have no Identity port, and adding one would make the
> actor a uuid on one side of a comparison whose other side is a subject —
> turning a working authorization check into one that silently denies
> everybody, or (worse, on a check written the other way round) silently
> permits.

Conformance for these three is **deliberately deferred**, because it is not a
code change: it needs a backfill of existing rows plus a port and adapter per
context. Tracked as a follow-up issue rather than a paragraph here (T13.2
instruction 12 / A6). This ADR states the target state; it does not pretend
the three contexts are in it.

### 6. The error code for a subject that resolves to no `User`

This is a case ADR-0013 did not cover (T13.2 instruction 11): the token
verified, so we know exactly who the caller is, but no `User` row is
registered to them.

> **Ruling: `domain.ErrUserNotFound` → `codes.PermissionDenied`.**

Justification, in the order the alternatives were eliminated:

- **Not `Unauthenticated`.** ADR-0013 §5 reserves that for "I do not know who
  you are". We do know. The token verified.
- **Not `NotFound`.** Two reasons, and the second is the decisive one. First,
  `NotFound` is used throughout this codebase for the resource the request
  *addresses*; reusing it for the *caller* makes a 404 on
  `RequestRecurringHire` ambiguous between "your Court does not exist" and
  "you do not exist", which a client cannot act on. Second, it would make
  every actor-taking endpoint a user-enumeration oracle.
- **`PermissionDenied` is also what the code already does.** Booking's
  `toStatus` has mapped `ErrUserNotFound` to `PermissionDenied` since T11.5,
  with that same anti-enumeration reasoning recorded on the case itself. This
  ruling reuses an existing mapping rather than inventing one, and it matches
  Identity's own handler, which reports an unregistered caller as
  `ErrNotSelf` → `PermissionDenied` for the identical reason.

**Consequence, stated because it is a behaviour change, not a refactor:**
`ListRecurringHireTemplatesForActor` previously answered an unknown actor with
an empty list. It now answers `PermissionDenied`, because the actor cannot be
resolved before the read is reached. This is the better answer — an
unregistered caller has no templates and no basis to ask — but it is
observable, and it is recorded here rather than discovered later.

### 7. Rejected: widening actor columns to `text` and storing subjects

The alternative had one real advantage — no lookup per request — and was
rejected on four counts:

1. **It leaks the IdP's identifier into six other tables.**
   `0019_identity_subject.sql` already argued this in writing: other contexts'
   references "must not be re-keyed to a string whose format is the identity
   provider's choice — nor leak the IdP identifier into six other tables". This
   ADR would have to overrule a decision made eight days earlier on reasoning
   that has not changed.
2. **It destroys the foreign key.**
   `recurring_hire_templates.requested_by_user_id REFERENCES identity_users (id)`
   is the only real referential integrity any actor column has. A `text` column
   holding subjects can reference nothing, so "this template belongs to a
   deleted user" stops being a database-enforced impossibility — directly
   against CLAUDE.md rule 4's dual-enforcement discipline.
3. **It makes changing identity provider a data migration of the whole
   database** rather than of one column in one table.
4. **It is not actually cheaper.** The role check
   (`EnsureClubRole`) already reads the `User`, so the request already touches
   Identity. Translation adds one keyed read on a `UNIQUE` column, not a new
   round trip class.

## Consequences

**Accepted cost — the resolution is a second lookup.** `RequestRecurringHire`
now reads Identity twice: once by subject (the boundary funnel) and once by id
(`EnsureClubRole`). This is deliberate. Collapsing them means either the port
returns the `User` (leaking the aggregate — see §3.1) or the handler resolves
the role and hands the app layer a privilege boolean, which is precisely the
"role self-assignment" shape the creation-RPC checklist exists to forbid,
merely with the server as the liar instead of the client. Two keyed reads on
indexed columns is the correct price for keeping authorization decisions inside
the app layer. If it ever shows up in a profile, the fix is a request-scoped
cache in the adapter, not a change to this ruling.

**`EnsureClubRole` is keyed on `User.ID` again, and that is not a revert of
#146's fix.** PR #151 changed the adapter to resolve by subject because, at the
time, a subject was what the app layer handed it. Under this ADR the app layer
hands it a uuid, so it resolves by id. The identifier space each port method
accepts is now stated in its own doc comment and pinned by tests in
`internal/booking/adapter/identity/lookup_test.go`, so the two methods cannot
drift back into ambiguity. Resolving the *wrong* space is what #146 was; both
methods now name theirs.

**Booking's Facility-owner checks are correct only once T13.3 lands.**
`ApproveRecurringHire`, `RejectRecurringHire`,
`ListRecurringHireTemplatesForFacility` and `CreateDiscountRule` compare the
actor against `facilities.owner_id`. After this ticket the actor is a uuid —
the right space. But `CreateFacility` still writes a subject into that uuid
column (#154), so no Facility can currently be created through the API at all.
This ADR makes Booking's side right; T13.3 makes the other side right. Neither
half is sufficient alone, which is why the sprint serialises them.

**No migration.** `identity_users.subject text NOT NULL UNIQUE` already exists
(`0019`), which is the entire storage requirement. `0020` is not claimed by
this ticket, and under this ruling T13.3 does not claim it either.

**Constructor signatures are unchanged.** Booking's `app.NewService` keeps its
seven positional parameters; `grpcapi.NewHandler` keeps its one. The seam adds
a *method* to the service, not a dependency to it, because `port.IdentityLookup`
was already a constructor parameter. (Recorded for T13.8, per A13 Gap 3.)

## What this ADR does not decide

- **How the three text-column contexts get to the target state.** Deferred to
  a tracked follow-up (§5a) — it needs a data backfill, not just code.
- **Whether `identity_users.id` should become the FK target for those columns.**
  That is the same follow-up's second half, and depends on the backfill.
- **Anything about role or permission modelling.** This ADR answers *who the
  caller is*, never *what they may do*. Every existing authorization check is
  unchanged by it, deliberately — the test
  `TestRequestRecurringHire_ResolvedActorStillFailsTheClubRoleCheck` exists to
  pin that a resolvable actor is still refused when they hold no `club` role.
