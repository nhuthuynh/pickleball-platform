package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// RecurringHireRepository is the persistence port for T11.4's
// domain.RecurringHireTemplate (T11.5). It mirrors port.Repository and
// port.DiscountRuleRepository's shape: domain types in, domain types out, and
// domain sentinels for every failure a caller can act on — the Postgres
// adapter translates infrastructure errors (CLAUDE.md rule 5).
type RecurringHireRepository interface {
	// Create persists a newly requested template.
	Create(ctx context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error)

	// GetByID returns a single template, or domain.
	// ErrRecurringHireTemplateNotFound.
	GetByID(ctx context.Context, id string) (domain.RecurringHireTemplate, error)

	// UpdateStatus persists t's Status (the only mutable field on a
	// template: T11.4 models approve/reject as status transitions and
	// nothing else). Named for what it actually writes rather than a generic
	// Update, mirroring the sqlc query it maps onto and port.Repository.
	// Update's own status-only shape for Booking.
	UpdateStatus(ctx context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error)

	// ListForCourts returns every template whose CourtID is in courtIDs,
	// oldest first.
	//
	// Keyed by court rather than by facility because a template *has* no
	// FacilityID — T11.4's aggregate references a Court, and adding a
	// denormalised facility_id column would create a second source of truth
	// for a fact the Facilities context owns. The facility-scoped read
	// (app.Service.ListRecurringHireTemplatesForFacility) resolves the
	// facility's courts through port.FacilityLookup first, so Booking never
	// reads the courts table itself — the same boundary
	// internal/booking/adapter/facilities' doc comment draws.
	//
	// An empty courtIDs slice returns an empty result without a query.
	ListForCourts(ctx context.Context, courtIDs []string) ([]domain.RecurringHireTemplate, error)

	// ListForRequester returns every template requested BY requestedByUserID,
	// oldest first — the Club-facing "my requests" read (T11.6), which T11.5
	// deliberately left unbuilt (see ListRecurringHireTemplatesForFacility's
	// doc comment in the app layer).
	//
	// Named for the domain field it filters on (RequestedByUserID) rather than
	// for the RPC's `actor_user_id` parameter, matching ListForCourts' own
	// "named for what it reads" shape: at this layer there is no actor
	// concept, only the user a template records as having requested it. That
	// the two are the same value is precisely the app layer's authorization
	// statement, and it belongs there, not here.
	//
	// An unknown requester returns an empty result, not an error: no rows is a
	// true and complete answer to "what did this user request".
	ListForRequester(ctx context.Context, requestedByUserID string) ([]domain.RecurringHireTemplate, error)
}
