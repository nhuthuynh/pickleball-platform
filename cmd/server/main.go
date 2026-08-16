// Command server wires the Booking, Social Play, Payments, Facilities,
// Competitions, and Identity/Users contexts' gRPC services and their
// grpc-gateway REST mappings into one process, backed by Postgres. It only
// compiles after `make generate` (see CLAUDE.md gotchas) since it depends on
// internal/gen/pickleball/booking/v1, internal/gen/pickleball/socialplay/v1,
// internal/gen/pickleball/payments/v1, internal/gen/pickleball/facilities/v1,
// internal/gen/pickleball/competitions/v1, and
// internal/gen/pickleball/identity/v1 (T10.2).
//
// Payments' RegistrationUpdater dependency (socialplayport.
// RegistrationPaymentUpdater, satisfied by internal/payments/adapter/
// socialplay, T6.5) is wired against the same, real socialplaySvc instance
// Social Play's own gRPC handler uses below, not a second/separate stack;
// its CompetitionEntryUpdater dependency (competitionsport.
// CompetitionEntryPaymentUpdater, satisfied by internal/payments/adapter/
// competitions, T10.6, closes #96) is wired the identical way against the
// same, real competitionsSvc instance Competitions' own gRPC handler uses
// below. One grpc.Server, one grpc-gateway mux, one RegisterXServiceServer/
// RegisterXServiceHandlerFromEndpoint pair per context.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bookingfacilities "github.com/nhuthuynh/white-label/internal/booking/adapter/facilities"
	bookinggrpc "github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	bookingidentity "github.com/nhuthuynh/white-label/internal/booking/adapter/identity"
	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	competitionsbooking "github.com/nhuthuynh/white-label/internal/competitions/adapter/booking"
	competitionsfacilities "github.com/nhuthuynh/white-label/internal/competitions/adapter/facilities"
	competitionsgrpc "github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"
	competitionspg "github.com/nhuthuynh/white-label/internal/competitions/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/competitions/adapter/sharetoken"
	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	facilitiesgrpc "github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	facilitiesidentity "github.com/nhuthuynh/white-label/internal/facilities/adapter/identity"
	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
	identityv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/identity/v1"
	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
	identitygrpc "github.com/nhuthuynh/white-label/internal/identity/adapter/grpcapi"
	identitypg "github.com/nhuthuynh/white-label/internal/identity/adapter/postgres"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	paymentsgrpc "github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	paymentsidentity "github.com/nhuthuynh/white-label/internal/payments/adapter/identity"
	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/webhookstub"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
	"github.com/nhuthuynh/white-label/internal/platform/grpcrecovery"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	"github.com/nhuthuynh/white-label/internal/platform/pg"
	socialplaybooking "github.com/nhuthuynh/white-label/internal/socialplay/adapter/booking"
	socialplayfacilities "github.com/nhuthuynh/white-label/internal/socialplay/adapter/facilities"
	socialplaygrpc "github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	socialplayidentity "github.com/nhuthuynh/white-label/internal/socialplay/adapter/identity"
	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
)

const shutdownTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	grpcAddr := envOr("GRPC_ADDR", ":8081")
	httpAddr := envOr("HTTP_ADDR", ":8080")
	dsn := envOr("DATABASE_URL", "postgres://pickleball:pickleball@localhost:5432/pickleball?sslmode=disable")

	pool, err := pg.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Identity/Users (T10.2). Its Repository is Postgres-backed like the
	// other contexts', and as of T12.9 it takes a port.IDGenerator like the
	// other contexts too — a User's id is now server-minted rather than the
	// caller-claimed actor_user_id it used to be. That exception was the
	// identity-squatting denial-of-service HANDOFF.md's T10.2 bullet
	// disclosed, and removing it is what closed it.
	//
	// Constructed before Booking as of T11.5, which is the first context to
	// call into Identity at all: Booking's RequestRecurringHire resolves the
	// actor's `club` role against this same, real identitySvc instance — not
	// a second/separate one — through internal/booking/adapter/identity.
	//
	// Moved above Facilities in T13.3, which makes Facilities the second
	// context to call into Identity (ADR-0014's seam, issue #154). This is a
	// pure reordering — no construction below it changed — and it is forced:
	// facilitiesSvc now takes an identity lookup built over identitySvc, so
	// identitySvc has to exist first.
	identityRepo := identitypg.NewRepository(pool)
	identitySvc := identityapp.NewService(identityRepo, idgen.UUID{})
	identityHandler := identitygrpc.NewHandler(identitySvc)

	// Facilities (T7.3). Its Repository is Postgres-backed like Booking's;
	// AddCourt inserts into the *same* courts table Booking's repo below
	// reads court_id from (0001_init.sql + 0010_facilities.sql's
	// facility_id column) — not a second courts table.
	//
	// Built before Booking as of T11.2: Booking now has a FacilityLookup
	// port of its own (its first call into Facilities at all) and is wired
	// against this same, real facilitiesSvc instance — not a second/separate
	// Facilities stack — exactly as Social Play and Competitions already are
	// further down.
	//
	// facilitiesIdentityLookup (T13.3) is Facilities' first outbound call
	// into any other context, and it resolves against the SAME real
	// identitySvc instance Booking's own lookup uses just below — not a
	// second Identity stack. That shared instance is what makes ADR-0014's
	// invariant hold across contexts rather than per-context: a given
	// subject resolves to one User.ID everywhere, so the uuid Facilities
	// stores in facilities.owner_id is the same uuid Booking compares
	// against in ApproveRecurringHire/CreateDiscountRule.
	facilitiesRepo := facilitiespg.NewRepository(pool)
	facilitiesIdentityLookup := facilitiesidentity.NewLookup(identitySvc)
	facilitiesSvc := facilitiesapp.NewService(facilitiesRepo, facilitiesIdentityLookup, idgen.UUID{})
	facilitiesHandler := facilitiesgrpc.NewHandler(facilitiesSvc)

	repo := bookingpg.NewRepository(pool)
	pricingRepo := bookingpg.NewPricingRuleRepository(pool)
	// discountRepo (T11.2) backs CreateDiscountRule/
	// ListDiscountRulesForFacility and the discount half of GetQuote,
	// against the discount_rules table (0017_booking_discount_rules.sql).
	discountRepo := bookingpg.NewDiscountRuleRepository(pool)
	// recurringRepo (T11.5) backs RequestRecurringHire/ApproveRecurringHire/
	// RejectRecurringHire/ListRecurringHireTemplatesForFacility, against the
	// recurring_hire_templates table
	// (0018_booking_recurring_hire_templates.sql).
	recurringRepo := bookingpg.NewRecurringHireRepository(pool)
	bookingFacilityLookup := bookingfacilities.NewLookup(facilitiesSvc)
	bookingIdentityLookup := bookingidentity.NewLookup(identitySvc)
	bookingSvc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       repo,
		PricingRules:   pricingRepo,
		DiscountRules:  discountRepo,
		RecurringHires: recurringRepo,
		Facilities:     bookingFacilityLookup,
		Identity:       bookingIdentityLookup,
		IDs:            idgen.UUID{},
	})
	bookingHandler := bookinggrpc.NewHandler(bookingSvc)

	// Social Play (T5.4): its GameRepository/RegistrationRepository are
	// Postgres-backed like Booking's; its CourtReservation port is
	// implemented against the real Booking app.Service (the one place
	// Social Play code is allowed to import internal/booking/*, per
	// CLAUDE.md rule 5 — see internal/socialplay/adapter/booking). Its
	// FacilityLookup port (T8.3) gets the same treatment against the real
	// Facilities app.Service already built above (internal/socialplay/
	// adapter/facilities) — one, real facilitiesSvc instance, not a second/
	// separate Facilities stack.
	gameRepo := socialplaypg.NewGameRepository(pool)
	registrationRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	// matchRepo (T10.4) is Postgres-backed the same way; RecordMatchResult/
	// ListMatchesForGame reuse Social Play's existing Host/Game-Admin
	// authorization shape, not a new mechanism, so no new port beyond this
	// one repository is wired here.
	matchRepo := socialplaypg.NewMatchRepository(pool)
	// gameAdminRepo (T14.4, partial fix for #168) is the durable Game-Admin
	// store. Before it, "who is a Game Admin of this Game" existed only as a
	// repeated string field on the requests that needed it, so every rule
	// wanting to include admins had to trust a list the caller wrote. It is
	// wired here as a plain fifth repository — no new cross-context
	// dependency: the assignment is a Social Play fact about a Social Play
	// aggregate, and its user ids are subjects in the same space
	// games.host_id already holds (ADR-0014 §5a).
	gameAdminRepo := socialplaypg.NewGameAdminRepository(pool)
	reservation := socialplaybooking.NewReservation(bookingSvc)
	facilityLookup := socialplayfacilities.NewLookup(facilitiesSvc)
	// socialplayIdentityLookup (T29.2, closes the Social Play third of #164)
	// is Social Play's first outbound call into Identity, built against the
	// SAME real identitySvc instance Booking's/Facilities'/Payments' own
	// lookups use above (bookingIdentityLookup, facilitiesIdentityLookup,
	// paymentsIdentityLookup below) — not a second/separate Identity stack.
	// That shared instance is what makes ADR-0014's invariant hold across
	// contexts rather than per-context: a given subject resolves to one
	// User.ID everywhere, so the uuid Social Play now stores in
	// games.host_id/registrations.player_id/waitlist_entries.player_id/
	// game_admins.user_id is the same uuid Payments' authorizeGameRecording
	// compares it against (closing Social Play's half of #237 as a side
	// effect — see internal/payments/app/service.go's own doc comment, which
	// this ticket does not modify).
	socialplayIdentityLookup := socialplayidentity.NewLookup(identitySvc)
	socialplaySvc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           idgen.UUID{},
		Games:         gameRepo,
		Registrations: registrationRepo,
		Waitlist:      waitlistRepo,
		Matches:       matchRepo,
		GameAdmins:    gameAdminRepo,
		Identity:      socialplayIdentityLookup,
	})
	socialplayHandler := socialplaygrpc.NewHandler(socialplaySvc, reservation, facilityLookup)

	// Competitions (T9.4). Same shape as Social Play's wiring above, one
	// level deeper: its Repository is Postgres-backed, its CourtReservation
	// port is implemented against the real Booking app.Service (reserving
	// one `competition`-source Booking per (session, court) pair, so a
	// Competition inherits the no-double-booking invariant), and its
	// FacilityLookup port against the real Facilities app.Service — both
	// against the SAME bookingSvc/facilitiesSvc instances built above, not
	// second/separate stacks.
	//
	// ShareTokens is the crypto/rand-backed generator
	// (internal/competitions/adapter/sharetoken): CreateCompetition mints a
	// real token for every Competition from the moment this endpoint is
	// live, so it must not be a placeholder — see that package's doc comment.
	//
	// Built BEFORE Payments below (T10.6, closes #96): Payments'
	// CompetitionEntryUpdater dependency needs the real competitionsSvc
	// instance already constructed, the same reordering constraint
	// RegistrationUpdater already imposed on Social Play above.
	competitionsRepo := competitionspg.NewRepository(pool)
	// The durable Competition-Admin store (T15.3, partial fix for #168) — its
	// own narrow repository over competition_admins, mirroring the Game-Admin
	// store wired for Social Play above. See
	// competitionsport.CompetitionAdminRepository for why it is a separate
	// interface rather than three more methods on the Competitions repository.
	competitionAdminRepo := competitionspg.NewCompetitionAdminRepository(pool)
	competitionsSvc := competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions:      competitionsRepo,
		IDs:               idgen.UUID{},
		Reservation:       competitionsbooking.NewReservation(bookingSvc),
		Facilities:        competitionsfacilities.NewLookup(facilitiesSvc),
		ShareTokens:       sharetoken.Generator{},
		CompetitionAdmins: competitionAdminRepo,
	})
	competitionsHandler := competitionsgrpc.NewHandler(competitionsSvc)

	// Payments (T6.4). stripestub stands in for a real Stripe adapter
	// (internal/payments/adapter/stripe, not yet built — T6.2's ACL is
	// designed so that swap is adapter-only, see port.PaymentProcessor's
	// doc comment) — there is no real Stripe SDK dependency this sprint.
	// RegistrationUpdater (T6.5) is the mirror image of Social Play's own
	// internal/socialplay/adapter/booking: it lets Payments push a
	// Registration's PaymentStatus forward without Social Play importing
	// anything under internal/payments (CLAUDE.md rule 3, context-map
	// direction in docs/process/t6-sprint-plan.md's kickoff note). It's
	// built against the same, real socialplaySvc instance Social Play's own
	// gRPC handler uses above, not a second/separate Social Play stack.
	// CompetitionEntryUpdater (T10.6, closes #96) is the identical pattern
	// mirrored for Competitions: built against the same, real
	// competitionsSvc instance Competitions' own gRPC handler uses above.
	// RegistrationLookup/GameLookup/GameAdminReader and EntryLookup/
	// CompetitionAdminReader (T16.2, closes #168) are the resolver ports
	// authorizeOfflineRecording uses to verify a Registration's Game's
	// Host/admins and a CompetitionEntry's Competition's entrant/admins
	// against the real stores, wired the identical way
	// registrationUpdater/competitionEntryUpdater already are: against the
	// same, real socialplaySvc/competitionsSvc instances those contexts'
	// own gRPC handlers use above, not a second/separate stack. See
	// internal/payments/app/service.go's Service doc comment for why these
	// five are not optional the way registrationUpdater/
	// competitionEntryUpdater are.
	// webhookVerifier/webhookEventRepo (T18.1, closes #167) back
	// ReceiveStripeWebhookEvent — webhookstub.NewVerifier stands in for a
	// real Stripe signature check the identical way stripestub.NewProcessor
	// stands in for the real capture/refund calls above: no Stripe SDK
	// dependency, no network call, no STRIPE_* environment variable read
	// this ticket (see webhookstub's own doc comment). The shared secret is
	// a placeholder literal, matching how stripestub.NewProcessor() needs no
	// configuration at all today — a future real
	// internal/payments/adapter/stripe package is what reads a real
	// STRIPE_WEBHOOK_SECRET-shaped env var, not this ticket.
	paymentsRepo := paymentspg.NewRepository(pool)
	registrationUpdater := paymentssocialplay.NewRegistrationUpdater(socialplaySvc)
	competitionEntryUpdater := paymentscompetitions.NewEntryUpdater(competitionsSvc)
	registrationLookup := paymentssocialplay.NewRegistrationLookup(socialplaySvc)
	gameLookup := paymentssocialplay.NewGameLookup(socialplaySvc)
	gameAdminReader := paymentssocialplay.NewGameAdminReader(socialplaySvc)
	entryLookup := paymentscompetitions.NewEntryLookup(competitionsSvc)
	competitionAdminReader := paymentscompetitions.NewCompetitionAdminReader(competitionsSvc)
	webhookVerifier := webhookstub.NewVerifier("whsec_placeholder_dev_only")
	webhookEventRepo := paymentspg.NewWebhookEventRepository(pool)
	// Identity lookup (T28.1, closes the Payments third of #164): built
	// against the same, real identitySvc instance Booking's/Facilities' own
	// lookups use above (bookingIdentityLookup, facilitiesIdentityLookup) —
	// not a second/separate stack — mirroring both exactly.
	paymentsIdentityLookup := paymentsidentity.NewLookup(identitySvc)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:                paymentsRepo,
		IDs:                     idgen.UUID{},
		Processor:               stripestub.NewProcessor(),
		RegistrationUpdater:     registrationUpdater,
		CompetitionEntryUpdater: competitionEntryUpdater,
		RegistrationLookup:      registrationLookup,
		GameLookup:              gameLookup,
		GameAdminReader:         gameAdminReader,
		EntryLookup:             entryLookup,
		CompetitionAdminReader:  competitionAdminReader,
		WebhookVerifier:         webhookVerifier,
		WebhookEvents:           webhookEventRepo,
		Identity:                paymentsIdentityLookup,
	})
	paymentsHandler := paymentsgrpc.NewHandler(paymentsSvc)

	// The recovery interceptors are the process's panic safety net and must
	// stay installed. grpc, unlike net/http, installs no recover() of its own:
	// an unrecovered panic in any one handler unwinds past the server and kills
	// this entire process, taking every other in-flight request and every other
	// bounded context with it. Until this was added, grpc.NewServer() was called
	// with no options at all, and an unauthenticated
	// `GET /v1/competitions/not-a-uuid` reached a Postgres adapter's mustUUID
	// and did exactly that — a one-request total outage.
	//
	// This protects every handler in all five contexts, present and future,
	// against every panic — not just the malformed-ID one that exposed it. It is
	// the backstop, not the fix: input is still validated at the boundary (see
	// the app-layer ID checks) so these panics don't happen in the first place.
	//
	// Chain* rather than the single-interceptor options so adding auth/metrics/
	// tracing later is an append, and recovery — which must be outermost to
	// cover the other interceptors too — stays first in the chain.
	// The auth interceptors sit immediately after recovery — inside its
	// protection, ahead of every handler. They are OBSERVE-ONLY (T12.2): when
	// a caller presents a valid bearer token they put an auth.Principal in the
	// request context, and in every other case — no token, expired, forged,
	// wrong audience, or no verifier configured at all — the request proceeds
	// to its handler exactly as it did before, with no principal and no error.
	// Nothing in this process enforces authentication yet; the per-context
	// migration tickets (T12.7/T12.8/T12.9) turn that on RPC by RPC. Adding
	// them here therefore cannot change the behavior of any shipped endpoint,
	// which is the entire point of landing the platform capability first.
	tokenVerifier, err := tokenVerifierFromEnv(logger)
	if err != nil {
		return err
	}

	// The authentication policy, composed from each context's own list
	// (A11 Ruling 2). One line per context, and each context decides which of
	// its RPCs are public next to the handlers that break if it is wrong —
	// see the AuthenticatedMethods/PublicMethods pair in each grpcapi package,
	// where a test fails if any RPC on the service is in neither list.
	//
	// T12.7 and T12.9 both appended to this composition in the same wave
	// (T12 sprint plan A12); T12.9 also independently built a second,
	// functionally-identical enforcement primitive
	// (auth.RequireAuthentication/RequireAuthenticationStream in
	// internal/platform/auth/enforce.go) before T12.7 had merged and it had
	// anything to consolidate onto. Resolved here by keeping the one
	// T12.7 already landed (auth.MethodSet/RequireUnaryInterceptor/
	// RequireStreamInterceptor) and adding identity's list to it —
	// enforce.go/enforce_test.go are deleted as part of this resolution
	// rather than left as a second, unused enforcement path in the same
	// package.
	// T12.8 adds the last three contexts, completing the migration: every
	// bounded context in this process now declares its own authenticated/public
	// split, and each declares it in its own file, so this call is the only
	// place they meet. No textual conflict occurred with T12.7's and T12.9's
	// entries — this ticket branched from a base already containing both — but
	// per A9(e) the diff still touches a file more than one ticket claimed this
	// sprint, so it was re-verified from a fresh worktree off the pushed branch.
	// T13.5 (#136) replaced the startup *warning* that used to live here with
	// a startup *failure*; the composition itself moved into
	// authenticationPolicy so both halves are reachable from a test.
	authenticatedMethods, err := authenticationPolicy(tokenVerifier, logger)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcrecovery.UnaryInterceptor(logger),
			auth.UnaryInterceptor(tokenVerifier, logger),
			auth.RequireUnaryInterceptor(authenticatedMethods),
		),
		grpc.ChainStreamInterceptor(
			grpcrecovery.StreamInterceptor(logger),
			auth.StreamInterceptor(tokenVerifier, logger),
			auth.RequireStreamInterceptor(authenticatedMethods),
		),
	)
	bookingv1.RegisterBookingServiceServer(grpcServer, bookingHandler)
	socialplayv1.RegisterSocialPlayServiceServer(grpcServer, socialplayHandler)
	paymentsv1.RegisterPaymentsServiceServer(grpcServer, paymentsHandler)
	facilitiesv1.RegisterFacilitiesServiceServer(grpcServer, facilitiesHandler)
	competitionsv1.RegisterCompetitionsServiceServer(grpcServer, competitionsHandler)
	identityv1.RegisterIdentityServiceServer(grpcServer, identityHandler)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}

	go func() {
		logger.Info("grpc server listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc server stopped", "error", err)
		}
	}()

	mux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := bookingv1.RegisterBookingServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}
	if err := socialplayv1.RegisterSocialPlayServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}
	if err := paymentsv1.RegisterPaymentsServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}
	if err := facilitiesv1.RegisterFacilitiesServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}
	if err := competitionsv1.RegisterCompetitionsServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}
	if err := identityv1.RegisterIdentityServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
		return err
	}

	httpServer := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		logger.Info("http gateway listening", "addr", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()
	return nil
}

