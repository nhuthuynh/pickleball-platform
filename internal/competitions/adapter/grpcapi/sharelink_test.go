// T9.5 — wire-level tests for the shareable registration link: the
// token-addressed public read, its deliberately uninformative not-found, the
// cancelled-Competition link, and entry-source attribution end to end.
//
// These run through the REAL grpcapi.Handler -> app.Service -> domain path
// with only persistence faked (the fakes and newTestHandler live in
// authz_regression_test.go), because most of what this ticket promises is a
// property of the TRANSLATION layers — what the wire discloses, and what a
// status code and message reveal — which a domain-level test cannot see.
//
// SCOPE NOTE: T9.5 does not build or modify the token generator.
// internal/competitions/adapter/sharetoken (T9.4) already implements
// port.ShareTokenGenerator over crypto/rand at 256 bits; everything here is
// downstream of a token that already exists.
package grpcapi_test

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// seedSharedCompetition creates a Competition through the real handler and
// returns it together with its share token — the token comes off
// CreateCompetitionResponse, which is the only place the contract ever
// discloses it (see the proto's Competition doc comment).
func seedSharedCompetition(t *testing.T, h *grpcapi.Handler, hostID string) (*competitionsv1.Competition, string) {
	t.Helper()
	resp, err := h.CreateCompetition(context.Background(), &competitionsv1.CreateCompetitionRequest{
		HostId:         hostID,
		Name:           "Spring Doubles Open",
		Sessions:       []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "court-1")},
		Capacity:       16,
		GuestAllowance: 2,
		PaymentMethod:  competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		EntryFee:       &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		Format:         competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
	})
	if err != nil {
		t.Fatalf("failed to seed fixture competition: %v", err)
	}
	if resp.GetShareToken() == "" {
		t.Fatal("fixture invalid: CreateCompetition must return a share token (T9.4)")
	}
	return resp.GetCompetition(), resp.GetShareToken()
}

// TestGetCompetitionByShareToken_ShapeMatchesTheIDAddressedRead is the
// ticket's required "no silent scope widening" proof.
//
// A share link is handed to strangers, so the temptation is to make its
// response *helpful* — a host contact, an entrant list, the token itself,
// a "spots left" the ID-addressed read doesn't carry. Any of those would
// mean the link discloses more than the app already shows, which is the
// likelier failure here than the narrowing the BA dossier §5 worried about.
//
// The assertion is therefore structural, not a spot-check of a few fields:
// the two response messages must have the SAME field set (names, numbers,
// kinds, and message types), so adding a field to only one of them fails
// this test. The value-level proto.Equal underneath then proves the two
// paths really render the same Competition, not merely the same shape.
func TestGetCompetitionByShareToken_ShapeMatchesTheIDAddressedRead(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	created, token := seedSharedCompetition(t, h, "host-1")

	byID, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{CompetitionId: created.GetId()})
	if err != nil {
		t.Fatalf("GetCompetition: %v", err)
	}
	byToken, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{ShareToken: token})
	if err != nil {
		t.Fatalf("GetCompetitionByShareToken: %v", err)
	}

	// 1. Same response SHAPE, field for field.
	assertSameFieldSet(t,
		byID.ProtoReflect().Descriptor(),
		byToken.ProtoReflect().Descriptor(),
	)

	// 2. Same VALUE — the identical Competition, not just the same shape.
	if !proto.Equal(byID.GetCompetition(), byToken.GetCompetition()) {
		t.Fatalf("token-addressed read returned a different Competition:\n byID: %v\nbyTok: %v", byID.GetCompetition(), byToken.GetCompetition())
	}

	// 3. The token itself is never echoed back. It is disclosed exactly once,
	//    on CreateCompetitionResponse, to the caller who is provably the Host.
	//    A token-addressed read that returned the token would hand it to
	//    anyone who already has it — harmless — but the field's mere
	//    existence on a public projection is the drift this guards against.
	assertNoFieldMentions(t, byToken.ProtoReflect().Descriptor(), "share_token")
	assertNoFieldMentions(t, byToken.GetCompetition().ProtoReflect().Descriptor(), "share_token")
}

// assertSameFieldSet compares two message descriptors field for field.
func assertSameFieldSet(t *testing.T, want, got protoreflect.MessageDescriptor) {
	t.Helper()

	wf, gf := want.Fields(), got.Fields()
	if wf.Len() != gf.Len() {
		t.Fatalf("%s has %d field(s), %s has %d — the token-addressed response must disclose exactly what the ID-addressed one does",
			want.FullName(), wf.Len(), got.FullName(), gf.Len())
	}
	for i := 0; i < wf.Len(); i++ {
		w, g := wf.Get(i), gf.Get(i)
		if w.Name() != g.Name() || w.Number() != g.Number() || w.Kind() != g.Kind() || w.Cardinality() != g.Cardinality() {
			t.Fatalf("field %d differs: %s{%s #%d %v %v} vs %s{%s #%d %v %v}",
				i, want.FullName(), w.Name(), w.Number(), w.Kind(), w.Cardinality(),
				got.FullName(), g.Name(), g.Number(), g.Kind(), g.Cardinality())
		}
		if w.Kind() == protoreflect.MessageKind && w.Message().FullName() != g.Message().FullName() {
			t.Fatalf("field %s: message type %s vs %s", w.Name(), w.Message().FullName(), g.Message().FullName())
		}
	}
}

