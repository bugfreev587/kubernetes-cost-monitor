# Setting Up TimescaleDB on Railway

This guide explains how to set up TimescaleDB for the API server on Railway.

## Overview

Railway provides a pre-configured TimescaleDB template that deploys a PostgreSQL database with the TimescaleDB extension installed. TimescaleDB is built on top of PostgreSQL and adds time-series capabilities.

## Step 1: Deploy TimescaleDB Service

1. **Navigate to your Railway project dashboard**
2. **Click "New" → "Template"**
3. **Search for "TimescaleDB"** or go directly to: https://railway.com/new/template/timescaledb
4. **Click "Deploy Now"**
5. **Follow the setup wizard** to add it to your existing project

## Step 2: Get Connection Details

After deployment:

1. **Click on the TimescaleDB service** in your Railway project
2. **Go to the "Variables" tab**
3. **Copy the connection URL** - Railway typically provides this as `DATABASE_URL` (if this is the only database) or `TIMESCALE_URL` / `TIMESCALE_DATABASE_URL` (if you have multiple database services)

   Note: If you already have a PostgreSQL service, Railway may prefix the variable name with the service name (e.g., `TIMESCALEDB_DATABASE_URL`).

4. **Add this to your API server service environment variables:**
   - Variable name: `TIMESCALE_URL` (or `TIMESCALE_DATABASE_URL`)
   - Value: The connection URL from the TimescaleDB service

## Step 3: Initialize the Database Schema

After connecting, you need to initialize the TimescaleDB schema:

1. **Connect to your TimescaleDB instance:**
   - From Railway dashboard: Click on TimescaleDB service → "Connect" tab → Use the provided connection string
   - Or use `psql` with the connection URL from Step 2

2. **Run the initialization SQL:**
   
   ```bash
   # Using psql (replace with your connection URL)
   psql "postgresql://user:password@host:port/database" < docker/timescale-init.sql
   ```

   Or manually execute the SQL from `docker/timescale-init.sql`:
   
   ```sql
   -- Enable TimescaleDB extension
   CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   
   -- Create pod_metrics table and hypertable
   CREATE TABLE IF NOT EXISTS pod_metrics (
     time timestamptz NOT NULL,
     tenant_id BIGINT NOT NULL,
     cluster_name TEXT,
     namespace TEXT,
     pod_name TEXT,
     node_name TEXT,
     cpu_millicores BIGINT,
     memory_bytes BIGINT,
     cpu_request_millicores BIGINT,
     memory_request_bytes BIGINT
   );
   SELECT create_hypertable('pod_metrics','time', if_not_exists => TRUE);
   
   -- Create node_metrics table and hypertable
   CREATE TABLE IF NOT EXISTS node_metrics (
     time timestamptz NOT NULL,
     tenant_id BIGINT NOT NULL,
     cluster_name TEXT,
     node_name TEXT,
     instance_type TEXT,
     cpu_capacity BIGINT,
     memory_capacity BIGINT,
     hourly_cost_usd NUMERIC(10,6)
   );
   SELECT create_hypertable('node_metrics','time', if_not_exists => TRUE);
   ```

## Step 4: Update Application Configuration

The API server code has been updated to check for `TIMESCALE_URL` environment variable first, then fall back to the config file.

### Environment Variables

Add to your API server service in Railway:

- `TIMESCALE_URL` (or `TIMESCALE_DATABASE_URL`) - Connection string from TimescaleDB service

### Configuration File (Alternative)

If not using environment variables, update `conf/api-server-prod.yaml`:

```yaml
timescale:
  dsn: "postgresql://user:password@host:port/database?sslmode=require"
```

## Step 5: Link Services (Optional)

Railway automatically makes services in the same project available to each other. You can also explicitly link them:

1. In your API server service
2. Go to "Settings" → "Connect" 
3. Link the TimescaleDB service

This may provide additional environment variables for connection.

## Step 6: Verify Connection

After deployment, check your API server logs. You should see:

```
✓ Timescale Database connected
```

You can also test the health endpoint:

```bash
curl https://your-api-url.railway.app/v1/health
```

The response should show:

```json
{
  "overall_status": "healthy",
  "postgresql": "healthy",
  "timescaledb": "healthy",
  "redis": "healthy"
}
```

## Troubleshooting

### Connection Failed

- Verify `TIMESCALE_URL` environment variable is set correctly
- Check that the TimescaleDB service is running
- Ensure the connection URL format is correct (includes SSL if required)
- Check Railway logs for both services

### Schema Initialization Failed

- Ensure you're connected to the correct database
- Verify the TimescaleDB extension is installed: `SELECT * FROM pg_extension WHERE extname = 'timescaledb';`
- Check that you have proper permissions to create tables and extensions

### Multiple Database Services

If you have both PostgreSQL and TimescaleDB services:
- Railway may prefix environment variables (e.g., `TIMESCALEDB_DATABASE_URL`)
- Update the code to check for the specific variable name Railway provides
- Check the Variables tab in both services to see what Railway provides

## Alternative: Use Same PostgreSQL Database

If you want to use the same PostgreSQL database for both regular data and time-series data:

1. Enable TimescaleDB extension in your existing PostgreSQL service:
   ```sql
   CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
   ```

2. Set `TIMESCALE_URL` to the same value as `DATABASE_URL`

3. Run the initialization SQL in that database

Note: This approach works since TimescaleDB is a PostgreSQL extension, but keeping them separate is recommended for better resource management and scaling.

