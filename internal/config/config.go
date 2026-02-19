// Package config provides configuration loading from flags, env vars, and YAML files.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"gopkg.in/yaml.v3"
)

// Config holds the complete application configuration.
type Config struct {
	Pulumi    PulumiConfig    `yaml:"pulumi"`
	Exporters ExportersConfig `yaml:"otlp"`
}

// PulumiConfig holds Pulumi Cloud API configuration.
type PulumiConfig struct {
	AccessToken     string        `yaml:"access-token"`
	APIURL          string        `yaml:"api-url"`
	Organizations   []string      `yaml:"organizations"`
	CollectInterval time.Duration `yaml:"collect-interval"`
}

// ExportersConfig holds exporter configuration.
type ExportersConfig struct {
	Endpoint string            `yaml:"endpoint"`
	URLPath  string            `yaml:"url-path"`
	Protocol string            `yaml:"protocol"`
	Insecure bool              `yaml:"insecure"`
	Headers  map[string]string `yaml:"headers"`
}

// RegisterFlags registers CLI flags on the given kingpin application and returns a Config.
// Call this before kingpin.Parse(). After Parse(), the Config will be populated with
// flag values, env var overrides, and defaults.
func RegisterFlags(app *kingpin.Application) *Config {
	cfg := &Config{}

	app.Flag("pulumi.access-token", "Pulumi Cloud access token.").
		Envar("PULUMI_ACCESS_TOKEN").
		StringVar(&cfg.Pulumi.AccessToken)

	app.Flag("pulumi.api-url", "Pulumi Cloud API URL.").
		Default("https://api.pulumi.com").
		Envar("PULUMI_API_URL").
		StringVar(&cfg.Pulumi.APIURL)

	app.Flag("pulumi.organizations", "Pulumi organizations to monitor.").
		Envar("PULUMI_ORGANIZATIONS").
		StringsVar(&cfg.Pulumi.Organizations)

	app.Flag("pulumi.collect-interval", "Metrics collection interval.").
		Default("60s").
		Envar("PULUMI_COLLECT_INTERVAL").
		DurationVar(&cfg.Pulumi.CollectInterval)

	app.Flag("otlp.endpoint", "OTLP exporter endpoint.").
		Default("localhost:4318").
		Envar("OTEL_EXPORTER_OTLP_ENDPOINT").
		StringVar(&cfg.Exporters.Endpoint)

	app.Flag("otlp.protocol", "OTLP exporter protocol (http/protobuf or grpc).").
		Default("http/protobuf").
		Envar("OTEL_EXPORTER_OTLP_PROTOCOL").
		StringVar(&cfg.Exporters.Protocol)

	app.Flag("otlp.url-path", "OTLP metrics URL path (e.g. /api/v1/otlp/v1/metrics for Prometheus).").
		Envar("OTEL_EXPORTER_OTLP_METRICS_URL_PATH").
		StringVar(&cfg.Exporters.URLPath)

	app.Flag("otlp.insecure", "Disable TLS for OTLP exporter.").
		Default("false").
		Envar("OTEL_EXPORTER_OTLP_INSECURE").
		BoolVar(&cfg.Exporters.Insecure)

	return cfg
}

// LoadFile loads a YAML configuration file and overlays it onto the existing config.
func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from user-provided config flag
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	return nil
}

// ParseHeaders parses a comma-separated list of key=value pairs into the Headers map.
func (c *Config) ParseHeaders(raw string) {
	if raw == "" {
		return
	}

	c.Exporters.Headers = make(map[string]string)

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			c.Exporters.Headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
}

// Validate checks that required configuration values are present.
func (c *Config) Validate() error {
	if c.Pulumi.AccessToken == "" {
		return fmt.Errorf("pulumi access token is required")
	}

	if len(c.Pulumi.Organizations) == 0 {
		return fmt.Errorf("at least one pulumi organization is required")
	}

	switch c.Exporters.Protocol {
	case "http/protobuf", "grpc":
		// valid
	default:
		return fmt.Errorf("unsupported OTLP protocol: %q (must be http/protobuf or grpc)", c.Exporters.Protocol)
	}

	return nil
}