// assertNoFieldMentions fails if any field name on the descriptor contains
// the given substring.
func assertNoFieldMentions(t *testing.T, d protoreflect.MessageDescriptor, substr string) {
	t.Helper()
	for i := 0; i < d.Fields().Len(); i++ {
		if name := string(d.Fields().Get(i).Name()); strings.Contains(name, substr) {
			t.Fatalf("%s exposes field %q — a public projection must not carry %q", d.FullName(), name, substr)
		}
	}
}

// TestGetCompetitionByShareToken_NotFoundIsNotAnOracle is the security
// requirement at the layer a probing client actually observes.
//
// This endpoint is unauthenticated and keyed by a secret. If a malformed
// token answered differently from an unknown one — a different code, a
// different message, even a different word order — an enumerator could use
// it to learn which guesses had the right shape, or that a token was real
// but its Competition was gone. Every miss must therefore be BYTE-identical:
// same code, same message, no details.
//
// The unknown-ID GetCompetition case is in the same comparison set on
// purpose: "is this token real" and "is this competition real" must be one
// indistinguishable answer, not two.
func TestGetCompetitionByShareToken_NotFoundIsNotAnOracle(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	created, _ := seedSharedCompetition(t, h, "host-1")

	probes := []struct {
		name string
		call func() error
	}{
		{"well-formed but unknown token", func() error {
			_, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{
				ShareToken: "Zm9vYmFyYmF6cXV4Y29ycmVjdGxlbmd0aHRva2VuMDEy",
			})
			return err
		}},
		{"malformed token (illegal characters)", func() error {
			_, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{
				ShareToken: "not a token!! ***",
			})
			return err
		}},
		{"empty token", func() error {
			_, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{})
			return err
		}},
		{"a competition id used as a token", func() error {
			_, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{
				ShareToken: created.GetId(),
			})
			return err
		}},
		{"unknown competition id (ID-addressed read)", func() error {
			_, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{CompetitionId: fixtureUnknownCompetitionID})
			return err
		}},
	}

	var want *status.Status
	for _, p := range probes {
		err := p.call()
		if err == nil {
			t.Fatalf("%s: succeeded, want NotFound", p.name)
		}
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("%s: err %v is not a gRPC status", p.name, err)
		}
		if st.Code() != codes.NotFound {
			t.Fatalf("%s: code = %v, want NotFound", p.name, st.Code())
		}
		if want == nil {
			want = st
			continue
		}
		if !proto.Equal(st.Proto(), want.Proto()) {
			t.Fatalf("%s: status %v differs from %v — this endpoint is an oracle for which tokens exist",
				p.name, st.Proto(), want.Proto())
		}
	}
}

// TestGetCompetitionByShareToken_CancelledCompetitionStillResolves is the
// ticket's Given/When/Then at the wire, where the 404-vs-cancelled
// distinction is a thing a client can actually see.
//
// GIVEN a Host cancels a Competition after posting its link,
// WHEN a Player follows that still-valid link,
// THEN they get the Competition with status CANCELLED — an honest "this
// competition was cancelled" state — and NOT a NotFound that is
// indistinguishable from a broken or mistyped link. NN/g heuristic #9: a
// dead link and a cancelled event are different facts, and only one of them
// is something the Player can act on.
//
// The final assertion covers the other half of the contract: reading is
// allowed, entering is still rejected — existing T9.3/T9.4 behaviour that
// this test VERIFIES rather than reimplements.
func TestGetCompetitionByShareToken_CancelledCompetitionStillResolves(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	// GIVEN: the link is out in the world, then the Host cancels.
	created, sharedLink := seedSharedCompetition(t, h, "host-1")
	if _, err := h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: created.GetId(),
		ActorUserId:   "host-1",
	}); err != nil {
		t.Fatalf("fixture cancel failed: %v", err)
	}

	// WHEN: a Player follows the link.
	resp, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{ShareToken: sharedLink})

	// THEN: not a 404 — an honest cancelled state.
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			t.Fatalf("a cancelled Competition's link returned NotFound — a Player cannot tell that from a broken link")
		}
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetCompetition().GetStatus(); got != competitionsv1.CompetitionStatus_COMPETITION_STATUS_CANCELLED {
		t.Fatalf("status = %v, want COMPETITION_STATUS_CANCELLED", got)
	}
	if resp.GetCompetition().GetId() != created.GetId() || resp.GetCompetition().GetName() != created.GetName() {
		t.Fatalf("resolved the wrong Competition: %v", resp.GetCompetition())
	}

	// AND: entering it is still rejected (FailedPrecondition — the
	// Competition's own lifecycle, not a capacity conflict).
	_, err = h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: created.GetId(),
		PlayerId:      "player-1",
		Source:        competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL,
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("entering a cancelled Competition: code = %v, want FailedPrecondition", st.Code())
	}
}

