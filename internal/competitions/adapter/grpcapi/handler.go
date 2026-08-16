// Package grpcapi is the Competitions context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/competitions/v1
// and the app/domain layers, and maps domain errors onto gRPC status codes
// so grpc-gateway's REST mapping produces the right HTTP status — mirrors
// internal/socialplay/adapter/grpcapi's shape. It only compiles after `make
// generate` has produced internal/gen/pickleball/competitions/v1 (see
// CLAUDE.md gotchas).
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// Handler serves CompetitionsService.
//
// It holds only the app.Service — unlike socialplay's Handler, which also
// carries CourtReservation and FacilityLookup because ScheduleGame takes
// them per-call. Competitions' app.Service stores every dependency on itself
// (see app.ServiceOptions), so there is nothing for this adapter to thread
// through, and the handler stays a pure translation layer.
type Handler struct {
	competitionsv1.UnimplementedCompetitionsServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// actor resolves the acting user for an authenticated RPC (T12.8), and since
// T29.1 it is also ADR-0014/ADR-0017's translation seam for this context —
// mirroring internal/booking/adapter/grpcapi/handler.go's,
// internal/facilities/adapter/grpcapi/handler.go's, and
// internal/payments/adapter/grpcapi/handler.go's identical funnels
// (T13.2/T13.3/T28.1, the last the closest and most recent reference this
// ticket followed).
//
// This one method is the whole of A11 Ruling 3 for this context: the
// Principal is translated into the plain actor string app.Service and
// domain.Competition's Host check already take, here at the grpcapi boundary,
// so internal/competitions/{domain,app} keep their existing signatures and
// never import internal/platform/auth.
//
// It takes only a context, deliberately. The request's actor_user_id /
// host_id / player_id fields are not passed in and cannot be consulted, so
// there is no fallback to the caller's claim when no principal is present —
// the failure mode the ticket calls out as "a handler that falls back to the
// claimed value has changed nothing". Missing principal is
// codes.Unauthenticated ("I do not know who you are"), never PermissionDenied
// (ADR-0013 §5).
//
// **It returns a User.ID (uuid), not the subject, as of T29.1.** That is
// ADR-0014/ADR-0017's ruling and the fix for #164's Competitions third — and,
// as a side effect, for Competitions' half of #237 (internal/payments/app.
// authorizeCompetitionEntryRecording's resolved-actor-vs-still-subject-shaped
// read regression T28.1 introduced: once this method's return value and
// competitions.host_id/competition_entries.player_id/competition_admins.
// user_id all hold the same identifier space, port.EntryLookup/
// port.CompetitionAdminReader's reads on the Payments side are correct again
// with no change under internal/payments/**). Two steps, in this order,
// identical to Booking's/Facilities'/Payments' funnel:
//
//  1. auth.RequireSubject — who is calling? A missing or unverified principal
//     is codes.Unauthenticated, never PermissionDenied (ADR-0013 §5).
//  2. app.Service.ResolveActorUserID — which User is that? A subject
//     registered to no User is codes.PermissionDenied, never NotFound
//     (ADR-0014 §6).
//
// **This is why the funnel change and the backfill migration
// (db/migrations/0025_competitions_identity_conformance.sql) land in the
// same PR, not two.** Every actor-taking RPC on this service calls this one
// method, including CancelCompetition and ListEntriesForCompetition, whose
// domain.Competition.EnsureHost/EnsureHostOrCompetitionAdmin compare the
// value this method returns against the *stored* c.HostID / the resolved
// admin set. The instant this method starts returning a uuid instead of a
// subject, those comparisons are only correct if the stored side is also a
// uuid — see domain.Competition.EnsureHost's/EnsureHostOrCompetitionAdmin's
// own doc comments for the full hazard analysis. Landing this method's
// change without the migration (or vice versa) would have created exactly
// the window that hazard describes; landing them together is what closes
// it.
//
// It is a method rather than a package function, as of this ticket, only
// because it needs the service to reach port.IdentityLookup (via
// Service.ResolveActorUserID) — NewHandler's own signature is unchanged.
func (h *Handler) actor(ctx context.Context) (string, error) {
	subject, err := auth.RequireSubject(ctx)
	if err != nil {
		return "", err
	}

	actorUserID, err := h.svc.ResolveActorUserID(ctx, subject)
	if err != nil {
		return "", toStatus(err)
	}
	return actorUserID, nil
}

// CreateCompetition schedules a Competition hosted by the verified caller
// (T12.8).
//
// host_id is MINTED from the principal, not accepted from the wire — the same
// treatment T12.7 gave CreateFacility's owner_id and for the same reason: this
// RPC *writes* the Competition.HostID that CancelCompetition later compares a
// verified subject against. It matters twice over here, because this response
// also carries the share token on the reasoning that it is "the one moment the
// caller is provably the Host" — a claim worth nothing if the Host were
// whoever the request named.
func (h *Handler) CreateCompetition(ctx context.Context, req *competitionsv1.CreateCompetitionRequest) (*competitionsv1.CreateCompetitionResponse, error) {
	hostID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	sessions, err := fromProtoSessions(req.GetSessions())
	if err != nil {
		// A malformed session range is a bad request, not a server fault.
		// Converted here rather than in the app layer because an invalid
		// wire value never becomes a domain type in the first place.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	competition, err := h.svc.ScheduleCompetition(ctx, app.ScheduleCompetitionInput{
		HostID:          hostID,
		Name:            req.GetName(),
		VenueFacilityID: req.GetVenueFacilityId(),
		Sessions:        sessions,
		Capacity:        int(req.GetCapacity()),
		GuestAllowance:  int(req.GetGuestAllowance()),
		PaymentMethod:   fromProtoPaymentMethod(req.GetPaymentMethod()),
		EntryFee:        fromProtoMoney(req.GetEntryFee()),
		Format:          fromProtoFormat(req.GetFormat()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	// share_token is returned on this response only — the one moment the
	// caller is provably the Host. It is deliberately absent from the
	// Competition message so it can't leak through the unauthenticated
	// Get/List reads; see the proto's own doc comments.
	return &competitionsv1.CreateCompetitionResponse{
		Competition: toProtoCompetition(competition),
		ShareToken:  competition.ShareToken,
	}, nil
}

func (h *Handler) GetCompetition(ctx context.Context, req *competitionsv1.GetCompetitionRequest) (*competitionsv1.GetCompetitionResponse, error) {
	competition, err := h.svc.GetCompetition(ctx, req.GetCompetitionId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.GetCompetitionResponse{Competition: toProtoCompetition(competition)}, nil
}

// GetCompetitionByShareToken serves the public, unauthenticated read behind
// a Competition's shareable registration link (T9.5).
//
// Three properties of this handler are load-bearing, and all three are
// regression-tested in sharelink_test.go:
//
//  1. It returns the SAME projection GetCompetition does, through the SAME
//     toProtoCompetition, into a response message that is field-for-field
//     identical to GetCompetitionResponse. Rendering it via a second
//     conversion function is how a link quietly starts disclosing more than
//     the app does, so there is deliberately only one.
//  2. It adds NO validation of the token's shape. A malformed token must
//     take the identical path an unknown one takes and produce the identical
//     NotFound — an InvalidArgument for "that isn't even a token" would tell
//     an enumerator which guesses were the right shape. app.Service returns
//     the bare domain.ErrCompetitionNotFound sentinel for every miss and
//     toStatus maps it to NotFound with that sentinel's own message, so all
//     misses are byte-identical.
//  3. It does NOT filter cancelled Competitions. A cancelled Competition
//     resolves and reports COMPETITION_STATUS_CANCELLED, because a link
//     already published outlives the Competition's scheduled state and a
//     Player deserves an honest cancelled state rather than a 404 that looks
//     like a broken link.
//
// The token generator itself is untouched by this ticket — T9.4's
// internal/competitions/adapter/sharetoken (crypto/rand, 256 bits) already
// mints one for every Competition at creation.
func (h *Handler) GetCompetitionByShareToken(ctx context.Context, req *competitionsv1.GetCompetitionByShareTokenRequest) (*competitionsv1.GetCompetitionByShareTokenResponse, error) {
	competition, err := h.svc.GetCompetitionByShareToken(ctx, req.GetShareToken())
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.GetCompetitionByShareTokenResponse{Competition: toProtoCompetition(competition)}, nil
}

// ListCompetitions serves the browse/filter read. starts_after/starts_before
// are only converted to a real time.Time when actually present on the wire
// (protobuf Timestamp field presence — GetStartsAfter() returns nil for an
// unset field); otherwise the zero value flows through, which
// port.CompetitionListingFilter and the adapter's nullableTimestamptz
// already treat as "no bound on this side".
func (h *Handler) ListCompetitions(ctx context.Context, req *competitionsv1.ListCompetitionsRequest) (*competitionsv1.ListCompetitionsResponse, error) {
	filter := port.CompetitionListingFilter{VenueFacilityID: req.GetVenueFacilityId()}
	if req.GetStartsAfter() != nil {
		filter.StartsAfter = req.GetStartsAfter().AsTime()
	}
	if req.GetStartsBefore() != nil {
		filter.StartsBefore = req.GetStartsBefore().AsTime()
	}

	listings, err := h.svc.ListCompetitions(ctx, filter)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*competitionsv1.CompetitionListing, 0, len(listings))
	for _, l := range listings {
		out = append(out, toProtoCompetitionListing(l))
	}
	return &competitionsv1.ListCompetitionsResponse{Competitions: out}, nil
}

// EnterCompetition enters the verified caller (and optionally their guests)
// into a Competition (T12.8).
//
// player_id comes from the principal, never the wire. Note this adds no
// ownership check to entering — any authenticated Player may still enter a
// published Competition, which is the deliberate asymmetry
// EnterCompetitionRequest's proto comment describes. What changed is only that
// the entrant is who they say they are, which matters beyond this context:
// Payments' competition-entry authorization compares its actor against exactly
// this stored value, so an entry created for a claimed player_id would decide
// who may pay for it.
func (h *Handler) EnterCompetition(ctx context.Context, req *competitionsv1.EnterCompetitionRequest) (*competitionsv1.EnterCompetitionResponse, error) {
	playerID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	entry, err := h.svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: req.GetCompetitionId(),
		PlayerID:      playerID,
		GuestCount:    int(req.GetGuestCount()),
		Source:        fromProtoEntrySource(req.GetSource()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.EnterCompetitionResponse{Entry: toProtoEntry(entry)}, nil
}

// CancelCompetition cancels a Competition on its Host's behalf (T12.8: the
// acting user is the verified principal, not req.GetActorUserId()). The
// Host-only object-level check is unchanged; what changed is that the identity
// it compares against is verified rather than claimed.
func (h *Handler) CancelCompetition(ctx context.Context, req *competitionsv1.CancelCompetitionRequest) (*competitionsv1.CancelCompetitionResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	competition, err := h.svc.CancelCompetition(ctx, req.GetCompetitionId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.CancelCompetitionResponse{Competition: toProtoCompetition(competition)}, nil
}

// ListEntriesForCompetition is the Host-or-Competition-Admin-facing roster
// read.
//
// T13.6 (partial fix for #147): no longer a public read. It returns every
// entrant's player_id and payment status, and it used to hand that to any
// caller who knew a competition_id — a value the public ListCompetitions and
// GetCompetition responses supply. The actor is the verified principal, so a
// missing one is codes.Unauthenticated here. T15.4 (closes #147) widened the
// entitled set from Host-only to Host-or-assigned-Competition-Admin, now
// that T15.3's store makes "assigned Competition Admin" a server fact; a
// caller who is neither becomes domain.ErrNotCompetitionHostOrAdmin ->
// codes.PermissionDenied in toStatus, never Internal. The RPC moved from
// PublicMethods() to AuthenticatedMethods() in authenticated.go to match.
// Exact twin of Social Play's ListRegistrationsForGame (T14.5).
func (h *Handler) ListEntriesForCompetition(ctx context.Context, req *competitionsv1.ListEntriesForCompetitionRequest) (*competitionsv1.ListEntriesForCompetitionResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	entries, err := h.svc.ListEntriesForCompetition(ctx, req.GetCompetitionId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*competitionsv1.CompetitionEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toProtoEntry(e))
	}
	return &competitionsv1.ListEntriesForCompetitionResponse{Entries: out}, nil
}

// AssignCompetitionAdmin records that the verified caller — who must be this
// Competition's Host — granted Competition-Admin authority over it to another
// user (T15.3, partial fix for #168).
//
// The authorization rule lives in domain.AssignCompetitionAdmin (via
// Competition.EnsureHost), reached through app.Service.AssignCompetitionAdmin;
// this handler only translates the request and maps domain errors through
// toStatus. Note what that means for the property #168 cares about most: an
// assigned Competition Admin calling this RPC is refused by the same
// EnsureHost check a stranger hits, because an admin is never the Host
// (domain.ErrHostCannotBeCompetitionAdmin keeps the two roles disjoint). There
// is no separate "reject admins" branch here that could be dropped.
//
// The actor is the verified principal, never the wire — the request has no
// actor field at all. A missing principal is codes.Unauthenticated ("I do not
// know who you are"), never PermissionDenied (ADR-0013 §5), which is what
// h.actor(ctx) returns on its own.
func (h *Handler) AssignCompetitionAdmin(ctx context.Context, req *competitionsv1.AssignCompetitionAdminRequest) (*competitionsv1.AssignCompetitionAdminResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	adminUserID, err := h.resolveTargetUserID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	admin, err := h.svc.AssignCompetitionAdmin(ctx, app.AssignCompetitionAdminInput{
		CompetitionID: req.GetCompetitionId(),
		ActorUserID:   actorUserID,
		AdminUserID:   adminUserID,
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.AssignCompetitionAdminResponse{CompetitionAdmin: toProtoCompetitionAdmin(admin)}, nil
}

// resolveTargetUserID is T29.1's second identity-resolution seam in this
// handler, distinct from h.actor above: AssignCompetitionAdminRequest.
// user_id / RevokeCompetitionAdminRequest.user_id name a user OTHER than the
// caller — the proto's own doc comment on AssignCompetitionAdminRequest.
// user_id says so explicitly ("the subject to grant authority to — a
// subject, not a uuid"). Before this ticket that subject was written
// straight into competition_admins.user_id/assigned_by unchanged (both
// `text`); as of this ticket that column is `uuid`, so the same
// subject-to-User.ID translation h.actor performs for the CALLER must also
// run for the person being NAMED, through the identical
// h.svc.ResolveActorUserID seam — it is a plain subject->User.ID translator
// regardless of whose subject it is given.
//
// This is why domain.CompetitionAdmin's UserID/AssignedBy comparisons
// (domain.AssignCompetitionAdmin's `adminUserID == c.HostID` check,
// domain.HasCompetitionAdmin's membership test against a later h.actor(ctx)
// value) are correct with NO logic change: both sides of every comparison
// are resolved User.IDs by the time they reach the domain, exactly the
// property this ticket's funnel change establishes everywhere else. Doing
// this resolution HERE, at the grpcapi boundary, rather than inside
// app.Service.AssignCompetitionAdmin/RevokeCompetitionAdmin, keeps those two
// methods receiving an already-resolved value on every parameter — the same
// convention every other app-layer method in this package (and Booking's/
// Facilities'/Payments' identical methods) already follows, so this
// package's existing app-layer and domain-layer tests, which construct
// AdminUserID as an already-resolved literal directly, need no change.
//
// An EMPTY subject is passed through UNCHANGED, never resolved: a blank
// AdminUserID is a distinct, request-shape defect
// (domain.ErrEmptyCompetitionAdminUserID -> InvalidArgument, T15.3's own
// rule), not an unregistered-user question (domain.ErrUserNotFound ->
// PermissionDenied). Resolving "" here would answer PermissionDenied to
// what is actually a plain bad request, and would also never reach
// domain.AssignCompetitionAdmin's own blank check, which exists precisely
// to give the bad-request answer for this case.
//
// A non-empty subject that resolves to no User answers PermissionDenied,
// mirroring h.actor's identical ADR-0014 §6 reasoning applied to the named
// user rather than the caller: NotFound would make this RPC an oracle for
// "does this subject belong to a registered user", the same enumeration
// hazard §6 rules against for the caller's own subject.
func (h *Handler) resolveTargetUserID(ctx context.Context, subject string) (string, error) {
	if subject == "" {
		return "", nil
	}
	userID, err := h.svc.ResolveActorUserID(ctx, subject)
	if err != nil {
		return "", toStatus(err)
	}
	return userID, nil
}

// RevokeCompetitionAdmin withdraws a Competition-Admin assignment on the
// Host's behalf (T15.3). Host-only for the same reason
// AssignCompetitionAdmin is — see domain.EnsureMayRevokeCompetitionAdmin —
// and with the same verified-principal treatment of the actor.
//
// Revoking a user who holds no assignment maps to codes.NotFound
// (domain.ErrCompetitionAdminNotFound). A caller who is not the Host never
// reaches that answer, nor the one an unknown competition_id produces: the
// authorization check runs before the store is consulted, so neither can be
// used to enumerate a Competition's admins.
func (h *Handler) RevokeCompetitionAdmin(ctx context.Context, req *competitionsv1.RevokeCompetitionAdminRequest) (*competitionsv1.RevokeCompetitionAdminResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	adminUserID, err := h.resolveTargetUserID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	if err := h.svc.RevokeCompetitionAdmin(ctx, app.RevokeCompetitionAdminInput{
		CompetitionID: req.GetCompetitionId(),
		ActorUserID:   actorUserID,
		AdminUserID:   adminUserID,
	}); err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.RevokeCompetitionAdminResponse{}, nil
}

// toProtoCompetitionAdmin converts a domain.CompetitionAdmin to its wire
// message (T15.3).
//
// One-directional on purpose: there is no fromProtoCompetitionAdmin, because
// nothing off the wire ever becomes a CompetitionAdmin.
// AssignCompetitionAdminRequest carries a competition_id and a user_id and
// nothing else — the assigner and the timestamp are server facts, minted from
// the verified principal and the server clock. A client-supplied
// CompetitionAdmin message would be exactly the caller-asserted admin fact
// #168 exists to abolish.
func toProtoCompetitionAdmin(a domain.CompetitionAdmin) *competitionsv1.CompetitionAdmin {
	return &competitionsv1.CompetitionAdmin{
		CompetitionId: a.CompetitionID,
		UserId:        a.UserID,
		AssignedBy:    a.AssignedBy,
		AssignedAt:    timestamppb.New(a.AssignedAt),
	}
}

// toStatus maps domain errors onto gRPC status codes; grpc-gateway then maps
// those onto HTTP statuses. Mirrors internal/socialplay/adapter/grpcapi's
// toStatus, with the four mappings T9.4 requires by name — none of which may
// ever fall through to Internal:
//
//   - ErrCompetitionFull -> AlreadyExists (409-shaped): matches Social
//     Play's ErrGameFull precedent exactly (same "capacity conflict" shape,
//     same status). T9.4's ticket text named FailedPrecondition but also
//     described the mapping as "409-shaped" — those are inconsistent,
//     since grpc-gateway's default HTTPStatusFromCode maps FailedPrecondition
//     to 400, not 409 (confirmed empirically against the real running
//     gateway during PR review: a full Competition returned HTTP 400 with
//     code 9 under the FailedPrecondition mapping this comment used to
//     specify). PE+QA review (PR #87) resolved the inconsistency in favor
//     of the HTTP shape and the cross-context precedent, for three reasons:
//     (1) a full Competition is a well-formed request that would have
//     succeeded moments earlier, which is what 409 Conflict — not 400 Bad
//     Request — means; (2) FailedPrecondition collapsed onto the same 400
//     as ErrGuestAllowanceExceeded, defeating the very distinction this
//     handler's own design wants a client to be able to make (retryable
//     capacity race vs. non-retryable malformed request); (3) this codebase
//     already made this exact tradeoff twice (ErrGameFull, ErrCourtUnavailable
//     below) — a third context choosing differently is the inconsistency,
//     not the fix.
//   - ErrFacilityNotFound -> NotFound (404-shaped): an unknown
//     venue_facility_id is a bad reference, never a 500 and never a silent
//     accept.
//   - ErrNotCompetitionHost -> PermissionDenied (403-shaped): the
//     object-level (BOLA) rejection. Regression-tested end to end in
//     authz_regression_test.go, which asserts specifically that this is NOT
//     Internal and NOT a silent success.
//   - ErrGuestAllowanceExceeded -> InvalidArgument (400-shaped): asking for
//     more guests than the Host permits is a malformed request against a
//     fixed, knowable limit, not a conflict over contended capacity. Keeping
//     it distinct from ErrCompetitionFull is what lets a client tell "you
//     may bring at most 2 guests" apart from "this competition is full" —
//     the distinction domain.Enter deliberately checks in that order.
//
// The remaining groups follow the same reasoning the other contexts use:
// not-found sentinels to NotFound, request-shape violations to
// InvalidArgument, and ErrCourtUnavailable/ErrAlreadyEntered to
// AlreadyExists (matching Booking's own ErrCourtDoubleBooked, since those
// ARE genuine "already taken"/"conflict" cases — ErrCompetitionFull now
// joins this group too, see above).
//
// ErrCompetitionCancelled stays on FailedPrecondition alone: "this
// Competition is not in a state that accepts entries" because it was
// deliberately cancelled is a precondition failure on the resource's own
// lifecycle, not a capacity conflict — it does not share ErrCompetitionFull's
// "would have succeeded moments earlier" shape, so it does not follow it
// into the AlreadyExists group.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrCompetitionCancelled),
		// T14.7 (closes #158): ErrIllegalStatusTransition joins
		// ErrCompetitionCancelled here rather than staying in the
		// InvalidArgument group below. The two are the same concept reached
		// from different guards — "this Competition's lifecycle state forbids
		// the operation" — and a client got FailedPrecondition for entering a
		// cancelled Competition but InvalidArgument for re-cancelling one.
		// That is #131's one-concept-two-codes defect, and this switch had
		// already conceded the point in its very first case.
		//
		// gRPC defines INVALID_ARGUMENT as a problem with the argument
		// *regardless of the state of the system*; Competition.Cancel's guard
		// is `if c.Status != StatusScheduled`, which is state-dependent by
		// construction — precisely what FAILED_PRECONDITION means and what
		// INVALID_ARGUMENT denies. Matches payments' own
		// ErrIllegalStatusTransition since T13.9.
		//
		// Client impact is nil over REST: grpc-gateway maps both
		// InvalidArgument and FailedPrecondition to HTTP 400.
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrNotCompetitionHost),
		// T15.4 (closes #147): ListEntriesForCompetition's widened
		// Host-or-Competition-Admin rejection. Same PermissionDenied
		// mapping as its stricter Host-only sibling above — see
		// domain.ErrNotCompetitionHostOrAdmin's DO-NOT-UNIFY note for why
		// the two stay distinct sentinels despite sharing this arm.
		errors.Is(err, domain.ErrNotCompetitionHostOrAdmin):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		// T29.1 (closes the Competitions third of #164): h.actor resolves a
		// subject to no User. PermissionDenied, not NotFound (ADR-0014 §6,
		// restated by ADR-0017) — the caller's token verified, so this is
		// not Unauthenticated either; they are simply not registered, and
		// answering NotFound would turn every actor-taking RPC into a
		// user-enumeration oracle. Mirrors booking/facilities/payments'
		// identical mapping for their own ErrUserNotFound.
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrCompetitionNotFound),
		// ErrFacilityNotFound is now produced by TWO raise sites (T17.3,
		// part of issue #195): app.Service.ScheduleCompetition's own
		// FacilityExists guard, and — since this ticket —
		// adapter/postgres.translateErr's 23503 arm for the narrow
		// concurrent-delete race between that guard and the INSERT.
		// Verified genuinely, not assumed covered: both raise sites produce
		// the SAME sentinel VALUE, so this one case arm maps both without
		// change. ErrCompetitionNotFound carries the identical two-raise-site
		// shape for competition_entries.competition_id / EnterCompetition's
		// GetByID guard.
		errors.Is(err, domain.ErrFacilityNotFound),
		// T15.6 (closes #185): ScheduleCompetition naming a court that does
		// not exist. Joins the not-found group — the same "this request
		// references something that isn't there" concept — and pointedly NOT
		// ErrCourtUnavailable's AlreadyExists group below: a court that never
		// existed is not a court that is busy. Before this ticket the
		// underlying FK violation reached this switch with no sentinel at
		// all and answered Internal. Twin of the arm in
		// internal/socialplay/adapter/grpcapi.toStatus.
		errors.Is(err, domain.ErrCourtNotFound),
		// T15.3: revoking a user who holds no assignment is NotFound, not a
		// silent success — see port.CompetitionAdminRepository.Revoke's doc
		// comment for why answering "done" to a revoke that removed nothing
		// is the wrong answer to a Host asserting who holds authority. A
		// caller who is not the Host never reaches this answer: the
		// authorization check runs before the store is consulted, so it
		// cannot be used to enumerate a Competition's admins.
		errors.Is(err, domain.ErrCompetitionAdminNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyEntered),
		errors.Is(err, domain.ErrCourtUnavailable),
		// T15.3: a second assignment of the same user to the same
		// Competition, from either the domain pre-check or
		// competition_admins' composite primary key — the two are
		// indistinguishable by design (CLAUDE.md rules 4 and 5). Joins the
		// conflict group for the same reason ErrAlreadyEntered is here.
		errors.Is(err, domain.ErrAlreadyCompetitionAdmin),
		errors.Is(err, domain.ErrCompetitionFull):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrGuestAllowanceExceeded),
		errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrEmptySessions),
		errors.Is(err, domain.ErrEmptyCourtIDs),
		// T14.8 (issue #156) — see the twin arm in
		// internal/socialplay/adapter/grpcapi.toStatus.
		errors.Is(err, domain.ErrMalformedCourtID),
		errors.Is(err, domain.ErrEmptyPlayerID),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidFormat),
		errors.Is(err, domain.ErrInvalidGuestAllowance),
		errors.Is(err, domain.ErrInvalidEntrySource),
		errors.Is(err, domain.ErrInvalidMoney),
		// T15.3's two input rules. ErrEmptyCompetitionAdminUserID is the
		// textbook InvalidArgument case: a blank user id names nobody,
		// regardless of the state of the system. ErrHostCannotBeCompetitionAdmin
		// is state-dependent in the literal sense — it compares the argument
		// against the Competition's host_id — but it stays here rather than
		// joining the FailedPrecondition group above, because the state it
		// reads is the *identity of the resource being addressed*, not a
		// lifecycle the caller could wait out: no sequence of other calls
		// makes assigning the Host to their own Competition valid later.
		// Matches Social Play's identical placement of
		// ErrHostCannotBeGameAdmin (T14.4).
		errors.Is(err, domain.ErrEmptyCompetitionAdminUserID),
		errors.Is(err, domain.ErrHostCannotBeCompetitionAdmin),
		errors.Is(err, domain.ErrOverlappingSessions):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// --- wire -> domain --------------------------------------------------------

// fromProtoSessions converts the wire's session list into domain.Sessions,
// validating each range through domain.NewTimeRange so a zero-duration or
// inverted range is rejected at the boundary with a clean InvalidArgument
// rather than reaching the domain as a malformed value.
//
// An empty list is passed through rather than rejected here, so that
// domain.NewCompetition's own ErrEmptySessions is the single place that rule
// lives (and the single error message clients see for it).
func fromProtoSessions(in []*competitionsv1.CompetitionSession) ([]domain.Session, error) {
	out := make([]domain.Session, 0, len(in))
	for _, s := range in {
		r, err := domain.NewTimeRange(s.GetStartsAt().AsTime(), s.GetEndsAt().AsTime())
		if err != nil {
			return nil, err
		}
		out = append(out, domain.Session{Range: r, CourtIDs: s.GetCourtIds()})
	}
	return out, nil
}

// fromProtoPaymentMethod resolves the wire zero value
// (PAYMENT_METHOD_UNSPECIFIED — sent by any client that never sets the
// field) to domain.PaymentMethodEither, the least restrictive value, so the
// field stays genuinely optional.
//
// Any OTHER unrecognized value (e.g. from a future proto version this build
// doesn't know) is deliberately NOT folded into the same default: it is
// passed through as an invalid domain.PaymentMethod so
// domain.NewCompetition's IsValid() check rejects it, rather than silently
// accepting an unknown value as if it meant "either". UNSPECIFIED and
// "unrecognized" are not the same thing and must not share a fallback — a PE
// review finding on T8.7's original PR, inherited here rather than
// rediscovered.
func fromProtoPaymentMethod(m competitionsv1.PaymentMethod) domain.PaymentMethod {
	switch m {
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED:
		return domain.PaymentMethodEither
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_ONLINE:
		return domain.PaymentMethodOnline
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return domain.PaymentMethodCash
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER:
		return domain.PaymentMethodEither
	default:
		return domain.PaymentMethod("")
	}
}

// fromProtoFormat converts the wire format enum.
//
// Unlike payment method, COMPETITION_FORMAT_UNSPECIFIED is NOT resolved to a
// default: there is no "least restrictive" format the way "either" is the
// least restrictive payment method, and silently picking singles or doubles
// on the Host's behalf would put a label on their Competition that they
// never chose — one players would read as fact. It is passed through as an
// invalid domain.Format so NewCompetition rejects it with
// ErrInvalidFormat/400, making the client state its choice.
func fromProtoFormat(f competitionsv1.CompetitionFormat) domain.Format {
	switch f {
	case competitionsv1.CompetitionFormat_COMPETITION_FORMAT_SINGLES:
		return domain.FormatSingles
	case competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES:
		return domain.FormatDoubles
	default:
		return domain.Format("")
	}
}

// fromProtoEntrySource resolves the wire zero value
// (ENTRY_SOURCE_UNSPECIFIED) to domain.EntrySourceApp — the in-app default,
// which is what a client that doesn't set the field is by definition doing —
// while passing any other unrecognized value through as invalid so
// domain.Enter rejects it (ErrInvalidEntrySource/400). Same
// UNSPECIFIED-is-not-unrecognized discipline as fromProtoPaymentMethod.
func fromProtoEntrySource(s competitionsv1.EntrySource) domain.EntrySource {
	switch s {
	case competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED:
		return domain.EntrySourceApp
	case competitionsv1.EntrySource_ENTRY_SOURCE_APP:
		return domain.EntrySourceApp
	case competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL:
		return domain.EntrySourceSocial
	default:
		return domain.EntrySource("")
	}
}

// fromProtoMoney maps an ABSENT entry_fee (a nil *Money — an older client,
// or a field left unset) onto the zero domain.Money, which means a FREE
// Competition. That is not an "unspecified" resolution step: zero is a real,
// well-formed value here, so there is nothing to resolve.
func fromProtoMoney(m *competitionsv1.Money) domain.Money {
	if m == nil {
		return domain.Money{}
	}
	return domain.Money{
		AmountCents:  m.GetAmountCents(),
		CurrencyCode: m.GetCurrencyCode(),
	}
}

// --- domain -> wire --------------------------------------------------------

func toProtoCompetitionStatus(s domain.Status) competitionsv1.CompetitionStatus {
	switch s {
	case domain.StatusScheduled:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_SCHEDULED
	case domain.StatusCancelled:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_CANCELLED
	default:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_UNSPECIFIED
	}
}

func toProtoFormat(f domain.Format) competitionsv1.CompetitionFormat {
	switch f {
	case domain.FormatSingles:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_SINGLES
	case domain.FormatDoubles:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES
	default:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_UNSPECIFIED
	}
}

func toProtoPaymentMethod(m domain.PaymentMethod) competitionsv1.PaymentMethod {
	switch m {
	case domain.PaymentMethodOnline:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_ONLINE
	case domain.PaymentMethodCash:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_CASH
	case domain.PaymentMethodEither:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER
	default:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func toProtoEntrySource(s domain.EntrySource) competitionsv1.EntrySource {
	switch s {
	case domain.EntrySourceApp:
		return competitionsv1.EntrySource_ENTRY_SOURCE_APP
	case domain.EntrySourceSocial:
		return competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL
	default:
		return competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED
	}
}

func toProtoEntryStatus(s domain.EntryStatus) competitionsv1.EntryStatus {
	switch s {
	case domain.EntryStatusEntered:
		return competitionsv1.EntryStatus_ENTRY_STATUS_ENTERED
	case domain.EntryStatusCancelled:
		return competitionsv1.EntryStatus_ENTRY_STATUS_CANCELLED
	default:
		return competitionsv1.EntryStatus_ENTRY_STATUS_UNSPECIFIED
	}
}

func toProtoPaymentStatus(s domain.PaymentStatus) competitionsv1.PaymentStatus {
	switch s {
	case domain.PaymentStatusUnpaid:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_UNPAID
	case domain.PaymentStatusPaid:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_PAID
	case domain.PaymentStatusRefunded:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// toProtoMoney always emits the message, including for a free Competition,
// so a client can distinguish "this Competition is free" (amount_cents 0)
// from a field the server never populated — clients render the former as the
// word "Free", never as blank or a bare "$0.00".
func toProtoMoney(m domain.Money) *competitionsv1.Money {
	return &competitionsv1.Money{
		AmountCents:  m.AmountCents,
		CurrencyCode: m.CurrencyCode,
	}
}

func toProtoSession(s domain.Session) *competitionsv1.CompetitionSession {
	return &competitionsv1.CompetitionSession{
		StartsAt: timestamppb.New(s.Range.Start),
		EndsAt:   timestamppb.New(s.Range.End),
		CourtIds: s.CourtIDs,
	}
}

// toProtoCompetition deliberately does NOT emit ShareToken — the wire
// Competition message has no such field, precisely so this function cannot
// leak it through GetCompetition or ListCompetitions. See the proto's
// Competition doc comment.
func toProtoCompetition(c domain.Competition) *competitionsv1.Competition {
	sessions := make([]*competitionsv1.CompetitionSession, 0, len(c.Sessions))
	for _, s := range c.Sessions {
		sessions = append(sessions, toProtoSession(s))
	}
	return &competitionsv1.Competition{
		Id:              c.ID,
		HostId:          c.HostID,
		Name:            c.Name,
		VenueFacilityId: c.VenueFacilityID,
		Sessions:        sessions,
		Capacity:        int32(c.Capacity),
		GuestAllowance:  int32(c.GuestAllowance),
		PaymentMethod:   toProtoPaymentMethod(c.PaymentMethod),
		EntryFee:        toProtoMoney(c.EntryFee),
		Format:          toProtoFormat(c.Format),
		Status:          toProtoCompetitionStatus(c.Status),
	}
}

// toProtoCompetitionListing converts a port.CompetitionListing to its wire
// message. SpotsLeft is already the weighted value computed server-side —
// see the proto's CompetitionListing doc comment for why the weighting
// matters and internal/competitions/domain's TestSpotsLeft for the boundary
// proof.
func toProtoCompetitionListing(l port.CompetitionListing) *competitionsv1.CompetitionListing {
	return &competitionsv1.CompetitionListing{
		Competition: toProtoCompetition(l.Competition),
		SpotsLeft:   int32(l.SpotsLeft),
	}
}

func toProtoEntry(e domain.CompetitionEntry) *competitionsv1.CompetitionEntry {
	return &competitionsv1.CompetitionEntry{
		Id:            e.ID,
		CompetitionId: e.CompetitionID,
		PlayerId:      e.PlayerID,
		GuestCount:    int32(e.GuestCount),
		Source:        toProtoEntrySource(e.Source),
		PaymentStatus: toProtoPaymentStatus(e.PaymentStatus),
		Status:        toProtoEntryStatus(e.Status),
	}
}
