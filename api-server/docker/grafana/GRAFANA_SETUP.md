# Grafana Setup Guide for K8s Cost Monitoring

This guide explains how to set up Grafana to visualize Kubernetes cost metrics stored in TimescaleDB.

## Overview

Grafana connects directly to TimescaleDB to visualize:
- Cost by namespace
- Cost by cluster
- Daily/weekly cost trends
- Resource utilization vs requests
- Right-sizing recommendations

## Prerequisites

- TimescaleDB running (from `docker-compose.yml`)
- Grafana Docker image
- Access to TimescaleDB credentials

## Quick Start

### Option 1: Using Docker Compose (Recommended)

1. **Start Grafana alongside existing services:**

```bash
cd docker
docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana
```

2. **Access Grafana:**
   - URL: http://localhost:3000
   - Default credentials:
     - Username: `admin`
     - Password: `admin` (change on first login)

3. **Verify Data Source:**
   - Go to Configuration → Data Sources
   - "TimescaleDB" should be pre-configured
   - Test the connection

4. **Import Dashboards:**
   - Dashboards are auto-provisioned from `docker/grafana/dashboards/`
   - Or manually import from Configuration → Dashboards

### Option 2: Standalone Grafana

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  --network docker_default \
  grafana/grafana:latest
```

Then manually configure the TimescaleDB data source:
- Host: `timescaledb:5432`
- Database: `timeseries`
- User: `ts_user`
- Password: `ts_pass`

## Data Source Configuration

The TimescaleDB data source is automatically configured via provisioning files:

**File:** `docker/grafana/provisioning/datasources/timescaledb.yml`

```yaml
datasource:
  name: TimescaleDB
  type: postgres
  url: timescaledb:5432
  database: timeseries
  user: ts_user
  password: ts_pass
```

### Manual Configuration

If provisioning doesn't work, configure manually:

1. Go to **Configuration → Data Sources → Add data source**
2. Select **PostgreSQL**
3. Configure:
   - **Host:** `timescaledb:5432` (or `localhost:5433` from host)
   - **Database:** `timeseries`
   - **User:** `ts_user`
   - **Password:** `ts_pass`
   - **SSL Mode:** `disable` (or configure for production)
   - **TimescaleDB:** Enable this option
   - **Version:** PostgreSQL 16

## Example Queries

### 1. Cost by Namespace (Last 7 Days)

```sql
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
ORDER BY time, namespace
```

### 2. Daily Cost Trends

```sql
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as total_cpu_cores,
  SUM(memory_request_bytes) / 1024.0 / 1024.0 / 1024.0 as total_memory_gb
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time
```

### 3. Cost by Cluster

```sql
SELECT
  cluster_name,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores,
  COUNT(DISTINCT namespace) as namespace_count,
  COUNT(DISTINCT pod_name) as pod_count
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY cluster_name
ORDER BY cpu_cores DESC
```

### 4. Resource Utilization vs Requests

```sql
SELECT
  time_bucket('1 hour', time) as time,
  namespace,
  pod_name,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as cpu_utilization_percent,
  AVG(memory_bytes) / NULLIF(AVG(memory_request_bytes), 0) * 100 as memory_utilization_percent
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY time_bucket('1 hour', time), namespace, pod_name
ORDER BY time, cpu_utilization_percent DESC
LIMIT 100
```

### 5. Top Underutilized Pods (Right-Sizing Candidates)

```sql
SELECT
  namespace,
  pod_name,
  AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) * 100 as cpu_util_pct,
  AVG(memory_bytes) / NULLIF(AVG(memory_request_bytes), 0) * 100 as mem_util_pct,
  AVG(cpu_request_millicores) as avg_cpu_request,
  AVG(cpu_millicores) as avg_cpu_usage
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
  AND cpu_request_millicores > 0
GROUP BY namespace, pod_name
HAVING AVG(cpu_millicores) / NULLIF(AVG(cpu_request_millicores), 0) < 0.5
ORDER BY cpu_util_pct ASC
LIMIT 50
```

### 6. Node Costs Over Time

```sql
SELECT
  time_bucket('1 day', time) as time,
  cluster_name,
  SUM(hourly_cost_usd) * 24 as daily_cost_usd
