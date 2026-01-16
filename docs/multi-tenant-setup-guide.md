# Multi-Tenant Setup Guide

Complete guide for setting up Grafana with Clerk authentication and row-level security for your Kubernetes Cost Monitor SaaS.

## Overview

This setup provides:
- **Clerk OAuth** authentication for Grafana
- **Automatic tenant isolation** via PostgreSQL Row-Level Security (RLS)
- **Grafana organizations** mapped to tenants
- **Automated provisioning** of organizations via API webhooks

## Architecture

```
User (Clerk Auth)
  → Grafana (OAuth)
    → TimescaleDB (RLS policies filter by tenant_id)
      → Only tenant's data visible
```

## Prerequisites

- Grafana deployed on Railway
- API server deployed on Railway
- TimescaleDB accessible from both
- Clerk account with OAuth configured

## Step 1: Configure Clerk OAuth for Grafana

### 1.1 Create OAuth Application in Clerk

Follow the guide in [`docs/grafana-clerk-oauth-setup.md`](./grafana-clerk-oauth-setup.md)

Key steps:
1. Create OAuth app in Clerk dashboard
2. Set callback URL: `https://your-grafana.up.railway.app/login/generic_oauth`
3. Select scopes: `openid`, `profile`, `email`, `public_metadata`
4. Save **Client ID** and **Client Secret**

### 1.2 Configure Grafana Environment Variables on Railway

In Railway, add these environment variables to your Grafana service:

```bash
# OAuth Config
GF_AUTH_GENERIC_OAUTH_ENABLED=true
GF_AUTH_GENERIC_OAUTH_NAME=Clerk
GF_AUTH_GENERIC_OAUTH_CLIENT_ID=oauth_xxxxx
GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=oauth_secret_xxxxx

# Clerk Endpoints
GF_AUTH_GENERIC_OAUTH_SCOPES=openid profile email public_metadata
GF_AUTH_GENERIC_OAUTH_AUTH_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/authorize
GF_AUTH_GENERIC_OAUTH_TOKEN_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/token
GF_AUTH_GENERIC_OAUTH_API_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/userinfo

# User Mapping
GF_AUTH_GENERIC_OAUTH_EMAIL_ATTRIBUTE_PATH=email
GF_AUTH_GENERIC_OAUTH_LOGIN_ATTRIBUTE_PATH=email
GF_AUTH_GENERIC_OAUTH_NAME_ATTRIBUTE_PATH=name

# Role Mapping (from Clerk public_metadata)
GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH=contains(public_metadata.roles[*], 'admin') && 'Admin' || 'Viewer'
GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP=true

# Database (use PostgreSQL for Grafana config)
GF_DATABASE_TYPE=postgres
GF_DATABASE_HOST=<your-postgres-host>
GF_DATABASE_NAME=<your-postgres-db>
GF_DATABASE_USER=<user>
GF_DATABASE_PASSWORD=<password>
GF_DATABASE_SSL_MODE=require

# Server
GF_SERVER_ROOT_URL=https://your-grafana.up.railway.app
```

## Step 2: Apply Row-Level Security Migration

### 2.1 Run the RLS Migration

Connect to your TimescaleDB instance and run:

```bash
psql -h <timescaledb-host> -U ts_user -d timeseries -f api-server/migrations/001_add_rls_policies.sql
```

Or via Railway CLI:

```bash
railway run psql $DATABASE_URL -f api-server/migrations/001_add_rls_policies.sql
```

This migration:
- Enables RLS on `pod_metrics` and `node_metrics` tables
- Creates policies to filter by `tenant_id`
- Adds helper functions for setting tenant context
- Creates indexes for performance

### 2.2 Verify RLS is Working

```sql
-- Test tenant isolation
SELECT set_tenant_context(1);
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Should only return: 1

-- Test admin mode
SELECT enable_admin_mode();
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Returns all tenant IDs
SELECT disable_admin_mode();
```

## Step 3: Set Up API Server Middleware

### 3.1 Update API Server to Set Tenant Context

The tenant context middleware is already created in:
- `api-server/internal/middleware/tenant_context.go`

### 3.2 Apply Middleware in Server Initialization

Edit `api-server/internal/api/server.go`:

```go
import (
    "github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
)

// After authentication middleware
router.Use(middleware.TenantContextMiddleware(timescaleDB))
```

This ensures that:
1. API key auth sets `tenant_id` in context
2. Tenant context middleware sets PostgreSQL session variable
3. All TimescaleDB queries are automatically filtered by tenant

## Step 4: Configure Grafana Organizations

### 4.1 Set Up Grafana Service in API Server

Edit `api-server/cmd/server/main.go`:

```go
import (
    "github.com/bugfreev587/k8s-cost-api-server/internal/services"
)

// Initialize Grafana service
grafanaService := services.NewGrafanaService(
    os.Getenv("GRAFANA_URL"),           // https://your-grafana.up.railway.app
    os.Getenv("GRAFANA_API_TOKEN"),     // Create in Grafana UI
    "",                                  // Or use username
    "",                                  // Or use password
)
```

### 4.2 Create Grafana API Token

