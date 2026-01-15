# Deployment Guide for Cost-Agent

## Deployment Order

**Important:** You must create the Kubernetes secret **BEFORE** deploying the deployment, otherwise the pods will fail to start.

## Step-by-Step Deployment

### Step 1: Get/Create an API Key

If you don't have an API key yet, create one using the API server:

```bash
# Replace with your actual API server URL
curl -X POST https://api-server-production-7a9d.up.railway.app/v1/admin/api_keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": 1,
    "scopes": [],
    "expires_at": null
  }'
```

The response will look like:
```json
{
  "key_id": "abc123...",
  "secret": "xyz789...",
  ...
}
```

**Important:** Save both the `key_id` and `secret` - you'll need them in the format: `key_id:secret`

### Step 2: Create the Kubernetes Secret (REQUIRED FIRST)

Create the secret with your API key:

```bash
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="your_key_id:your_secret"
```

**Example:**
```bash
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="abc123-def456:xyz789-secret123"
```

**Verify the secret was created:**
```bash
kubectl get secret cost-agent-api-key
```

**Note:** If the secret already exists and you want to update it:
```bash
kubectl delete secret cost-agent-api-key
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="your_new_key_id:your_new_secret"
```

### Step 3: (Optional) Create Service Account and RBAC

If you're deploying the complete manifest, it includes the ServiceAccount and RBAC. If deploying separately, create them first:

```bash
# Apply just the RBAC resources
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cost-agent
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cost-agent
rules:
- apiGroups: [""]
  resources: ["pods", "nodes"]
  verbs: ["list", "get"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cost-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cost-agent
subjects:
- kind: ServiceAccount
  name: cost-agent
  namespace: default
EOF
```

### Step 4: Deploy the Cost-Agent

Now you can safely deploy the agent:

```bash
kubectl apply -f cost-agent-deployment-complete.yaml
```

**Or if using the complete manifest with RBAC:**
```bash
# This includes ServiceAccount, RBAC, and Deployment
kubectl apply -f cost-agent-deployment-complete.yaml
```

### Step 5: Verify Deployment

Check that the pod is running:

```bash
# Check pod status
kubectl get pods -l app=cost-agent

# Check pod logs
kubectl logs -f -l app=cost-agent

# Check deployment status
kubectl get deployment cost-agent
```

### Step 6: Troubleshooting

If the pod is in `CrashLoopBackOff` or `Error` state:

1. **Check logs:**
   ```bash
   kubectl logs -l app=cost-agent --tail=50
   ```

2. **Common issues:**
   - **"API key not provided"**: Secret doesn't exist or has wrong key name
   - **"load config: failed to read config file"**: This shouldn't happen with env vars
   - **"send failed: ..."**: Cannot reach API server (check `AGENT_SERVER_URL`)
   - **"collector init: ..."**: RBAC permissions issue (check ServiceAccount)

3. **Verify secret exists:**
   ```bash
   kubectl get secret cost-agent-api-key -o yaml
   ```

4. **Check environment variables in pod:**
   ```bash
   kubectl exec -it <pod-name> -- env | grep AGENT
   ```

## Complete Quick Start

```bash
# 1. Create API key secret (REQUIRED FIRST!)
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="your_key_id:your_secret"

# 2. Deploy everything (includes RBAC)
kubectl apply -f cost-agent-deployment-complete.yaml

# 3. Check status
kubectl get pods -l app=cost-agent
kubectl logs -f -l app=cost-agent
```

## Updating the API Key

To update the API key without redeploying:

```bash
# Delete old secret
kubectl delete secret cost-agent-api-key

# Create new secret
kubectl create secret generic cost-agent-api-key \
  --from-literal=api-key="new_key_id:new_secret"

# Restart pods to pick up new secret
kubectl rollout restart deployment cost-agent
```

## Image Pull Secret (GHCR)

If your image is private on GitHub Container Registry, you also need to create an image pull secret:

```bash
# Create GitHub token secret (if not already created)
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=bugfreev587 \
  --docker-password=ghp_your_github_token \
  --docker-email=your-email@example.com
```

If your package is public, you don't need this secret.

