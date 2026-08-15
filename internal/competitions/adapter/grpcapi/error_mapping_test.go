// T14.7 (closes issue #158, and with it the cross-context half of #131) —
// the CompetitionsService's sentinel -> gRPC code mapping, asserted
// service-wide rather than per-RPC.
//
// The defect this file exists to prevent, and which it caught: toStatus sent
// ErrIllegalStatusTransition to InvalidArgument while sending its own sibling
// state-condition sentinel ErrCompetitionCancelled to FailedPrecondition —
// the same one-concept-two-codes shape as #131, in a switch whose very first
// case already conceded the point.
//
// The code it now answers with is FailedPrecondition, per gRPC's own
// definitions: INVALID_ARGUMENT is a problem with the argument *regardless of
// the state of the system*, while FAILED_PRECONDITION is "the system is not in
// a state required for the operation's execution." This context's only raise
// site is Competition.Cancel's `if c.Status != StatusScheduled` guard — state-
// dependent by construction. The request is well-formed; nothing about the
// argument needs fixing.
//
// Structure mirrors internal/payments/adapter/grpcapi/error_mapping_test.go
// (T13.9), which this ticket's instruction 3 names as the template: a
// per-sentinel table, a cross-RPC equality assertion, and two source-level
// guards (a second error mapper; a domain sentinel with no mapping row).
//
// Handler-level with the in-memory port fakes this package already defines, for
// the reason authz_regression_test.go documents: the mapping under test is
// unaffected by which port implementation is behind it, and this environment
// has no Docker daemon, so the test has to be runnable where it was written
// (CLAUDE.md rule 10).
package grpcapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// --- fixtures ---------------------------------------------------------

const (
	mapHostID   = "map-host"
	mapPlayerID = "map-player"
)

// cancelCompetitionTwice seeds a Competition, cancels it as its Host, then
// cancels it again and returns the second call's error.
//
// Unlike Social Play's CancelGame — which checks EnsureNotCancelled first and
// so never reaches its aggregate's own transition guard —
// app.Service.CancelCompetition calls competition.Cancel() directly, so this
// path genuinely produces ErrIllegalStatusTransition. Established by reading
// the app method rather than assumed from the sentinel's name.
func cancelCompetitionTwice(t *testing.T) error {
	t.Helper()

	h, _ := newTestHandler()
	c := seedCompetition(t, h, mapHostID, 16, 2)
	req := &competitionsv1.CancelCompetitionRequest{CompetitionId: c.GetId()}
	if _, err := h.CancelCompetition(ctxAs(mapHostID), req); err != nil {
		t.Fatalf("seed CancelCompetition: %v", err)
	}
	_, err := h.CancelCompetition(ctxAs(mapHostID), req)
	return err
}

// --- the table --------------------------------------------------------

