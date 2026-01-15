# K8s Cost API Server

A Kubernetes cost monitoring API server that collects metrics from agents, stores time-series data, and provides cost optimization recommendations.

## Overview

This API server is designed to monitor and analyze Kubernetes cluster costs by:
- Receiving metrics from Kubernetes cost monitoring agents
- Storing time-series metrics in TimescaleDB
- Managing API keys for secure agent authentication
- Providing cost optimization recommendations
- Tracking recommendation application and dismissal

## Architecture

The server uses a multi-database architecture:
- **PostgreSQL**: Stores relational data (API keys, recommendations, tenants)
- **TimescaleDB**: Stores time-series metrics (node metrics, pod metrics)
- **Redis**: Caches API key lookups for improved performance

## Tech Stack

- **Language**: Go 1.25.4
- **Web Framework**: Gin
- **Databases**: 
  - PostgreSQL 15 (relational data)
  - TimescaleDB (time-series data)
- **Cache**: Redis 7
- **ORM**: GORM

## Project Structure

```
api-server/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── conf/
│   ├── api-server-dev.yaml     # Development configuration
│   ├── api-server-prod.yaml    # Production configuration
│   └── dev.env                 # Environment variables
├── docker/
│   ├── docker-compose.yml      # Infrastructure services
│   ├── postgres-init.sql       # PostgreSQL initialization
│   └── timescale-init.sql      # TimescaleDB initialization
├── internal/
│   ├── api/                    # HTTP handlers and routes
│   ├── config/                 # Configuration management
│   ├── db/                     # Database connections and wrappers
│   ├── middleware/             # HTTP middleware (auth, rate limiting)
│   ├── models/                 # Data models
│   └── services/               # Business logic services
└── Dockerfile                  # Multi-stage Docker build
```

## Features

### Core Features
- **API Key Authentication**: Secure agent authentication using API keys with Redis caching
- **Metrics Ingestion**: Accepts cluster metrics including individual pod metrics, namespace costs, and node metrics
- **Health Checks**: Built-in health check endpoint for monitoring
- **Rate Limiting**: Configurable rate limiting per minute
- **CORS Support**: Configurable CORS origins
- **Multi-tenant Support**: Tenant isolation via API keys

### MVP Cost Monitoring Features
- **Cost by Namespace**: Query cost breakdown by namespace with resource usage and estimated costs
- **Cost by Cluster**: Query cost breakdown by cluster with aggregated metrics
- **Resource Utilization vs Requests**: Analyze actual resource usage compared to requests for right-sizing insights
- **Daily/Weekly Cost Trends**: Time-series cost analysis with hourly, daily, or weekly intervals
- **Right-Sizing Recommendations**: Automated recommendations for optimizing pod resource requests based on actual usage patterns

## API Endpoints

### Health Check
- `GET /v1/health` - Health check endpoint

### Admin
- `POST /v1/admin/api_keys` - Create new API keys

### Metrics Ingestion (Protected)
- `POST /v1/ingest` - Ingest cluster metrics from agents
  - Requires API key authentication
  - Accepts: cluster name, timestamp, pod metrics, namespace costs, node metrics
  - Pod metrics include: CPU/memory usage, requests, and limits

### Cost Analysis (Protected)
- `GET /v1/costs/namespaces` - Get cost breakdown by namespace
  - Query params: `start_time`, `end_time` (RFC3339 format)
  - Returns: namespace costs, resource usage, pod counts, estimated costs
  
- `GET /v1/costs/clusters` - Get cost breakdown by cluster
  - Query params: `start_time`, `end_time` (RFC3339 format)
  - Returns: cluster costs, resource usage, pod/namespace counts, estimated costs
  
- `GET /v1/costs/utilization` - Get resource utilization vs requests
  - Query params: `start_time`, `end_time`, `namespace` (optional), `cluster` (optional)
  - Returns: pod-level metrics with utilization percentages
  
- `GET /v1/costs/trends` - Get cost trends over time
  - Query params: `start_time`, `end_time`, `interval` (hourly/daily/weekly)
  - Returns: time-series cost data with resource usage trends

### Recommendations (Protected)
- `GET /v1/recommendations` - Get all recommendations for the authenticated tenant
- `POST /v1/recommendations/generate` - Generate right-sizing recommendations
  - Query params: `lookback_hours` (default: 24)
  - Analyzes pod metrics and creates recommendations for underutilized resources
- `POST /v1/recommendations/:id/apply` - Apply a recommendation
- `POST /v1/recommendations/:id/dismiss` - Dismiss a recommendation

