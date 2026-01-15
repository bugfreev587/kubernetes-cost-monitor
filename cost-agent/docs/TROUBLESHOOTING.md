# Troubleshooting Cost-Agent Crash Loop

## Common Issues

### 1. Missing Environment Variables

The agent requires at minimum:
- `AGENT_API_KEY` (required)
- `AGENT_SERVER_URL` (optional, has default but should be set for Kubernetes)
- `AGENT_CLUSTER_NAME` (optional, has default)

### 2. Config File Not Found

If `AGENT_CONFIG_FILE` is not set, the agent tries to load `./conf/cost-agent-dev.yaml` which doesn't exist in the container. Either:
- Set environment variables instead (recommended)
- Or don't set `AGENT_CONFIG_FILE` and let it fail gracefully (it will use env vars)

### 3. Check Pod Logs

To see why the container is crashing:

```bash
# Get the pod name
kubectl get pods -l app=cost-agent

# Check logs
kubectl logs <pod-name>

# Or follow logs in real-time
kubectl logs -f <pod-name>

# Check previous container logs if it restarted
kubectl logs <pod-name> --previous
```

### 4. Common Error Messages

- **"API key not provided"**: Missing `AGENT_API_KEY` environment variable
- **"load config: failed to read config file"**: Config file path doesn't exist (set env vars instead)
- **"collector init: ..."**: Kubernetes API access issues (check RBAC/service account)
- **"send failed: ..."**: Cannot reach API server (check `AGENT_SERVER_URL`)

## Quick Fix

1. **Create the API key secret** (if not already created):
   ```bash
   kubectl create secret generic cost-agent-api-key \
     --from-literal=api-key=your_key_id:your_secret
   ```

2. **Update the deployment** to include required environment variables (see deployment example below)

