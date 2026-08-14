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
	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
	"github.com/nhuthuynh/white-label/internal/platform/grpcrecovery"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	"github.com/nhuthuynh/white-label/internal/platform/pg"
	socialplaybooking "github.com/nhuthuynh/white-label/internal/socialplay/adapter/booking"
	socialplayfacilities "github.com/nhuthuynh/white-label/internal/socialplay/adapter/facilities"
	socialplaygrpc "github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
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
	facilitiesRepo := facilitiespg.NewRepository(pool)
	facilitiesSvc := facilitiesapp.NewService(facilitiesRepo, idgen.UUID{})
	facilitiesHandler := facilitiesgrpc.NewHandler(facilitiesSvc)

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
	identityRepo := identitypg.NewRepository(pool)
	identitySvc := identityapp.NewService(identityRepo, idgen.UUID{})
	identityHandler := identitygrpc.NewHandler(identitySvc)

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
	bookingSvc := bookingapp.NewService(repo, pricingRepo, discountRepo, recurringRepo,
		bookingFacilityLookup, bookingIdentityLookup, idgen.UUID{})
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
	reservation := socialplaybooking.NewReservation(bookingSvc)
	facilityLookup := socialplayfacilities.NewLookup(facilitiesSvc)
	socialplaySvc := socialplayapp.NewService(idgen.UUID{}, gameRepo, registrationRepo, waitlistRepo, matchRepo)
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
	competitionsSvc := competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions: competitionsRepo,
		IDs:          idgen.UUID{},
		Reservation:  competitionsbooking.NewReservation(bookingSvc),
		Facilities:   competitionsfacilities.NewLookup(facilitiesSvc),
		ShareTokens:  sharetoken.Generator{},
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
	paymentsRepo := paymentspg.NewRepository(pool)
	registrationUpdater := paymentssocialplay.NewRegistrationUpdater(socialplaySvc)
	competitionEntryUpdater := paymentscompetitions.NewEntryUpdater(competitionsSvc)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:                paymentsRepo,
		IDs:                     idgen.UUID{},
		Processor:               stripestub.NewProcessor(),
		RegistrationUpdater:     registrationUpdater,
		CompetitionEntryUpdater: competitionEntryUpdater,
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

	// Per-context authenticated-method policy (T12 sprint plan A11 Ruling
	// 2): each context's grpcapi package owns the list of ITS OWN RPCs that
	// require a verified principal, next to the handlers that break if that
	// list is wrong, and this is the only place those lists are composed.
	// Each migrated context contributes exactly one append line below.
	//
	// Everything not listed stays exactly as public as it is today —
	// composing an empty policy is what made T12.2's observe-only landing
	// incapable of changing a shipped flow, and adding a context here is
	// the deliberate act of turning enforcement on for it.
	var authenticatedMethods []string
	authenticatedMethods = append(authenticatedMethods, identitygrpc.AuthenticatedMethods()...) // T12.9

	// OPERATIONAL CONSEQUENCE, stated here rather than discovered in
	// production: from this point on the process enforces authentication on
	// the methods above, and a principal can only exist if a token verifier
	// is configured (AUTH_ISSUER + AUTH_AUDIENCE + AUTH_JWKS_FILE — see
	// tokenVerifierFromEnv). A deployment with no verifier configured will
	// therefore reject CreateUser and UpdateSelfReportedLevel with
	// Unauthenticated. That is fail-CLOSED and it is the intended
	// direction: the alternative — skipping enforcement whenever no
	// verifier happens to be configured — is a silent downgrade that would
	// look like a check in review and in logs while protecting nothing
	// (ADR-0013 §3, and the fail-open half of known gap #136).
	//
	// Ordering is load-bearing. Recovery stays outermost so it covers the
	// other two. auth.UnaryInterceptor RESOLVES a token into a principal
	// and must run before RequireAuthentication, which only checks whether
	// one is present — registering them the other way round would reject
	// every request including well-authenticated ones.
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcrecovery.UnaryInterceptor(logger),
			auth.UnaryInterceptor(tokenVerifier, logger),
			auth.RequireAuthentication(authenticatedMethods),
		),
		grpc.ChainStreamInterceptor(
			grpcrecovery.StreamInterceptor(logger),
			auth.StreamInterceptor(tokenVerifier, logger),
			auth.RequireAuthenticationStream(authenticatedMethods),
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

// tokenVerifierFromEnv builds the token verifier this process runs with, or
// returns nil when the deployment has not been given an identity provider to
// verify against.
//
// A nil verifier is a supported, currently-normal state, not a failure: no
// identity provider tenant is provisioned for this project, because doing so
// is server-side infrastructure work no coding session here can perform (the
// same gap HANDOFF.md records honestly for the Jenkins server). With a nil
// verifier the interceptors resolve nothing and every request behaves exactly
// as it does today. See ADR-0013.
//
// Partial configuration, by contrast, is a hard startup failure. Someone who
// sets AUTH_ISSUER intends tokens to be verified, and a process that silently
// verified nothing because AUTH_JWKS_FILE had a typo would be the worst of
// both worlds — configured-looking and inert. Once enforcement lands, this
// function must go further and refuse to start with no verifier at all, since
// by then a nil verifier is fail-open rather than merely quiet.
func tokenVerifierFromEnv(logger *slog.Logger) (auth.TokenVerifier, error) {
	var (
		issuer   = os.Getenv("AUTH_ISSUER")
		audience = os.Getenv("AUTH_AUDIENCE")
		jwksFile = os.Getenv("AUTH_JWKS_FILE")
	)

	if issuer == "" && audience == "" && jwksFile == "" {
		logger.Info("auth: no token verifier configured; running with no caller identity " +
			"(set AUTH_ISSUER, AUTH_AUDIENCE and AUTH_JWKS_FILE to enable verification)")
		return nil, nil
	}
	if issuer == "" || audience == "" || jwksFile == "" {
		return nil, errors.New("auth: AUTH_ISSUER, AUTH_AUDIENCE and AUTH_JWKS_FILE must be set together")
	}

	// A file rather than a URL: fetching a live JWKS needs an HTTP client,
	// a cache, and a rotation policy, none of which exist yet and none of
	// which can be tested against a provider this project cannot provision.
	// rs256.KeySource is the seam a remote implementation slots into later
	// without touching the verification logic.
	document, err := os.ReadFile(jwksFile)
	if err != nil {
		return nil, fmt.Errorf("auth: reading AUTH_JWKS_FILE: %w", err)
	}
	keys, err := rs256.NewStaticKeysFromJWKS(document)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	verifier, err := rs256.NewVerifier(rs256.Config{
		Issuer:   issuer,
		Audience: audience,
		Keys:     keys,
	})
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	logger.Info("auth: token verifier configured (observe-only; nothing is enforced yet)",
		"issuer", issuer,
		"audience", audience,
		"jwks_file", jwksFile,
	)
	return verifier, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
