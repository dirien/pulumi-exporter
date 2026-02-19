# pulumi-exporter

Standalone OpenTelemetry metrics exporter for [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/). Polls the Pulumi API on a configurable interval and pushes metrics over OTLP to any observability backend.

```
                                              ┌──OTLP/HTTP──▶ DataDog, NewRelic, Dash0, Prometheus 2.47+
Pulumi Cloud API  ◀──poll──  [pulumi-exporter]┤
                                              └──OTLP/gRPC──▶ Dynatrace, OTel Collector, Grafana Alloy
```

## Install

### Binary (GitHub Releases)

Download the latest release for your platform from the [releases page](https://github.com/dirien/pulumi-exporter/releases). Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

```bash
# macOS / Linux
curl -sSfL https://github.com/dirien/pulumi-exporter/releases/latest/download/pulumi-exporter_linux_amd64.tar.gz | tar xz
chmod +x pulumi-exporter
sudo mv pulumi-exporter /usr/local/bin/
```

All release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) and include an SBOM.

### Docker

Multi-arch images are published to GitHub Container Registry on every release:

```bash
docker pull ghcr.io/dirien/pulumi-exporter:latest
```

The image is built on `cgr.dev/chainguard/static` (distroless, zero CVEs).

```bash
docker run --rm \
  -e PULUMI_ACCESS_TOKEN=pul-xxx \
  -e PULUMI_ORGANIZATIONS=my-org \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4318 \
  -e OTEL_EXPORTER_OTLP_INSECURE=true \
  -p 8080:8080 \
  ghcr.io/dirien/pulumi-exporter:latest
```

### Build from Source

Requires Go 1.24+.

```bash
git clone https://github.com/dirien/pulumi-exporter.git
cd pulumi-exporter
go build -o pulumi-exporter .
```

## Quick Start

### 1. Get a Pulumi access token

