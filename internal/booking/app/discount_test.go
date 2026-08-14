package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// facilityID mints a deterministic, UUID-shaped Facility ID, for the same
// reason courtID does in malformed_id_test.go: Facility IDs are real UUIDs in
// Postgres (discount_rules.facility_id REFERENCES facilities (id)), so a
// fixture like "facility-1" would be a shape no real Facility has and the
// adapter cannot store — exactly the fixture infidelity T11.9 generalized a
// helper to prevent.
func facilityID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-c000-%012d", n)
}

func userID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-b000-%012d", n)
}

// fakeDiscountRepo is an in-memory port.DiscountRuleRepository.
//
// createCalls/listCalls are atomic for the same reason every other counter in
// this package is: they exist so a test can prove a rejected request never
// reached the repository at all, and they must stay race-free if a future
// test shares one fixture across t.Parallel() subtests.
type fakeDiscountRepo struct {
	rulesByFacility map[string][]domain.DiscountRule

	createCalls atomic.Int64
	listCalls   atomic.Int64
}

func newFakeDiscountRepo() *fakeDiscountRepo {
	return &fakeDiscountRepo{rulesByFacility: make(map[string][]domain.DiscountRule)}
}

func (r *fakeDiscountRepo) Create(_ context.Context, rule domain.DiscountRule) (domain.DiscountRule, error) {
	r.createCalls.Add(1)
	r.rulesByFacility[rule.FacilityID] = append(r.rulesByFacility[rule.FacilityID], rule)
	return rule, nil
}

func (r *fakeDiscountRepo) ListForFacility(_ context.Context, id string) ([]domain.DiscountRule, error) {
	r.listCalls.Add(1)
	return r.rulesByFacility[id], nil
}

// fakeFacilityLookup is an in-memory port.FacilityLookup: a facility->owner
// map plus a court->facility map, which is exactly the pair of answers the
// real adapter gets out of the Facilities context.
type fakeFacilityLookup struct {
	ownerByFacility  map[string]string
	facilityByCourt  map[string]string
	ensureOwnerCalls atomic.Int64
}

func (l *fakeFacilityLookup) EnsureFacilityOwner(_ context.Context, facilityID, actorUserID string) error {
	l.ensureOwnerCalls.Add(1)
	owner, ok := l.ownerByFacility[facilityID]
	if !ok {
		return domain.ErrFacilityNotFound
	}
	if owner != actorUserID {
		return domain.ErrNotFacilityOwner
	}
	return nil
}

func (l *fakeFacilityLookup) FacilityIDForCourt(_ context.Context, courtID string) (string, error) {
	id, ok := l.facilityByCourt[courtID]
	if !ok || id == "" {
		return "", domain.ErrFacilityNotFound
	}
	return id, nil
}

// CourtIDsForFacility is the inverse of facilityByCourt (T11.5). An unknown
// Facility is ErrFacilityNotFound rather than an empty slice, matching the
// port's contract — a Facility with no Courts and a Facility that does not
// exist are different answers.
func (l *fakeFacilityLookup) CourtIDsForFacility(_ context.Context, facilityID string) ([]string, error) {
	if _, ok := l.ownerByFacility[facilityID]; !ok {
		return nil, domain.ErrFacilityNotFound
	}
	out := make([]string, 0, len(l.facilityByCourt))
	for court, fID := range l.facilityByCourt {
		if fID == facilityID {
			out = append(out, court)
		}
	}
	return out, nil
}

func percentInput(fid, actor string) app.CreateDiscountRuleInput {
	return app.CreateDiscountRuleInput{
		FacilityID:   fid,
		ActorUserID:  actor,
		DiscountType: domain.DiscountTypePercent,
		Amount:       domain.DiscountAmount{Percent: 15},
		AppliesTo:    []domain.Source{domain.SourceIndividual},
		StartsAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndCondition: domain.NoEnd(),
	}
}

// newDiscountSvc builds a Service with the two new T11.2 dependencies wired
// to fakes, plus a facility (owned by owner) and a court belonging to it.
func newDiscountSvc(owner string) (*app.Service, *fakeDiscountRepo, *fakeFacilityLookup) {
	discounts := newFakeDiscountRepo()
	lookup := &fakeFacilityLookup{
		ownerByFacility: map[string]string{facilityID(1): owner},
		facilityByCourt: map[string]string{courtID(1): facilityID(1)},
	}
	svc := app.NewService(newInMemoryRepo(), &fakePricingRepo{}, discounts, newFakeRecurringHireRepo(), lookup, &fakeIdentityLookup{}, &sequentialIDs{})
	return svc, discounts, lookup
}

