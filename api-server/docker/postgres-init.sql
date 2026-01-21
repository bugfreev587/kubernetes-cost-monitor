-- postgres-init.sql
\echo "Initializing k8s_cost database..."

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================
-- Pricing Plans Table
-- ============================
CREATE TABLE IF NOT EXISTS pricing_plans (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,            -- 'Starter', 'Premium', 'Business'
  display_name TEXT NOT NULL,
  price_cents INTEGER NOT NULL,         -- 0, 4900, 19900
  cluster_limit INTEGER NOT NULL,       -- -1 = unlimited
  node_limit INTEGER NOT NULL,
  user_limit INTEGER NOT NULL,
  retention_days INTEGER NOT NULL,
  features TEXT[],                      -- Array of feature descriptions
  created_at timestamptz DEFAULT now()
);

-- Seed default pricing plans
INSERT INTO pricing_plans (name, display_name, price_cents, cluster_limit, node_limit, user_limit, retention_days, features) VALUES
  ('Starter', 'Starter', 0, 1, 5, 1, 7, ARRAY['1 cluster', 'Up to 5 nodes', '7-day data retention', 'Basic cost tracking', 'Email support']),
  ('Premium', 'Premium', 4900, 10, 100, 10, 30, ARRAY['Up to 10 clusters', 'Up to 100 nodes', '30-day data retention', 'Advanced analytics', 'Cost optimization recommendations', 'Custom alerts']),
  ('Business', 'Business', 19900, -1, -1, -1, 365, ARRAY['Unlimited clusters', 'Unlimited nodes', '1 year data retention', 'Enterprise analytics', '24/7 support', 'Custom integrations', 'SLA guarantee']);

CREATE TABLE IF NOT EXISTS tenants ( -- Changed id to BIGSERIAL for consistency
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  pricing_plan TEXT DEFAULT 'Starter',  -- References pricing_plans.name: 'Starter', 'Premium', 'Business'
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'viewer',  -- 'admin', 'editor', 'viewer'
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, email)
);

CREATE TABLE IF NOT EXISTS api_keys (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT REFERENCES tenants(id) ON DELETE CASCADE,
  key_id UUID NOT NULL UNIQUE,
  salt BYTEA NOT NULL,
  secret_hash BYTEA NOT NULL,
  scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
  revoked BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at timestamptz NULL,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendations (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT REFERENCES tenants(id) ON DELETE CASCADE,
  created_at timestamptz DEFAULT now(),
  cluster_name TEXT,
  namespace TEXT,
  pod_name TEXT,
  resource_type TEXT,
  current_request BIGINT,
  recommended_request BIGINT,
  potential_savings_usd NUMERIC(12,4),
  confidence NUMERIC(4,3),
  reason TEXT,
  status TEXT DEFAULT 'open'
);

\echo "k8s_cost database initialized."

-- ============================
-- Test Data: Tenants
-- ============================
INSERT INTO tenants (id, name, pricing_plan) VALUES
  (1, 'Acme Corporation', 'Premium'),
  (2, 'Globex Industries', 'Business'),
  (3, 'Skynet Systems', 'Starter');

-- ============================
-- Test Data: Users
-- ============================
-- password is 'password' for all users
-- First user of each tenant is admin, others are viewers
INSERT INTO users (tenant_id, email, password_hash, role) VALUES
  (1, 'wile.coyote@acme.com', crypt('password', gen_salt('bf')), 'admin'),
  (1, 'road.runner@acme.com', crypt('password', gen_salt('bf')), 'viewer'),
  (2, 'hank.scorpio@globex.com', crypt('password', gen_salt('bf')), 'admin'),
  (3, 'miles.dyson@skynet.com', crypt('password', gen_salt('bf')), 'admin');

-- ============================
-- Test Data: API Keys
-- ============================
-- store a fake secret (never used in prod)
INSERT INTO api_keys (tenant_id, key_id, salt, secret_hash, scopes, revoked, expires_at)
VALUES
  (
    1,
    gen_random_uuid(),
    gen_random_bytes(16),
    digest('super-secret-key-acme', 'sha256'),
    ARRAY['metrics:read', 'recommendations:read'],
    FALSE,
    now() + interval '90 days'
  ),
  (
    2,
    gen_random_uuid(),
    gen_random_bytes(16),
    digest('globex-test-key', 'sha256'),
    ARRAY['*'],
    FALSE,
    now() + interval '365 days'
  ),
  (
    3,
    gen_random_uuid(),
    gen_random_bytes(16),
    digest('terminator-key', 'sha256'),
    ARRAY['metrics:read'],
    TRUE,
    NULL
  );

-- ============================
-- Test Data: Recommendations
-- ============================
INSERT INTO recommendations (
  tenant_id, cluster_name, namespace, pod_name, resource_type,
  current_request, recommended_request, potential_savings_usd,
  confidence, reason, status
)
VALUES
  (1, 'cluster-a', 'default', 'api-server-1', 'cpu', 500, 250, 12.55, 0.92, 'CPU consistently underutilized', 'open'),
  (1, 'cluster-a', 'ml', 'training-pod', 'memory', 2048, 1024, 8.32, 0.87, 'Memory spikes rare', 'open'),
  (2, 'cluster-b', 'analytics', 'spark-worker-3', 'cpu', 2000, 1500, 21.78, 0.75, 'Workload stabilized', 'closed'),
  (3, 'skynet-cluster', 'war', 't800-control', 'cpu', 1000, 750, 5.00, 0.80, 'AI load reduced', 'open');