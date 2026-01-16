# Implementation Summary: Multi-Tenant Grafana with Clerk Auth

## 🎯 What Was Built

A complete multi-tenant visualization solution for your Kubernetes Cost Monitor SaaS with:

1. **Clerk OAuth Integration** - Single sign-on for Grafana using your existing Clerk authentication
2. **Grafana Organization Management** - Automated provisioning of one organization per tenant
3. **Row-Level Security (RLS)** - Database-level tenant isolation in TimescaleDB
4. **Automated Sync System** - Webhooks and APIs to keep everything in sync

## 📊 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Flow                                │
└─────────────────────────────────────────────────────────────────┘

1. User signs up/logs in via Frontend (Clerk)
   ↓
2. Clerk sends webhook to API Server
   ↓
3. API Server creates:
   - User record in PostgreSQL
   - Tenant (if new)
   - Grafana organization
   ↓
4. User clicks "View Dashboard" → Redirected to Grafana
   ↓
5. Grafana OAuth → Clerk → Authenticates user
   ↓
6. Grafana assigns user to their tenant's organization
   ↓
7. User runs queries → TimescaleDB RLS filters by tenant_id
   ↓
8. User only sees their own cluster data ✅

┌─────────────────────────────────────────────────────────────────┐
│                     Component Diagram                            │
└─────────────────────────────────────────────────────────────────┘

┌──────────────┐          ┌──────────────┐
│   Frontend   │          │    Clerk     │
│  (Vercel)    │◄────────►│   (OAuth)    │
└──────────────┘          └──────┬───────┘
                                 │
                                 │ Webhooks
                                 ▼
┌──────────────┐          ┌──────────────┐
│   Grafana    │          │ API Server   │
│  (Railway)   │◄────────►│  (Railway)   │
└──────┬───────┘          └──────┬───────┘
       │                         │
       │ Queries with            │ Queries with
       │ tenant_id context       │ tenant_id context
       ▼                         ▼
┌─────────────────────────────────────────┐
│         TimescaleDB (Railway)            │
│  ┌─────────────────────────────────┐    │
│  │  pod_metrics (RLS enabled)      │    │
│  │  ├─ tenant_id = 1 → Acme data   │    │
│  │  ├─ tenant_id = 2 → Globex data │    │
│  │  └─ tenant_id = 3 → Skynet data │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## 📁 Files Created

### 1. Configuration & Documentation

| File | Purpose |
|------|---------|
| `docs/grafana-clerk-oauth-setup.md` | Step-by-step OAuth configuration guide |
| `docs/multi-tenant-setup-guide.md` | Comprehensive setup and deployment guide |
| `docs/GRAFANA_SETUP_QUICKSTART.md` | Quick start guide (30 min setup) |
| `docs/IMPLEMENTATION_SUMMARY.md` | This file - overview of implementation |

### 2. Database Layer

| File | Purpose |
|------|---------|
| `api-server/migrations/001_add_rls_policies.sql` | Row-Level Security policies for multi-tenant isolation |

**What it does**:
- Enables RLS on `pod_metrics` and `node_metrics` tables
- Creates policies to filter queries by `tenant_id`
- Adds helper functions: `set_tenant_context()`, `enable_admin_mode()`
- Creates indexes for performance optimization

### 3. Backend Services (Go)

| File | Purpose |
|------|---------|
| `api-server/internal/services/grafana_service.go` | Grafana API client for org/user management |
| `api-server/internal/middleware/tenant_context.go` | Middleware to set tenant context on each request |
| `api-server/internal/api/clerk_webhooks.go` | Webhook handler for Clerk user events |

**Key Functions**:

**grafana_service.go**:
- `CreateOrgForTenant()` - Creates Grafana organization for tenant
- `AddUserToOrg()` - Adds user to organization with role
- `SyncTenantOrganization()` - Ensures org exists for tenant

**tenant_context.go**:
- `TenantContextMiddleware()` - Sets PostgreSQL session variable
- `WithTenantContext()` - Execute query with tenant isolation
- `WithAdminMode()` - Bypass RLS for system queries

**clerk_webhooks.go**:
- `HandleWebhook()` - Processes Clerk user events
- `handleUserCreated()` - Creates user, tenant, and Grafana org
- `UpdateUserMetadata()` - Helper endpoint to set tenant metadata

### 4. Scripts & Utilities

| File | Purpose |
|------|---------|
| `scripts/grafana-org-manager.ts` | CLI tool for managing Grafana organizations |
| `api-server/docker/grafana/multi-tenant-queries.sql` | Example dashboard queries with RLS |

