// T14.7 (closes issue #158, and with it the cross-context half of #131) —
// the BookingService's sentinel -> gRPC code mapping, asserted service-wide
// rather than per-RPC.
//
// The defect this file exists to prevent, and which it caught: Booking's
// *single* toStatus mapped two status-transition sentinels to two different
// codes — ErrInvalidRecurringHireStatusTransition to FailedPrecondition and
// ErrIllegalStatusTransition to InvalidArgument — inside one function, with no
// RPC boundary to explain it. That is #131's one-concept-two-codes shape
// without even the excuse of a second mapper.
//
// The code both now answer with is FailedPrecondition, per gRPC's own
// definitions: INVALID_ARGUMENT is a problem with the argument *regardless of
// the state of the system*, while FAILED_PRECONDITION is "the system is not in
// a state required for the operation's execution." Every raise site of
// ErrIllegalStatusTransition in this context is
// `if b.Status != StatusConfirmed { return ErrIllegalStatusTransition }` —
// state-dependent by construction. The request is well-formed; re-sending it
// unchanged after the world moves back would succeed, which is precisely the
// distinction FailedPrecondition draws and InvalidArgument denies.
//
// Structure mirrors internal/payments/adapter/grpcapi/error_mapping_test.go
// (T13.9), which this ticket's instruction 3 names as the template:
//
//  1. TestBookingService_SentinelToCode — one row per sentinel in
//     internal/booking/domain/errors.go. Each row pins the code directly
//     through the mapper, and rows whose sentinel a real RPC can surface
//     additionally drive that RPC and assert it agrees. A mapping cannot be
//     changed without a test changing with it.
//  2. TestBookingService_SameSentinelSameCodeAcrossRPCs — the #131 claim
//     stated directly: one sentinel driven through two *different* RPCs, with
//     the codes asserted equal to each other rather than to a constant, so a
//     future re-divergence fails even if someone updates the table to match it.
//  3. TestBookingService_HasExactlyOneErrorMapper /
//     TestSentinelToCodeTableCoversEveryDomainSentinel — source-level guards: a
//     second `func(error) error` mapper in handler.go, or a new sentinel added
//     to domain/errors.go with no mapping row, fails the tests rather than
//     silently shipping. #131 was created by exactly the former; the latter is
//     what makes a future ticket's new sentinel enforced rather than remembered.
//
// Handler-level with the in-memory port fakes this package already defines, for
// the reason discount_regression_test.go documents: the mapping under test is
// unaffected by which port.Repository implementation is behind it, and this
// environment has no Docker daemon, so the test has to be runnable where it was
// written (CLAUDE.md rule 10).
package grpcapi_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/booking/domain"

	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// --- fixtures ---------------------------------------------------------

// mapSlot is a Monday inside the priced band newTestHandler installs
// (courtID(1), Mondays 06:00-17:00), so a booking over it is well-formed.
func mapSlot() (time.Time, time.Time) {
	start := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	return start, start.Add(time.Hour)
}

func mapCreateBookingRequest() *bookingv1.CreateBookingRequest {
	start, end := mapSlot()
	return &bookingv1.CreateBookingRequest{
		CourtId:  courtID(1),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
		StartsAt: timestamppb.New(start),
		EndsAt:   timestamppb.New(end),
	}
}

// mustCreateBooking creates a booking through the real RPC and returns its
// server-minted id, so the cancel rows below operate on a genuinely stored
// aggregate rather than a hand-built one.
func mustCreateBooking(t *testing.T, h *recurringHarness) string {
	t.Helper()

	resp, err := h.handler.CreateBooking(anonymous(), mapCreateBookingRequest())
	if err != nil {
		t.Fatalf("seed CreateBooking: %v", err)
	}
	return resp.GetBooking().GetId()
}

