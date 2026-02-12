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

-- Seed default pricing plans (idempotent - skip if already exists)
INSERT INTO pricing_plans (name, display_name, price_cents, cluster_limit, node_limit, user_limit, retention_days, features)
VALUES
  ('Starter', 'Starter', 0, 1, 5, 1, 7, ARRAY['1 cluster', 'Up to 5 nodes', '7-day data retention', 'Basic cost tracking', 'Email support']),
  ('Premium', 'Premium', 4900, 10, 100, 10, 30, ARRAY['Up to 10 clusters', 'Up to 100 nodes', '30-day data retention', 'Advanced analytics', 'Cost optimization recommendations', 'Custom alerts']),
  ('Business', 'Business', 19900, -1, -1, -1, 365, ARRAY['Unlimited clusters', 'Unlimited nodes', '1 year data retention', 'Enterprise analytics', '24/7 support', 'Custom integrations', 'SLA guarantee'])
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenants (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  pricing_plan TEXT DEFAULT 'Starter',  -- References pricing_plans.name: 'Starter', 'Premium', 'Business'
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,  -- Clerk user ID (e.g., 'user_2lLjFe4cXYZ...')
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  name TEXT,
  role TEXT NOT NULL DEFAULT 'viewer',  -- 'owner', 'admin', 'editor', 'viewer'
  status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'suspended'
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT users_role_check CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
  CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended', 'pending'))
);

CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

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

-- ============================
-- Cloud Pricing Configuration Tables
-- ============================

-- Pricing configurations for different cloud providers/regions
CREATE TABLE IF NOT EXISTS pricing_configs (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  provider VARCHAR(20) NOT NULL,  -- aws, gcp, azure, oci, custom
  region VARCHAR(50),
  is_default BOOLEAN DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),
  UNIQUE(tenant_id, name)
);

-- Ensure only one default per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_pricing_configs_default
  ON pricing_configs(tenant_id) WHERE is_default = true;

-- Pricing rates for each configuration
CREATE TABLE IF NOT EXISTS pricing_rates (
  id BIGSERIAL PRIMARY KEY,
  config_id BIGINT NOT NULL REFERENCES pricing_configs(id) ON DELETE CASCADE,
  resource_type VARCHAR(20) NOT NULL,  -- cpu, memory, gpu, storage, network
  pricing_tier VARCHAR(20) DEFAULT 'on_demand',  -- on_demand, spot, reserved_1yr, reserved_3yr
  instance_family VARCHAR(50),  -- m5, c5, n1-standard, e2 (NULL = default for all)
  unit VARCHAR(20) NOT NULL,  -- core-hour, gb-hour, gpu-hour
  cost_per_unit DECIMAL(12,8) NOT NULL,
  effective_from DATE DEFAULT CURRENT_DATE,
  effective_to DATE,  -- NULL = currently active
  created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pricing_rates_config ON pricing_rates(config_id);
CREATE INDEX IF NOT EXISTS idx_pricing_rates_effective ON pricing_rates(config_id, effective_from, effective_to);

-- Cluster to pricing config mapping
CREATE TABLE IF NOT EXISTS cluster_pricing (
  cluster_name VARCHAR(255) NOT NULL,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  config_id BIGINT NOT NULL REFERENCES pricing_configs(id) ON DELETE CASCADE,
  created_at timestamptz DEFAULT now(),
  PRIMARY KEY(cluster_name, tenant_id)
);

-- Node-level pricing overrides (for mixed instance types/spot nodes)
CREATE TABLE IF NOT EXISTS node_pricing (
  id BIGSERIAL PRIMARY KEY,
  node_name VARCHAR(255) NOT NULL,
  cluster_name VARCHAR(255) NOT NULL,
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  instance_type VARCHAR(50),
  pricing_tier VARCHAR(20) DEFAULT 'on_demand',
  hourly_cost_override DECIMAL(10,6),  -- Direct cost override if known
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),
  UNIQUE(node_name, cluster_name, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_node_pricing_cluster ON node_pricing(cluster_name, tenant_id);

\echo "k8s_cost database initialized."

-- -- ============================
-- -- Test Data: Tenants
-- -- ============================
-- INSERT INTO tenants (id, name, pricing_plan) VALUES
--   (1, 'Acme Corporation', 'Premium'),
--   (2, 'Globex Industries', 'Business'),
--   (3, 'Skynet Systems', 'Starter');

-- -- ============================
-- -- Test Data: Users
-- -- ============================
-- -- First user of each tenant is owner, others are viewers
-- -- id is the Clerk user ID (using test IDs here)
-- INSERT INTO users (id, tenant_id, email, name, role, status) VALUES
--   ('user_test_wile_coyote', 1, 'wile.coyote@acme.com', 'Wile E. Coyote', 'owner', 'active'),
--   ('user_test_road_runner', 1, 'road.runner@acme.com', 'Road Runner', 'viewer', 'active'),
--   ('user_test_hank_scorpio', 2, 'hank.scorpio@globex.com', 'Hank Scorpio', 'owner', 'active'),
--   ('user_test_miles_dyson', 3, 'miles.dyson@skynet.com', 'Miles Dyson', 'owner', 'active');

-- -- ============================
-- -- Test Data: API Keys
-- -- ============================
-- -- store a fake secret (never used in prod)
-- INSERT INTO api_keys (tenant_id, key_id, salt, secret_hash, scopes, revoked, expires_at)
-- VALUES
--   (
--     1,
--     gen_random_uuid(),
--     gen_random_bytes(16),
--     digest('super-secret-key-acme', 'sha256'),
--     ARRAY['metrics:read', 'recommendations:read'],
--     FALSE,
--     now() + interval '90 days'
--   ),
--   (
--     2,
--     gen_random_uuid(),
--     gen_random_bytes(16),
--     digest('globex-test-key', 'sha256'),
--     ARRAY['*'],
--     FALSE,
--     now() + interval '365 days'
--   ),
--   (
--     3,
--     gen_random_uuid(),
--     gen_random_bytes(16),
--     digest('terminator-key', 'sha256'),
--     ARRAY['metrics:read'],
--     TRUE,
--     NULL
--   );

-- -- ============================
-- -- Test Data: Recommendations
-- -- ============================
-- INSERT INTO recommendations (
--   tenant_id, cluster_name, namespace, pod_name, resource_type,
--   current_request, recommended_request, potential_savings_usd,
--   confidence, reason, status
-- )
-- VALUES
--   (1, 'cluster-a', 'default', 'api-server-1', 'cpu', 500, 250, 12.55, 0.92, 'CPU consistently underutilized', 'open'),
--   (1, 'cluster-a', 'ml', 'training-pod', 'memory', 2048, 1024, 8.32, 0.87, 'Memory spikes rare', 'open'),
--   (2, 'cluster-b', 'analytics', 'spark-worker-3', 'cpu', 2000, 1500, 21.78, 0.75, 'Workload stabilized', 'closed'),
--   (3, 'skynet-cluster', 'war', 't800-control', 'cpu', 1000, 750, 5.00, 0.80, 'AI load reduced', 'open');