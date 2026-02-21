// Package pulumiexporter provides the entrypoint for the pulumi-exporter application.
package pulumiexporter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"

	"github.com/pulumi-labs/pulumi-exporter/internal/appinfo"
	"github.com/pulumi-labs/pulumi-exporter/internal/client"
	"github.com/pulumi-labs/pulumi-exporter/internal/collector"
	"github.com/pulumi-labs/pulumi-exporter/internal/config"
	"github.com/pulumi-labs/pulumi-exporter/internal/exporter"
)

// Main is the entrypoint for the pulumi-exporter application.
func Main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	app := kingpin.New("pulumi-exporter", "OpenTelemetry Pulumi Cloud metrics exporter.")
	app.Version(appinfo.Print("pulumi-exporter"))
	app.HelpFlag.Short('h')

	cfg := config.RegisterFlags(app)

	configFile := app.Flag("config.file", "Path to configuration file.").
		Envar("PULUMI_EXPORTER_CONFIG_FILE").
		String()

	headersRaw := app.Flag("otlp.headers", "OTLP headers as comma-separated key=value pairs.").
		Envar("OTEL_EXPORTER_OTLP_HEADERS").
		String()

	listenAddr := app.Flag("web.listen-address", "Address to listen on for health checks.").
		Default(":8080").
		Envar("PULUMI_EXPORTER_LISTEN_ADDRESS").
		String()

	if _, err := app.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	// Load config file if specified.
	if *configFile != "" {
		if err := cfg.LoadFile(*configFile); err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Parse headers and normalize organizations.
	cfg.ParseHeaders(*headersRaw)
	cfg.NormalizeOrganizations()

	// Validate configuration.
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("starting pulumi-exporter", "version", appinfo.Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OTel exporter.
	otlpCfg := &exporter.OTLPConfig{
		Endpoint: cfg.Exporters.Endpoint,
		URLPath:  cfg.Exporters.URLPath,
		Protocol: cfg.Exporters.Protocol,
		Insecure: cfg.Exporters.Insecure,
		Headers:  cfg.Exporters.Headers,
	}

	exp, err := exporter.NewExporter(ctx, otlpCfg, appinfo.Version)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create Pulumi API client backed by the generated OpenAPI client.
	apiClient, err := client.NewClient(cfg.Pulumi.APIURL, cfg.Pulumi.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Create collector.
	coll, err := collector.NewCollector(apiClient, cfg, exp.Meter(), logger)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	// Start collector in background.
	go func() {
		if err := coll.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("collector error", "error", err)
		}
	}()

	// Start health check server.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("health check server listening", "address", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health check server error", "error", err)
		}
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("health check server shutdown error", "error", err)
	}

	if err := exp.Shutdown(shutdownCtx); err != nil {
		logger.Error("exporter shutdown error", "error", err)
	}

	logger.Info("pulumi-exporter stopped")
	return nil
}