// cancelTwice creates a booking, cancels it, then cancels it again and returns
// the second call's error — a Booking.Cancel() state-machine violation reached
// through production code end to end.
func cancelTwice(t *testing.T) error {
	t.Helper()

	h := newRecurringHandler()
	id := mustCreateBooking(t, h)
	if _, err := h.handler.CancelBooking(anonymous(), &bookingv1.CancelBookingRequest{BookingId: id}); err != nil {
		t.Fatalf("seed CancelBooking: %v", err)
	}
	_, err := h.handler.CancelBooking(anonymous(), &bookingv1.CancelBookingRequest{BookingId: id})
	return err
}

// approveTwice requests a template, approves it, then approves it again and
// returns the second call's error — the recurring-hire state machine's own
// violation, the sibling sentinel this ticket makes ErrIllegalStatusTransition
// agree with.
func approveTwice(t *testing.T) error {
	t.Helper()

	h := newRecurringHandler()
	tmpl := mustRequestTemplate(t, h, 4)
	req := &bookingv1.ApproveRecurringHireRequest{TemplateId: tmpl.GetId()}
	if _, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), req); err != nil {
		t.Fatalf("seed ApproveRecurringHire: %v", err)
	}
	_, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), req)
	return err
}

// --- the table --------------------------------------------------------

// errorMappingCase is one sentinel, the code the whole service must answer with
// for it, and optionally the RPC call that provokes it through real production
// code.
type errorMappingCase struct {
	name string
	// sentinel is the domain.Err* identifier this row pins, matched against
	// internal/booking/domain/errors.go by
	// TestSentinelToCodeTableCoversEveryDomainSentinel.
	sentinel string
	// err is the sentinel value itself, pinned directly through the mapper so
	// every sentinel is covered whether or not an RPC can currently reach it.
	err      error
	wantCode codes.Code
	// invoke, when set, drives a real RPC that surfaces this sentinel and must
	// agree with wantCode. Left nil for sentinels no current RPC can raise, and
	// for those the handler rejects at the wire boundary before toStatus is
	// reached (fromProtoSource, fromProtoDiscountType and domain.NewTimeRange
	// all return status.Error directly) — attaching an invoke there would
	// assert the code of a path this file is not about.
	invoke func(t *testing.T) error
	// why records the reasoning for a code that is not self-evident, so a
	// future change has to argue with a stated position rather than a bare
	// constant.
	why string
}