// TestCreateDiscountRule_OwnerCanCreate is the happy path, and also the
// creation-RPC checklist's item 1 in test form (T10 retro finding 3, sprint
// plan A4): the ID is minted server-side by the IDGenerator port, never taken
// from the caller — CreateDiscountRuleInput has no ID field at all, so there
// is no identifier for an anonymous caller to squat.
func TestCreateDiscountRule_OwnerCanCreate(t *testing.T) {
	t.Parallel()

	owner := userID(1)
	svc, discounts, _ := newDiscountSvc(owner)

	rule, err := svc.CreateDiscountRule(context.Background(), percentInput(facilityID(1), owner))
	if err != nil {
		t.Fatalf("CreateDiscountRule: %v", err)
	}
	if rule.ID != seqID(1) {
		t.Errorf("ID = %q, want the server-generated %q", rule.ID, seqID(1))
	}
	if rule.FacilityID != facilityID(1) {
		t.Errorf("FacilityID = %q, want %q", rule.FacilityID, facilityID(1))
	}
	if calls := discounts.createCalls.Load(); calls != 1 {
		t.Errorf("repository Create called %d times, want 1", calls)
	}
}

// TestCreateDiscountRule_NonOwnerIsRejectedBeforeTheRepository is T11.2's
// authorization AC: the actor check runs before anything is constructed or
// persisted, so a rejected actor has no code path to the repository at all
// (mirroring facilities' AddCourt, T7.7). Asserting the error alone would
// not prove that — only the untouched call counter does.
func TestCreateDiscountRule_NonOwnerIsRejectedBeforeTheRepository(t *testing.T) {
	t.Parallel()

	owner := userID(1)
	svc, discounts, _ := newDiscountSvc(owner)

	_, err := svc.CreateDiscountRule(context.Background(), percentInput(facilityID(1), userID(2)))
	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("error = %v, want %v", err, domain.ErrNotFacilityOwner)
	}
	if calls := discounts.createCalls.Load(); calls != 0 {
		t.Fatalf("a non-owner's rule reached the repository (%d calls); the ownership check must run first", calls)
	}
}

// TestCreateDiscountRule_UnknownAndMalformedFacilityAreNotFound covers the
// ticket's NotFound mapping. The malformed cases additionally prove the
// app-layer uuidShape guard (PR #89's Layer 2 pattern) fires before the
// lookup, so a non-UUID never reaches the Facilities adapter's mustUUID.
func TestCreateDiscountRule_UnknownAndMalformedFacilityAreNotFound(t *testing.T) {
	t.Parallel()

	owner := userID(1)

	t.Run("unknown but well-formed facility", func(t *testing.T) {
		t.Parallel()
		svc, discounts, lookup := newDiscountSvc(owner)

		_, err := svc.CreateDiscountRule(context.Background(), percentInput(facilityID(77), owner))
		if !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("error = %v, want %v", err, domain.ErrFacilityNotFound)
		}
		if calls := lookup.ensureOwnerCalls.Load(); calls != 1 {
			t.Errorf("lookup called %d times, want 1 — a well-formed id must still be resolved", calls)
		}
		if calls := discounts.createCalls.Load(); calls != 0 {
			t.Errorf("an unknown facility's rule reached the repository (%d calls)", calls)
		}
	})

	for _, id := range malformedFacilityIDs() {
		id := id
		t.Run("malformed/"+id, func(t *testing.T) {
			t.Parallel()
			svc, discounts, lookup := newDiscountSvc(owner)

			_, err := svc.CreateDiscountRule(context.Background(), percentInput(id, owner))
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("CreateDiscountRule(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
			if calls := lookup.ensureOwnerCalls.Load(); calls != 0 {
				t.Errorf("a malformed facility id reached the Facilities lookup (%d calls) — against real Postgres that hits a `uuid` column and panics", calls)
			}
			if calls := discounts.createCalls.Load(); calls != 0 {
				t.Errorf("a malformed facility id reached the repository (%d calls)", calls)
			}
		})
	}
}

