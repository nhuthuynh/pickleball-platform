// T9.4 — object-level authorization regression tests for Competitions'
// ownership-gated write endpoint, run through the REAL gRPC handler (not
// just the domain-level unit tests internal/competitions/domain and
// internal/competitions/app already have).
//
// This is the "does the guarantee survive the full stack" proof the T9
// sprint plan requires in place of a separate authz ticket (T9.4's
// non-functional requirements; the same method T7.7 and T8.5 used). The path
// under test is grpcapi.Handler -> app.Service -> domain.Competition, with
// only the persistence boundary faked — every layer that could drop or
// mistranslate the check is real.
//
// SCOPE: CancelCompetition is the only RPC in this contract that acts *on* an
// existing Competition and is therefore the only one with a cross-actor
// rejection to prove. CreateCompetition has no prior owner to be scoped
// against (the caller is the one declaring host_id), and EnterCompetition
// deliberately has NO ownership check — entering is the invitation being
// accepted, not an act on the Competition (see app.Service.EnterCompetition's
// doc comment). That asymmetry is intentional, and
// TestEnterCompetition_IsNotOwnershipGated below pins it down so a future
// reader doesn't "fix" it into a bug.
//
// CAVEAT, stated once and not re-litigated: this proves the OBJECT-LEVEL
// check given a claimed actor_user_id. It is not authentication — there is
// no JWT/session layer yet (HANDOFF.md's open cross-cutting Auth item), so
// actor_user_id is a caller-supplied claim. What is proven here is that a
// mismatched claim is rejected cleanly; what is not proven is that the claim
// is true.
//
// NON-VACUITY: verified per CLAUDE.md rule 10 by temporarily short-circuiting
// domain.Competition.EnsureHost to `return nil`, confirming
// TestCancelCompetition_RejectsNonHostActor FAILED, then restoring it. The
// observed failure output is recorded in the T9.4 PR description — a
// regression test that has never been seen to fail is not yet evidence of
// anything.
//
// Handler-level rather than a -tags=integration testcontainers test, for the
// same two reasons T5.5/T7.7/T8.5 documented: the check under test lives
// entirely in domain.Competition.EnsureHost, which port.Repository has no
// influence over (a real Postgres round trip would add infrastructure, not
// proof), and this environment has no Docker daemon, so the ticket's own
// regression test needs to actually run somewhere we can execute it.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// --- in-memory fakes ------------------------------------------------------
//
// These stand in for internal/competitions/adapter/{postgres,booking,
// facilities,sharetoken} for these tests only. They implement the exact same
// ports the real adapters do, so app.Service and grpcapi.Handler run
// unmodified, real production code.

type fakeRepo struct {
	competitions map[string]domain.Competition
	entries      map[string]domain.CompetitionEntry
	// order fields keep the list reads deterministic (Go map iteration order
	// is unspecified).
	competitionOrder []string
	entryOrder       []string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		competitions: make(map[string]domain.Competition),
		entries:      make(map[string]domain.CompetitionEntry),
	}
}

