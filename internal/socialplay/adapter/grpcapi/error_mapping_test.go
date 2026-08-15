// T14.7 (closes issue #158, and with it the cross-context half of #131) —
// the SocialPlayService's sentinel -> gRPC code mapping, asserted service-wide
// rather than per-RPC.
//
// The defect this file exists to prevent, and which it caught: toStatus sent
// ErrIllegalStatusTransition to InvalidArgument while sending its own sibling
// state-condition sentinel ErrGameCancelled to FailedPrecondition — the same
// one-concept-two-codes shape as #131, in a context that had already conceded
// the point one case earlier in the same switch.
//
// The code it now answers with is FailedPrecondition, per gRPC's own
// definitions: INVALID_ARGUMENT is a problem with the argument *regardless of
// the state of the system*, while FAILED_PRECONDITION is "the system is not in
// a state required for the operation's execution." Every raise site of
// ErrIllegalStatusTransition in this context — Game.Cancel,
// Registration.Cancel, WaitlistEntry.Promote/Expire/Cancel — is a
// `if Status != required { return ErrIllegalStatusTransition }` guard, so it is
// state-dependent by construction. The request is well-formed; nothing about
// the argument needs fixing.
//
// Structure mirrors internal/payments/adapter/grpcapi/error_mapping_test.go
// (T13.9), which this ticket's instruction 3 names as the template: a
// per-sentinel table, a cross-RPC equality assertion, and two source-level
// guards (a second error mapper; a domain sentinel with no mapping row).
//
// Handler-level with the in-memory port fakes this package already defines, for
// the reason authz_regression_test.go documents at length: the mapping under
// test is unaffected by which port implementation is behind it, and this
// environment has no Docker daemon, so the test has to be runnable where it was
// written (CLAUDE.md rule 10).
package grpcapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// --- fixtures ---------------------------------------------------------

const (
	mapHostID    = "map-host"
	mapPlayerID  = "map-player"
	mapStranger  = "map-stranger"
	mapUnknownID = "00000000-0000-4000-d000-000000009999"
)

// cancelGameTwice seeds a Game, cancels it as its Host, then cancels it again
// and returns the second call's error.
//
// Note which sentinel this actually produces: app.Service.CancelGame calls
// game.EnsureNotCancelled() *before* game.Cancel(), so the second call is
// rejected with ErrGameCancelled and Game.Cancel's own
// ErrIllegalStatusTransition guard is shadowed — unreachable from this RPC.
// Established by reading the app method, not assumed from the sentinel's name.
// CancelRegistration (below) is the path that genuinely reaches
// ErrIllegalStatusTransition.
func cancelGameTwice(t *testing.T) error {
	t.Helper()

	h, gameRepo, _ := newTestHandler()
	game := seedGameWithHost(t, gameRepo, "map-cancel-game", mapHostID)
	req := &socialplayv1.CancelGameRequest{GameId: game.ID}
	if _, err := h.CancelGame(ctxAs(mapHostID), req); err != nil {
		t.Fatalf("seed CancelGame: %v", err)
	}
	_, err := h.CancelGame(ctxAs(mapHostID), req)
	return err
}

// cancelRegistrationTwice registers a player, cancels the registration, then
// cancels it again — the same state-machine violation from a different
// aggregate and a different RPC.
func cancelRegistrationTwice(t *testing.T) error {
	t.Helper()

	h, gameRepo, _ := newTestHandler()
	game := seedGameWithHost(t, gameRepo, "map-cancel-reg", mapHostID)

	resp, err := h.RegisterForGame(ctxAs(mapPlayerID), &socialplayv1.RegisterForGameRequest{GameId: game.ID})
	if err != nil {
		t.Fatalf("seed RegisterForGame: %v", err)
	}
	req := &socialplayv1.CancelRegistrationRequest{RegistrationId: resp.GetRegistration().GetId()}
	if _, err := h.CancelRegistration(ctxAs(mapPlayerID), req); err != nil {
		t.Fatalf("seed CancelRegistration: %v", err)
	}
	_, err = h.CancelRegistration(ctxAs(mapPlayerID), req)
	return err
}

