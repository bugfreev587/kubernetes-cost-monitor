-- clear-all-tables.sql
-- Clears ALL data from PostgreSQL and TimescaleDB tables
-- Run this to reset the database to a clean state
--
-- Usage:
--   PostgreSQL: psql -h localhost -p 5431 -U postgres -d k8s_cost -f clear-all-tables.sql
--   TimescaleDB: psql -h localhost -p 5433 -U postgres -d k8s_cost_metrics -f clear-all-tables.sql
--
-- Or connect to each database and run the relevant sections

\echo '========================================'
\echo 'Clearing all database tables...'
\echo '========================================'

-- ============================================
-- POSTGRESQL TABLES (k8s_cost database)
-- ============================================

\echo ''
\echo '--- Clearing PostgreSQL tables ---'

-- Disable triggers temporarily for faster deletion
SET session_replication_role = 'replica';

-- Delete in order respecting foreign key constraints
-- (child tables first, then parent tables)

-- 1. Clear users (references tenants)
\echo 'Clearing users table...'
DELETE FROM users;

-- 2. Clear api_keys (references tenants)
\echo 'Clearing api_keys table...'
DELETE FROM api_keys;

-- 3. Clear recommendations (references tenants)
\echo 'Clearing recommendations table...'
DELETE FROM recommendations;

-- 4. Clear tenants (parent table)
\echo 'Clearing tenants table...'
DELETE FROM tenants;

-- 5. Clear pricing_plans (standalone table)
\echo 'Clearing pricing_plans table...'
DELETE FROM pricing_plans;

-- Re-enable triggers
SET session_replication_role = 'origin';

-- Reset sequences
\echo ''
\echo '--- Resetting sequences ---'
ALTER SEQUENCE IF EXISTS tenants_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS api_keys_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS recommendations_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS pricing_plans_id_seq RESTART WITH 1;

-- ============================================
-- TIMESCALEDB TABLES (k8s_cost_metrics database)
-- ============================================

\echo ''
\echo '--- Clearing TimescaleDB tables ---'

-- Clear hypertables (TimescaleDB)
-- Use TRUNCATE for better performance on hypertables
\echo 'Clearing pod_metrics table...'
TRUNCATE TABLE pod_metrics;

\echo 'Clearing node_metrics table...'
TRUNCATE TABLE node_metrics;

-- ============================================
-- RE-SEED PRICING PLANS (required for app to work)
-- ============================================

\echo ''
\echo '--- Re-seeding pricing plans ---'

INSERT INTO pricing_plans (name, display_name, price_cents, cluster_limit, node_limit, user_limit, retention_days, features) VALUES
  ('Starter', 'Starter', 0, 1, 5, 1, 7, ARRAY['1 cluster', 'Up to 5 nodes', '7-day data retention', 'Basic cost tracking', 'Email support']),
  ('Premium', 'Premium', 4900, 10, 100, 10, 30, ARRAY['Up to 10 clusters', 'Up to 100 nodes', '30-day data retention', 'Advanced analytics', 'Cost optimization recommendations', 'Custom alerts']),
  ('Business', 'Business', 19900, -1, -1, -1, 365, ARRAY['Unlimited clusters', 'Unlimited nodes', '1 year data retention', 'Enterprise analytics', '24/7 support', 'Custom integrations', 'SLA guarantee'])
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- VERIFICATION
-- ============================================

\echo ''
\echo '========================================'
\echo 'Verification - Row counts:'
\echo '========================================'

SELECT 'pricing_plans' as table_name, COUNT(*) as row_count FROM pricing_plans
UNION ALL SELECT 'tenants', COUNT(*) FROM tenants
UNION ALL SELECT 'users', COUNT(*) FROM users
UNION ALL SELECT 'api_keys', COUNT(*) FROM api_keys
UNION ALL SELECT 'recommendations', COUNT(*) FROM recommendations
UNION ALL SELECT 'pod_metrics', COUNT(*) FROM pod_metrics
UNION ALL SELECT 'node_metrics', COUNT(*) FROM node_metrics;

\echo ''
\echo 'Database cleanup complete!'
\echo 'Note: pricing_plans has been re-seeded with default plans.'
