package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds observability configuration.
type Config struct {
	ServiceName  string
	OTLPEndpoint string
	MetricsAddr  string
}

// Setup initializes OpenTelemetry tracing and metrics providers.
// The returned shutdown function should be called on application exit.
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var firstErr error
		for _, fn := range shutdownFuncs {
			if e := fn(ctx); e != nil && firstErr == nil {
				firstErr = e
			}
		}
		return firstErr
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return shutdown, err
	}

	// Trace provider
	if cfg.OTLPEndpoint != "" {
		exporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return shutdown, err
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Metrics provider (Prometheus)
	promExp, err := promexporter.New()
	if err != nil {
		return shutdown, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExp),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)

	// Start metrics HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	metricsSrv := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()
	shutdownFuncs = append(shutdownFuncs, metricsSrv.Shutdown)

	return shutdown, nil
}
