package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// DiscountRuleRepository is the persistence boundary for T11.1's
// domain.DiscountRule. Unlike PricingRuleRepository (read-only — Booking
// never created pricing rules), this one has a write side: CreateDiscountRule
// is a real RPC, so Facility Owners configure discounts through this context.
//
// Implementors translate infrastructure errors into domain errors
// (CLAUDE.md rule 5); adapter/postgres implements it against the
// discount_rules table (0017_booking_discount_rules.sql), tests implement it
// in memory.
type DiscountRuleRepository interface {
	// Create persists a new DiscountRule. Callers must have already
	// validated it via domain.NewDiscountRule and authorized the actor
	// against the Facility — this method only persists.
	Create(ctx context.Context, r domain.DiscountRule) (domain.DiscountRule, error)

	// ListForFacility returns every DiscountRule configured for facilityID,
	// unfiltered by time or Source — matching the rule set
	// domain.ResolveDiscount expects to be handed (it does its own
	// window/Source matching, exactly as ResolvePrice does for pricing).
	// An unknown facilityID is not an error; it simply has no rules.
	ListForFacility(ctx context.Context, facilityID string) ([]domain.DiscountRule, error)
}