// --- the table --------------------------------------------------------

type errorMappingCase struct {
	name string
	// sentinel is the domain.Err* identifier this row pins, matched against
	// internal/socialplay/domain/errors.go by
	// TestSentinelToCodeTableCoversEveryDomainSentinel.
	sentinel string
	// err is the sentinel value itself, pinned directly through the mapper so
	// every sentinel is covered whether or not an RPC can currently reach it.
	err      error
	wantCode codes.Code
	// invoke, when set, drives a real RPC that surfaces this sentinel and must
	// agree with wantCode. Left nil for sentinels no client-facing RPC can
	// raise, and for those the handler rejects at the wire boundary before
	// toStatus is reached.
	invoke func(t *testing.T) error
	why    string
}

func errorMappingCases() []errorMappingCase {
	return []errorMappingCase{
		// --- the sentinel this ticket is about ---
		{
			name:     "cancelling an already-cancelled registration",
			sentinel: "ErrIllegalStatusTransition",
			err:      domain.ErrIllegalStatusTransition,
			wantCode: codes.FailedPrecondition,
			why: "gRPC defines InvalidArgument as a problem with the argument *regardless of the state of the system*; " +
				"every raise site of this sentinel (Game.Cancel, Registration.Cancel, WaitlistEntry.Promote/Expire/Cancel) " +
				"is a `Status != required` guard, so it is state-dependent by construction — exactly FailedPrecondition. " +
				"#158: it must also agree with ErrGameCancelled, which this same switch already sends to FailedPrecondition",
			// CancelRegistration, not CancelGame: see cancelGameTwice's doc
			// comment for why the Game path never reaches this sentinel.
			invoke: cancelRegistrationTwice,
		},
		{
			name:     "recording a match against a cancelled game",
			sentinel: "ErrGameCancelled",
			err:      domain.ErrGameCancelled,
			wantCode: codes.FailedPrecondition,
			why:      "the sibling that already conceded the point — the Game is found and the request well-formed; its state is what forbids the operation",
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-cancelled-match", mapHostID)
				if _, err := h.CancelGame(ctxAs(mapHostID), &socialplayv1.CancelGameRequest{GameId: game.ID}); err != nil {
					t.Fatalf("seed CancelGame: %v", err)
				}
				_, err := h.RecordMatchResult(ctxAs(mapHostID), &socialplayv1.RecordMatchResultRequest{
					GameId:  game.ID,
					Players: []string{mapPlayerID, mapStranger},
					Score:   map[string]int32{mapPlayerID: 11, mapStranger: 7},
				})
				return err
			},
		},

		// --- conflict ---
		{
			name:     "registering for a game at capacity",
			sentinel: "ErrGameFull",
			err:      domain.ErrGameFull,
			wantCode: codes.AlreadyExists,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-full", 1)
				if _, err := h.RegisterForGame(ctxAs(mapPlayerID), &socialplayv1.RegisterForGameRequest{GameId: game.ID}); err != nil {
					t.Fatalf("seed RegisterForGame: %v", err)
				}
				_, err := h.RegisterForGame(ctxAs(mapStranger), &socialplayv1.RegisterForGameRequest{GameId: game.ID})
				return err
			},
		},
		{
			name:     "registering twice for the same game",
			sentinel: "ErrAlreadyRegistered",
			err:      domain.ErrAlreadyRegistered,
			wantCode: codes.AlreadyExists,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-dup-register", 4)
				req := &socialplayv1.RegisterForGameRequest{GameId: game.ID}
				if _, err := h.RegisterForGame(ctxAs(mapPlayerID), req); err != nil {
					t.Fatalf("seed RegisterForGame: %v", err)
				}
				_, err := h.RegisterForGame(ctxAs(mapPlayerID), req)
				return err
			},
		},
		{
			name:     "joining a waitlist twice",
			sentinel: "ErrAlreadyOnWaitlist",
			err:      domain.ErrAlreadyOnWaitlist,
			wantCode: codes.AlreadyExists,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-dup-waitlist", 1)
				if _, err := h.RegisterForGame(ctxAs(mapHostID), &socialplayv1.RegisterForGameRequest{GameId: game.ID}); err != nil {
					t.Fatalf("seed RegisterForGame: %v", err)
				}
				req := &socialplayv1.JoinWaitlistRequest{GameId: game.ID}
				if _, err := h.JoinWaitlist(ctxAs(mapPlayerID), req); err != nil {
					t.Fatalf("seed JoinWaitlist: %v", err)
				}
				_, err := h.JoinWaitlist(ctxAs(mapPlayerID), req)
				return err
			},
		},
		{
			name:     "court already booked for the requested range",
			sentinel: "ErrCourtUnavailable",
			err:      domain.ErrCourtUnavailable,
			wantCode: codes.AlreadyExists,
			why:      "Social Play's context-local translation of booking's ErrCourtDoubleBooked, and it takes that sentinel's conflict code",
		},

		// --- not found ---
		{
			name:     "unknown game id",
			sentinel: "ErrGameNotFound",
			err:      domain.ErrGameNotFound,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h, _, _ := newTestHandler()
				_, err := h.CancelGame(ctxAs(mapHostID), &socialplayv1.CancelGameRequest{GameId: mapUnknownID})
				return err
			},
		},
		{
			name:     "unknown registration id",
			sentinel: "ErrRegistrationNotFound",
			err:      domain.ErrRegistrationNotFound,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h, _, _ := newTestHandler()
				_, err := h.CancelRegistration(ctxAs(mapPlayerID), &socialplayv1.CancelRegistrationRequest{
					RegistrationId: mapUnknownID,
				})
				return err
			},
		},
		{
			name:     "unknown waitlist entry id",
			sentinel: "ErrWaitlistEntryNotFound",
			err:      domain.ErrWaitlistEntryNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "unknown venue facility id",
			sentinel: "ErrFacilityNotFound",
			err:      domain.ErrFacilityNotFound,
			wantCode: codes.NotFound,
			why:      "an unknown CreateGameRequest.venue_facility_id is a 404, not a 500 or a silent accept (T8.3)",
		},

		// --- permission denied ---
		{
			name:     "a different player cancels the registration",
			sentinel: "ErrNotRegistrationOwner",
			err:      domain.ErrNotRegistrationOwner,
			wantCode: codes.PermissionDenied,
			why:      "a verified principal failing an object-level ownership check is 403-shaped, never 500-shaped (T5.2/T5.5's BOLA rule)",
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-bola-reg", 4)
				resp, err := h.RegisterForGame(ctxAs(mapPlayerID), &socialplayv1.RegisterForGameRequest{GameId: game.ID})
				if err != nil {
					t.Fatalf("seed RegisterForGame: %v", err)
				}
				_, err = h.CancelRegistration(ctxAs(mapStranger), &socialplayv1.CancelRegistrationRequest{
					RegistrationId: resp.GetRegistration().GetId(),
				})
				return err
			},
		},
		{
			name:     "a different player cancels the waitlist entry",
			sentinel: "ErrNotWaitlistEntryOwner",
			err:      domain.ErrNotWaitlistEntryOwner,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "a non-host cancels the game",
			sentinel: "ErrNotGameHost",
			err:      domain.ErrNotGameHost,
			wantCode: codes.PermissionDenied,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-bola-game", mapHostID)
				_, err := h.CancelGame(ctxAs(mapStranger), &socialplayv1.CancelGameRequest{GameId: game.ID})
				return err
			},
		},
		{
			name:     "a stranger records a match result",
			sentinel: "ErrNotGameHostOrAdmin",
			err:      domain.ErrNotGameHostOrAdmin,
			wantCode: codes.PermissionDenied,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-match-authz", mapHostID)
				_, err := h.RecordMatchResult(ctxAs(mapStranger), &socialplayv1.RecordMatchResultRequest{
					GameId:  game.ID,
					Players: []string{mapPlayerID, mapStranger},
					Score:   map[string]int32{mapPlayerID: 11, mapStranger: 7},
				})
				return err
			},
		},

		// --- invalid argument: genuinely state-independent request defects ---
		{
			name:     "inverted or zero-length time range",
			sentinel: "ErrInvalidTimeRange",
			err:      domain.ErrInvalidTimeRange,
			wantCode: codes.InvalidArgument,
			why:      "malformed request regardless of system state — the textbook InvalidArgument",
		},
		{
			name:     "non-positive capacity",
			sentinel: "ErrInvalidCapacity",
			err:      domain.ErrInvalidCapacity,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "no court ids",
			sentinel: "ErrEmptyCourtIDs",
			err:      domain.ErrEmptyCourtIDs,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "malformed court id",
			sentinel: "ErrMalformedCourtID",
			err:      domain.ErrMalformedCourtID,
			wantCode: codes.InvalidArgument,
			why:      "T14.8 (#156): a non-uuid court id cannot name any court that exists, so it is a client-input defect regardless of system state — not an existence oracle, since the value could never resolve either way",
		},
		{
			name:     "empty player id",
			sentinel: "ErrEmptyPlayerID",
			err:      domain.ErrEmptyPlayerID,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid payment method",
			sentinel: "ErrInvalidPaymentMethod",
			err:      domain.ErrInvalidPaymentMethod,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "negative guest allowance",
			sentinel: "ErrInvalidGuestAllowance",
			err:      domain.ErrInvalidGuestAllowance,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "guest count exceeds the allowance",
			sentinel: "ErrGuestAllowanceExceeded",
			err:      domain.ErrGuestAllowanceExceeded,
			wantCode: codes.InvalidArgument,
			why: "a guest count outside a fixed, knowable, caller-visible limit is a request defect — the same " +
				"reasoning competitions states for its own copy of this sentinel",
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-guests", 4)
				_, err := h.RegisterForGame(ctxAs(mapPlayerID), &socialplayv1.RegisterForGameRequest{
					GameId:     game.ID,
					GuestCount: 5,
				})
				return err
			},
		},
		{
			name:     "negative or malformed money",
			sentinel: "ErrInvalidMoney",
			err:      domain.ErrInvalidMoney,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "no players on a match",
			sentinel: "ErrEmptyPlayers",
			err:      domain.ErrEmptyPlayers,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-no-players", mapHostID)
				_, err := h.RecordMatchResult(ctxAs(mapHostID), &socialplayv1.RecordMatchResultRequest{
					GameId:  game.ID,
					Players: nil,
					Score:   map[string]int32{mapPlayerID: 11},
				})
				return err
			},
		},
		{
			name:     "a match with one player",
			sentinel: "ErrTooFewPlayers",
			err:      domain.ErrTooFewPlayers,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-one-player", mapHostID)
				_, err := h.RecordMatchResult(ctxAs(mapHostID), &socialplayv1.RecordMatchResultRequest{
					GameId:  game.ID,
					Players: []string{mapPlayerID},
					Score:   map[string]int32{mapPlayerID: 11},
				})
				return err
			},
		},
		{
			name:     "a match with no score",
			sentinel: "ErrEmptyScore",
			err:      domain.ErrEmptyScore,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGameWithHost(t, gameRepo, "map-no-score", mapHostID)
				_, err := h.RecordMatchResult(ctxAs(mapHostID), &socialplayv1.RecordMatchResultRequest{
					GameId:  game.ID,
					Players: []string{mapPlayerID, mapStranger},
					Score:   nil,
				})
				return err
			},
		},
		{
			name:     "joining the waitlist for a game that is not full",
			sentinel: "ErrGameNotFull",
			err:      domain.ErrGameNotFull,
			wantCode: codes.InvalidArgument,
			why: "LEFT AS-IS BY T14.7, AND DISCLOSED RATHER THAN CHANGED IN PASSING. This sentinel has a real " +
				"claim on FailedPrecondition by the same argument the ticket applies to ErrIllegalStatusTransition " +
				"— \"the game is not full\" is a fact about system state, not about the argument. But #158 scopes " +
				"itself to the status-transition sentinels, and T13.9's own discipline (\"disclose it and open an " +
				"issue; do not silently widen this ticket into a second context\") is what kept #131 from being " +
				"fixed piecemeal. Pinned at today's code so a deliberate future change is a visible test change",
			invoke: func(t *testing.T) error {
				h, gameRepo, _ := newTestHandler()
				game := seedGame(t, gameRepo, "map-not-full", 4)
				_, err := h.JoinWaitlist(ctxAs(mapPlayerID), &socialplayv1.JoinWaitlistRequest{GameId: game.ID})
				return err
			},
		},

		// --- deliberately unreachable from any client-facing RPC ---
		//
		// toStatus does not match these four, so they fall through to Internal.
		// That is the existing, documented decision (see toStatus's own comment
		// on ErrWaitlistPromotionNotExpired/ErrNoWaitingEntries): reaching this
		// handler with one of them would mean something upstream called an
		// app-layer method no RPC exposes, which is a server fault and Internal
		// is the honest code for it. Verified still true at T14.7 — none of the
		// nine RPCs on this service can produce any of them:
		//
		//   - ErrEmptyGameID: RecordMatchResult and ListMatchesForGame both
		//     apply app.Service's uuidShape guard first, so an empty game id is
		//     already ErrGameNotFound before domain.RecordMatch is reached.
		//   - ErrInvalidPaymentStatus: raised only by
		//     Registration.MarkPaymentStatus, driven by the Payments context
		//     through port.RegistrationPaymentUpdater, never by an RPC here.
		//   - ErrNoWaitingEntries / ErrWaitlistPromotionNotExpired: promotion
		//     and expiry are app-layer-internal; ErrNoWaitingEntries is
		//     swallowed as an expected outcome by app.Service itself.
		//
		// They are pinned rather than skipped so that the day an RPC does reach
		// one, this table is what fails.
		{
			name:     "empty game id on a match (unreachable)",
			sentinel: "ErrEmptyGameID",
			err:      domain.ErrEmptyGameID,
			wantCode: codes.Internal,
			why:      "unreachable: app.Service's uuidShape guard answers ErrGameNotFound first — see the block comment above",
		},
		{
			name:     "invalid payment status (unreachable)",
			sentinel: "ErrInvalidPaymentStatus",
			err:      domain.ErrInvalidPaymentStatus,
			wantCode: codes.Internal,
			why:      "unreachable: raised only through port.RegistrationPaymentUpdater, never by an RPC on this service",
		},
		{
			name:     "no waiting waitlist entries (unreachable)",
			sentinel: "ErrNoWaitingEntries",
			err:      domain.ErrNoWaitingEntries,
			wantCode: codes.Internal,
			why:      "unreachable: an expected outcome app.Service swallows; deliberately unmapped since T6.6",
		},
		{
			name:     "promotion window not yet expired (unreachable)",
			sentinel: "ErrWaitlistPromotionNotExpired",
			err:      domain.ErrWaitlistPromotionNotExpired,
			wantCode: codes.Internal,
			why:      "unreachable: ExpireWaitlistPromotion is app-layer-internal; deliberately unmapped since T6.6",
		},
	}
}

