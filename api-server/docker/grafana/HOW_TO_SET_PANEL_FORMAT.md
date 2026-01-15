# How to Set Panel Format in Grafana

## Step-by-Step Guide

### Step 1: Open Your Dashboard Panel

1. Go to your Grafana dashboard
2. Click on the panel you want to edit (or click **"Add panel"** to create a new one)
3. Click **"Edit"** button (pencil icon) at the top of the panel

### Step 2: Find the Format Setting

There are **two places** where you can set the format:

---

## Method 1: In the Query Editor (Recommended)

1. **Scroll down** in the query editor panel (left side)
2. Look for a section called **"Format as"** or **"Format"**
3. You'll see a **dropdown** that says:
   - `Time series` (default)
   - `Table`
   - `Logs`
   - etc.

4. **Change it to "Table"** for your namespace query

**Visual Guide:**
```
┌─────────────────────────────────────┐
│ Query Editor                        │
├─────────────────────────────────────┤
│ SELECT namespace, ...               │
│ FROM pod_metrics                    │
│ WHERE tenant_id = 1                 │
│ ...                                 │
│                                     │
│ ┌─────────────────────────────┐   │
│ │ Format as: [Time series ▼] │   │ ← Change this!
│ └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

---

## Method 2: Change Visualization Type (Right Sidebar)

1. After clicking **"Edit"** on your panel
2. Look at the **right sidebar** (panel options)
3. At the very top, you'll see **"Visualization"** section
4. Click the dropdown that shows:
   - `Time series`
   - `Table`
   - `Bar chart`
   - `Stat`
   - etc.

5. **Select "Table"** for aggregated queries without time

**Visual Guide:**
```
┌─────────────────┐  ┌──────────────────────┐
│ Query Editor    │  │ Panel Options        │
│                 │  │                      │
│ SELECT ...      │  │ Visualization:       │
│                 │  │ [Time series ▼]      │ ← Change this!
│                 │  │                      │
│                 │  │ Options              │
│                 │  │ ...                  │
└─────────────────┘  └──────────────────────┘
```

---

## Detailed Steps with Screenshots Description

### Option A: Format Dropdown in Query Editor

**Location:** Bottom of the query editor panel

1. In Grafana, edit your panel
2. Scroll down in the **left panel** (where you write SQL)
3. Below your SQL query, you'll see:
   ```
   Format as: [Time series ▼]
   ```
4. Click the dropdown and select **"Table"**

### Option B: Visualization Type in Right Panel

**Location:** Top of the right sidebar

1. Edit your panel
2. Look at the **right side** of the screen
3. At the top, find **"Visualization"** section
4. Click the visualization dropdown
5. Select **"Table"**

---

## Quick Reference: What Format for What Query?

### Use "Table" Format When:

- Query aggregates data (SUM, COUNT, AVG)
- Query groups by non-time columns (namespace, cluster_name, etc.)
- Query doesn't have `time_bucket()` or time column
- You want to see a summary/table view

**Example:**
```sql
SELECT namespace, SUM(cpu_cores) 
FROM pod_metrics 
GROUP BY namespace
```

### Use "Time series" Format When:

- Query includes `time_bucket()` function
- Query has a `time` column in SELECT
- You want to see trends over time
- Query groups by time

**Example:**
```sql
SELECT time_bucket('1 day', time) as time, SUM(cpu_cores)
FROM pod_metrics 
GROUP BY time_bucket('1 day', time)
```

---

## Troubleshooting: Can't Find the Format Setting?

### If you don't see "Format as" dropdown:

1. **Make sure you're in Edit mode:**
   - Click the panel title → Click **"Edit"** (pencil icon)

2. **Check the query editor:**
   - Scroll all the way down in the left panel
   - Look below the SQL query box

3. **Try the Visualization dropdown:**
   - Right sidebar → Top section → "Visualization" dropdown

4. **Check Grafana version:**
   - Older versions might have it in different locations
   - Try: Right sidebar → "Transform" tab → "Format" option

### If Format dropdown is grayed out:

- Make sure your query runs successfully first
- Click "Run query" button
- Then try changing the format

---

## Visual Example: Complete Panel Setup

```
┌─────────────────────────────────────────────────────────────┐
│ Dashboard: Cost Overview                    [Save] [Apply] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────┐  ┌──────────────────────┐ │
│  │ Query Editor             │  │ Panel Options         │ │
│  ├──────────────────────────┤  │                       │ │
│  │ SELECT                   │  │ Visualization:        │ │
│  │   namespace,             │  │ [Table ▼]            │ │ ← Set here
│  │   SUM(...) as cpu_cores  │  │                       │ │
│  │ FROM pod_metrics         │  │                       │ │
│  │ WHERE tenant_id = 1      │  │ Format as:            │ │
│  │ GROUP BY namespace       │  │ [Table ▼]            │ │ ← Or here
│  │                          │  │                       │ │
│  │ [Run query]              │  │ Options               │ │
│  │                          │  │ ...                   │ │
│  │ Format as: [Table ▼]    │  │                       │ │
│  └──────────────────────────┘  └──────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Your Specific Query Setup

For your namespace query:

1. **Query:**
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

2. **Format:** Set to **"Table"**

3. **Visualization:** Set to **"Table"**

4. **Result:** You'll see a table with columns: namespace, cpu_cores, memory_gb, pod_count

---

## Still Having Issues?

If you can't find these settings:

1. **Take a screenshot** of your Grafana panel editor
2. **Check Grafana version:** Help → About (bottom left)
3. **Try creating a new panel** and see if format options appear
4. **Check browser console** (F12) for any errors

The format setting should be visible in either:
- Bottom of query editor (left side)
- Top of visualization options (right side)