// authenticationPolicy composes each bounded context's own
// AuthenticatedMethods() into the single policy this process enforces, and
// refuses to hand it back when the process cannot actually perform it.
//
// # The composition
//
// One line per context (A11 Ruling 2): each context decides which of its RPCs
// are public next to the handlers that break if it is wrong — see the
// AuthenticatedMethods/PublicMethods pair in each grpcapi package, where a
// test fails if any RPC on the service is in neither list. This function is
// the only place those six lists meet.
//
// # The refusal (T13.5, issue #136)
//
// Until T13.5 a nil verifier here produced a warning and a running server.
// That was defensible while the alternative was an unstartable process — but
// it means a deployment can advertise "these RPCs require authentication"
// while holding nothing that can verify a token, and every symptom of that
// state arrives disguised as something else: a fleet of Unauthenticated
// responses that look, from the client side, exactly like expired credentials.
// An operator debugging that reads their identity provider's logs, not ours.
//
// Startup is the only moment the condition is legible, so it fails here, and
// the error names the three variables that fix it. The remedy text lives in
// this file rather than in internal/platform/auth because AUTH_ISSUER and its
// siblings are cmd/server's vocabulary — the platform package states the rule,
// this one states how to satisfy it in this binary (CLAUDE.md rule 3).
//
// # The cost, stated
//
// This binary no longer starts without AUTH_ISSUER, AUTH_AUDIENCE and
// AUTH_JWKS_FILE, because all six contexts declare authenticated RPCs and
// none of them can be honoured without a verifier. That includes `make up`
// and a bare `go run ./cmd/server`. This is the intended consequence rather
// than an oversight — a server that cannot verify a token must not claim to
// require one — but it is a real cost to local development, and closing it
// properly needs a committed dev keypair, a JWKS fixture and a token-minting
// helper. That is a distinct piece of work, deliberately not smuggled into a
// hardening ticket; it is tracked as issue #160.
//
// It is *not* a security downgrade in the meantime: a developer who wants the
// old behaviour can get it honestly by pointing AUTH_JWKS_FILE at a key set,
// and cannot get it by accident.
func authenticationPolicy(verifier auth.TokenVerifier, logger *slog.Logger) (auth.MethodSet, error) {
	set := auth.NewMethodSet(
		bookinggrpc.AuthenticatedMethods(),      // T12.7
		facilitiesgrpc.AuthenticatedMethods(),   // T12.7
		identitygrpc.AuthenticatedMethods(),     // T12.9
		socialplaygrpc.AuthenticatedMethods(),   // T12.8
		paymentsgrpc.AuthenticatedMethods(),     // T12.8
		competitionsgrpc.AuthenticatedMethods(), // T12.8
	)

	if err := auth.EnsureVerifierConfigured(verifier, set); err != nil {
		return auth.MethodSet{}, fmt.Errorf(
			"%w; set AUTH_ISSUER, AUTH_AUDIENCE and exactly one of AUTH_JWKS_FILE or AUTH_JWKS_URL "+
				"so this process can verify the tokens it demands", err)
	}

	logger.Info("auth: enforcement active", "authenticated_methods", set.Len())
	return set, nil
}

