-- Multi-Tenant Grafana Query Examples
-- These queries automatically filter data based on the authenticated user's tenant_id
-- The tenant context is set automatically via Grafana's OAuth integration

-- ============================
-- Setup Instructions
-- ============================

/*
1. In Grafana, configure TimescaleDB data source with these connection settings:
   - Host: your-timescaledb-host:5433
   - Database: timeseries
   - User: ts_user
   - SSL Mode: require (for production)

2. In the data source settings, add "Before Connect" initialization query:
   SET app.current_tenant_id = <tenant_id_from_oauth>;

   However, Grafana doesn't natively support setting variables from OAuth claims.

   WORKAROUND: Use Grafana's query variables or set via backend proxy.
*/

-- ============================
-- Option 1: Backend Proxy Approach (Recommended)
-- ============================

/*
Create a backend proxy endpoint in api-server that:
1. Validates OAuth token
2. Extracts tenant_id from JWT claims
3. Sets tenant context
4. Proxies query to TimescaleDB
5. Returns results

Example endpoint: POST /v1/grafana/query
*/

-- ============================
-- Option 2: Grafana Dashboard Variables
-- ============================

/*
In Grafana dashboard settings, create a variable:
- Name: tenant_id
- Type: Constant
- Value: Will be set by OAuth at runtime

Then use queries like:
*/

-- Set tenant context using dashboard variable
-- This would be called before each query in a scripted dashboard
SELECT set_tenant_context($tenant_id);

-- ============================
-- Dashboard Queries (Tenant-Isolated)
-- ============================

-- Query 1: CPU Usage Over Time (Auto-filtered by RLS)
-- Panel Type: Time series
SELECT
  time AS "time",
  cluster_name,
  namespace,
  pod_name,
  cpu_millicores / 1000.0 AS "CPU Cores"
FROM pod_metrics
WHERE
  $__timeFilter(time)
  AND cluster_name = '$cluster'
ORDER BY time;

-- Query 2: Memory Usage Over Time
-- Panel Type: Time series
SELECT
  time AS "time",
  cluster_name,
  namespace,
  pod_name,
  memory_bytes / 1024.0 / 1024.0 / 1024.0 AS "Memory (GB)"
FROM pod_metrics
WHERE
  $__timeFilter(time)
  AND cluster_name = '$cluster'
ORDER BY time;

-- Query 3: Resource Utilization vs Requests
-- Panel Type: Bar chart
SELECT
  namespace,
  AVG(cpu_millicores::FLOAT / NULLIF(cpu_request_millicores, 0) * 100) AS "CPU Utilization %",
  AVG(memory_bytes::FLOAT / NULLIF(memory_request_bytes, 0) * 100) AS "Memory Utilization %"
FROM pod_metrics
WHERE
  $__timeFilter(time)
  AND cluster_name = '$cluster'
GROUP BY namespace
ORDER BY namespace;

-- Query 4: Cost by Namespace
-- Panel Type: Pie chart
WITH pod_costs AS (
  SELECT
    p.namespace,
    p.pod_name,
    AVG(n.hourly_cost_usd) *
    (AVG(p.cpu_request_millicores)::FLOAT / AVG(n.cpu_capacity)) AS hourly_cost
  FROM pod_metrics p
  JOIN node_metrics n ON
    p.cluster_name = n.cluster_name
    AND p.node_name = n.node_name
    AND p.time = n.time
  WHERE
    $__timeFilter(p.time)
    AND p.cluster_name = '$cluster'
  GROUP BY p.namespace, p.pod_name
)
SELECT
  namespace,
  SUM(hourly_cost) * 24 * 30 AS "Monthly Cost (USD)"
FROM pod_costs
GROUP BY namespace
ORDER BY "Monthly Cost (USD)" DESC;

-- Query 5: Top Resource Consumers
-- Panel Type: Table
SELECT
  namespace,
  pod_name,
  AVG(cpu_millicores) AS "Avg CPU (m)",
  AVG(memory_bytes) / 1024 / 1024 / 1024 AS "Avg Memory (GB)",
  MAX(cpu_millicores) AS "Max CPU (m)",
  MAX(memory_bytes) / 1024 / 1024 / 1024 AS "Max Memory (GB)"
FROM pod_metrics
WHERE
  $__timeFilter(time)
  AND cluster_name = '$cluster'
GROUP BY namespace, pod_name
ORDER BY "Avg CPU (m)" DESC
LIMIT 10;

-- Query 6: Cluster Overview Stats
-- Panel Type: Stat panels
-- Total Pods
SELECT COUNT(DISTINCT pod_name) AS value
FROM pod_metrics
WHERE $__timeFilter(time)
AND cluster_name = '$cluster';

-- Total Namespaces
SELECT COUNT(DISTINCT namespace) AS value
FROM pod_metrics
WHERE $__timeFilter(time)
AND cluster_name = '$cluster';

-- Total Nodes
SELECT COUNT(DISTINCT node_name) AS value
FROM node_metrics
WHERE $__timeFilter(time)
AND cluster_name = '$cluster';