// TestEnterCompetition_SourceIsValidatedNotInferred pins the entry-source
// contract end to end on the wire.
//
// The rule is that the CLIENT declares the channel and the SERVER validates
// it against the closed enum — the server never infers it. In particular it
// must not decide "this must be social because a share token was involved":
// how a client reached a Competition is not something the backend can
// observe, and a guess that renders as a fact on a Host's roster is worse
// than no attribution. The unrecognized-value case is what makes "validates"
// more than a word — a future or bogus enum number is rejected 400, not
// quietly stored or silently defaulted.
func TestEnterCompetition_SourceIsValidatedNotInferred(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		source   competitionsv1.EntrySource
		want     competitionsv1.EntrySource
		wantCode codes.Code
	}{
		{
			name:   "declared social (arrived via a share link)",
			source: competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL,
			want:   competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL,
		},
		{
			name:   "declared app",
			source: competitionsv1.EntrySource_ENTRY_SOURCE_APP,
			want:   competitionsv1.EntrySource_ENTRY_SOURCE_APP,
		},
		{
			// Unset is a client that never set the field, which is by
			// definition entering in-app: resolved to APP, never guessed
			// into SOCIAL.
			name:   "unset resolves to app",
			source: competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED,
			want:   competitionsv1.EntrySource_ENTRY_SOURCE_APP,
		},
		{
			// A value this build does not know (a future proto version, or
			// a hand-rolled client). UNSPECIFIED and "unrecognized" are not
			// the same thing and must not share a fallback.
			name:     "unrecognized value is rejected, not defaulted",
			source:   competitionsv1.EntrySource(99),
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			created, token := seedSharedCompetition(t, h, "host-1")

			// The entrant really did arrive through the share link...
			if _, err := h.GetCompetitionByShareToken(ctx, &competitionsv1.GetCompetitionByShareTokenRequest{ShareToken: token}); err != nil {
				t.Fatalf("share-link read failed: %v", err)
			}

			// ...but only their own declaration decides what is stored.
			resp, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
				CompetitionId: created.GetId(),
				PlayerId:      "player-1",
				Source:        tt.source,
			})

			if tt.wantCode != codes.OK {
				st, ok := status.FromError(err)
				if !ok || st.Code() != tt.wantCode {
					t.Fatalf("got err %v, want code %v", err, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got := resp.GetEntry().GetSource(); got != tt.want {
				t.Fatalf("stored source = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestListEntriesForCompetition_ReturnsEachEntrySource proves the roster
// carries the channel each entrant came through, per entry — the read a
// later ticket's Host-facing roster UI needs in order to show "12 entrants,
// 5 from your Facebook post".
//
// Asserting per-entry (rather than that the field merely exists) is what
// catches the plausible bug: a handler that emitted one source for the whole
// list, or defaulted every row to app, would still populate the field.
func TestListEntriesForCompetition_ReturnsEachEntrySource(t *testing.T) {
	ctx := context.Background()
	h, _ := newTestHandler()

	created, _ := seedSharedCompetition(t, h, "host-1")

	entrants := []struct {
		playerID string
		source   competitionsv1.EntrySource
	}{
		{"player-in-app", competitionsv1.EntrySource_ENTRY_SOURCE_APP},
		{"player-from-link", competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL},
		{"player-unset", competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED},
	}
	for _, e := range entrants {
		if _, err := h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
			CompetitionId: created.GetId(),
			PlayerId:      e.playerID,
			Source:        e.source,
		}); err != nil {
			t.Fatalf("entering as %s: %v", e.playerID, err)
		}
	}

	resp, err := h.ListEntriesForCompetition(ctx, &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: created.GetId(),
	})
	if err != nil {
		t.Fatalf("ListEntriesForCompetition: %v", err)
	}
	if len(resp.GetEntries()) != len(entrants) {
		t.Fatalf("roster has %d entries, want %d", len(resp.GetEntries()), len(entrants))
	}

	want := map[string]competitionsv1.EntrySource{
		"player-in-app":    competitionsv1.EntrySource_ENTRY_SOURCE_APP,
		"player-from-link": competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL,
		"player-unset":     competitionsv1.EntrySource_ENTRY_SOURCE_APP,
	}
	for _, got := range resp.GetEntries() {
		if got.GetSource() != want[got.GetPlayerId()] {
			t.Fatalf("entry for %s: source = %v, want %v", got.GetPlayerId(), got.GetSource(), want[got.GetPlayerId()])
		}
	}
}
