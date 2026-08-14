// T11.2 — object-level authorization and error-code regression tests for
// Booking's new DiscountRule endpoints, run through the real gRPC handler
// rather than only the app-layer unit tests in internal/booking/app.
// This is the same "does the guarantee survive the full stack" pattern
// T5.5 (Social Play), T6.7 (Payments) and T7.7 (Facilities) already
// established, applied to the first Booking RPC that has an owner check at
// all.
//
// Handler-level (real grpcapi.Handler + real app.Service + real domain, with
// in-memory fakes standing in for the Postgres adapters) rather than a
// -tags=integration testcontainers test, for the two reasons T7.7's own file
// gives: the checks under test live entirely in the app/domain/port layers
// that a real Postgres round trip would not influence, and this environment
// has no Docker daemon, so a test that cannot run here would prove nothing
// (CLAUDE.md rule 10).
//
// It does need internal/gen, so `make test-domain` does not cover it —
// `go test ./...` and `make test` do.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// --- fixtures ---------------------------------------------------------
//
// UUID-shaped, for the reason T11.9 generalized across every context: the
// real adapters store these against `uuid` columns, so a "facility-1"-shaped
// fixture would be a value production can never hold, and would hide exactly
// the class of bug the app layer's shape guards exist to catch.

func facilityID(n int) string { return fmt.Sprintf("00000000-0000-4000-c000-%012d", n) }
func courtID(n int) string    { return fmt.Sprintf("00000000-0000-4000-9000-%012d", n) }
func userID(n int) string     { return fmt.Sprintf("00000000-0000-4000-b000-%012d", n) }

const (
	ownerUser    = 1
	attackerUser = 2
)

// --- port fakes -------------------------------------------------------

type fakeBookingRepo struct{}

func (fakeBookingRepo) Create(_ context.Context, b domain.Booking) (domain.Booking, error) {
	return b, nil
}

func (fakeBookingRepo) ListActiveForCourt(_ context.Context, _ string, _ domain.TimeRange) ([]domain.Booking, error) {
	return nil, nil
}

func (fakeBookingRepo) GetByID(_ context.Context, _ string) (domain.Booking, error) {
	return domain.Booking{}, domain.ErrBookingNotFound
}

func (fakeBookingRepo) Update(_ context.Context, b domain.Booking) (domain.Booking, error) {
	return b, nil
}

type fakePricingRepo struct{ rules []domain.PricingRule }

func (r *fakePricingRepo) ListForCourt(_ context.Context, courtID string) ([]domain.PricingRule, error) {
	out := make([]domain.PricingRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.CourtID == courtID {
			out = append(out, rule)
		}
	}
	return out, nil
}

type fakeDiscountRepo struct {
	byFacility  map[string][]domain.DiscountRule
	createCalls int
}

func (r *fakeDiscountRepo) Create(_ context.Context, rule domain.DiscountRule) (domain.DiscountRule, error) {
	r.createCalls++
	r.byFacility[rule.FacilityID] = append(r.byFacility[rule.FacilityID], rule)
	return rule, nil
}

func (r *fakeDiscountRepo) ListForFacility(_ context.Context, id string) ([]domain.DiscountRule, error) {
	return r.byFacility[id], nil
}

// fakeFacilityLookup stands in for internal/booking/adapter/facilities,
// returning the same two Booking-local sentinels that adapter translates the
// Facilities context's errors into.
type fakeFacilityLookup struct{}

func (fakeFacilityLookup) EnsureFacilityOwner(_ context.Context, fID, actorUserID string) error {
	if fID != facilityID(1) {
		return domain.ErrFacilityNotFound
	}
	if actorUserID != userID(ownerUser) {
		return domain.ErrNotFacilityOwner
	}
	return nil
}

func (fakeFacilityLookup) FacilityIDForCourt(_ context.Context, cID string) (string, error) {
	if cID != courtID(1) {
		return "", domain.ErrFacilityNotFound
	}
	return facilityID(1), nil
}

// CourtIDsForFacility (T11.5) is the inverse of FacilityIDForCourt above:
// facilityID(1) owns exactly courtID(1).
func (fakeFacilityLookup) CourtIDsForFacility(_ context.Context, fID string) ([]string, error) {
	if fID != facilityID(1) {
		return nil, domain.ErrFacilityNotFound
	}
	return []string{courtID(1)}, nil
}

type fakeIDs struct{ n int }