func (r *fakeRepo) Create(_ context.Context, c domain.Competition) (domain.Competition, error) {
	if _, exists := r.competitions[c.ID]; !exists {
		r.competitionOrder = append(r.competitionOrder, c.ID)
	}
	r.competitions[c.ID] = c
	return c, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (domain.Competition, error) {
	c, ok := r.competitions[id]
	if !ok {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	return c, nil
}

func (r *fakeRepo) UpdateStatus(_ context.Context, id string, s domain.Status) (domain.Competition, error) {
	c, ok := r.competitions[id]
	if !ok {
		return domain.Competition{}, domain.ErrCompetitionNotFound
	}
	c.Status = s
	r.competitions[id] = c
	return c, nil
}

func (r *fakeRepo) CreateEntry(_ context.Context, e domain.CompetitionEntry) (domain.CompetitionEntry, error) {
	if _, exists := r.entries[e.ID]; !exists {
		r.entryOrder = append(r.entryOrder, e.ID)
	}
	r.entries[e.ID] = e
	return e, nil
}

func (r *fakeRepo) ListActiveEntriesForCompetition(_ context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	out := make([]domain.CompetitionEntry, 0)
	for _, id := range r.entryOrder {
		e := r.entries[id]
		if e.CompetitionID == competitionID && e.Status != domain.EntryStatusCancelled {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListEntriesForCompetition(_ context.Context, competitionID string) ([]domain.CompetitionEntry, error) {
	out := make([]domain.CompetitionEntry, 0)
	for _, id := range r.entryOrder {
		if e := r.entries[id]; e.CompetitionID == competitionID {
			out = append(out, e)
		}
	}
	return out, nil
}

// ListCompetitions mirrors the real Postgres adapter's contract: scheduled
// Competitions only, each paired with its WEIGHTED SpotsLeft. It delegates
// that computation to domain.SpotsLeft — the same rule the production SQL
// mirrors — rather than reimplementing it, so this fake cannot silently
// disagree with the invariant under test.
func (r *fakeRepo) ListCompetitions(_ context.Context, filter port.CompetitionListingFilter) ([]port.CompetitionListing, error) {
	out := make([]port.CompetitionListing, 0)
	for _, id := range r.competitionOrder {
		c := r.competitions[id]
		if c.Status != domain.StatusScheduled {
			continue
		}
		if filter.VenueFacilityID != "" && c.VenueFacilityID != filter.VenueFacilityID {
			continue
		}
		var entries []domain.CompetitionEntry
		for _, eid := range r.entryOrder {
			if e := r.entries[eid]; e.CompetitionID == c.ID {
				entries = append(entries, e)
			}
		}
		out = append(out, port.CompetitionListing{
			Competition: c,
			SpotsLeft:   domain.SpotsLeft(c, entries),
		})
	}
	return out, nil
}

// fakeReservation always succeeds — court contention is not what these tests
// are about, and a fake that failed would only obscure the check under test.
type fakeReservation struct{ n int }

func (f *fakeReservation) ReserveCourt(_ context.Context, _ string, _, _ time.Time, _ string) (string, error) {
	f.n++
	return fmt.Sprintf("booking-%d", f.n), nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, _ string) error { return nil }

// fakeFacilities accepts every ID: these tests never set a venue, so the
// lookup is never even called (app.Service skips it for an empty
// VenueFacilityID).
type fakeFacilities struct{}

func (fakeFacilities) FacilityExists(_ context.Context, _ string) error { return nil }

type fakeIDs struct{ n int }

func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("id-%d", f.n)
}

type fakeShareTokens struct{ n int }

func (f *fakeShareTokens) NewShareToken() (string, error) {
	f.n++
	return fmt.Sprintf("token-%d", f.n), nil
}

// newTestHandler wires the REAL app.Service and the REAL grpcapi.Handler —
// exactly what cmd/server wires in production — against the fakes above.
func newTestHandler() (*grpcapi.Handler, *fakeRepo) {
	repo := newFakeRepo()
	svc := app.NewService(app.ServiceOptions{
		Competitions: repo,
		IDs:          &fakeIDs{},
		Reservation:  &fakeReservation{},
		Facilities:   fakeFacilities{},
		ShareTokens:  &fakeShareTokens{},
	})
	return grpcapi.NewHandler(svc), repo
}

func protoSession(start, end string, courtIDs ...string) *competitionsv1.CompetitionSession {
	s, _ := time.Parse(time.RFC3339, start)
	e, _ := time.Parse(time.RFC3339, end)
	return &competitionsv1.CompetitionSession{
		StartsAt: timestamppb.New(s),
		EndsAt:   timestamppb.New(e),
		CourtIds: courtIDs,
	}
}

// seedCompetition creates a Competition through the real handler, so the
// fixture itself exercises the production path rather than being injected
// straight into the fake.
func seedCompetition(t *testing.T, h *grpcapi.Handler, hostID string, capacity, guestAllowance int32) *competitionsv1.Competition {
	t.Helper()
	resp, err := h.CreateCompetition(context.Background(), &competitionsv1.CreateCompetitionRequest{
		HostId:         hostID,
		Name:           "Spring Doubles Open",
		Sessions:       []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
		Capacity:       capacity,
		GuestAllowance: guestAllowance,
		PaymentMethod:  competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		EntryFee:       &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		Format:         competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
	})
	if err != nil {
		t.Fatalf("failed to seed fixture competition: %v", err)
	}
	return resp.GetCompetition()
}

// --- CancelCompetition: object-level (BOLA) regression --------------------

// TestCancelCompetition_RejectsNonHostActor is the ticket's required test:
// create a Competition hosted by host-1, then attempt CancelCompetition as a
// different actor_user_id, through the real handler -> app -> domain path,
// and assert it is rejected as PermissionDenied — not Internal, not a silent
// success — and that NO state changed.
func TestCancelCompetition_RejectsNonHostActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 2)

	// The BOLA attempt: a different actor_user_id tries to cancel host-1's
	// Competition.
	resp, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
		ActorUserId:   "attacker",
	})
	if err == nil {
		t.Fatalf("CancelCompetition(attacker) succeeded silently — a non-Host cancelled host-1's Competition (BOLA regression); response: %v", resp)
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CancelCompetition(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CancelCompetition(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CancelCompetition(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// "Not a silent success": prove the Competition is still scheduled in
	// storage, not merely that an error came back on the wire. A check that
	// only inspected the error would pass against an implementation that
	// returned an error AFTER already writing the cancellation.
	stored, getErr := repo.GetByID(ctx, competition.GetId())
	if getErr != nil {
		t.Fatalf("fixture competition vanished from the repo: %v", getErr)
	}
	if stored.Status != domain.StatusScheduled {
		t.Errorf("competition status = %q after a rejected CancelCompetition, want %q (the attacker's rejected attempt must have no side effect)", stored.Status, domain.StatusScheduled)
	}

	// And prove it through the real read path too, so a client observing the
	// system sees the unchanged state as well.
	getResp, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{CompetitionId: competition.GetId()})
	if err != nil {
		t.Fatalf("GetCompetition after rejected cancel: unexpected error: %v", err)
	}
	if got := getResp.GetCompetition().GetStatus(); got != competitionsv1.CompetitionStatus_COMPETITION_STATUS_SCHEDULED {
		t.Errorf("GetCompetition status = %v after a rejected CancelCompetition, want SCHEDULED", got)
	}
}

// TestCancelCompetition_AllowsHostActor is the symmetric positive case.
// Without it, the rejection test alone couldn't distinguish "the ownership
// check correctly rejects a mismatched actor" from "CancelCompetition is
// broken and rejects everyone" — which is exactly how a vacuous authz test
// passes forever.
func TestCancelCompetition_AllowsHostActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 2)

	resp, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
		ActorUserId:   "host-1",
	})
	if err != nil {
		t.Fatalf("CancelCompetition(host-1) (the real Host) should succeed, got: %v", err)
	}
	if got := resp.GetCompetition().GetStatus(); got != competitionsv1.CompetitionStatus_COMPETITION_STATUS_CANCELLED {
		t.Errorf("response status = %v, want CANCELLED", got)
	}

	stored, err := repo.GetByID(ctx, competition.GetId())
	if err != nil {
		t.Fatalf("GetByID after successful cancel: %v", err)
	}
	if stored.Status != domain.StatusCancelled {
		t.Errorf("stored status = %q, want %q after the Host's successful cancel", stored.Status, domain.StatusCancelled)
	}
}

