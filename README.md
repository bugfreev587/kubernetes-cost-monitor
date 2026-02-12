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
- **Role-Based Access Control**: Owner/Admin/Editor/Viewer roles with granular permissions

## User Roles & Permissions

The system implements a hierarchical role-based access control (RBAC) system:

### Role Hierarchy

| Role | Description |
|------|-------------|
| **Owner** | Tenant creator with full control. Cannot be removed. Can manage billing, admins, and delete tenant. |
| **Admin** | Can manage team members (editors/viewers), API keys, and invite users. Cannot manage other admins. |
| **Editor** | Can modify data including apply/dismiss recommendations. Has read + write access. |
| **Viewer** | Read-only access to dashboard, costs, and recommendations. |

### Permissions Matrix

| Permission | Viewer | Editor | Admin | Owner |
|------------|:------:|:------:|:-----:|:-----:|
| View dashboard & costs | Y | Y | Y | Y |
| View recommendations | Y | Y | Y | Y |
| View/edit own profile | Y | Y | Y | Y |
| Generate recommendations | | Y | Y | Y |
| Apply/dismiss recommendations | | Y | Y | Y |
| View team members | | | Y | Y |
| Invite users | | | Y | Y |
| Suspend/remove users | | | Y | Y |
| Manage API keys | | | Y | Y |
| Promote to editor | | | Y | Y |
| Promote to admin | | | | Y |
| Remove admins | | | | Y |
| Change pricing plan | | | | Y |
| Transfer ownership | | | | Y |
| Delete tenant | | | | Y |

### User Status

Users can have the following status:
- **active**: Normal access based on role
- **suspended**: Account temporarily disabled, cannot log in. When suspended:
  - All Clerk sessions are revoked (forced logout from all devices)
  - Clerk metadata is cleared (prevents Grafana OAuth access)
  - User cannot sign in until unsuspended
- **pending**: Invited but not yet signed up

### User Removal

When a user is removed from a tenant:
- User record is deleted from database
- Clerk metadata is cleared (tenant_id, role, grafana_org_id)
- All Clerk sessions are revoked (forced logout)
- User cannot access Grafana via OAuth for the removed tenant

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

### Public Endpoints (No Authentication)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/health` | GET | Health check |
| `/v1/plans` | GET | List available pricing plans |
| `/v1/auth/sync` | POST | Sync user after Clerk authentication |

### Agent Endpoints (API Key Required)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/ingest` | POST | Ingest metrics from cost-agent |

### Viewer+ Endpoints (Authenticated Users)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/costs/namespaces` | GET | Cost breakdown by namespace |
| `/v1/costs/clusters` | GET | Cost breakdown by cluster |
| `/v1/costs/utilization` | GET | Resource utilization vs requests |
| `/v1/costs/trends` | GET | Cost trends over time |
| `/v1/recommendations` | GET | Get optimization recommendations |
| `/v1/allocation` | GET | OpenCost-compatible allocation API |
| `/v1/users` | GET | List team members |

### Editor+ Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/recommendations/generate` | POST | Generate new recommendations |
| `/v1/recommendations/:id/apply` | POST | Apply a recommendation |
| `/v1/recommendations/:id/dismiss` | POST | Dismiss a recommendation |

### Admin+ Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/admin/api_keys` | GET | List API keys (masked) |
| `/v1/admin/api_keys` | POST | Create new API key |
| `/v1/admin/api_keys/:key_id` | DELETE | Revoke an API key |
| `/v1/admin/users/invite` | POST | Invite a new user |
| `/v1/admin/users/:user_id/suspend` | PATCH | Suspend a user |
| `/v1/admin/users/:user_id/unsuspend` | PATCH | Unsuspend a user |
| `/v1/admin/users/:user_id/role` | PATCH | Update user role (viewer/editor) |
| `/v1/admin/users/:user_id` | DELETE | Remove a user |
| `/v1/admin/tenants/:tenant_id/pricing-plan` | GET | Get tenant's pricing plan |
| `/v1/admin/tenants/:tenant_id/usage` | GET | Get tenant usage stats |

