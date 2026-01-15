# Grafana Dashboards

This directory contains Grafana dashboard JSON files that can be imported into Grafana.

## Importing Dashboards

### Method 1: Auto-provisioning
Dashboards in this directory are automatically loaded by Grafana if provisioning is configured correctly.

### Method 2: Manual Import
1. Go to Grafana → Dashboards → Import
2. Click "Upload JSON file"
3. Select a dashboard JSON file from this directory
4. Select the TimescaleDB data source
5. Click "Import"

## Creating Dashboards

### Quick Start Dashboard

1. **Create a new dashboard** in Grafana
2. **Add a panel** → Choose visualization type
3. **Configure data source** → Select "TimescaleDB"
4. **Write SQL query** (see `../example-queries.sql`)

### Example: Cost Trends Panel

1. Add panel → Time series
2. Data source: TimescaleDB
3. Query:
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

4. Format: Time series
5. Save panel

## Dashboard Variables

Add variables for dynamic filtering:

### Tenant ID Variable
- Name: `tenant_id`
- Type: Query
- Data source: TimescaleDB
- Query: `SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id`
- Use in queries: `WHERE tenant_id = $tenant_id`

### Namespace Variable
- Name: `namespace`
- Type: Query
- Data source: TimescaleDB
- Query: `SELECT DISTINCT namespace FROM pod_metrics WHERE tenant_id = $tenant_id ORDER BY namespace`
- Use in queries: `AND namespace = $namespace`

## Recommended Panels

1. **Cost Trends** - Time series showing daily/weekly costs
2. **Cost by Namespace** - Table or bar chart
3. **Cost by Cluster** - Bar chart comparing clusters
4. **Utilization vs Requests** - Time series showing efficiency
5. **Top Underutilized Pods** - Table for right-sizing
6. **Node Costs** - Time series from node_metrics

See `../example-queries.sql` for ready-to-use SQL queries.

