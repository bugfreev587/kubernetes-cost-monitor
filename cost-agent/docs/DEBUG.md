# Debugging CrashLoopBackOff

The pod is crashing. Check the logs to see the exact error:

```bash
# Get current pod logs
kubectl logs cost-agent-5b8d84bf78-rjj2h

# Or get logs from the most recent pod
kubectl logs -l app=cost-agent --tail=50

# If the container has restarted multiple times, check previous logs
kubectl logs cost-agent-5b8d84bf78-rjj2h --previous
```

## Common Issues Based on Logs

### 1. "API key not provided"
**Solution:** The secret doesn't exist or has wrong key name
```bash
# Check if secret exists
kubectl get secret cost-agent-api-key

# Check secret contents (base64 encoded)
kubectl get secret cost-agent-api-key -o yaml

# Verify the key name is 'api-key'
kubectl describe secret cost-agent-api-key
```

### 2. "load config: failed to read config file"
**Solution:** The code tries to load a config file. This should be fixed in the latest code, but if you're using an older image, you may need to rebuild.

### 3. "collector init: ..."
**Solution:** Kubernetes API access issue - check RBAC permissions
```bash
# Verify service account exists
kubectl get serviceaccount cost-agent

# Verify cluster role and binding
kubectl get clusterrole cost-agent
kubectl get clusterrolebinding cost-agent

# Test permissions
kubectl auth can-i list pods --as=system:serviceaccount:default:cost-agent
kubectl auth can-i list nodes --as=system:serviceaccount:default:cost-agent
```

### 4. "send failed: ..." or connection errors
**Solution:** Cannot reach API server
- Check `AGENT_SERVER_URL` is correct
- Verify the API server is accessible from the cluster
- Check network policies if any

### 5. Image pull errors
**Solution:** If using private image, need image pull secret
```bash
# Check if image pull secret exists
kubectl get secret ghcr-secret

# If missing and image is private, create it
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=bugfreev587 \
  --docker-password=ghp_your_token \
  --docker-email=your-email@example.com
```

## Verify Environment Variables

Check what environment variables the pod is actually using:

```bash
# Get environment variables from the running pod (before it crashes)
kubectl exec cost-agent-5b8d84bf78-rjj2h -- env | grep AGENT

# Or check from the deployment
kubectl get deployment cost-agent -o yaml | grep -A 20 env:
```

## Check Secret Value

Verify the secret value is correct format (should be `key_id:secret`):

```bash
# Decode and view the secret value
kubectl get secret cost-agent-api-key -o jsonpath='{.data.api-key}' | base64 -d
echo ""
```

## Quick Checklist

- [ ] Secret `cost-agent-api-key` exists
- [ ] Secret key name is `api-key` (not `AGENT_API_KEY`)
- [ ] Secret value format is `key_id:secret`
- [ ] ServiceAccount `cost-agent` exists
- [ ] ClusterRole and ClusterRoleBinding exist
- [ ] Image is accessible (public or imagePullSecret exists)
- [ ] API server URL is correct and reachable

