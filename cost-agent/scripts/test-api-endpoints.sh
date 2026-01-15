#!/bin/sh
# API Server Endpoint Testing Script
# Designed to run inside cost-agent pod (Alpine Linux)
# Tests all API server endpoints

set -e

# Colors for output (Alpine compatible)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_SERVER_URL="${AGENT_SERVER_URL:-https://api-server-production-7a9d.up.railway.app}"
API_KEY="${AGENT_API_KEY:-17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o}"

# Remove /v1/ingest if present (we'll add paths per endpoint)
BASE_URL=$(echo "$API_SERVER_URL" | sed 's|/v1/ingest$||')

# Test counter
PASSED=0
FAILED=0

# Function to print colored output
print_success() {
    printf "${GREEN}✓${NC} $1\n"
}

print_error() {
    printf "${RED}✗${NC} $1\n"
}

print_info() {
    printf "${YELLOW}→${NC} $1\n"
}

print_header() {
    printf "\n${BLUE}=== $1 ===${NC}\n"
}

# Check if wget or curl is available
if command -v wget >/dev/null 2>&1; then
    HTTP_CLIENT="wget"
elif command -v curl >/dev/null 2>&1; then
    HTTP_CLIENT="curl"
else
    print_error "Neither wget nor curl is available. Cannot run tests."
    exit 1
fi

# HTTP request function using wget
wget_request() {
    method="$1"
    url="$2"
    headers="$3"
    data="$4"
    expected_status="$5"
    
    if [ -z "$data" ]; then
        response=$(wget -O- -T 30 --timeout=30 --header="Content-Type: application/json" \
            $(echo "$headers" | sed 's/^/--header="/;s/$/"/') \
            --method="$method" \
            "$url" 2>&1)
    else
        response=$(echo "$data" | wget -O- -T 30 --timeout=30 --header="Content-Type: application/json" \
            $(echo "$headers" | sed 's/^/--header="/;s/$/"/') \
            --method="$method" \
            --body-data=- \
            "$url" 2>&1)
    fi
    
    http_code=$(echo "$response" | grep -oP 'HTTP/[0-9.]+ \K[0-9]+' | tail -1 || echo "")
    body=$(echo "$response" | sed -n '/^HTTP/,${/^HTTP/d; p}')
    
    if [ "$http_code" = "$expected_status" ]; then
        print_success "HTTP $http_code"
        echo "$body"
        return 0
    else
        print_error "Expected HTTP $expected_status, got $http_code"
        echo "$body"
        return 1
    fi
}

# HTTP request function using curl
curl_request() {
    method="$1"
    url="$2"
    headers="$3"
    data="$4"
    expected_status="$5"
    
    headers_arg=""
    for h in $headers; do
        headers_arg="$headers_arg -H \"$h\""
    done
    
    if [ -z "$data" ]; then
        response=$(eval curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            $headers_arg \
            "$url")
    else
        response=$(echo "$data" | eval curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            $headers_arg \
            -d @- \
            "$url")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        print_success "HTTP $http_code"
        echo "$body"
        return 0
    else
        print_error "Expected HTTP $expected_status, got $http_code"
        echo "$body"
        return 1
    fi
}

# Unified request function
make_request() {
    method="$1"
    endpoint="$2"
    auth_header="$3"
    data="$4"
    expected_status="$5"
    
    url="${BASE_URL}${endpoint}"
    headers="$auth_header"
    
    print_info "Testing: $method $endpoint"
    
    if [ "$HTTP_CLIENT" = "wget" ]; then
        if wget_request "$method" "$url" "$headers" "$data" "$expected_status"; then
            PASSED=$((PASSED + 1))
            return 0
        else
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        if curl_request "$method" "$url" "$headers" "$data" "$expected_status"; then
            PASSED=$((PASSED + 1))
            return 0
        else
            FAILED=$((FAILED + 1))
            return 1
        fi
    fi
}

# Main test execution
print_header "API Server Endpoint Tests"
echo "Base URL: $BASE_URL"
echo "HTTP Client: $HTTP_CLIENT"
echo "API Key: ${API_KEY%%:*}:***"  # Show only key_id, hide secret
echo ""

# Test 1: Health Check (No auth required)
print_header "1. Health Check"
make_request "GET" "/v1/health" "" "" "200"
echo ""

# Test 2: Get Recommendations (No auth required)
print_header "2. Get Recommendations"
make_request "GET" "/v1/recommendations" "" "" "200"
echo ""

# Test 3: Ingest Metrics (Requires API key)
print_header "3. Ingest Metrics (Protected)"
test_payload='{
  "cluster_name": "test-cluster",
  "timestamp": 0,
  "namespace_costs": {
    "default": {
      "namespace": "default",
      "pod_count": 1,
      "total_cpu_millicores": 100,
      "total_memory_bytes": 1073741824,
      "estimated_cost_usd": 0.0
    }
  },
  "node_metrics": [
    {
      "node_name": "test-node",
      "instance_type": "t3.medium",
      "cpu_capacity": 2000,
      "memory_capacity": 4294967296,
      "hourly_cost_usd": 0.0416
    }
  ]
}'
make_request "POST" "/v1/ingest" "Authorization: ApiKey $API_KEY" "$test_payload" "202"
echo ""

# Test 4: Test Invalid API Key (should fail)
print_header "4. Test Invalid API Key"
make_request "POST" "/v1/ingest" "Authorization: ApiKey invalid:key" "$test_payload" "401"
echo ""

# Test 5: Test Missing API Key (should fail)
print_header "5. Test Missing API Key"
make_request "POST" "/v1/ingest" "" "$test_payload" "401"
echo ""

# Summary
print_header "Test Summary"
printf "Tests Passed: ${GREEN}%d${NC}\n" $PASSED
printf "Tests Failed: ${RED}%d${NC}\n" $FAILED
echo ""

if [ $FAILED -eq 0 ]; then
    print_success "All tests passed!"
    exit 0
else
    print_error "Some tests failed"
    exit 1
fi

