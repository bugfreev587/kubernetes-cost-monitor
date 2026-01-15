# Fixed Grafana Queries

## Issue: Syntax Error with Variables

Grafana variables like `$tenant_id` must be defined in the dashboard before use. If you get `syntax error at or near "$"`, the variable isn't defined.

## Quick Fix: Use Literal Value

Replace `$tenant_id` with your actual tenant ID (usually `1`):

```sql
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY namespace
ORDER BY cpu_cores DESC
LIMIT 50
```

## Proper Fix: Define Dashboard Variable

### Step 1: Create Tenant ID Variable

1. In Grafana, go to your dashboard
2. Click **Dashboard settings** (gear icon) → **Variables** tab
3. Click **Add variable**
4. Configure:
   - **Name:** `tenant_id`
   - **Type:** `Query`
   - **Data source:** `TimescaleDB Production`
   - **Query:** 
     ```sql
     SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id
     ```
   - **Refresh:** `On Dashboard Load`
   - Click **Apply**

### Step 2: Use Variable in Query

Now your original query will work:

```sql
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY namespace
ORDER BY cpu_cores DESC
LIMIT 50
```

## Alternative: Find Your Tenant ID First

If you don't know your tenant ID, run this query first:

```sql
SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id;
```

Then use that value in your queries.

## Time Macros

The `$__timeFrom()` and `$__timeTo()` macros are built-in Grafana macros that work automatically - you don't need to define them. They use the time range selector at the top of the dashboard.

## Panel Format Settings

**Important:** The panel format must match the query type:

- **Table queries** (no time column) → Set Format to **"Table"**
- **Time series queries** (has time column) → Set Format to **"Time series"**

### How to Change Panel Format

1. In the query editor, look for **"Format as"** dropdown (usually at the bottom)
2. Select:
   - **"Table"** for queries without time column
   - **"Time series"** for queries with time column

## All Fixed Queries

### Cost by Namespace (Table Format)

**Panel Format:** Set to **"Table"**

```sql
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY namespace
ORDER BY cpu_cores DESC
LIMIT 50
```

**Note:** This query aggregates across all time, so it's a table, not a time series.

### Cost by Namespace (Time Series)

```sql
SELECT
  time_bucket('1 day', time) as time,
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time), namespace
ORDER BY time, namespace
```

### Cost by Cluster

```sql
SELECT
  cluster_name,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as memory_gb,
  COUNT(DISTINCT namespace) as namespace_count,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY cluster_name
ORDER BY cpu_cores DESC
```

### Daily Cost Trends

```sql
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as total_cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as total_memory_gb,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time
```

### Resource Utilization

```sql
SELECT
  time_bucket('1 hour', time) as time,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as cpu_utilization_percent,
  AVG(memory_bytes) / NULLIF(AVG(memory_request_bytes), 0) * 100 as memory_utilization_percent
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY time_bucket('1 hour', time)
ORDER BY time
```

