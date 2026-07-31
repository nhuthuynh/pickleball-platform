package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// PricingRuleRepository is the read side of the Pricing context's data as
// seen from Booking's Quote use case. Booking only ever reads pricing rules
// (creating/editing them is Pricing/Facilities-context CRUD, out of scope
// for T1) — see HANDOFF.md T1.
type PricingRuleRepository interface {
	ListForCourt(ctx context.Context, courtID string) ([]domain.PricingRule, error)
}
