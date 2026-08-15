// Package identity is the one place Booking code is allowed to import
// internal/identity/* (CLAUDE.md rule 5). It plays the same role for the
// Identity/Users context that internal/booking/adapter/facilities plays for
// Facilities — same shape, separate implementation, deliberately not shared
// code across contexts: it's the adapter that implements
// internal/booking/port.IdentityLookup against the real Identity context's
// app.Service. internal/booking/domain and internal/booking/app never see an
// identitydomain.User or identityapp.Service directly.
//
// T11.5 is the first time the Booking context calls into Identity/Users at
// all (the sprint plan's A8 dependency check confirms it).
package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// Lookup implements port.IdentityLookup against Identity's real app.Service,
// translating identitydomain sentinels into Booking's own context-local ones
// (CLAUDE.md rule 5) — callers on the Booking side of this boundary must never
// see an identitydomain error type.
type Lookup struct {
	identitySvc *identityapp.Service
}

func NewLookup(identitySvc *identityapp.Service) *Lookup {
	return &Lookup{identitySvc: identitySvc}
}

// UserIDBySubject is ADR-0014's translation primitive on the Booking side:
// verified IdP subject in, this platform's User.ID (uuid) out.
//
// It is the ONLY method here that accepts a subject. The handler's actor()
// funnel calls it once per authenticated RPC, and everything below that
// boundary — app, domain, both repositories — sees only the uuid. That is the
// whole of issues #146 and #152: the two identifier spaces have been
// deliberately distinct since T12.9 (identitydomain.User.Subject and
// db/migrations/0019_identity_subject.sql), and this adapter sits exactly on
// the seam, so it is where the crossing belongs.
//
// The resolution goes through identityapp.Service.UserBySubject, which is the
// documented translation point for exactly this, and NOT through GetUser:
// GetUser keys on identity_users.id and guards on uuidShape, so it can never
// resolve a subject — passing one to it returned ErrUserNotFound for every
// caller alive, which is issue #146.
//
// Only u.ID is returned, never u: port.IdentityLookup's design restriction is
// that no identitydomain aggregate crosses into Booking, and a uuid string is
// not an aggregate. Nothing on the Booking side can read Roles or
// SelfReportedStartingLevel through this.
func (l *Lookup) UserIDBySubject(ctx context.Context, subject string) (string, error) {
	u, err := l.identitySvc.UserBySubject(ctx, subject)
	if err != nil {
		return "", translate(subject, err)
	}
	return u.ID, nil
}

// EnsureClubRole resolves the real User and checks their real Roles for
// identitydomain.RoleClub.
//
// actorUserID is a **User.ID** (uuid), not a subject — so the resolution goes
// through identityapp.Service.GetUser, which keys on identity_users.id.
//
// **This is not a revert of the #146 fix, and the distinction matters enough
// to spell out.** PR #151 changed this method to resolve by subject because
// at that time the app layer handed it a subject: T12.7 made the actor
// auth.RequireSubject(ctx) and nothing translated it. ADR-0014 moved the
// translation up to the grpcapi boundary (see UserIDBySubject above), so the
// value arriving here is now a uuid, and resolving it by subject would fail
// for every caller — the same defect as #146, mirrored. #146 was never "use
// UserBySubject"; it was "resolve the space you are actually given". Each
// method on this adapter now names its space, and lookup_test.go pins both
// directions so they cannot drift back into ambiguity.
//
// This is the server-side half of the creation-RPC checklist's role
// self-assignment item (T10 retro finding 3, sprint plan A4): the `club` fact
// is read out of Identity's own store, never taken from the request. The User
// itself is discarded — port.IdentityLookup deliberately has no method that
// could return one, so nothing on the Booking side can start reading another
// context's aggregate.
//
// The role comparison is made here against identitydomain.RoleClub rather than
// against a string literal on the Booking side, so the two contexts cannot
// drift on what the role is called; Identity's Role enum stays the single
// definition (CLAUDE.md rule 7). Unlike facilities' EnsureFacilityOwner, there
// is no identitydomain method to delegate the decision to — User has
// EnsureSelf but no role predicate — so the check is spelled out here, in the
// adapter, which is the only Booking-side code allowed to see Identity's
// types at all.
func (l *Lookup) EnsureClubRole(ctx context.Context, actorUserID string) error {
	u, err := l.identitySvc.GetUser(ctx, actorUserID)
	if err != nil {
		return translate(actorUserID, err)
	}
	for _, r := range u.Roles {
		if r == identitydomain.RoleClub {
			return nil
		}
	}
	return domain.ErrNotClub
}

// translate maps the Identity sentinels Booking has its own names for, and
// strips the original type off anything else — %s, not %w — so a caller can
// never accidentally errors.Is() against an identitydomain sentinel from the
// Booking side (CLAUDE.md rule 5), mirroring
// internal/booking/adapter/facilities.translate's identical non-%w wrapping
// for its own "any other error" case.
//
// Both callers funnel their "no such User" case through here and both get
// domain.ErrUserNotFound, from opposite directions:
//
//   - identityapp.Service.UserBySubject answers an empty subject exactly like
//     an unregistered one. It applies no uuidShape guard, and must not: a
//     subject is not a uuid and must never be validated as one.
//   - identityapp.Service.GetUser DOES guard on uuidShape, and answers a
//     malformed id exactly like an unknown one — so a subject accidentally
//     reaching EnsureClubRole is refused rather than resolved. That guard is
//     load-bearing under ADR-0014: it is what makes a seam bypass fail
//     closed and visibly, instead of silently authorizing the wrong person.
func translate(id string, err error) error {
	if errors.Is(err, identitydomain.ErrUserNotFound) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("booking identity adapter: %s: %s", id, err)
}
