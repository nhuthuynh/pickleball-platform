-- name: CreateUser :one
-- T12.9: id is now SERVER-MINTED (app.Service.CreateUser via
-- port.IDGenerator, like every other aggregate in this codebase) and the row
-- is keyed to the caller's verified IdP subject. It used to be the
-- caller-claimed actor_user_id, which is the identity-squatting DoS
-- HANDOFF.md's T10.2 bullet disclosed and T12.9 closed.
--
-- The reachable conflict here is now identity_users.subject's UNIQUE
-- constraint, not the primary key: a second registration for an
-- already-registered subject raises 23505 and the adapter translates it to
-- domain.ErrUserAlreadyExists (CLAUDE.md rule 5), never a raw Postgres
-- error. Because the subject is verified rather than claimed, that
-- collision is always a self-collision.
--
-- Every query returns subject, so a domain.User handed out by this adapter
-- is always fully populated. Selecting it only on some paths would make
-- User.Subject silently empty depending on which query loaded it — the kind
-- of half-built aggregate that turns into an ownership comparison against
-- "" later. It is not exposed on the wire; grpcapi.toProto drops it.
INSERT INTO identity_users (id, subject, display_name, roles, self_reported_starting_level)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, subject, display_name, roles, self_reported_starting_level;

-- name: GetUserByID :one
SELECT id, subject, display_name, roles, self_reported_starting_level
FROM identity_users
WHERE id = $1;

-- name: GetUserBySubject :one
-- T12.9: resolves a verified IdP subject to the User registered to it. The
-- grpcapi boundary uses this to translate an auth.Principal into the actor
-- ID the domain already understands (domain.User.EnsureSelf compares User
-- IDs, not subjects), which is what keeps internal/platform/auth out of the
-- app and domain layers — T12 sprint plan A11 Ruling 3.
--
-- Matched as plain text with no normalisation: an IdP subject is an opaque,
-- case-sensitive, provider-defined string (`auth0|abc123`), so lower()ing or
-- trimming it here would be inventing a rule the issuer never agreed to and
-- could collapse two distinct identities into one.
SELECT id, subject, display_name, roles, self_reported_starting_level
FROM identity_users
WHERE subject = $1;

-- name: UpdateSelfReportedLevel :one
-- T10.2: the write path domain.User.UpdateSelfReportedLevel's EnsureSelf
-- check gates before this is ever called — see port.Repository.
-- UpdateSelfReportedLevel's doc comment. As of T12.9 the actor that check
-- receives is resolved from the verified principal (via GetUserBySubject
-- above), not read off the request.
UPDATE identity_users
SET self_reported_starting_level = $2, updated_at = now()
WHERE id = $1
RETURNING id, subject, display_name, roles, self_reported_starting_level;
