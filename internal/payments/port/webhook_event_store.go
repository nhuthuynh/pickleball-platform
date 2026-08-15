package port

import "context"

// WebhookEventStore is Payments' idempotency ledger for inbound Stripe
// webhook deliveries (T18.1, closes #167): a webhook is redelivered by
// design on any ambiguous response, so a second delivery of the same event
// must be a safe no-op, not a second capture attempt.
type WebhookEventStore interface {
	// ClaimEvent atomically records eventID as seen — an
	// `INSERT INTO payments_webhook_events (event_id) VALUES ($1) ON
	// CONFLICT (event_id) DO NOTHING`-shaped write in the Postgres adapter
	// — and reports whether *this* call is the one that claimed it.
	// claimed == false means eventID was already claimed by an earlier
	// call (this delivery is a redelivery of one already processed, or
	// concurrently being processed), and the caller must not reprocess it.
	ClaimEvent(ctx context.Context, eventID string) (claimed bool, err error)
}
