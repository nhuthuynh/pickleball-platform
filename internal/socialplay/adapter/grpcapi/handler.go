// Package grpcapi is the Social Play context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/socialplay/v1
// and the app/domain layers, and maps domain errors onto gRPC status codes
// so grpc-gateway's REST mapping produces the right HTTP status — mirrors
// internal/booking/adapter/grpcapi's shape. It only compiles after `make
// generate` has produced internal/gen/pickleball/socialplay/v1 (see
// CLAUDE.md gotchas).
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// Handler serves SocialPlayService. It holds the CourtReservation and
// FacilityLookup ports alongside the app.Service because ScheduleGame takes
// both per-call (T5.3's original design for CourtReservation — see
// app.Service.ScheduleGame's doc comment; T8.3 gives FacilityLookup the
// same treatment) rather than storing them on Service itself.
// actor resolves the acting identity for an authenticated RPC (T12.8).
//
// This one function is the whole of A11 Ruling 3 for this context: the
// Principal is translated into the plain actor string app.Service and
// domain.Game.EnsureHost/domain.Registration.Cancel already take, here at the
// grpcapi boundary, so internal/socialplay/{domain,app} keep their existing
// signatures and never import internal/platform/auth.
//
// It takes only a context, deliberately. The request's actor_player_id /
// actor_user_id / player_id / host_id fields are not passed in and cannot be
// consulted, so there is no fallback to the caller's claim when no principal
// is present — the failure mode the ticket calls out as "a handler that falls
// back to the claimed value has changed nothing". Missing principal is
// codes.Unauthenticated ("I do not know who you are"), never PermissionDenied
// (ADR-0013 §5).
//
// # One subject, two field names
//
// Social Play is the context where actor_player_id and actor_user_id meet:
// CancelGame and CancelRegistration name the actor a *player*, while
// RecordMatchResult names it a *user*, and RecordMatchResult's actor and
// CancelGame's actor are both compared against the same stored Game.HostID.
// They were never two identifier spaces — see this ticket's PR body and the
// tracked issue for the finding — so both resolve to the same verified subject
// here rather than through an invented mapping.
func actor(ctx context.Context) (string, error) {
	return auth.RequireSubject(ctx)
}

type Handler struct {
	socialplayv1.UnimplementedSocialPlayServiceServer
	svc         *app.Service
	reservation port.CourtReservation
	facilities  port.FacilityLookup
}

func NewHandler(svc *app.Service, reservation port.CourtReservation, facilities port.FacilityLookup) *Handler {
	return &Handler{svc: svc, reservation: reservation, facilities: facilities}
}

