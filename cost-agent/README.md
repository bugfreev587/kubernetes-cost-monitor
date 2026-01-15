# K8s Cost Agent

A Kubernetes agent that collects cluster metrics and sends them to the K8s Cost API Server for cost monitoring and analysis.

## Overview

The cost-agent is a lightweight Kubernetes agent that:
- Collects pod and node metrics from Kubernetes clusters
- Aggregates metrics by namespace
- Sends metrics to the API server at configurable intervals
- Supports both in-cluster and local (kubeconfig) deployment
- Uses exponential backoff for resilient metric delivery

## Architecture

The agent operates as a daemon that:
1. **Collects Metrics**: Uses Kubernetes API and Metrics API to gather:
   - Pod metrics (CPU/memory usage, requests, and limits)
   - Node metrics (capacity, allocatable resources, instance types)
2. **Aggregates Data**: Groups pod metrics by namespace (for backward compatibility)
3. **Sends Metrics**: Posts both individual pod metrics and aggregated namespace data to the API server's `/v1/ingest` endpoint

## Tech Stack

- **Language**: Go 1.25.4
- **Kubernetes Client**: `k8s.io/client-go` v0.34.2
- **Metrics API**: `k8s.io/metrics` v0.34.2
- **HTTP Client**: Standard library with exponential backoff retry logic
- **Configuration**: Viper for environment-based configuration

## Project Structure

```
cost-agent/
├── main.go                    # Application entry point
├── Dockerfile                 # Multi-stage Docker build
├── Makefile                   # Build and deployment commands
├── internal/
│   ├── collector/            # Kubernetes metrics collection
│   │   ├── metrics.go        # Pod and node metric collection
│   │   └── aggregator.go     # Namespace aggregation logic
│   ├── config/               # Configuration management
│   │   └── config.go         # Environment-based config loader
│   └── sender/               # API server communication
│       └── sender.go         # HTTP client with retry logic
└── deploy/                   # Kubernetes deployment manifests (create as needed)
```

## Features

- **Comprehensive Metrics Collection**:
  - Uses Kubernetes Metrics API when available (actual usage)
  - Falls back to resource requests when Metrics API is unavailable
  - Collects CPU/memory limits in addition to requests
- **Individual Pod Metrics**: Sends detailed pod-level metrics for accurate cost analysis
- **Namespace Aggregation**: Also provides aggregated namespace data for backward compatibility
- **Namespace Filtering**: Optional namespace filter for targeted collection
- **Resilient Delivery**: Exponential backoff retry for transient failures
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean shutdown
- **In-Cluster or Local**: Works both inside Kubernetes and with local kubeconfig
- **Multi-Architecture**: Supports multiple CPU architectures via Docker buildx

## Configuration

The agent is configured via environment variables with the `AGENT_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENT_SERVER_URL` | `http://host.docker.internal:8080` | API server endpoint URL |
| `AGENT_API_KEY` | (required) | API key for authentication (format: `keyid:secret`) |
| `AGENT_CLUSTER_NAME` | `cost-dashboard-dev` | Name identifier for the cluster |
| `AGENT_COLLECT_INTERVAL` | `600` | Collection interval in seconds (10 minutes) |
| `AGENT_HTTP_TIMEOUT` | `10` | HTTP request timeout in seconds |
| `AGENT_USE_METRICS_API` | `true` | Whether to use Kubernetes Metrics API |
| `AGENT_NAMESPACE_FILTER` | `""` | Optional namespace filter (empty = all namespaces) |

### Kubernetes Configuration

When running in-cluster, the agent uses the service account's credentials. You can also provide:
- `KUBECONFIG`: Path to kubeconfig file (for local development)

## Getting Started

### Prerequisites

- Go 1.25.4 or later (for local development)
- Docker (for containerized builds)
- Kubernetes cluster access (for deployment)
- API key from the API server

### Local Development

1. **Get an API Key**

   First, create an API key using the API server's admin endpoint:
   ```bash
   curl -X POST http://localhost:8080/v1/admin/api_keys
   ```

2. **Set Environment Variables**

   ```bash
   export AGENT_SERVER_URL=http://localhost:8080
   export AGENT_API_KEY=your_key_id:your_secret
   export AGENT_CLUSTER_NAME=my-cluster
   export KUBECONFIG=~/.kube/config  # Optional, for local kubeconfig
   ```

3. **Run the Agent**

   ```bash
   go run ./main.go
   ```

### Docker Build and Run

1. **Build the Image**

   ```bash
   make build
   ```

   Or manually:
   ```bash
   docker build -t cost-agent:local .
   ```

2. **Run the Container**

   ```bash
   make run
   ```

   Or manually:
   ```bash
   docker run --rm \
     -e AGENT_SERVER_URL=http://host.docker.internal:8080 \
     -e AGENT_API_KEY=your_key_id:your_secret \
     -e AGENT_CLUSTER_NAME=my-cluster \
     cost-agent:local
   ```

### Kubernetes Deployment

1. **Create API Key Secret**

   ```bash
   kubectl create secret generic cost-agent-api-key \
     --from-literal=api-key=your_key_id:your_secret
   ```

