// T68.2 (ADR-0005, closes #357) — the wire half: GetQuoteResponse names the
// currency its two amounts are in, and that field agrees with the currency on
// the DiscountRule nested in the same response.
//
// This is the level at which #357 was reported, so it is the level that has
// to be asserted. The domain and app tests prove the rule; only a response
// built by the real handler proves the contract — that the new field is
// populated rather than left at protobuf's zero value, which is the one
// mistake an app-layer test cannot see.
package grpcapi_test

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	"github.com/nhuthuynh/white-label/internal/platform/tz"
)

// newCurrencyQuoteHandler builds a handler whose single court is priced in
// bandCurrency on Mondays 06:00-17:00 (the same window newZoneHandler uses,
// and the one mapSlot's 09:00 falls inside), with discounts configured on its
// facility. The facility's zone is UTC and the REAL tz loader sits behind it,
// so nothing on the quote path is faked except the two repositories.
func newCurrencyQuoteHandler(t *testing.T, bandCurrency string, discounts ...domain.DiscountRule) *grpcapi.Handler {
	t.Helper()

	start, err := domain.NewClockTime(6, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	end, err := domain.NewClockTime(17, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}

	byFacility := map[string][]domain.DiscountRule{}
	if len(discounts) > 0 {
		byFacility[facilityID(1)] = discounts
	}

	svc := app.NewService(app.ServiceOptions{
		Bookings: fakeBookingRepo{},
		PricingRules: &fakePricingRepo{rules: []domain.PricingRule{{
			ID: "band", CourtID: courtID(1), Band: domain.BandWeekday,
			Weekdays: []time.Weekday{time.Monday}, Start: start, End: end,
			PriceCents: 2000, Currency: bandCurrency,
		}}},
		DiscountRules:  &fakeDiscountRepo{byFacility: byFacility},
		RecurringHires: newFakeRecurringRepo(),
		Facilities:     zoneOverrideFacilityLookup{zone: "UTC"},
		Identity:       newFakeIdentityLookup(),
		Zones:          tz.Loader{},
		IDs:            &fakeIDs{},
	})
	return grpcapi.NewHandler(svc)
}

func currencyQuoteRequest() *bookingv1.GetQuoteRequest {
	start, end := mapSlot()
	return &bookingv1.GetQuoteRequest{
		CourtId:  courtID(1),
		StartsAt: timestamppb.New(start),
		EndsAt:   timestamppb.New(end),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
	}
}

// TestGetQuoteResponse_NamesItsCurrency is requirement 1 on the wire.
//
// The `currency == ""` assertion is the one that matters and the one a
// hand-wired handler gets wrong: protobuf's zero value for a string field is
// "", so a handler that simply forgot to copy the field would return a
// perfectly valid response that still says nothing — exactly the state #357
// reported, with a field added to the schema to prove it was fixed.
func TestGetQuoteResponse_NamesItsCurrency(t *testing.T) {
	for _, want := range []string{"VND", "USD"} {
		want := want
		t.Run(want, func(t *testing.T) {
			h := newCurrencyQuoteHandler(t, want)

			resp, err := h.GetQuote(anonymous(), currencyQuoteRequest())
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if resp.GetCurrency() != want {
				t.Errorf("currency = %q, want %q — a response carrying two amounts and no unit is #357",
					resp.GetCurrency(), want)
			}
			if resp.GetPriceCents() != 2000 || resp.GetBandPriceCents() != 2000 {
				t.Errorf("price_cents/band_price_cents = %d/%d, want 2000/2000",
					resp.GetPriceCents(), resp.GetBandPriceCents())
			}
		})
	}
}

// TestGetQuoteResponse_CurrencyAgreesWithItsNestedDiscount is REQUIREMENT 2
// on the wire, and it is #357's complaint stated exactly: one message must
// not carry two conventions.
//
// The assertion compares the two fields of the one response to each other,
// never to a literal, so what is pinned is the agreement rather than two
// equalities that happen to coincide.
func TestGetQuoteResponse_CurrencyAgreesWithItsNestedDiscount(t *testing.T) {
	h := newCurrencyQuoteHandler(t, "VND", fixedAmountRule(t, "VND", 500))

	resp, err := h.GetQuote(anonymous(), currencyQuoteRequest())
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if resp.GetDiscount() == nil {
		t.Fatal("no discount on the response, so this test would assert nothing")
	}
	if resp.GetDiscount().GetCurrency() != resp.GetCurrency() {
		t.Errorf("discount.currency = %q but the response's own currency = %q — one message, two conventions "+
			"is the defect #357 reported", resp.GetDiscount().GetCurrency(), resp.GetCurrency())
	}
	if resp.GetPriceCents() != 1500 {
		t.Errorf("price_cents = %d, want 1500 — the discount must actually have been applied", resp.GetPriceCents())
	}
}

// TestGetQuoteResponse_CannotDisagreeWithItsNestedDiscount is the test
// REQUIREMENT 2 asks for: the one that fails if the two currencies could
// disagree.
//
// There is no response here to inspect, and that is the point — the only way
// to guarantee the two fields agree is to refuse to build a response in which
// they would not. Without app.Service.GetQuote's EnsureCurrency call this
// returns 200 with `currency: "VND"` and `discount.currency: "USD"`, having
// taken 500 US cents off a dong price.
//
// FailedPrecondition, not InvalidArgument: the request is well formed and
// will succeed unchanged once the Facility Owner fixes the promotion.
func TestGetQuoteResponse_CannotDisagreeWithItsNestedDiscount(t *testing.T) {
	h := newCurrencyQuoteHandler(t, "VND", fixedAmountRule(t, "USD", 500))

	resp, err := h.GetQuote(anonymous(), currencyQuoteRequest())
	if err == nil {
		t.Fatalf("GetQuote returned %+v with no error; a response whose currency and discount.currency "+
			"disagree must never be built", resp)
	}
	if got := status.Code(err); got != codes.FailedPrecondition {
		t.Errorf("code = %s, want %s", got, codes.FailedPrecondition)
	}
}

// fixedAmountRule builds a fixed_amount DiscountRule in currency, through the
// real constructor so the fixture is a rule that could have been created.
func fixedAmountRule(t *testing.T, currency string, cents int64) domain.DiscountRule {
	t.Helper()
	r, err := domain.NewDiscountRule(
		"currency-fixture", facilityID(1),
		domain.DiscountTypeFixedAmount,
		domain.DiscountAmount{Fixed: domain.Money{Cents: cents, Currency: currency}},
		[]domain.Source{domain.SourceIndividual},
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		domain.NoEnd(),
	)
	if err != nil {
		t.Fatalf("NewDiscountRule fixture: %v", err)
	}
	return r
}