type errorMappingCase struct {
	name string
	// sentinel is the domain.Err* identifier this row pins, matched against
	// internal/competitions/domain/errors.go by
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
			name:     "cancelling an already-cancelled competition",
			sentinel: "ErrIllegalStatusTransition",
			err:      domain.ErrIllegalStatusTransition,
			wantCode: codes.FailedPrecondition,
			why: "gRPC defines InvalidArgument as a problem with the argument *regardless of the state of the system*; " +
				"Competition.Cancel's `Status != StatusScheduled` guard is state-dependent by construction, which is " +
				"exactly FailedPrecondition. #158: it must also agree with ErrCompetitionCancelled, which this same " +
				"switch already sends to FailedPrecondition",
			invoke: cancelCompetitionTwice,
		},
		{
			name:     "entering a cancelled competition",
			sentinel: "ErrCompetitionCancelled",
			err:      domain.ErrCompetitionCancelled,
			wantCode: codes.FailedPrecondition,
			why: "the sibling that already conceded the point — a precondition failure on the resource's own lifecycle, " +
				"not a capacity conflict and not a malformed request",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				if _, err := h.CancelCompetition(ctxAs(mapHostID), &competitionsv1.CancelCompetitionRequest{
					CompetitionId: c.GetId(),
				}); err != nil {
					t.Fatalf("seed CancelCompetition: %v", err)
				}
				_, err := h.EnterCompetition(ctxAs(mapPlayerID), &competitionsv1.EnterCompetitionRequest{
					CompetitionId: c.GetId(),
				})
				return err
			},
		},

		// --- conflict ---
		{
			name:     "entering the same competition twice",
			sentinel: "ErrAlreadyEntered",
			err:      domain.ErrAlreadyEntered,
			wantCode: codes.AlreadyExists,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				req := &competitionsv1.EnterCompetitionRequest{CompetitionId: c.GetId()}
				if _, err := h.EnterCompetition(ctxAs(mapPlayerID), req); err != nil {
					t.Fatalf("seed EnterCompetition: %v", err)
				}
				_, err := h.EnterCompetition(ctxAs(mapPlayerID), req)
				return err
			},
		},
		{
			name:     "entering a competition at capacity",
			sentinel: "ErrCompetitionFull",
			err:      domain.ErrCompetitionFull,
			wantCode: codes.AlreadyExists,
			why: "AlreadyExists per PR #87's review, matching Social Play's ErrGameFull precedent — a contended-capacity " +
				"conflict that would have succeeded moments earlier, unlike ErrCompetitionCancelled's lifecycle failure",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 1, 0)
				if _, err := h.EnterCompetition(ctxAs(mapPlayerID), &competitionsv1.EnterCompetitionRequest{
					CompetitionId: c.GetId(),
				}); err != nil {
					t.Fatalf("seed EnterCompetition: %v", err)
				}
				_, err := h.EnterCompetition(ctxAs("map-other-player"), &competitionsv1.EnterCompetitionRequest{
					CompetitionId: c.GetId(),
				})
				return err
			},
		},
		{
			name:     "court already held by an overlapping booking",
			sentinel: "ErrCourtUnavailable",
			err:      domain.ErrCourtUnavailable,
			wantCode: codes.AlreadyExists,
			why:      "this context's local translation of booking's ErrCourtDoubleBooked, and it takes that sentinel's conflict code",
		},

		// --- not found ---
		{
			name:     "unknown competition id",
			sentinel: "ErrCompetitionNotFound",
			err:      domain.ErrCompetitionNotFound,
			wantCode: codes.NotFound,
			why: "also the sentinel translateEntryErr's 23503 arm returns for a concurrent-delete race on " +
				"competition_entries.competition_id (T17.3, #195) — same value, so this row already covers " +
				"that raise site; proven against a real Postgres in adapter/postgres's FK-race integration test, " +
				"which no in-memory fake can model",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CancelCompetition(ctxAs(mapHostID), &competitionsv1.CancelCompetitionRequest{
					CompetitionId: fixtureUnknownCompetitionID,
				})
				return err
			},
		},
		{
			name:     "assigning a competition admin who already holds an assignment",
			sentinel: "ErrAlreadyCompetitionAdmin",
			err:      domain.ErrAlreadyCompetitionAdmin,
			wantCode: codes.AlreadyExists,
			why: "T15.3 (#168): the request names a state that already exists, the group ErrAlreadyEntered occupies. " +
				"The domain pre-check and competition_admins' composite primary key produce the same sentinel by " +
				"design (CLAUDE.md rules 4 and 5), so this code covers both",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				req := &competitionsv1.AssignCompetitionAdminRequest{CompetitionId: c.GetId(), UserId: mapPlayerID}
				if _, err := h.AssignCompetitionAdmin(ctxAs(mapHostID), req); err != nil {
					t.Fatalf("seed AssignCompetitionAdmin: %v", err)
				}
				_, err := h.AssignCompetitionAdmin(ctxAs(mapHostID), req)
				return err
			},
		},
		{
			name:     "revoking a competition admin assignment that does not exist",
			sentinel: "ErrCompetitionAdminNotFound",
			err:      domain.ErrCompetitionAdminNotFound,
			wantCode: codes.NotFound,
			why: "T15.3 (#168): the Competition is real, the assignment is not — the same distinction " +
				"ErrCompetitionEntryNotFound draws against its own parent Competition. NOT a silent success: a revoke " +
				"that removed nothing must not confirm a Host's belief about who holds authority",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.RevokeCompetitionAdmin(ctxAs(mapHostID), &competitionsv1.RevokeCompetitionAdminRequest{
					CompetitionId: c.GetId(),
					UserId:        "map-never-assigned",
				})
				return err
			},
		},
		{
			name:     "assigning a blank competition admin user id",
			sentinel: "ErrEmptyCompetitionAdminUserID",
			err:      domain.ErrEmptyCompetitionAdminUserID,
			wantCode: codes.InvalidArgument,
			why: "T15.3 (#168): a blank user id names nobody regardless of system state — the textbook " +
				"InvalidArgument, and the sibling of ErrEmptyPlayerID",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.AssignCompetitionAdmin(ctxAs(mapHostID), &competitionsv1.AssignCompetitionAdminRequest{
					CompetitionId: c.GetId(),
					UserId:        "",
				})
				return err
			},
		},
		{
			name:     "assigning the host as their own competition admin",
			sentinel: "ErrHostCannotBeCompetitionAdmin",
			err:      domain.ErrHostCannotBeCompetitionAdmin,
			wantCode: codes.InvalidArgument,
			why: "T15.3 (#168), and the row here whose code is a judgement call rather than a lookup. It is " +
				"state-dependent in the literal sense — the same user_id is valid against a Competition they do not " +
				"host — so T14.7's own ErrIllegalStatusTransition argument appears to point at FailedPrecondition. It " +
				"does not: the distinction this context already draws is ErrGuestAllowanceExceeded's, where a limit " +
				"that is fixed, knowable and caller-visible (a Competition's host_id is on the Competition message the " +
				"caller read to get the competition_id) makes a violating request a client-input defect. " +
				"FailedPrecondition's sentinels here are about a *lifecycle* that changes under the caller's feet; a " +
				"Competition's Host does not. Matches Social Play's identical placement of ErrHostCannotBeGameAdmin",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.AssignCompetitionAdmin(ctxAs(mapHostID), &competitionsv1.AssignCompetitionAdminRequest{
					CompetitionId: c.GetId(),
					UserId:        mapHostID,
				})
				return err
			},
		},
		{
			name:     "unknown venue facility id",
			sentinel: "ErrFacilityNotFound",
			err:      domain.ErrFacilityNotFound,
			wantCode: codes.NotFound,
			why: "an unknown venue_facility_id is a 404-shaped answer, not a 500 (T9.4); also the sentinel " +
				"translateErr's 23503 arm returns for a concurrent-delete race on competitions.venue_facility_id " +
				"(T17.3, #195) — same value, so this row already covers that raise site too; proven against a " +
				"real Postgres in adapter/postgres's FK-race integration test, which no in-memory fake can model",
		},
		{
			name:     "court id naming no court",
			sentinel: "ErrCourtNotFound",
			err:      domain.ErrCourtNotFound,
			wantCode: codes.NotFound,
			why: "T15.6 (closes #185): an unknown-but-well-formed court_ids entry answered Internal/500 until this " +
				"ticket — its Postgres FK violation was never translated, and the (correct) %s-stripping at the " +
				"Booking boundary left this switch nothing to match. Deliberately NOT ErrCourtUnavailable's " +
				"AlreadyExists: a court that does not exist is not a court that is busy. This row was added " +
				"because THIS FILE's own TestSentinelToCodeTableCoversEveryDomainSentinel failed on the new " +
				"sentinel first",
			invoke: func(t *testing.T) error {
				// The REAL Booking chain over an FK-modelling repository
				// fake — see unknown_court_test.go.
				h, _, _ := newBookingBackedHandler()
				_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
					protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeUnknownCourt),
				))
				return err
			},
		},

		// --- permission denied ---
		{
			name:     "a non-host cancels the competition",
			sentinel: "ErrNotCompetitionHost",
			err:      domain.ErrNotCompetitionHost,
			wantCode: codes.PermissionDenied,
			why:      "a verified principal failing an object-level ownership check is 403-shaped, never 500-shaped (the BOLA rule)",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.CancelCompetition(ctxAs("map-stranger"), &competitionsv1.CancelCompetitionRequest{
					CompetitionId: c.GetId(),
				})
				return err
			},
		},
		{
			name:     "a non-host, non-admin reads the roster",
			sentinel: "ErrNotCompetitionHostOrAdmin",
			err:      domain.ErrNotCompetitionHostOrAdmin,
			wantCode: codes.PermissionDenied,
			why: "T15.4 (closes #147): the widened roster-read rejection — same BOLA-shaped 403 as " +
				"ErrNotCompetitionHost above, kept a distinct sentinel per its own DO-NOT-UNIFY note " +
				"because it gates a different permitted-actor set (Host or admin, not Host-only)",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.ListEntriesForCompetition(ctxAs("map-stranger"), &competitionsv1.ListEntriesForCompetitionRequest{
					CompetitionId: c.GetId(),
				})
				return err
			},
		},

		// --- invalid argument: genuinely state-independent request defects ---
		{
			name:     "guest count exceeds the allowance",
			sentinel: "ErrGuestAllowanceExceeded",
			err:      domain.ErrGuestAllowanceExceeded,
			wantCode: codes.InvalidArgument,
			why: "asking for more guests than the Host permits is a malformed request against a fixed, knowable limit, " +
				"not a conflict over contended capacity — the distinction domain.Enter deliberately checks in that order",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				c := seedCompetition(t, h, mapHostID, 16, 2)
				_, err := h.EnterCompetition(ctxAs(mapPlayerID), &competitionsv1.EnterCompetitionRequest{
					CompetitionId: c.GetId(),
					GuestCount:    9,
				})
				return err
			},
		},
		{
			name:     "inverted or zero-length time range",
			sentinel: "ErrInvalidTimeRange",
			err:      domain.ErrInvalidTimeRange,
			wantCode: codes.InvalidArgument,
			why:      "malformed request regardless of system state — the textbook InvalidArgument",
		},
		{
			name:     "malformed court id",
			sentinel: "ErrMalformedCourtID",
			err:      domain.ErrMalformedCourtID,
			wantCode: codes.InvalidArgument,
			why:      "T14.8 (#156): a non-uuid court id cannot name any court that exists, so it is a client-input defect regardless of system state — not an existence oracle, since the value could never resolve either way",
		},
		{
			name:     "non-positive capacity",
			sentinel: "ErrInvalidCapacity",
			err:      domain.ErrInvalidCapacity,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CreateCompetition(ctxAs(mapHostID), &competitionsv1.CreateCompetitionRequest{
					Name:          "No Capacity Open",
					Sessions:      []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
					Capacity:      0,
					PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
					EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
					Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
				})
				return err
			},
		},
		{
			name:     "no sessions",
			sentinel: "ErrEmptySessions",
			err:      domain.ErrEmptySessions,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CreateCompetition(ctxAs(mapHostID), &competitionsv1.CreateCompetitionRequest{
					Name:          "No Sessions Open",
					Sessions:      nil,
					Capacity:      16,
					PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
					EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
					Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
				})
				return err
			},
		},
		{
			name:     "a session with no court ids",
			sentinel: "ErrEmptyCourtIDs",
			err:      domain.ErrEmptyCourtIDs,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CreateCompetition(ctxAs(mapHostID), &competitionsv1.CreateCompetitionRequest{
					Name:          "No Courts Open",
					Sessions:      []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z")},
					Capacity:      16,
					PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
					EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
					Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
				})
				return err
			},
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
			name:     "invalid competition format",
			sentinel: "ErrInvalidFormat",
			err:      domain.ErrInvalidFormat,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "negative guest allowance",
			sentinel: "ErrInvalidGuestAllowance",
			err:      domain.ErrInvalidGuestAllowance,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid entry source",
			sentinel: "ErrInvalidEntrySource",
			err:      domain.ErrInvalidEntrySource,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "amount coupled with no or malformed currency",
			sentinel: "ErrInvalidMoney",
			err:      domain.ErrInvalidMoney,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CreateCompetition(ctxAs(mapHostID), &competitionsv1.CreateCompetitionRequest{
					Name:          "Bad Money Open",
					Sessions:      []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
					Capacity:      16,
					PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
					EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: ""},
					Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
				})
				return err
			},
		},
		{
			name:     "a competition whose own sessions double-book a court",
			sentinel: "ErrOverlappingSessions",
			err:      domain.ErrOverlappingSessions,
			wantCode: codes.InvalidArgument,
			why: "detectable up front from the request alone, without consulting any stored state — unlike " +
				"ErrCourtUnavailable, which is about a court another aggregate already holds",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CreateCompetition(ctxAs(mapHostID), &competitionsv1.CreateCompetitionRequest{
					Name: "Self-Overlapping Open",
					Sessions: []*competitionsv1.CompetitionSession{
						protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1"),
						protoSession("2026-09-01T10:00:00Z", "2026-09-01T13:00:00Z", "court-1"),
					},
					Capacity:      16,
					PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
					EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
					Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
				})
				return err
			},
		},

		// --- deliberately unreachable from any client-facing RPC ---
		//
		// toStatus does not match these two, so they fall through to Internal.
		// Verified at T14.7 rather than assumed: both are produced only through
		// port.CompetitionEntryPaymentUpdater, which the *Payments* context
		// drives (app.Service.MarkEntryPaymentStatus is its only caller, and no
		// RPC on this service reaches it). None of the seven RPCs on
		// CompetitionsService can raise either. Reaching this handler with one
		// would mean something upstream called an app method no RPC exposes,
		// which is a server fault — Internal is the honest code for it, and the
		// same decision toStatus's own comment records for Social Play's
		// equivalent pair.
		//
		// They are pinned rather than skipped so that the day an RPC does reach
		// one, this table is what fails.
		{
			name:     "unknown competition entry id (unreachable)",
			sentinel: "ErrCompetitionEntryNotFound",
			err:      domain.ErrCompetitionEntryNotFound,
			wantCode: codes.Internal,
			why:      "unreachable: produced only via port.CompetitionEntryPaymentUpdater, driven by Payments — see the block comment above",
		},
		{
			name:     "invalid payment status (unreachable)",
			sentinel: "ErrInvalidPaymentStatus",
			err:      domain.ErrInvalidPaymentStatus,
			wantCode: codes.Internal,
			why:      "unreachable: produced only via port.CompetitionEntryPaymentUpdater, driven by Payments — see the block comment above",
		},
	}
}

