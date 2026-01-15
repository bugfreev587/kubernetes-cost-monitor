-- Example SQL Queries for Grafana Dashboards
-- These queries can be used directly in Grafana panels

-- ==========================================
-- 1. Cost by Namespace (Time Series)
-- ==========================================
-- Use this for a time series panel showing cost trends by namespace
SELECT
  time_bucket('1 day', time) as time,
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time), namespace
ORDER BY time, namespace;

-- ==========================================
-- 2. Total Daily Cost Trend
-- ==========================================
-- Shows overall cost trend over time
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as total_cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as total_memory_gb,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time;

-- ==========================================
-- 3. Cost by Cluster (Table)
-- ==========================================
-- Use this for a table panel showing cluster breakdown
SELECT
  cluster_name,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb,
  COUNT(DISTINCT namespace) as namespace_count,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY cluster_name
ORDER BY cpu_cores DESC;

-- ==========================================
-- 4. Resource Utilization Percentage
-- ==========================================
-- Shows CPU and memory utilization vs requests
SELECT
  time_bucket('1 hour', time) as time,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as cpu_utilization_percent,
  AVG(memory_bytes) / NULLIF(AVG(memory_request_bytes), 0) * 100 as memory_utilization_percent
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY time_bucket('1 hour', time)
ORDER BY time;

-- ==========================================
-- 5. Top Underutilized Pods
-- ==========================================
-- Identifies pods with low utilization (right-sizing candidates)
SELECT
  namespace,
  pod_name,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as cpu_util_pct,
  AVG(memory_bytes) / NULLIF(AVG(memory_request_bytes), 0) * 100 as mem_util_pct,
  AVG(cpu_request_millicores) as avg_cpu_request,
  AVG(cpu_millicores) as avg_cpu_usage,
  AVG(cpu_request_millicores) - AVG(cpu_millicores) as cpu_waste
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY namespace, pod_name
HAVING AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) < 0.5
ORDER BY cpu_util_pct ASC
LIMIT 50;

-- ==========================================
-- 6. Node Costs Over Time
-- ==========================================
-- Shows infrastructure costs from node_metrics
SELECT
  time_bucket('1 day', time) as time,
  cluster_name,
  AVG(hourly_cost_usd) * 24 as daily_cost_usd,
  COUNT(DISTINCT node_name) as node_count
FROM node_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
GROUP BY time_bucket('1 day', time), cluster_name
ORDER BY time, cluster_name;

-- ==========================================
-- 7. Namespace Resource Summary
-- ==========================================
-- Current resource usage by namespace
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as total_cpu_cores_requested,
  AVG(cpu_millicores) / 1000.0 as avg_cpu_cores_used,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as total_memory_gb_requested,
  AVG(memory_bytes) / 1024.0 / 1024.0 / 1024.0 as avg_memory_gb_used,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY namespace
ORDER BY total_cpu_cores_requested DESC;

-- ==========================================
-- 8. Weekly Cost Comparison
-- ==========================================
-- Compare costs week-over-week
SELECT
  time_bucket('1 week', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 week', time)
ORDER BY time;

-- ==========================================
-- 9. Pod Utilization Heatmap
-- ==========================================
-- Shows utilization patterns across pods
SELECT
  time_bucket('1 hour', time) as time,
  namespace,
  pod_name,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as utilization_pct
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY time_bucket('1 hour', time), namespace, pod_name
ORDER BY time, utilization_pct DESC
LIMIT 100;

-- ==========================================
-- 10. Cost Efficiency Metrics
-- ==========================================
-- Shows cost efficiency (usage vs requests)
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_requested_cores,
  AVG(cpu_millicores) / 1000.0 as cpu_used_cores,
  CASE
    WHEN SUM(cpu_request_millicores) > 0
    THEN (AVG(cpu_millicores) / SUM(cpu_request_millicores)) * 100
    ELSE 0
  END as cpu_efficiency_pct
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY namespace
ORDER BY cpu_efficiency_pct ASC;