1. Log in to Grafana as admin
2. Go to **Administration** → **Service Accounts**
3. Click **Add service account**
   - Name: `api-server`
   - Role: `Admin`
4. Click **Add service account token**
5. Copy the token and add to Railway env vars:
   ```bash
   GRAFANA_API_TOKEN=glsa_xxxxxxxxxxxxx
   GRAFANA_URL=https://your-grafana.up.railway.app
   ```

### 4.3 Sync Existing Tenants to Grafana

Run the TypeScript sync script:

```bash
cd scripts
npm install tsx pg dotenv
export GRAFANA_URL=https://your-grafana.up.railway.app
export GRAFANA_API_TOKEN=glsa_xxxxxx
export POSTGRES_DSN=postgresql://user:pass@host:5432/k8s_costs

tsx grafana-org-manager.ts sync-all
```

This creates a Grafana organization for each tenant in your database.

## Step 5: Set Up Clerk Webhooks

### 5.1 Configure Webhook Endpoint

In Clerk Dashboard:
1. Go to **Webhooks**
2. Click **Add Endpoint**
3. Set URL: `https://your-api-server.up.railway.app/webhooks/clerk`
4. Select events:
   - ✅ `user.created`
   - ✅ `user.updated`
   - ✅ `user.deleted`
5. Save and copy **Signing Secret**

### 5.2 Add Webhook Handler to API Server

The webhook handler is already created in:
- `api-server/internal/api/clerk_webhooks.go`

Register it in `api-server/cmd/server/main.go`:

```go
import (
    "github.com/bugfreev587/k8s-cost-api-server/internal/api"
)

// Initialize webhook handler
clerkHandler := api.NewClerkWebhookHandler(postgresDB, grafanaService)
api.RegisterClerkWebhookRoutes(router, clerkHandler)
```

### 5.3 Test Webhook

Create a test user in Clerk:
1. Sign up via your frontend
2. Check API server logs - should see:
   ```
   Received Clerk webhook: type=user.created
   Created new tenant 1 for user test@example.com
   Synced Grafana org: tenant_id=1, grafana_org_id=2
   ```

## Step 6: Set User Metadata

When a user signs up, set their `public_metadata` in Clerk:

### Option A: Via Clerk Dashboard (Testing)

1. Go to **Users** → Select user
2. Click **Metadata** tab
3. Add to **Public metadata**:
```json
{
  "tenant_id": "1",
  "roles": ["viewer"]
}
```

### Option B: Programmatically (Production)

In your frontend after signup:

```typescript
import { useAuth } from '@clerk/clerk-react';

const { getToken } = useAuth();

async function setUserMetadata(tenantId: number, roles: string[]) {
  const token = await getToken();

  const response = await fetch('https://your-api-server/v1/admin/users/metadata', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      email: user.emailAddress,
      tenant_id: tenantId,
      roles: roles
    })
  });

  const result = await response.json();

  // Now update Clerk via their API
  // You'll need to call Clerk's API from your backend
  console.log('Set this metadata in Clerk:', result.metadata);
}
```

### Option C: Via Clerk Backend API

In your API server:

```go
import "github.com/clerk/clerk-sdk-go/v2"

func updateClerkMetadata(userID string, tenantID uint, roles []string) error {
    client := clerk.NewClient(os.Getenv("CLERK_SECRET_KEY"))

    _, err := client.Users().Update(ctx, userID, &clerk.UpdateUserParams{
        PublicMetadata: map[string]interface{}{
            "tenant_id": tenantID,
            "roles": roles,
        },
    })

    return err
}
```

## Step 7: Create Grafana Dashboards

### 7.1 Configure TimescaleDB Data Source in Grafana

1. Log in to Grafana
2. Go to **Administration** → **Data sources**
3. Click **Add data source** → **PostgreSQL**
4. Configure:
   - **Name**: `TimescaleDB`
   - **Host**: `your-timescaledb-host:5433`
   - **Database**: `timeseries`
   - **User**: `ts_user`
   - **Password**: `ts_pass`
   - **SSL Mode**: `require` (production)
   - **Version**: `12+`
   - **TimescaleDB**: ✅ Enabled

### 7.2 Important: Handling Tenant Context in Grafana

**Challenge**: Grafana doesn't natively pass OAuth metadata to database queries.

**Solution Options**:

#### Option A: Backend Proxy (Recommended for Production)

Create a proxy endpoint in your API server:

```go
// POST /v1/grafana/query
func (s *Server) HandleGrafanaQuery(c *gin.Context) {
    // 1. Validate OAuth token from Grafana
    // 2. Extract tenant_id from JWT
    // 3. Set tenant context
    // 4. Execute query
    // 5. Return results
}
```

Then configure this as a JSON API data source in Grafana.

#### Option B: Grafana Backend Plugin (Advanced)

Create a custom Grafana backend data source plugin that:
- Intercepts queries
- Extracts tenant_id from user session
- Sets `app.current_tenant_id` before each query

#### Option C: Pre-Authenticated Connection Per Org (Workaround)

For each Grafana organization:
1. Create a separate PostgreSQL user
2. Set a default tenant context in the user's login script
3. Use organization-specific data sources

