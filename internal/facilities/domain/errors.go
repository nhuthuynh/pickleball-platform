package domain

import (
	"errors"
	"fmt"
)

// Domain errors. Mirrors the sentinel-error pattern used across the other
// bounded contexts (internal/booking/domain/errors.go,
// internal/payments/domain/errors.go) — deliberately not shared with them;
// each bounded context owns its own errors (CLAUDE.md rule 3).
var (
	// ErrEmptyFacilityField is returned by NewFacility when OwnerID, Name,
	// or Address is empty. One sentinel covers all three fields rather than
	// three separate stringly-typed sentinels, mirroring
	// payments.ErrInvalidAmount's single-sentinel-plus-context pattern
	// (T6.1 kickoff note). Use errors.As to recover a *FieldError and
	// inspect its Field member for which field failed.
	ErrEmptyFacilityField = errors.New("facilities: required facility field is empty")

	// ErrEmptyCourtField is NewCourt's equivalent of ErrEmptyFacilityField,
	// for FacilityID/Name. A distinct sentinel from ErrEmptyFacilityField
	// so a caller can tell which aggregate's validation failed via
	// errors.Is, even though both wrap the same *FieldError shape.
	ErrEmptyCourtField = errors.New("facilities: required court field is empty")

	// ErrCameraConsentRequired is returned by Facility.AddCameraLink when
	// CameraConsentAttested is false. This is the round-10 design review's
	// §2b finding ("the consent checkbox must default to unchecked...
	// saving is blocked until the user actively checks it",
	// docs/design/v1-review-round-10-final.md) encoded as a domain
	// invariant rather than left as a UI-only concern — see
	// Facility.AddCameraLink's doc comment.
	ErrCameraConsentRequired = errors.New("facilities: camera consent must be attested before adding a camera link")

	// ErrFacilityNotFound is returned when a Facility lookup by ID has no
	// match. Added in T7.3 (postgres/grpc wiring) alongside the persistence
	// adapter, mirroring booking.ErrBookingNotFound — CLAUDE.md rule 5
	// requires adapters to translate infra errors (e.g. a Postgres
	// pgx.ErrNoRows) into a domain sentinel like this one, not let the raw
	// infra error cross into app/grpcapi.
	ErrFacilityNotFound = errors.New("facilities: facility not found")

	// ErrCourtNotFound is the Court-shaped equivalent of
	// ErrFacilityNotFound, added in T11.2 alongside app.Service.GetCourt —
	// the read Booking's new port.FacilityLookup needs to answer "which
	// Facility owns this Court?". A distinct sentinel from
	// ErrFacilityNotFound so a caller can tell "no such Court" apart from
	// "no such Facility" (they are different 404s, and Booking's adapter
	// translates them into different context-local sentinels).
	ErrCourtNotFound = errors.New("facilities: court not found")

	// ErrNotFacilityOwner is T7.7's object-level (BOLA) authorization
	// sentinel: returned by Facility.EnsureOwner/AddCameraLink when a
	// caller-supplied actor_user_id does not match the Facility's OwnerID.
	// Mirrors internal/socialplay/domain.ErrNotRegistrationOwner (T5.2/
	// T5.5) and internal/payments/domain.ErrNotPaymentRecorder (T6.3): a
	// distinct sentinel (not a generic "unauthorized" string) so
	// grpcapi.toStatus can map it to codes.PermissionDenied (-> HTTP 403)
	// rather than a 500, and so a mismatched actor is distinguishable by
	// type from any other rejection. As with those two precedents, this
	// only proves the *object-level* check given a claimed actor_user_id —
	// it is not itself authentication; see HANDOFF.md's Auth cross-cutting
	// item.
	ErrNotFacilityOwner = errors.New("facilities: actor is not authorized to modify this facility")

	// ErrUserNotFound is Facilities' own, context-local sentinel for the one
	// answer port.IdentityLookup can give (T13.3, Facilities' first call into
	// Identity/Users at all — before this ticket internal/facilities/port
	// held exactly idgenerator.go and repository.go).
	//
	// Like ErrFacilityNotFound and ErrNotFacilityOwner above, it is
	// deliberately NOT internal/identity/domain's sentinel of the same name:
	// CLAUDE.md rule 5 requires the adapter to translate at the boundary so
	// no identitydomain error type ever reaches Facilities' app or grpcapi
	// layers. internal/booking/domain declares its own for the same reason
	// (T11.5) — the duplication is the point, not an oversight.
	//
	// It maps to PermissionDenied at the gRPC boundary, not NotFound. That
	// is ADR-0014 §6's ruling, reused rather than re-decided here: the
	// caller's token verified, so we know exactly who they are and the
	// honest answer is "you may not act", not "you do not exist". Answering
	// NotFound would also make every actor-taking endpoint on this service a
	// user-enumeration oracle, and would make a 404 from AddCourt ambiguous
	// between "no such Facility" and "no such you".
	ErrUserNotFound = errors.New("facilities: user not found")
)

// FieldError wraps one of the Err*Field sentinels above with the specific
// field name that failed validation, so callers get exactly one sentinel to
// errors.Is against (via Unwrap) while still being able to recover which
// field was empty via errors.As. This mirrors the "single sentinel plus
// context" pattern payments.domain already uses for ErrInvalidAmount (T6.1)
// rather than a sprawling per-field enum of sentinels.
type FieldError struct {
	Field string
	err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%v: %s", e.err, e.Field)
}

func (e *FieldError) Unwrap() error {
	return e.err
}

// newFieldError constructs a *FieldError wrapping sentinel and naming
// field, returned as a plain error so call sites don't need to know the
// concrete type.
func newFieldError(sentinel error, field string) error {
	return &FieldError{Field: field, err: sentinel}
}
