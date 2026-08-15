// Boundary validation for the caller-supplied Game ID on the roster read.
//
// internal/socialplay/adapter/postgres's mustUUID panics on a non-UUID, and
// grpc installs no recover() of its own, so an unvalidated Game ID off the wire
// could take the whole server process down. The guard rejects it at the app
// boundary; internal/platform/grpcrecovery is the backstop, not the fix.
package app_test

import (
	"context"
	"errors"
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
//
// T13.6 added a Host-only check to this read and deliberately left this
// invariant alone. The actor below is a stranger, which makes this test do
// double duty: it proves the malformed-ID guard still runs *before* the Host
// check, so a malformed ID cannot be told apart from an unknown one by
// authorization status either.
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

			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         newFakeGameRepository(),
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
				GameAdmins:    newFakeGameAdminRepository(),
			})

			got, err := svc.ListRegistrationsForGame(context.Background(), id, "auth0|a-stranger")
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
//
// T13.6 note: the Game is now seeded into the game repository and read as its
// Host, because the Host check needs a Game to compare against. That is not
// test scaffolding for its own sake — it is what makes this guard rail still
// mean "the ID guard is not too strict" rather than accidentally becoming a
// test of the new authorization check.
func TestListRegistrationsForGame_WellFormedIDStillReads(t *testing.T) {
	t.Parallel()

	games := newFakeGameRepository()
	registrations := newFakeRegistrationRepository()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: registrations,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	const host = "auth0|host-1"
	id := gameID(7)
	games.games[id] = domain.Game{ID: id, HostID: host}
	registrations.registrations["r-1"] = domain.Registration{
		ID: "r-1", GameID: id, PlayerID: "player-1",
		Status: domain.RegistrationStatusRegistered, PaymentStatus: domain.PaymentStatusUnpaid,
	}

	got, err := svc.ListRegistrationsForGame(context.Background(), id, host)
	if err != nil {
		t.Fatalf("ListRegistrationsForGame on a real Game: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d registrations for a real Game, want 1 — the guard is rejecting valid IDs", len(got))
	}
}

// --- T10.7 (closing issue #97): RegisterForGame, CancelRegistration, and
// JoinWaitlist's own malformed-id guards -------------------------------
//
// All three were found by this ticket's required inspection sweep (grepping
// cmd/server/main.go's route registrations for any other public write
// handler taking a caller-supplied id — issue #97's own instruction not to
// assume its named handler list was exhaustive): each calls
// GameRepository.GetByID or RegistrationRepository.GetByID first, the
// identical shape ListRegistrationsForGame's already-guarded read has, and
// each already returns the bare domain.ErrGameNotFound/
// ErrRegistrationNotFound for an unknown-but-well-formed id — but, unlike
// that read, none had a uuidShape guard at all.

func TestRegisterForGame_MalformedGameIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"", "not-a-uuid", "g-1", "0",
		"'; DROP TABLE games;--", "../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			games := newFakeGameRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         games,
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
				GameAdmins:    newFakeGameAdminRepository(),
			})

			_, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{
				GameID: id, PlayerID: "player-1",
			})
			if !errors.Is(err, domain.ErrGameNotFound) {
				t.Fatalf("RegisterForGame(GameID=%q) error = %v, want %v", id, err, domain.ErrGameNotFound)
			}
			if calls := games.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed GameID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

func TestJoinWaitlist_MalformedGameIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"", "not-a-uuid", "g-1", "0",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			games := newFakeGameRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         games,
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
				GameAdmins:    newFakeGameAdminRepository(),
			})

			_, err := svc.JoinWaitlist(context.Background(), app.JoinWaitlistInput{
				GameID: id, PlayerID: "player-1",
			})
			if !errors.Is(err, domain.ErrGameNotFound) {
				t.Fatalf("JoinWaitlist(GameID=%q) error = %v, want %v", id, err, domain.ErrGameNotFound)
			}
			if calls := games.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed GameID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

