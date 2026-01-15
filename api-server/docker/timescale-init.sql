-- timescale-init.sql
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS pod_metrics (
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

CREATE TABLE IF NOT EXISTS node_metrics (
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

-- ============================
-- Test Data: pod_metrics
-- ============================
INSERT INTO pod_metrics (
  time, tenant_id, cluster_name, namespace, pod_name, node_name,
  cpu_millicores, memory_bytes, cpu_request_millicores, memory_request_bytes, cpu_limit_millicores, memory_limit_bytes
)
VALUES
  -- Tenant 1: Acme
  (now() - interval '5 minutes', 1, 'cluster-a', 'default', 'api-server-1', 'node-a1', 120, 150000000, 500, 500000000, 1000, 1000000000),
  (now() - interval '3 minutes', 1, 'cluster-a', 'default', 'api-server-1', 'node-a1', 80, 120000000, 500, 500000000, 1000, 1000000000),
  (now() - interval '1 minutes', 1, 'cluster-a', 'ml', 'training-pod', 'node-a2', 950, 1800000000, 2000, 4096000000, 4000, 8192000000),

  -- Tenant 2: Globex
  (now() - interval '2 minutes', 2, 'cluster-b', 'analytics', 'spark-worker-3', 'node-b3', 1300, 4000000000, 2000, 8192000000, 4000, 16384000000),

  -- Tenant 3: Skynet
  (now() - interval '4 minutes', 3, 'skynet-cluster', 'war', 't800-control', 'node-s1', 600, 1200000000, 1000, 2048000000, 2000, 4096000000);

-- ============================
-- Test Data: node_metrics
-- ============================
INSERT INTO node_metrics (
  time, tenant_id, cluster_name, node_name,
  instance_type, cpu_capacity, memory_capacity, hourly_cost_usd
)
VALUES
  (now() - interval '5 minutes', 1, 'cluster-a', 'node-a1', 'm5.large', 2000, 8000000000, 0.096),
  (now() - interval '5 minutes', 1, 'cluster-a', 'node-a2', 'm5.xlarge', 4000, 16000000000, 0.192),

  (now() - interval '5 minutes', 2, 'cluster-b', 'node-b3', 'c5.2xlarge', 8000, 16000000000, 0.340),

  (now() - interval '5 minutes', 3, 'skynet-cluster', 'node-s1', 't800-prototype', 16000, 64000000000, 1.500);