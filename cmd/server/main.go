// Command server wires the Booking, Social Play, Payments, and Facilities
// contexts' gRPC services and their grpc-gateway REST mappings into one
// process, backed by Postgres. It only compiles after `make generate` (see
// CLAUDE.md gotchas) since it depends on internal/gen/pickleball/booking/v1,
// internal/gen/pickleball/socialplay/v1, internal/gen/pickleball/
// payments/v1, and internal/gen/pickleball/facilities/v1.
//
// Payments' RegistrationUpdater dependency (socialplayport.
// RegistrationPaymentUpdater, satisfied by internal/payments/adapter/
// socialplay, T6.5) is wired against the same, real socialplaySvc instance
// Social Play's own gRPC handler uses below, not a second/separate stack:
// one grpc.Server, one grpc-gateway mux, one RegisterXServiceServer/
// RegisterXServiceHandlerFromEndpoint pair per context.
package main

import (
	"context"
	"errors"
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

	bookinggrpc "github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	facilitiesgrpc "github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
	paymentsgrpc "github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	"github.com/nhuthuynh/white-label/internal/platform/pg"
	socialplaybooking "github.com/nhuthuynh/white-label/internal/socialplay/adapter/booking"
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

	repo := bookingpg.NewRepository(pool)
	pricingRepo := bookingpg.NewPricingRuleRepository(pool)
	bookingSvc := bookingapp.NewService(repo, pricingRepo, idgen.UUID{})
	bookingHandler := bookinggrpc.NewHandler(bookingSvc)

	// Facilities (T7.3). Its Repository is Postgres-backed like Booking's;
	// AddCourt inserts into the *same* courts table Booking's repo above
	// reads court_id from (0001_init.sql + 0010_facilities.sql's
	// facility_id column) — not a second courts table.
	facilitiesRepo := facilitiespg.NewRepository(pool)
	facilitiesSvc := facilitiesapp.NewService(facilitiesRepo, idgen.UUID{})
	facilitiesHandler := facilitiesgrpc.NewHandler(facilitiesSvc)

	// Social Play (T5.4): its GameRepository/RegistrationRepository are
	// Postgres-backed like Booking's; its CourtReservation port is
	// implemented against the real Booking app.Service (the one place
	// Social Play code is allowed to import internal/booking/*, per
	// CLAUDE.md rule 5 — see internal/socialplay/adapter/booking).
	gameRepo := socialplaypg.NewGameRepository(pool)
	registrationRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	reservation := socialplaybooking.NewReservation(bookingSvc)
	socialplaySvc := socialplayapp.NewService(idgen.UUID{}, gameRepo, registrationRepo, waitlistRepo)
	socialplayHandler := socialplaygrpc.NewHandler(socialplaySvc, reservation)

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
	paymentsRepo := paymentspg.NewRepository(pool)
	registrationUpdater := paymentssocialplay.NewRegistrationUpdater(socialplaySvc)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:            paymentsRepo,
		IDs:                 idgen.UUID{},
		Processor:           stripestub.NewProcessor(),
		RegistrationUpdater: registrationUpdater,
	})
	paymentsHandler := paymentsgrpc.NewHandler(paymentsSvc)

	grpcServer := grpc.NewServer()
	bookingv1.RegisterBookingServiceServer(grpcServer, bookingHandler)
	socialplayv1.RegisterSocialPlayServiceServer(grpcServer, socialplayHandler)
	paymentsv1.RegisterPaymentsServiceServer(grpcServer, paymentsHandler)
	facilitiesv1.RegisterFacilitiesServiceServer(grpcServer, facilitiesHandler)

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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
