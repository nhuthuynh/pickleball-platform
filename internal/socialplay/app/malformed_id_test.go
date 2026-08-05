// Boundary validation for the caller-supplied Game ID on the roster read.
//
// internal/socialplay/adapter/postgres's mustUUID panics on a non-UUID, and
// grpc installs no recover() of its own, so an unvalidated Game ID off the wire
// could take the whole server process down. The guard rejects it at the app
// boundary; internal/platform/grpcrecovery is the backstop, not the fix.
package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// gameID mints a deterministic, UUID-shaped Game ID — the shape
// internal/platform/idgen actually produces.
func gameID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-a000-%012d", n)
}

// TestListRegistrationsForGame_MalformedIDIsEmptyNotAPanic pins the fix. The
// read answers an unknown Game with an empty roster, so a malformed Game ID
// must answer identically.
func TestListRegistrationsForGame_MalformedIDIsEmptyNotAPanic(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"",
		"not-a-uuid",
		"g-1", // the old fixture shape
		"0",
		"'; DROP TABLE registrations;--",
		"../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		// Accepted by github.com/google/uuid's Validate, rejected by
		// pgtype.UUID.Scan — why the guard is a canonical-form check.
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), newFakeRegistrationRepository(), newFakeWaitlistRepository())

			got, err := svc.ListRegistrationsForGame(context.Background(), id)
			if err != nil {
				t.Fatalf("ListRegistrationsForGame(%q) error = %v, want nil", id, err)
			}
			if len(got) != 0 {
				t.Fatalf("ListRegistrationsForGame(%q) = %d registrations, want 0", id, len(got))
			}
		})
	}
}

// TestListRegistrationsForGame_WellFormedIDStillReads is the too-strict guard
// rail: rejecting real Game IDs would silently empty every Host's roster.
func TestListRegistrationsForGame_WellFormedIDStillReads(t *testing.T) {
	t.Parallel()

	registrations := newFakeRegistrationRepository()
	svc := app.NewService(&sequentialIDs{}, newFakeGameRepository(), registrations, newFakeWaitlistRepository())

	id := gameID(7)
	registrations.registrations["r-1"] = domain.Registration{
		ID: "r-1", GameID: id, PlayerID: "player-1",
		Status: domain.RegistrationStatusRegistered, PaymentStatus: domain.PaymentStatusUnpaid,
	}

	got, err := svc.ListRegistrationsForGame(context.Background(), id)
	if err != nil {
		t.Fatalf("ListRegistrationsForGame on a real Game: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d registrations for a real Game, want 1 — the guard is rejecting valid IDs", len(got))
	}
}