Create one at [app.pulumi.com/account/tokens](https://app.pulumi.com/account/tokens).

### 2. Start a local OTel Collector

```bash
docker run -d --name otel-collector -p 4318:4318 otel/opentelemetry-collector:latest
```

### 3. Run the exporter

```bash
export PULUMI_ACCESS_TOKEN=pul-xxx

./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:4318 \
  --otlp.insecure
```

### 4. Verify

```bash
# Health check
curl http://localhost:8080/healthz
# ok
```

Metrics flow through the OTel Collector to whichever backend(s) you've configured.

## Configuration

Configure via CLI flags, environment variables, or a YAML file. Flags take precedence over env vars, and env vars take precedence over the config file.

### Flags and Environment Variables

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--pulumi.access-token` | `PULUMI_ACCESS_TOKEN` | *(required)* | Pulumi Cloud access token |
| `--pulumi.api-url` | `PULUMI_API_URL` | `https://api.pulumi.com` | Pulumi Cloud API base URL |
| `--pulumi.organizations` | `PULUMI_ORGANIZATIONS` | *(required)* | Organizations to monitor (repeatable) |
| `--pulumi.collect-interval` | `PULUMI_COLLECT_INTERVAL` | `60s` | Polling interval |
| `--otlp.endpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | OTLP receiver endpoint |
| `--otlp.protocol` | `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `http/protobuf` or `grpc` |
| `--otlp.insecure` | `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Disable TLS |
| `--otlp.headers` | `OTEL_EXPORTER_OTLP_HEADERS` | *(empty)* | Comma-separated `key=value` pairs |
| `--config.file` | `PULUMI_EXPORTER_CONFIG_FILE` | *(none)* | Path to YAML config file |
| `--web.listen-address` | `PULUMI_EXPORTER_LISTEN_ADDRESS` | `:8080` | Health check listen address |

OTLP environment variable names follow the [OpenTelemetry SDK specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for interoperability with other OTel tools.

### YAML Config File

```yaml
pulumi:
  access-token: "pul-xxx"
  api-url: "https://api.pulumi.com"
  organizations:
    - "my-org"
  collect-interval: 60s

otlp:
  endpoint: "localhost:4318"
  protocol: "http/protobuf"   # or "grpc"
  insecure: false
  headers:
    Authorization: "Bearer <token>"
```

```bash
./pulumi-exporter --config.file=config.yaml
```

See [`config.example.yaml`](config.example.yaml) for a full template.

## Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_stack_resource_count` | Gauge | `org`, `project`, `stack` | Number of resources in a stack |
| `pulumi_update_duration_seconds` | Histogram | `org`, `project`, `stack`, `kind`, `result` | Duration of stack updates (seconds) |
| `pulumi_update_total` | Counter | `org`, `project`, `stack`, `kind`, `result` | Total number of stack updates |
| `pulumi_update_resource_changes` | Counter | `org`, `project`, `stack`, `kind`, `operation` | Resource changes per update |
| `pulumi_deployment_status` | Gauge | `org`, `status` | Deployments by status |
| `pulumi_stack_last_update_timestamp` | Gauge | `org`, `project`, `stack` | Unix timestamp of last update |

### Label Values

- **`kind`**: `update`, `preview`, `destroy`, `refresh`, `import`
- **`result`**: `succeeded`, `failed`, `in-progress`
- **`operation`**: `create`, `update`, `delete`, `same`, `replace`
- **`status`**: `running`, `succeeded`, `failed`, `not-started`, `accepted`

### Histogram Buckets

`pulumi_update_duration_seconds` uses bucket boundaries tuned for IaC operations:

```
5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m
```

## Backend Setup

### Prometheus (v2.47+ with native OTLP receiver)

```bash
# prometheus.yml
otlp:
  protocols:
    http:
      endpoint: "0.0.0.0:4318"

# Start Prometheus
prometheus --config.file=prometheus.yml --web.enable-otlp-receiver
```

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:4318 \
  --otlp.insecure
```

### Grafana Alloy / OTel Collector

Point the exporter at your Alloy or Collector's OTLP receiver:

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=alloy:4317 \
  --otlp.protocol=grpc \
  --otlp.insecure
```

### DataDog

Use the DataDog Agent's OTLP ingestion or send directly:

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<DD_AGENT_HOST>:4318 \
  --otlp.insecure
```

### NewRelic

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=otlp.nr-data.net:4318 \
  --otlp.headers="api-key=<NEWRELIC_LICENSE_KEY>"
```

### Dynatrace

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<ENV_ID>.live.dynatrace.com:4317 \
  --otlp.protocol=grpc \
  --otlp.headers="Authorization=Api-Token <DT_TOKEN>"
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pulumi-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pulumi-exporter
  template:
    metadata:
      labels:
        app: pulumi-exporter
    spec:
      containers:
        - name: pulumi-exporter
          image: ghcr.io/dirien/pulumi-exporter:latest
          args:
            - --pulumi.organizations=my-org
            - --otlp.endpoint=otel-collector.monitoring:4318
            - --otlp.insecure
          env:
            - name: PULUMI_ACCESS_TOKEN
              valueFrom:
                secretKeyRef:
                  name: pulumi-credentials
                  key: access-token
          ports:
            - name: health
              containerPort: 8080
          livenessProbe:
            httpGet:
              path: /healthz
              port: health
            initialDelaySeconds: 5
          readinessProbe:
            httpGet:
              path: /readyz
              port: health
            initialDelaySeconds: 5
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              memory: 128Mi
```

## Development

### Prerequisites

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2

### Makefile

The project ships a `Makefile` that wraps all common operations. Run `make help` to see available targets:

```bash
make help
```

| Target | Description |
|--------|-------------|
| `make` | Lint, test, and build (default) |
| `make build` | Build the binary with version ldflags |
| `make test` | Run all tests |
| `make test-race` | Run tests with Go race detector |
| `make test-cover` | Generate HTML coverage report |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code (gofumpt + goimports) |
| `make vet` | Run `go vet` |
| `make generate` | Download Pulumi OpenAPI spec and regenerate Go client |
| `make download-spec` | Download the latest Pulumi Cloud OpenAPI spec |
| `make release-snapshot` | GoReleaser dry run (no publish) |
| `make docker` | Build Docker image locally |
| `make deps` | Download and tidy Go module dependencies |
| `make tools` | Install development tools (oapi-codegen) |
| `make clean` | Remove build artifacts |

### Build and Test

```bash
# Build with version info injected
make build

# Run all tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage report
make test-cover
```

### Lint

```bash
make lint
```

The project uses a strict linter configuration (`.golangci.yaml`) with gosec, revive, gocritic, gocyclo, and more. Formatting is enforced by gofumpt and goimports.

### Regenerating the API Client

The project includes a generated Go client from the [Pulumi Cloud OpenAPI spec](https://www.pulumi.com/blog/announcing-openapi-support-pulumi-cloud/). The spec is published at `https://api.pulumi.com/api/openapi/pulumi-spec.json`.

To regenerate the client after Pulumi updates their API:

```bash
make generate
```

This downloads the latest spec and runs [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to produce `internal/pulumiapi/client.gen.go`. The generation is scoped to only the 5 operations the exporter uses (configured in `oapi-codegen.yaml`):

- `ListUserStacks` -- `GET /api/user/stacks`
- `GetStackResourceCount` -- `GET /api/stacks/{org}/{project}/{stack}/resources/count`
- `GetStackUpdates` -- `GET /api/stacks/{org}/{project}/{stack}/updates`
- `ListStackDeploymentsHandlerV2` -- `GET /api/stacks/{org}/{project}/{stack}/deployments`
- `ListOrgDeployments` -- `GET /api/orgs/{org}/deployments`

The hand-written client in `internal/client/` is the one used at runtime. The generated client in `internal/pulumiapi/` serves as a reference and can be wired in as a replacement when the full typed client is preferred.

### Project Structure

```
pulumi-exporter/
├── main.go                           # Delegates to cmd/pulumiexporter
├── Makefile                          # Build, test, lint, generate targets
├── oapi-codegen.yaml                 # OpenAPI code generation config
├── cmd/pulumiexporter/
│   └── main.go                       # CLI flags, wiring, signal handling
├── internal/
│   ├── client/                       # Hand-written Pulumi Cloud API client
│   │   ├── client.go                 # PulumiClient struct, doRequest helper
│   │   ├── types.go                  # API response types
│   │   ├── stacks.go                 # ListStacks, GetResourceCount
│   │   ├── updates.go                # ListUpdates
│   │   ├── deployments.go            # ListDeployments, ListOrgDeployments
│   │   └── client_test.go            # httptest-based tests
│   ├── pulumiapi/                    # Generated client from OpenAPI spec
│   │   └── client.gen.go             # oapi-codegen output (DO NOT EDIT)
│   ├── config/                       # CLI flags + env vars + YAML config
│   │   ├── config.go                 # Config struct, RegisterFlags, LoadFile, Validate
│   │   └── config_test.go
│   ├── collector/                    # Metrics collection logic
│   │   ├── collector.go              # PulumiAPI interface, ticker loop
│   │   ├── instruments.go            # OTel instrument definitions
│   │   ├── stack.go                  # Per-stack collection
│   │   ├── deployments.go            # Org deployment collection
│   │   └── collector_test.go         # Mock API + ManualReader tests
│   └── exporter/                     # OTel MeterProvider setup
│       ├── exporter.go               # OTLP HTTP/gRPC exporter creation
│       └── exporter_test.go
├── Dockerfile                        # Chainguard static base
├── .goreleaser.yaml                  # Multi-arch builds, signing, SBOM
├── .golangci.yaml                    # Linter configuration
└── .github/workflows/
    ├── ci.yaml                       # Build, test, release
    └── lint.yaml                     # golangci-lint
```

All application code lives under `internal/` (compiler-enforced import restriction). The `collector` package depends on a `PulumiAPI` interface rather than the concrete client, making it straightforward to mock in tests.

### Running Locally

Start a local OTel Collector to receive metrics:

```bash
docker run -d --name otel-collector -p 4318:4318 \
  otel/opentelemetry-collector:latest
```

Run the exporter:

```bash
PULUMI_ACCESS_TOKEN=pul-xxx make run \
  ARGS="--pulumi.organizations=my-org --otlp.endpoint=localhost:4318 --otlp.insecure"
```

Or directly with `go run`:

```bash
PULUMI_ACCESS_TOKEN=pul-xxx go run . \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:4318 \
  --otlp.insecure
```

### GoReleaser Dry Run

Test the full release pipeline without publishing:

```bash
make release-snapshot
```

## Contributing

1. Fork and clone the repository
2. Install tools: `make tools`
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Make your changes and add tests
5. Verify:
   ```bash
   make test-race
   make lint
   ```
6. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add something useful"
   ```
7. Push and open a pull request

### Adding a New Metric

1. Add the instrument to `internal/collector/instruments.go`
2. Record values in `internal/collector/stack.go` or `internal/collector/deployments.go`
3. If the metric requires new API data, add the endpoint to `internal/client/`
4. Add tests

### Adding a New API Endpoint

1. Add response types to `internal/client/types.go`
2. Add the method to the appropriate file in `internal/client/`
3. Add the method to the `PulumiAPI` interface in `internal/collector/collector.go`
4. Add httptest-based tests to `internal/client/client_test.go`
5. Optionally, add the operationId to `oapi-codegen.yaml` and run `make generate` to update the generated client

### Regenerating After Pulumi API Changes

When Pulumi releases API changes:

```bash
# Regenerate the typed client
make generate

# Verify everything compiles
make build

# Run tests
make test
```

## License

Apache-2.0