func TestCancelRegistration_MalformedRegistrationIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"", "not-a-uuid", "r-1", "0",
		"'; DROP TABLE registrations;--", "../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			registrations := newFakeRegistrationRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         newFakeGameRepository(),
				Registrations: registrations,
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
				GameAdmins:    newFakeGameAdminRepository(),
			})

			_, err := svc.CancelRegistration(context.Background(), id, "player-1")
			if !errors.Is(err, domain.ErrRegistrationNotFound) {
				t.Fatalf("CancelRegistration(%q) error = %v, want %v", id, err, domain.ErrRegistrationNotFound)
			}
			if calls := registrations.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed RegistrationID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestRegisterForGameAndJoinWaitlistAndCancelRegistration_WellFormedUnknownIDsStillReachTheRepository
// is the too-strict guard rail for all three new guards at once: a
// well-formed but unknown id must still reach the repository and get the
// repository's own not-found sentinel, or every real Game/Registration
// would silently fail.
func TestRegisterForGameAndJoinWaitlistAndCancelRegistration_WellFormedUnknownIDsStillReachTheRepository(t *testing.T) {
	t.Parallel()

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	t.Run("RegisterForGame", func(t *testing.T) {
		t.Parallel()
		games := newFakeGameRepository()
		svc := app.NewService(app.ServiceOptions{
			IDs:           &sequentialIDs{},
			Games:         games,
			Registrations: newFakeRegistrationRepository(),
			Waitlist:      newFakeWaitlistRepository(),
			Matches:       newFakeMatchRepository(),
			GameAdmins:    newFakeGameAdminRepository(),
		})

		_, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{GameID: unknown, PlayerID: "player-1"})
		if !errors.Is(err, domain.ErrGameNotFound) {
			t.Fatalf("RegisterForGame(%q) error = %v, want %v", unknown, err, domain.ErrGameNotFound)
		}
		if calls := games.getByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", calls)
		}
	})

	t.Run("JoinWaitlist", func(t *testing.T) {
		t.Parallel()
		games := newFakeGameRepository()
		svc := app.NewService(app.ServiceOptions{
			IDs:           &sequentialIDs{},
			Games:         games,
			Registrations: newFakeRegistrationRepository(),
			Waitlist:      newFakeWaitlistRepository(),
			Matches:       newFakeMatchRepository(),
			GameAdmins:    newFakeGameAdminRepository(),
		})

		_, err := svc.JoinWaitlist(context.Background(), app.JoinWaitlistInput{GameID: unknown, PlayerID: "player-1"})
		if !errors.Is(err, domain.ErrGameNotFound) {
			t.Fatalf("JoinWaitlist(%q) error = %v, want %v", unknown, err, domain.ErrGameNotFound)
		}
		if calls := games.getByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", calls)
		}
	})

	t.Run("CancelRegistration", func(t *testing.T) {
		t.Parallel()
		registrations := newFakeRegistrationRepository()
		svc := app.NewService(app.ServiceOptions{
			IDs:           &sequentialIDs{},
			Games:         newFakeGameRepository(),
			Registrations: registrations,
			Waitlist:      newFakeWaitlistRepository(),
			Matches:       newFakeMatchRepository(),
			GameAdmins:    newFakeGameAdminRepository(),
		})

		_, err := svc.CancelRegistration(context.Background(), unknown, "player-1")
		if !errors.Is(err, domain.ErrRegistrationNotFound) {
			t.Fatalf("CancelRegistration(%q) error = %v, want %v", unknown, err, domain.ErrRegistrationNotFound)
		}
		if calls := registrations.getByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown RegistrationID did not reach the repository (%d calls)", calls)
		}
	})
}

// --- T10.4 (this ticket): RecordMatchResult and ListMatchesForGame's own
// malformed-id guards --------------------------------------------------
//
// Same shape as the T10.7 suite above, applied to the two new RPCs this
// ticket adds: reuses the existing uuidShape helper (app/service.go), and
// proves — via the fake repositories' atomic call counters, not a
// return-value check alone (the fake-fidelity trap
// docs/process/t9-retro.md finding 3 names, and this ticket's own
// instructions require avoiding) — that a malformed GameID never reaches
// either GameRepository.GetByID or MatchRepository.Create/ListForGame.

