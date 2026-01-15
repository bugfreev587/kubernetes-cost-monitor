# Viewing Cost-Agent Logs in OKE

## Quick Commands

### View Current Logs

```bash
# Get logs from the most recent pod
kubectl logs -l app=cost-agent

# Get logs from a specific pod (get pod name first)
kubectl get pods -l app=cost-agent
kubectl logs <pod-name>

# Follow logs in real-time (like tail -f)
kubectl logs -f -l app=cost-agent

# Get last 100 lines
kubectl logs --tail=100 -l app=cost-agent
```

### Get Pod Name First

```bash
# List all cost-agent pods
kubectl get pods -l app=cost-agent

# Example output:
# NAME                          READY   STATUS    RESTARTS   AGE
# cost-agent-7cf6fd9648-c5drt   1/1     Running   0          13h
```

Then use the pod name:
```bash
kubectl logs cost-agent-7cf6fd9648-c5drt
```

## Detailed Log Commands

### Follow Logs in Real-Time

```bash
# Follow logs from all pods with label app=cost-agent
kubectl logs -f -l app=cost-agent

# Follow logs from specific pod
kubectl logs -f <pod-name>
```

### Get Recent Logs

```bash
# Last 50 lines
kubectl logs --tail=50 -l app=cost-agent

# Last 200 lines
kubectl logs --tail=200 -l app=cost-agent

# Last 10 minutes of logs
kubectl logs --since=10m -l app=cost-agent

# Last 1 hour of logs
kubectl logs --since=1h -l app=cost-agent
```

### View Logs from Previous Container (After Restart)

If the pod has restarted and you want to see logs from the previous container:

```bash
# Get logs from previous container instance
kubectl logs <pod-name> --previous

# Or with label selector
kubectl logs -l app=cost-agent --previous
```

### Multiple Pods (if scaled)

If you have multiple replicas:

```bash
# View logs from all pods (interleaved)
kubectl logs -l app=cost-agent

# View logs from all pods with timestamps
kubectl logs -l app=cost-agent --timestamps

# View logs from specific pod only
kubectl logs <pod-name>
```

## Filtering Logs

### Search for Specific Text

```bash
# View logs and pipe to grep
kubectl logs -l app=cost-agent | grep "error"

# Case insensitive search
kubectl logs -l app=cost-agent | grep -i "error"

# Search for multiple terms
kubectl logs -l app=cost-agent | grep -E "error|failed|timeout"
```

### View Logs with Timestamps

```bash
# Add timestamps to log output
kubectl logs -l app=cost-agent --timestamps

# Follow with timestamps
kubectl logs -f -l app=cost-agent --timestamps
```

## Common Log Patterns

### Check for Errors

```bash
# View logs and filter for errors
kubectl logs -l app=cost-agent --tail=200 | grep -i error

# View recent errors
kubectl logs -l app=cost-agent --since=30m | grep -i error
```

### Check Collection Success

```bash
# Search for successful collections (or lack of errors)
kubectl logs -l app=cost-agent --tail=100 | grep -v "error"

# Check for collection messages
kubectl logs -l app=cost-agent | grep -i "collect"
```

### Check API Server Connection

```bash
# Look for connection or timeout errors
kubectl logs -l app=cost-agent | grep -iE "timeout|connection|failed|send"
```

## Export Logs to File

```bash
# Save logs to file
kubectl logs -l app=cost-agent > cost-agent-logs.txt

# Save with timestamps
kubectl logs -l app=cost-agent --timestamps > cost-agent-logs-timestamped.txt

# Save last 1000 lines
kubectl logs --tail=1000 -l app=cost-agent > cost-agent-logs-recent.txt
```

## One-Liner Quick Reference

```bash
# Most common: Follow logs in real-time
kubectl logs -f -l app=cost-agent

# Get pod name and view logs
POD_NAME=$(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}') && kubectl logs -f $POD_NAME

# View last 50 lines with timestamps
kubectl logs --tail=50 --timestamps -l app=cost-agent
```

## Troubleshooting with Logs

### Check if Agent is Running

```bash
# See if there are any pods
kubectl get pods -l app=cost-agent

# Check pod status
kubectl describe pod -l app=cost-agent
```

### Check for Startup Errors

```bash
# View logs from pod start
kubectl logs <pod-name> | head -50

# Or view previous container if it crashed
kubectl logs <pod-name> --previous
```

### Check Collection Frequency

```bash
# Count how many collection cycles happened
kubectl logs -l app=cost-agent | grep -c "collect send error\|initial collect"

# View collection timestamps
kubectl logs -l app=cost-agent --timestamps | grep -i collect
```

## OKE-Specific Notes

If you're using OKE (Oracle Kubernetes Engine), the commands are the same. Make sure you:

1. **Have kubectl configured** for your OKE cluster:
   ```bash
   # Verify cluster access
   kubectl cluster-info
   kubectl get nodes
   ```

2. **Are in the correct namespace** (if not using default):
   ```bash
   # List pods in default namespace
   kubectl get pods -n default -l app=cost-agent
   
   # View logs from specific namespace
   kubectl logs -n default -l app=cost-agent
   ```

3. **Have proper permissions** to view logs:
   ```bash
   # Test access
   kubectl auth can-i get pods
   kubectl auth can-i get logs
   ```

