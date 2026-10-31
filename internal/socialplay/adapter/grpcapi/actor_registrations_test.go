// T67.2 (part of issue #353) — ListRegistrationsForActor at the gRPC boundary.
//
// Every test here drives the REAL grpcapi.Handler and the REAL app.Service
// (newPrincipalTestHandler, shared with principal_authz_test.go), and seeds its
// world through the REAL CreateGame/RegisterForGame/CancelGame RPCs. There is
// no "pretend this player registered" seam: the only way a Registration exists
// below is a RegisterForGame call under that player's own principal, which is
// what makes the scoping assertions mean anything.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// secondPlayerSubject is a player other than playerSubject, for the
// "one caller never sees another's rows" assertions.
const secondPlayerSubject = "auth0|player-2"

// TestListRegistrationsForActor_ScopesToThePrincipal is the headline boundary
// assertion: two Players join the SAME Game, and each reads back exactly their
// own row. A read that ignored the actor would hand each of them both rows and
// still pass a bare error-is-nil check, so the count and the player id are both
// asserted on each side.
func TestListRegistrationsForActor_ScopesToThePrincipal(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGameWithCapacity(t, h, 4)

	for _, subject := range []string{playerSubject, secondPlayerSubject} {
		if _, err := h.RegisterForGame(ctxAs(subject), &socialplayv1.RegisterForGameRequest{
			GameId: game.GetId(),
		}); err != nil {
			t.Fatalf("RegisterForGame as %q: %v", subject, err)
		}
	}

	for _, subject := range []string{playerSubject, secondPlayerSubject} {
		resp, err := h.ListRegistrationsForActor(ctxAs(subject),
			&socialplayv1.ListRegistrationsForActorRequest{})
		if err != nil {
			t.Fatalf("ListRegistrationsForActor as %q: %v", subject, err)
		}
		got := resp.GetRegistrations()
		if len(got) != 1 {
			t.Fatalf("%q read %d listings, want exactly 1 (their own) — "+
				"both players joined the same Game, so a read that ignored the "+
				"principal would return 2", subject, len(got))
		}
		if want := resolvedUserID(subject); got[0].GetRegistration().GetPlayerId() != want {
			t.Errorf("%q read a listing for player_id %q, want their own %q",
				subject, got[0].GetRegistration().GetPlayerId(), want)
		}
		if got[0].GetRegistration().GetGameId() != game.GetId() {
			t.Errorf("%q: listing game_id = %q, want %q",
				subject, got[0].GetRegistration().GetGameId(), game.GetId())
		}
	}
}

// TestListRegistrationsForActor_NoPrincipalIsUnauthenticated pins ADR-0013 §5's
// distinction. With no verified principal there is no scope to answer within,
// so the honest answer is "I do not know who you are" — not PermissionDenied
// (which would claim the caller is known and refused) and emphatically not an
// empty 200, which a client would render as "you have joined nothing".
func TestListRegistrationsForActor_NoPrincipalIsUnauthenticated(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()

	_, err := h.ListRegistrationsForActor(anonymous(),
		&socialplayv1.ListRegistrationsForActorRequest{})
	requireCode(t, "ListRegistrationsForActor with no principal", err, codes.Unauthenticated)
}

