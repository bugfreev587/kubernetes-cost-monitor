-- Clear TimescaleDB Data
-- This script clears all data from TimescaleDB hypertables while preserving the schema
-- Use with caution in production!

-- Option 1: Clear all data from both tables
TRUNCATE TABLE pod_metrics;
TRUNCATE TABLE node_metrics;

-- Option 2: Clear data for a specific tenant (uncomment to use)
-- DELETE FROM pod_metrics WHERE tenant_id = 1;
-- DELETE FROM node_metrics WHERE tenant_id = 1;

-- Option 3: Clear data older than a specific time (uncomment to use)
-- DELETE FROM pod_metrics WHERE time < NOW() - INTERVAL '7 days';
-- DELETE FROM node_metrics WHERE time < NOW() - INTERVAL '7 days';

-- Option 4: Clear data for a specific cluster (uncomment to use)
-- DELETE FROM pod_metrics WHERE cluster_name = 'cluster-a';
-- DELETE FROM node_metrics WHERE cluster_name = 'cluster-a';

-- Verify tables are empty
SELECT 'pod_metrics' as table_name, COUNT(*) as row_count FROM pod_metrics
UNION ALL
SELECT 'node_metrics' as table_name, COUNT(*) as row_count FROM node_metrics;

