package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
	"github.com/incu6us/product-catalog-service/internal/services"
	grpcproduct "github.com/incu6us/product-catalog-service/internal/transport/grpc/product"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("Failed to load .env file", "error", err)
		os.Exit(1)
	}

	observability.SetupLogger()

	appCtx := context.Background()

	cmd := &cli.Command{
		Name:  "product-catalog-service",
		Usage: "Product catalog gRPC service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-addr",
				Value:   ":50051",
				Sources: cli.EnvVars("GRPC_ADDR"),
				Usage:   "gRPC listen address",
			},
			&cli.StringFlag{
				Name:    "spanner-database",
				Value:   "",
				Sources: cli.EnvVars("SPANNER_DATABASE"),
				Usage:   "Spanner database path",
			},
			&cli.StringFlag{
				Name:    "otel-service-name",
				Value:   "product-catalog-service",
				Sources: cli.EnvVars("OTEL_SERVICE_NAME"),
				Usage:   "OpenTelemetry service name",
			},
			&cli.StringFlag{
				Name:    "otel-exporter-otlp-endpoint",
				Sources: cli.EnvVars("OTEL_EXPORTER_OTLP_ENDPOINT"),
				Usage:   "OTLP exporter endpoint",
			},
			&cli.StringFlag{
				Name:    "metrics-addr",
				Value:   ":9090",
				Sources: cli.EnvVars("METRICS_ADDR"),
				Usage:   "Prometheus metrics listen address",
			},
			&cli.StringFlag{
				Name:    "health-addr",
				Value:   ":50052",
				Sources: cli.EnvVars("HEALTH_ADDR"),
				Usage:   "Health check gRPC listen address",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			spannerDB := cmd.String("spanner-database")
			if spannerDB == "" {
				return errors.New("spanner-database is required")
			}

			otelCfg := observability.Config{
				ServiceName:  cmd.String("otel-service-name"),
				OTLPEndpoint: cmd.String("otel-exporter-otlp-endpoint"),
				MetricsAddr:  cmd.String("metrics-addr"),
			}

			otelShutdown, err := observability.Setup(ctx, otelCfg)
			if err != nil {
				slog.Error("Failed to setup observability", "error", err)
				return err
			}
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(appCtx, 5*time.Second)
				defer shutdownCancel()
				if err := otelShutdown(shutdownCtx); err != nil {
					slog.Error("Failed to shutdown observability", "error", err)
				}
			}()

			// Spanner connection
			client, err := spanner.NewClient(ctx, spannerDB)
			if err != nil {
				slog.Error("Failed to create Spanner client", "error", err)
				return err
			}
			defer client.Close()

			// Wire up application
			app := services.NewApp(client, clock.RealClock{})

			// Start gRPC server
			addr := cmd.String("grpc-addr")
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				slog.Error("Failed to listen", "error", err)
				return err
			}

			srv := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
			pb.RegisterProductServiceServer(srv, grpcproduct.NewHandler(app))

			// Start health check on a separate port
			healthAddr := cmd.String("health-addr")
			healthLis, err := net.Listen("tcp", healthAddr)
			if err != nil {
				slog.Error("Failed to listen for health check", "error", err)
				return err
			}

			healthSrv := health.NewServer()
			healthGrpcSrv := grpc.NewServer()
			healthpb.RegisterHealthServer(healthGrpcSrv, healthSrv)
			healthSrv.SetServingStatus("product-catalog-service", healthpb.HealthCheckResponse_SERVING)

			errCh := make(chan error, 2)
			go func() {
				slog.Info("Health gRPC server listening", "addr", healthAddr)
				errCh <- healthGrpcSrv.Serve(healthLis)
			}()
			go func() {
				slog.Info("gRPC server listening", "addr", addr)
				errCh <- srv.Serve(lis)
			}()

			// Graceful shutdown
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			select {
			case <-quit:
			case err := <-errCh:
				slog.Error("gRPC server failed", "error", err)
				return err
			}

			slog.Info("Shutting down server...")
			healthGrpcSrv.GracefulStop()
			srv.GracefulStop()

			return nil
		},
	}

	if err := cmd.Run(appCtx, os.Args); err != nil {
		os.Exit(1)
	}
}
