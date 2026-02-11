# Cost-Agent Helm Chart

A Helm chart for deploying the Kubernetes cost monitoring agent.

This is a standalone Helm chart repository. You only need this chart to deploy cost-agent to your Kubernetes cluster.

## Version v1.0.11 Features

- **Individual Pod Metrics**: Collects and sends detailed pod-level metrics (CPU/memory usage, requests, and limits)
- **Enhanced Cost Analysis**: Enables accurate cost allocation by namespace and cluster
- **Right-Sizing Support**: Provides data for resource utilization vs requests analysis
- **Time-Series Trends**: Supports daily/weekly cost trend analysis

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- kubectl configured to access your cluster
- API key from the API server

## Installation

### Quick Start

1. **Create the API key secret** (if not using the chart's secret creation):

```bash
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="your_key_id:your_secret"
```

2. **Install with default values**:

```bash
helm install cost-agent ./cost-agent
```

3. **Install with custom values**:

```bash
helm install cost-agent ./cost-agent -f my-values.yaml
```

### Upgrade

```bash
helm upgrade cost-agent ./cost-agent
```

### Uninstall

```bash
helm uninstall cost-agent
```

## Configuration

The following table lists the configurable parameters and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/bugfreev587/cost-agent` |
| `image.tag` | Image tag | `cost-agent-v1.0.11-amd64-20260105-234955-03be71d` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `config.serverUrl` | API server URL | `https://api-server-production-7a9d.up.railway.app/v1/ingest` |
| `config.apiKeySecret.secretName` | Secret name for API key | `cost-agent-api-key` |
| `config.apiKeySecret.secretKey` | Key in secret | `api-key` |
| `config.clusterName` | Cluster name identifier | `cluster1` |
| `config.collectInterval` | Collection interval (seconds) | `600` (10 minutes) |
| `config.httpTimeout` | HTTP timeout (seconds) | `60` |
| `config.useMetricsAPI` | Use Kubernetes Metrics API | `true` |
| `config.namespaceFilter` | Namespace filter (empty = all) | `""` |
| `serviceAccount.create` | Create service account | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `256Mi` |

## Examples

### Install with Custom API Server URL

```bash
helm install cost-agent ./cost-agent \
  --set config.serverUrl="https://your-api-server.com/v1/ingest"
```

### Install with Custom Cluster Name

```bash
helm install cost-agent ./cost-agent \
  --set config.clusterName="production-cluster"
```

### Install with Custom Collection Interval

```bash
helm install cost-agent ./cost-agent \
  --set config.collectInterval=120
```

### Install and Create Secret

```bash
helm install cost-agent ./cost-agent \
  --set config.apiKeySecret.create=true \
  --set config.apiKeySecret.value="key_id:secret"
```

### Install with Custom Resource Limits

```bash
helm install cost-agent ./cost-agent \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=512Mi
```

### Install with Multiple Values Files

```bash
helm install cost-agent ./cost-agent \
  -f values-production.yaml \
  -f values-override.yaml
```

## Values File Example

Create a `my-values.yaml` file:

```yaml
replicaCount: 1

image:
  repository: ghcr.io/bugfreev587/cost-agent
  tag: "cost-agent-v1.0.11-amd64-20260105-234955-03be71d"

config:
  serverUrl: "https://api-server-production-7a9d.up.railway.app/v1/ingest"
  apiKeySecret:
    secretName: "cost-agent-api-key"
    secretKey: "api-key"
    create: false
  clusterName: "production-cluster"
  collectInterval: 600
  httpTimeout: 60
  useMetricsAPI: true
  namespaceFilter: ""

serviceAccount:
  create: true

rbac:
  create: true

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

Then install:

```bash
helm install cost-agent ./cost-agent -f my-values.yaml
```

## Verifying Installation

```bash
# Check deployment status
kubectl get deployment cost-agent

# Check pods
kubectl get pods -l app.kubernetes.io/name=cost-agent

# View logs
kubectl logs -f -l app.kubernetes.io/name=cost-agent

# Check service account
kubectl get serviceaccount cost-agent

# Check RBAC
kubectl get clusterrole cost-agent
kubectl get clusterrolebinding cost-agent
```

## Troubleshooting

### Pod Not Starting

1. Check pod status:
   ```bash
   kubectl describe pod -l app.kubernetes.io/name=cost-agent
   ```

2. Check logs:
   ```bash
   kubectl logs -l app.kubernetes.io/name=cost-agent
   ```

3. Verify secret exists:
   ```bash
   kubectl get secret cost-agent-api-key
   ```

### API Key Issues

If using secret, verify it's created:
```bash
kubectl get secret cost-agent-api-key -o yaml
```

### RBAC Issues

Verify RBAC resources are created:
```bash
kubectl get clusterrole cost-agent
kubectl get clusterrolebinding cost-agent
```

## Upgrading

To upgrade to a new version:

```bash
# Update values
helm upgrade cost-agent ./cost-agent \
  --set image.tag="cost-agent-v1.0.11-amd64-20260105-234955-03be71d"

# Or with values file
helm upgrade cost-agent ./cost-agent -f my-values.yaml
```

## Uninstalling

```bash
helm uninstall cost-agent
```

This will remove all resources created by the chart, but **will not** delete:
- The API key secret (if it was created outside the chart)
- Any persistent data

To also delete the secret:
```bash
kubectl delete secret cost-agent-api-key
```

## Repository Structure

```
cost-agent/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration values
├── README.md               # This file
└── templates/
    ├── _helpers.tpl        # Template helper functions
    ├── deployment.yaml     # Main deployment
    ├── serviceaccount.yaml # Service account
    ├── rbac.yaml          # ClusterRole and ClusterRoleBinding
    ├── secret.yaml        # Optional API key secret
    └── hpa.yaml           # Optional HorizontalPodAutoscaler
```