// tokenVerifierFromEnv builds the token verifier this process runs with, or
// returns nil when the deployment has not been given an identity provider to
// verify against.
//
// It reports the *configuration*, and deliberately does not decide whether a
// nil result is acceptable — that depends on whether this process enforces
// anything, which is authenticationPolicy's question and not this one's. Since
// T13.5 the answer in this binary is always "no", because all six contexts
// declare authenticated RPCs; a build of this server that enforced nothing
// would still be free to run without a verifier, and keeping the two concerns
// apart is what makes that true by construction rather than by luck.
//
// Partial configuration, by contrast, is a hard startup failure right here.
// Someone who sets AUTH_ISSUER intends tokens to be verified, and a process
// that silently verified nothing because AUTH_JWKS_FILE had a typo would be
// the worst of both worlds — configured-looking and inert. See ADR-0013.
func tokenVerifierFromEnv(logger *slog.Logger) (auth.TokenVerifier, error) {
	var (
		issuer   = os.Getenv("AUTH_ISSUER")
		audience = os.Getenv("AUTH_AUDIENCE")
		jwksFile = os.Getenv("AUTH_JWKS_FILE")
		jwksURL  = os.Getenv("AUTH_JWKS_URL")
	)

	if issuer == "" && audience == "" && jwksFile == "" && jwksURL == "" {
		logger.Info("auth: no token verifier configured " +
			"(set AUTH_ISSUER, AUTH_AUDIENCE and one of AUTH_JWKS_FILE / AUTH_JWKS_URL to enable verification)")
		return nil, nil
	}
	if issuer == "" || audience == "" || (jwksFile == "" && jwksURL == "") {
		return nil, errors.New(
			"auth: AUTH_ISSUER, AUTH_AUDIENCE and exactly one of AUTH_JWKS_FILE or AUTH_JWKS_URL must be set together")
	}
	// Both set is a startup error rather than a precedence rule (T15.7).
	// Picking a winner would mean a deployment that believes it verifies
	// against its live provider while actually trusting a stale file someone
	// left in the image — the "configured-looking and inert" failure this
	// function already refuses elsewhere, wearing a different hat. There is no
	// deployment that legitimately wants both, so ambiguity here is always a
	// mistake, and the cheapest place to find a mistake is at startup.
	if jwksFile != "" && jwksURL != "" {
		return nil, errors.New(
			"auth: AUTH_JWKS_FILE and AUTH_JWKS_URL are mutually exclusive; set exactly one " +
				"(a file for local development, a URL for a live identity provider)")
	}

	keys, source, err := keySourceFromEnv(jwksFile, jwksURL)
	if err != nil {
		return nil, err
	}

	verifier, err := rs256.NewVerifier(rs256.Config{
		Issuer:   issuer,
		Audience: audience,
		Keys:     keys,
	})
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	logger.Info("auth: token verifier configured",
		"issuer", issuer,
		"audience", audience,
		"jwks_source", source,
	)
	return verifier, nil
}

