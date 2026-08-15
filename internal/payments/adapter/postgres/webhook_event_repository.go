package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	paymentsdb "github.com/nhuthuynh/white-label/internal/gen/paymentsdb"
)

// WebhookEventRepository implements port.WebhookEventStore against the
// payments_webhook_events table (T18.1, closes #167; db/migrations/
// 0022_payments_webhook_events.sql). Mirrors CompetitionAdminRepository's
// shape in internal/competitions/adapter/postgres: a separate type from
// Repository, matching the separate port interface, needing no pool of its
// own beyond the Queries handle since ClaimEvent is a single-statement
// write.
type WebhookEventRepository struct {
	q *paymentsdb.Queries
}

func NewWebhookEventRepository(pool *pgxpool.Pool) *WebhookEventRepository {
	return &WebhookEventRepository{q: paymentsdb.New(pool)}
}

// ClaimEvent implements port.WebhookEventStore. The `event_id text PRIMARY
// KEY` constraint (0022) is the authoritative idempotency guard under
// concurrency (CLAUDE.md rule 4's "sufficient on its own" case, see that
// migration's own header comment) — ClaimWebhookEvent's ON CONFLICT DO
// NOTHING makes a redelivered event_id insert zero rows rather than error,
// and :execrows tells this method whether *this* call's insert was the one
// that succeeded (1) or the event was already claimed (0), in one round
// trip, with no separate SELECT-then-INSERT race window.
func (r *WebhookEventRepository) ClaimEvent(ctx context.Context, eventID string) (bool, error) {
	rows, err := r.q.ClaimWebhookEvent(ctx, eventID)
	if err != nil {
		// event_id text PRIMARY KEY has no other constraint that could fire
		// here (see 0022's own header comment) — there is no
		// application-meaningful sentinel to translate a failure into, so
		// this wraps rather than leaks a raw pgconn error unadorned,
		// mirroring translateCompetitionAdminErr's own default branch
		// (CLAUDE.md rule 5).
		return false, fmt.Errorf("payments postgres adapter (webhook events): %w", err)
	}
	return rows == 1, nil
}