func errorMappingCases() []errorMappingCase {
	return []errorMappingCase{
		// --- the sentinel this ticket is about ---
		{
			name:     "cancelling an already-cancelled booking",
			sentinel: "ErrIllegalStatusTransition",
			err:      domain.ErrIllegalStatusTransition,
			wantCode: codes.FailedPrecondition,
			why: "gRPC defines InvalidArgument as a problem with the argument *regardless of the state of the system*; " +
				"every raise site of this sentinel is a `Status != required` guard, so it is state-dependent by " +
				"construction, which is exactly FailedPrecondition (\"the system is not in a state required for the " +
				"operation's execution\"). #158: it must also agree with its own sibling " +
				"ErrInvalidRecurringHireStatusTransition, which this same function has always sent to FailedPrecondition",
			invoke: func(t *testing.T) error { return cancelTwice(t) },
		},
		{
			name:     "approving an already-decided template",
			sentinel: "ErrInvalidRecurringHireStatusTransition",
			err:      domain.ErrInvalidRecurringHireStatusTransition,
			wantCode: codes.FailedPrecondition,
			why:      "the sibling that already conceded the point — a legal request against an object in the wrong state",
			invoke:   func(t *testing.T) error { return approveTwice(t) },
		},

		// --- conflict ---
		{
			name:     "overlapping booking on the same court",
			sentinel: "ErrCourtDoubleBooked",
			err:      domain.ErrCourtDoubleBooked,
			wantCode: codes.AlreadyExists,
			why:      "the no-double-booking invariant; AlreadyExists is what makes README.md's \"overlapping booking returns 409\" smoke test true",
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				mustCreateBooking(t, h)
				_, err := h.handler.CreateBooking(anonymous(), mapCreateBookingRequest())
				return err
			},
		},

		// --- not found ---
		{
			name:     "unknown booking id",
			sentinel: "ErrBookingNotFound",
			err:      domain.ErrBookingNotFound,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.CancelBooking(anonymous(), &bookingv1.CancelBookingRequest{
					BookingId: "00000000-0000-4000-a000-000000009999",
				})
				return err
			},
		},
		{
			name:     "no pricing rule covers the slot",
			sentinel: "ErrNoPricingRule",
			err:      domain.ErrNoPricingRule,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				// Sunday — outside the Monday-only band newTestHandler installs.
				start := time.Date(2026, 1, 4, 9, 0, 0, 0, time.UTC)
				_, err := h.GetQuote(anonymous(), &bookingv1.GetQuoteRequest{
					CourtId:  courtID(1),
					StartsAt: timestamppb.New(start),
					EndsAt:   timestamppb.New(start.Add(time.Hour)),
				})
				return err
			},
		},
		{
			name:     "unknown or malformed facility id",
			sentinel: "ErrFacilityNotFound",
			err:      domain.ErrFacilityNotFound,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				req := validCreateRequest()
				req.FacilityId = facilityID(404)
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(ownerUser)), req)
				return err
			},
		},
		{
			name:     "unknown recurring hire template id",
			sentinel: "ErrRecurringHireTemplateNotFound",
			err:      domain.ErrRecurringHireTemplateNotFound,
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
					TemplateId: "00000000-0000-4000-a000-000000009999",
				})
				return err
			},
		},

		// --- permission denied ---
		{
			name:     "actor does not own the facility",
			sentinel: "ErrNotFacilityOwner",
			err:      domain.ErrNotFacilityOwner,
			wantCode: codes.PermissionDenied,
			why:      "a verified principal failing an object-level ownership check is 403-shaped, never 500-shaped (T5.2/T5.5's BOLA rule)",
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(attackerUser)), validCreateRequest())
				return err
			},
		},
		{
			name:     "actor subject resolves to no user",
			sentinel: "ErrUserNotFound",
			err:      domain.ErrUserNotFound,
			wantCode: codes.PermissionDenied,
			why: "PermissionDenied, not NotFound: RequestRecurringHire would otherwise become a user-enumeration " +
				"oracle reporting which subjects are registered (the sentinel's own doc comment in domain/errors.go)",
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.RequestRecurringHire(ctxAs("auth0|nobody-at-all"), validRequest(4))
				return err
			},
		},
		{
			name:     "actor does not hold the club role",
			sentinel: "ErrNotClub",
			err:      domain.ErrNotClub,
			wantCode: codes.PermissionDenied,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(playerUser)), validRequest(4))
				return err
			},
		},

		// --- failed precondition: data misconfiguration ---
		{
			name:     "two pricing rules match the slot",
			sentinel: "ErrAmbiguousPricingRule",
			err:      domain.ErrAmbiguousPricingRule,
			wantCode: codes.FailedPrecondition,
			why:      "a pricing-data misconfiguration (ADR-0002), distinguishable from an actual server bug so ops/clients can tell them apart",
		},
		{
			name:     "two discount rules match the slot",
			sentinel: "ErrAmbiguousDiscountRule",
			err:      domain.ErrAmbiguousDiscountRule,
			wantCode: codes.FailedPrecondition,
			why:      "same class of data misconfiguration as its pricing twin, so it reuses that mapping rather than inventing one",
		},

		// --- invalid argument: genuinely state-independent request defects ---
		{
			name:     "empty court id",
			sentinel: "ErrEmptyCourtID",
			err:      domain.ErrEmptyCourtID,
			wantCode: codes.InvalidArgument,
			why:      "malformed request regardless of system state — the textbook InvalidArgument",
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				req := mapCreateBookingRequest()
				req.CourtId = ""
				_, err := h.handler.CreateBooking(anonymous(), req)
				return err
			},
		},
		{
			name:     "inverted or zero-length time range",
			sentinel: "ErrInvalidTimeRange",
			err:      domain.ErrInvalidTimeRange,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "clock time outside a real day",
			sentinel: "ErrInvalidClockTime",
			err:      domain.ErrInvalidClockTime,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				req := validRequest(4)
				req.StartMinute = 99 * 60
				_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
				return err
			},
		},
		{
			name:     "source outside the four locked values",
			sentinel: "ErrInvalidSource",
			err:      domain.ErrInvalidSource,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "pricing slot spans two calendar days",
			sentinel: "ErrPricingSlotSpansMultipleDays",
			err:      domain.ErrPricingSlotSpansMultipleDays,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "empty facility id",
			sentinel: "ErrEmptyFacilityID",
			err:      domain.ErrEmptyFacilityID,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid discount type",
			sentinel: "ErrInvalidDiscountType",
			err:      domain.ErrInvalidDiscountType,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "discount amount out of range for its type",
			sentinel: "ErrInvalidDiscountAmount",
			err:      domain.ErrInvalidDiscountAmount,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				req := validCreateRequest()
				req.Percent = 900
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(ownerUser)), req)
				return err
			},
		},
		{
			name:     "discount rule applies to nothing",
			sentinel: "ErrEmptyAppliesTo",
			err:      domain.ErrEmptyAppliesTo,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				req := validCreateRequest()
				req.AppliesTo = nil
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(ownerUser)), req)
				return err
			},
		},
		{
			name:     "non-positive end-after-occurrences on a discount rule",
			sentinel: "ErrInvalidEndConditionOccurrences",
			err:      domain.ErrInvalidEndConditionOccurrences,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				req := validCreateRequest()
				req.EndCondition = &bookingv1.EndCondition{
					Kind:        bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_OCCURRENCES,
					Occurrences: 0,
				}
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(ownerUser)), req)
				return err
			},
		},
		{
			name:     "recurring hire template with no requester",
			sentinel: "ErrEmptyRequestedByUserID",
			err:      domain.ErrEmptyRequestedByUserID,
			wantCode: codes.InvalidArgument,
			why: "no current RPC can raise this — RequestRecurringHire derives the requester from the verified " +
				"principal (T12.7), so it is never empty by the time domain.NewRecurringHireTemplate sees it. " +
				"Pinned anyway: an unpinned sentinel is one that falls through to Internal the day a caller can reach it",
		},
		{
			name:     "recurring hire start not before end",
			sentinel: "ErrInvalidRecurringHireTimeRange",
			err:      domain.ErrInvalidRecurringHireTimeRange,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				req := validRequest(4)
				req.StartMinute, req.EndMinute = 10*60, 9*60
				_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
				return err
			},
		},
		{
			name:     "non-positive end-after-occurrences on a recurring hire",
			sentinel: "ErrInvalidRecurringHireEndAfterOccurrences",
			err:      domain.ErrInvalidRecurringHireEndAfterOccurrences,
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), validRequest(0))
				return err
			},
		},

		// --- deliberately unclassified ---
		{
			name:     "malformed court id on CreateBooking",
			sentinel: "ErrInvalidCourtReference",
			err:      domain.ErrInvalidCourtReference,
			wantCode: codes.Internal,
			why: "deliberately left unmatched in toStatus (T10.7, #97), matching this codebase's existing behavior " +
				"for a well-formed-but-unknown CourtID, whose Postgres FK violation is also unclassified and also " +
				"Internal today. A disclosed gap, not an oversight — pinned here so closing it is a visible test " +
				"change rather than a silent one. See the sentinel's own doc comment in domain/errors.go",
			invoke: func(t *testing.T) error {
				h := newRecurringHandler()
				req := mapCreateBookingRequest()
				req.CourtId = "not-a-uuid"
				_, err := h.handler.CreateBooking(anonymous(), req)
				return err
			},
		},
	}
}

