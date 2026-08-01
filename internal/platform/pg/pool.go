// Package pg wires the shared pgx connection pool used by every context's
// Postgres adapter. Keeping pool construction here (rather than duplicating
// it per context) is the one deliberate exception to "package per bounded
// context" — a connection pool isn't domain state.
package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New opens a pgx pool against dsn (e.g. from DATABASE_URL) and verifies
// connectivity with a ping before returning.
func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pg: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg: ping: %w", err)
	}
	return pool, nil
}
