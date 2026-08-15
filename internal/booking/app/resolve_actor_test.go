// App-layer coverage for ADR-0014's resolution seam (T13.2).
//
// This file exists in internal/booking/app specifically so the seam is covered
// by `make test-domain`, the gate that needs neither Docker nor `make
// generate`. The genuinely end-to-end proof — real Identity service, real
// adapter, real handler — lives in internal/booking/adapter/{identity,grpcapi};
// what is pinned HERE is the contract app.Service itself promises, since that
// is what every other actor-taking method in this package depends on.
package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// subjectOf is the verified IdP subject for fixture user n — deliberately NOT
// uuid-shaped, because a real `sub` claim never is. userID(n) is the uuid this
// platform mints for the same person. The two identifier spaces being
// different strings is the entire subject of ADR-0014, and it is why this
// helper exists rather than the tests reusing userID(n) for both.
func subjectOf(n int) string { return fmt.Sprintf("auth0|fixture-user-%d", n) }

// TestResolveActorUserID is the app-layer contract for the seam that closes
// issues #146 and #152.
func TestResolveActorUserID(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		wantID  string
		wantErr error
	}{
		{
			name:    "verified subject resolves to the caller's User.ID",
			subject: subjectOf(clubUser),
			wantID:  userID(clubUser),
		},
		{
			// Resolution answers "who are you", never "what may you do" — a
			// Player resolves exactly as readily as a Club. Without this case
			// a seam that only resolved privileged callers would pass every
			// other test here.
			name:    "resolution is independent of the User's roles",
			subject: subjectOf(playerUser),
			wantID:  userID(playerUser),
		},
		{
			name:    "unregistered subject",
			subject: "auth0|never-registered",
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:    "empty subject",
			subject: "",
			wantErr: domain.ErrUserNotFound,
		},
		{
			// A User.ID is not a subject. This is the case that stops a
			// future "accept either space" convenience patch, which would
			// reintroduce exactly the ambiguity #146 came through.
			name:    "a User.ID is not a subject",
			subject: userID(clubUser),
			wantErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecurringSvc()

			got, err := f.svc.ResolveActorUserID(context.Background(), tt.subject)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolveActorUserID(%q) error = %v, want %v", tt.subject, err, tt.wantErr)
				}
				if got != "" {
					t.Fatalf("ResolveActorUserID(%q) = %q on the error path, want the zero value", tt.subject, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveActorUserID(%q) = %v, want nil", tt.subject, err)
			}
			if got != tt.wantID {
				t.Fatalf("ResolveActorUserID(%q) = %q, want %q", tt.subject, got, tt.wantID)
			}
		})
	}
}

// TestResolveActorUserID_DelegatesToTheIdentityPort pins that the resolution
// is a real lookup rather than a local transformation of the string.
//
// This matters more than it looks: the cheapest wrong implementation of this
// seam is one that derives a uuid from the subject (hashing it, say) instead
// of reading the mapping Identity owns. That would satisfy every assertion in
// the table above except the unregistered cases, and would produce ids that
// reference no identity_users row — an FK violation at the first real write.
// Counting the port call is what distinguishes the two.
func TestResolveActorUserID_DelegatesToTheIdentityPort(t *testing.T) {
	f := newRecurringSvc()

	if _, err := f.svc.ResolveActorUserID(context.Background(), subjectOf(clubUser)); err != nil {
		t.Fatalf("ResolveActorUserID: %v", err)
	}

	if f.identity.resolves != 1 {
		t.Fatalf("port.IdentityLookup.UserIDBySubject called %d times, want exactly 1 — "+
			"the mapping from subject to User.ID belongs to the Identity context, and a "+
			"resolution that never asks it is deriving an id rather than reading one",
			f.identity.resolves)
	}
	if f.identity.checks != 0 {
		t.Fatalf("EnsureClubRole was called %d times during resolution, want 0 — "+
			"resolving who the caller is must not depend on, or leak into, what they may do",
			f.identity.checks)
	}
}

// TestResolveActorUserID_ProducesAnActorTheUseCasesAccept closes the loop
// inside this package: the value the seam returns is a value
// RequestRecurringHire actually accepts.
//
// Asserting the two halves separately would leave the gap #152 lived in — an
// actor value that every layer agrees is well-formed and that the next layer
// down still rejects. Here the output of one is fed directly into the other.
func TestResolveActorUserID_ProducesAnActorTheUseCasesAccept(t *testing.T) {
	f := newRecurringSvc()

	actorUserID, err := f.svc.ResolveActorUserID(context.Background(), subjectOf(clubUser))
	if err != nil {
		t.Fatalf("ResolveActorUserID: %v", err)
	}

	template, err := f.svc.RequestRecurringHire(context.Background(),
		requestInput(t, actorUserID, courtID(1), 4))
	if err != nil {
		t.Fatalf("RequestRecurringHire(resolved actor) = %v, want success — the seam's output "+
			"must be an actor the use cases accept, which is exactly what #152 reported it was not", err)
	}
	if template.RequestedByUserID != userID(clubUser) {
		t.Fatalf("RequestedByUserID = %q, want %q", template.RequestedByUserID, userID(clubUser))
	}
}