-- Total Monthly Cost
WITH pod_costs AS (
  SELECT
    AVG(n.hourly_cost_usd) *
    (AVG(p.cpu_request_millicores)::FLOAT / AVG(n.cpu_capacity)) AS hourly_cost
  FROM pod_metrics p
  JOIN node_metrics n ON
    p.cluster_name = n.cluster_name
    AND p.node_name = n.node_name
    AND p.time = n.time
  WHERE
    $__timeFilter(p.time)
    AND p.cluster_name = '$cluster'
  GROUP BY p.pod_name
)
SELECT SUM(hourly_cost) * 24 * 30 AS value
FROM pod_costs;

-- Query 7: Over-Provisioned Pods (Recommendations)
-- Panel Type: Table
SELECT
  namespace,
  pod_name,
  AVG(cpu_millicores) AS "Actual CPU (m)",
  AVG(cpu_request_millicores) AS "Requested CPU (m)",
  AVG(cpu_request_millicores) - AVG(cpu_millicores) AS "Over-provisioned CPU (m)",
  AVG(memory_bytes) / 1024 / 1024 / 1024 AS "Actual Memory (GB)",
  AVG(memory_request_bytes) / 1024 / 1024 / 1024 AS "Requested Memory (GB)",
  (AVG(memory_request_bytes) - AVG(memory_bytes)) / 1024 / 1024 / 1024 AS "Over-provisioned Memory (GB)"
FROM pod_metrics
WHERE
  $__timeFilter(time)
  AND cluster_name = '$cluster'
  AND cpu_request_millicores > cpu_millicores * 1.5  -- 50% over-provisioned
GROUP BY namespace, pod_name
ORDER BY "Over-provisioned CPU (m)" DESC
LIMIT 20;

-- Query 8: Cost Trend Over Time
-- Panel Type: Time series
SELECT
  time_bucket('1 hour', p.time) AS "time",
  SUM(
    n.hourly_cost_usd *
    (p.cpu_request_millicores::FLOAT / n.cpu_capacity)
  ) AS "Hourly Cost (USD)"
FROM pod_metrics p
JOIN node_metrics n ON
  p.cluster_name = n.cluster_name
  AND p.node_name = n.node_name
  AND p.time = n.time
WHERE
  $__timeFilter(p.time)
  AND p.cluster_name = '$cluster'
GROUP BY time_bucket('1 hour', p.time)
ORDER BY "time";

-- ============================
-- Admin Queries (Bypass RLS)
-- ============================

-- These queries require admin mode and should NOT be exposed in tenant dashboards
-- Use these for internal monitoring and analytics

-- Query 9: All Tenants Cost Summary (Admin Only)
-- Requires: SELECT enable_admin_mode();
SELECT
  tenant_id,
  cluster_name,
  COUNT(DISTINCT pod_name) AS pod_count,
  SUM(cpu_millicores) / 1000.0 AS total_cpu_cores,
  SUM(memory_bytes) / 1024 / 1024 / 1024 AS total_memory_gb
FROM pod_metrics
WHERE $__timeFilter(time)
GROUP BY tenant_id, cluster_name
ORDER BY tenant_id;

-- Query 10: Cross-Tenant Usage Comparison (Admin Only)
SELECT
  tenant_id,
  AVG(cpu_millicores) AS avg_cpu,
  AVG(memory_bytes) / 1024 / 1024 / 1024 AS avg_memory_gb,
  COUNT(*) AS metric_count
FROM pod_metrics
WHERE $__timeFilter(time)
GROUP BY tenant_id
ORDER BY tenant_id;

-- ============================
-- Testing RLS Policies
-- ============================

-- Test 1: Set tenant context and query (should only see tenant 1 data)
SELECT set_tenant_context(1);
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Should only return: 1

-- Test 2: Clear context and query (should return nothing)
SELECT clear_tenant_context();
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Should return: (empty)

-- Test 3: Enable admin mode (should see all data)
SELECT enable_admin_mode();
SELECT DISTINCT tenant_id FROM pod_metrics;  -- Should return: 1, 2, 3, etc.

-- Test 4: Disable admin mode
SELECT disable_admin_mode();

-- ============================
-- Grafana Dashboard Variables
-- ============================

/*
In Grafana, create these dashboard variables:

1. cluster (dropdown)
   - Query: SELECT DISTINCT cluster_name FROM pod_metrics ORDER BY cluster_name
   - Multi-value: Yes
   - Include All: Yes

2. namespace (dropdown)
   - Query: SELECT DISTINCT namespace FROM pod_metrics WHERE cluster_name = '$cluster' ORDER BY namespace
   - Multi-value: Yes
   - Include All: Yes

3. time_range (interval)
   - Auto: Yes
   - Options: 1m, 5m, 10m, 30m, 1h, 6h, 12h, 24h, 7d, 30d

Note: These queries will automatically be filtered by RLS based on tenant_id
*/
