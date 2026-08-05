// Boundary validation for caller-supplied Competition IDs.
//
// # The bug these tests pin
//
// internal/competitions/adapter/postgres.mustUUID panics on a string that is
// not a UUID, by design ("a malformed one here means an upstream invariant was
// already violated"). That reasoning is sound only if a malformed ID genuinely
// cannot reach it. It could: Service.GetCompetition passed req.GetCompetitionId()
// — an HTTP path parameter, unvalidated, on a public unauthenticated endpoint —
// straight through to Repository.GetByID. `GET /v1/competitions/not-a-uuid`
// panicked the adapter, and because grpc installs no recover() of its own, that
// panic killed the whole server process.
//
// internal/platform/grpcrecovery now stops the process dying. These tests cover
// the other half: the malformed ID is rejected at the boundary so the panic
// never happens, and the caller gets the same honest 404 an unknown ID gets.
//
// # Why the answer is NotFound, not InvalidArgument
//
// Identical reasoning to GetCompetitionByShareToken's (see sharelink_test.go and
// the shareTokenShape comment in service.go): an unknown ID and an unparseable
// one must be indistinguishable. These reads are public, so a distinguishable
// "that isn't even a valid ID" reply is an oracle that tells an enumerator which
// guesses had the right shape. Every miss returns the one
// domain.ErrCompetitionNotFound sentinel, which toStatus already maps to
// NotFound.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// malformedIDs is the shared corpus of things a client can put in a path
// parameter that are not IDs this system ever minted.
//
// The braced and urn: forms are the interesting entries and the reason this
// validation is a strict canonical-form check rather than a call to
// github.com/google/uuid's Validate. uuid.Validate accepts both of them;
// pgtype.UUID.Scan — which is what mustUUID actually calls — rejects both. Had
// the guard been written with uuid.Validate, those two inputs would have sailed
// through the "validated" boundary and panicked the adapter anyway. The check
// must be no wider than what the adapter accepts, and this corpus is what keeps
// it that way.
var malformedIDs = []struct {
	name string
	id   string
}{
	{"empty", ""},
	{"obviously not a uuid", "not-a-uuid"},
	{"the literal path segment", "anything-not-a-uuid"},
	{"numeric", "0"},
	{"sql injection attempt", "'; DROP TABLE competitions;--"},
	{"path traversal attempt", "../../etc/passwd"},
	{"nul byte", "6ba7b810-9dad-11d1-80b4-00c04fd430c\x00"},
	{"truncated uuid", "6ba7b810-9dad-11d1-80b4-00c04fd430c"},
	{"overlong uuid", "6ba7b810-9dad-11d1-80b4-00c04fd430c8f"},
	{"non-hex characters", "zzzzzzzz-9dad-11d1-80b4-00c04fd430c8"},
	{"wrong dash placement", "6ba7b8109-dad-11d1-80b4-00c04fd430c8"},
	{"braced form (uuid.Validate accepts, pgtype rejects)", "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}"},
	{"urn form (uuid.Validate accepts, pgtype rejects)", "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
	{"whitespace padded", " 6ba7b810-9dad-11d1-80b4-00c04fd430c8 "},
}

// mustUUIDRepo stands in for the real Postgres adapter: it panics on exactly
// the inputs pgtype.UUID.Scan rejects, which is exactly what mustUUID does.
//
// This is what makes these tests prove something. Asserting only "the service
// returned NotFound" would still pass if the ID were forwarded to a tolerant
// fake — the guarantee being pinned is that the malformed ID never reaches the
// adapter at all, and a panicking fake is the only way to state that.
type mustUUIDRepo struct {
	port.Repository
	calls int
}

func (r *mustUUIDRepo) guard(id string) {
	r.calls++
	var scannable bool
	if len(id) == 36 {
		scannable = canonicalUUIDForTest(id)
	}
	if !scannable {
		panic("competitions postgres adapter: invalid uuid " + id)
	}
}

func (r *mustUUIDRepo) GetByID(_ context.Context, id string) (domain.Competition, error) {
	r.guard(id)
	return domain.Competition{}, domain.ErrCompetitionNotFound
}

func (r *mustUUIDRepo) ListEntriesForCompetition(_ context.Context, id string) ([]domain.CompetitionEntry, error) {
	r.guard(id)
	return nil, nil
}

func canonicalUUIDForTest(s string) bool {
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}

func newGuardedService(repo port.Repository) *app.Service {
	return app.NewService(app.ServiceOptions{
		Competitions: repo,
		IDs:          &sequentialIDs{},
		Reservation:  newFakeReservation(),
		Facilities:   newFakeFacilityLookup(),
		ShareTokens:  &fakeShareTokens{},
	})
}

// TestGetCompetition_MalformedIDIsNotFoundAndNeverReachesTheAdapter is the
// regression test for the reported crash. Before the fix, every one of these
// inputs panicked the Postgres adapter and, with no recovery interceptor
// installed, killed the server process.
func TestGetCompetition_MalformedIDIsNotFoundAndNeverReachesTheAdapter(t *testing.T) {
	t.Parallel()

	for _, tc := range malformedIDs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &mustUUIDRepo{}
			svc := newGuardedService(repo)

			_, err := svc.GetCompetition(context.Background(), tc.id)

			if !errors.Is(err, domain.ErrCompetitionNotFound) {
				t.Fatalf("GetCompetition(%q) error = %v, want %v", tc.id, err, domain.ErrCompetitionNotFound)
			}
			if repo.calls != 0 {
				t.Errorf("malformed ID %q reached the repository (%d calls); it must be rejected at the boundary", tc.id, repo.calls)
			}
		})
	}
}

