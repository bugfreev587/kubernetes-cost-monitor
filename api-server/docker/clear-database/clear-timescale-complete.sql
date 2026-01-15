-- Complete TimescaleDB Reset
-- WARNING: This will DROP and recreate all tables and hypertables!
-- Use only in development/testing environments!

-- Drop existing hypertables (this will delete all data and schema)
DROP TABLE IF EXISTS pod_metrics CASCADE;
DROP TABLE IF EXISTS node_metrics CASCADE;

-- Recreate pod_metrics table and hypertable
CREATE TABLE pod_metrics (
  time timestamptz NOT NULL,
  tenant_id BIGINT NOT NULL,
  cluster_name TEXT,
  namespace TEXT,
  pod_name TEXT,
  node_name TEXT,
  cpu_millicores BIGINT,
  memory_bytes BIGINT,
  cpu_request_millicores BIGINT,
  memory_request_bytes BIGINT,
  cpu_limit_millicores BIGINT,
  memory_limit_bytes BIGINT
);
SELECT create_hypertable('pod_metrics','time', if_not_exists => TRUE);

-- Recreate node_metrics table and hypertable
CREATE TABLE node_metrics (
  time timestamptz NOT NULL,
  tenant_id BIGINT NOT NULL,
  cluster_name TEXT,
  node_name TEXT,
  instance_type TEXT,
  cpu_capacity BIGINT,
  memory_capacity BIGINT,
  hourly_cost_usd NUMERIC(10,6)
);
SELECT create_hypertable('node_metrics','time', if_not_exists => TRUE);

-- Verify tables are recreated
SELECT 
    schemaname,
    tablename,
    hypertable_name
FROM timescaledb_information.hypertables
WHERE tablename IN ('pod_metrics', 'node_metrics');

