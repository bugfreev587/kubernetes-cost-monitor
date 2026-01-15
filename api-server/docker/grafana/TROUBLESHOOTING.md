# Grafana Troubleshooting Guide

## Issue: "Run query" Button Does Nothing / No Response

### Step 1: Verify Data Source Connection

1. **Check Data Source Status:**
   - Go to **Configuration → Data Sources**
   - Click on **TimescaleDB**
   - Click **Test** button
   - Should show: "Data source is working"

2. **If Test Fails, Check:**
   - TimescaleDB is running: `docker ps | grep timescaledb`
   - Network connectivity: Grafana and TimescaleDB on same Docker network
   - Credentials are correct

### Step 2: Test Simple Query

Try a very simple query first to verify connectivity:

```sql
SELECT 1 as test
```

If this doesn't work, the data source connection is the issue.

### Step 3: Check Browser Console

1. Open browser Developer Tools (F12)
2. Go to **Console** tab
3. Click "Run query" again
4. Look for JavaScript errors or network errors

Common errors:
- `NetworkError` - Connection issue
- `401 Unauthorized` - Authentication issue
- `500 Internal Server Error` - Database query error

### Step 4: Verify Data Exists

Connect to TimescaleDB directly to verify data:

```bash
# From host machine
docker exec -it k8s_cost_timescaledb psql -U ts_user -d timeseries

# Or using psql from host
psql -h localhost -p 5433 -U ts_user -d timeseries
```

Then run:
```sql
-- Check if data exists
SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;

-- Check time range of data
SELECT MIN(time) as earliest, MAX(time) as latest FROM pod_metrics WHERE tenant_id = 1;

-- Check if time_bucket function exists (TimescaleDB extension)
SELECT * FROM pg_extension WHERE extname = 'timescaledb';
```

### Step 5: Fix Common Query Issues

#### Issue: Query uses `time_bucket` but function not found

**Solution:** Ensure TimescaleDB extension is enabled:
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
```

#### Issue: Query returns no data

**Possible causes:**
1. **Time range mismatch:**
   - Check Grafana time range (top right)
   - Ensure it overlaps with your data time range
   - Try: "Last 30 days" or custom range

2. **Tenant ID mismatch:**
   - Verify tenant_id in query matches your data
   - Check: `SELECT DISTINCT tenant_id FROM pod_metrics;`

3. **No data in database:**
   - Verify cost-agent is sending data
   - Check: `SELECT COUNT(*) FROM pod_metrics;`

#### Issue: Query syntax error

**Common mistakes:**
- Missing `time_bucket()` function (need TimescaleDB extension)
- Wrong table/column names
- Missing WHERE clause filters

**Test query (simplified):**
```sql
SELECT 
  time,
  cpu_millicores
FROM pod_metrics
WHERE tenant_id = 1
LIMIT 10
```

### Step 6: Check Grafana Logs

View Grafana container logs:
```bash
docker logs k8s_cost_grafana
```

Look for:
- Connection errors
- Query errors
- Authentication failures

### Step 7: Verify Network Configuration

Ensure Grafana can reach TimescaleDB:

```bash
# From Grafana container
docker exec -it k8s_cost_grafana sh
wget -O- http://timescaledb:5432  # Should fail (not HTTP), but confirms network
```

Or test from Grafana container:
```bash
docker exec -it k8s_cost_grafana sh
apk add postgresql-client
psql -h timescaledb -U ts_user -d timeseries -c "SELECT 1;"
```

### Step 8: Data Source Configuration Fix

If data source test fails, manually reconfigure:

1. **Configuration → Data Sources → TimescaleDB → Edit**

2. **Check these settings:**
   - **Host:** `timescaledb:5432` (from Grafana container) or `host.docker.internal:5433` (from host)
   - **Database:** `timeseries`
   - **User:** `ts_user`
   - **Password:** `ts_pass`
   - **SSL Mode:** `disable` (for local dev)
   - **TimescaleDB:** ✅ Enabled
   - **Version:** PostgreSQL 16

3. **Advanced Settings:**
   - **Max open:** 100
   - **Max idle:** 100
   - **Max lifetime:** 14400

4. **Click "Save & Test"**

### Step 9: Alternative Query Format

If raw SQL doesn't work, try the query builder:

1. Click **"Builder"** tab (instead of "Code")
2. Select table: `pod_metrics`
3. Add filters manually
4. Switch back to "Code" to see generated SQL

### Step 10: Check Query Format Setting

In the query editor:
- **Format:** Should be set to **"Time series"** for time-based queries
- **Format:** Should be set to **"Table"** for table queries

For the cost trends query, use **"Time series"** format.

## Quick Diagnostic Queries

### Test 1: Basic Connectivity
```sql
SELECT 1 as test
```
**Expected:** Returns a single row with value 1

### Test 2: Table Exists
```sql
SELECT COUNT(*) FROM pod_metrics
```
**Expected:** Returns count of rows (may be 0 if no data)

### Test 3: Data Time Range
```sql
SELECT 
  MIN(time) as earliest,
  MAX(time) as latest,
  COUNT(*) as total_rows
FROM pod_metrics
WHERE tenant_id = 1
```
**Expected:** Shows time range and row count

### Test 4: TimescaleDB Functions
```sql
SELECT time_bucket('1 day', NOW()) as test_bucket
```
**Expected:** Returns a timestamp (today's date)

### Test 5: Simple Time Series
```sql
SELECT 
  time,
  cpu_millicores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= NOW() - INTERVAL '7 days'
ORDER BY time DESC
LIMIT 10
```
**Expected:** Returns 10 most recent rows

## Common Solutions

### Solution 1: Fix Time Range
If query returns no data, adjust Grafana time range:
- Click time range selector (top right)
- Select "Last 30 days" or custom range
- Ensure range includes your data

### Solution 2: Fix Tenant ID
If using wrong tenant_id:
```sql
-- Find available tenant IDs
SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id;

-- Update query to use correct tenant_id
WHERE tenant_id = 1  -- Change to your tenant_id
```

### Solution 3: Simplify Query
Start with simplest query, then add complexity:
```sql
-- Step 1: Basic select
SELECT * FROM pod_metrics LIMIT 10

-- Step 2: Add time filter
SELECT * FROM pod_metrics 
WHERE time >= NOW() - INTERVAL '7 days'
LIMIT 10

-- Step 3: Add aggregation
SELECT 
  time_bucket('1 day', time) as time,
  COUNT(*) as count
FROM pod_metrics
WHERE time >= NOW() - INTERVAL '7 days'
GROUP BY time_bucket('1 day', time)
```

### Solution 4: Check Data Source UID
In dashboard JSON or query, ensure data source UID matches:
- Check: Configuration → Data Sources → TimescaleDB → UID
- Use this UID in queries: `"datasource": {"uid": "your-uid-here"}`

## Still Not Working?

1. **Restart Grafana:**
   ```bash
   docker restart k8s_cost_grafana
   ```

2. **Check TimescaleDB:**
   ```bash
   docker logs k8s_cost_timescaledb | tail -20
   ```

3. **Verify Docker Network:**
   ```bash
   docker network inspect docker_default | grep -A 5 timescaledb
   docker network inspect docker_default | grep -A 5 grafana
   ```

4. **Recreate Grafana Container:**
   ```bash
   docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml down grafana
   docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana
   ```

## Getting Help

If still stuck, provide:
1. Browser console errors (F12 → Console)
2. Grafana logs: `docker logs k8s_cost_grafana`
3. Data source test result
4. Query you're trying to run
5. Sample data check: `SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;`

