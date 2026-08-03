package domain

// CameraLink is a link to a facility's camera feed. CourtID is empty for a
// facility-wide link (covers the whole venue) and set to a specific Court's
// ID for a link scoped to that Court only.
type CameraLink struct {
	URL     string
	CourtID string
}

// Facility is the aggregate a Facility Owner creates: a physical venue
// containing one or more Courts, with photos and optional camera links.
// CameraConsentAttested gates AddCameraLink — see its doc comment for the
// round-10 design-review finding this field exists to encode as a domain
// invariant, not just a UI default.
type Facility struct {
	ID                    string
	OwnerID               string
	Name                  string
	Description           string
	Address               string
	PhotoURLs             []string
	CameraLinks           []CameraLink
	CameraConsentAttested bool
}

// NewFacility constructs a Facility, validating the invariants that don't
// require knowledge of other Facilities: OwnerID, Name, and Address must
// all be non-empty. CameraConsentAttested always starts false regardless of
// caller intent — the round-10 design review's finding was specifically
// that consent must default to unchecked (docs/design/
// v1-review-round-10-final.md §2b), so there is deliberately no constructor
// parameter that could set it true on creation. Attesting consent and
// adding a camera link are both separate, explicit steps after
// construction (set the field, then call AddCameraLink).
//
// id is supplied by the caller (the app layer, via its IDGenerator port),
// the same pattern domain.NewBooking/domain.NewPayment use — Facility does
// not generate its own identity.
func NewFacility(id, ownerID, name, description, address string, photoURLs []string) (Facility, error) {
	if ownerID == "" {
		return Facility{}, newFieldError(ErrEmptyFacilityField, "OwnerID")
	}
	if name == "" {
		return Facility{}, newFieldError(ErrEmptyFacilityField, "Name")
	}
	if address == "" {
		return Facility{}, newFieldError(ErrEmptyFacilityField, "Address")
	}
	return Facility{
		ID:          id,
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		Address:     address,
		PhotoURLs:   photoURLs,
	}, nil
}

// EnsureOwner returns ErrNotFacilityOwner unless actorUserID matches
// f.OwnerID exactly (an empty actorUserID is always rejected). This is
// T7.7's object-level (BOLA) authorization check for Facilities' write
// RPCs — mirrors internal/socialplay/domain.Registration.Cancel's
// actorPlayerID-vs-PlayerID check (T5.2/T5.5) applied to Facility's own
// ownership fact instead. As with that precedent, actorUserID is a
// caller-supplied claim, not a verified identity — see
// ErrNotFacilityOwner's doc comment and HANDOFF.md's Auth cross-cutting
// item for the caveat this must not re-litigate.
func (f Facility) EnsureOwner(actorUserID string) error {
	if actorUserID == "" || actorUserID != f.OwnerID {
		return ErrNotFacilityOwner
	}
	return nil
}

// AddCameraLink appends a facility-wide CameraLink (empty CourtID) to the
// Facility, but only once the caller has been proven to own the Facility
// (EnsureOwner, T7.7) and only once CameraConsentAttested is true (the
// round-10 design review's one concrete, no-judgment-call finding,
// v1-review-round-10-final.md §2b: "the consent checkbox must default to
// unchecked... saving is blocked until the user actively checks it",
// translated into an actual domain invariant rather than left as a UI
// concern alone). The ownership check runs first: a non-owner is rejected
// with ErrNotFacilityOwner regardless of the Facility's consent state,
// rather than leaking the consent state to a caller who has no business
// mutating this Facility at all. Returns ErrCameraConsentRequired, and
// leaves CameraLinks untouched, when consent has not been attested.
func (f *Facility) AddCameraLink(actorUserID, url string) error {
	if err := f.EnsureOwner(actorUserID); err != nil {
		return err
	}
	if !f.CameraConsentAttested {
		return ErrCameraConsentRequired
	}
	f.CameraLinks = append(f.CameraLinks, CameraLink{URL: url})
	return nil
}

// AttestCameraConsent sets CameraConsentAttested to true, but only once the
// caller has been proven to own the Facility (EnsureOwner, T7.7's pattern,
// checked first — a non-owner is rejected with ErrNotFacilityOwner without
// ever learning the Facility's current consent state, the same ordering
// AddCameraLink uses). This is T8.4's write path for the field: T7.2/T7.3
// shipped no way to ever set it to true server-side, so every correct
// client submission to AddCameraLink was rejected with
// ErrCameraConsentRequired regardless of what the caller intended — see
// AddCameraLink's doc comment and HANDOFF.md's Cross-cutting section.
// Idempotent: attesting an already-attested Facility is not an error, it
// just confirms the same state (consent isn't a one-way ratchet with
// side effects beyond the boolean itself).
func (f *Facility) AttestCameraConsent(actorUserID string) error {
	if err := f.EnsureOwner(actorUserID); err != nil {
		return err
	}
	f.CameraConsentAttested = true
	return nil
}
