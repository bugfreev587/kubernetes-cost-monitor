-- clear-postgres-only.sql
-- Clears PostgreSQL tables only (users, tenants, api_keys, recommendations)
-- Preserves pricing_plans and TimescaleDB metrics data
--
-- Usage:
--   psql -h localhost -p 5431 -U postgres -d k8s_cost -f clear-postgres-only.sql

\echo '========================================'
\echo 'Clearing PostgreSQL user/tenant data...'
\echo '========================================'

-- Disable triggers temporarily for faster deletion
SET session_replication_role = 'replica';

-- Delete in order respecting foreign key constraints

\echo 'Clearing users table...'
DELETE FROM users;

\echo 'Clearing api_keys table...'
DELETE FROM api_keys;

\echo 'Clearing recommendations table...'
DELETE FROM recommendations;

\echo 'Clearing tenants table...'
DELETE FROM tenants;

-- Re-enable triggers
SET session_replication_role = 'origin';

-- Reset sequences
\echo ''
\echo '--- Resetting sequences ---'
ALTER SEQUENCE IF EXISTS tenants_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS api_keys_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS recommendations_id_seq RESTART WITH 1;

-- Verification
\echo ''
\echo '--- Verification ---'
SELECT 'tenants' as table_name, COUNT(*) as row_count FROM tenants
UNION ALL SELECT 'users', COUNT(*) FROM users
UNION ALL SELECT 'api_keys', COUNT(*) FROM api_keys
UNION ALL SELECT 'recommendations', COUNT(*) FROM recommendations;

\echo ''
\echo 'PostgreSQL cleanup complete!'
\echo 'Note: pricing_plans and TimescaleDB metrics were preserved.'
