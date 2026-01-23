-- Migration: Add pricing plans and tenant pricing_plan
-- Applies to existing production PostgreSQL databases
-- Date: 2026-01-20

BEGIN;

-- Add pricing_plan column to tenants (default to Starter)
ALTER TABLE tenants
  ADD COLUMN IF NOT EXISTS pricing_plan TEXT DEFAULT 'Starter';

-- Backfill existing tenants that predate this column
UPDATE tenants
SET pricing_plan = 'Starter'
WHERE pricing_plan IS NULL;

-- Create pricing_plans table (idempotent)
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
INSERT INTO pricing_plans (
  name, display_name, price_cents, cluster_limit, node_limit, user_limit, retention_days, features
) VALUES
  ('Starter', 'Starter', 0, 1, 5, 1, 7, ARRAY['1 cluster', 'Up to 5 nodes', '7-day data retention', 'Basic cost tracking', 'Email support']),
  ('Premium', 'Premium', 4900, 10, 100, 10, 30, ARRAY['Up to 10 clusters', 'Up to 100 nodes', '30-day data retention', 'Advanced analytics', 'Cost optimization recommendations', 'Custom alerts']),
  ('Business', 'Business', 19900, -1, -1, -1, 365, ARRAY['Unlimited clusters', 'Unlimited nodes', '1 year data retention', 'Enterprise analytics', '24/7 support', 'Custom integrations', 'SLA guarantee'])
ON CONFLICT (name) DO NOTHING;

COMMIT;
