# Connecting Grafana to Production TimescaleDB

This guide shows how to connect Grafana to your production TimescaleDB database on Railway.

## Production Database Connection

**Connection URL:**
```
postgresql://railway:g39zhg3gb1xis72ac7yv6clzbag3z2dl@tramway.proxy.rlwy.net:43259/railway
```

**Connection Details:**
- Host: `tramway.proxy.rlwy.net`
- Port: `43259`
- Database: `railway`
- User: `railway`
- Password: `g39zhg3gb1xis72ac7yv6clzbag3z2dl`
- SSL: Required (Railway uses SSL)

## Option 1: Auto-Provisioning (Recommended)

### Update Data Source Configuration

1. **Replace the local data source** with production:

```bash
# Backup local config
mv docker/grafana/provisioning/datasources/timescaledb.yml docker/grafana/provisioning/datasources/timescaledb-local.yml

# Use production config
cp docker/grafana/provisioning/datasources/timescaledb-production.yml docker/grafana/provisioning/datasources/timescaledb.yml
```

2. **Restart Grafana:**
```bash
docker restart k8s_cost_grafana
```

3. **Verify Connection:**
   - Go to Configuration → Data Sources
   - Click "TimescaleDB Production"
   - Click "Test" - should show "Data source is working"

## Option 2: Manual Configuration

1. **Go to Grafana UI:** http://localhost:3000
2. **Configuration → Data Sources → Add data source**
3. **Select PostgreSQL**
4. **Configure:**
   - **Name:** `TimescaleDB Production`
   - **Host:** `tramway.proxy.rlwy.net:43259`
   - **Database:** `railway`
   - **User:** `railway`
   - **Password:** `g39zhg3gb1xis72ac7yv6clzbag3z2dl`
   - **SSL Mode:** `require` (important for Railway)
   - **TimescaleDB:** ✅ Enable
   - **Version:** PostgreSQL 16
5. **Click "Save & Test"**

## Option 3: Environment Variables (Most Secure)

For production deployments, use environment variables instead of hardcoded credentials:

### Update docker-compose.grafana.yml

```yaml
services:
  grafana:
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      # Production DB credentials from environment
      - GF_DATASOURCES_TIMESCALEDB_HOST=tramway.proxy.rlwy.net:43259
      - GF_DATASOURCES_TIMESCALEDB_DATABASE=railway
      - GF_DATASOURCES_TIMESCALEDB_USER=railway
      - GF_DATASOURCES_TIMESCALEDB_PASSWORD_FILE=/run/secrets/timescale_password
    secrets:
      - timescale_password
```

Create a secrets file:
```bash
echo "g39zhg3gb1xis72ac7yv6clzbag3z2dl" > docker/grafana/secrets/timescale_password
chmod 600 docker/grafana/secrets/timescale_password
```

## Testing the Connection

### Quick Test Query

Once connected, test with a simple query:

```sql
SELECT COUNT(*) as total_rows 
FROM pod_metrics 
WHERE tenant_id = 1;
```

### Check Data Time Range

```sql
SELECT 
  MIN(time) as earliest,
  MAX(time) as latest,
  COUNT(*) as total_rows
FROM pod_metrics
WHERE tenant_id = 1;
```

## Security Considerations

⚠️ **Important Security Notes:**

1. **Credentials in Files:**
   - The production password is currently in the config file
   - For production Grafana, use environment variables or secrets
   - Consider using Grafana's encrypted secure storage

2. **Network Access:**
   - Ensure Grafana can reach Railway's database
   - Railway databases are typically accessible from the internet
   - Consider IP whitelisting if possible

3. **SSL/TLS:**
   - Railway requires SSL connections
   - Always use `sslmode: require` or `sslmode: verify-full`

4. **Read-Only Access:**
   - Consider creating a read-only database user for Grafana
   - Limits potential damage if credentials are compromised

## Switching Between Local and Production

### Quick Switch Script

Create a script to switch between local and production:

```bash
#!/bin/bash
# switch-datasource.sh

if [ "$1" == "production" ]; then
    cp docker/grafana/provisioning/datasources/timescaledb-production.yml \
       docker/grafana/provisioning/datasources/timescaledb.yml
    echo "Switched to production TimescaleDB"
elif [ "$1" == "local" ]; then
    cp docker/grafana/provisioning/datasources/timescaledb-local.yml \
       docker/grafana/provisioning/datasources/timescaledb.yml
    echo "Switched to local TimescaleDB"
else
    echo "Usage: $0 [production|local]"
    exit 1
fi

docker restart k8s_cost_grafana
```

## Query Adjustments for Production

Production database may have:
- Different tenant IDs
- Different time ranges
- More data (may need query optimization)

### Find Your Tenant IDs

```sql
SELECT DISTINCT tenant_id 
FROM pod_metrics 
ORDER BY tenant_id;
```

### Optimize Queries for Large Datasets

For production with lots of data:

1. **Use appropriate time ranges:**
   ```sql
   WHERE time >= NOW() - INTERVAL '7 days'  -- Instead of 30 days
   ```

2. **Add LIMIT clauses:**
   ```sql
   ORDER BY time DESC
   LIMIT 1000
   ```

3. **Use time_bucket for aggregations:**
   ```sql
   time_bucket('1 hour', time)  -- Instead of raw timestamps
   ```

## Troubleshooting Production Connection

### Connection Timeout

If connection times out:
- Check firewall rules
- Verify Railway database is accessible
- Test from command line:
  ```bash
  psql "postgresql://railway:g39zhg3gb1xis72ac7yv6clzbag3z2dl@tramway.proxy.rlwy.net:43259/railway?sslmode=require" -c "SELECT 1;"
  ```

### SSL Errors

If you get SSL errors:
- Ensure `sslmode: require` is set
- Try `sslmode: verify-full` if certificate validation fails
- Check Railway's SSL certificate requirements

### Authentication Errors

If authentication fails:
- Verify credentials are correct
- Check if database user has proper permissions
- Ensure user can read from `pod_metrics` and `node_metrics` tables

## Example Production Queries

### Cost Trends (Production)

```sql
SELECT
  time_bucket('1 day', time) as time,
  SUM(cpu_request_millicores) / 1000.0 as cpu_cores
FROM pod_metrics
WHERE tenant_id = $tenant_id
  AND time >= $__timeFrom()
  AND time <= $__timeTo()
  AND pod_name != '__aggregate__'
GROUP BY time_bucket('1 day', time)
ORDER BY time
```

### Cost by Namespace (Production)

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

## Next Steps

1. **Test the connection** using the test queries above
2. **Create dashboards** using production data
3. **Set up alerts** for cost thresholds
4. **Optimize queries** for your data volume
5. **Consider read replicas** if query load is high