func TestRecordMatchResult_MalformedGameIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"", "not-a-uuid", "g-1", "0",
		"'; DROP TABLE matches;--", "../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			games := newFakeGameRepository()
			matches := newFakeMatchRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         games,
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       matches,
				GameAdmins:    newFakeGameAdminRepository(),
			})

			_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
				GameID:      id,
				Players:     []string{"player-1", "player-2"},
				Score:       map[string]int{"player-1": 11, "player-2": 7},
				ActorUserID: "host-1",
			})
			if !errors.Is(err, domain.ErrGameNotFound) {
				t.Fatalf("RecordMatchResult(GameID=%q) error = %v, want %v", id, err, domain.ErrGameNotFound)
			}
			if calls := games.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed GameID %q reached the game repository (%d calls); it must be rejected at the boundary", id, calls)
			}
			if calls := matches.createCalls.Load(); calls != 0 {
				t.Errorf("malformed GameID %q reached the match repository (%d Create calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

func TestListMatchesForGame_MalformedGameIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	malformed := []string{
		"", "not-a-uuid", "g-1", "0",
		"'; DROP TABLE matches;--", "../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range malformed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			games := newFakeGameRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         games,
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
				GameAdmins:    newFakeGameAdminRepository(),
			})

			_, err := svc.ListMatchesForGame(context.Background(), id)
			if !errors.Is(err, domain.ErrGameNotFound) {
				t.Fatalf("ListMatchesForGame(%q) error = %v, want %v", id, err, domain.ErrGameNotFound)
			}
			if calls := games.getByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed GameID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestRecordMatchResultAndListMatchesForGame_WellFormedUnknownIDsStillReachTheRepository
// is the too-strict guard rail: a well-formed but unknown GameID must still
// reach the repository and get its own not-found sentinel, or every real
// Game's match-recording/listing would silently fail.
func TestRecordMatchResultAndListMatchesForGame_WellFormedUnknownIDsStillReachTheRepository(t *testing.T) {
	t.Parallel()

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	t.Run("RecordMatchResult", func(t *testing.T) {
		t.Parallel()
		games := newFakeGameRepository()
		svc := app.NewService(app.ServiceOptions{
			IDs:           &sequentialIDs{},
			Games:         games,
			Registrations: newFakeRegistrationRepository(),
			Waitlist:      newFakeWaitlistRepository(),
			Matches:       newFakeMatchRepository(),
			GameAdmins:    newFakeGameAdminRepository(),
		})

		_, err := svc.RecordMatchResult(context.Background(), app.RecordMatchResultInput{
			GameID:      unknown,
			Players:     []string{"player-1", "player-2"},
			Score:       map[string]int{"player-1": 11, "player-2": 7},
			ActorUserID: "host-1",
		})
		if !errors.Is(err, domain.ErrGameNotFound) {
			t.Fatalf("RecordMatchResult(%q) error = %v, want %v", unknown, err, domain.ErrGameNotFound)
		}
		if calls := games.getByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", calls)
		}
	})

	t.Run("ListMatchesForGame", func(t *testing.T) {
		t.Parallel()
		games := newFakeGameRepository()
		svc := app.NewService(app.ServiceOptions{
			IDs:           &sequentialIDs{},
			Games:         games,
			Registrations: newFakeRegistrationRepository(),
			Waitlist:      newFakeWaitlistRepository(),
			Matches:       newFakeMatchRepository(),
			GameAdmins:    newFakeGameAdminRepository(),
		})

		_, err := svc.ListMatchesForGame(context.Background(), unknown)
		if !errors.Is(err, domain.ErrGameNotFound) {
			t.Fatalf("ListMatchesForGame(%q) error = %v, want %v", unknown, err, domain.ErrGameNotFound)
		}
		if calls := games.getByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown GameID did not reach the repository (%d calls)", calls)
		}
	})
}
