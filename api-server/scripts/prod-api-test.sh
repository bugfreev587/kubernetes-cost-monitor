#!/bin/bash

# Production API Server Testing Script
# Tests endpoints on Railway deployment

BASE_URL="${1:-https://api-server-production-7a9d.up.railway.app}"
API_KEY=""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
test_endpoint() {
    local test_name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="$4"
    local data="$5"
    local auth_header="$6"
    
    print_info "Testing: $test_name"
    
    # Build curl command parts
    local curl_args=(
        -s
        -w "\n%{http_code}"
        -X "$method"
        "${BASE_URL}${endpoint}"
        -H "Content-Type: application/json"
    )
    
    if [ -n "$auth_header" ]; then
        curl_args+=(-H "$auth_header")
    fi
    
    if [ -n "$data" ]; then
        curl_args+=(-d "$data")
    fi
    
    response=$(curl "${curl_args[@]}")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq "$expected_status" ]; then
        print_success "$test_name - HTTP $http_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        print_error "$test_name - Expected HTTP $expected_status, got $http_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo ""
print_header "Production API Server Testing"
echo "Base URL: ${BASE_URL}"
echo ""

# Test 1: Health Check
print_header "1. Health Check"
test_endpoint "Health Check" "GET" "/v1/health" 200
echo ""

# Test 2: Create API Key
print_header "2. Create API Key"
response=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/v1/admin/api_keys" \
    -H "Content-Type: application/json" \
    -d '{"tenant_id": 1, "scopes": [], "expires_at": null}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 201 ]; then
    print_success "API key created - HTTP $http_code"
    key_id=$(echo "$body" | jq -r '.key_id' 2>/dev/null)
    secret=$(echo "$body" | jq -r '.secret' 2>/dev/null)
    API_KEY="${key_id}:${secret}"
    print_info "API Key: ${key_id}:${secret:0:20}..."
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    print_error "Failed to create API key - HTTP $http_code"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo ""
    print_error "Cannot continue without API key. Exiting."
    exit 1
fi
echo ""

# Test 3: Ingest Metrics (Protected Endpoint)
if [ -n "$API_KEY" ]; then
    print_header "3. Ingest Metrics (Protected)"
    test_endpoint "Ingest Metrics" "POST" "/v1/ingest" 202 \
        '{
            "cluster_name": "test-cluster",
            "timestamp": 0,
            "namespace_costs": {
                "default": {
                    "namespace": "default",
                    "pod_count": 3,
                    "total_cpu_millicores": 1500,
                    "total_memory_bytes": 3221225472,
                    "estimated_cost_usd": 0.0
                }
            },
            "node_metrics": [
                {
                    "node_name": "test-node-1",
                    "instance_type": "t3.medium",
                    "cpu_capacity": 2000,
                    "memory_capacity": 4294967296,
                    "hourly_cost_usd": 0.0416
                }
            ]
        }' \
        "Authorization: ApiKey ${API_KEY}"
    echo ""
else
    print_error "Skipping ingest test - no API key"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 4: Get Recommendations
print_header "4. Get Recommendations"
test_endpoint "Get Recommendations" "GET" "/v1/recommendations" 200
echo ""

# Test 5: Test Invalid API Key (should fail)
print_header "5. Test Invalid API Key"
test_endpoint "Invalid API Key Test" "POST" "/v1/ingest" 401 \
    '{"cluster_name": "test"}' \
    "Authorization: ApiKey invalid:key"
echo ""

# Test 6: Test Missing API Key (should fail)
print_header "6. Test Missing API Key"
test_endpoint "Missing API Key Test" "POST" "/v1/ingest" 401 \
    '{"cluster_name": "test"}'
echo ""

# Summary
print_header "Test Summary"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    print_success "All tests passed!"
    exit 0
else
    print_error "Some tests failed"
    exit 1
fi

