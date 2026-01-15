#!/bin/sh
# API Server Endpoint Testing Script for Alpine Linux
# Tests all API server endpoints from cost-agent pod

# Configuration
API_SERVER_URL="${AGENT_SERVER_URL:-https://api-server-production-7a9d.up.railway.app}"
API_KEY="${AGENT_API_KEY:-17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o}"

# Remove /v1/ingest if present
BASE_URL=$(echo "$API_SERVER_URL" | sed 's|/v1/ingest$||')

# Test result tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_TEST_NAMES=""

echo "=========================================="
echo "API Server Endpoint Tests"
echo "Base URL: $BASE_URL"
echo "API Key: ${API_KEY%%:*}:***"
echo "=========================================="
echo ""

# Test 1: Health Check
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 1: GET /v1/health"
if wget -q -O- -T 30 "${BASE_URL}/v1/health" 2>/dev/null; then
    echo "✓ Health check passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Health check failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 1 (Health Check)\n"
fi
echo ""

# Test 2: Get Recommendations (Protected)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 2: GET /v1/recommendations (Protected)"
if wget -q -O- -T 30 \
    --header="Authorization: ApiKey $API_KEY" \
    "${BASE_URL}/v1/recommendations" 2>/dev/null; then
    echo "✓ Get recommendations passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Get recommendations failed"
    echo "  Note: This endpoint requires API key authentication"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 2 (Get Recommendations)\n"
fi
echo ""

# Test 3: Ingest Metrics (Protected)
echo "Test 3: POST /v1/ingest (Protected)"
test_data='{"cluster_name":"test","timestamp":0,"pod_metrics":[],"namespace_costs":{},"node_metrics":[]}'
# Use temp file for POST data (more compatible with busybox wget)
echo "$test_data" > /tmp/test-payload.json 2>/dev/null || echo "$test_data" > /tmp/test-payload.json

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if wget -q -O- -T 60 \
    --header="Content-Type: application/json" \
    --header="Authorization: ApiKey $API_KEY" \
    --post-file=/tmp/test-payload.json \
    "${BASE_URL}/v1/ingest" 2>/dev/null; then
    echo "✓ Ingest metrics passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    # Fallback: try with --post-data if --post-file doesn't work
    if wget -q -O- -T 60 \
        --header="Content-Type: application/json" \
        --header="Authorization: ApiKey $API_KEY" \
        --post-data="$test_data" \
        "${BASE_URL}/v1/ingest" 2>/dev/null; then
        echo "✓ Ingest metrics passed (fallback method)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "✗ Ingest metrics failed"
        echo "  Try: wget -O- --header=\"Authorization: ApiKey \$AGENT_API_KEY\" --post-data='...' ${BASE_URL}/v1/ingest"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 3 (Ingest Metrics)\n"
    fi
fi
rm -f /tmp/test-payload.json 2>/dev/null
echo ""

# Test 4: Test Invalid API Key
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 4: POST /v1/ingest with invalid API key (should fail)"
response=$(wget -q -O- -T 30 \
    --header="Content-Type: application/json" \
    --header="Authorization: ApiKey invalid:key" \
    --post-data="$test_data" \
    "${BASE_URL}/v1/ingest" 2>&1)
if echo "$response" | grep -q "401\|Unauthorized\|error"; then
    echo "✓ Correctly rejected invalid API key"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "? Check response (expected 401)"
    echo "  Response: $response"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 4 (Invalid API Key Validation)\n"
fi
echo ""

# Test 5: Get Cost by Namespace (Protected)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 5: GET /v1/costs/namespaces (Protected)"
if wget -q -O- -T 30 \
    --header="Authorization: ApiKey $API_KEY" \
    "${BASE_URL}/v1/costs/namespaces" 2>/dev/null; then
    echo "✓ Get cost by namespace passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Get cost by namespace failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 5 (Cost by Namespace)\n"
fi
echo ""

# Test 6: Get Cost by Cluster (Protected)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 6: GET /v1/costs/clusters (Protected)"
if wget -q -O- -T 30 \
    --header="Authorization: ApiKey $API_KEY" \
    "${BASE_URL}/v1/costs/clusters" 2>/dev/null; then
    echo "✓ Get cost by cluster passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Get cost by cluster failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 6 (Cost by Cluster)\n"
fi
echo ""

# Test 7: Get Utilization vs Requests (Protected)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 7: GET /v1/costs/utilization (Protected)"
if wget -q -O- -T 30 \
    --header="Authorization: ApiKey $API_KEY" \
    "${BASE_URL}/v1/costs/utilization" 2>/dev/null; then
    echo "✓ Get utilization vs requests passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Get utilization vs requests failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 7 (Utilization vs Requests)\n"
fi
echo ""

# Test 8: Get Cost Trends (Protected)
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo "Test 8: GET /v1/costs/trends (Protected)"
if wget -q -O- -T 30 \
    --header="Authorization: ApiKey $API_KEY" \
    "${BASE_URL}/v1/costs/trends?interval=daily" 2>/dev/null; then
    echo "✓ Get cost trends passed"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "✗ Get cost trends failed"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    FAILED_TEST_NAMES="${FAILED_TEST_NAMES}Test 8 (Cost Trends)\n"
fi
echo ""

echo "=========================================="
echo "Tests completed"
echo "=========================================="
echo ""
echo "=========================================="
echo "TEST REPORT SUMMARY"
echo "=========================================="
echo "Total Tests:    $TOTAL_TESTS"
echo "Passed:         $PASSED_TESTS"
echo "Failed:         $FAILED_TESTS"
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo "Success Rate:   ${SUCCESS_RATE}%"
else
    echo "Success Rate:   N/A"
fi
echo ""
if [ $FAILED_TESTS -gt 0 ]; then
    echo "Failed Tests:"
    printf "$FAILED_TEST_NAMES"
    echo ""
fi
if [ $FAILED_TESTS -eq 0 ]; then
    echo "✓ All tests passed!"
    exit 0
else
    echo "✗ Some tests failed. Please review the output above."
    exit 1
fi

