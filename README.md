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
- **Multi-Tenant**: API key-based tenant isolation

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

### Frontend

Environment variables:
- `VITE_CLERK_PUBLISHABLE_KEY` - Clerk authentication key

## Deployment

| Service | Platform | Configuration |
|---------|----------|---------------|
| api-server | Railway | Root directory: `api-server` |
| auth-service-frontend | Vercel | Root directory: `auth-service-frontend` |
| cost-agent | Kubernetes | Via Helm or kubectl |

## License

Private and proprietary.
