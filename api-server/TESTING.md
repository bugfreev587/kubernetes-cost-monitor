# API Server Testing Guide

This document provides curl commands to test all API server endpoints.

## Prerequisites

- API server running on `http://localhost:8080` (default)
- All infrastructure services running (PostgreSQL, TimescaleDB, Redis)

## Endpoints

### 1. Health Check

Check the health status of the API server and its dependencies.

```bash
# GET /v1/health
curl -X GET http://localhost:8080/v1/health \
  -H "Content-Type: application/json" \
  -v
```

**Expected Response (200 OK):**
```json
{
  "overall_status": "healthy",
  "postgresql": "healthy",
  "timescaledb": "healthy",
  "redis": "healthy",
  "message": ""
}
```

---

### 2. Create API Key

Create a new API key for authenticating agents.

```bash
# POST /v1/admin/api_keys
curl -X POST http://localhost:8080/v1/admin/api_keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": 1,
    "scopes": [],
    "expires_at": null
  }' \
  -v
```

**With expiration date:**
```bash
curl -X POST http://localhost:8080/v1/admin/api_keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": 1,
    "scopes": ["ingest"],
    "expires_at": "2025-12-31T23:59:59Z"
  }' \
  -v
```

**Expected Response (201 Created):**
```json
{
  "key_id": "550e8400-e29b-41d4-a716-446655440000",
  "secret": "abc123xyz..."
}
```

**Important:** Save the `key_id` and `secret` - you'll need them for authenticated requests. The format for API key is: `key_id:secret`

---

### 3. Ingest Metrics (Protected Endpoint)

Send cluster metrics to the API server. **Requires API key authentication.**

First, set your API key as an environment variable:
```bash
export API_KEY="your_key_id:your_secret"
```

Then send metrics:

```bash
# POST /v1/ingest
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey ${API_KEY}" \
  -d '{
    "cluster_name": "my-cluster",
    "timestamp": 1704067200,
    "namespace_costs": {
      "default": {
        "namespace": "default",
        "pod_count": 5,
        "total_cpu_millicores": 2000,
        "total_memory_bytes": 4294967296,
        "estimated_cost_usd": 0.0
      },
      "kube-system": {
        "namespace": "kube-system",
        "pod_count": 10,
        "total_cpu_millicores": 1000,
        "total_memory_bytes": 2147483648,
        "estimated_cost_usd": 0.0
      }
    },
    "node_metrics": [
      {
        "node_name": "node-1",
        "instance_type": "m5.large",
        "cpu_capacity": 2000,
        "memory_capacity": 8589934592,
        "hourly_cost_usd": 0.096
      },
      {
        "node_name": "node-2",
        "instance_type": "m5.large",
        "cpu_capacity": 2000,
        "memory_capacity": 8589934592,
        "hourly_cost_usd": 0.096
      }
    ]
  }' \
  -v
```

**Alternative: Using X-Api-Key header:**
```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: ${API_KEY}" \
  -d '{
    "cluster_name": "my-cluster",
    "timestamp": 1704067200,
    "namespace_costs": {
      "default": {
        "namespace": "default",
        "pod_count": 5,
        "total_cpu_millicores": 2000,
        "total_memory_bytes": 4294967296,
        "estimated_cost_usd": 0.0
      }
    },
    "node_metrics": [
      {
        "node_name": "node-1",
        "instance_type": "m5.large",
        "cpu_capacity": 2000,
        "memory_capacity": 8589934592,
        "hourly_cost_usd": 0.096
      }
    ]
  }' \
  -v
```

**With current timestamp (omit timestamp field or set to 0):**
```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey ${API_KEY}" \
  -d '{
    "cluster_name": "my-cluster",
    "timestamp": 0,
    "namespace_costs": {
      "production": {
        "namespace": "production",
        "pod_count": 20,
        "total_cpu_millicores": 8000,
        "total_memory_bytes": 17179869184,
        "estimated_cost_usd": 0.0
      }
    },
    "node_metrics": [
      {
        "node_name": "node-1",
        "instance_type": "m5.xlarge",
        "cpu_capacity": 4000,
        "memory_capacity": 17179869184,
        "hourly_cost_usd": 0.192
      }
    ]
  }' \
  -v
```

**Expected Response (202 Accepted):**
```json
{
  "status": "accepted"
}
```

**Error Responses:**
- `401 Unauthorized` - Missing or invalid API key
- `400 Bad Request` - Invalid payload format

---

### 4. Get Recommendations

Retrieve all cost optimization recommendations.

```bash
# GET /v1/recommendations
curl -X GET http://localhost:8080/v1/recommendations \
  -H "Content-Type: application/json" \
  -v
```

**Expected Response (200 OK):**
```json
[
  {
    "id": 1,
    "tenant_id": 1,
    "created_at": "2024-01-01T12:00:00Z",
    "cluster_name": "my-cluster",
    "namespace": "default",
    "pod_name": "my-pod",
    "resource_type": "cpu",
    "current_request": 500,
    "recommended_request": 200,
    "potential_savings_usd": 0.05,
    "confidence": 0.95,
    "reason": "Pod is over-provisioned",
    "status": "open"
  }
]
```

