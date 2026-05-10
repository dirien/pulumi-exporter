package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/kingpin/v2"
)

const testOrgName = "org1"

func TestDefaults(t *testing.T) {
	t.Parallel()

	app := kingpin.New("test", "")
	cfg := RegisterFlags(app)

	_, err := app.Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if cfg.Pulumi.APIURL != "https://api.pulumi.com" {
		t.Errorf("expected api-url %q, got %q", "https://api.pulumi.com", cfg.Pulumi.APIURL)
	}

	if cfg.Pulumi.CollectInterval != 60*time.Second {
		t.Errorf("expected collect-interval %v, got %v", 60*time.Second, cfg.Pulumi.CollectInterval)
	}

	if cfg.Exporters.Endpoint != "localhost:4318" {
		t.Errorf("expected endpoint %q, got %q", "localhost:4318", cfg.Exporters.Endpoint)
	}

	if cfg.Exporters.Protocol != protocolHTTPProtobuf {
		t.Errorf("expected protocol %q, got %q", protocolHTTPProtobuf, cfg.Exporters.Protocol)
	}

	if cfg.Exporters.Insecure != false {
		t.Errorf("expected insecure %v, got %v", false, cfg.Exporters.Insecure)
	}
}

func TestLoadFile(t *testing.T) {
	t.Parallel()

	yamlContent := `
pulumi:
  access-token: "test-token-from-file"
  api-url: "https://custom.pulumi.com"
  organizations:
    - "org1"
    - "org2"
  collect-interval: 30s
otlp:
  endpoint: "otel-collector:4317"
  protocol: "grpc"
  insecure: true
  headers:
    Authorization: "Bearer secret"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg := &Config{}
	if err := cfg.LoadFile(path); err != nil {
		t.Fatalf("LoadFile error: %v", err)
	}

	if cfg.Pulumi.AccessToken != "test-token-from-file" {
		t.Errorf("expected access-token %q, got %q", "test-token-from-file", cfg.Pulumi.AccessToken)
	}

	if cfg.Pulumi.APIURL != "https://custom.pulumi.com" {
		t.Errorf("expected api-url %q, got %q", "https://custom.pulumi.com", cfg.Pulumi.APIURL)
	}

	if len(cfg.Pulumi.Organizations) != 2 || cfg.Pulumi.Organizations[0] != testOrgName || cfg.Pulumi.Organizations[1] != "org2" {
		t.Errorf("expected organizations [org1 org2], got %v", cfg.Pulumi.Organizations)
	}

	if cfg.Pulumi.CollectInterval != 30*time.Second {
		t.Errorf("expected collect-interval %v, got %v", 30*time.Second, cfg.Pulumi.CollectInterval)
	}

	if cfg.Exporters.Endpoint != "otel-collector:4317" {
		t.Errorf("expected endpoint %q, got %q", "otel-collector:4317", cfg.Exporters.Endpoint)
	}

	if cfg.Exporters.Protocol != protocolGRPC {
		t.Errorf("expected protocol %q, got %q", protocolGRPC, cfg.Exporters.Protocol)
	}

	if cfg.Exporters.Insecure != true {
		t.Errorf("expected insecure true, got false")
	}

	if cfg.Exporters.Headers["Authorization"] != "Bearer secret" {
		t.Errorf("expected header Authorization=%q, got %q", "Bearer secret", cfg.Exporters.Headers["Authorization"])
	}
}

func TestEnvVarOverrides(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv().
	t.Setenv("PULUMI_ACCESS_TOKEN", "env-token")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "env-endpoint:4317")

	app := kingpin.New("test", "")
	cfg := RegisterFlags(app)

	_, err := app.Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if cfg.Pulumi.AccessToken != "env-token" {
		t.Errorf("expected access-token %q, got %q", "env-token", cfg.Pulumi.AccessToken)
	}

	if cfg.Exporters.Endpoint != "env-endpoint:4317" {
		t.Errorf("expected endpoint %q, got %q", "env-endpoint:4317", cfg.Exporters.Endpoint)
	}
}

func TestValidateMissingToken(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Pulumi: PulumiConfig{
			Organizations: []string{testOrgName},
		},
		Exporters: ExportersConfig{
			Protocol: protocolHTTPProtobuf,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}

	if err.Error() != "pulumi access token is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateMissingOrgs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Pulumi: PulumiConfig{
			AccessToken:    "some-token",
			MaxConcurrency: 10,
		},
		Exporters: ExportersConfig{
			Protocol: protocolHTTPProtobuf,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing organizations, got nil")
	}

	if err.Error() != "at least one pulumi organization is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateInvalidProtocol(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Pulumi: PulumiConfig{
			AccessToken:    "some-token",
			Organizations:  []string{testOrgName},
			MaxConcurrency: 10,
		},
		Exporters: ExportersConfig{
			Protocol: "invalid",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid protocol, got nil")
	}

	expected := `unsupported OTLP protocol: "invalid" (must be http/protobuf or grpc)`
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Pulumi: PulumiConfig{
			AccessToken:    "pul-token",
			Organizations:  []string{"myorg"},
			MaxConcurrency: 10,
		},
		Exporters: ExportersConfig{
			Protocol: protocolGRPC,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	cfg.Exporters.Protocol = protocolHTTPProtobuf
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for http/protobuf, got: %v", err)
	}
}

func TestParseHeaders(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	cfg.ParseHeaders("key1=value1,key2=value2")

	if len(cfg.Exporters.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(cfg.Exporters.Headers))
	}

	if cfg.Exporters.Headers["key1"] != "value1" {
		t.Errorf("expected key1=%q, got %q", "value1", cfg.Exporters.Headers["key1"])
	}

	if cfg.Exporters.Headers["key2"] != "value2" {
		t.Errorf("expected key2=%q, got %q", "value2", cfg.Exporters.Headers["key2"])
	}
}

func TestParseHeadersEmpty(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	cfg.ParseHeaders("")

	if cfg.Exporters.Headers != nil {
		t.Errorf("expected nil headers for empty input, got %v", cfg.Exporters.Headers)
	}
}

func TestMaxConcurrencyDefault(t *testing.T) {
	t.Parallel()

	app := kingpin.New("test", "")
	cfg := RegisterFlags(app)

	_, err := app.Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if cfg.Pulumi.MaxConcurrency != 10 {
		t.Errorf("expected max-concurrency 10, got %d", cfg.Pulumi.MaxConcurrency)
	}
}

func TestMaxConcurrencyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"too high", 101, true},
		{"minimum", 1, false},
		{"middle", 50, false},
		{"maximum", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Pulumi: PulumiConfig{
					AccessToken:    "token",
					Organizations:  []string{"org"},
					MaxConcurrency: tt.value,
				},
				Exporters: ExportersConfig{
					Protocol: protocolHTTPProtobuf,
				},
			}

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for max-concurrency=%d, got nil", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for max-concurrency=%d: %v", tt.value, err)
			}
		})
	}
}