// TestListRegistrationsForActor_RequestCarriesNoForgeableActor is the ticket's
// "a forged actor field changes nothing" requirement, discharged in the
// strongest form available: the request message has NO actor field to forge.
//
// # Why this is a compile-time assertion and not a runtime one
//
// Booking's equivalent read retains a deprecated `actor_user_id` on the wire
// (deleting a shipped proto field is a breaking change), so its test has to
// populate that field with somebody else's id and prove the server ignores it.
// ListRegistrationsForActorRequest is new and never acquired the field, so the
// equivalent proof is that no such field EXISTS — and the way to pin that in Go
// is a struct literal with every field named. If a later ticket adds an
// actor-shaped field to this message, this file stops compiling, which is a
// louder failure than a runtime assertion someone could delete.
//
// The runtime half still runs below: the same empty request, sent by two
// different principals, is answered differently — which is only possible if the
// answer comes from the context rather than from the message.
func TestListRegistrationsForActor_RequestCarriesNoForgeableActor(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGameWithCapacity(t, h, 4)

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}

	// The compile-time half. An unkeyed, fully-specified literal: this
	// compiles only while the message has exactly zero fields.
	req := &socialplayv1.ListRegistrationsForActorRequest{}

	// The runtime half. The attacker sends the byte-identical request the
	// registrant does. They have joined nothing, so they get nothing — the
	// request cannot carry a claim about whose list to return.
	victim, err := h.ListRegistrationsForActor(ctxAs(playerSubject), req)
	if err != nil {
		t.Fatalf("the registrant's own read: %v", err)
	}
	if len(victim.GetRegistrations()) != 1 {
		t.Fatalf("the registrant read %d listings, want 1", len(victim.GetRegistrations()))
	}

	attacker, err := h.ListRegistrationsForActor(ctxAs(attackerSubject), req)
	if err != nil {
		t.Fatalf("the attacker's read should succeed and be empty, got: %v", err)
	}
	if n := len(attacker.GetRegistrations()); n != 0 {
		t.Errorf("the attacker read %d listings from the identical request, want 0 — "+
			"the actor must come from the principal, never from the message", n)
	}
}

// TestListRegistrationsForActor_CancelledGameSurfacesOnTheWire is the whole
// reason the game_* fields are on this message, asserted end-to-end through the
// handler rather than only at the app layer.
//
// ListGames filters `WHERE g.status = 'scheduled'` and this service has NO
// GetGame RPC, so if the cancelled Game did not come back HERE, with
// GAME_STATUS_CANCELLED on it, a Player could not be told their Game was
// called off by any route this API offers.
func TestListRegistrationsForActor_CancelledGameSurfacesOnTheWire(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGameWithCapacity(t, h, 4)

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}
	if _, err := h.CancelGame(ctxAs(hostSubject), &socialplayv1.CancelGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("the Host cancelling the Game: %v", err)
	}

	// Precondition, asserted rather than assumed: the cancelled Game really is
	// invisible to the only other read that could have served this screen.
	games, err := h.ListGames(anonymous(), &socialplayv1.ListGamesRequest{})
	if err != nil {
		t.Fatalf("ListGames: %v", err)
	}
	for _, l := range games.GetGames() {
		if l.GetGame().GetId() == game.GetId() {
			t.Fatalf("ListGames returned the cancelled Game — this test's premise " +
				"(that a cancelled Game is unreachable through ListGames) no longer holds")
		}
	}

	resp, err := h.ListRegistrationsForActor(ctxAs(playerSubject),
		&socialplayv1.ListRegistrationsForActorRequest{})
	if err != nil {
		t.Fatalf("ListRegistrationsForActor after the Host cancelled: %v", err)
	}
	got := resp.GetRegistrations()
	if len(got) != 1 {
		t.Fatalf("the Player read %d listings after their Game was cancelled, want 1 — "+
			"ListGames hides it and there is no GetGame RPC, so dropping it here "+
			"makes the cancellation unreportable", len(got))
	}
	if got[0].GetGameStatus() != socialplayv1.GameStatus_GAME_STATUS_CANCELLED {
		t.Errorf("game_status = %v, want GAME_STATUS_CANCELLED", got[0].GetGameStatus())
	}
}

