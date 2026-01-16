# Grafana Multi-Tenant Setup - Quick Start

Quick reference for setting up Grafana with Clerk authentication and multi-tenant data isolation.

## 🎯 What You Get

- **Clerk OAuth Login** for Grafana (same auth as your frontend)
- **Automatic tenant isolation** - each customer only sees their own data
- **Grafana organizations** - one per tenant
- **Row-level security** in TimescaleDB

## 📋 Prerequisites

- [ ] Grafana running on Railway
- [ ] API server running on Railway
- [ ] TimescaleDB accessible
- [ ] Clerk account configured

## 🚀 Quick Setup (30 minutes)

### 1. Clerk OAuth Application (5 min)

1. Go to [Clerk Dashboard](https://dashboard.clerk.com) → **OAuth Applications**
2. Click **Add OAuth Application**
   - Name: `Grafana`
   - Callback URL: `https://your-grafana.up.railway.app/login/generic_oauth`
   - Scopes: `openid`, `profile`, `email`, `public_metadata`
3. Save **Client ID** and **Client Secret**

### 2. Configure Grafana on Railway (5 min)

Add these environment variables in Railway:

```bash
# OAuth
GF_AUTH_GENERIC_OAUTH_ENABLED=true
GF_AUTH_GENERIC_OAUTH_NAME=Clerk
GF_AUTH_GENERIC_OAUTH_CLIENT_ID=<your-clerk-client-id>
GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=<your-clerk-client-secret>

# Clerk Endpoints
GF_AUTH_GENERIC_OAUTH_SCOPES=openid profile email public_metadata
GF_AUTH_GENERIC_OAUTH_AUTH_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/authorize
GF_AUTH_GENERIC_OAUTH_TOKEN_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/token
GF_AUTH_GENERIC_OAUTH_API_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/userinfo

# User Mapping
GF_AUTH_GENERIC_OAUTH_EMAIL_ATTRIBUTE_PATH=email
GF_AUTH_GENERIC_OAUTH_LOGIN_ATTRIBUTE_PATH=email
GF_AUTH_GENERIC_OAUTH_NAME_ATTRIBUTE_PATH=name
GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP=true

# Role Mapping
GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH=contains(public_metadata.roles[*], 'admin') && 'Admin' || 'Viewer'
```

**Note**: Replace `settling-magpie-54` with your Clerk domain.

Restart Grafana. You should now see "Sign in with Clerk" button.

### 3. Apply Database Row-Level Security (5 min)

```bash
# Connect to TimescaleDB and run migration
psql -h <your-timescaledb-host> -U ts_user -d timeseries -f api-server/migrations/001_add_rls_policies.sql
```

Or via Railway:
```bash
railway run psql $DATABASE_URL -f api-server/migrations/001_add_rls_policies.sql
```

Test it works:
```sql
SELECT set_tenant_context(1);
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Should only return: 1
```

### 4. Create Grafana API Token (3 min)

1. Log in to Grafana as admin
2. **Administration** → **Service Accounts** → **Add service account**
   - Name: `api-server`
   - Role: `Admin`
3. **Add service account token**
4. Copy token and add to Railway env vars:
   ```bash
   GRAFANA_API_TOKEN=glsa_xxxxx
   GRAFANA_URL=https://your-grafana.up.railway.app
   ```

### 5. Sync Tenants to Grafana Organizations (5 min)

```bash
cd scripts
npm install tsx pg dotenv

# Set environment variables
export GRAFANA_URL=https://your-grafana.up.railway.app
export GRAFANA_API_TOKEN=glsa_xxxxx
export POSTGRES_DSN=postgresql://user:pass@host:5432/k8s_costs

# Run sync
tsx grafana-org-manager.ts sync-all
```

This creates a Grafana organization for each tenant in your database.

### 6. Set User Metadata in Clerk (2 min)

For each user, set their `public_metadata`:

**Via Clerk Dashboard** (for testing):
1. **Users** → Select user → **Metadata** tab
2. Add to **Public metadata**:
```json
{
  "tenant_id": "1",
  "roles": ["viewer"]
}
```

**Programmatically** (for production):
- Use the webhook handler in `api-server/internal/api/clerk_webhooks.go`
- Or call Clerk's API from your backend

### 7. Test It! (5 min)

1. Go to your Grafana URL
2. Click **"Sign in with Clerk"**
3. Log in with a test user
4. You should be logged in and assigned to the correct organization
5. Create a test dashboard and verify you only see your tenant's data

## 📚 Detailed Guides

- **Full Setup Guide**: [docs/multi-tenant-setup-guide.md](./multi-tenant-setup-guide.md)
- **Clerk OAuth Details**: [docs/grafana-clerk-oauth-setup.md](./grafana-clerk-oauth-setup.md)

## 🔧 Created Files

### Configuration & Migrations
- `api-server/migrations/001_add_rls_policies.sql` - Database row-level security
- `docs/grafana-clerk-oauth-setup.md` - Detailed OAuth setup instructions

### Backend Services
- `api-server/internal/services/grafana_service.go` - Grafana API client
- `api-server/internal/middleware/tenant_context.go` - Tenant context middleware
- `api-server/internal/api/clerk_webhooks.go` - Clerk webhook handler

### Scripts & Queries
- `scripts/grafana-org-manager.ts` - Organization management CLI
- `api-server/docker/grafana/multi-tenant-queries.sql` - Example dashboard queries

## 🛠️ Common Tasks

### Sync a Single Tenant
```bash
tsx scripts/grafana-org-manager.ts create-org --tenant-id=1 --tenant-name="Acme Corp"
```

### Add User to Organization
```bash
tsx scripts/grafana-org-manager.ts add-user --org-id=2 --email=user@example.com --role=Editor
```

### List All Organizations
```bash
tsx scripts/grafana-org-manager.ts list-orgs
```

### Test RLS Policies
```sql
-- Tenant 1 context
SELECT set_tenant_context(1);
SELECT COUNT(*) FROM pod_metrics;

-- Switch to tenant 2
SELECT set_tenant_context(2);
SELECT COUNT(*) FROM pod_metrics;

-- Admin mode (see all data)
SELECT enable_admin_mode();
SELECT COUNT(*) FROM pod_metrics;
SELECT disable_admin_mode();
```

## 🔒 Security Checklist

- [ ] RLS policies enabled on all metrics tables
- [ ] Grafana OAuth configured with correct callback URL
- [ ] API tokens stored in Railway environment variables (not in code)
- [ ] Database connections use SSL in production
- [ ] Clerk webhook signatures verified (add this to webhook handler)
- [ ] User metadata (`tenant_id`) set for all users
- [ ] Test with multiple tenants to verify isolation

## 🐛 Troubleshooting

| Problem | Solution |
|---------|----------|
| "Sign in with Clerk" not showing | Check `GF_AUTH_GENERIC_OAUTH_ENABLED=true` and restart Grafana |
| User sees other tenants' data | Verify RLS policies enabled and tenant context is set |
| Webhook not creating org | Check `GRAFANA_API_TOKEN` is valid and has Admin role |
| No data in Grafana | Verify TimescaleDB data source connection and RLS context |

## 📞 Support

For detailed explanations and advanced configuration, see:
- [Multi-Tenant Setup Guide](./multi-tenant-setup-guide.md)
- [Grafana Clerk OAuth Setup](./grafana-clerk-oauth-setup.md)

## 🎉 What's Next?

Once setup is complete:
1. Create dashboards for your customers
2. Set up alerts for cost spikes
3. Configure email reports
4. Monitor tenant usage and growth

Your multi-tenant SaaS Grafana is ready! 🚀
