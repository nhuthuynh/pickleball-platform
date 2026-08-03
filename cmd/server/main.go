// Command server wires the Booking context's gRPC service and its
// grpc-gateway REST mapping into one process, backed by Postgres. It only
// compiles after `make generate` (see CLAUDE.md gotchas) since it depends on
// internal/gen/pickleball/booking/v1.
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
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
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

	grpcServer := grpc.NewServer()
	bookingv1.RegisterBookingServiceServer(grpcServer, bookingHandler)
	socialplayv1.RegisterSocialPlayServiceServer(grpcServer, socialplayHandler)

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
	err = bookingv1.RegisterBookingServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts)
	if err != nil {
		return err
	}
	err = socialplayv1.RegisterSocialPlayServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts)
	if err != nil {
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
