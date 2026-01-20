-- Migration: Add Priority 1 Pod Metric Enhancements
-- Adds: labels, phase, qos_class, and containers to pod_metrics table
-- Date: 2026-01-17

-- ============================
-- Add New Columns to pod_metrics
-- ============================

-- Pod labels (JSONB for flexible key-value storage)
ALTER TABLE pod_metrics ADD COLUMN IF NOT EXISTS labels JSONB;

-- Pod phase (Running, Pending, Succeeded, Failed, Unknown)
ALTER TABLE pod_metrics ADD COLUMN IF NOT EXISTS phase TEXT;

-- QoS class (Guaranteed, Burstable, BestEffort)
ALTER TABLE pod_metrics ADD COLUMN IF NOT EXISTS qos_class TEXT;

-- Container-level metrics (JSONB array)
ALTER TABLE pod_metrics ADD COLUMN IF NOT EXISTS containers JSONB;

-- ============================
-- Create Indexes for Performance
-- ============================

-- GIN index for label queries (enables fast JSON queries)
CREATE INDEX IF NOT EXISTS idx_pod_metrics_labels
  ON pod_metrics USING GIN (labels);

-- B-tree index for phase queries
CREATE INDEX IF NOT EXISTS idx_pod_metrics_phase
  ON pod_metrics (tenant_id, phase, time DESC);

-- B-tree index for QoS class queries
CREATE INDEX IF NOT EXISTS idx_pod_metrics_qos_class
  ON pod_metrics (tenant_id, qos_class, time DESC);

-- Composite index for cost allocation by label
CREATE INDEX IF NOT EXISTS idx_pod_metrics_tenant_time_labels
  ON pod_metrics (tenant_id, time DESC) INCLUDE (labels)
  WHERE labels IS NOT NULL;

-- ============================
-- Add Comments for Documentation
-- ============================

COMMENT ON COLUMN pod_metrics.labels IS
  'Pod labels as JSON object for cost allocation (e.g., {"app": "api", "team": "platform", "environment": "prod"})';

COMMENT ON COLUMN pod_metrics.phase IS
  'Pod lifecycle phase: Running, Pending, Succeeded, Failed, Unknown. Only Running pods incur costs.';

COMMENT ON COLUMN pod_metrics.qos_class IS
  'Quality of Service class: Guaranteed (requests = limits), Burstable (requests < limits), BestEffort (no requests/limits)';

COMMENT ON COLUMN pod_metrics.containers IS
  'Array of container-level metrics in JSON format. Each element contains: container_name, cpu_millicores, memory_bytes, cpu_request_millicores, memory_request_bytes, cpu_limit_millicores, memory_limit_bytes';

-- ============================
-- Example Queries Using New Fields
-- ============================

/*
-- Query 1: Cost by team (using labels)
SELECT
  labels->>'team' AS team,
  SUM(cpu_millicores) / 1000.0 AS total_cpu_cores,
  SUM(memory_bytes) / 1024 / 1024 / 1024 AS total_memory_gb
FROM pod_metrics
WHERE
  time > NOW() - INTERVAL '24 hours'
  AND tenant_id = 1
  AND labels ? 'team'  -- Only rows with 'team' label
GROUP BY labels->>'team'
ORDER BY total_cpu_cores DESC;

-- Query 2: Only count running pods (accurate billing)
SELECT COUNT(*), AVG(cpu_millicores)
FROM pod_metrics
WHERE
  time > NOW() - INTERVAL '1 hour'
  AND tenant_id = 1
  AND phase = 'Running';

-- Query 3: Identify over-provisioned pods (Burstable/BestEffort)
SELECT
  namespace,
  pod_name,
  qos_class,
  cpu_request_millicores,
  cpu_millicores,
  (cpu_request_millicores - cpu_millicores) AS wasted_cpu_millicores
FROM pod_metrics
WHERE
  time > NOW() - INTERVAL '24 hours'
  AND tenant_id = 1
  AND qos_class IN ('Burstable', 'BestEffort')
  AND cpu_millicores < cpu_request_millicores * 0.5
ORDER BY wasted_cpu_millicores DESC
LIMIT 20;

-- Query 4: Cost allocation by environment
SELECT
  labels->>'environment' AS environment,
  COUNT(DISTINCT pod_name) AS pod_count,
  SUM(cpu_millicores) AS total_cpu,
  SUM(memory_bytes) / 1024 / 1024 / 1024 AS total_memory_gb
FROM pod_metrics
WHERE
  time > NOW() - INTERVAL '7 days'
  AND tenant_id = 1
  AND labels ? 'environment'
GROUP BY labels->>'environment';

-- Query 5: Container-level breakdown for multi-container pods
SELECT
  pod_name,
  jsonb_array_elements(containers)->>'container_name' AS container_name,
  (jsonb_array_elements(containers)->>'cpu_millicores')::bigint AS cpu_usage,
  (jsonb_array_elements(containers)->>'memory_bytes')::bigint / 1024 / 1024 AS memory_mb
FROM pod_metrics
WHERE
  time > NOW() - INTERVAL '1 hour'
  AND tenant_id = 1
  AND jsonb_array_length(containers) > 1  -- Multi-container pods only
LIMIT 100;

-- Query 6: Sidecar cost attribution (e.g., Istio proxy)
SELECT
  pod_name,
  container_name,
  SUM((container_metrics->>'cpu_millicores')::bigint) AS total_cpu,
  SUM((container_metrics->>'memory_bytes')::bigint) / 1024 / 1024 / 1024 AS total_memory_gb
FROM pod_metrics,
  jsonb_array_elements(containers) AS container_metrics(container_metrics)
WHERE
  time > NOW() - INTERVAL '7 days'
  AND tenant_id = 1
  AND container_metrics->>'container_name' LIKE '%istio%'
GROUP BY pod_name, container_name
ORDER BY total_cpu DESC;
*/

-- ============================
-- Rollback Script (if needed)
-- ============================

/*
-- To rollback this migration:

DROP INDEX IF EXISTS idx_pod_metrics_labels;
DROP INDEX IF EXISTS idx_pod_metrics_phase;
DROP INDEX IF EXISTS idx_pod_metrics_qos_class;
DROP INDEX IF EXISTS idx_pod_metrics_tenant_time_labels;

ALTER TABLE pod_metrics DROP COLUMN IF EXISTS labels;
ALTER TABLE pod_metrics DROP COLUMN IF EXISTS phase;
ALTER TABLE pod_metrics DROP COLUMN IF EXISTS qos_class;
ALTER TABLE pod_metrics DROP COLUMN IF EXISTS containers;
*/

-- ============================
-- Verify Migration
-- ============================

-- Check columns exist
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'pod_metrics'
  AND column_name IN ('labels', 'phase', 'qos_class', 'containers')
ORDER BY column_name;

-- Check indexes exist
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'pod_metrics'
  AND indexname LIKE 'idx_pod_metrics_%'
ORDER BY indexname;