### Owner-Only Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/owner/tenants/:tenant_id/pricing-plan` | PATCH | Change pricing plan |
| `/v1/owner/users/:user_id/promote-admin` | POST | Promote user to admin |
| `/v1/owner/users/:user_id/demote-admin` | DELETE | Demote admin to editor |
| `/v1/owner/transfer-ownership` | POST | Transfer tenant ownership |
| `/v1/owner/tenants/:tenant_id` | DELETE | Delete tenant and all data |

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
- `VITE_API_SERVER_URL` - API server URL (default: http://localhost:8080)

### API Server Additional Environment Variables

```bash
# Clerk (for user management and invitations)
CLERK_SECRET_KEY=<clerk-secret-key>
CLERK_FRONTEND_URL=https://your-frontend.vercel.app

# Grafana (for automatic org management)
GRAFANA_URL=https://your-grafana.railway.app
GRAFANA_USERNAME=admin
GRAFANA_PASSWORD=<admin-password>
# Or use API token:
GRAFANA_API_TOKEN=<service-account-token>

# Security
API_KEY_PEPPER=<random-secret-for-api-key-hashing>
```

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

#### Automatic Grafana Organization Management

When properly configured, the API server automatically manages Grafana organizations:

- **New tenant created** → Grafana organization is automatically created
- **Tenant deleted** → Associated Grafana organization is deleted
- **User metadata** → Clerk stores `grafana_org_id` for OAuth org mapping

#### Grafana Environment Variables (API Server)

```bash
GRAFANA_URL=https://your-grafana.railway.app
GRAFANA_USERNAME=admin
GRAFANA_PASSWORD=<admin-password>
# Or use API token instead of username/password:
GRAFANA_API_TOKEN=<service-account-token>
```

#### Grafana OAuth Configuration

Set these on your Grafana service:

```bash
GF_AUTH_GENERIC_OAUTH_ENABLED=true
GF_AUTH_GENERIC_OAUTH_NAME=Clerk
GF_AUTH_GENERIC_OAUTH_CLIENT_ID=<clerk-oauth-client-id>
GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=<clerk-oauth-client-secret>
GF_AUTH_GENERIC_OAUTH_SCOPES=openid profile email public_metadata
GF_AUTH_GENERIC_OAUTH_AUTH_URL=https://<clerk-domain>/oauth/authorize
GF_AUTH_GENERIC_OAUTH_TOKEN_URL=https://<clerk-domain>/oauth/token
GF_AUTH_GENERIC_OAUTH_API_URL=https://<clerk-domain>/oauth/userinfo
GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH=contains(public_metadata.roles[*], 'admin') && 'Admin' || contains(public_metadata.roles[*], 'editor') && 'Editor' || 'Viewer'
GF_AUTH_GENERIC_OAUTH_ORG_ATTRIBUTE_PATH=public_metadata.grafana_org_id
GF_AUTH_GENERIC_OAUTH_ORG_MAPPING=<grafana_org_id>:<grafana_org_id>
GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP=true
```

See `docs/grafana-clerk-oauth-setup.md` for detailed setup instructions.

## Dashboard Data Calculation

The dashboard displays five panels. All cost calculations use the **OpenCost model** where the effective billed resource is `max(request, usage)` — you pay for whichever is higher.

### Data Sources

The cost-agent collects two types of data every collection interval:

| Data | Source | Fields |
|------|--------|--------|
| **Requested resources** | Kubernetes API (`pod.spec.containers[].resources.requests`) | CPU millicores, memory bytes |
| **Actual usage** | Kubernetes Metrics API (`metrics-server`) | CPU millicores, memory bytes |
| **Node capacity** | Kubernetes API (`node.status.capacity`) | CPU capacity, memory capacity, instance type |

### Cost Model

Costs are computed using configurable pricing rates (set via the Pricing Configuration page):

```
cpuCost     = cpuCoreHours × cpuRate       (default: $0.031611/core-hour)
memoryCost  = ramGBHours × memoryRate      (default: $0.004237/GB-hour)
totalCost   = cpuCost + memoryCost
```

The pricing lookup chain: **cluster-specific config** → **tenant default config** → **system defaults**.

### Panel 1: Summary Cards

**Endpoint**: `GET /v1/allocation/summary/topline?window=7d`

| Card | Formula |
|------|---------|
| **Total Cost** | `SUM(totalCost)` across all allocations + idle cost |
| **CPU Cost** | `SUM(cpuCoreHours × cpuRate)` for each namespace/cluster |
| **Memory Cost** | `SUM(ramGBHours × memoryRate)` for each namespace/cluster |
| **Efficiency** | `AVG((cpuUsage/cpuRequest + memUsage/memRequest) / 2)` across all allocations |

Where:
- `cpuCoreHours = max(avgCpuCoresRequest, avgCpuCoresUsage) × durationHours`
- `ramGBHours = max(avgRamBytesRequest, avgRamBytesUsage) × durationHours / 1GB`

### Panel 2: Cost by Namespace

**Endpoint**: `GET /v1/allocation/summary?window=7d&aggregate=namespace`

Groups pod metrics by namespace and computes per-namespace:

```
cpuCost   = cpuCoreHours × cpuRate
ramCost   = (ramByteHours / 1GB) × memoryRate
totalCost = cpuCost + ramCost
```

Displayed as a horizontal bar chart sorted by total cost descending.

### Panel 3: Cost Trend Chart

**Endpoint**: `GET /v1/costs/trends?interval=daily`

Uses TimescaleDB `time_bucket('1 day', time)` to aggregate pod metrics into daily buckets over the last 30 days:

```sql
SELECT time_bucket('1 day', time) as bucket_time,
       SUM(cpu_request_millicores), SUM(memory_request_bytes),
       AVG(cpu_millicores), AVG(memory_bytes), COUNT(DISTINCT pod_name)
FROM pod_metrics
WHERE tenant_id = $1 AND time >= now() - interval '30 days'
GROUP BY bucket_time ORDER BY bucket_time ASC
```

For each bucket, estimated cost is computed from average node hourly cost. The CPU/memory cost split is estimated proportionally using resource request weights and standard cloud pricing ratios.

### Panel 4: Top Cost Drivers

**Endpoint**: `GET /v1/costs/utilization`

Shows the top 10 pods by CPU utilization with their resource efficiency:

```sql
SELECT cluster_name, namespace, pod_name,
       AVG(cpu_millicores) / AVG(cpu_request_millicores) * 100 as cpu_utilization,
       AVG(memory_bytes) / AVG(memory_request_bytes) * 100 as memory_utilization
FROM pod_metrics
WHERE tenant_id = $1 AND pod_name != '__aggregate__'
GROUP BY cluster_name, namespace, pod_name
ORDER BY cpu_utilization DESC LIMIT 100
```

Helps identify over-provisioned pods (low utilization = wasted resources).

### Panel 5: Recommendations

**Endpoint**: `GET /v1/recommendations`

Right-sizing recommendations generated by `POST /v1/recommendations/generate`:

1. For each pod, compute **P95 usage** over the lookback period (default 24h)
2. Add a **20% buffer**: `recommended = P95 × 1.2`
3. Apply a **10% floor**: `recommended = max(recommended, currentRequest × 0.1)`
4. Calculate **estimated savings**: `(currentRequest - recommended) × hourlyRate`
5. Compute **confidence**: `1 - (avgUsage / currentRequest)` — higher confidence when usage is much lower than request

Recommendations have statuses: `open`, `applied`, `dismissed`.

## Documentation

Additional documentation in the `docs/` directory:
- `GRAFANA_SETUP_QUICKSTART.md` - Grafana multi-tenant setup guide
- `grafana-clerk-oauth-setup.md` - Grafana + Clerk OAuth integration
- `DEPLOYMENT_QUICKSTART.md` - Quick deployment guide
- `multi-tenant-setup-guide.md` - Complete multi-tenant architecture

## License

Private and proprietary.
