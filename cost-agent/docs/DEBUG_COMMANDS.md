# Debugging Commands for Cost-Agent Pod

## Exec into Pod

The container uses Alpine Linux (no bash), so use `sh`:

```bash
kubectl exec -it cost-agent-7cf6fd9648-c5drt -- sh
```

## Test Network Connectivity

### From inside the pod:

```bash
# Test basic connectivity to Railway
wget -O- --timeout=10 https://api-server-production-7a9d.up.railway.app/v1/health

# Or test with a simple HTTP request (if wget not available, use built-in tools)
```

### From outside (one-liner):

```bash
# Test connectivity from pod
kubectl exec cost-agent-7cf6fd9648-c5drt -- wget -O- --timeout=10 https://api-server-production-7a9d.up.railway.app/v1/health

# Check DNS resolution
kubectl exec cost-agent-7cf6fd9648-c5drt -- nslookup api-server-production-7a9d.up.railway.app

# Test with curl (if available)
kubectl exec cost-agent-7cf6fd9648-c5drt -- curl -v --max-time 10 https://api-server-production-7a9d.up.railway.app/v1/health
```

## Check Environment Variables

```bash
# Check all environment variables
kubectl exec cost-agent-7cf6fd9648-c5drt -- env | grep AGENT

# Or from inside pod with sh
kubectl exec -it cost-agent-7cf6fd9648-c5drt -- sh
# Then inside:
env | grep AGENT
```

## Check Logs

```bash
# Follow logs in real-time
kubectl logs -f cost-agent-7cf6fd9648-c5drt

# Get last 100 lines
kubectl logs --tail=100 cost-agent-7cf6fd9648-c5drt

# Get logs from all pods with label
kubectl logs -l app=cost-agent --tail=50
```

## Verify Configuration

```bash
# Check what the pod sees
kubectl describe pod cost-agent-7cf6fd9648-c5drt | grep -A 20 "Environment:"

# Check if secret exists and is mounted correctly
kubectl get secret cost-agent-api-key
kubectl describe secret cost-agent-api-key
```

## Test API Server from Pod

```bash
# Test health endpoint
kubectl exec cost-agent-7cf6fd9648-c5drt -- wget -O- --timeout=30 --header="Content-Type: application/json" https://api-server-production-7a9d.up.railway.app/v1/health

# Test with API key (replace with your key)
kubectl exec cost-agent-7cf6fd9648-c5drt -- sh -c 'wget -O- --timeout=30 --header="Content-Type: application/json" --header="Authorization: ApiKey YOUR_KEY_ID:YOUR_SECRET" https://api-server-production-7a9d.up.railway.app/v1/health'
```

## Check Timeout Setting

```bash
# Verify the timeout environment variable
kubectl exec cost-agent-7cf6fd9648-c5drt -- env | grep HTTP_TIMEOUT

# Should show: AGENT_HTTP_TIMEOUT=60 (or whatever value is set)
```

## Common Alpine Linux Commands

Since it's Alpine, these commands are available:
- `sh` (not bash)
- `wget` (usually available)
- `cat`, `echo`, `env`, `ls`, `ps`, etc.
- `nc` (netcat) for network testing
- `ping` (if available)

## Network Diagnostics

```bash
# Check if port 443 is reachable (HTTPS)
kubectl exec cost-agent-7cf6fd9648-c5drt -- nc -zv api-server-production-7a9d.up.railway.app 443

# Check DNS resolution
kubectl exec cost-agent-7cf6fd9648-c5drt -- nslookup api-server-production-7a9d.up.railway.app

# Or use getent if nslookup not available
kubectl exec cost-agent-7cf6fd9648-c5drt -- getent hosts api-server-production-7a9d.up.railway.app
```