// TestCancelCompetition_RejectsEmptyActor covers the unauthenticated caller:
// an empty actor_user_id must never satisfy the Host check. Worth its own
// case because "" is the value a client that simply omits the field sends,
// and an EnsureHost implemented as a bare string comparison against a
// Competition with an empty HostID would wrongly accept it.
func TestCancelCompetition_RejectsEmptyActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 2)

	_, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
		ActorUserId:   "",
	})
	if err == nil {
		t.Fatal("CancelCompetition with an empty actor_user_id succeeded — an unidentified caller is never the Host")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Errorf("status code = %v, want PermissionDenied", st.Code())
	}

	stored, _ := repo.GetByID(ctx, competition.GetId())
	if stored.Status != domain.StatusScheduled {
		t.Errorf("competition status = %q, want %q (no side effect)", stored.Status, domain.StatusScheduled)
	}
}

// TestEnterCompetition_IsNotOwnershipGated pins down the deliberate
// asymmetry: entering is NOT ownership-gated, because entering is the
// invitation being accepted rather than an act on the Competition (see
// app.Service.EnterCompetition). A future reader who notices "EnterCompetition
// has no EnsureHost call" and adds one would break the product; this test
// makes that a failing change rather than a silent one.
func TestEnterCompetition_IsNotOwnershipGated(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 2)

	resp, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
		PlayerId:      "some-other-player",
		GuestCount:    0,
		Source:        competitionsv1.EntrySource_ENTRY_SOURCE_APP,
	})
	if err != nil {
		t.Fatalf("EnterCompetition by a non-Host player should succeed (entering is not ownership-gated), got: %v", err)
	}
	if resp.GetEntry().GetPlayerId() != "some-other-player" {
		t.Errorf("entry player_id = %q, want %q", resp.GetEntry().GetPlayerId(), "some-other-player")
	}
}

