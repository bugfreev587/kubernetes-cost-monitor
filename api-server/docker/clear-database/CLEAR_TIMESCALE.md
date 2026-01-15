# Clearing TimescaleDB Data

This guide provides several methods to clear data from TimescaleDB, depending on your needs.

## Quick Reference

### Using Makefile (Docker Compose)

```bash
# Clear all data (preserves schema)
make clear-timescale

# Complete reset (drops and recreates tables)
make reset-timescale
```

### Using psql Directly

```bash
# Clear all data
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/clear-timescale-data.sql

# Complete reset
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/clear-timescale-complete.sql
```

## Methods

### Method 1: TRUNCATE (Recommended for Development)

**Fastest method** - Clears all data but preserves schema and hypertable structure.

```sql
TRUNCATE TABLE pod_metrics;
TRUNCATE TABLE node_metrics;
```

**Pros:**
- Very fast
- Preserves schema
- Preserves hypertable configuration
- Safe for development

**Cons:**
- Cannot filter (clears everything)
- Cannot rollback

**Usage:**
```bash
# Using Docker
docker exec -i k8s_cost_timescaledb psql -U ts_user -d timeseries -c "TRUNCATE TABLE pod_metrics, node_metrics;"

# Using psql
psql -h localhost -p 5433 -U ts_user -d timeseries -c "TRUNCATE TABLE pod_metrics, node_metrics;"
```

### Method 2: DELETE with Filters

**Selective deletion** - Remove specific data based on criteria.

```sql
-- Delete data for a specific tenant
DELETE FROM pod_metrics WHERE tenant_id = 1;
DELETE FROM node_metrics WHERE tenant_id = 1;

-- Delete old data (older than 7 days)
DELETE FROM pod_metrics WHERE time < NOW() - INTERVAL '7 days';
DELETE FROM node_metrics WHERE time < NOW() - INTERVAL '7 days';

-- Delete data for a specific cluster
DELETE FROM pod_metrics WHERE cluster_name = 'cluster-a';
DELETE FROM node_metrics WHERE cluster_name = 'cluster-a';
```

**Pros:**
- Can filter by tenant, time, cluster, etc.
- More control over what gets deleted
- Can be rolled back (if in a transaction)

**Cons:**
- Slower than TRUNCATE
- May need VACUUM after large deletions

**Usage:**
```bash
# Edit docker/clear-timescale-data.sql to uncomment desired DELETE statements
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/clear-timescale-data.sql
```

### Method 3: Complete Reset (DROP & RECREATE)

**Nuclear option** - Drops and recreates tables. Use only in development!

```sql
DROP TABLE IF EXISTS pod_metrics CASCADE;
DROP TABLE IF EXISTS node_metrics CASCADE;
-- Then recreate using timescale-init.sql
```

**Pros:**
- Complete clean slate
- Removes any schema inconsistencies

**Cons:**
- **DESTROYS ALL DATA**
- Requires recreating hypertables
- Only for development/testing

**Usage:**
```bash
# Using Makefile (includes safety prompt)
make reset-timescale

# Or manually
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/clear-timescale-complete.sql
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/timescale-init.sql
```

## Docker-Specific Commands

### If using Docker Compose:

```bash
# Connect to TimescaleDB container
docker exec -it k8s_cost_timescaledb psql -U ts_user -d timeseries

# Then run SQL commands directly
TRUNCATE TABLE pod_metrics, node_metrics;
```

### If using Docker volumes:

```bash
# Stop containers
docker-compose down

# Remove TimescaleDB volume (WARNING: Deletes all data permanently!)
docker volume rm docker_tsdata

# Restart containers (will recreate with fresh data)
docker-compose up -d
```

## Production Considerations

⚠️ **WARNING**: Never use `DROP TABLE` or `TRUNCATE` in production without:
1. **Backup first**: `pg_dump -h <host> -U <user> -d <database> > backup.sql`
2. **Maintenance window**: Schedule during low-traffic periods
3. **Verification**: Test in staging first
4. **Rollback plan**: Have a restore procedure ready

### Recommended Production Approach:

```sql
-- Use DELETE with time-based filtering for production
DELETE FROM pod_metrics WHERE time < NOW() - INTERVAL '90 days';
DELETE FROM node_metrics WHERE time < NOW() - INTERVAL '90 days';

-- Then VACUUM to reclaim space
VACUUM ANALYZE pod_metrics;
VACUUM ANALYZE node_metrics;
```

## Verification

After clearing data, verify:

```sql
-- Check row counts
SELECT 'pod_metrics' as table_name, COUNT(*) as row_count FROM pod_metrics
UNION ALL
SELECT 'node_metrics' as table_name, COUNT(*) as row_count FROM node_metrics;

-- Check hypertable status
SELECT 
    schemaname,
    tablename,
    hypertable_name,
    num_dimensions
FROM timescaledb_information.hypertables
WHERE tablename IN ('pod_metrics', 'node_metrics');
```

## Troubleshooting

### "Cannot truncate a hypertable"
TimescaleDB hypertables can be truncated normally. If you get this error, try:
```sql
-- Use DELETE instead
DELETE FROM pod_metrics;
DELETE FROM node_metrics;
```

### "Table does not exist"
If tables are missing, recreate them:
```bash
psql -h localhost -p 5433 -U ts_user -d timeseries -f docker/timescale-init.sql
```

### Performance Issues After Large Deletions
Run VACUUM to reclaim space:
```sql
VACUUM ANALYZE pod_metrics;
VACUUM ANALYZE node_metrics;
```