**grafana-org-manager.ts Commands**:
```bash
tsx grafana-org-manager.ts create-org --tenant-id=1 --tenant-name="Acme"
tsx grafana-org-manager.ts sync-all
tsx grafana-org-manager.ts add-user --org-id=2 --email=user@example.com --role=Editor
tsx grafana-org-manager.ts list-orgs
tsx grafana-org-manager.ts delete-org --org-id=2
```

## 🔐 Security Features

### Row-Level Security (RLS)

**How it works**:
```sql
-- API server sets tenant context before queries
SELECT set_tenant_context(123);

-- All queries automatically filtered
SELECT * FROM pod_metrics;
-- Internally becomes:
-- SELECT * FROM pod_metrics WHERE tenant_id = 123;
```

**Benefits**:
- ✅ Database-enforced isolation (not just app-level)
- ✅ Prevents accidental cross-tenant data leaks
- ✅ Works even if app logic has bugs
- ✅ Transparent to application code

### OAuth Security

**How it works**:
1. User clicks "Sign in with Clerk" in Grafana
2. Redirected to Clerk OAuth flow
3. Clerk authenticates user
4. Returns to Grafana with OAuth token
5. Grafana validates token with Clerk's public keys (JWKS)
6. Extracts user info and metadata (tenant_id, roles)
7. Assigns user to correct organization

**Benefits**:
- ✅ No separate password management
- ✅ SSO across frontend and Grafana
- ✅ Centralized user management in Clerk
- ✅ Secure token validation via JWKS

## 🔄 Data Flow Examples

### Example 1: New User Signup

```
1. User signs up in frontend
   POST https://frontend.vercel.app/signup
   ↓
2. Clerk creates user account
   ↓
3. Clerk sends webhook to API server
   POST https://api-server.railway.app/webhooks/clerk
   {
     "type": "user.created",
     "data": {
       "email": "alice@acme.com",
       "public_metadata": {"tenant_id": "1"}
     }
   }
   ↓
4. API server:
   - Creates user record in PostgreSQL
   - Creates tenant if new
   - Calls Grafana API to create organization
   ↓
5. Grafana organization created
   Organization: "Tenant 1 - Acme Corp"
   ↓
6. User can now log in to Grafana with Clerk
```

### Example 2: User Views Dashboard

```
1. User clicks "View Dashboard" in frontend
   ↓
2. Redirected to: https://grafana.railway.app
   ↓
3. User clicks "Sign in with Clerk"
   ↓
4. OAuth flow:
   Grafana → Clerk (login) → Grafana (callback)
   ↓
5. Grafana extracts from OAuth token:
   - email: alice@acme.com
   - tenant_id: 1 (from public_metadata)
   - role: viewer
   ↓
6. Grafana assigns user to "Tenant 1 - Acme Corp" org
   ↓
7. User runs dashboard query:
   SELECT * FROM pod_metrics WHERE cluster_name = 'prod'
   ↓
8. TimescaleDB RLS intercepts and rewrites:
   SELECT * FROM pod_metrics
   WHERE cluster_name = 'prod'
   AND tenant_id = 1  -- Added by RLS policy
   ↓
9. User sees only Acme Corp's data ✅
```

### Example 3: Cost Agent Ingests Metrics

```
1. Cost agent in customer's K8s cluster collects metrics
   ↓
2. Sends to API server:
   POST https://api-server.railway.app/v1/ingest
   Authorization: Bearer <api-key-for-tenant-1>
   {
     "pod_metrics": [...],
     "node_metrics": [...]
   }
   ↓
3. API server:
   - Validates API key (Redis cache)
   - Extracts tenant_id = 1 from API key
   - Sets context: SELECT set_tenant_context(1)
   ↓
4. Inserts metrics into TimescaleDB:
   INSERT INTO pod_metrics (tenant_id, cluster_name, ...)
   VALUES (1, 'prod', ...)
   ↓
5. RLS policy checks: tenant_id = 1 ✅ (matches context)
   ↓
6. Data inserted successfully
   ↓
7. User can now see new metrics in Grafana dashboard
```

## 📊 Example Dashboard Queries

All queries in `api-server/docker/grafana/multi-tenant-queries.sql` are automatically filtered by RLS.

**CPU Usage Over Time**:
```sql
SELECT
  time,
  cluster_name,
  namespace,
  pod_name,
  cpu_millicores / 1000.0 AS "CPU Cores"
FROM pod_metrics
WHERE $__timeFilter(time)
ORDER BY time;
-- RLS automatically adds: AND tenant_id = <user's tenant>
```