## Getting Started

### Prerequisites

- Go 1.25.4 or later
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### Local Development

1. **Start Infrastructure Services**

   ```bash
   make infra-up
   ```
   
   This starts PostgreSQL, TimescaleDB, and Redis using Docker Compose.

2. **Configure Environment**

   The server uses configuration from `conf/api-server-dev.yaml` by default. You can override the config file path using the `API_SERVER_CONF_FILE` environment variable.

3. **Run the API Server**

   ```bash
   make api
   ```
   
   Or directly:
   ```bash
   go run ./cmd/server
   ```

   The server will start on `http://localhost:8080` by default.

4. **Stop Infrastructure**

   ```bash
   make infra-down
   ```

### Docker Deployment

1. **Build the Docker Image**

   ```bash
   make build-api
   ```

2. **Run with Docker Compose**

   The `docker/docker-compose.yml` includes an API service that builds and runs the server along with the infrastructure services:

   ```bash
   cd docker && docker-compose up -d
   ```

## Configuration

Configuration is managed via YAML files in the `conf/` directory:

- **Development**: `conf/api-server-dev.yaml`
- **Production**: `conf/api-server-prod.yaml`

Key configuration sections:
- `server`: Host, port, timeouts, CORS, rate limiting
- `postgres`: PostgreSQL connection string
- `timescale`: TimescaleDB connection string
- `redis`: Redis connection details
- `security`: API key pepper and cache TTL
- `ingest`: Maximum payload size
- `agent`: Default API key ID

### Environment Variables

- `ENVIRONMENT`: Set to `production` or `prod` for production mode
- `API_SERVER_CONF_FILE`: Override the default config file path

## Development

### Available Make Targets

- `make infra-up` - Start infrastructure services (Postgres, TimescaleDB, Redis)
- `make infra-down` - Stop infrastructure services
- `make api` - Run the API server locally
- `make build-api` - Build the API server Docker image

### Database Setup

The infrastructure services are initialized with SQL scripts:
- `docker/postgres-init.sql` - Sets up PostgreSQL schema (tenants, API keys, recommendations)
- `docker/timescale-init.sql` - Sets up TimescaleDB schema and hypertables
  - `pod_metrics` hypertable: Stores pod-level metrics (CPU/memory usage, requests, limits)
  - `node_metrics` hypertable: Stores node-level metrics (capacity, instance types, hourly costs)

### Database Schema Updates

The `pod_metrics` table includes:
- `cpu_millicores` - Actual CPU usage
- `memory_bytes` - Actual memory usage
- `cpu_request_millicores` - CPU requests
- `memory_request_bytes` - Memory requests
- `cpu_limit_millicores` - CPU limits (new in MVP)
- `memory_limit_bytes` - Memory limits (new in MVP)

### Applying Migrations to Existing Databases

If you have an existing production TimescaleDB database, you need to apply migrations to add new columns.

**Quick Migration Guide:**

1. **Backup your database first:**
   ```bash
   pg_dump -h <host> -U <user> -d <database> > backup_before_migration.sql
   ```

2. **Apply the migration:**
   ```bash
   psql -h <host> -U <user> -d <database> -f docker/migrations/001_add_pod_limits.sql
   ```

3. **Verify the migration:**
   ```sql
   SELECT column_name FROM information_schema.columns 
   WHERE table_name = 'pod_metrics' 
   AND column_name IN ('cpu_limit_millicores', 'memory_limit_bytes');
   ```

For detailed migration instructions, see [`docker/migrations/README.md`](docker/migrations/README.md).

## Security

- API keys are hashed using a configurable pepper value
- API key lookups are cached in Redis to reduce database load
- Rate limiting is configurable per minute
- CORS origins are configurable for cross-origin requests

## Visualization

### Grafana Setup

Grafana can be used to visualize cost metrics stored in TimescaleDB.

**Quick Start:**
```bash
cd docker
docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana
```

Access Grafana at http://localhost:3000 (admin/admin)

**Documentation:**
- See [`docker/grafana/GRAFANA_SETUP.md`](docker/grafana/GRAFANA_SETUP.md) for detailed setup
- Example queries: [`docker/grafana/example-queries.sql`](docker/grafana/example-queries.sql)
- Pre-built dashboards: [`docker/grafana/dashboards/`](docker/grafana/dashboards/)

**Features:**
- Direct TimescaleDB connection
- Pre-configured data source
- Example dashboards for cost visualization
- SQL query examples for common metrics

## License

[Add your license here]
