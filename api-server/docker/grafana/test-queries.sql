-- Test Queries for Grafana Troubleshooting
-- Run these in order to diagnose issues

-- ==========================================
-- Test 1: Basic Connectivity
-- ==========================================
-- Should return: 1 row with value 1
SELECT 1 as test;

-- ==========================================
-- Test 2: Check if Table Exists
-- ==========================================
-- Should return: Count of rows (may be 0)
SELECT COUNT(*) as row_count FROM pod_metrics;

-- ==========================================
-- Test 3: Check Data for Tenant
-- ==========================================
-- Should return: Row count for tenant_id = 1
SELECT COUNT(*) as row_count 
FROM pod_metrics 
WHERE tenant_id = 1;

-- ==========================================
-- Test 4: Check Time Range of Data
-- ==========================================
-- Shows when your data starts and ends
SELECT 
  MIN(time) as earliest_data,
  MAX(time) as latest_data,
  COUNT(*) as total_rows,
  NOW() as current_time
FROM pod_metrics
WHERE tenant_id = 1;

-- ==========================================
-- Test 5: Simple Data Query (No Aggregation)
-- ==========================================
-- Returns raw data - use this if time_bucket doesn't work
SELECT 
  time,
  namespace,
  pod_name,
  cpu_millicores,
  cpu_request_millicores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= NOW() - INTERVAL '7 days'
ORDER BY time DESC
LIMIT 10;

-- ==========================================
-- Test 6: Check TimescaleDB Extension
-- ==========================================
-- Should return: timescaledb extension info
SELECT * FROM pg_extension WHERE extname = 'timescaledb';

-- ==========================================
-- Test 7: Test time_bucket Function
-- ==========================================
-- Should return: A timestamp (today's date)
SELECT time_bucket('1 day', NOW()) as test_bucket;

-- ==========================================
-- Test 8: Simple Aggregation (No time_bucket)
-- ==========================================
-- Use this if time_bucket doesn't work
SELECT 
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  COUNT(*) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= NOW() - INTERVAL '7 days'
GROUP BY namespace
ORDER BY cpu_cores DESC
LIMIT 10;

-- ==========================================
-- Test 9: Time Series Query (Simplified)
-- ==========================================
-- Use "Time series" format in Grafana
-- Returns one series
SELECT 
  time,
  cpu_millicores / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= NOW() - INTERVAL '7 days'
ORDER BY time
LIMIT 100;

-- ==========================================
-- Test 10: Time Series with Aggregation
-- ==========================================
-- Use "Time series" format in Grafana
-- This is the working version of the cost trends query
SELECT 
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= NOW() - INTERVAL '30 days'
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time;

