// Package identity is the one place Facilities code is allowed to import
// internal/identity/* (CLAUDE.md rule 5). It is the adapter that implements
// internal/facilities/port.IdentityLookup against the real Identity context's
// app.Service, playing exactly the role internal/booking/adapter/identity
// plays for Booking — same shape, separate implementation, deliberately not
// shared code across contexts. internal/facilities/domain and
// internal/facilities/app never see an identitydomain.User or an
// identityapp.Service directly.
//
// T13.3 is the first time the Facilities context calls into any other context
// at all: before this ticket internal/facilities/port held exactly
// idgenerator.go and repository.go, and nothing under internal/facilities
// imported another bounded context. The dependency introduced here is
// Facilities -> Identity, one-directional, and expressed only through the
// port.
package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// Lookup implements port.IdentityLookup against Identity's real app.Service,
// translating identitydomain sentinels into Facilities' own context-local
// ones (CLAUDE.md rule 5) — callers on the Facilities side of this boundary
// must never see an identitydomain error type.
type Lookup struct {
	identitySvc *identityapp.Service
}

func NewLookup(identitySvc *identityapp.Service) *Lookup {
	return &Lookup{identitySvc: identitySvc}
}

// UserIDBySubject is ADR-0014's translation primitive on the Facilities side:
// verified IdP subject in, this platform's User.ID (uuid) out.
//
// It is the ONLY place in the Facilities context that accepts a subject. The
// handler's actor() funnel calls it once per authenticated RPC, and
// everything below that boundary — app, domain, the repository — sees only
// the uuid. That is the whole of issue #154: the two identifier spaces have
// been deliberately distinct since T12.9 (identitydomain.User.Subject and
// db/migrations/0019_identity_subject.sql), CreateFacility has been writing
// the wrong one into a `uuid NOT NULL` column since T12.7, and this adapter
// sits exactly on the seam, so it is where the crossing belongs.
//
// The resolution goes through identityapp.Service.UserBySubject, which is the
// documented translation point for exactly this, and NOT through GetUser:
// GetUser keys on identity_users.id and guards on uuidShape, so it can never
// resolve a subject — passing one to it is issue #146, one context over.
//
// Only u.ID is returned, never u: port.IdentityLookup's design restriction is
// that no identitydomain aggregate crosses into Facilities, and a uuid string
// is not an aggregate. Nothing on the Facilities side can read Roles or
// SelfReportedStartingLevel through this.
func (l *Lookup) UserIDBySubject(ctx context.Context, subject string) (string, error) {
	u, err := l.identitySvc.UserBySubject(ctx, subject)
	if err != nil {
		return "", translate(subject, err)
	}
	return u.ID, nil
}

// translate maps the one Identity sentinel Facilities has its own name for,
// and strips the original type off anything else — %s, not %w — so a caller
// can never accidentally errors.Is() against an identitydomain sentinel from
// the Facilities side (CLAUDE.md rule 5). This mirrors
// internal/booking/adapter/identity.translate and
// internal/booking/adapter/facilities.translate, both of which use the same
// non-%w wrapping for their "any other error" case.
//
// An empty subject arrives here as ErrUserNotFound too, and that is correct
// rather than incidental: identityapp.Service.UserBySubject answers an empty
// subject exactly like an unregistered one, applying no uuidShape guard — and
// it must not apply one, because a subject is not a uuid and must never be
// validated as one.
func translate(subject string, err error) error {
	if errors.Is(err, identitydomain.ErrUserNotFound) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("facilities identity adapter: %s: %s", subject, err)
}