// keySourceFromEnv builds the one key source this process trusts, and returns
// a description of it for the startup log.
//
// The two sources are deliberately asymmetric in one respect only: the remote
// one is fetched once here, before the server starts.
func keySourceFromEnv(jwksFile, jwksURL string) (rs256.KeySource, string, error) {
	if jwksFile != "" {
		document, err := os.ReadFile(jwksFile)
		if err != nil {
			return nil, "", fmt.Errorf("auth: reading AUTH_JWKS_FILE: %w", err)
		}
		keys, err := rs256.NewStaticKeysFromJWKS(document)
		if err != nil {
			return nil, "", fmt.Errorf("auth: AUTH_JWKS_FILE: %w", err)
		}
		return keys, "file:" + jwksFile, nil
	}

	keys, err := rs256.NewRemoteKeys(rs256.RemoteConfig{URL: jwksURL})
	if err != nil {
		return nil, "", fmt.Errorf("auth: AUTH_JWKS_URL: %w", err)
	}

	// Warm the cache, and refuse to start if it cannot be warmed.
	//
	// The alternative — start, and let the first RPC discover the endpoint is
	// wrong — is strictly worse. RemoteKeys is fail-closed with no cache to
	// fall back on, so a typo'd or unreachable URL produces a process that
	// boots, passes its health check, and denies every authenticated request
	// with an error that looks to clients exactly like expired credentials.
	// That is the #136 failure mode, one layer down. A process that cannot
	// fetch its provider's keys cannot do its job either way; the only
	// question is whether it says so at 03:00 or at the first request, and
	// startup is the moment the condition is legible (ADR-0013, T13.5).
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := keys.Refresh(ctx); err != nil {
		return nil, "", fmt.Errorf(
			"auth: fetching the key set named by AUTH_JWKS_URL: %w "+
				"(the process would otherwise start and deny every authenticated request)", err)
	}
	return keys, "url:" + jwksURL, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