// TestListEntriesForCompetition_MalformedIDIsEmptyAndNeverReachesTheAdapter
// covers the roster read, which is list-shaped rather than get-shaped.
//
// A list read answers an unknown Competition with an empty roster rather than
// an error, so a malformed one must answer identically — same
// indistinguishability argument as above, applied to this method's own contract
// instead of imported from GetCompetition's.
func TestListEntriesForCompetition_MalformedIDIsEmptyAndNeverReachesTheAdapter(t *testing.T) {
	t.Parallel()

	for _, tc := range malformedIDs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &mustUUIDRepo{}
			svc := newGuardedService(repo)

			got, err := svc.ListEntriesForCompetition(context.Background(), tc.id)
			if err != nil {
				t.Fatalf("ListEntriesForCompetition(%q) error = %v, want nil", tc.id, err)
			}
			if len(got) != 0 {
				t.Errorf("ListEntriesForCompetition(%q) = %d entries, want 0", tc.id, len(got))
			}
			if repo.calls != 0 {
				t.Errorf("malformed ID %q reached the repository (%d calls)", tc.id, repo.calls)
			}
		})
	}
}

// TestWellFormedIDsStillReachTheAdapter is the other half of the contract, and
// the test that stops the guard from being "reject everything". A validator
// that is too strict silently 404s real Competitions, which is a worse and much
// quieter bug than the crash it replaced.
func TestWellFormedIDsStillReachTheAdapter(t *testing.T) {
	t.Parallel()

	// Canonical lowercase is what internal/platform/idgen actually mints;
	// uppercase and mixed case are accepted because pgtype.UUID.Scan accepts
	// them and a client may echo an ID back in any case.
	wellFormed := []string{
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"00000000-0000-0000-0000-000000000000",
		"6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
		"6ba7b810-9DAD-11d1-80B4-00c04fd430c8",
	}

	for _, id := range wellFormed {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := &mustUUIDRepo{}
			svc := newGuardedService(repo)

			if _, err := svc.GetCompetition(context.Background(), id); !errors.Is(err, domain.ErrCompetitionNotFound) {
				t.Fatalf("GetCompetition(%q) error = %v, want the repository's own %v", id, err, domain.ErrCompetitionNotFound)
			}
			if repo.calls != 1 {
				t.Fatalf("well-formed ID %q did not reach the repository (%d calls) — the guard is too strict and would 404 real Competitions", id, repo.calls)
			}
		})
	}
}