FROM node_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
GROUP BY time_bucket('1 day', time), cluster_name
ORDER BY time, cluster_name
```

## Dashboard Variables

Use Grafana variables for dynamic filtering:

### Tenant ID Variable

```sql
-- Variable Query
SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id

-- Use in queries
WHERE tenant_id = $tenant_id
```

### Namespace Variable

```sql
-- Variable Query
SELECT DISTINCT namespace FROM pod_metrics WHERE tenant_id = $tenant_id ORDER BY namespace

-- Use in queries
WHERE tenant_id = $tenant_id AND namespace = $namespace
```

### Cluster Variable

```sql
-- Variable Query
SELECT DISTINCT cluster_name FROM pod_metrics WHERE tenant_id = $tenant_id ORDER BY cluster_name

-- Use in queries
WHERE tenant_id = $tenant_id AND cluster_name = $cluster
```

## Pre-built Dashboards

### Cost Overview Dashboard

Located at: `docker/grafana/dashboards/cost-overview.json`

Includes:
- Cost by namespace table
- Daily cost trends graph
- Cost by cluster bar chart
- Resource utilization vs requests

**Import:**
1. Go to Dashboards → Import
2. Upload `docker/grafana/dashboards/cost-overview.json`
3. Select TimescaleDB data source
4. Adjust tenant_id variable as needed

## Production Considerations

### Security

1. **Change default admin password**
2. **Use environment variables for credentials:**
   ```yaml
   environment:
     - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
   ```

3. **Enable SSL/TLS for TimescaleDB connection**
4. **Use Grafana API keys for programmatic access**
5. **Restrict network access** (use internal networks)

### Performance

1. **Query optimization:**
   - Use `time_bucket()` for aggregations
   - Add indexes on frequently queried columns
   - Limit result sets with `LIMIT`

2. **Caching:**
   - Enable query caching in Grafana
   - Use appropriate refresh intervals

3. **Connection pooling:**
   - Configure max connections in data source
   - Monitor connection usage

### Multi-tenant Support

For multi-tenant dashboards:

1. **Create tenant-specific dashboards** using variables
2. **Use Grafana organizations** for tenant isolation
3. **Implement row-level security** in PostgreSQL (if needed)
4. **Use API proxy** instead of direct DB access for better security

## Troubleshooting

### Grafana can't connect to TimescaleDB

- Verify TimescaleDB is running: `docker ps | grep timescaledb`
- Check network connectivity: `docker network ls`
- Ensure Grafana is on the same Docker network
- Verify credentials in data source configuration

### Queries return no data

- Check time range in dashboard
- Verify tenant_id matches your data
- Ensure data exists in TimescaleDB:
  ```sql
  SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;
  ```

### Performance issues

- Add indexes on `time`, `tenant_id`, `namespace`, `cluster_name`
- Use time_bucket() for aggregations
- Limit query time ranges
- Reduce panel refresh intervals

## Example Dashboard Panels

### Panel Types

1. **Time Series** - For cost trends over time
2. **Table** - For cost breakdowns by namespace/cluster
3. **Bar Chart** - For comparing costs across clusters
4. **Gauge** - For utilization percentages
5. **Stat** - For summary metrics (total cost, pod count)

### Useful Transformations

- **Group by** - Group metrics by namespace/cluster
- **Calculate field** - Convert millicores to cores, bytes to GB
- **Organize fields** - Rename columns for clarity
- **Filter by name** - Filter specific namespaces/pods

## Next Steps

1. **Customize dashboards** for your specific needs
2. **Set up alerts** for cost thresholds
3. **Create additional dashboards** for specific use cases
4. **Integrate with your React frontend** (optional) using Grafana embed

## Resources

- [Grafana Documentation](https://grafana.com/docs/grafana/latest/)
- [TimescaleDB + Grafana Guide](https://docs.timescale.com/timescaledb/latest/tutorials/grafana/)
- [PostgreSQL Data Source](https://grafana.com/docs/grafana/latest/datasources/postgres/)

