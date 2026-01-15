# Grafana Quick Start Guide

Get Grafana up and running in 5 minutes to visualize your Kubernetes cost metrics.

## Step 1: Start Grafana

```bash
cd /Users/xiaoboyu/api-server/docker
docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana
```

## Step 2: Access Grafana

1. Open http://localhost:3000 in your browser
2. Login with:
   - Username: `admin`
   - Password: `admin`
3. Change password when prompted (recommended)

## Step 3: Verify Data Source

The TimescaleDB data source should be pre-configured. To verify:

1. Go to **Configuration → Data Sources**
2. Click on **TimescaleDB**
3. Click **Test** - should show "Data source is working"

If not configured:
- Click **Add data source** → **PostgreSQL**
- Host: `timescaledb:5432` (or `localhost:5433` from host)
- Database: `timeseries`
- User: `ts_user`
- Password: `ts_pass`
- Enable **TimescaleDB** option
- Click **Save & Test**

## Step 4: Create Your First Dashboard

### Troubleshooting: If "Run query" Does Nothing

**Before creating dashboards, test the connection:**

1. **Test Data Source:**
   - Go to Configuration → Data Sources → TimescaleDB
   - Click "Test" - should show "Data source is working"

2. **Run Test Query:**
   - Create new panel
   - Use this simple test query first:
   ```sql
   SELECT 1 as test
   ```
   - Click "Run query" - should return 1 row
   - If this doesn't work, see [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

3. **Check Data Exists:**
   ```bash
   docker exec -it k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;"
   ```

### Create Cost Trends Panel

1. Click **+ → Create Dashboard**
2. Click **Add visualization**
3. Select **TimescaleDB** data source
4. Switch to **Code** mode (top right)
5. **IMPORTANT:** Set Format to **"Time series"** (not "Table")
6. Paste this query:

```sql
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = 1
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time
```

6. **Format:** Must be set to **"Time series"** (dropdown in query editor)
7. Click **Run query**
   - If no response, check browser console (F12) for errors
   - Verify data source connection (Step 3 above)
8. If query works, click **Apply** to add panel
9. Click **Save dashboard**

### Add More Panels

Repeat the process with queries from `example-queries.sql`:

- **Cost by Namespace** - Use query #1 (table format)
- **Cost by Cluster** - Use query #3 (bar chart format)
- **Utilization** - Use query #4 (time series format)

## Step 5: Add Dashboard Variables (Optional)

For dynamic filtering:

1. Click **Dashboard settings** (gear icon)
2. Go to **Variables** tab
3. Click **Add variable**
4. Configure:
   - **Name:** `tenant_id`
   - **Type:** Query
   - **Data source:** TimescaleDB
   - **Query:** `SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id`
5. Use `$tenant_id` in your queries

## Example Queries

See [`example-queries.sql`](./example-queries.sql) for 10 ready-to-use queries including:
- Cost trends
- Cost by namespace/cluster
- Utilization metrics
- Right-sizing candidates
- Node costs

## Troubleshooting

### "Run query" Button Does Nothing

**Quick Fixes:**

1. **Check Data Source Connection:**
   - Go to Configuration → Data Sources → TimescaleDB
   - Click "Test" - should show "Data source is working"
   - If it fails, check TimescaleDB is running: `docker ps | grep timescaledb`

2. **Check Browser Console:**
   - Press F12 → Console tab
   - Click "Run query" again
   - Look for errors (network, authentication, etc.)

3. **Test Simple Query First:**
   ```sql
   SELECT 1 as test
   ```
   If this doesn't work, it's a connection issue.

4. **Verify Data Exists:**
   ```bash
   docker exec -it k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;"
   ```

5. **Check Time Range:**
   - Ensure Grafana time range (top right) includes your data
   - Try "Last 30 days" or custom range that includes your data

6. **Verify Query Format:**
   - For time series: Format should be "Time series"
   - For tables: Format should be "Table"

**Common Issues:**
- **No response:** Usually data source connection problem
- **Network error:** Grafana can't reach TimescaleDB
- **No data:** Time range doesn't match data, or no data in DB
- **Query error:** Check browser console for SQL errors

See [`TROUBLESHOOTING.md`](./TROUBLESHOOTING.md) for detailed troubleshooting steps.

**Can't connect to TimescaleDB?**
- Verify TimescaleDB is running: `docker ps | grep timescaledb`
- Check network: Ensure Grafana and TimescaleDB are on same network
- Test connection: `psql -h localhost -p 5433 -U ts_user -d timeseries`

**No data in queries?**
- Check time range (use last 7 days or longer)
- Verify tenant_id matches your data
- Check if data exists: `SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;`

**Performance issues?**
- Use `time_bucket()` for aggregations
- Limit time ranges
- Add indexes on frequently queried columns

## Next Steps

- Explore more visualization types (heatmaps, gauges, stat panels)
- Set up alerts for cost thresholds
- Create additional dashboards for specific use cases
- Integrate with your React frontend using Grafana embed

For detailed documentation, see [`GRAFANA_SETUP.md`](./GRAFANA_SETUP.md).

