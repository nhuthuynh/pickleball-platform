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

// EnsureClubRole resolves the real User and checks their real Roles for
// identitydomain.RoleClub.
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
// Note that identityapp.Service.GetUser already answers a malformed id
// exactly like an unknown one (its own uuidShape guard), so both arrive here
// as ErrUserNotFound and leave as domain.ErrUserNotFound.
func translate(id string, err error) error {
	if errors.Is(err, identitydomain.ErrUserNotFound) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("booking identity adapter: %s: %s", id, err)
}
