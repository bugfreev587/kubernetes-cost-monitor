# Grafana + Clerk OAuth Integration Setup Guide

## Step 1: Create OAuth Application in Clerk

### 1.1 Access Clerk Dashboard
1. Go to [Clerk Dashboard](https://dashboard.clerk.com)
2. Select your application (settling-magpie-54)
3. Navigate to **JWT Templates** → **New Template** → **Blank**

### 1.2 Create Custom JWT Template for Grafana
Name: `grafana-jwt`

Add these claims to include tenant info:
```json
{
  "tenant_id": "{{user.public_metadata.tenant_id}}",
  "roles": "{{user.public_metadata.roles}}",
  "cluster_name": "{{user.public_metadata.cluster_name}}"
}
```

### 1.3 Create OAuth Application
1. Go to **OAuth Applications** in sidebar
2. Click **"Add OAuth Application"**
3. Configure:
   - **Name**: `Grafana`
   - **Application Type**: Web Application
   - **Callback URLs**:
     - Development: `http://localhost:3000/login/generic_oauth`
     - Production: `https://your-grafana-domain.up.railway.app/login/generic_oauth`
   - **Scopes**: Select:
     - ✅ `openid` (required)
     - ✅ `profile` (required)
     - ✅ `email` (required)
     - ✅ `public_metadata` (for tenant_id and roles)

4. Click **Create**
5. **IMPORTANT**: Copy the **Client ID** and **Client Secret** immediately
   - Client ID: `oauth_xxxxxxxxxxxxx`
   - Client Secret: `oauth_secret_xxxxxxxxxxxxx`

## Step 2: Railway Environment Variables for Grafana

Add these environment variables to your Grafana service on Railway:

```bash
# === Basic OAuth Config ===
GF_AUTH_GENERIC_OAUTH_ENABLED=true
GF_AUTH_GENERIC_OAUTH_NAME=Clerk
GF_AUTH_GENERIC_OAUTH_ICON=signin
GF_AUTH_GENERIC_OAUTH_CLIENT_ID=<YOUR_CLERK_OAUTH_CLIENT_ID>
GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=<YOUR_CLERK_OAUTH_CLIENT_SECRET>

# === Clerk OAuth Endpoints ===
GF_AUTH_GENERIC_OAUTH_SCOPES=openid profile email public_metadata
GF_AUTH_GENERIC_OAUTH_AUTH_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/authorize
GF_AUTH_GENERIC_OAUTH_TOKEN_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/token
GF_AUTH_GENERIC_OAUTH_API_URL=https://settling-magpie-54.clerk.accounts.dev/oauth/userinfo

# === User Attribute Mapping (JMESPath) ===
GF_AUTH_GENERIC_OAUTH_EMAIL_ATTRIBUTE_PATH=email
GF_AUTH_GENERIC_OAUTH_LOGIN_ATTRIBUTE_PATH=preferred_username || username || email
GF_AUTH_GENERIC_OAUTH_NAME_ATTRIBUTE_PATH=name

# === Role Mapping ===
# Maps public_metadata.roles array to Grafana roles (Admin, Editor, Viewer)
GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH=contains(public_metadata.roles[*], 'admin') && 'Admin' || contains(public_metadata.roles[*], 'editor') && 'Editor' || 'Viewer'
GF_AUTH_GENERIC_OAUTH_ALLOW_ASSIGN_GRAFANA_ADMIN=true
GF_AUTH_GENERIC_OAUTH_AUTO_ASSIGN_ORG_ROLE=Viewer
GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_STRICT=false

# === Organization Mapping (Multi-Tenant) ===
# Maps users to Grafana organizations based on grafana_org_id
# IMPORTANT: Use grafana_org_id (not tenant_id) - this is the actual Grafana org ID
GF_AUTH_GENERIC_OAUTH_ORG_ATTRIBUTE_PATH=public_metadata.grafana_org_id

# === OAuth Settings ===
GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP=true
GF_AUTH_GENERIC_OAUTH_USE_REFRESH_TOKEN=true
GF_AUTH_GENERIC_OAUTH_EMPTY_SCOPES=false

# === Server Settings ===
GF_SERVER_ROOT_URL=https://your-grafana-domain.up.railway.app
GF_SERVER_DOMAIN=your-grafana-domain.up.railway.app

# === Security ===
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=<GENERATE_STRONG_PASSWORD>
GF_SECURITY_SECRET_KEY=<GENERATE_RANDOM_SECRET>

# === Database (Use PostgreSQL for config storage) ===
GF_DATABASE_TYPE=postgres
GF_DATABASE_HOST=<your-postgres-host>:<port>
GF_DATABASE_NAME=<your-postgres-db>
GF_DATABASE_USER=<postgres-user>
GF_DATABASE_PASSWORD=<postgres-password>
GF_DATABASE_SSL_MODE=require  # For Railway/production
```

## Step 3: Set Up Clerk User Metadata

### 3.1 User Metadata Structure

When a user signs up, their Clerk `public_metadata` is automatically set by the API server and includes:

```json
{
  "tenant_id": 123,
  "role": "viewer",
  "roles": ["viewer"],
  "grafana_org_id": 5
}
```

### 3.2 Set Metadata via Clerk Dashboard (Manual Testing)

1. Go to **Users** in Clerk Dashboard
2. Select a user
3. Click **Metadata** tab
4. Add to **Public metadata**:
```json
{
  "tenant_id": "1",
  "roles": ["admin"]
}
```

### 3.3 Set Metadata Programmatically (Production)

See the API endpoint created in `api-server/internal/api/clerk_webhooks.go` to automatically set metadata when users sign up.

## Step 4: Testing the Integration

1. Restart Grafana on Railway
2. Navigate to your Grafana URL
3. You should see **"Sign in with Clerk"** button
4. Click it → redirects to Clerk login
5. Sign in with test user
6. Should redirect back to Grafana and be logged in

## Step 5: Verify User Attributes

In Grafana, after login:
1. Go to **Administration** → **Users**
2. Find your user
3. Verify:
   - Email is populated
   - Role is assigned correctly (Admin/Editor/Viewer)
   - Organization is assigned (based on tenant_id)

## Troubleshooting

### Common Issues

1. **"Sign in with Clerk" button not appearing**
   - Check `GF_AUTH_GENERIC_OAUTH_ENABLED=true` is set
   - Restart Grafana service

2. **OAuth redirect error**
   - Verify callback URL in Clerk matches: `https://your-domain/login/generic_oauth`
   - Check no trailing slashes

3. **User role not assigned**
   - Verify user has `public_metadata.roles` set in Clerk
   - Check JMESPath expression in `GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH`

4. **User not assigned to organization / "User sync failed"**
   - Ensure Grafana organization exists for the tenant
   - Run the migration: `007_add_grafana_org_id.sql`
   - Update the tenant's `grafana_org_id` in the database with the actual Grafana org ID
   - Update Grafana env var: `GF_AUTH_GENERIC_OAUTH_ORG_ATTRIBUTE_PATH=public_metadata.grafana_org_id`
   - Have the user sign out and sign back in to trigger metadata sync

### Debug Logs

Enable Grafana OAuth debug logs:
```bash
GF_LOG_LEVEL=debug
GF_LOG_FILTERS=oauth:debug
```

Check Railway logs for OAuth flow details.

## Security Notes

1. **Never commit secrets** - Use Railway's environment variables
2. **Rotate Client Secret** - Periodically regenerate OAuth client secret in Clerk
3. **Use HTTPS only** - Ensure all redirects use HTTPS in production
4. **Validate JWT** - Grafana automatically validates JWT signatures via JWKS

## Next Steps

- Set up automated Grafana organization provisioning (see grafana-org-sync script)
- Configure row-level security in TimescaleDB for multi-tenant data isolation
- Create custom Grafana dashboards filtered by tenant_id