// --- assertions -------------------------------------------------------

// TestSocialPlayService_SentinelToCode pins every domain sentinel to exactly
// one status code across the whole service, and — where an RPC can surface it —
// proves the running handler agrees.
func TestSocialPlayService_SentinelToCode(t *testing.T) {
	t.Parallel()

	for _, tc := range errorMappingCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped := grpcapi.ToStatusForTest(tc.err)
			st, ok := status.FromError(mapped)
			if !ok {
				t.Fatalf("%s: mapper returned a non-gRPC-status error: %v", tc.sentinel, mapped)
			}
			if st.Code() != tc.wantCode {
				t.Fatalf("%s mapped to %v, want %v%s", tc.sentinel, st.Code(), tc.wantCode, reason(tc.why))
			}

			if tc.invoke == nil {
				return
			}

			err := tc.invoke(t)
			if err == nil {
				t.Fatalf("%s: RPC succeeded, want %v (%s)", tc.sentinel, tc.wantCode, tc.why)
			}
			rpcSt, ok := status.FromError(err)
			if !ok {
				t.Fatalf("%s: RPC error is not a gRPC status: %v", tc.sentinel, err)
			}
			if rpcSt.Code() != tc.wantCode {
				t.Fatalf("%s: RPC answered %v, want %v%s (err: %v)",
					tc.sentinel, rpcSt.Code(), tc.wantCode, reason(tc.why), err)
			}
		})
	}
}