// --- assertions -------------------------------------------------------

// TestBookingService_SentinelToCode pins every domain sentinel to exactly one
// status code across the whole service, and — where an RPC can surface it —
// proves the running handler agrees.
func TestBookingService_SentinelToCode(t *testing.T) {
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

// TestBookingService_SameSentinelSameCodeAcrossRPCs is #131's claim stated
// directly: it asserts the codes two *different* RPCs answer with are equal to
// each other for the same sentinel, rather than each matching a constant.
// Re-introducing a per-RPC mapper fails here even if its author also updates
// the table above to describe the divergence.
//
// ErrIllegalStatusTransition itself has only one raise path to the wire in this
// context (CancelBooking), so the pairs below are the sentinels two RPCs really
// can both surface. The equal-to-each-other property is what is under test, not
// which sentinel demonstrates it.
func TestBookingService_SameSentinelSameCodeAcrossRPCs(t *testing.T) {
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
			name:     "unknown facility",
			sentinel: "ErrFacilityNotFound",
			rpcA:     "CreateDiscountRule",
			viaA: func(t *testing.T) error {
				h, _ := newTestHandler(t)
				req := validCreateRequest()
				req.FacilityId = facilityID(404)
				_, err := h.CreateDiscountRule(ctxAs(subjectOf(ownerUser)), req)
				return err
			},
			rpcB: "RequestRecurringHire",
			viaB: func(t *testing.T) error {
				h := newRecurringHandler()
				req := validRequest(4)
				// A Court that resolves to no Facility.
				req.CourtId = courtID(404)
				_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
				return err
			},
		},
		{
			name:     "unresolvable actor",
			sentinel: "ErrUserNotFound",
			rpcA:     "RequestRecurringHire",
			viaA: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.RequestRecurringHire(ctxAs("auth0|nobody-at-all"), validRequest(4))
				return err
			},
			rpcB: "ListRecurringHireTemplatesForActor",
			viaB: func(t *testing.T) error {
				h := newRecurringHandler()
				_, err := h.handler.ListRecurringHireTemplatesForActor(
					ctxAs("auth0|nobody-at-all"),
					&bookingv1.ListRecurringHireTemplatesForActorRequest{},
				)
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

// TestBookingService_HasExactlyOneErrorMapper is the structural guard against
// #131's actual shape. That inconsistency was not a typo in a mapping — it was
// a *second mapper* added beside the shared one, which no assertion about codes
// would have flagged as long as both mappers agreed with their own tests.
// Adding another `func(error) error` to this handler now fails a test.
func TestBookingService_HasExactlyOneErrorMapper(t *testing.T) {
	t.Parallel()

	mappers := errorMapperNames(t, "handler.go")
	if len(mappers) != 1 || mappers[0] != "toStatus" {
		t.Fatalf("handler.go declares error mappers %v, want exactly [toStatus] — "+
			"a per-RPC error mapper is how issue #131's one-sentinel-two-codes split arose; "+
			"map the sentinel once in toStatus, or introduce a distinct sentinel if two RPCs "+
			"genuinely mean different things", mappers)
	}
}

// errorMapperNames returns the names of every package-level `func(error) error`
// declared in the given file — the shape of this package's status mappers, and
// narrow enough to exclude helpers like toProtoStatus(domain.Status).
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
// as the domain grows: a sentinel added to internal/booking/domain/errors.go
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

// uncoveredSentinels parses a domain errors.go and returns every Err* it
// declares that the covered set does not mention.
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

// compile-time assurance the context import stays used if every invoke above is
// ever removed; context is what ctxAs/anonymous return.
var _ = context.Background
