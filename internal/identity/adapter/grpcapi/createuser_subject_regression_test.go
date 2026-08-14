// T12.9 — security regression test for the CreateUser identity-squatting
// denial-of-service that HANDOFF.md's T10.2 bullet disclosed and this
// ticket closes.
//
// # The original attack this file reproduces
//
// Before T12.9, CreateUserRequest.actor_user_id BECAME the new row's own
// permanent primary key (identity_users.id). No token was required and
// nothing was verified, so any anonymous caller could pick any UUID —
// including one a real-auth integration would later mint deterministically
// for a real person — and permanently occupy that identity. The real
// owner's later registration then failed with ErrUserAlreadyExists and they
// could never claim their own account. HANDOFF.md called that "a
// persistent, targeted denial-of-service, not a rejected mutation", with a
// closure condition in its own words: "Must close the moment real auth
// exists."
//
// Real auth exists (T12.2, internal/platform/auth). This file is the proof
// the gap is closed rather than the assertion that it is (CLAUDE.md rule
// 10): TestCreateUser_AnonymousSquatterIsRejected replays the exact attack,
// TestCreateUser_LegitimateSubjectOwnerCanRegister proves the fix did not
// simply break registration for everyone, and
// TestCreateUser_PrincipalOverridesWireActorClaim proves the handler
// ignores the wire field rather than falling back to it.
//
// # Non-vacuity, verified per CLAUDE.md rule 10
//
// Verified the same way T10.2's own authz_regression_test.go in this
// package documents: the principal check in Handler.CreateUser
// (internal/identity/adapter/grpcapi/handler.go) was temporarily removed
// and the handler made to fall back to req.GetActorUserId(), confirming
// TestCreateUser_AnonymousSquatterIsRejected and
// TestCreateUser_PrincipalOverridesWireActorClaim both FAIL (the squatter's
// chosen UUID was accepted and became the row's ID), then the check was
// restored and both PASS again. Captured output is in the PR description.
//
// This is a handler-level test — real grpcapi.Handler, real app.Service,
// real domain, with an in-memory fake at the persistence boundary only —
// for the same two reasons T10.2's authz_regression_test.go states: the
// behavior under test lives entirely at the handler/app seam, so a real
// Postgres round trip would add infrastructure rather than proof, and this
// environment has no reachable Docker daemon.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/identity/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/identity/app"
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	identityv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/identity/v1"
)

// victimSubject is the IdP subject of the real person the attacker is
// trying to lock out. Note the shape: an IdP subject is NOT a UUID — it is
// an arbitrary provider-specific string — which is exactly why T12.9 gives
// identity_users its own `subject` column instead of making the subject the
// row's uuid primary key. See db/migrations/0019_identity_subject.sql.
const victimSubject = "auth0|the-real-person"

// attackerChosenID is a well-formed UUID the attacker picks out of the air:
// under the pre-T12.9 contract this value became identity_users.id
// verbatim, which is the whole squatting primitive.
const attackerChosenID = "11111111-2222-3333-4444-555555555555"

// TestCreateUser_AnonymousSquatterIsRejected replays the original attack:
// an unauthenticated caller submitting a chosen UUID as actor_user_id.
//
// Pre-T12.9 this returned a created User whose id was attackerChosenID,
// permanently burning that identity. It must now fail with Unauthenticated
// — "I do not know who you are" — and persist nothing at all. The squatting
// surface is removed rather than narrowed: there is no anonymous path to
// User creation left to squat through.
func TestCreateUser_AnonymousSquatterIsRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	handler := grpcapi.NewHandler(app.NewService(repo, stubIDs{id: mintedID}))

	// No principal in this context: exactly the anonymous caller the
	// pre-fix contract accepted.
	resp, err := handler.CreateUser(context.Background(), &identityv1.CreateUserRequest{
		ActorUserId:               attackerChosenID,
		DisplayName:               "Squatter",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	})
	if err == nil {
		t.Fatalf("anonymous CreateUser succeeded and returned %+v: the identity-squatting DoS is NOT closed", resp)
	}
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Fatalf("status code = %v, want %v (no principal is 'I do not know who you are', never PermissionDenied)", got, codes.Unauthenticated)
	}

	// Nothing persisted: a rejected squat must leave no artifact behind,
	// which is the property that distinguishes this from the pre-fix
	// behavior's permanent row.
	if n := len(repo.users); n != 0 {
		t.Fatalf("repo holds %d users after a rejected anonymous CreateUser, want 0", n)
	}
}

