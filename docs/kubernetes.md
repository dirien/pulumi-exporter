# Kubernetes & Helm

## Helm Chart (recommended)

### OCI Registry

```bash
helm install pulumi-exporter oci://ghcr.io/pulumi-labs/charts/pulumi-exporter \
  --set existingSecret=pulumi-credentials \
  --set "pulumiOrganizations={my-org}" \
  --set otlp.endpoint=otel-collector:4318 \
  --set otlp.insecure=true
```

### Helm Repository

```bash
helm repo add pulumi-exporter https://pulumi-labs.github.io/pulumi-exporter
helm repo update
helm install pulumi-exporter pulumi-exporter/pulumi-exporter
```

### Secret Management

For production, create the secret separately and reference it:

```bash
kubectl create secret generic pulumi-credentials \
  --from-literal=access-token=pul-xxxxxxxxxxxx
```

```bash
helm install pulumi-exporter oci://ghcr.io/pulumi-labs/charts/pulumi-exporter \
  --set existingSecret=pulumi-credentials \
  --set "pulumiOrganizations={my-org,another-org}" \
  --set otlp.endpoint=otel-collector:4318
```

For development, pass the token directly (the chart creates a Secret):

```bash
helm install pulumi-exporter oci://ghcr.io/pulumi-labs/charts/pulumi-exporter \
  --set pulumiAccessToken=pul-xxx \
  --set "pulumiOrganizations={my-org}" \
  --set otlp.endpoint=otel-collector:4318
```

### All Values

See the full [chart README](../charts/pulumi-exporter/README.md) for all configurable values.

### ServiceMonitor

Enable the Prometheus Operator ServiceMonitor:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  labels:
    release: prometheus
```

## Raw Kubernetes Manifests

If you prefer not to use Helm:

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
          image: ghcr.io/pulumi-labs/pulumi-exporter:latest
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

## Chart CI/CD

The chart ships with two GitHub Actions workflows:

| Workflow | Trigger | What it does |
|----------|---------|-------------|
| `helm-publish.yaml` | Push to `main` touching `charts/**` | ArtifactHub lint, chart-releaser (GH Pages), OCI push to GHCR, cosign sign |
| `lint-and-test.yaml` | PR touching `charts/**` | Trivy IaC scan, ct lint, ArtifactHub lint, kind cluster + ct install |

Charts are published both as a GitHub Pages Helm repo and as OCI artifacts to `ghcr.io/pulumi-labs/charts/pulumi-exporter`.
