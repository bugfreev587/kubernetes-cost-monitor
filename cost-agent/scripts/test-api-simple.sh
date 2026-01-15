#!/bin/sh
# Simple API Server Endpoint Testing Script
# Uses wget (most common in Alpine) with simpler logic
# Designed to run inside cost-agent pod

# Configuration
API_SERVER_URL="${AGENT_SERVER_URL:-https://api-server-production-7a9d.up.railway.app}"
API_KEY="${AGENT_API_KEY:-17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o}"

# Remove /v1/ingest if present
BASE_URL=$(echo "$API_SERVER_URL" | sed 's|/v1/ingest$||')

echo "=========================================="
echo "API Server Endpoint Tests"
echo "Base URL: $BASE_URL"
echo "API Key: ${API_KEY%%:*}:***"
echo "=========================================="
echo ""

# Test 1: Health Check
echo "Test 1: GET /v1/health"
if wget -q -O- -T 30 "${BASE_URL}/v1/health" >/dev/null 2>&1; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
fi
echo ""

# Test 2: Get Recommendations
echo "Test 2: GET /v1/recommendations"
if wget -q -O- -T 30 "${BASE_URL}/v1/recommendations" >/dev/null 2>&1; then
    echo "✓ Get recommendations passed"
else
    echo "✗ Get recommendations failed"
fi
echo ""

# Test 3: Ingest Metrics (Protected)
echo "Test 3: POST /v1/ingest (Protected)"
test_data='{"cluster_name":"test","timestamp":0,"namespace_costs":{},"node_metrics":[]}'
response=$(echo "$test_data" | wget -q -O- -T 60 \
    --header="Content-Type: application/json" \
    --header="Authorization: ApiKey $API_KEY" \
    --post-data=- \
    "${BASE_URL}/v1/ingest" 2>&1)
if [ $? -eq 0 ]; then
    echo "✓ Ingest metrics passed"
    echo "Response: $response"
else
    echo "✗ Ingest metrics failed"
    echo "Error: $response"
fi
echo ""

echo "=========================================="
echo "Tests completed"
echo "=========================================="