// CreateGame schedules a Game hosted by the verified caller (T12.8).
//
// host_id is MINTED from the principal, not accepted from the wire. This is
// the same treatment T12.7 gave CreateFacility's owner_id, for the same
// reason: this RPC *writes* the Game.HostID that CancelGame and
// RecordMatchResult later compare a verified subject against. Had it kept
// taking that value from the wire, every new Game would be hosted by an
// arbitrary client-supplied string — so either real Hosts are locked out of
// their own Games, or a caller hands Host-ship to a subject that is not
// theirs.
func (h *Handler) CreateGame(ctx context.Context, req *socialplayv1.CreateGameRequest) (*socialplayv1.CreateGameResponse, error) {
	hostID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	rng, err := domain.NewTimeRange(req.GetStartsAt().AsTime(), req.GetEndsAt().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	g, err := h.svc.ScheduleGame(ctx, app.ScheduleGameInput{
		HostID:          hostID,
		FacilityID:      req.GetFacilityId(),
		VenueFacilityID: req.GetVenueFacilityId(),
		CourtIDs:        req.GetCourtIds(),
		Range:           rng,
		Capacity:        int(req.GetCapacity()),
		PaymentMethod:   fromProtoPaymentMethod(req.GetPaymentMethod()),
		GuestAllowance:  int(req.GetGuestAllowance()),
		EntryFee:        fromProtoMoney(req.GetEntryFee()),
	}, h.reservation, h.facilities)
	if err != nil {
		return nil, toStatus(err)
	}

	return &socialplayv1.CreateGameResponse{Game: toProtoGame(g)}, nil
}

// RegisterForGame registers the verified caller for a Game (T12.8). player_id
// comes from the principal, never the wire: it is the value
// domain.Registration.Cancel compares an actor against, so a Registration
// minted for a claimed player_id could never be cancelled by the human it
// names.
func (h *Handler) RegisterForGame(ctx context.Context, req *socialplayv1.RegisterForGameRequest) (*socialplayv1.RegisterForGameResponse, error) {
	playerID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	reg, err := h.svc.RegisterForGame(ctx, app.RegisterForGameInput{
		GameID:     req.GetGameId(),
		PlayerID:   playerID,
		GuestCount: int(req.GetGuestCount()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.RegisterForGameResponse{Registration: toProtoRegistration(reg)}, nil
}

// CancelRegistration cancels the verified caller's own Registration (T12.8).
// The object-level check still lives in domain.Registration.Cancel; what
// changed here is that the actor it compares against is verified rather than
// claimed.
func (h *Handler) CancelRegistration(ctx context.Context, req *socialplayv1.CancelRegistrationRequest) (*socialplayv1.CancelRegistrationResponse, error) {
	actorPlayerID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	reg, err := h.svc.CancelRegistration(ctx, req.GetRegistrationId(), actorPlayerID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.CancelRegistrationResponse{Registration: toProtoRegistration(reg)}, nil
}

// CancelGame cancels a Game on its Host's behalf (T12.4). The object-level
// authorization check lives in domain.Game.EnsureHost, reached via
// app.Service.CancelGame; this handler only translates the wire request and
// maps domain errors through toStatus (non-Host -> PermissionDenied,
// unknown/malformed game_id -> NotFound, already-cancelled ->
// FailedPrecondition).
//
// T12.8: the acting player is now the verified principal's subject, not
// req.GetActorPlayerId(). CancelGame is named explicitly in that ticket
// because it shipped one wave earlier with a claimed actor, which makes it the
// RPC most likely to be missed — and the most costly to miss, since cancelling
// is destructive, irreversible, and deliberately not delegated to Game Admins.
func (h *Handler) CancelGame(ctx context.Context, req *socialplayv1.CancelGameRequest) (*socialplayv1.CancelGameResponse, error) {
	actorPlayerID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	game, err := h.svc.CancelGame(ctx, req.GetGameId(), actorPlayerID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.CancelGameResponse{Game: toProtoGame(game)}, nil
}

// JoinWaitlist puts the verified caller on a Game's waitlist (T12.8).
// player_id comes from the principal for the reason RegisterForGame's does,
// one step further out: a WaitlistEntry's player becomes a Registration's
// player on promotion, so a claimed value here propagates into the aggregate
// CancelRegistration guards.
func (h *Handler) JoinWaitlist(ctx context.Context, req *socialplayv1.JoinWaitlistRequest) (*socialplayv1.JoinWaitlistResponse, error) {
	playerID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	entry, err := h.svc.JoinWaitlist(ctx, app.JoinWaitlistInput{
		GameID:   req.GetGameId(),
		PlayerID: playerID,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.JoinWaitlistResponse{Entry: toProtoWaitlistEntry(entry)}, nil
}

// ListGames serves the Discover & Join Games browse/filter read (T8.9).
// starts_after/starts_before are only converted to a real time.Time when
// present on the wire (protobuf's Timestamp field presence, via
// GetStartsAfter()/GetStartsBefore() returning nil for an unset field) —
// otherwise the zero value flows through, which
// port.GameListingFilter/nullableTimestamptz (internal/socialplay/adapter/
// postgres) already treat as "no bound on this side". No error mapping
// beyond toStatus's default is needed: this is a pure read with no
// domain-error-producing invariant to violate.
func (h *Handler) ListGames(ctx context.Context, req *socialplayv1.ListGamesRequest) (*socialplayv1.ListGamesResponse, error) {
	filter := port.GameListingFilter{VenueFacilityID: req.GetVenueFacilityId()}
	if req.GetStartsAfter() != nil {
		filter.StartsAfter = req.GetStartsAfter().AsTime()
	}
	if req.GetStartsBefore() != nil {
		filter.StartsBefore = req.GetStartsBefore().AsTime()
	}

	listings, err := h.svc.ListGames(ctx, filter)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*socialplayv1.GameListing, 0, len(listings))
	for _, l := range listings {
		out = append(out, toProtoGameListing(l))
	}
	return &socialplayv1.ListGamesResponse{Games: out}, nil
}

// ListRegistrationsForGame serves the Host pending-cash-payments dashboard's
// read (T8.10 — see ListRegistrationsForGameRequest's proto doc comment for
// why this RPC exists).
//
// T13.6 (partial fix for #147): this is no longer a public read. It returns
// per-person data — every registrant's player_id, payment_status and
// guest_count — and it used to hand that to any caller who knew a game_id,
// which the public ListGames response supplies. The actor is the verified
// principal, so a missing one is codes.Unauthenticated here and a non-Host
// one becomes domain.ErrNotGameHost -> codes.PermissionDenied in toStatus,
// never Internal. The RPC moved from PublicMethods() to
// AuthenticatedMethods() in authenticated.go to match.
func (h *Handler) ListRegistrationsForGame(ctx context.Context, req *socialplayv1.ListRegistrationsForGameRequest) (*socialplayv1.ListRegistrationsForGameResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	regs, err := h.svc.ListRegistrationsForGame(ctx, req.GetGameId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*socialplayv1.Registration, 0, len(regs))
	for _, r := range regs {
		out = append(out, toProtoRegistration(r))
	}
	return &socialplayv1.ListRegistrationsForGameResponse{Registrations: out}, nil
}

// RecordMatchResult records a Match result against an existing Game (T10.4),
// authorized to the Game's Host or an assigned Game Admin only. score comes
// off the wire as map[string]int32 (proto3's only integer map value type);
// domain.RecordMatch takes map[string]int, so this converts key-for-key —
// see toProtoMatch's inverse comment for the return direction.
// T12.8: ActorUserID is the verified principal's subject rather than
// req.GetActorUserId(). assigned_game_admin_user_ids remains a caller-supplied
// fact — this codebase has never persisted Game-Admin assignments, and
// building that store is a domain change A11 Ruling 3 puts out of this
// ticket's scope. Disclosed in the PR body with a tracked issue, per A5.
func (h *Handler) RecordMatchResult(ctx context.Context, req *socialplayv1.RecordMatchResultRequest) (*socialplayv1.RecordMatchResultResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	m, err := h.svc.RecordMatchResult(ctx, app.RecordMatchResultInput{
		GameID:                   req.GetGameId(),
		Players:                  req.GetPlayers(),
		Score:                    fromProtoScore(req.GetScore()),
		ActorUserID:              actorUserID,
		AssignedGameAdminUserIDs: req.GetAssignedGameAdminUserIds(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.RecordMatchResultResponse{Match: toProtoMatch(m)}, nil
}

// ListMatchesForGame lists every Match recorded against a Game (T10.4).
// Unlike ListRegistrationsForGame, an unknown game_id maps to NotFound here
// — see app.Service.ListMatchesForGame's doc comment for why the two RPCs'
// error-handling requirements differ despite the similar shape.
func (h *Handler) ListMatchesForGame(ctx context.Context, req *socialplayv1.ListMatchesForGameRequest) (*socialplayv1.ListMatchesForGameResponse, error) {
	matches, err := h.svc.ListMatchesForGame(ctx, req.GetGameId())
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*socialplayv1.Match, 0, len(matches))
	for _, m := range matches {
		out = append(out, toProtoMatch(m))
	}
	return &socialplayv1.ListMatchesForGameResponse{Matches: out}, nil
}

// AssignGameAdmin records that the verified caller — who must be this Game's
// Host — granted Game-Admin authority over it to another user (T14.4, partial
// fix for #168).
//
// The authorization rule lives in domain.AssignGameAdmin (via
// Game.EnsureHost), reached through app.Service.AssignGameAdmin; this handler
// only translates the request and maps domain errors through toStatus. Note
// what that means for the property #168 cares about most: an assigned Game
// Admin calling this RPC is refused by the same EnsureHost check a stranger
// hits, because an admin is never the Host (domain.ErrHostCannotBeGameAdmin
// keeps the two roles disjoint). There is no separate "reject admins" branch
// here that could be dropped.
//
// The actor is the verified principal, never the wire — the request has no
// actor field at all. A missing principal is codes.Unauthenticated ("I do not
// know who you are"), never PermissionDenied (ADR-0013 §5), which is what
// actor(ctx) returns on its own.
func (h *Handler) AssignGameAdmin(ctx context.Context, req *socialplayv1.AssignGameAdminRequest) (*socialplayv1.AssignGameAdminResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	admin, err := h.svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
		GameID:      req.GetGameId(),
		ActorUserID: actorUserID,
		AdminUserID: req.GetUserId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.AssignGameAdminResponse{GameAdmin: toProtoGameAdmin(admin)}, nil
}

// RevokeGameAdmin withdraws a Game-Admin assignment on the Host's behalf
// (T14.4). Host-only for the same reason AssignGameAdmin is — see
// domain.EnsureMayRevokeGameAdmin — and with the same verified-principal
// treatment of the actor.
//
// Revoking a user who holds no assignment maps to codes.NotFound
// (domain.ErrGameAdminNotFound), which is deliberately a different answer from
// the codes.NotFound an unknown game_id produces only in the sense that a
// caller who is not the Host never reaches either: the authorization check
// runs before the store is consulted, so neither answer can be used to
// enumerate a Game's admins.
func (h *Handler) RevokeGameAdmin(ctx context.Context, req *socialplayv1.RevokeGameAdminRequest) (*socialplayv1.RevokeGameAdminResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.svc.RevokeGameAdmin(ctx, app.RevokeGameAdminInput{
		GameID:      req.GetGameId(),
		ActorUserID: actorUserID,
		AdminUserID: req.GetUserId(),
	}); err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.RevokeGameAdminResponse{}, nil
}

// toStatus maps domain errors to gRPC status codes; grpc-gateway then maps
// those onto HTTP statuses (AlreadyExists -> 409, InvalidArgument -> 400,
// NotFound -> 404, PermissionDenied -> 403) — mirrors
// internal/booking/adapter/grpcapi.toStatus. ErrGameFull, ErrAlreadyRegistered,
// and ErrCourtUnavailable all map to the same conflict status
// (AlreadyExists/409) that ErrCourtDoubleBooked already uses for Booking,
// per the ticket's smoke-test AC. ErrNotRegistrationOwner maps to
// PermissionDenied/403 — the BOLA-shaped rejection T5.2/T5.5 require ("a
// clear rejection, not a 500").
//
// T6.6 additions: ErrAlreadyOnWaitlist and ErrGameNotFull join the
// AlreadyExists/InvalidArgument groups respectively (mirroring
// ErrAlreadyRegistered's and the validation-error group's own reasoning —
// "not full" is a precondition violation on the request, not a conflict).
// ErrNotWaitlistEntryOwner joins ErrNotRegistrationOwner's PermissionDenied
// group. ErrWaitlistEntryNotFound joins the NotFound group.
// ErrWaitlistPromotionNotExpired and ErrNoWaitingEntries are deliberately
// NOT mapped here: neither is reachable from a client-facing RPC in this
// ticket (JoinWaitlist is the only new RPC; promotion/expiry are
// app-layer-internal, triggered by CancelRegistration and a future
// sweep/admin path, not exposed directly) — falling through to Internal for
// either would be a signal something upstream called an app method it
// shouldn't have, not a case this handler needs to translate for a client.
//
// T8.3 addition: ErrFacilityNotFound joins the NotFound group — an unknown
// CreateGameRequest.venue_facility_id is a 404, not a 500 or a silent
// accept (the ticket's explicit requirement).
//
// T9.2 addition: ErrInvalidMoney joins the same InvalidArgument group — a
// negative entry fee, or a non-zero fee with a missing/malformed currency
// code, is a malformed request, not a server fault.
//
// T8.7 additions: ErrInvalidPaymentMethod, ErrInvalidGuestAllowance, and
// ErrGuestAllowanceExceeded join the validation-error/InvalidArgument
// group — all three are precondition violations on the request itself
// (a bad enum value, a negative allowance, or a guest count outside the
// Game's own allowance), the same category ErrInvalidCapacity/
// ErrEmptyCourtIDs already occupy.
//
// T10.4 additions (RecordMatchResult/ListMatchesForGame — gRPC codes only
// per this ticket's own error-handling instructions, no HTTP status
// restated alongside them, per the T9.4 Ceremony-1-adopted rule):
// ErrNotGameHostOrAdmin joins the PermissionDenied group (the object-level
// check, same shape as ErrNotRegistrationOwner above); ErrGameCancelled is
// its own new FailedPrecondition case — "recording a match against a
// cancelled Game," distinct from every existing NotFound/InvalidArgument
// group, since the Game *is* found and the request *is* well-formed, it's
// the Game's own state that makes the request currently illegal (mirrors
// the standard NotFound/FailedPrecondition/InvalidArgument distinction:
// the resource exists, but is in the wrong state for this operation);
// ErrEmptyScore joins the InvalidArgument group alongside
// ErrEmptyPlayers/ErrTooFewPlayers (domain.RecordMatch's own field
// validation — ErrEmptyPlayers/ErrTooFewPlayers were already unreachable-
// but-harmless additions from T10.3 since no RPC called domain.RecordMatch
// before this ticket; now they're live).
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrGameFull),
		errors.Is(err, domain.ErrAlreadyRegistered),
		errors.Is(err, domain.ErrCourtUnavailable),
		errors.Is(err, domain.ErrAlreadyOnWaitlist),
		// T14.4 (#168): a user already holding a Game-Admin assignment for
		// this Game joins the same conflict group ErrAlreadyRegistered and
		// ErrAlreadyOnWaitlist occupy — the request names a state that
		// already exists, which is what AlreadyExists is for.
		errors.Is(err, domain.ErrAlreadyGameAdmin):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotRegistrationOwner),
		errors.Is(err, domain.ErrNotWaitlistEntryOwner),
		errors.Is(err, domain.ErrNotGameHostOrAdmin),
		// ErrNotGameHost (T12.4, CancelGame) joins the same
		// PermissionDenied group as ErrNotGameHostOrAdmin but stays a
		// distinct sentinel: the two gate different permitted-actor sets
		// (Host-only vs Host-or-Game-Admin). A shared status code is not a
		// reason to share a sentinel — see domain/errors.go.
		errors.Is(err, domain.ErrNotGameHost):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrGameNotFound),
		errors.Is(err, domain.ErrRegistrationNotFound),
		errors.Is(err, domain.ErrWaitlistEntryNotFound),
		errors.Is(err, domain.ErrFacilityNotFound),
		// T14.4 (#168): revoking an assignment that does not exist. Joins the
		// NotFound group rather than becoming a silent success — see
		// port.GameAdminRepository.Revoke's doc comment for why answering
		// "done" to a revoke that removed nothing is the wrong answer. It sits
		// beside ErrGameNotFound rather than replacing it: the Game is real,
		// the assignment is not, the same distinction ErrRegistrationNotFound
		// already draws against its own parent Game.
		errors.Is(err, domain.ErrGameAdminNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrGameCancelled),
		// T14.7 (closes #158): ErrIllegalStatusTransition joins
		// ErrGameCancelled here rather than staying in the InvalidArgument
		// group below. The two are the same concept reached from different
		// guards — "this aggregate's state forbids the operation" — and a
		// client cancelling something twice got FailedPrecondition from
		// CancelGame (which checks EnsureNotCancelled first) but
		// InvalidArgument from CancelRegistration (which falls through to
		// Registration.Cancel's own guard). That is #131's
		// one-concept-two-codes defect, split across two RPCs.
		//
		// gRPC defines INVALID_ARGUMENT as a problem with the argument
		// *regardless of the state of the system*; every raise site of this
		// sentinel (Game.Cancel, Registration.Cancel,
		// WaitlistEntry.Promote/Expire/Cancel) is a `Status != required`
		// guard, so it is state-dependent by construction — precisely what
		// FAILED_PRECONDITION means and what INVALID_ARGUMENT denies. Matches
		// payments' own ErrIllegalStatusTransition since T13.9.
		//
		// Client impact is nil over REST: grpc-gateway maps both
		// InvalidArgument and FailedPrecondition to HTTP 400.
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrEmptyCourtIDs),
		// T14.8 (issue #156): a malformed court id joins the empty-list
		// case it sat next to in the request. Both are the client's
		// mistake; only one of them used to say so.
		errors.Is(err, domain.ErrMalformedCourtID),
		errors.Is(err, domain.ErrEmptyPlayerID),
		errors.Is(err, domain.ErrGameNotFull),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidGuestAllowance),
		errors.Is(err, domain.ErrGuestAllowanceExceeded),
		errors.Is(err, domain.ErrInvalidMoney),
		errors.Is(err, domain.ErrEmptyPlayers),
		errors.Is(err, domain.ErrTooFewPlayers),
		errors.Is(err, domain.ErrEmptyScore),
		// T14.4 (#168) — both join the InvalidArgument group, and the second
		// one is a judgement call worth stating rather than assuming.
		//
		// ErrEmptyGameAdminUserID is the textbook case: a blank user id names
		// nobody regardless of system state.
		//
		// ErrHostCannotBeGameAdmin is state-dependent in the literal sense —
		// the same user_id is perfectly valid against a Game they do not host
		// — so T14.7's ErrIllegalStatusTransition argument might seem to point
		// at FailedPrecondition. It does not, and the distinction this context
		// already draws is ErrGuestAllowanceExceeded's: a limit that is fixed,
		// knowable and *caller-visible* (a Game's host_id is right there on the
		// Game message the caller read to get the game_id) makes a violating
		// request a client-input defect, not a system-state condition the
		// client could not have anticipated. The state-condition sentinels
		// FailedPrecondition covers — ErrGameCancelled,
		// ErrIllegalStatusTransition — are about a *lifecycle* that changes
		// under the caller's feet; a Game's Host does not.
		errors.Is(err, domain.ErrEmptyGameAdminUserID),
		errors.Is(err, domain.ErrHostCannotBeGameAdmin):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoGameStatus(s domain.Status) socialplayv1.GameStatus {
	switch s {
	case domain.StatusScheduled:
		return socialplayv1.GameStatus_GAME_STATUS_SCHEDULED
	case domain.StatusCancelled:
		return socialplayv1.GameStatus_GAME_STATUS_CANCELLED
	default:
		return socialplayv1.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

func toProtoRegistrationStatus(s domain.RegistrationStatus) socialplayv1.RegistrationStatus {
	switch s {
	case domain.RegistrationStatusRegistered:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_REGISTERED
	case domain.RegistrationStatusCancelled:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_CANCELLED
	default:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_UNSPECIFIED
	}
}

func toProtoPaymentStatus(s domain.PaymentStatus) socialplayv1.PaymentStatus {
	switch s {
	case domain.PaymentStatusUnpaid:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_UNPAID
	case domain.PaymentStatusPaid:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_PAID
	case domain.PaymentStatusRefunded:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// toProtoPaymentMethod converts a domain.PaymentMethod to its wire enum
// (T8.7) — mirrors toProtoGameStatus/toProtoRegistrationStatus's shape. Every
// domain.PaymentMethod value is one of its own closed enum's non-zero
// values, so there is no "unspecified" case to fall into on this direction
// (unlike fromProtoPaymentMethod below, which does have to handle the wire
// zero value).
func toProtoPaymentMethod(m domain.PaymentMethod) socialplayv1.PaymentMethod {
	switch m {
	case domain.PaymentMethodOnline:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_ONLINE
	case domain.PaymentMethodCash:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_CASH
	case domain.PaymentMethodEither:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER
	default:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

// fromProtoPaymentMethod converts a wire PaymentMethod into a
// domain.PaymentMethod (T8.7). PAYMENT_METHOD_UNSPECIFIED — the wire zero
// value, sent by any client that never sets this field, including every
// pre-T8.7 client — resolves to domain.PaymentMethodEither, the least
// restrictive value: see db/migrations/0012_socialplay_guest_capacity.sql's
// payment_method column default for the identical "closest thing to
// unspecified" reasoning applied at the DB layer. This keeps CreateGame
// backward compatible: a request that predates this field behaves exactly
// as it did before T8.6/T8.7 introduced PaymentMethod at all. Any OTHER,
// unrecognized wire value (not one of the enum's named non-zero constants —
// e.g. a value from a future proto version this build doesn't know about)
// is deliberately NOT folded into the same "unspecified" default: it's
// passed through as an invalid domain.PaymentMethod so domain.NewGame's own
// IsValid() check rejects it via ErrInvalidPaymentMethod, rather than
// silently accepting an unrecognized value as if it meant "either" (a
// PE-review finding on T8.7's original PR — UNSPECIFIED and "unrecognized"
// are not the same thing and must not share a fallback).
func fromProtoPaymentMethod(m socialplayv1.PaymentMethod) domain.PaymentMethod {
	switch m {
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED:
		return domain.PaymentMethodEither
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_ONLINE:
		return domain.PaymentMethodOnline
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return domain.PaymentMethodCash
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER:
		return domain.PaymentMethodEither
	default:
		return domain.PaymentMethod("")
	}
}

func toProtoGame(g domain.Game) *socialplayv1.Game {
	return &socialplayv1.Game{
		Id:              g.ID,
		HostId:          g.HostID,
		FacilityId:      g.FacilityID,
		VenueFacilityId: g.VenueFacilityID,
		CourtIds:        g.CourtIDs,
		StartsAt:        timestamppb.New(g.Range.Start),
		EndsAt:          timestamppb.New(g.Range.End),
		Capacity:        int32(g.Capacity),
		Status:          toProtoGameStatus(g.Status),
		PaymentMethod:   toProtoPaymentMethod(g.PaymentMethod),
		GuestAllowance:  int32(g.GuestAllowance),
		EntryFee:        toProtoMoney(g.EntryFee),
	}
}

// toProtoMoney/fromProtoMoney translate domain.Money across the wire (T9.2).
//
// fromProtoMoney maps an ABSENT entry_fee message (a nil *Money — an older
// client, or a field simply left unset) onto the zero domain.Money, which
// means a free Game. That is deliberately not an "unspecified" resolution
// step of the kind fromProtoPaymentMethod performs: zero is a real,
// well-formed value here, so there is nothing to resolve. It is also the
// same value db/migrations/0013_socialplay_entry_fee.sql backfilled onto
// every pre-existing row, so an unaware client and a pre-T9.2 row agree.
//
// toProtoMoney always emits the message, including for a free Game, so a
// client can distinguish "this Game is free" (amount_cents 0) from a field
// the server never populated — clients render the former as the word
// "Free", never as blank or a bare "$0.00" (T9.2 non-functional
// requirement).
func toProtoMoney(m domain.Money) *socialplayv1.Money {
	return &socialplayv1.Money{
		AmountCents:  m.Cents,
		CurrencyCode: m.Currency,
	}
}

func fromProtoMoney(m *socialplayv1.Money) domain.Money {
	if m == nil {
		return domain.Money{}
	}
	return domain.Money{
		Cents:    m.GetAmountCents(),
		Currency: m.GetCurrencyCode(),
	}
}

// toProtoGameListing converts a port.GameListing to its wire message (T8.9)
// — see GameListing's proto doc comment for why SpotsLeft is a sibling
// field on this wrapper rather than a Game field.
func toProtoGameListing(l port.GameListing) *socialplayv1.GameListing {
	return &socialplayv1.GameListing{
		Game:      toProtoGame(l.Game),
		SpotsLeft: int32(l.SpotsLeft),
	}
}

func toProtoRegistration(r domain.Registration) *socialplayv1.Registration {
	return &socialplayv1.Registration{
		Id:            r.ID,
		GameId:        r.GameID,
		PlayerId:      r.PlayerID,
		Status:        toProtoRegistrationStatus(r.Status),
		PaymentStatus: toProtoPaymentStatus(r.PaymentStatus),
		GuestCount:    int32(r.GuestCount),
	}
}

func toProtoWaitlistStatus(s domain.WaitlistStatus) socialplayv1.WaitlistStatus {
	switch s {
	case domain.WaitlistStatusWaiting:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_WAITING
	case domain.WaitlistStatusPromoted:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_PROMOTED
	case domain.WaitlistStatusExpired:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_EXPIRED
	case domain.WaitlistStatusCancelled:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_CANCELLED
	default:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_UNSPECIFIED
	}
}

func toProtoWaitlistEntry(e domain.WaitlistEntry) *socialplayv1.WaitlistEntry {
	out := &socialplayv1.WaitlistEntry{
		Id:       e.ID,
		GameId:   e.GameID,
		PlayerId: e.PlayerID,
		Position: int32(e.Position),
		Status:   toProtoWaitlistStatus(e.Status),
	}
	if !e.PromotedAt.IsZero() {
		out.PromotedAt = timestamppb.New(e.PromotedAt)
	}
	return out
}

// toProtoMatch converts a domain.Match to its wire message (T10.4).
// domain.Match.Score is map[string]int; the wire message uses
// map[string]int32 (proto3's only integer map value type) — see
// fromProtoScore's doc comment for the inbound direction of this same
// conversion.
func toProtoMatch(m domain.Match) *socialplayv1.Match {
	return &socialplayv1.Match{
		Id:         m.ID,
		GameId:     m.GameID,
		Players:    m.Players,
		Score:      toProtoScore(m.Score),
		RecordedAt: timestamppb.New(m.RecordedAt),
	}
}

// toProtoScore/fromProtoScore translate domain.Match.Score (map[string]int)
// across the wire (map[string]int32, T10.4) — a nil/empty input produces a
// nil/empty output on both directions, rather than allocating an empty map
// for a Score that domain.RecordMatch's own ErrEmptyScore check (T10.4)
// already guarantees is never legitimately empty on a successfully recorded
// Match.
func toProtoScore(score map[string]int) map[string]int32 {
	if len(score) == 0 {
		return nil
	}
	out := make(map[string]int32, len(score))
	for k, v := range score {
		out[k] = int32(v)
	}
	return out
}

// toProtoGameAdmin converts a domain.GameAdmin to its wire message (T14.4).
//
// One-directional on purpose: there is no fromProtoGameAdmin, because nothing
// off the wire ever becomes a GameAdmin. AssignGameAdminRequest carries a
// game_id and a user_id and nothing else — the assigner and the timestamp are
// server facts, minted from the verified principal and the server clock. A
// client-supplied GameAdmin message would be exactly the caller-asserted
// admin fact #168 exists to abolish.
func toProtoGameAdmin(a domain.GameAdmin) *socialplayv1.GameAdmin {
	return &socialplayv1.GameAdmin{
		GameId:     a.GameID,
		UserId:     a.UserID,
		AssignedBy: a.AssignedBy,
		AssignedAt: timestamppb.New(a.AssignedAt),
	}
}

func fromProtoScore(score map[string]int32) map[string]int {
	if len(score) == 0 {
		return nil
	}
	out := make(map[string]int, len(score))
	for k, v := range score {
		out[k] = int(v)
	}
	return out
}
