package exporter

import (
	"context"
	"testing"
)

func TestNewExporterHTTP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4318",
		Protocol: protocolHTTPProtobuf,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}
	defer func() { _ = exp.Shutdown(ctx) }()

	if m := exp.Meter(); m == nil {
		t.Fatal("Meter() returned nil")
	}
}

func TestNewExporterGRPC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4317",
		Protocol: protocolGRPC,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}
	defer func() { _ = exp.Shutdown(ctx) }()

	if m := exp.Meter(); m == nil {
		t.Fatal("Meter() returned nil")
	}
}

func TestNewExporterInvalidProtocol(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4317",
		Protocol: "invalid",
		Insecure: true,
	}

	_, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err == nil {
		t.Fatal("NewExporter() expected error for invalid protocol, got nil")
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4318",
		Protocol: protocolHTTPProtobuf,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}

	// Shutdown may return a connection error when no collector is running,
	// which is expected in tests. We just verify it does not panic.
	_ = exp.Shutdown(ctx)
}
