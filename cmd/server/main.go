// Command server wires the Booking and Payments contexts' gRPC services and
// their grpc-gateway REST mappings into one process, backed by Postgres. It
// only compiles after `make generate` (see CLAUDE.md gotchas) since it
// depends on internal/gen/pickleball/booking/v1 and internal/gen/
// pickleball/payments/v1.
//
// Social Play (T5) is not registered here: internal/socialplay does not
// exist on this branch — T5 has not yet been merged into
// claude/go-backend-pickleball-7up34j (see the T6.4 PR description's
// judgment-call note on why app.ServiceOptions also omits the T6.5
// RegistrationPaymentUpdater field for the same reason). The T5 sprint
// branches' cmd/server (see sprint/t5.5-authz-regression-tests) show the
// pattern this file will extend once Social Play lands: one grpc.Server,
// one grpc-gateway mux, one RegisterXServiceServer/
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
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
	paymentsgrpc "github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	"github.com/nhuthuynh/white-label/internal/platform/pg"
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

	// Payments (T6.4). stripestub stands in for a real Stripe adapter
	// (internal/payments/adapter/stripe, not yet built — T6.2's ACL is
	// designed so that swap is adapter-only, see port.PaymentProcessor's
	// doc comment) — there is no real Stripe SDK dependency this sprint.
	paymentsRepo := paymentspg.NewRepository(pool)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:  paymentsRepo,
		IDs:       idgen.UUID{},
		Processor: stripestub.NewProcessor(),
	})
	paymentsHandler := paymentsgrpc.NewHandler(paymentsSvc)

	grpcServer := grpc.NewServer()
	bookingv1.RegisterBookingServiceServer(grpcServer, bookingHandler)
	paymentsv1.RegisterPaymentsServiceServer(grpcServer, paymentsHandler)

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
	if err := paymentsv1.RegisterPaymentsServiceHandlerFromEndpoint(ctx, mux, grpcAddr, dialOpts); err != nil {
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