func reason(why string) string {
	if why == "" {
		return ""
	}
	return " — " + why
}

// TestSocialPlayService_SameSentinelSameCodeAcrossRPCs is #131's claim stated
// directly: the codes two *different* RPCs answer with for one sentinel are
// asserted equal to each other rather than each matching a constant, so a
// future re-divergence fails here even if someone updates the table above to
// describe it.
func TestSocialPlayService_SameSentinelSameCodeAcrossRPCs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel string
		rpcA     string
		viaA     func(t *testing.T) error
		rpcB     string
		viaB     func(t *testing.T) error
	}{
		{
			name:     "unknown game",
			sentinel: "ErrGameNotFound",
			rpcA:     "CancelGame",
			viaA: func(t *testing.T) error {
				h, _, _ := newTestHandler()
				_, err := h.CancelGame(ctxAs(mapHostID), &socialplayv1.CancelGameRequest{GameId: mapUnknownID})
				return err
			},
			rpcB: "JoinWaitlist",
			viaB: func(t *testing.T) error {
				h, _, _ := newTestHandler()
				_, err := h.JoinWaitlist(ctxAs(mapPlayerID), &socialplayv1.JoinWaitlistRequest{GameId: mapUnknownID})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := codeOf(t, tc.rpcA, tc.viaA(t))
			b := codeOf(t, tc.rpcB, tc.viaB(t))

			if a != b {
				t.Fatalf("%s produced %v from %s but %v from %s — one domain sentinel must mean one gRPC code "+
					"regardless of which RPC surfaced it (issues #131, #158)", tc.sentinel, a, tc.rpcA, b, tc.rpcB)
			}
		})
	}
}

