# SQL Query Analysis: Why "Time Column Missing" Error?

## The Query

```sql
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
```

## Problem Analysis

### What the Query Does

1. **Filters by time range:** Uses `time >= $__timeFrom()` and `time <= $__timeTo()` in WHERE clause
2. **Aggregates data:** Groups by `cluster_name` and calculates totals
3. **Returns summary:** One row per cluster with aggregated metrics

### Why It Fails with "Time series" Format

**The Issue:**
- ✅ Query **uses** `time` column (in WHERE clause for filtering)
- ❌ Query **does NOT SELECT** `time` column (not in SELECT clause)
- ❌ Query **does NOT GROUP BY** `time` (only groups by `cluster_name`)

**Grafana's "Time series" Format Requirements:**
- **MUST** have a `time` column in the SELECT clause
- **MUST** group by `time` (usually with `time_bucket()`)
- Returns multiple rows over time (one per time bucket)

**This Query's Output:**
- Returns aggregated summary (one row per cluster)
- No time dimension in results
- Perfect for **Table** format, NOT time series

## Solutions

### Solution 1: Change Format to "Table" (Recommended)

**Why:** This query is designed to show a summary table, not a time series.

**Steps:**
1. In Grafana panel editor, find **"Format"** dropdown
2. Change from **"Time series"** to **"Table"**
3. Click **"Run query"**

**Result:** You'll see a table with columns:
- cluster_name
- cpu_cores
- memory_gb
- namespace_count
- pod_count

### Solution 2: Convert to Time Series Query

**If you want to see cost trends over time by cluster:**

```sql
SELECT
  time_bucket('1 day', time) as time,  -- ← Add time column
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
GROUP BY time_bucket('1 day', time), cluster_name  -- ← Group by time too
ORDER BY time, cpu_cores DESC;
```

**Changes:**
- ✅ Added `time_bucket('1 day', time) as time` to SELECT
- ✅ Added `time_bucket('1 day', time)` to GROUP BY
- ✅ Changed ORDER BY to include `time`

**Result:** Time series showing cost trends per cluster over time

## Key Differences

| Aspect | Current Query (Table) | Time Series Query |
|--------|----------------------|-------------------|
| **SELECT** | cluster_name, metrics | time, cluster_name, metrics |
| **GROUP BY** | cluster_name | time_bucket(...), cluster_name |
| **Output** | One row per cluster | Multiple rows (one per time bucket per cluster) |
| **Format** | Table | Time series |
| **Use Case** | Current summary | Trends over time |

## Visual Comparison

### Table Format Output:
```
cluster_name    | cpu_cores | memory_gb | namespace_count | pod_count
----------------|-----------|-----------|----------------|----------
cluster-1       | 45.2      | 128.5     | 5              | 23
cluster-2       | 32.1      | 96.3      | 3              | 15
```

### Time Series Format Output:
```
time            | cluster_name | cpu_cores | memory_gb
----------------|--------------|-----------|----------
2026-01-01      | cluster-1    | 45.2      | 128.5
2026-01-01      | cluster-2    | 32.1      | 96.3
2026-01-02      | cluster-1    | 46.8      | 130.2
2026-01-02      | cluster-2    | 33.5      | 98.1
...
```

## Summary

**Root Cause:**
- Query aggregates by `cluster_name` only (no time dimension)
- Panel format is set to "Time series" (requires time column)
- Mismatch between query structure and format setting

**Fix:**
- **Option 1:** Change format to "Table" (for summary view)
- **Option 2:** Add `time_bucket()` to query (for time series view)

**Recommendation:** Use **Option 1** (Table format) since this query is clearly designed as a summary table, not a time series.

