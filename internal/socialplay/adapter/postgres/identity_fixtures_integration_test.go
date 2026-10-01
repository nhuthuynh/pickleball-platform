//go:build integration

// T61 — the shared fixtures this package's integration tests need in order to
// run at all against the current schema, and the record of why they were
// missing.
//
// # What rotted
//
// Migration 0026 (`0026_socialplay_identity_conformance.sql`, T29.2) turned
// games.host_id, registrations.player_id and waitlist_entries.player_id from
// `text NOT NULL` into `uuid NOT NULL REFERENCES identity_users (id)`, and
// `app.ServiceOptions` gained a required `Identity` dependency in the same
// ticket. Every integration test in this package predates that change and
// none was updated: they wired a Service without `Identity` (an immediate
// `panic` from `NewService`) and seeded actor ids like `"host-x"` and
// `"player-07"` (a `mustUUID` panic in the adapter, or a 23503 if it got
// past that).
//
// # Why that was invisible for 32 sprints
//
// No gate on a Docker-free machine executes these files. `make
// vet-integration` COMPILES them, which is what caught the T12 breakage it
// was added for — but a missing map key, a NOT NULL column and a
// constructor's runtime validation are all invisible to the compiler. The
// first full `make ci-integration` run is what surfaced this, and the whole
// class of defect it surfaced is what T61 is.
//
// # Why a stub Identity rather than the real adapter
//
// Not one test in this package drives the subject path: they all pass a
// resolved `PlayerID`/`HostID` straight into an app input, because their
// subject is the Postgres layer below `app`, not the resolution seam above
// it. Wiring `internal/socialplay/adapter/identity` here would mean standing
// up Identity's own app.Service and repository to satisfy a dependency no
// assertion in this package reads — and `internal/socialplay/adapter/
// identity/lookup_test.go` already proves that translation. So the stub
// below satisfies the constructor and nothing else.
//
// It **fails loudly** rather than returning a plausible id: if a future test
// here does reach the subject path, it should discover that this stub does
// not model it, not silently receive a fixture uuid that no identity_users
// row backs.
package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stubIdentityLookup satisfies port.IdentityLookup for tests that never
// resolve a subject. See this file's header for why it answers with an error
// rather than a fixture id.
type stubIdentityLookup struct{}

func (stubIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	return "", errors.New("socialplay/adapter/postgres tests: UserIDBySubject(" + subject +
		") was called, but this package's stub Identity models no subject -> User.ID " +
		"mapping. A test that needs the resolution seam should wire " +
		"internal/socialplay/adapter/identity against a real Identity service, " +
		"or assert at the app layer instead")
}

// seedSocialplayUser inserts an identity_users row and returns its uuid id.
//
// Returning a fresh uuid per call, rather than taking one, is deliberate:
// every actor id in this package's fixtures now has to be both uuid-shaped
// AND present in identity_users, and a helper that owns both halves cannot
// produce a fixture that satisfies one and not the other. `label` is only for
// the display name and the subject, so a failing row is identifiable in a
// container's logs.
func seedSocialplayUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string) string {
	t.Helper()

	id := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES ($1, $2, ARRAY['player'], 3, $3)
	`, id, "T61 "+label, "auth0|t61-socialplay-"+id); err != nil {
		t.Fatalf("seeding identity_users for %s: %v", label, err)
	}
	return id
}