---

### 5. Apply Recommendation

Apply a cost optimization recommendation.

```bash
# POST /v1/recommendations/:id/apply
curl -X POST http://localhost:8080/v1/recommendations/1/apply \
  -H "Content-Type: application/json" \
  -v
```

**Expected Response (200 OK):**
```json
{
  "status": "applied"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid recommendation ID
- `500 Internal Server Error` - Failed to apply recommendation

---

### 6. Dismiss Recommendation

Dismiss a cost optimization recommendation.

```bash
# POST /v1/recommendations/:id/dismiss
curl -X POST http://localhost:8080/v1/recommendations/1/dismiss \
  -H "Content-Type: application/json" \
  -v
```

**Expected Response (200 OK):**
```json
{
  "status": "dismissed"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid recommendation ID
- `500 Internal Server Error` - Failed to dismiss recommendation

---

### 7. Allocation API (OpenCost-Compatible)

The allocation API provides cost allocation data in an OpenCost-compatible format.

#### 7.1 Basic Allocation Query

```bash
# GET /v1/allocation
# Get cost allocations by namespace for the last 24 hours
curl -X GET "http://localhost:8080/v1/allocation?window=24h&aggregate=namespace" \
  -H "Authorization: ApiKey ${API_KEY}" \
  -v
```

**Expected Response (200 OK):**
```json
{
  "code": 200,
  "status": "success",
  "data": [
    {
      "allocations": {
        "default": {
          "name": "default",
          "properties": {
            "cluster": "my-cluster",
            "namespace": "default"
          },
          "window": {
            "start": "2024-01-01T00:00:00Z",
            "end": "2024-01-02T00:00:00Z"
          },
          "cpuCores": 0.5,
          "cpuCoreRequestAverage": 0.5,
          "cpuCoreUsageAverage": 0.25,
          "cpuCoreHours": 12.0,
          "cpuCost": 0.38,
          "cpuEfficiency": 0.5,
          "ramBytes": 536870912,
          "ramByteRequestAverage": 536870912,
          "ramByteUsageAverage": 268435456,
          "ramByteHours": 12884901888,
          "ramCost": 0.05,
          "ramEfficiency": 0.5,
          "totalCost": 0.43,
          "totalEfficiency": 0.5,
          "podCount": 3
        }
      },
      "window": {
        "start": "2024-01-01T00:00:00Z",
        "end": "2024-01-02T00:00:00Z"
      },
      "totalCost": 0.43
    }
  ]
}
```

#### 7.2 Allocation with Different Aggregations

```bash
# By cluster
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=cluster" \
  -H "Authorization: ApiKey ${API_KEY}"

# By node
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=node" \
  -H "Authorization: ApiKey ${API_KEY}"

# By pod
curl -X GET "http://localhost:8080/v1/allocation?window=24h&aggregate=pod" \
  -H "Authorization: ApiKey ${API_KEY}"

# By label (e.g., app label)
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=label:app" \
  -H "Authorization: ApiKey ${API_KEY}"

# Multi-aggregation (namespace + label)
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace,label:team" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.3 Window Formats

```bash
# Duration format
curl -X GET "http://localhost:8080/v1/allocation?window=24h" -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=7d" -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=2w" -H "Authorization: ApiKey ${API_KEY}"

# Named windows
curl -X GET "http://localhost:8080/v1/allocation?window=today" -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=yesterday" -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=lastweek" -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=thismonth" -H "Authorization: ApiKey ${API_KEY}"

# Date range (RFC3339 or date format)
curl -X GET "http://localhost:8080/v1/allocation?window=2024-01-01,2024-01-07" \
  -H "Authorization: ApiKey ${API_KEY}"
curl -X GET "http://localhost:8080/v1/allocation?window=2024-01-01T00:00:00Z,2024-01-07T23:59:59Z" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.4 Time-Series with Step

```bash
# Daily breakdown over 7 days
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace&step=1d&accumulate=false" \
  -H "Authorization: ApiKey ${API_KEY}"

# Hourly breakdown over 24 hours
curl -X GET "http://localhost:8080/v1/allocation?window=24h&aggregate=namespace&step=1h&accumulate=false" \
  -H "Authorization: ApiKey ${API_KEY}"

# Weekly breakdown over 30 days
curl -X GET "http://localhost:8080/v1/allocation?window=30d&step=1w&accumulate=false" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.5 Idle Cost Handling

```bash
# Include idle costs as separate allocation
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace&idle=true" \
  -H "Authorization: ApiKey ${API_KEY}"

# Distribute idle costs proportionally (weighted)
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace&idle=true&shareIdle=weighted" \
  -H "Authorization: ApiKey ${API_KEY}"

# Distribute idle costs evenly
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace&idle=true&shareIdle=true" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.6 Filtering

```bash
# Filter by namespace
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filter=namespace:production" \
  -H "Authorization: ApiKey ${API_KEY}"

# Filter by multiple namespaces
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filter=namespace:production,staging" \
  -H "Authorization: ApiKey ${API_KEY}"

# Filter by cluster
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=namespace&filter=cluster:prod-east" \
  -H "Authorization: ApiKey ${API_KEY}"

# Filter by label
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filter=label:app=nginx" \
  -H "Authorization: ApiKey ${API_KEY}"

# Multiple filters
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filter=namespace:production&filter=cluster:prod-east" \
  -H "Authorization: ApiKey ${API_KEY}"

# Legacy filter format (also supported)
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&filterNamespaces=production,staging&filterClusters=prod-east" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.7 Pagination

```bash
# Get first 10 results
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&limit=10" \
  -H "Authorization: ApiKey ${API_KEY}"

# Get results 11-20
curl -X GET "http://localhost:8080/v1/allocation?window=7d&aggregate=pod&offset=10&limit=10" \
  -H "Authorization: ApiKey ${API_KEY}"
```

#### 7.8 Allocation Summary

```bash
# GET /v1/allocation/summary
# Get condensed summary of allocations
curl -X GET "http://localhost:8080/v1/allocation/summary?window=7d&aggregate=namespace" \
  -H "Authorization: ApiKey ${API_KEY}"
```

**Expected Response:**
```json
{
  "code": 200,
  "status": "success",
  "data": {
    "items": [
      {
        "name": "default",
        "cpuCoreHours": 168.0,
        "cpuCost": 5.31,
        "ramByteHours": 90194313216,
        "ramCost": 0.35,
        "totalCost": 5.66,
        "totalEfficiency": 0.45
      }
    ],
    "totalCost": 5.66,
    "totalCPUCost": 5.31,
    "totalRAMCost": 0.35,
    "window": "7d",
    "aggregate": "namespace"
  }
}
```

#### 7.9 Allocation Topline

```bash
# GET /v1/allocation/summary/topline
# Get aggregated totals across all allocations
curl -X GET "http://localhost:8080/v1/allocation/summary/topline?window=7d" \
  -H "Authorization: ApiKey ${API_KEY}"
```

**Expected Response:**
```json
{
  "code": 200,
  "status": "success",
  "data": {
    "totalCost": 125.50,
    "totalCPUCost": 98.20,
    "totalRAMCost": 27.30,
    "totalIdleCost": 15.00,
    "totalCPUCoreHours": 3108.0,
    "totalRAMByteHours": 6442450944000,
    "avgEfficiency": 0.42,
    "allocationCount": 15,
    "window": "7d"
  }
}
```

#### 7.10 On-Demand Compute

```bash
# GET /v1/allocation/compute
# Same as /v1/allocation but for real-time (uncached) computation
curl -X GET "http://localhost:8080/v1/allocation/compute?window=1h&aggregate=namespace" \
  -H "Authorization: ApiKey ${API_KEY}"
```

---

## Complete Testing Workflow

Here's a complete workflow to test the entire API:

```bash
# 1. Check health
curl -X GET http://localhost:8080/v1/health

# 2. Create an API key
export API_KEY=$(curl -s -X POST http://localhost:8080/v1/admin/api_keys \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": 1}' | jq -r '"\(.key_id):\(.secret)"')

echo "API Key: ${API_KEY}"

# 3. Send metrics
curl -X POST http://localhost:8080/v1/ingest \
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
        "node_name": "test-node",
        "instance_type": "t3.medium",
        "cpu_capacity": 2000,
        "memory_capacity": 4294967296,
        "hourly_cost_usd": 0.0416
      }
    ]
  }'