// TestSocialPlayService_DoubleCancelAnswersOneCodeAcrossAggregates is the
// sharper half of #158 for this context, and the assertion that actually failed
// before the fix.
//
// Cancelling something twice is one situation to a client. Social Play answers
// it from two RPCs, via two *different* sentinels, because the two app methods
// happen to guard in different orders: CancelGame checks EnsureNotCancelled
// first and so raises ErrGameCancelled, while CancelRegistration falls through
// to Registration.Cancel's own guard and raises ErrIllegalStatusTransition.
// Both mean "this aggregate is not in a state that permits the operation", so
// both must answer one code — and before T14.7 they did not: FailedPrecondition
// from one, InvalidArgument from the other.
//
// This is deliberately not folded into SameSentinelSameCodeAcrossRPCs above:
// that test is about one sentinel seen from two RPCs, this one is about one
// *concept* spread across two sentinels. Conflating them would have hidden
// which property was being asserted — and the sentinel-level test alone would
// not have caught this, since each sentinel is internally consistent.
func TestSocialPlayService_DoubleCancelAnswersOneCodeAcrossAggregates(t *testing.T) {
	t.Parallel()

	game := codeOf(t, "CancelGame (twice)", cancelGameTwice(t))
	registration := codeOf(t, "CancelRegistration (twice)", cancelRegistrationTwice(t))

	if game != registration {
		t.Fatalf("cancelling an already-cancelled Game answered %v but an already-cancelled Registration answered %v — "+
			"one concept (\"this aggregate is already in a terminal state\") must mean one gRPC code whichever "+
			"aggregate and RPC a client reaches it through (issues #131, #158)", game, registration)
	}
	if game != codes.FailedPrecondition {
		t.Fatalf("both double-cancel paths answered %v, want FailedPrecondition — consistent but wrong is still wrong: "+
			"a state-machine violation is state-dependent by definition, which is what FailedPrecondition means and "+
			"what InvalidArgument (\"regardless of the state of the system\") denies", game)
	}
}

