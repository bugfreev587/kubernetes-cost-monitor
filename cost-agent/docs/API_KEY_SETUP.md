# API Key Setup Verification

## Your API Key

Your API key format: `17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o`

Format: `key_id:secret`

## Verify Secret is Configured Correctly

### Option 1: Quick Check

```bash
# Check if secret exists
kubectl get secret cost-agent-api-key

# Decode and view the secret value
kubectl get secret cost-agent-api-key -o jsonpath='{.data.api-key}' | base64 -d
echo ""
```

Compare the output with your API key above. They should match exactly.

### Option 2: Check from Pod

```bash
# Get pod name
POD_NAME=$(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}')

# Check environment variable (will show as empty for security, but verify it exists)
kubectl exec $POD_NAME -- env | grep AGENT_API_KEY
```

If the variable exists, it should show `AGENT_API_KEY=<set to the key 'api-key' in secret 'cost-agent-api-key'>`

### Option 3: Verify Secret Matches Expected Value

```bash
# Check what's in the secret
kubectl get secret cost-agent-api-key -o yaml

# Decode the base64 value
kubectl get secret cost-agent-api-key -o jsonpath='{.data.api-key}' | base64 -d
```

Expected output: `17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o`

## If Secret is Missing or Incorrect

If the secret doesn't exist or has the wrong value, create/update it:

```bash
# Delete old secret (if exists)
kubectl delete secret cost-agent-api-key

# Create new secret with correct value
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o"

# Restart deployment to pick up new secret
kubectl rollout restart deployment/cost-agent

# Verify pod is running
kubectl get pods -l app=cost-agent
```

## Test API Key Works

Test the API key directly against the API server:

```bash
curl -X GET https://api-server-production-7a9d.up.railway.app/v1/health \
  -H "Authorization: ApiKey 17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o"
```

Or test the ingest endpoint:

```bash
curl -X POST https://api-server-production-7a9d.up.railway.app/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey 17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o" \
  -d '{
    "cluster_name": "test",
    "timestamp": 0,
    "namespace_costs": {},
    "node_metrics": []
  }'
```

Expected response: `202 Accepted` or `200 OK`

## Security Note

⚠️ **Important**: Your API key is sensitive information. Treat it like a password:
- Don't commit it to version control
- Don't share it publicly
- Rotate it if exposed
- Use Kubernetes secrets (not environment variables directly in YAML)

The secret is properly stored in Kubernetes and only mounted as an environment variable at runtime.

