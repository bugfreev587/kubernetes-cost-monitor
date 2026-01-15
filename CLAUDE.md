# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kubernetes Cost Monitor is a monorepo with three services for monitoring and analyzing Kubernetes cluster costs:

- **api-server/** (Go) - REST API for cost monitoring, metrics ingestion, and analysis
- **cost-agent/** (Go) - Kubernetes agent that collects pod/node metrics and sends to API server
- **auth-service-frontend/** (React/TypeScript) - User authentication and dashboard frontend

## Build & Run Commands

### API Server (api-server/)
```bash
make infra-up              # Start PostgreSQL, TimescaleDB, Redis via Docker Compose
make infra-down            # Stop infrastructure services
make api                   # Run API server locally (go run ./cmd/server)
make build-api             # Build Docker image
make clear-timescale       # Clear TimescaleDB data (preserves schema)
make reset-timescale       # Drop and recreate TimescaleDB tables
```

### Cost Agent (cost-agent/)
```bash
cd cost-agent
make build                 # Build Docker image
make run                   # Run container locally
make release               # Build and push to GHCR
make deploy                # Deploy to Kubernetes (kubectl apply)
make undeploy              # Remove from Kubernetes
```

### Frontend (auth-service-frontend/)
```bash
cd auth-service-frontend
npm install                # Install dependencies
npm run dev                # Start Vite dev server (http://localhost:5173)
npm run build              # TypeScript compile + Vite build
npm run lint               # Run ESLint
npm run preview            # Preview production build
```

## Architecture

### Multi-Database Design
- **PostgreSQL**: Relational data (tenants, api_keys, recommendations tables)
- **TimescaleDB**: Time-series metrics via hypertables (pod_metrics, node_metrics)
- **Redis**: API key lookup cache

### Data Flow
1. Cost agent collects metrics from Kubernetes API and Metrics API
2. Agent sends metrics to API server's `/v1/ingest` endpoint with API key auth
3. API server stores in TimescaleDB and generates cost optimization recommendations
4. Frontend authenticates via Clerk and displays dashboard

### Key API Endpoints
- `POST /v1/ingest` - Metrics ingestion (requires API key)
- `GET /v1/costs/namespaces` - Cost breakdown by namespace
- `GET /v1/costs/clusters` - Cost breakdown by cluster
- `GET /v1/costs/utilization` - Resource utilization vs requests
- `GET /v1/costs/trends` - Cost trends over time
- `GET /v1/recommendations` - Get cost optimization recommendations
- `POST /v1/recommendations/generate` - Generate right-sizing recommendations
- `POST /v1/admin/api_keys` - Create new API keys

### Code Organization

**api-server/internal/**
- `api/` - Gin HTTP handlers (server.go, ingest.go, costs.go, recommendations.go)
- `services/` - Business logic (cost_service.go, recommendation_service.go)
- `db/` - Database connections (postgres.go, timescale.go)
- `middleware/` - Auth and rate limiting
- `config/` - YAML config loading

**cost-agent/internal/**
- `collector/` - Kubernetes metrics collection (metrics.go, aggregator.go)
- `sender/` - HTTP client with exponential backoff retry
- `config/` - Environment-based config (uses `AGENT_` prefix)

## Configuration

### API Server
Config files in `api-server/conf/`:
- `api-server-dev.yaml` (default)
- `api-server-prod.yaml`
- Override with `API_SERVER_CONF_FILE` env var

### Cost Agent Environment Variables
- `AGENT_SERVER_URL` - API server endpoint
- `AGENT_API_KEY` - Format: `keyid:secret`
- `AGENT_CLUSTER_NAME` - Cluster identifier
- `AGENT_COLLECT_INTERVAL` - Collection interval in seconds (default: 600)
- `AGENT_USE_METRICS_API` - Use Kubernetes Metrics API (default: true)

### Frontend
- `VITE_CLERK_PUBLISHABLE_KEY` - Clerk authentication key (in .env)

## Testing

### API Server
See `api-server/TESTING.md` for curl commands to test all endpoints.

### Cost Agent
Container includes `/tmp/test-api.sh` for endpoint validation:
```bash
kubectl exec -it <pod-name> -- /tmp/test-api.sh
```

## Docker Compose Services
Located in `api-server/docker/docker-compose.yml`:
- PostgreSQL 15 on port 5431
- TimescaleDB on port 5433
- Redis 7 on port 6379
- API server on port 8080