func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("00000000-0000-4000-a000-%012d", f.n)
}

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against the fakes above.
// The court courtID(1) is priced at 2000 on Mondays 06:00-17:00 and belongs
// to facilityID(1), which is owned by userID(ownerUser).
func newTestHandler(t *testing.T) (*grpcapi.Handler, *fakeDiscountRepo) {
	t.Helper()

	start, err := domain.NewClockTime(6, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	end, err := domain.NewClockTime(17, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}

	pricing := &fakePricingRepo{rules: []domain.PricingRule{{
		ID: "band", CourtID: courtID(1), Band: domain.BandWeekday,
		Weekdays: []time.Weekday{time.Monday}, Start: start, End: end, PriceCents: 2000,
	}}}
	discounts := &fakeDiscountRepo{byFacility: make(map[string][]domain.DiscountRule)}
	svc := app.NewService(fakeBookingRepo{}, pricing, discounts, newFakeRecurringRepo(),
		fakeFacilityLookup{}, newFakeIdentityLookup(), &fakeIDs{})
	return grpcapi.NewHandler(svc), discounts
}

func validCreateRequest() *bookingv1.CreateDiscountRuleRequest {
	return &bookingv1.CreateDiscountRuleRequest{
		FacilityId:   facilityID(1),
		DiscountType: bookingv1.DiscountType_DISCOUNT_TYPE_PERCENT,
		Percent:      15,
		AppliesTo:    []bookingv1.Source{bookingv1.Source_SOURCE_INDIVIDUAL},
		StartsAt:     timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndCondition: &bookingv1.EndCondition{Kind: bookingv1.EndConditionKind_END_CONDITION_KIND_NO_END},
	}
}

func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("request succeeded, want %v", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	if st.Code() != want {
		t.Fatalf("code = %v (%v), want %v", st.Code(), st.Message(), want)
	}
}

// --- CreateDiscountRule: authorization ---------------------------------

// TestCreateDiscountRule_OwnerSucceeds is the positive control the rejection
// tests below need: without it, a handler that rejected *everyone* would pass
// every authz assertion in this file.
func TestCreateDiscountRule_OwnerSucceeds(t *testing.T) {
	h, discounts := newTestHandler(t)

	resp, err := h.CreateDiscountRule(ownerCtx(), validCreateRequest())
	if err != nil {
		t.Fatalf("CreateDiscountRule(owner): %v", err)
	}
	if got := resp.GetDiscountRule().GetId(); got != "00000000-0000-4000-a000-000000000001" {
		t.Errorf("id = %q, want the server-minted id — CreateDiscountRuleRequest has no id field to squat", got)
	}
	if resp.GetDiscountRule().GetDiscountType() != bookingv1.DiscountType_DISCOUNT_TYPE_PERCENT {
		t.Errorf("discount_type = %v, want PERCENT", resp.GetDiscountRule().GetDiscountType())
	}
	if discounts.createCalls != 1 {
		t.Errorf("repository Create called %d times, want 1", discounts.createCalls)
	}
}

// TestCreateDiscountRule_NonOwnerIsPermissionDenied is the BOLA regression:
// a different actor_user_id than the Facility's owner must be rejected with
// PermissionDenied through the full stack, and must never reach the
// repository — asserting only the code would still pass if the write had
// already happened.
func TestCreateDiscountRule_NonOwnerIsPermissionDenied(t *testing.T) {
	h, discounts := newTestHandler(t)

	req := validCreateRequest()

	// T12.7: the non-owner is now a *verified* non-owner. Before this ticket
	// this line was req.ActorUserId = userID(attackerUser) — a claim the
	// handler took at face value.
	_, err := h.CreateDiscountRule(ctxAs(userID(attackerUser)), req)
	requireCode(t, err, codes.PermissionDenied)
	if discounts.createCalls != 0 {
		t.Fatalf("a non-owner's rule reached the repository (%d calls) — the ownership check must run before anything is persisted", discounts.createCalls)
	}
}

// TestCreateDiscountRule_UnknownOrMalformedFacilityIsNotFound covers the
// ticket's NotFound mapping for both shapes of unresolvable Facility. The
// malformed values include the braced and urn: forms that pass
// github.com/google/uuid's Validate but fail pgtype.UUID.Scan — the reason
// the app-layer guard is a canonical-form check.
func TestCreateDiscountRule_UnknownOrMalformedFacilityIsNotFound(t *testing.T) {
	ids := []string{
		facilityID(77), // well-formed, unknown
		"",
		"not-a-uuid",
		"facility-1",
		"'; DROP TABLE discount_rules;--",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			h, discounts := newTestHandler(t)

			req := validCreateRequest()
			req.FacilityId = id

			_, err := h.CreateDiscountRule(ownerCtx(), req)
			requireCode(t, err, codes.NotFound)
			if discounts.createCalls != 0 {
				t.Fatalf("an unresolvable facility's rule reached the repository (%d calls)", discounts.createCalls)
			}
		})
	}
}

