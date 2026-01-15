package api

import (
	"context"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

type PodMetricData struct {
	PodName              string `json:"pod_name"`
	Namespace            string `json:"namespace"`
	NodeName             string `json:"node_name"`
	CPUUsageMillicores   int64  `json:"cpu_usage_millicores"`
	MemoryUsageBytes     int64  `json:"memory_usage_bytes"`
	CPURequestMillicores int64  `json:"cpu_request_millicores"`
	MemoryRequestBytes   int64  `json:"memory_request_bytes"`
	CPULimitMillicores   int64  `json:"cpu_limit_millicores,omitempty"`
	MemoryLimitBytes     int64  `json:"memory_limit_bytes,omitempty"`
}

type AgentMetricsPayload struct {
	ClusterName    string                       `json:"cluster_name"`
	Timestamp      int64                        `json:"timestamp"`
	PodMetrics     []PodMetricData              `json:"pod_metrics"`
	NamespaceCosts map[string]NamespaceCostData `json:"namespace_costs"`
	NodeMetrics    []NodeMetricData             `json:"node_metrics"`
}
type NamespaceCostData struct {
	Namespace          string  `json:"namespace"`
	PodCount           int     `json:"pod_count"`
	TotalCPUMillicores int64   `json:"total_cpu_millicores"`
	TotalMemoryBytes   int64   `json:"total_memory_bytes"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
}
type NodeMetricData struct {
	NodeName       string  `json:"node_name"`
	InstanceType   string  `json:"instance_type"`
	CPUCapacity    int64   `json:"cpu_capacity"`
	MemoryCapacity int64   `json:"memory_capacity"`
	HourlyCostUSD  float64 `json:"hourly_cost_usd"`
}

func (s *Server) makeIngestHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p AgentMetricsPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "details": err.Error()})
			return
		}
		ctx := context.Background()
		ts := time.Unix(p.Timestamp, 0)
		if p.Timestamp == 0 {
			ts = time.Now().UTC()
		}
		// tenant id from api key in context
		akI, exists := c.Get("api_key")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no api key context"})
			return
		}
		ak := akI.(*models.APIKey) // import models if needed
		tenantID := int64(ak.TenantID)
		// insert node metrics
		for _, nm := range p.NodeMetrics {
			_ = s.timescaleDB.InsertNodeMetric(ctx, ts, tenantID, p.ClusterName, nm.NodeName, nm.InstanceType, nm.CPUCapacity, nm.MemoryCapacity, nm.HourlyCostUSD)
		}
		// insert individual pod metrics
		for _, pm := range p.PodMetrics {
			_ = s.timescaleDB.InsertPodMetric(ctx, ts, tenantID, p.ClusterName, pm.Namespace, pm.PodName, pm.NodeName, pm.CPUUsageMillicores, pm.MemoryUsageBytes, pm.CPURequestMillicores, pm.MemoryRequestBytes, pm.CPULimitMillicores, pm.MemoryLimitBytes)
		}
		// for namespaceCost we write synthetic pod metrics aggregated by namespace (backward compatibility)
		for _, ns := range p.NamespaceCosts {
			_ = s.timescaleDB.InsertPodMetric(ctx, ts, tenantID, p.ClusterName, ns.Namespace, "__aggregate__", "", ns.TotalCPUMillicores, ns.TotalMemoryBytes, ns.TotalCPUMillicores, ns.TotalMemoryBytes, 0, 0)
		}
		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
	}
}
