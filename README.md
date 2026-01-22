# Kubernetes Cost Monitor

A comprehensive solution for monitoring and analyzing Kubernetes cluster costs. Collect metrics from your clusters, visualize spending, and get right-sizing recommendations to optimize resource allocation.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Kubernetes     │     │   API Server    │     │    Frontend     │
│  Clusters       │     │   (Railway)     │     │   (Vercel)      │
│                 │     │                 │     │                 │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │Cost Agent │──┼────▶│  │  Gin API  │  │◀────┼──│  React    │  │
│  └───────────┘  │     │  └───────────┘  │     │  │  + Clerk  │  │
│                 │     │        │        │     │  └───────────┘  │
└─────────────────┘     │        ▼        │     └─────────────────┘
                        │  ┌───────────┐  │
                        │  │TimescaleDB│  │
                        │  │PostgreSQL │  │
                        │  │  Redis    │  │
                        │  └───────────┘  │
                        └─────────────────┘
```

## Components

| Component | Description | Tech Stack |
|-----------|-------------|------------|
| **api-server** | REST API for metrics ingestion, cost analysis, and recommendations | Go, Gin, GORM, TimescaleDB, PostgreSQL, Redis |
| **cost-agent** | Kubernetes agent that collects pod/node metrics | Go, client-go, Kubernetes Metrics API |
| **auth-service-frontend** | User authentication and dashboard | React 19, TypeScript, Vite, Clerk |
| **helm** | Helm charts for Kubernetes deployment | Helm |

## Features

- **Metrics Collection**: Automated collection of CPU/memory usage, requests, and limits from Kubernetes clusters
- **Cost Analysis**: Breakdown costs by namespace, cluster, and time period
- **Resource Utilization**: Compare actual usage vs requested resources
- **Right-Sizing Recommendations**: AI-generated suggestions to optimize resource allocation
- **Multi-Cluster Support**: Monitor multiple Kubernetes clusters from a single dashboard
- **Multi-Tenant**: API key-based tenant isolation with row-level security
- **OpenCost-Compatible API**: Drop-in compatible allocation API for existing OpenCost integrations
- **Pricing Plans**: Tiered plans (Starter, Premium, Business) with cluster/node/user limits
- **Grafana Integration**: Multi-tenant Grafana with Clerk OAuth and automatic tenant isolation
- **Enhanced Pod Metrics**: Labels, pod phase, QoS class, and per-container metrics

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.25+ (for local development)
- Node.js 18+ (for frontend development)
- Kubernetes cluster access (for cost-agent deployment)

### Local Development

1. **Start infrastructure services**
   ```bash
   cd api-server
   make infra-up
   ```

2. **Run the API server**
   ```bash
   make api
   ```

3. **Create an API key**
   ```bash
   curl -X POST http://localhost:8080/v1/admin/api_keys
   ```

4. **Run the frontend** (in a new terminal)
   ```bash
   cd auth-service-frontend
   npm install
   npm run dev
   ```

### Deploy Cost Agent to Kubernetes

1. Create a secret with your API key:
   ```bash
   kubectl create secret generic cost-agent-api-key \
     --from-literal=api-key=<keyid>:<secret>
   ```

2. Deploy the agent:
   ```bash
   cd cost-agent
   make deploy
   ```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/health` | GET | Health check |
| `/v1/admin/api_keys` | POST | Create API key |
| `/v1/ingest` | POST | Ingest metrics from agents |
| `/v1/costs/namespaces` | GET | Cost breakdown by namespace |
| `/v1/costs/clusters` | GET | Cost breakdown by cluster |
| `/v1/costs/utilization` | GET | Resource utilization vs requests |
| `/v1/costs/trends` | GET | Cost trends over time |
| `/v1/recommendations` | GET | Get optimization recommendations |
| `/v1/recommendations/generate` | POST | Generate new recommendations |
| `/v1/allocation` | GET | OpenCost-compatible allocation API |
| `/v1/tenants/current` | GET | Get current tenant info and plan |

