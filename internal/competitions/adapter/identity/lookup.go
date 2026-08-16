// Package identity is the one place Competitions code is allowed to import
// internal/identity/* (CLAUDE.md rule 5). It is the adapter that implements
// internal/competitions/port.IdentityLookup against the real Identity
// context's app.Service, playing exactly the role
// internal/booking/adapter/identity, internal/facilities/adapter/identity,
// and internal/payments/adapter/identity play for their own contexts — same
// shape, separate implementation, deliberately not shared code across
// contexts. internal/competitions/domain and internal/competitions/app
// never see an identitydomain.User or an identityapp.Service directly.
//
// T29.1 is the first time the Competitions context calls into Identity at
// all: before this ticket internal/competitions/port held
// FacilityLookup/CourtReservation/ShareTokenGenerator/IDGenerator/
// CompetitionAdminRepository but none into Identity, and Competitions'
// actor(ctx) (internal/competitions/adapter/grpcapi/handler.go) returned
// auth.RequireSubject(ctx) unresolved — ADR-0014 §5a's explicit, deliberate
// deferral for this context, closed by ADR-0017 and this ticket. The
// dependency introduced here is Competitions -> Identity, one-directional,
// expressed only through the port.
package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"

	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// Lookup implements port.IdentityLookup against Identity's real app.Service,
// translating identitydomain sentinels into Competitions' own context-local
// ones (CLAUDE.md rule 5) — callers on the Competitions side of this
// boundary must never see an identitydomain error type.
type Lookup struct {
	identitySvc *identityapp.Service
}

func NewLookup(identitySvc *identityapp.Service) *Lookup {
	return &Lookup{identitySvc: identitySvc}
}

// UserIDBySubject is ADR-0014/ADR-0017's translation primitive on the
// Competitions side: verified IdP subject in, this platform's User.ID
// (uuid) out.
//
// It is the ONLY place in the Competitions context that accepts a subject.
// The handler's actor() method calls it once per authenticated RPC (via
// app.Service.ResolveActorUserID), and everything below that boundary —
// app, domain, the repository — sees only the uuid. That is the whole of
// #164's Competitions third: the two identifier spaces have been
// deliberately distinct since T12.9 (identitydomain.User.Subject and
// db/migrations/0019_identity_subject.sql), Competitions'
// EnsureHost/EnsureHostOrCompetitionAdmin compared a subject against a
// stored subject "correctly by coincidence" (ADR-0014 §5a) rather than by
// design, and this adapter — together with the same-PR backfill migration
// (ADR-0017) — is where the crossing becomes real rather than coincidental.
//
// The resolution goes through identityapp.Service.UserBySubject, which is
// the documented translation point for exactly this, and NOT through
// GetUser: GetUser keys on identity_users.id and guards on uuidShape, so it
// can never resolve a subject — passing one to it is the #146/#154 mistake,
// one context over.
//
// Only u.ID is returned, never u: port.IdentityLookup's design restriction
// is that no identitydomain aggregate crosses into Competitions, and a uuid
// string is not an aggregate. Nothing on the Competitions side can read
// Roles or SelfReportedStartingLevel through this.
func (l *Lookup) UserIDBySubject(ctx context.Context, subject string) (string, error) {
	u, err := l.identitySvc.UserBySubject(ctx, subject)
	if err != nil {
		return "", translate(subject, err)
	}
	return u.ID, nil
}

// translate maps the one Identity sentinel Competitions has its own name
// for, and strips the original type off anything else — %s, not %w — so a
// caller can never accidentally errors.Is() against an identitydomain
// sentinel from the Competitions side (CLAUDE.md rule 5). This mirrors
// internal/booking/adapter/identity.translate,
// internal/facilities/adapter/identity.translate, and
// internal/payments/adapter/identity.translate, all of which use the same
// non-%w wrapping for their "any other error" case.
//
// An empty subject arrives here as ErrUserNotFound too, and that is correct
// rather than incidental: identityapp.Service.UserBySubject answers an empty
// subject exactly like an unregistered one, applying no uuidShape guard —
// and it must not apply one, because a subject is not a uuid and must never
// be validated as one.
func translate(subject string, err error) error {
	if errors.Is(err, identitydomain.ErrUserNotFound) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("competitions identity adapter: %s: %s", subject, err)
}