// --- toStatus mapping: the four codes T9.4 names explicitly ---------------

// TestErrorMapping_NeverInternal walks the domain rejections T9.4 calls out
// by name and asserts each maps to its required gRPC code — and, separately,
// that NONE of them is Internal. The "never Internal" assertion is stated on
// its own rather than being implied by the positive check, because Internal
// is the specific failure this requirement exists to prevent: a 500 tells a
// client nothing actionable and reads as a server bug rather than a
// rejection the caller can fix.
func TestErrorMapping_NeverInternal(t *testing.T) {
	ctx := context.Background()

	t.Run("ErrCompetitionFull -> AlreadyExists", func(t *testing.T) {
		h, _ := newTestHandler()
		// Capacity 2, guest allowance 1: one entry bringing a guest fills it
		// exactly (weight 2), so the next entry of any size cannot fit.
		competition := seedCompetition(t, h, "host-1", 2, 1)

		if _, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
			CompetitionId: competition.GetId(), PlayerId: "p1", GuestCount: 1,
		}); err != nil {
			t.Fatalf("first entry (weight 2, capacity 2) should fit: %v", err)
		}

		_, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
			CompetitionId: competition.GetId(), PlayerId: "p2", GuestCount: 0,
		})
		if err == nil {
			t.Fatal("second entry should have been rejected — the competition is full")
		}
		assertCode(t, err, codes.AlreadyExists, "ErrCompetitionFull")
	})

	t.Run("ErrGuestAllowanceExceeded -> InvalidArgument", func(t *testing.T) {
		h, _ := newTestHandler()
		competition := seedCompetition(t, h, "host-1", 16, 1)

		// 5 guests against an allowance of 1. Note capacity is ample, so
		// this can only be the allowance rejection — not the full one.
		_, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
			CompetitionId: competition.GetId(), PlayerId: "p1", GuestCount: 5,
		})
		if err == nil {
			t.Fatal("entry with 5 guests against an allowance of 1 should be rejected")
		}
		assertCode(t, err, codes.InvalidArgument, "ErrGuestAllowanceExceeded")
	})

	t.Run("ErrNotCompetitionHost -> PermissionDenied", func(t *testing.T) {
		h, _ := newTestHandler()
		competition := seedCompetition(t, h, "host-1", 16, 1)

		_, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
			CompetitionId: competition.GetId(), ActorUserId: "attacker",
		})
		if err == nil {
			t.Fatal("non-Host cancel should be rejected")
		}
		assertCode(t, err, codes.PermissionDenied, "ErrNotCompetitionHost")
	})

	t.Run("ErrCompetitionNotFound -> NotFound", func(t *testing.T) {
		h, _ := newTestHandler()
		_, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{CompetitionId: "no-such-competition"})
		if err == nil {
			t.Fatal("GetCompetition on an unknown id should be rejected")
		}
		assertCode(t, err, codes.NotFound, "ErrCompetitionNotFound")
	})

	t.Run("ErrCompetitionCancelled -> FailedPrecondition", func(t *testing.T) {
		h, _ := newTestHandler()
		competition := seedCompetition(t, h, "host-1", 16, 1)

		if _, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
			CompetitionId: competition.GetId(), ActorUserId: "host-1",
		}); err != nil {
			t.Fatalf("host cancel should succeed: %v", err)
		}

		_, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
			CompetitionId: competition.GetId(), PlayerId: "p1",
		})
		if err == nil {
			t.Fatal("entering a cancelled competition should be rejected")
		}
		assertCode(t, err, codes.FailedPrecondition, "ErrCompetitionCancelled")
	})
}

