// T29.1 (closes the Competitions third of #164) — this package's shared
// port.IdentityLookup test double, plus the deterministic subject->User.ID
// mapping every other *_test.go file in grpcapi_test needs once handler.go's
// actor() method starts resolving subjects instead of passing them through
// unchanged. Mirrors internal/payments/adapter/grpcapi/
// identity_fixtures_test.go (T28.1) exactly, one context over.
//
// Every existing test in this package authenticated as a literal string
// ("host-1", "attacker", "auth0|host-1", ...) and, before this ticket, that
// SAME string flowed unchanged into Competition.HostID, CompetitionEntry.
// PlayerID, and CompetitionAdmin.UserID/AssignedBy — because actor(ctx)
// returned the raw subject. As of T29.1 it returns a resolved User.ID, so
// every wire-level assertion that compares a response field against the raw
// subject literal needs the SAME resolved value, not the raw subject, to
// keep comparing like with like.
//
// resolvedUserID is a pure, deterministic function rather than a shared
// stateful fake instance threaded through every constructor's return values,
// specifically so each test file's existing helper signatures don't all need
// a new return value: any test body can compute "what will subject X
// resolve to" independently, and get the same answer fakeIdentityLookup's
// own UserIDBySubject does, because both call this same function.
package grpcapi_test

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// resolvedUserID deterministically maps a non-empty subject to a fixed,
// uuidShape-matching User.ID — the value a real port.IdentityLookup would
// resolve it to, if that subject had been registered through Identity's
// CreateUser. Deterministic (not random, not counter-based) so it can be
// called independently, any number of times, from any test file, and always
// agree with itself and with fakeIdentityLookup.UserIDBySubject below,
// without any shared mutable state.
//
// An empty subject maps to "" — mirrors port.IdentityLookup's documented
// convention that an empty subject is unregistered, never a valid input to
// resolve.
func resolvedUserID(subject string) string {
	if subject == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("competitions-grpcapi-test-fixture:" + subject))
	b := sum[:16]
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// fakeIdentityLookup stands in for internal/competitions/adapter/identity,
// returning the same context-local sentinel that adapter translates
// Identity's ErrUserNotFound into (T29.1). It resolves EVERY non-empty,
// non-blocked subject via resolvedUserID above — auto-registration was
// chosen over an explicit subjects map for the same reason
// internal/payments/adapter/grpcapi's identical fixture gives: this
// package's *_test.go files between them authenticate as dozens of distinct
// literal subjects, and requiring each one to be enumerated in one central
// map here would be exactly the kind of change-amplification CLAUDE.md's
// simplicity bar warns against.
//
// unregistered names subjects that must resolve to domain.ErrUserNotFound —
// a verified-but-not-yet-a-User caller.
type fakeIdentityLookup struct {
	unregistered map[string]bool
}

func newFakeIdentityLookup(unregisteredSubjects ...string) *fakeIdentityLookup {
	l := &fakeIdentityLookup{unregistered: make(map[string]bool, len(unregisteredSubjects))}
	for _, s := range unregisteredSubjects {
		l.unregistered[s] = true
	}
	return l
}

func (l *fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	if subject == "" || l.unregistered[subject] {
		return "", domain.ErrUserNotFound
	}
	return resolvedUserID(subject), nil
}