# 4. Get recommendations (if any exist)
curl -X GET http://localhost:8080/v1/recommendations

# 5. Apply or dismiss a recommendation (replace 1 with actual ID)
# curl -X POST http://localhost:8080/v1/recommendations/1/apply
# curl -X POST http://localhost:8080/v1/recommendations/1/dismiss
```

## Testing with jq (Pretty JSON Output)

Install `jq` for better JSON formatting:

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq
```

Example with jq:
```bash
curl -s -X GET http://localhost:8080/v1/health | jq '.'
curl -s -X POST http://localhost:8080/v1/admin/api_keys \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": 1}' | jq '.'
```

## Error Testing

### Test Missing API Key
```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -d '{"cluster_name": "test"}' \
  -v
# Expected: 401 Unauthorized
```

### Test Invalid API Key
```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey invalid:key" \
  -d '{"cluster_name": "test"}' \
  -v
# Expected: 401 Unauthorized
```

### Test Invalid Payload
```bash
curl -X POST http://localhost:8080/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey ${API_KEY}" \
  -d '{"invalid": "payload"}' \
  -v
# Expected: 400 Bad Request
```

## Notes

- All timestamps are Unix timestamps (seconds since epoch)
- API keys use the format: `key_id:secret`
- The API key must be included in either `Authorization: ApiKey ...` or `X-Api-Key` header
- The ingest endpoint requires valid API key authentication
- Recommendations endpoints do not require authentication (in current implementation)
- CPU values are in millicores (1000 = 1 core)
- Memory values are in bytes

