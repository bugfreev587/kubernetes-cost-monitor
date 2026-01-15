# Grafana Panel Format Guide

## Common Error: "Data is missing a time field"

This error occurs when:
- Your query doesn't return a `time` column
- But the panel format is set to **"Time series"**

## Solution: Match Panel Format to Query Type

### For Table Queries (No Time Column)

**Query Example:**
```sql
SELECT
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
GROUP BY namespace
ORDER BY cpu_cores DESC
```

**Panel Settings:**
1. **Format as:** Select **"Table"** (not "Time series")
2. **Visualization:** Choose "Table" panel type

### For Time Series Queries (Has Time Column)

**Query Example:**
```sql
SELECT
  time_bucket('1 day', time) as time,
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
GROUP BY time_bucket('1 day', time), namespace
ORDER BY time
```

**Panel Settings:**
1. **Format as:** Select **"Time series"**
2. **Visualization:** Choose "Time series" panel type
3. **Time column:** Must be named `time` (or use alias `as time`)

## How to Change Panel Format

### Step-by-Step Instructions

1. **Open your dashboard panel:**
   - Click on the panel you want to edit
   - Click **"Edit"** button (pencil icon) at the top

2. **Find the Format setting** (two locations):

   **Location 1: Query Editor (Bottom)**
   - Scroll down in the **left panel** (where you write SQL)
   - Below your SQL query, find **"Format as"** dropdown
   - Change from `Time series` to `Table`

   **Location 2: Right Sidebar (Top)**
   - Look at the **right side** of the screen
   - At the top, find **"Visualization"** section
   - Click the dropdown and select **"Table"**

3. **Click "Run query"** to see the results

### Visual Guide

```
┌─────────────────────────────┐  ┌──────────────────────┐
│ Query Editor               │  │ Panel Options        │
├─────────────────────────────┤  │                      │
│ SELECT namespace, ...       │  │ Visualization:       │
│ FROM pod_metrics            │  │ [Table ▼]            │ ← Option 2
│ WHERE tenant_id = 1        │  │                      │
│ GROUP BY namespace          │  │                      │
│                             │  │                      │
│ Format as: [Table ▼]       │  │                      │ ← Option 1
└─────────────────────────────┘  └──────────────────────┘
```

### Quick Reference

- **Table queries** (no time column) → Set Format to **"Table"**
- **Time series queries** (has time column) → Set Format to **"Time series"**

See [`HOW_TO_SET_PANEL_FORMAT.md`](./HOW_TO_SET_PANEL_FORMAT.md) for detailed step-by-step guide with screenshots.

## Query Examples by Format

### Table Format Queries

**Cost by Namespace (Current Summary)**
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

**Cost by Cluster (Summary)**
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

### Time Series Format Queries

**Cost Trends Over Time**
```sql
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as total_cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as total_memory_gb
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time
```

**Cost by Namespace Over Time**
```sql
SELECT
  time_bucket('1 day', time) as time,
  namespace,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time), namespace
ORDER BY time, namespace
```

## Quick Reference

| Query Has Time Column? | Panel Format | Visualization Type |
|------------------------|--------------|-------------------|
| ❌ No                  | **Table**    | Table             |
| ✅ Yes                 | **Time series** | Time series    |
| ✅ Yes (multiple series) | **Time series** | Time series (with legend) |

## Troubleshooting

### Error: "Data is missing a time field"

**Fix:** Change panel format from "Time series" to "Table"

### Error: "No data"

**Check:**
1. Time range selector (top right) - is it set correctly?
2. Does data exist for that time range?
3. Are filters correct (tenant_id, pod_name, etc.)?

### Error: "Query returned no data"

**Test with simple query:**
```sql
SELECT COUNT(*) as total
FROM pod_metrics
WHERE tenant_id = 1
```

If this works, your filters might be too restrictive.

