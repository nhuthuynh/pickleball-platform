package grpcapi_test

import (
	"testing"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// T57.1 (Competitions' half of issue #126) — CompetitionEntry.amount_owed
// survives the full wire -> app -> domain -> wire path.
//
// This file exists because of what its absence allowed, which was measured
// rather than assumed on the Social Play side: deleting the single
// `AmountOwed:` line from that context's toProtoRegistration left
// `make test-domain`, `make test-adapters` and `make test-cmd` ALL GREEN.
// Every Go gate passed while the feature was entirely broken — a client
// would read 0, send 0, and T57.2's validation would then refuse every
// payment it was supposed to permit. Competitions had the identical
// exposure through toProtoEntry and the identical lack of a test.
//
// The per-head arithmetic itself is proven in the domain
// (internal/competitions/domain/amount_owed_test.go). What is proven HERE,
// and only here, is that the number reaches the client at all.

// seedCompetition's fixture charges 2500 AUD per entrant.
const seededEntryFeeCents = 2500

func TestEnterCompetition_AmountOwedRoundTrip(t *testing.T) {
	h, _ := newTestHandler()

	// Capacity 16, guest allowance 3 — room for a party of three.
	competition := seedCompetition(t, h, "host-1", 16, 3)

	// Two guests: three heads at 2500 = 7500. A response carrying 2500
	// would mean the per-head rule never reached the wire; one carrying 0
	// would mean the field did not.
	resp, err := h.EnterCompetition(ctxAs("player-1"), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
		GuestCount:    2,
		Source:        competitionsv1.EntrySource_ENTRY_SOURCE_APP,
	})
	if err != nil {
		t.Fatalf("EnterCompetition: %v", err)
	}

	got := resp.GetEntry().GetAmountOwed()
	if got == nil {
		t.Fatal("amount_owed is absent from the wire — the client cannot pay what it cannot see")
	}
	if want := int64(seededEntryFeeCents * 3); got.GetAmountCents() != want {
		t.Fatalf("amount_owed cents = %d, want %d (%d × 3 heads)", got.GetAmountCents(), want, seededEntryFeeCents)
	}
	if got.GetCurrencyCode() != "AUD" {
		t.Fatalf("amount_owed currency = %q, want AUD — the currency must ride along with the amount (ADR-0005)", got.GetCurrencyCode())
	}
}

// An entrant bringing nobody owes exactly one entry fee. Pinned separately
// because it is the boundary the whole rule turns on: one head, not zero.
func TestEnterCompetition_SoloEntrantOwesOneFee(t *testing.T) {
	h, _ := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 3)

	resp, err := h.EnterCompetition(ctxAs("player-1"), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
		GuestCount:    0,
		Source:        competitionsv1.EntrySource_ENTRY_SOURCE_APP,
	})
	if err != nil {
		t.Fatalf("EnterCompetition: %v", err)
	}

	if got := resp.GetEntry().GetAmountOwed().GetAmountCents(); got != seededEntryFeeCents {
		t.Fatalf("amount_owed cents = %d, want %d — an entrant alone is one head, not zero", got, seededEntryFeeCents)
	}
}

// The roster read carries the figure too, not just the response to the write
// that created it. This is the path CompetitionManage.vue reads to tell a
// Host what an unpaid entry owes in cash — a separate toProtoEntry call site
// reached by a different RPC.
func TestListEntriesForCompetition_CarriesAmountOwed(t *testing.T) {
	h, _ := newTestHandler()

	competition := seedCompetition(t, h, "host-1", 16, 3)

	if _, err := h.EnterCompetition(ctxAs("player-1"), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
		GuestCount:    2,
		Source:        competitionsv1.EntrySource_ENTRY_SOURCE_APP,
	}); err != nil {
		t.Fatalf("EnterCompetition: %v", err)
	}

	resp, err := h.ListEntriesForCompetition(ctxAs("host-1"), &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	if err != nil {
		t.Fatalf("ListEntriesForCompetition: %v", err)
	}
	entries := resp.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if want := int64(seededEntryFeeCents * 3); entries[0].GetAmountOwed().GetAmountCents() != want {
		t.Fatalf("roster amount_owed = %d, want %d — a Host chasing cash must be told what is actually owed",
			entries[0].GetAmountOwed().GetAmountCents(), want)
	}
}
