#!/bin/bash

# API Server Testing Script
# Usage: ./test-api.sh [base_url]
# Example: ./test-api.sh http://localhost:8080

BASE_URL="${1:-http://localhost:8080}"
API_KEY=""

echo "=== API Server Testing Script ==="
echo "Base URL: ${BASE_URL}"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
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

# Test 1: Health Check
echo "1. Testing Health Check..."
response=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/v1/health")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    print_success "Health check passed"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
else
    print_error "Health check failed (HTTP $http_code)"
    echo "$body"
fi
echo ""

# Test 2: Create API Key
echo "2. Creating API Key..."
response=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/v1/admin/api_keys" \
    -H "Content-Type: application/json" \
    -d '{"tenant_id": 1, "scopes": [], "expires_at": null}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 201 ]; then
    print_success "API key created"
    key_id=$(echo "$body" | jq -r '.key_id' 2>/dev/null)
    secret=$(echo "$body" | jq -r '.secret' 2>/dev/null)
    API_KEY="${key_id}:${secret}"
    print_info "API Key: ${API_KEY}"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
else
    print_error "Failed to create API key (HTTP $http_code)"
    echo "$body"
    exit 1
fi
echo ""

# Test 3: Ingest Metrics
if [ -n "$API_KEY" ]; then
    echo "3. Testing Ingest Endpoint..."
    response=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/v1/ingest" \
        -H "Content-Type: application/json" \
        -H "Authorization: ApiKey ${API_KEY}" \
        -d '{
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
        }')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 202 ]; then
        print_success "Metrics ingested successfully"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        print_error "Failed to ingest metrics (HTTP $http_code)"
        echo "$body"
    fi
else
    print_error "Skipping ingest test - no API key"
fi
echo ""

# Test 4: Get Recommendations
echo "4. Testing Get Recommendations..."
response=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/v1/recommendations")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    print_success "Recommendations retrieved"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    rec_count=$(echo "$body" | jq 'length' 2>/dev/null || echo "0")
    print_info "Found $rec_count recommendation(s)"
else
    print_error "Failed to get recommendations (HTTP $http_code)"
    echo "$body"
fi
echo ""

# Test 5: Test invalid API key
echo "5. Testing Invalid API Key (should fail)..."
response=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/v1/ingest" \
    -H "Content-Type: application/json" \
    -H "Authorization: ApiKey invalid:key" \
    -d '{"cluster_name": "test"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 401 ]; then
    print_success "Correctly rejected invalid API key (HTTP 401)"
else
    print_error "Expected 401 but got HTTP $http_code"
fi
echo ""

echo "=== Testing Complete ==="
echo ""
print_info "To use the created API key in other requests:"
echo "export API_KEY=\"${API_KEY}\""
echo ""
echo "Example:"
echo "curl -X POST ${BASE_URL}/v1/ingest \\"
echo "  -H 'Authorization: ApiKey \${API_KEY}' \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{...}'"