func codeOf(t *testing.T, what string, err error) codes.Code {
	t.Helper()

	if err == nil {
		t.Fatalf("%s succeeded, want an error to map", what)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", what, err)
	}
	return st.Code()
}

// TestSocialPlayService_HasExactlyOneErrorMapper is the structural guard
// against #131's actual shape. That inconsistency was not a typo in a mapping —
// it was a *second mapper* added beside the shared one, which no assertion about
// codes would have flagged as long as both mappers agreed with their own tests.
func TestSocialPlayService_HasExactlyOneErrorMapper(t *testing.T) {
	t.Parallel()

	mappers := errorMapperNames(t, "handler.go")
	if len(mappers) != 1 || mappers[0] != "toStatus" {
		t.Fatalf("handler.go declares error mappers %v, want exactly [toStatus] — "+
			"a per-RPC error mapper is how issue #131's one-sentinel-two-codes split arose; "+
			"map the sentinel once in toStatus, or introduce a distinct sentinel if two RPCs "+
			"genuinely mean different things", mappers)
	}
}

func errorMapperNames(t *testing.T, filename string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}

	var mappers []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if isErrorToErrorFunc(fn.Type) {
			mappers = append(mappers, fn.Name.Name)
		}
	}
	return mappers
}

func isErrorToErrorFunc(sig *ast.FuncType) bool {
	if sig.Params == nil || len(sig.Params.List) != 1 || len(sig.Params.List[0].Names) != 1 {
		return false
	}
	if sig.Results == nil || len(sig.Results.List) != 1 {
		return false
	}
	return isIdent(sig.Params.List[0].Type, "error") && isIdent(sig.Results.List[0].Type, "error")
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// TestSentinelToCodeTableCoversEveryDomainSentinel keeps the table above honest
// as the domain grows: a sentinel added to internal/socialplay/domain/errors.go
// with no mapping row falls through toStatus's default to Internal, which is a
// 500 for a condition the domain anticipated. That is a silent bug today; here
// it is a failing test.
func TestSentinelToCodeTableCoversEveryDomainSentinel(t *testing.T) {
	t.Parallel()

	covered := map[string]bool{}
	for _, tc := range errorMappingCases() {
		covered[tc.sentinel] = true
	}

	missing := uncoveredSentinels(t, "../../domain/errors.go", covered)
	if len(missing) > 0 {
		t.Fatalf("domain sentinels with no row in errorMappingCases: %v — "+
			"an unmapped sentinel falls through toStatus's default to Internal (HTTP 500) "+
			"for a condition the domain explicitly anticipated", missing)
	}
}

func uncoveredSentinels(t *testing.T, path string, covered map[string]bool) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var missing []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range spec.Names {
			if len(name.Name) > 3 && name.Name[:3] == "Err" && !covered[name.Name] {
				missing = append(missing, name.Name)
			}
		}
		return true
	})
	return missing
}