// --- assertions -------------------------------------------------------

// TestCompetitionsService_SentinelToCode pins every domain sentinel to exactly
// one status code across the whole service, and — where an RPC can surface it —
// proves the running handler agrees.
func TestCompetitionsService_SentinelToCode(t *testing.T) {
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

// TestCompetitionsService_SameSentinelSameCodeAcrossRPCs is #131's claim stated
// directly: the codes two *different* RPCs answer with are asserted equal to
// each other for the same sentinel, rather than each matching a constant.
// Re-introducing a per-RPC mapper fails here even if its author also updates
// the table above to describe the divergence.
//
// ErrIllegalStatusTransition has only one raise path to the wire in this
// context (CancelCompetition), so the pairs below are the sentinels two RPCs
// really can both surface. The equal-to-each-other property is what is under
// test, not which sentinel demonstrates it.
func TestCompetitionsService_SameSentinelSameCodeAcrossRPCs(t *testing.T) {
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
			name:     "unknown competition",
			sentinel: "ErrCompetitionNotFound",
			rpcA:     "CancelCompetition",
			viaA: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.CancelCompetition(ctxAs(mapHostID), &competitionsv1.CancelCompetitionRequest{
					CompetitionId: fixtureUnknownCompetitionID,
				})
				return err
			},
			rpcB: "EnterCompetition",
			viaB: func(t *testing.T) error {
				h, _ := newTestHandler()
				_, err := h.EnterCompetition(ctxAs(mapPlayerID), &competitionsv1.EnterCompetitionRequest{
					CompetitionId: fixtureUnknownCompetitionID,
				})
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

// TestCompetitionsService_LifecycleRejectionsAnswerOneCode is the sharper half
// of #158 for this context: EnterCompetition on a cancelled Competition
// (ErrCompetitionCancelled) and CancelCompetition on a cancelled Competition
// (ErrIllegalStatusTransition) are two sentinels for one concept — "this
// Competition's own lifecycle state forbids the operation". Before T14.7 they
// answered FailedPrecondition and InvalidArgument respectively.
//
// Kept separate from SameSentinelSameCodeAcrossRPCs above because that test is
// about one sentinel seen from two RPCs, while this is about one *concept*
// spread across two sentinels — which the per-sentinel table alone would not
// catch, since each sentinel is internally consistent.
func TestCompetitionsService_LifecycleRejectionsAnswerOneCode(t *testing.T) {
	t.Parallel()

	entering := codeOf(t, "EnterCompetition (cancelled)", func() error {
		h, _ := newTestHandler()
		c := seedCompetition(t, h, mapHostID, 16, 2)
		if _, err := h.CancelCompetition(ctxAs(mapHostID), &competitionsv1.CancelCompetitionRequest{
			CompetitionId: c.GetId(),
		}); err != nil {
			t.Fatalf("seed CancelCompetition: %v", err)
		}
		_, err := h.EnterCompetition(ctxAs(mapPlayerID), &competitionsv1.EnterCompetitionRequest{
			CompetitionId: c.GetId(),
		})
		return err
	}())

	cancelling := codeOf(t, "CancelCompetition (twice)", cancelCompetitionTwice(t))

	if entering != cancelling {
		t.Fatalf("entering a cancelled Competition answered %v but re-cancelling one answered %v — "+
			"one concept (\"this Competition's lifecycle state forbids the operation\") must mean one gRPC code "+
			"whichever RPC a client reaches it through (issues #131, #158)", entering, cancelling)
	}
	if entering != codes.FailedPrecondition {
		t.Fatalf("both lifecycle rejections answered %v, want FailedPrecondition — consistent but wrong is still wrong: "+
			"a state-machine violation is state-dependent by definition, which is what FailedPrecondition means and "+
			"what InvalidArgument (\"regardless of the state of the system\") denies", entering)
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

// TestCompetitionsService_HasExactlyOneErrorMapper is the structural guard
// against #131's actual shape. That inconsistency was not a typo in a mapping —
// it was a *second mapper* added beside the shared one, which no assertion about
// codes would have flagged as long as both mappers agreed with their own tests.
func TestCompetitionsService_HasExactlyOneErrorMapper(t *testing.T) {
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
// as the domain grows: a sentinel added to
// internal/competitions/domain/errors.go with no mapping row falls through
// toStatus's default to Internal, which is a 500 for a condition the domain
// anticipated. That is a silent bug today; here it is a failing test.
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