// TestCreateUser_LegitimateSubjectOwnerCanRegister is the other half of the
// proof: closing the hole must not close registration itself. The real
// owner of a verified subject registers successfully, and the resulting
// User is keyed to a SERVER-MINTED id — never to anything the caller sent.
func TestCreateUser_LegitimateSubjectOwnerCanRegister(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	handler := grpcapi.NewHandler(app.NewService(repo, stubIDs{id: mintedID}))
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: victimSubject})

	resp, err := handler.CreateUser(ctx, &identityv1.CreateUserRequest{
		DisplayName:               "The Real Person",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	})
	if err != nil {
		t.Fatalf("the verified owner of %q could not register: %v", victimSubject, err)
	}
	if got := resp.GetUser().GetId(); got != mintedID {
		t.Fatalf("User.Id = %q, want the server-minted %q", got, mintedID)
	}
	if got := resp.GetUser().GetDisplayName(); got != "The Real Person" {
		t.Fatalf("User.DisplayName = %q, want %q", got, "The Real Person")
	}

	// The row is keyed to the verified subject, which is what makes a
	// second registration for that subject a self-collision rather than
	// an attacker's collision.
	stored, ok := repo.users[mintedID]
	if !ok {
		t.Fatalf("no User persisted under the minted id %q", mintedID)
	}
	if stored.Subject != victimSubject {
		t.Fatalf("stored Subject = %q, want %q", stored.Subject, victimSubject)
	}
}

// TestCreateUser_PrincipalOverridesWireActorClaim is the assertion the
// sprint calls the whole point: a request whose wire actor_user_id claims
// one identity while the verified principal says another must be resolved
// from the PRINCIPAL, never the claim.
//
// A handler that merely required a principal but still read actor_user_id
// when one was present would have changed nothing about the squatting
// primitive — any authenticated caller could then still burn any UUID. This
// test fails against that mistake specifically.
func TestCreateUser_PrincipalOverridesWireActorClaim(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	handler := grpcapi.NewHandler(app.NewService(repo, stubIDs{id: mintedID}))
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: "auth0|attacker"})

	resp, err := handler.CreateUser(ctx, &identityv1.CreateUserRequest{
		// The lie: an authenticated attacker still naming the UUID they
		// want to occupy.
		ActorUserId:               attackerChosenID,
		DisplayName:               "Authenticated Squatter",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	})
	if err != nil {
		t.Fatalf("CreateUser with a verified principal failed: %v", err)
	}
	if got := resp.GetUser().GetId(); got == attackerChosenID {
		t.Fatalf("User.Id = %q: the handler honored the wire actor_user_id claim, so the squatting primitive survives", got)
	}
	if got := resp.GetUser().GetId(); got != mintedID {
		t.Fatalf("User.Id = %q, want the server-minted %q", got, mintedID)
	}

	// The stored row is keyed to the attacker's OWN verified subject —
	// they can only ever occupy their own identity, which is precisely
	// the reduction from "targeted DoS against a stranger" to "you may
	// register yourself once".
	stored := repo.users[mintedID]
	if stored.Subject != "auth0|attacker" {
		t.Fatalf("stored Subject = %q, want the principal's own %q", stored.Subject, "auth0|attacker")
	}
	if _, squatted := repo.users[attackerChosenID]; squatted {
		t.Fatalf("a User was persisted under the attacker's chosen id %q", attackerChosenID)
	}
}

// TestCreateUser_SecondRegistrationForSameSubjectIsRejected pins T12.9's
// idempotent-or-rejected decision: REJECTED, via the pre-existing
// domain.ErrUserAlreadyExists sentinel, which grpcapi.toStatus already
// mapped to codes.AlreadyExists before this ticket (no new mapping was
// invented — see the ticket's error-handling instruction).
//
// Rejection rather than idempotent replay, because a second CreateUser is
// not a repeat of the first: it carries its own display name and level, so
// returning the ALREADY-STORED user would answer a request for "create me
// as X" with a user named Y and silently discard the difference. That is a
// wrong answer dressed as a successful one. AlreadyExists tells the truth,
// and the DoS this ticket closes does not come back with it: a subject
// collision is now only ever a self-collision, because the subject is
// verified rather than claimed.
func TestCreateUser_SecondRegistrationForSameSubjectIsRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	// Distinct minted IDs per call, so a rejection can only come from the
	// subject uniqueness rule and never from an id collision.
	handler := grpcapi.NewHandler(app.NewService(repo, &sequentialIDs{}))
	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: victimSubject})

	req := &identityv1.CreateUserRequest{
		DisplayName:               "The Real Person",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 3,
	}
	if _, err := handler.CreateUser(ctx, req); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	second := &identityv1.CreateUserRequest{
		DisplayName:               "Different Name Entirely",
		Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
		SelfReportedStartingLevel: 5,
	}
	if _, err := handler.CreateUser(ctx, second); err == nil {
		t.Fatal("second CreateUser for an already-registered subject succeeded, want AlreadyExists")
	} else if got := status.Code(err); got != codes.AlreadyExists {
		t.Fatalf("status code = %v, want %v", got, codes.AlreadyExists)
	}

	if n := len(repo.users); n != 1 {
		t.Fatalf("repo holds %d users, want exactly 1 (the rejected call must persist nothing)", n)
	}
}

