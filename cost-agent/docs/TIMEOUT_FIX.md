# Fixing Timeout Issues

## Problem
The cost-agent is getting "context deadline exceeded" errors when sending data to Railway API server from OKE cluster.

## Root Cause
The HTTP timeout (30 seconds) may be too short for:
- Network latency from OKE to Railway
- SSL/TLS handshake time
- Large payload processing time

## Solution

### Option 1: Update Deployment (Quick Fix - No Rebuild)

Update the deployment to increase the timeout:

```bash
kubectl set env deployment/cost-agent AGENT_HTTP_TIMEOUT=60
```

Or edit the deployment YAML and change:
```yaml
- name: AGENT_HTTP_TIMEOUT
  value: "60"  # Increased from 30 to 60 seconds
```

Then apply:
```bash
kubectl apply -f cost-agent-deployment-complete.yaml
kubectl rollout restart deployment/cost-agent
```

### Option 2: Test Network Connectivity

First, verify the API server is reachable from OKE:

```bash
# Exec into the pod
kubectl exec -it <pod-name> -- sh

# Test connectivity
wget -O- --timeout=10 https://api-server-production-7a9d.up.railway.app/v1/health

# Or with curl (if available)
curl -v --max-time 10 https://api-server-production-7a9d.up.railway.app/v1/health
```

### Option 3: Check API Server Response Time

Test the API server response time from your local machine:

```bash
time curl -X POST https://api-server-production-7a9d.up.railway.app/v1/ingest \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey your_key_id:your_secret" \
  -d '{"cluster_name":"test","timestamp":0,"namespace_costs":{},"node_metrics":[]}'
```

If it takes more than 30 seconds, you definitely need to increase the timeout.

## Recommended Timeout Values

- **30 seconds**: Too short for cross-cloud (OKE → Railway)
- **60 seconds**: Recommended for cross-cloud deployments
- **90 seconds**: If payloads are large or network is consistently slow
- **120 seconds**: Maximum recommended (2 minutes)

## Additional Considerations

1. **Payload Size**: Large payloads take longer to send. Check the size of metrics being sent.

2. **Network Policies**: Ensure OKE cluster allows outbound HTTPS to Railway.

3. **API Server Performance**: Check Railway logs to see if the API server is slow to respond.

4. **Retry Logic**: The sender already has exponential backoff with 2-minute max elapsed time, so it will retry automatically.

## Monitoring

After increasing the timeout, monitor the logs:

```bash
kubectl logs -f -l app=cost-agent
```

Look for:
- Successful sends (no errors)
- Timeout errors (should decrease)
- Other connection errors

