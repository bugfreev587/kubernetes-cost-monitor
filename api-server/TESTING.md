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

