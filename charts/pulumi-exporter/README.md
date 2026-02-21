# Pulumi Cloud Exporter

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

![Pulumi](https://img.shields.io/badge/Pulumi-8A3391?style=for-the-badge&logo=pulumi&logoColor=white)
![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-3D348B?style=for-the-badge&logo=opentelemetry&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Docker](https://img.shields.io/badge/docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Helm](https://img.shields.io/badge/helm-0F1689?style=for-the-badge&logo=helm&logoColor=white)

## Description

A Helm chart for the Pulumi Cloud OpenTelemetry metrics exporter

Standalone OpenTelemetry metrics exporter for [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/).
Polls the Pulumi API on a configurable interval and pushes metrics over OTLP to any observability backend
(Prometheus, Grafana, Datadog, Dynatrace, NewRelic, etc.).

## Usage (via OCI Registry)

To install the chart using the OCI artifact, run:

```bash
helm install pulumi-exporter oci://ghcr.io/pulumi-labs/charts/pulumi-exporter --version 0.1.0
```

Requires Helm >= 3.8.0.

## Usage (via Helm Repository)

```bash
helm repo add pulumi-exporter https://pulumi-labs.github.io/pulumi-exporter
helm repo update
```

### Installing the Chart

```bash
helm install pulumi-exporter pulumi-exporter/pulumi-exporter --version 0.1.0
```

### Example: Minimal Installation

```bash
helm install pulumi-exporter pulumi-exporter/pulumi-exporter \
  --set pulumiAccessToken=pul-xxxxxxxxxxxx \
  --set "pulumiOrganizations={my-org}" \
  --set otlp.endpoint=otel-collector:4318 \
  --set otlp.insecure=true
```

### Example: Using an Existing Secret

Create the secret first:

```bash
kubectl create secret generic pulumi-credentials \
  --from-literal=access-token=pul-xxxxxxxxxxxx
```

Then install with:

```bash
helm install pulumi-exporter pulumi-exporter/pulumi-exporter \
  --set existingSecret=pulumi-credentials \
  --set "pulumiOrganizations={my-org,another-org}" \
  --set otlp.endpoint=otel-collector:4318 \
  --set otlp.insecure=true
```

### Uninstalling the Chart

```bash
helm uninstall pulumi-exporter
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Set the affinity for the pod. |
| collectInterval | string | `"60s"` | Polling interval for metrics collection |
| existingSecret | string | `""` | Use an existing Secret for the Pulumi access token. The secret must contain a key named `access-token`. |
| extraEnv | list | `[]` | Extra environment variables |
| fullnameOverride | string | `""` | String to override the default generated fullname |
| image.pullPolicy | string | `"IfNotPresent"` | The docker image pull policy |
| image.repository | string | `"ghcr.io/pulumi-labs/pulumi-exporter"` | The docker image repository to use |
| image.tag | string | `""` | The docker image tag to use @default Chart appVersion |
| maxConcurrency | int | `10` | Maximum number of concurrent stack API calls (1-100) |
| nameOverride | string | `""` | String to override the default generated name |
| nodeSelector | object | `{}` | Set the node selector for the pod. |
| otlp.endpoint | string | `""` | OTLP exporter endpoint (host:port) |
| otlp.headers | string | `""` | Additional OTLP headers as comma-separated key=value pairs |
| otlp.insecure | bool | `false` | Disable TLS for OTLP endpoint |
| otlp.protocol | string | `"http/protobuf"` | OTLP protocol: "http/protobuf" or "grpc" |
| otlp.urlPath | string | `""` | Custom URL path for OTLP metrics endpoint |
| podAnnotations | object | `{}` | Annotations for the pods |
| podSecurityContext.fsGroup | int | `10003` |  |
| podSecurityContext.runAsGroup | int | `10003` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.runAsUser | int | `10003` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| pulumiAPIURL | string | `"https://api.pulumi.com"` | Pulumi Cloud API base URL |
| pulumiAccessToken | string | `""` | Pulumi Cloud access token (required). Use existingSecret instead for production. |
| pulumiOrganizations | list | `[]` | List of Pulumi organizations to monitor (required) |
| replicaCount | int | `1` | Number of replicas |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"50m","memory":"64Mi"}}` | Set the resources requests and limits |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.privileged | bool | `false` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsGroup | int | `10003` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.runAsUser | int | `10003` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| service.annotations | object | `{}` | Additional annotations |
| service.port | int | `8080` | Health check service port |
| service.type | string | `"ClusterIP"` | Specifies what type of Service should be created |
| serviceAccount.create | bool | `true` | Specifies whether a ServiceAccount should be created |
| serviceAccount.name | string | `nil` | The name of the ServiceAccount to use. If not set and create is true, a name is generated using the fullname template |
| serviceMonitor.enabled | bool | `false` | When set true then use a ServiceMonitor to configure scraping |
| tolerations | list | `[]` | Set the tolerations for the pod. |

**Homepage:** <https://github.com/pulumi-labs/pulumi-exporter/>

## Source Code

* <https://github.com/pulumi-labs/pulumi-exporter/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| pulumi-labs | <engin.diri@ediri.de> |  |