// malformedFacilityIDs mirrors the corpus the other contexts' malformed-ID
// tests use. The braced and urn: forms are the ones that matter: they pass
// github.com/google/uuid's Validate but fail pgtype.UUID.Scan.
func malformedFacilityIDs() []string {
	return []string{
		"",
		"not-a-uuid",
		"facility-1",
		"0",
		"'; DROP TABLE discount_rules;--",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
}

// TestCreateDiscountRule_InvalidRuleIsRejected proves each of T11.1's
// constructor sentinels still surfaces through the app layer (they become
// InvalidArgument at the gRPC boundary), and that an invalid rule is never
// persisted. The actor here is always the real owner, so these failures are
// attributable to validation and not to the authorization check above.
func TestCreateDiscountRule_InvalidRuleIsRejected(t *testing.T) {
	t.Parallel()

	owner := userID(1)

	tests := []struct {
		name    string
		mutate  func(*app.CreateDiscountRuleInput)
		wantErr error
	}{
		{
			name:    "unknown discount type",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.DiscountType = "buy_one_get_one" },
			wantErr: domain.ErrInvalidDiscountType,
		},
		{
			name:    "percent above 100",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.Amount = domain.DiscountAmount{Percent: 120} },
			wantErr: domain.ErrInvalidDiscountAmount,
		},
		{
			name:    "percent of zero",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.Amount = domain.DiscountAmount{Percent: 0} },
			wantErr: domain.ErrInvalidDiscountAmount,
		},
		{
			name: "fixed amount of zero",
			mutate: func(in *app.CreateDiscountRuleInput) {
				in.DiscountType = domain.DiscountTypeFixedAmount
				in.Amount = domain.DiscountAmount{Fixed: domain.Money{Cents: 0, Currency: "USD"}}
			},
			wantErr: domain.ErrInvalidDiscountAmount,
		},
		{
			name:    "empty applies_to",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.AppliesTo = nil },
			wantErr: domain.ErrEmptyAppliesTo,
		},
		{
			name:    "applies_to outside the four locked sources",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.AppliesTo = []domain.Source{"birthday_party"} },
			wantErr: domain.ErrInvalidSource,
		},
		{
			name:    "end after zero occurrences",
			mutate:  func(in *app.CreateDiscountRuleInput) { in.EndCondition = domain.EndAfterOccurrences(0) },
			wantErr: domain.ErrInvalidEndConditionOccurrences,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, discounts, _ := newDiscountSvc(owner)

			in := percentInput(facilityID(1), owner)
			tt.mutate(&in)

			if _, err := svc.CreateDiscountRule(context.Background(), in); !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if calls := discounts.createCalls.Load(); calls != 0 {
				t.Errorf("an invalid rule reached the repository (%d calls)", calls)
			}
		})
	}
}

// TestListDiscountRulesForFacility covers the read side, including the
// list-shaped malformed-ID convention this codebase already applies to
// ListCourtBookings: an unresolvable Facility has no rules, which is an empty
// list rather than an error the endpoint never otherwise returns.
func TestListDiscountRulesForFacility(t *testing.T) {
	t.Parallel()

	owner := userID(1)
	svc, discounts, _ := newDiscountSvc(owner)
	ctx := context.Background()

	if _, err := svc.CreateDiscountRule(ctx, percentInput(facilityID(1), owner)); err != nil {
		t.Fatalf("CreateDiscountRule: %v", err)
	}

	got, err := svc.ListDiscountRulesForFacility(ctx, facilityID(1))
	if err != nil {
		t.Fatalf("ListDiscountRulesForFacility: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rules, want 1", len(got))
	}

	before := discounts.listCalls.Load()
	for _, id := range malformedFacilityIDs() {
		rules, err := svc.ListDiscountRulesForFacility(ctx, id)
		if err != nil {
			t.Fatalf("ListDiscountRulesForFacility(%q) error = %v, want an empty list and no error", id, err)
		}
		if len(rules) != 0 {
			t.Fatalf("ListDiscountRulesForFacility(%q) returned %d rules, want 0", id, len(rules))
		}
	}
	if after := discounts.listCalls.Load(); after != before {
		t.Fatalf("a malformed facility id reached the repository (%d extra calls)", after-before)
	}
}
