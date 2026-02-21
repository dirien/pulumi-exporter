# Agent Guide — pulumi-exporter

Standalone OpenTelemetry metrics exporter for Pulumi Cloud. Polls the Pulumi API and pushes metrics over OTLP.

## Build, Test, Lint

```bash
make build          # CGO_ENABLED=0 go build with ldflags
make test           # go test ./...
make test-race      # go test -race ./...
make lint           # golangci-lint run (v2, strict config)
make fmt            # gofumpt + goimports via golangci-lint
make generate       # Download Pulumi OpenAPI spec + regenerate Go client
```

Always run `make lint` and `make test-race` before considering work done. The linter config (`.golangci.yaml`) enforces:

- **Style**: revive, gocritic, goconst, gofumpt, goimports, tagliatelle (snake_case JSON, kebab-case YAML), misspell, dupword
- **Bugs**: errcheck, errorlint, copyloopvar, unconvert, unparam, wastedassign, bodyclose
- **Security**: gosec (G117 excluded — config field name false positive)
- **Complexity**: gocyclo (max 15), prealloc
- **Tests**: tparallel, thelper
- **Hygiene**: nolintlint, forbidigo

## Project Layout

```
main.go                                → delegates to cmd/pulumiexporter
cmd/pulumiexporter/main.go             → CLI flags, wiring, signal handling
internal/pulumiapi/client.gen.go        → GENERATED — never edit manually
internal/client/client.go               → typed wrappers around generated client
internal/client/types.go                → response structs used by collector
internal/collector/collector.go         → PulumiAPI interface, ticker loop, collect()
internal/collector/instruments.go       → all OTel metric instrument definitions
internal/collector/stack.go             → per-stack metric collection
internal/collector/deployments.go       → org deployment collection
internal/collector/org.go               → org-level collection (members, teams, policies, etc.)
internal/collector/collector_test.go    → mock API + OTel ManualReader tests
internal/config/                        → CLI flags + env vars + YAML config
internal/exporter/                      → OTel MeterProvider / OTLP exporter setup
internal/appinfo/                       → build-time version info (ldflags)
dashboards/pulumi-exporter.json         → Grafana dashboard JSON (26 panels, 17 metrics)
deploy/docker-compose/                  → Prometheus + Grafana + exporter stack
charts/pulumi-exporter/                 → Helm chart (templates, values, ci test values)
.github/configs/                        → ct-lint, cr, lintconf YAML configs
.github/workflows/helm-publish.yaml     → Chart publish (chart-releaser + OCI + cosign)
.github/workflows/lint-and-test.yaml    → CT lint + Trivy + kind install on PRs
```

## Go Version and Module

- Go 1.24+, module `github.com/pulumi-labs/pulumi-exporter`
- Dependencies: OTel SDK, oapi-codegen runtime, kingpin CLI, golang.org/x/sync

## Coding Conventions

### Error Handling
- Wrap all errors with context: `fmt.Errorf("doing X: %w", err)`
- In collector functions, log errors and return early — never panic
- Check `resp.StatusCode()` and `resp.JSON200 != nil` on every API call

### Context Propagation
- Every blocking or API-calling function takes `context.Context` as first parameter
- The collection loop applies a timeout of 90% of the collect interval (min 10s)
- Respect context cancellation in all goroutines

### Concurrency Patterns
- Per-stack collection uses a semaphore (`chan struct{}`) + `sync.WaitGroup`
- Org-level collection uses `golang.org/x/sync/errgroup` with bounded parallelism
- Org-level metric sub-collectors run in parallel via `errgroup.Go()`
- Never start a goroutine without a clear shutdown path

### Interface Design
- `PulumiAPI` interface in `collector.go` defines all API methods the collector needs
- Keep interfaces small — one method per API operation
- Mock implementations (`mockAPI`, `slowMockAPI`) live in `collector_test.go`

### Naming
- Package-level types use exported PascalCase
- Struct fields in `types.go` are exported (used across packages)
- Internal instrument fields in `Instruments` are unexported
- Metric names follow `pulumi_<scope>_<name>` pattern (e.g. `pulumi_org_member_count`)

### Testing
- Tests use `testing.T` with `t.Parallel()` on every test function
- Use OTel `sdkmetric.ManualReader` to verify metric recording
- Helper `newTestCollector(t, api)` creates a wired-up collector for tests
- When adding a new API method: add stubs to both `mockAPI` and `slowMockAPI`

## Adding a New Metric

1. Add the OTel instrument to `internal/collector/instruments.go` (in the appropriate init function — `NewInstruments` for stack-level, `newOrgInstruments` for org-level)
2. Record values in the relevant collector file (`stack.go`, `deployments.go`, or `org.go`)
3. If the metric needs a new API endpoint:
   - Add the operationId to `oapi-codegen.yaml`
   - Run `make generate` to regenerate `internal/pulumiapi/client.gen.go`
   - Add a response type to `internal/client/types.go`
   - Add a wrapper method to `internal/client/client.go`
   - Add the method to the `PulumiAPI` interface in `internal/collector/collector.go`
   - Add stubs to `mockAPI` and `slowMockAPI` in `internal/collector/collector_test.go`