// TestListRegistrationsForActor_ProjectsTheGameFactsOntoTheWire pins the
// translation itself. A handler that returned the Registration and left every
// game_* field at its zero value would pass every test above, and leave the
// screen unable to say when or where the Game is.
func TestListRegistrationsForActor_ProjectsTheGameFactsOntoTheWire(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGameWithCapacity(t, h, 4)

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}

	resp, err := h.ListRegistrationsForActor(ctxAs(playerSubject),
		&socialplayv1.ListRegistrationsForActorRequest{})
	if err != nil {
		t.Fatalf("ListRegistrationsForActor: %v", err)
	}
	if len(resp.GetRegistrations()) != 1 {
		t.Fatalf("got %d listings, want 1", len(resp.GetRegistrations()))
	}
	l := resp.GetRegistrations()[0]

	// Each field is checked against the Game CreateGame actually returned,
	// not merely for non-emptiness — a projection wired to the wrong column
	// is non-empty too.
	if !l.GetGameStartsAt().AsTime().Equal(game.GetStartsAt().AsTime()) {
		t.Errorf("game_starts_at = %v, want %v",
			l.GetGameStartsAt().AsTime(), game.GetStartsAt().AsTime())
	}
	if !l.GetGameEndsAt().AsTime().Equal(game.GetEndsAt().AsTime()) {
		t.Errorf("game_ends_at = %v, want %v",
			l.GetGameEndsAt().AsTime(), game.GetEndsAt().AsTime())
	}
	if l.GetGameVenueFacilityId() != game.GetVenueFacilityId() {
		t.Errorf("game_venue_facility_id = %q, want %q",
			l.GetGameVenueFacilityId(), game.GetVenueFacilityId())
	}
	if l.GetGameStatus() != socialplayv1.GameStatus_GAME_STATUS_SCHEDULED {
		t.Errorf("game_status = %v, want GAME_STATUS_SCHEDULED", l.GetGameStatus())
	}
	if len(l.GetGameCourtIds()) != len(game.GetCourtIds()) {
		t.Fatalf("game_court_ids = %v, want %v", l.GetGameCourtIds(), game.GetCourtIds())
	}
	for i, id := range game.GetCourtIds() {
		if l.GetGameCourtIds()[i] != id {
			t.Errorf("game_court_ids[%d] = %q, want %q", i, l.GetGameCourtIds()[i], id)
		}
	}

	// The Registration half the screen needs alongside the Game facts.
	reg := l.GetRegistration()
	if reg.GetStatus() != socialplayv1.RegistrationStatus_REGISTRATION_STATUS_REGISTERED {
		t.Errorf("registration.status = %v, want REGISTRATION_STATUS_REGISTERED", reg.GetStatus())
	}
	if reg.GetPaymentStatus() != socialplayv1.PaymentStatus_PAYMENT_STATUS_UNPAID {
		t.Errorf("registration.payment_status = %v, want PAYMENT_STATUS_UNPAID", reg.GetPaymentStatus())
	}
	if reg.GetAmountOwed() == nil {
		t.Error("registration.amount_owed is nil — it is always emitted, " +
			"including for a free Game (see Registration.amount_owed's proto comment)")
	}
}

// TestListRegistrationsForActor_EmptyResultIsAnEmptyList pins the REST contract
// requirement: a Player who has joined nothing gets an empty list and no error,
// so `GET /v1/registrations` answers 200 rather than a 404 or a 500.
func TestListRegistrationsForActor_EmptyResultIsAnEmptyList(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()

	resp, err := h.ListRegistrationsForActor(ctxAs(playerSubject),
		&socialplayv1.ListRegistrationsForActorRequest{})
	if err != nil {
		t.Fatalf("a Player who has joined nothing: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil, want an empty list")
	}
	if n := len(resp.GetRegistrations()); n != 0 {
		t.Errorf("got %d listings, want 0", n)
	}
}

// TestListRegistrationsForActor_PlayerCancelledRegistrationLeavesTheList pins
// the other half of the scope decision (port.RegistrationRepository.
// ListForPlayer's doc comment): the Player's own withdrawal removes the row,
// while a HOST's cancellation keeps it. Both are asserted, because the pair is
// what makes the rule a decision rather than a side effect of one query — and
// because the Registration's own status is 'cancelled' in both cases.
func TestListRegistrationsForActor_PlayerCancelledRegistrationLeavesTheList(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGameWithCapacity(t, h, 4)

	regResp, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}
	if _, err := h.CancelRegistration(ctxAs(playerSubject), &socialplayv1.CancelRegistrationRequest{
		RegistrationId: regResp.GetRegistration().GetId(),
	}); err != nil {
		t.Fatalf("the Player cancelling their own Registration: %v", err)
	}

	resp, err := h.ListRegistrationsForActor(ctxAs(playerSubject),
		&socialplayv1.ListRegistrationsForActorRequest{})
	if err != nil {
		t.Fatalf("ListRegistrationsForActor: %v", err)
	}
	if n := len(resp.GetRegistrations()); n != 0 {
		t.Errorf("got %d listings after the Player withdrew, want 0 — "+
			"they took the action and already know the outcome", n)
	}
}