This is less scalable but works for small deployments.

### 7.3 Create Dashboard Using Example Queries

Use the queries from `api-server/docker/grafana/multi-tenant-queries.sql`

Example dashboard panels:
- CPU Usage Over Time
- Memory Usage Over Time
- Cost by Namespace (pie chart)
- Top Resource Consumers (table)
- Over-Provisioned Pods (recommendations)

## Step 8: Testing End-to-End

### 8.1 Test User Flow

1. **Sign up** in your frontend (Clerk auth)
2. **Verify** webhook created:
   - User in database
   - Tenant in database
   - Grafana organization created
3. **Log in to Grafana** using "Sign in with Clerk"
4. **View dashboards** - should only see your tenant's data

### 8.2 Test Multi-Tenancy

Create two test users:
- User A with `tenant_id: 1`
- User B with `tenant_id: 2`

Ingest metrics for both tenants:
```bash
# Tenant 1 data
curl -X POST https://api-server/v1/ingest \
  -H "Authorization: Bearer <tenant-1-api-key>" \
  -d '{"pod_metrics": [...], "node_metrics": [...]}'

# Tenant 2 data
curl -X POST https://api-server/v1/ingest \
  -H "Authorization: Bearer <tenant-2-api-key>" \
  -d '{"pod_metrics": [...], "node_metrics": [...]}'
```

Verify in Grafana:
- User A only sees tenant 1 data
- User B only sees tenant 2 data

### 8.3 Test RLS in Database

```sql
-- Set tenant 1 context
SELECT set_tenant_context(1);
SELECT COUNT(*) FROM pod_metrics;  -- e.g., 100 rows

-- Switch to tenant 2 context
SELECT set_tenant_context(2);
SELECT COUNT(*) FROM pod_metrics;  -- e.g., 50 rows (different data)

-- Clear context (no access)
SELECT clear_tenant_context();
SELECT COUNT(*) FROM pod_metrics;  -- 0 rows
```

## Troubleshooting

### Issue: "Sign in with Clerk" button not showing

**Solution**:
- Verify `GF_AUTH_GENERIC_OAUTH_ENABLED=true`
- Restart Grafana service
- Check Grafana logs for OAuth config errors

### Issue: User can see other tenants' data

**Solution**:
- Verify RLS policies are enabled: `SELECT tablename, rowsecurity FROM pg_tables WHERE tablename LIKE '%metrics';`
- Check tenant context is being set: Add logging to middleware
- Verify `tenant_id` is correct in user's Clerk metadata

### Issue: No data visible in Grafana

**Solution**:
- Check if tenant context is set (might be filtered to empty tenant)
- Use admin mode temporarily to verify data exists
- Check TimescaleDB data source connection

### Issue: Grafana organization not created

**Solution**:
- Verify `GRAFANA_API_TOKEN` is valid
- Check API server logs for errors
- Run sync script manually: `tsx grafana-org-manager.ts sync-all`

## Security Considerations

1. **API Keys**: Store Grafana API token securely in Railway environment variables
2. **Clerk Secret**: Never expose `CLERK_SECRET_KEY` in frontend
3. **Database Access**: Use SSL for all database connections in production
4. **RLS Policies**: Regularly audit policies and test with different tenant contexts
5. **Webhook Verification**: Verify Clerk webhook signatures (add signature validation to webhook handler)

## Next Steps

1. **Custom Dashboards**: Create tenant-specific dashboards for different use cases
2. **Alerting**: Set up Grafana alerts for cost spikes and over-provisioning
3. **Reporting**: Generate PDF reports for customers
4. **Usage Metrics**: Track which tenants are using Grafana most
5. **Cost Allocation**: Implement chargeback based on actual usage

## Maintenance

### Regular Tasks

- **Weekly**: Review Grafana logs for errors
- **Monthly**: Audit RLS policies and test tenant isolation
- **Quarterly**: Review and optimize database indexes
- **As needed**: Sync new tenants to Grafana organizations

### Monitoring

Monitor these metrics:
- Failed Clerk webhooks
- RLS policy performance (query times)
- Grafana organization count vs tenant count (should match)
- User login success rate

## Reference Files

- Clerk OAuth Setup: [`docs/grafana-clerk-oauth-setup.md`](./grafana-clerk-oauth-setup.md)
- RLS Migration: [`api-server/migrations/001_add_rls_policies.sql`](../api-server/migrations/001_add_rls_policies.sql)
- Grafana Service: [`api-server/internal/services/grafana_service.go`](../api-server/internal/services/grafana_service.go)
- Tenant Middleware: [`api-server/internal/middleware/tenant_context.go`](../api-server/internal/middleware/tenant_context.go)
- Webhook Handler: [`api-server/internal/api/clerk_webhooks.go`](../api-server/internal/api/clerk_webhooks.go)
- Sync Script: [`scripts/grafana-org-manager.ts`](../scripts/grafana-org-manager.ts)
- Example Queries: [`api-server/docker/grafana/multi-tenant-queries.sql`](../api-server/docker/grafana/multi-tenant-queries.sql)