// TestCreateUser_MalformedInputIsInvalidArgument keeps the third gRPC code
// in this RPC's contract distinct from the other two: an authenticated
// caller sending a bad payload is InvalidArgument, not Unauthenticated
// (which would misreport a known caller as unknown) and not AlreadyExists.
func TestCreateUser_MalformedInputIsInvalidArgument(t *testing.T) {
	t.Parallel()

	ctx := auth.ContextWithPrincipal(context.Background(), auth.Principal{Subject: victimSubject})

	tests := []struct {
		name string
		req  *identityv1.CreateUserRequest
	}{
		{
			name: "empty display name",
			req: &identityv1.CreateUserRequest{
				Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
				SelfReportedStartingLevel: 3,
			},
		},
		{
			name: "no roles",
			req: &identityv1.CreateUserRequest{
				DisplayName:               "Nobody",
				SelfReportedStartingLevel: 3,
			},
		},
		{
			name: "self-reported level out of range",
			req: &identityv1.CreateUserRequest{
				DisplayName:               "Nobody",
				Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLAYER},
				SelfReportedStartingLevel: 9,
			},
		},
		{
			// The role restriction T10.2 added is unchanged by T12.9 and
			// must stay enforced now that the path is authenticated:
			// being a verified caller is not a licence to self-assign an
			// elevated role.
			name: "non-self-assignable role",
			req: &identityv1.CreateUserRequest{
				DisplayName:               "Aspiring Admin",
				Roles:                     []identityv1.Role{identityv1.Role_ROLE_PLATFORM_ADMIN},
				SelfReportedStartingLevel: 3,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepo()
			handler := grpcapi.NewHandler(app.NewService(repo, stubIDs{id: mintedID}))

			if _, err := handler.CreateUser(ctx, tc.req); err == nil {
				t.Fatal("CreateUser succeeded, want InvalidArgument")
			} else if got := status.Code(err); got != codes.InvalidArgument {
				t.Fatalf("status code = %v, want %v", got, codes.InvalidArgument)
			}
			if n := len(repo.users); n != 0 {
				t.Fatalf("repo holds %d users after a rejected call, want 0", n)
			}
		})
	}
}

// TestAuthenticatedMethods declares the per-context policy A11 Ruling 2
// puts next to the handlers that break if it is wrong, and pins the two
// decisions in it:
//
//   - CreateUser and UpdateSelfReportedLevel REQUIRE a principal. Both mint
//     or mutate a fact about a specific person's identity.
//   - GetUser stays PUBLIC. It is a shipped unauthenticated read, and
//     silently authenticating it would break a live flow — the specific
//     mistake T12.7's instruction 2 warns about.
func TestAuthenticatedMethods(t *testing.T) {
	t.Parallel()

	got := grpcapi.AuthenticatedMethods()
	want := map[string]bool{
		"/pickleball.identity.v1.IdentityService/CreateUser":              true,
		"/pickleball.identity.v1.IdentityService/UpdateSelfReportedLevel": true,
	}

	if len(got) != len(want) {
		t.Fatalf("AuthenticatedMethods() = %v, want exactly %d entries", got, len(want))
	}
	for _, m := range got {
		if !want[m] {
			t.Errorf("AuthenticatedMethods() includes unexpected method %q", m)
		}
	}

	// Guard the public read explicitly rather than by omission, so a
	// future edit that adds it has to delete this assertion on purpose.
	for _, m := range got {
		if m == "/pickleball.identity.v1.IdentityService/GetUser" {
			t.Error("GetUser must stay public: authenticating a shipped anonymous read breaks a live flow")
		}
	}
}