// --- CreateDiscountRule: validation ------------------------------------

// TestCreateDiscountRule_InvalidInputIsInvalidArgument walks every way a
// DiscountRule can be malformed and requires InvalidArgument for each — none
// of them may fall through to the default Internal branch, which would report
// a caller's mistake as a server fault.
func TestCreateDiscountRule_InvalidInputIsInvalidArgument(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*bookingv1.CreateDiscountRuleRequest)
	}{
		{
			name: "unspecified discount type",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) {
				r.DiscountType = bookingv1.DiscountType_DISCOUNT_TYPE_UNSPECIFIED
			},
		},
		{
			name:   "percent above 100",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) { r.Percent = 120 },
		},
		{
			name:   "percent of zero",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) { r.Percent = 0 },
		},
		{
			name: "fixed amount of zero",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) {
				r.DiscountType = bookingv1.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT
				r.Percent = 0
				r.FixedAmountCents = 0
			},
		},
		{
			name:   "empty applies_to",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) { r.AppliesTo = nil },
		},
		{
			name: "applies_to containing an unspecified source",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) {
				r.AppliesTo = []bookingv1.Source{bookingv1.Source_SOURCE_UNSPECIFIED}
			},
		},
		{
			name: "end after zero occurrences",
			mutate: func(r *bookingv1.CreateDiscountRuleRequest) {
				r.EndCondition = &bookingv1.EndCondition{
					Kind:        bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_OCCURRENCES,
					Occurrences: 0,
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h, discounts := newTestHandler(t)

			req := validCreateRequest()
			tt.mutate(req)

			_, err := h.CreateDiscountRule(ownerCtx(), req)
			requireCode(t, err, codes.InvalidArgument)
			if discounts.createCalls != 0 {
				t.Fatalf("an invalid rule reached the repository (%d calls)", discounts.createCalls)
			}
		})
	}
}

// --- GetQuote: the discount actually reaching the wire ------------------

