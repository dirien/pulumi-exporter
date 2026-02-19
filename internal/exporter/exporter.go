// Package exporter manages the OpenTelemetry MeterProvider for OTLP metric export.
package exporter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials/insecure"
)

// OTLPConfig holds OTLP exporter configuration.
type OTLPConfig struct {
	Endpoint string
	URLPath  string
	Protocol string
	Insecure bool
	Headers  map[string]string
}

// Exporter manages the OTel MeterProvider.
type Exporter struct {
	meterProvider *sdkmetric.MeterProvider
}

// NewExporter creates a new Exporter with an OTLP metric exporter.
func NewExporter(ctx context.Context, cfg *OTLPConfig, version string) (*Exporter, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("pulumi-exporter"),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	var exp sdkmetric.Exporter

	switch cfg.Protocol {
	case "http/protobuf":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
		}
		if cfg.URLPath != "" {
			opts = append(opts, otlpmetrichttp.WithURLPath(cfg.URLPath))
		}

		exp, err = otlpmetrichttp.New(ctx, opts...)
	case "grpc":
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
		}

		exp, err = otlpmetricgrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %q", cfg.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(exp)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	return &Exporter{meterProvider: mp}, nil
}

// Meter returns a named Meter from the MeterProvider.
func (e *Exporter) Meter() metric.Meter {
	return e.meterProvider.Meter("pulumi-exporter")
}

// Shutdown gracefully shuts down the MeterProvider, flushing any remaining metrics.
func (e *Exporter) Shutdown(ctx context.Context) error {
	return e.meterProvider.Shutdown(ctx)
}