// ErrFacilityNotFound is the fifth mapping T9.4 names. It needs a
// FacilityLookup that actually rejects, so it gets its own handler rather
// than newTestHandler's always-accepting fake.
func TestErrorMapping_FacilityNotFound(t *testing.T) {
	ctx := context.Background()

	repo := newFakeRepo()
	svc := app.NewService(app.ServiceOptions{
		Competitions: repo,
		IDs:          &fakeIDs{},
		Reservation:  &fakeReservation{},
		Facilities:   rejectingFacilities{},
		ShareTokens:  &fakeShareTokens{},
	})
	h := grpcapi.NewHandler(svc)

	_, err := h.CreateCompetition(ctx, &competitionsv1.CreateCompetitionRequest{
		HostId:          "host-1",
		Name:            "Bad Venue Open",
		VenueFacilityId: "no-such-facility",
		Sessions:        []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
		Capacity:        16,
		PaymentMethod:   competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		Format:          competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
	})
	if err == nil {
		t.Fatal("CreateCompetition with an unknown venue_facility_id should be rejected")
	}
	assertCode(t, err, codes.NotFound, "ErrFacilityNotFound")

	// The venue check must run BEFORE any court is reserved, so a bogus
	// venue can't leave a dangling Booking behind (T8.3's requirement,
	// restated for Competitions where a single call may hold many courts).
	if len(repo.competitions) != 0 {
		t.Errorf("repo holds %d competitions after a rejected CreateCompetition, want 0", len(repo.competitions))
	}
}

type rejectingFacilities struct{}

func (rejectingFacilities) FacilityExists(_ context.Context, _ string) error {
	return domain.ErrFacilityNotFound
}

func assertCode(t *testing.T, err error, want codes.Code, sentinel string) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", sentinel, err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("%s mapped to Internal (500-shaped) — T9.4 requires it never be Internal: %v", sentinel, err)
	}
	if st.Code() != want {
		t.Fatalf("%s status code = %v, want %v", sentinel, st.Code(), want)
	}
}

// --- ListCompetitions: the weighted spots_left boundary -------------------

// TestListCompetitions_SpotsLeftIsWeighted is T9.4's required boundary test
// for the listing's server-computed spots_left, asserted at the WIRE level —
// the value a client actually receives, through the real handler.
//
// domain.TestSpotsLeft already pins the rule itself; this test pins that the
// rule survives the trip to the wire (including the int32 conversion) rather
// than being recomputed, rounded, or replaced by a headcount somewhere in the
// adapter. An unweighted spots_left would be a visible lie the moment anyone
// brings a guest, which is precisely what the numbers below are chosen to
// expose: capacity 8 with one 3-guest entry is 4 free, and a headcount
// implementation would report 7.
func TestListCompetitions_SpotsLeftIsWeighted(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 8, 3)

	// Freshly created: every place free.
	if got := listedSpotsLeft(t, h, competition.GetId()); got != 8 {
		t.Fatalf("spots_left on an empty capacity-8 competition = %d, want 8", got)
	}

	// One entry bringing 3 guests occupies 4 places (1 entrant + 3 guests).
	if _, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(), PlayerId: "p1", GuestCount: 3,
	}); err != nil {
		t.Fatalf("EnterCompetition(p1, 3 guests): %v", err)
	}
	if got := listedSpotsLeft(t, h, competition.GetId()); got != 4 {
		t.Errorf("spots_left after one entry with 3 guests = %d, want 4 — an UNWEIGHTED (headcount) implementation would report 7, advertising places that don't exist", got)
	}

	// A second identical entry fills it exactly: 8 of 8 places occupied.
	if _, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(), PlayerId: "p2", GuestCount: 3,
	}); err != nil {
		t.Fatalf("EnterCompetition(p2, 3 guests): %v", err)
	}
	if got := listedSpotsLeft(t, h, competition.GetId()); got != 0 {
		t.Errorf("spots_left on an exactly-full competition = %d, want 0 — a headcount implementation would report 6", got)
	}

	// The boundary that matters most: spots_left == 0 must agree with Enter
	// actually refusing. A listing that says "full" while entries still
	// succeed (or vice versa) is the drift this whole design exists to
	// prevent.
	_, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(), PlayerId: "p3", GuestCount: 0,
	})
	if err == nil {
		t.Fatal("spots_left reported 0 but a further entry was accepted — the listing and the capacity check disagree")
	}
	assertCode(t, err, codes.AlreadyExists, "ErrCompetitionFull")
}