4. Add Grafana panels to `dashboards/pulumi-exporter.json`
5. Update the metrics table in `README.md`

### Cyclomatic Complexity
The gocyclo limit is 15. `NewInstruments` delegates org-level instruments to `newOrgInstruments` to stay under the limit. Follow this split pattern when adding instruments.

## OpenAPI Code Generation

The generated client in `internal/pulumiapi/client.gen.go` is produced by oapi-codegen from the Pulumi Cloud OpenAPI spec. Config lives in `oapi-codegen.yaml`. Only operations listed in `include-operation-ids` are generated. The response type suffix is `Resp` (e.g. `GetPolicyResultsMetadataResp`).

Never edit `client.gen.go` manually — always modify `oapi-codegen.yaml` and run `make generate`.

**Never create hand-rolled HTTP/REST calls** (e.g. `http.NewRequest`, `http.Get`, raw `net/http` usage) to the Pulumi Cloud API. Always use the OpenAPI-generated client in `internal/pulumiapi/`. If an endpoint is not yet available, add its operationId to `oapi-codegen.yaml`, run `make generate`, then write a typed wrapper in `internal/client/client.go`. This ensures consistent auth, error handling, and response parsing across all API calls.

## Helm Chart

The chart lives under `charts/pulumi-exporter/`. It follows the same patterns as [pulumi-labs/minecraft-prometheus-exporter](https://github.com/pulumi-labs/minecraft-prometheus-exporter).

### Chart Structure

```
charts/pulumi-exporter/
├── Chart.yaml              # Metadata, appVersion, ArtifactHub annotations
├── values.yaml             # All configurable values (helm-docs comments)
├── README.md.gotmpl        # helm-docs template
├── README.md               # Generated — do not edit by hand
├── .helmignore
├── ci/
│   └── ct-values.yaml      # Test values for ct install (dummy token, pullPolicy: Never)
└── templates/
    ├── _helpers.tpl         # name, fullname, chart, image, serviceAccountName, secretName
    ├── NOTES.txt
    ├── deployment.yaml
    ├── service.yaml
    ├── serviceaccount.yaml
    ├── secret.yaml          # Conditional — skipped when existingSecret is set
    ├── servicemonitor.yaml  # Optional — requires Prometheus Operator CRD
    └── tests/
        └── test-connection.yaml
```

### Makefile Targets

```bash
make helm-lint      # helm lint
make helm-template  # Render templates with test values
make helm-docs      # Regenerate chart README via helm-docs
make ct-lint        # chart-testing lint (schema, YAML, maintainers)
make helm-test      # All of the above in sequence
make kind-create    # Create a kind cluster for e2e testing
make ct-install     # Build image, load into kind, ct install
make kind-delete    # Delete the kind cluster
make helm-test-e2e  # kind-create + ct-install + kind-delete
```

### Versioning

- `Chart.yaml` `version` = chart version (bump when templates/values change)
- `Chart.yaml` `appVersion` = application image version
- `_helpers.tpl` `pulumi-exporter.image` defaults tag to `.Chart.AppVersion` when `image.tag` is empty
- Update `artifacthub.io/images` annotation when bumping `appVersion`

### CI/CD Workflows

- **`helm-publish.yaml`** — triggers on push to `main` touching `charts/**`. Runs ArtifactHub lint, chart-releaser (GH Pages), pushes OCI to `ghcr.io`, cosign signs.
- **`lint-and-test.yaml`** — triggers on PRs touching `charts/**`. Runs Trivy IaC scan, ct lint, ArtifactHub lint, kind cluster + ct install.
- Config files: `.github/configs/cr.yaml`, `ct-lint.yaml`, `lintconf.yaml`

### Secret Management

The chart supports two modes for the Pulumi access token:
- `pulumiAccessToken: "pul-xxx"` — chart creates a Secret (dev/testing)
- `existingSecret: "my-secret"` — references a pre-existing Secret with key `access-token` (production)

### Modifying the Chart

1. Edit templates/values as needed
2. Run `make helm-docs` to regenerate `charts/pulumi-exporter/README.md`
3. Run `make helm-test` to validate (lint, template, ct lint)
4. Bump `version` in `Chart.yaml` if templates/values changed
5. Bump `appVersion` + `artifacthub.io/images` if the app image changed

## Docker Compose Stack

```bash
cd deploy/docker-compose
cp .env.example .env    # set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS
docker compose up --build -d
```

- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Dashboard JSON: `dashboards/pulumi-exporter.json` (mounted into Grafana via docker-compose volume)

## Key Files to Read First

If you're new to this codebase, read in this order:
1. `internal/collector/collector.go` — the central interface and collection loop
2. `internal/collector/instruments.go` — all metric definitions
3. `internal/client/client.go` — how API calls are made
4. `internal/collector/org.go` — org-level collection pattern (errgroup)
5. `internal/collector/collector_test.go` — mock setup and test patterns
