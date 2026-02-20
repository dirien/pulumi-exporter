# pulumi-exporter

An OpenTelemetry metrics exporter for [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/). It polls the Pulumi API on a schedule and pushes metrics over OTLP to whatever backend you use.

```mermaid
graph LR
    A[Pulumi Cloud API] -->|poll| B[pulumi-exporter]
    B -->|OTLP/HTTP| C[Prometheus 2.47+]
    B -->|OTLP/HTTP| D[DataDog]
    B -->|OTLP/HTTP| E[NewRelic]
    B -->|OTLP/gRPC| F[Dynatrace]
    B -->|OTLP/gRPC| G[OTel Collector]
    B -->|OTLP/gRPC| H[Grafana Alloy]
```

The API client is generated from the official [Pulumi Cloud OpenAPI spec](https://www.pulumi.com/blog/announcing-openapi-support-pulumi-cloud/) using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). The image is built on `cgr.dev/chainguard/static` (distroless, zero CVEs). All release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) and include an SBOM.

## Quick start

The fastest way to get metrics flowing is Docker Compose. It spins up Prometheus, Grafana, and the exporter with a pre-built dashboard.

```bash
cd deploy/docker-compose
cp .env.example .env
# Set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS in .env
make compose-up
```

Open Grafana at http://localhost:3000 (admin / admin). The dashboard loads automatically.

## Install

### Helm (Kubernetes)

```bash
helm install pulumi-exporter oci://ghcr.io/dirien/charts/pulumi-exporter \
  --set existingSecret=pulumi-credentials \
  --set "pulumiOrganizations={my-org}" \
  --set otlp.endpoint=otel-collector:4318 \
  --set otlp.insecure=true
```

See the [chart README](charts/pulumi-exporter/README.md) for all values.

### Docker

```bash
docker run --rm \
  -e PULUMI_ACCESS_TOKEN=pul-xxx \
  -e PULUMI_ORGANIZATIONS=my-org \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4318 \
  -e OTEL_EXPORTER_OTLP_INSECURE=true \
  ghcr.io/dirien/pulumi-exporter:latest
```

### Binary

```bash
curl -sSfL https://github.com/dirien/pulumi-exporter/releases/latest/download/pulumi-exporter_linux_amd64.tar.gz | tar xz
./pulumi-exporter --pulumi.organizations=my-org --otlp.endpoint=localhost:4318 --otlp.insecure
```

Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

## Multi-org support

The exporter can monitor multiple Pulumi organizations from a single instance. Pass them as a comma-separated list:

```bash
PULUMI_ORGANIZATIONS=my-org,another-org,third-org
```

Every metric includes an `org` label, so you can filter and compare across organizations in your dashboards. The bundled Grafana dashboard has a multi-select Organization dropdown that looks like this:

```mermaid
block-beta
    columns 4
    block:header:4
        dropdown["Organization: my-org | another-org | All"]
    end
    block:row1a
        m1["Members\n42\nmy-org"]
    end
    block:row1b
        t1["Teams\n8\nmy-org"]
    end
    block:row1c
        e1["ESC Envs\n15\nmy-org"]
    end
    block:row1d
        p1["Policy Packs\n3\nmy-org"]
    end
    block:row2a
        m2["Members\n31\nanother-org"]
    end
    block:row2b
        t2["Teams\n5\nanother-org"]
    end
    block:row2c
        e2["ESC Envs\n7\nanother-org"]
    end
    block:row2d
        p2["Policy Packs\n12\nanother-org"]
    end
```

Each org's data is collected in parallel. See [docs/configuration.md](docs/configuration.md) for tuning concurrency and intervals when monitoring large organizations.

## Metrics

17 metrics across stacks and organizations:

| Scope | Metrics |
|-------|---------|
| Stack | `resource_count`, `last_update_timestamp`, `update_total`, `update_duration_seconds`, `update_resource_changes`, `deployment_status` |
| Organization | `member_count`, `team_count`, `environment_count`, `policy_group_count`, `policy_pack_count`, `policy_violations`, `neo_task_count` |
| Compliance | `policy_total`, `policy_with_issues`, `governed_resources_total`, `governed_resources_with_issues` |

All metric names are prefixed with `pulumi_` (stack-level) or `pulumi_org_` (org-level). Full details with types, labels, and histogram buckets in [docs/metrics.md](docs/metrics.md).

## Makefile

Run `make help` to see everything. Here are the ones you'll use most:

| Target | What it does |
|--------|-------------|
| `make` | Lint, test, build |
| `make build` | Build the binary |
| `make test-race` | Tests with race detector |
| `make lint` | golangci-lint |
| `make generate` | Regenerate the OpenAPI client |
| `make helm-test` | Lint + template + helm-docs + ct lint |
| `make helm-test-e2e` | Full Helm e2e: kind cluster + image build + ct install |
| `make compose-up` | Start local Prometheus + Grafana + exporter |
| `make compose-down` | Stop the local stack |
| `make compose-logs` | Tail exporter logs |
| `make compose-restart` | Rebuild and restart the exporter container |

## Documentation

| | |
|---|---|
| [Configuration](docs/configuration.md) | Flags, env vars, YAML config, multi-org, large orgs |
| [Metrics reference](docs/metrics.md) | All 17 metrics with types, labels, histogram buckets |
| [Backend setup](docs/backends.md) | Prometheus, Grafana Alloy, DataDog, NewRelic, Dynatrace |
| [Kubernetes and Helm](docs/kubernetes.md) | Helm chart, raw manifests, chart CI/CD |
| [Development](docs/development.md) | Build, test, project structure, OpenAPI generation, contributing |

## License

Apache-2.0