// listedSpotsLeft reads spots_left for one Competition back through the real
// ListCompetitions RPC.
func listedSpotsLeft(t *testing.T, h *grpcapi.Handler, competitionID string) int32 {
	t.Helper()
	resp, err := h.ListCompetitions(context.Background(), &competitionsv1.ListCompetitionsRequest{})
	if err != nil {
		t.Fatalf("ListCompetitions: %v", err)
	}
	for _, l := range resp.GetCompetitions() {
		if l.GetCompetition().GetId() == competitionID {
			return l.GetSpotsLeft()
		}
	}
	t.Fatalf("competition %s not present in ListCompetitions response", competitionID)
	return 0
}

// TestListCompetitions_ExcludesCancelled pins the other half of the listing
// contract: a cancelled Competition can't be entered, so it must not appear
// in a player-facing browse list.
func TestListCompetitions_ExcludesCancelled(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 8, 1)

	resp, err := h.ListCompetitions(ctx, &competitionsv1.ListCompetitionsRequest{})
	if err != nil {
		t.Fatalf("ListCompetitions: %v", err)
	}
	if len(resp.GetCompetitions()) != 1 {
		t.Fatalf("listing count before cancel = %d, want 1", len(resp.GetCompetitions()))
	}

	if _, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(), ActorUserId: "host-1",
	}); err != nil {
		t.Fatalf("CancelCompetition(host-1): %v", err)
	}

	resp, err = h.ListCompetitions(ctx, &competitionsv1.ListCompetitionsRequest{})
	if err != nil {
		t.Fatalf("ListCompetitions after cancel: %v", err)
	}
	if len(resp.GetCompetitions()) != 0 {
		t.Errorf("listing count after cancel = %d, want 0 (a cancelled Competition isn't enterable, so it has no place in a browse list)", len(resp.GetCompetitions()))
	}
}

// --- share_token containment ----------------------------------------------

// TestShareToken_NotLeakedByReadPaths guards the decision documented on the
// proto's Competition message: the share token is returned ONCE, on
// CreateCompetitionResponse, and must never be reachable through the
// unauthenticated Get/List reads. If it were, "unpublished" would mean
// nothing — anyone who could list Competitions could enter every one of them
// via its share link.
//
// This is a contract test, not just a field check: it would fail loudly if
// someone later added share_token to the Competition message "for
// convenience".
func TestShareToken_NotLeakedByReadPaths(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	createResp, err := h.CreateCompetition(ctx, &competitionsv1.CreateCompetitionRequest{
		HostId:        "host-1",
		Name:          "Private Open",
		Sessions:      []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
		Capacity:      8,
		PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_SINGLES,
	})
	if err != nil {
		t.Fatalf("CreateCompetition: %v", err)
	}

	// The Host does receive it, exactly once — otherwise they could never
	// publish the link at all.
	token := createResp.GetShareToken()
	if token == "" {
		t.Fatal("CreateCompetitionResponse.share_token is empty — the Host has no way to obtain their own share link")
	}

	// And it really was persisted, so this isn't passing because tokens are
	// silently absent everywhere.
	stored, err := repo.GetByID(ctx, createResp.GetCompetition().GetId())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.ShareToken != token {
		t.Fatalf("stored share token = %q, want %q", stored.ShareToken, token)
	}

	// The read paths must not carry it. Marshalling to text and searching
	// for the token value catches the leak regardless of which field it
	// might have been added to.
	getResp, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{CompetitionId: stored.ID})
	if err != nil {
		t.Fatalf("GetCompetition: %v", err)
	}
	if containsToken(getResp.String(), token) {
		t.Errorf("GetCompetition response leaks the share token: %s", getResp.String())
	}

	listResp, err := h.ListCompetitions(ctx, &competitionsv1.ListCompetitionsRequest{})
	if err != nil {
		t.Fatalf("ListCompetitions: %v", err)
	}
	if containsToken(listResp.String(), token) {
		t.Errorf("ListCompetitions response leaks the share token: %s", listResp.String())
	}
}

func containsToken(haystack, token string) bool {
	return token != "" && len(haystack) >= len(token) && stringsContains(haystack, token)
}

// stringsContains is a tiny local helper so this file's intent stays obvious
// at the call site (the check is "does the serialized response contain the
// secret", not a generic substring test).
func stringsContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