2. **Create Deployment Manifest**

   Create `deploy/agent.yaml`:
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: cost-agent
     namespace: default
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: cost-agent
     template:
       metadata:
         labels:
           app: cost-agent
       spec:
         serviceAccountName: cost-agent
         containers:
         - name: agent
           image: ghcr.io/bugfreev587/cost-agent:v1.0.8
           env:
           - name: AGENT_SERVER_URL
             value: "http://api-server:8080"
           - name: AGENT_API_KEY
             valueFrom:
               secretKeyRef:
                 name: cost-agent-api-key
                 key: api-key
           - name: AGENT_CLUSTER_NAME
             value: "production-cluster"
   ```

3. **Create Service Account and RBAC**

   The agent needs permissions to:
   - List pods and nodes
   - Read pod metrics (if using Metrics API)

   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: cost-agent
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: cost-agent
   rules:
   - apiGroups: [""]
     resources: ["pods", "nodes"]
     verbs: ["list", "get"]
   - apiGroups: ["metrics.k8s.io"]
     resources: ["pods", "nodes"]
     verbs: ["get", "list"]
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRoleBinding
   metadata:
     name: cost-agent
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: cost-agent
   subjects:
   - kind: ServiceAccount
     name: cost-agent
     namespace: default
   ```

4. **Deploy**

   ```bash
   make deploy
   ```

   Or manually:
   ```bash
   kubectl apply -f deploy/agent.yaml
   ```

## Available Make Targets

- `make build` - Build the agent Docker image
- `make push` - Push the image to GitHub Container Registry (requires `GHCR_TOKEN`)
- `make login` - Login to GitHub Container Registry
- `make release` - Build and push the image
- `make run` - Run the agent container locally
- `make run-detached` - Run the agent container in detached mode
- `make deploy` - Deploy the agent to Kubernetes
- `make undeploy` - Remove the agent deployment
- `make info` - Show image and configuration info

## Metrics Collection Details

### Pod Metrics

The agent collects individual pod metrics including:
- **CPU Usage**: Actual CPU usage from Metrics API (if available), in millicores
- **Memory Usage**: Actual memory usage from Metrics API (if available), in bytes
- **CPU Requests**: Sum of CPU requests from all containers in the pod spec, in millicores
- **Memory Requests**: Sum of memory requests from all containers in the pod spec, in bytes
- **CPU Limits**: Sum of CPU limits from all containers in the pod spec, in millicores (new in MVP)
- **Memory Limits**: Sum of memory limits from all containers in the pod spec, in bytes (new in MVP)
- **Fallback**: If Metrics API is unavailable, uses requests as estimates for usage

These individual pod metrics enable:
- Accurate cost allocation by namespace and cluster
- Resource utilization vs requests analysis
- Right-sizing recommendations based on actual usage patterns
- Time-series cost trend analysis

### Node Metrics

The agent collects:
- **Node Name**: Kubernetes node name
- **Instance Type**: From `node.kubernetes.io/instance-type` label
- **CPU Capacity**: Total CPU capacity in millicores
- **Memory Capacity**: Total memory capacity in bytes
- **CPU Allocatable**: Allocatable CPU in millicores
- **Memory Allocatable**: Allocatable memory in bytes

### Aggregation

Pod metrics are aggregated by namespace (for backward compatibility):
- Total CPU (max of usage or requests)
- Total Memory (max of usage or requests)
- Pod count per namespace

**Note**: The agent now sends both individual pod metrics and aggregated namespace data. Individual pod metrics enable advanced cost analysis features like utilization tracking and right-sizing recommendations.

## Testing

The cost-agent image includes a test script for validating API server connectivity and endpoints.

### Running API Tests

The test script is available in multiple locations:
- `/tmp/test-api.sh` - Quick access from pod (recommended)
- `/home/agent/scripts/test-api.sh` - User-specific location
- `/usr/local/bin/test-api.sh` - System-wide location

**From inside the pod:**
```bash
# Exec into the pod
kubectl exec -it <pod-name> -- sh

# Run the test script
/tmp/test-api.sh
```

The script tests:
- Health check endpoint
- Recommendations endpoint (with authentication)
- Metrics ingestion endpoint
- Invalid API key validation
- Cost by namespace endpoint
- Cost by cluster endpoint
- Utilization vs requests endpoint
- Cost trends endpoint

**Test Report Summary:**
The script provides a summary report at the end showing:
- Total tests run
- Tests passed/failed
- Success rate percentage
- List of failed tests (if any)

**Exit Codes:**
- `0` - All tests passed
- `1` - One or more tests failed

### Debugging Tools

The container includes:
- **vim** - Text editor for debugging configuration files
- **wget** - HTTP client for testing API endpoints
- **test-api.sh** - Comprehensive API endpoint test script

## Error Handling

- **Collection Errors**: Non-fatal, logged and collection continues
- **Send Errors**: Uses exponential backoff retry (up to 2 minutes)
- **Permanent Errors** (4xx except 429): No retry, logged as permanent failure
- **Transient Errors** (5xx, 429): Retried with exponential backoff

## Security

- API keys are passed via environment variables or Kubernetes secrets
- Uses Kubernetes service account for in-cluster authentication
- Supports RBAC for fine-grained permissions
- Runs as non-root user in container

## Troubleshooting

### Agent can't connect to API server

- Verify `AGENT_SERVER_URL` is correct
- Check network connectivity from agent to API server
- Ensure API server is running and accessible

### Authentication failures

- Verify `AGENT_API_KEY` is set correctly (format: `keyid:secret`)
- Check that the API key exists in the API server
- Ensure the API key hasn't expired

### Metrics collection fails

- Verify Kubernetes API access
- Check RBAC permissions if running in-cluster
- If using Metrics API, ensure metrics-server is installed
- Set `AGENT_USE_METRICS_API=false` to fall back to requests

### No metrics being sent

- Check agent logs for errors
- Verify collection interval is appropriate (default: 600 seconds / 10 minutes)
- Ensure pods/nodes exist in the cluster
- Check namespace filter if set

## License

[Add your license here]
