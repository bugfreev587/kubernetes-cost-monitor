# Testing API Server Endpoints from Cost-Agent Pod

## Scripts

Two scripts are available:

1. **`test-api-endpoints.sh`** - Comprehensive test script with detailed output
2. **`test-api-simple.sh`** - Simple version with basic pass/fail output

Both scripts are designed to run in Alpine Linux (sh shell, not bash).

## Usage

### Option 1: Copy Script to Pod and Run

```bash
# Get pod name
POD_NAME=$(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}')

# Copy script to pod
kubectl cp scripts/test-api-simple.sh default/$POD_NAME:/tmp/test-api.sh

# Execute script in pod
kubectl exec $POD_NAME -- sh /tmp/test-api.sh
```

### Option 2: Run Script Directly (One-liner)

```bash
# Get pod name
POD_NAME=$(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}')

# Copy and run in one command
cat scripts/test-api-simple.sh | kubectl exec -i $POD_NAME -- sh
```

### Option 3: Run Script with Custom API Key

The scripts use environment variables. You can override them:

```bash
# Copy script
kubectl cp scripts/test-api.sh default/$POD_NAME:/tmp/test-api.sh

# Run with custom API key
kubectl exec $POD_NAME -- sh -c "AGENT_API_KEY='your_key_id:your_secret' sh /tmp/test-api.sh"
```

### Option 4: Exec into Pod and Run Manually

```bash
# Exec into pod
kubectl exec -it $POD_NAME -- sh

# Inside pod, copy script content or run commands directly
# The script reads AGENT_API_KEY from environment variable
```

## Script Features

### scripts/test-api-simple.sh

- Simple pass/fail output
- Uses `wget` (common in Alpine)
- Tests:
  - GET /v1/health
  - GET /v1/recommendations
  - POST /v1/ingest (with API key)

### scripts/test-api-endpoints.sh

- Detailed output with colors
- Supports both `wget` and `curl`
- Tests all endpoints including:
  - GET /v1/health
  - GET /v1/recommendations
  - POST /v1/ingest (with API key)
  - POST /v1/ingest (with invalid key - should fail)
  - POST /v1/ingest (without key - should fail)
- Provides test summary

## Configuration

Scripts use these environment variables (already set in pod):

- `AGENT_SERVER_URL` - API server URL (defaults to Railway URL)
- `AGENT_API_KEY` - API key for authentication

The scripts automatically extract the base URL from `AGENT_SERVER_URL` if it includes `/v1/ingest`.

## Expected Results

### Successful Tests

- **Health Check**: Should return 200 OK with health status
- **Get Recommendations**: Should return 200 OK (may be empty array)
- **Ingest Metrics**: Should return 202 Accepted or 200 OK

### Expected Failures (Testing Error Handling)

- **Invalid API Key**: Should return 401 Unauthorized
- **Missing API Key**: Should return 401 Unauthorized

## Manual Testing (Alternative)

If scripts don't work, you can test endpoints manually:

```bash
# Exec into pod
kubectl exec -it $POD_NAME -- sh

# Test health (no auth required)
wget -O- --timeout=30 https://api-server-production-7a9d.up.railway.app/v1/health

# Test ingest with API key
echo '{"cluster_name":"test","timestamp":0,"namespace_costs":{},"node_metrics":[]}' | \
  wget -O- --timeout=60 \
    --header="Content-Type: application/json" \
    --header="Authorization: ApiKey $AGENT_API_KEY" \
    --method=POST \
    --body-data=- \
    https://api-server-production-7a9d.up.railway.app/v1/ingest
```

## Troubleshooting

### Script not found

Make sure you copied the script to the pod:
```bash
kubectl cp scripts/test-api.sh default/$POD_NAME:/tmp/test-api.sh
```

### Permission denied

Scripts are executable, but if needed:
```bash
kubectl exec $POD_NAME -- chmod +x /tmp/test-api.sh
```

### wget not found

Alpine should have wget. If not, the comprehensive script will try curl as fallback.

### Connection timeout

If you see timeout errors, check:
1. API server is accessible from OKE cluster
2. Network policies allow outbound HTTPS
3. Firewall rules
4. DNS resolution

### 401 Unauthorized

If you get 401 errors:
1. Verify API key is correct
2. Check secret is mounted correctly: `kubectl exec $POD_NAME -- env | grep AGENT_API_KEY`
3. Ensure API key format is `key_id:secret`

## Quick Test Command

```bash
# One-liner to test health endpoint
kubectl exec $(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}') -- \
  wget -O- --timeout=30 https://api-server-production-7a9d.up.railway.app/v1/health
```

