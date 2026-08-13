-- name: CreateUser :one
-- id is caller-claimed (app.Service.CreateUser's doc comment) rather than
-- server-generated — a second CreateUser call for the same id hits this
-- table's PRIMARY KEY and is translated to domain.ErrUserAlreadyExists by
-- the adapter (CLAUDE.md rule 5), not left as a raw Postgres error.
INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level)
VALUES ($1, $2, $3, $4)
RETURNING id, display_name, roles, self_reported_starting_level;

-- name: GetUserByID :one
SELECT id, display_name, roles, self_reported_starting_level
FROM identity_users
WHERE id = $1;

-- name: UpdateSelfReportedLevel :one
-- T10.2: the write path domain.User.UpdateSelfReportedLevel's EnsureSelf
-- check gates before this is ever called — see port.Repository.
-- UpdateSelfReportedLevel's doc comment.
UPDATE identity_users
SET self_reported_starting_level = $2, updated_at = now()
WHERE id = $1
RETURNING id, display_name, roles, self_reported_starting_level;