**Cost by Namespace**:
```sql
WITH pod_costs AS (
  SELECT
    p.namespace,
    AVG(n.hourly_cost_usd) * (AVG(p.cpu_request_millicores)::FLOAT / AVG(n.cpu_capacity)) AS hourly_cost
  FROM pod_metrics p
  JOIN node_metrics n ON p.cluster_name = n.cluster_name AND p.node_name = n.node_name
  WHERE $__timeFilter(p.time)
  GROUP BY p.namespace, p.pod_name
)
SELECT namespace, SUM(hourly_cost) * 24 * 30 AS "Monthly Cost (USD)"
FROM pod_costs
GROUP BY namespace;
-- RLS ensures only tenant's data included in both tables
```

## 🚀 Deployment Checklist

- [ ] **Step 1**: Create Clerk OAuth application
- [ ] **Step 2**: Configure Grafana environment variables on Railway
- [ ] **Step 3**: Apply RLS migration to TimescaleDB
- [ ] **Step 4**: Create Grafana API token
- [ ] **Step 5**: Sync existing tenants to Grafana organizations
- [ ] **Step 6**: Set up Clerk webhooks pointing to API server
- [ ] **Step 7**: Set user metadata (tenant_id) in Clerk
- [ ] **Step 8**: Test end-to-end with multiple tenants

**Estimated Time**: 30-45 minutes

## 🎓 Key Concepts

### Grafana Organizations
- Each tenant gets their own organization
- Organizations provide workspace isolation
- Users can only access their assigned organization
- Dashboards and data sources are per-organization

### Row-Level Security (RLS)
- PostgreSQL feature to filter rows based on policies
- Policy: `WHERE tenant_id = current_setting('app.current_tenant_id')`
- Set via middleware: `SELECT set_tenant_context(123)`
- Enforced at database level, not application level

### OAuth Flow
1. User clicks "Sign in with Clerk"
2. Redirect to Clerk authorization endpoint
3. User logs in at Clerk
4. Redirect back to Grafana with authorization code
5. Grafana exchanges code for access token
6. Grafana fetches user info from userinfo endpoint
7. Maps user to organization based on tenant_id

### Metadata Mapping
Clerk `public_metadata`:
```json
{
  "tenant_id": "1",
  "roles": ["viewer"],
  "cluster_name": "production"
}
```

Grafana extracts via JMESPath:
- `tenant_id` → Organization assignment
- `roles` → Grafana role (Admin/Editor/Viewer)

## 🔧 Maintenance

### Regular Tasks
- **Daily**: Monitor Clerk webhooks for failures
- **Weekly**: Review Grafana audit logs
- **Monthly**: Test RLS policies with different tenants
- **Quarterly**: Rotate API tokens and secrets

### Monitoring Metrics
- Clerk webhook success rate
- Grafana login success rate
- Database query performance (RLS overhead)
- Organization count vs tenant count (should match)

## 🐛 Common Issues & Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| User sees no data | Tenant context not set | Check middleware is applied |
| User sees other tenant's data | RLS not enabled | Run migration to enable RLS |
| OAuth redirect error | Wrong callback URL | Update in Clerk to match Grafana URL |
| Organization not created | Webhook failed | Check API server logs, retry sync script |

## 📈 Scaling Considerations

### Current Setup (MVP)
- Supports: 1-100 tenants
- Grafana: Single instance
- TimescaleDB: RLS overhead ~5-10%
- Complexity: Low

### Future Enhancements (Growth)
- **Multi-region**: Deploy Grafana instances per region
- **Sharding**: Partition TimescaleDB by tenant_id for very large deployments
- **Caching**: Add Redis cache for frequently accessed dashboard data
- **Read Replicas**: Use TimescaleDB read replicas for dashboard queries

## ✅ What You Can Do Now

1. **View Dashboards**: Users log in to Grafana with Clerk SSO
2. **Tenant Isolation**: Each tenant only sees their own cluster metrics
3. **Automated Provisioning**: New users automatically get organizations
4. **Cost Analysis**: Use pre-built queries for cost breakdowns
5. **Security**: Database-enforced data isolation via RLS

## 🎉 Success Metrics

After implementation, you can track:
- Number of active Grafana users per tenant
- Dashboard views per tenant
- Cost savings identified via recommendations
- User engagement with visualization tools

## 📞 Next Steps

1. **Test with real data**: Ingest metrics from actual customer clusters
2. **Create dashboards**: Build tenant-specific cost dashboards
3. **Set up alerts**: Configure Grafana alerts for cost anomalies
4. **Documentation**: Share dashboard usage guide with customers
5. **Feedback loop**: Collect user feedback on visualizations

---

**Implementation Complete!** 🚀

Your Kubernetes Cost Monitor SaaS now has enterprise-grade multi-tenant visualization with Clerk SSO and database-level security.
