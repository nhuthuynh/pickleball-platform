package grpcapi

import (
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
)

// AuthenticatedMethods lists the Facilities RPCs that may only be called by a
// verified caller. cmd/server composes this with every other context's list
// into one auth.MethodSet (T12 sprint plan A11 Ruling 2: the knowledge of
// which of *this* context's RPCs are public belongs here, next to the handlers
// that break if it is wrong — not in a single list in main.go).
//
// The names come from the generated *_FullMethodName constants rather than
// hand-written strings. A hand-written "/pickleball.facilities.v1.FacilitiesService/AddCourt"
// that drifted from the proto — a renamed RPC, a changed package — would
// silently stop matching and silently stop enforcing, which is the one failure
// mode of a name-keyed policy. Using the constant makes that a compile error.
//
// # Why each of these four requires a principal
//
//   - CreateFacility: establishes ownership. Its owner_id is no longer read
//     off the wire (see the handler); the Facility is owned by the verified
//     caller. Leaving it anonymous would let anyone mint a Facility owned by
//     any subject they cared to name, which is the identity-squatting shape
//     T12.9 closes for identity_users, in a different table.
//   - AddCourt, AddCameraLink, AttestCameraConsent: all three mutate an
//     *existing* Facility and all three are guarded by an owner check that,
//     before this ticket, compared the Facility's owner against a string the
//     caller supplied about themselves.
func AuthenticatedMethods() []string {
	return []string{
		facilitiesv1.FacilitiesService_CreateFacility_FullMethodName,
		facilitiesv1.FacilitiesService_AddCourt_FullMethodName,
		facilitiesv1.FacilitiesService_AddCameraLink_FullMethodName,
		facilitiesv1.FacilitiesService_AttestCameraConsent_FullMethodName,
	}
}

// PublicMethods lists the Facilities RPCs that deliberately stay callable
// without a token.
//
// It exists so the split is *exhaustive and checkable* rather than implied by
// omission. authenticated_test.go asserts that every method in the generated
// ServiceDesc appears in exactly one of these two lists, which turns "a new
// RPC was added and nobody decided whether it needs auth" from a silent
// default-to-public into a failing test. Absence from AuthenticatedMethods is
// what makes a method public at runtime; this list is what makes that absence
// a decision someone made on purpose.
//
// # Why each of these two stays public
//
//   - ListFacilities: the browse path. A player looking for somewhere to play
//     has not signed in yet; requiring a token here would break a shipped
//     flow, which the T12.7 ticket names explicitly as the thing not to do.
//   - GetFacility: the public detail page behind that browse path, returning
//     the same information a facility publishes to attract players (T8.2 added
//     its Courts). It exposes no per-caller data.
func PublicMethods() []string {
	return []string{
		facilitiesv1.FacilitiesService_ListFacilities_FullMethodName,
		facilitiesv1.FacilitiesService_GetFacility_FullMethodName,
	}
}