// TestGetQuote_DiscountReachesTheWire is T11.2's headline AC at the API
// boundary: a discount a Facility Owner created through CreateDiscountRule
// changes the price GetQuote returns, in the same process, without any
// fixture reaching into the repository by hand. Both numbers are asserted —
// a response that discounted price_cents but lost band_price_cents would
// leave the UI unable to show what was taken off.
func TestGetQuote_DiscountReachesTheWire(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler(t)

	if _, err := h.CreateDiscountRule(ownerCtx(), validCreateRequest()); err != nil {
		t.Fatalf("CreateDiscountRule: %v", err)
	}

	// 2026-08-03 is a Monday; 09:00-10:00 is inside the 2000-cent band.
	resp, err := h.GetQuote(ctx, &bookingv1.GetQuoteRequest{
		CourtId:  courtID(1),
		StartsAt: timestamppb.New(time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)),
		EndsAt:   timestamppb.New(time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if resp.GetPriceCents() != 1700 {
		t.Errorf("price_cents = %d, want 1700 (2000 less 15%%) — the discount is decorative if this stays 2000", resp.GetPriceCents())
	}
	if resp.GetBandPriceCents() != 2000 {
		t.Errorf("band_price_cents = %d, want the pre-discount 2000", resp.GetBandPriceCents())
	}
	if resp.GetDiscount() == nil {
		t.Fatal("discount is absent on a quote that was discounted")
	}
	if resp.GetDiscount().GetPercent() != 15 {
		t.Errorf("discount.percent = %v, want 15", resp.GetDiscount().GetPercent())
	}
}

// TestGetQuote_UnspecifiedSourceUsesTheDocumentedDefault pins the contract
// GetQuoteRequest.source states: an unspecified source is SOURCE_INDIVIDUAL.
// Without this, every already-shipped client of an endpoint that predates the
// field (web/src/composables/useCourtBooking.ts sends no source) would quietly
// stop seeing discounts it should see.
func TestGetQuote_UnspecifiedSourceUsesTheDocumentedDefault(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler(t)

	if _, err := h.CreateDiscountRule(ownerCtx(), validCreateRequest()); err != nil {
		t.Fatalf("CreateDiscountRule: %v", err)
	}

	resp, err := h.GetQuote(ctx, &bookingv1.GetQuoteRequest{
		CourtId:  courtID(1),
		StartsAt: timestamppb.New(time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)),
		EndsAt:   timestamppb.New(time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)),
		// Source deliberately omitted.
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if resp.GetPriceCents() != 1700 {
		t.Errorf("price_cents = %d, want 1700 — an unspecified source must resolve as SOURCE_INDIVIDUAL", resp.GetPriceCents())
	}
}

// TestGetQuote_NoDiscountLeavesTheFieldAbsent proves "no discount" is absence
// rather than a zero-valued rule on the wire. A client reading
// `discount != null` to decide whether to show a struck-through price would
// otherwise show a 0%-off discount on every undiscounted quote.
func TestGetQuote_NoDiscountLeavesTheFieldAbsent(t *testing.T) {
	h, _ := newTestHandler(t)

	resp, err := h.GetQuote(context.Background(), &bookingv1.GetQuoteRequest{
		CourtId:  courtID(1),
		StartsAt: timestamppb.New(time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)),
		EndsAt:   timestamppb.New(time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if resp.GetPriceCents() != 2000 || resp.GetBandPriceCents() != 2000 {
		t.Errorf("price_cents/band_price_cents = %d/%d, want 2000/2000", resp.GetPriceCents(), resp.GetBandPriceCents())
	}
	if resp.GetDiscount() != nil {
		t.Errorf("discount = %v on an undiscounted quote, want absent", resp.GetDiscount())
	}
}

// TestGetQuote_AmbiguousDiscountIsFailedPrecondition pins ADR-0002's
// precedent at the wire: two rules matching the same (facility, source, time)
// is a data-configuration error the owner must fix, reported with the same
// code ErrAmbiguousPricingRule already uses — not silently resolved by
// picking one, and not a 500.
func TestGetQuote_AmbiguousDiscountIsFailedPrecondition(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler(t)

	for i := 0; i < 2; i++ {
		if _, err := h.CreateDiscountRule(ownerCtx(), validCreateRequest()); err != nil {
			t.Fatalf("CreateDiscountRule #%d: %v", i+1, err)
		}
	}

	_, err := h.GetQuote(ctx, &bookingv1.GetQuoteRequest{
		CourtId:  courtID(1),
		StartsAt: timestamppb.New(time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)),
		EndsAt:   timestamppb.New(time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
	})
	requireCode(t, err, codes.FailedPrecondition)
}

// --- ListDiscountRulesForFacility --------------------------------------

// TestListDiscountRulesForFacility reads back what CreateDiscountRule wrote,
// and pins the list-shaped convention for an unresolvable Facility: an empty
// list, not an error this endpoint never otherwise returns.
func TestListDiscountRulesForFacility(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler(t)

	if _, err := h.CreateDiscountRule(ownerCtx(), validCreateRequest()); err != nil {
		t.Fatalf("CreateDiscountRule: %v", err)
	}

	resp, err := h.ListDiscountRulesForFacility(ctx, &bookingv1.ListDiscountRulesForFacilityRequest{
		FacilityId: facilityID(1),
	})
	if err != nil {
		t.Fatalf("ListDiscountRulesForFacility: %v", err)
	}
	if len(resp.GetDiscountRules()) != 1 {
		t.Fatalf("got %d rules, want 1", len(resp.GetDiscountRules()))
	}
	if got := resp.GetDiscountRules()[0].GetAppliesTo(); len(got) != 1 || got[0] != bookingv1.Source_SOURCE_INDIVIDUAL {
		t.Errorf("applies_to = %v, want [SOURCE_INDIVIDUAL]", got)
	}

	for _, id := range []string{facilityID(77), "", "not-a-uuid"} {
		resp, err := h.ListDiscountRulesForFacility(ctx, &bookingv1.ListDiscountRulesForFacilityRequest{FacilityId: id})
		if err != nil {
			t.Fatalf("ListDiscountRulesForFacility(%q) = %v, want an empty list and no error", id, err)
		}
		if len(resp.GetDiscountRules()) != 0 {
			t.Errorf("ListDiscountRulesForFacility(%q) returned %d rules, want 0", id, len(resp.GetDiscountRules()))
		}
	}
}
