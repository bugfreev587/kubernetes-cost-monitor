#!/bin/bash
# Quick script to test Grafana → TimescaleDB connection

echo "=========================================="
echo "Grafana → TimescaleDB Connection Test"
echo "=========================================="
echo ""

# Check if containers are running
echo "1. Checking containers..."
TIMESCALE_RUNNING=$(docker ps | grep -c k8s_cost_timescaledb)
GRAFANA_RUNNING=$(docker ps | grep -c k8s_cost_grafana)

if [ "$TIMESCALE_RUNNING" -eq 0 ]; then
    echo "   ✗ TimescaleDB container is not running"
    echo "   Run: docker-compose up -d timescaledb"
    exit 1
else
    echo "   ✓ TimescaleDB container is running"
fi

if [ "$GRAFANA_RUNNING" -eq 0 ]; then
    echo "   ✗ Grafana container is not running"
    echo "   Run: docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana"
    exit 1
else
    echo "   ✓ Grafana container is running"
fi

echo ""
echo "2. Testing TimescaleDB connection..."
docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT 1 as test;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ TimescaleDB is accessible"
else
    echo "   ✗ Cannot connect to TimescaleDB"
    exit 1
fi

echo ""
echo "3. Checking data in pod_metrics..."
ROW_COUNT=$(docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -t -c "SELECT COUNT(*) FROM pod_metrics WHERE tenant_id = 1;" 2>/dev/null | tr -d ' ')
if [ -z "$ROW_COUNT" ] || [ "$ROW_COUNT" = "0" ]; then
    echo "   ⚠ No data found for tenant_id = 1"
    echo "   Checking all tenants..."
    docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT DISTINCT tenant_id FROM pod_metrics ORDER BY tenant_id;"
else
    echo "   ✓ Found $ROW_COUNT rows for tenant_id = 1"
fi

echo ""
echo "4. Checking data time range..."
docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -c "
SELECT 
  MIN(time) as earliest_data,
  MAX(time) as latest_data,
  NOW() as current_time
FROM pod_metrics
WHERE tenant_id = 1;
"

echo ""
echo "5. Testing TimescaleDB extension..."
docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT * FROM pg_extension WHERE extname = 'timescaledb';" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ TimescaleDB extension is installed"
else
    echo "   ✗ TimescaleDB extension not found"
    echo "   Run: CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;"
fi

echo ""
echo "6. Testing time_bucket function..."
docker exec k8s_cost_timescaledb psql -U ts_user -d timeseries -c "SELECT time_bucket('1 day', NOW()) as test_bucket;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ time_bucket() function works"
else
    echo "   ✗ time_bucket() function not available"
fi

echo ""
echo "7. Testing network connectivity from Grafana..."
docker exec k8s_cost_grafana sh -c "nc -zv timescaledb 5432" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "   ✓ Grafana can reach TimescaleDB on port 5432"
else
    echo "   ✗ Grafana cannot reach TimescaleDB"
    echo "   Check Docker network configuration"
fi

echo ""
echo "=========================================="
echo "Test Complete"
echo "=========================================="
echo ""
echo "If all tests pass but queries still don't work:"
echo "1. Check Grafana data source configuration"
echo "2. Open browser console (F12) and look for errors"
echo "3. Verify time range in Grafana includes your data"
echo "4. Check TROUBLESHOOTING.md for more help"

