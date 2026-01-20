# Priority 1 Improvements - Quick Deployment Guide

## TL;DR - 3 Steps to Deploy

```bash
# 1. Apply database migration (5 min)
psql -h <timescaledb-host> -U ts_user -d timeseries \
  -f api-server/migrations/002_add_pod_enhancements.sql

# 2. Deploy API server (2 min)
# Railway: git push → auto-deploys
# Manual: cd api-server && make build-api

# 3. Deploy cost-agent (3 min)
cd cost-agent
make build && make release && make deploy
```

**Total time**: ~10 minutes

## What You Get

✅ **Pod labels** → Cost allocation by team/environment
✅ **Pod phase** → Only bill Running pods (more accurate)
✅ **QoS class** → Identify over-provisioned pods
✅ **Container metrics** → Sidecar cost attribution

## Verify It's Working

```sql
-- Wait 10 minutes, then run:
SELECT
  pod_name,
  phase,
  qos_class,
  labels->>'team' AS team,
  jsonb_array_length(containers) AS containers
FROM pod_metrics
WHERE time > NOW() - INTERVAL '15 minutes'
  AND labels IS NOT NULL
LIMIT 5;
```

Should show populated labels, phase, qos_class!

## Example Use Cases

### Cost by Team
```sql
SELECT labels->>'team', SUM(cpu_millicores)
FROM pod_metrics
WHERE time > NOW() - INTERVAL '24 hours'
  AND phase = 'Running'
GROUP BY labels->>'team';
```

### Over-Provisioned Pods
```sql
SELECT pod_name, qos_class, cpu_request_millicores, AVG(cpu_millicores)
FROM pod_metrics
WHERE time > NOW() - INTERVAL '7 days'
  AND qos_class = 'Burstable'
GROUP BY pod_name, qos_class, cpu_request_millicores
HAVING AVG(cpu_millicores) < cpu_request_millicores * 0.5;
```

## Troubleshooting

**Problem**: New fields are NULL

**Solution**:
```bash
# Check agent logs
kubectl logs -l app=cost-agent | grep "collectLabels"
# Should show: collectLabels=true, collectContainers=true
```

---

**Full docs**: [PRIORITY1_IMPROVEMENTS_COMPLETE.md](./PRIORITY1_IMPROVEMENTS_COMPLETE.md)