### OpenCost-Compatible Allocation API

The `/v1/allocation` endpoint provides OpenCost-compatible cost allocation queries:

```bash
# Cost by namespace for last 7 days
curl "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace" \
  -H "X-API-Key: <keyid>:<secret>"

# Cost by namespace and label, with idle cost distribution
curl "http://localhost:8080/v1/allocation?window=24h&aggregate=namespace,label:app&idle=true&shareIdle=weighted"

# Time-series data with daily buckets
curl "http://localhost:8080/v1/allocation?window=30d&aggregate=cluster&step=1d&accumulate=false"

# Filtered by namespace
curl "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filter=namespace:production"
```

Query parameters:
- `window`: Time window (required) - `24h`, `7d`, `today`, `lastweek`, or date range `2024-01-01,2024-01-07`
- `aggregate`: Grouping - `namespace`, `cluster`, `node`, `pod`, `controller`, `label:<key>`
- `step`: Time bucket size - `1h`, `1d`, `1w`
- `accumulate`: Result accumulation - `true`, `false`, `hour`, `day`, `week`
- `idle`: Include idle costs - `true` or `false`
- `shareIdle`: Distribute idle costs - `true`, `false`, `weighted`
- `filter`: Filter expressions - `namespace:value`, `cluster:value`, `label:key=value`

## Pricing Plans

| Plan | Price | Clusters | Nodes | Users | Retention |
|------|-------|----------|-------|-------|-----------|
| **Starter** | Free | 1 | 5 | 1 | 7 days |
| **Premium** | $49/mo | 5 | 50 | 10 | 30 days |
| **Business** | $199/mo | Unlimited | Unlimited | Unlimited | 90 days |

Plan limits are enforced at the API level during metrics ingestion.

## Configuration

### API Server

Configuration files are in `api-server/conf/`:
- `api-server-dev.yaml` - Development settings
- `api-server-prod.yaml` - Production settings

### Cost Agent

Environment variables (prefix `AGENT_`):
- `AGENT_SERVER_URL` - API server endpoint
- `AGENT_API_KEY` - Authentication key (`keyid:secret`)
- `AGENT_CLUSTER_NAME` - Cluster identifier
- `AGENT_COLLECT_INTERVAL` - Collection interval in seconds (default: 600)
- `AGENT_COLLECT_LABELS` - Collect pod labels (default: true)
- `AGENT_COLLECT_CONTAINERS` - Collect per-container metrics (default: true)

### Frontend

Environment variables:
- `VITE_CLERK_PUBLISHABLE_KEY` - Clerk authentication key

## Deployment

| Service | Platform | Configuration |
|---------|----------|---------------|
| api-server | Railway | Root directory: `api-server` |
| auth-service-frontend | Vercel | Root directory: `auth-service-frontend` |
| cost-agent | Kubernetes | Via Helm or kubectl |
| grafana | Railway | With Clerk OAuth |

### Helm Deployment

Deploy the cost-agent using the Helm chart:

```bash
# Add values
helm install cost-agent ./helm/cost-agent \
  --set config.serverUrl=https://your-api.railway.app \
  --set config.apiKey=<keyid>:<secret> \
  --set config.clusterName=my-cluster

# Or use a values file
helm install cost-agent ./helm/cost-agent -f values.yaml
```

### Grafana Multi-Tenant Setup

The API server integrates with Grafana for visualization with automatic tenant isolation:

1. Deploy Grafana on Railway with Clerk OAuth
2. Configure row-level security in TimescaleDB
3. Users logging in via Clerk are automatically assigned to their tenant's Grafana organization

See `docs/GRAFANA_SETUP_QUICKSTART.md` for detailed setup instructions.

## Documentation

Additional documentation in the `docs/` directory:
- `GRAFANA_SETUP_QUICKSTART.md` - Grafana multi-tenant setup guide
- `DEPLOYMENT_QUICKSTART.md` - Quick deployment guide
- `multi-tenant-setup-guide.md` - Complete multi-tenant architecture

## License

Private and proprietary.
