package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CostService struct {
	pool *pgxpool.Pool
}

func NewCostService(pool *pgxpool.Pool) *CostService {
	return &CostService{pool: pool}
}

// CostByNamespace returns cost breakdown by namespace for a tenant
func (s *CostService) CostByNamespace(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]NamespaceCost, error) {
	query := `
		SELECT 
			namespace,
			SUM(cpu_request_millicores) as total_cpu_request,
			SUM(memory_request_bytes) as total_memory_request,
			AVG(cpu_millicores) as avg_cpu_usage,
			AVG(memory_bytes) as avg_memory_usage,
			COUNT(DISTINCT pod_name) as pod_count
		FROM pod_metrics
		WHERE tenant_id = $1 
			AND time >= $2 
			AND time <= $3
			AND pod_name != '__aggregate__'
		GROUP BY namespace
		ORDER BY total_cpu_request DESC
	`

	rows, err := s.pool.Query(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []NamespaceCost
	for rows.Next() {
		var nc NamespaceCost
		if err := rows.Scan(
			&nc.Namespace,
			&nc.TotalCPURequest,
			&nc.TotalMemoryRequest,
			&nc.AvgCPUUsage,
			&nc.AvgMemoryUsage,
			&nc.PodCount,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		// Calculate cost based on node hourly costs
		nc.EstimatedCostUSD = s.calculateNamespaceCost(ctx, tenantID, nc.Namespace, startTime, endTime)
		results = append(results, nc)
	}

	return results, rows.Err()
}

// CostByCluster returns cost breakdown by cluster for a tenant
func (s *CostService) CostByCluster(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]ClusterCost, error) {
	query := `
		SELECT 
			cluster_name,
			SUM(cpu_request_millicores) as total_cpu_request,
			SUM(memory_request_bytes) as total_memory_request,
			AVG(cpu_millicores) as avg_cpu_usage,
			AVG(memory_bytes) as avg_memory_usage,
			COUNT(DISTINCT pod_name) as pod_count,
			COUNT(DISTINCT namespace) as namespace_count
		FROM pod_metrics
		WHERE tenant_id = $1 
			AND time >= $2 
			AND time <= $3
			AND pod_name != '__aggregate__'
		GROUP BY cluster_name
		ORDER BY total_cpu_request DESC
	`

	rows, err := s.pool.Query(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []ClusterCost
	for rows.Next() {
		var cc ClusterCost
		if err := rows.Scan(
			&cc.ClusterName,
			&cc.TotalCPURequest,
			&cc.TotalMemoryRequest,
			&cc.AvgCPUUsage,
			&cc.AvgMemoryUsage,
			&cc.PodCount,
			&cc.NamespaceCount,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		// Calculate cost based on node hourly costs
		cc.EstimatedCostUSD = s.calculateClusterCost(ctx, tenantID, cc.ClusterName, startTime, endTime)
		results = append(results, cc)
	}

	return results, rows.Err()
}

// UtilizationVsRequests returns resource utilization vs requests for pods
func (s *CostService) UtilizationVsRequests(ctx context.Context, tenantID int64, startTime, endTime time.Time, namespace, cluster string) ([]UtilizationMetric, error) {
	query := `
		SELECT 
			cluster_name,
			namespace,
			pod_name,
			AVG(cpu_millicores) as avg_cpu_usage,
			AVG(cpu_request_millicores) as avg_cpu_request,
			AVG(memory_bytes) as avg_memory_usage,
			AVG(memory_request_bytes) as avg_memory_request,
			CASE 
				WHEN AVG(cpu_request_millicores) > 0 
				THEN (AVG(cpu_millicores)::numeric / AVG(cpu_request_millicores)::numeric) * 100
				ELSE 0
			END as cpu_utilization_percent,
			CASE 
				WHEN AVG(memory_request_bytes) > 0 
				THEN (AVG(memory_bytes)::numeric / AVG(memory_request_bytes)::numeric) * 100
				ELSE 0
			END as memory_utilization_percent
		FROM pod_metrics
		WHERE tenant_id = $1 
			AND time >= $2 
			AND time <= $3
			AND pod_name != '__aggregate__'
	`

	args := []interface{}{tenantID, startTime, endTime}
	argIdx := 4

	if namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, namespace)
		argIdx++
	}

	if cluster != "" {
		query += fmt.Sprintf(" AND cluster_name = $%d", argIdx)
		args = append(args, cluster)
		argIdx++
	}

	query += `
		GROUP BY cluster_name, namespace, pod_name
		ORDER BY cpu_utilization_percent DESC
		LIMIT 100
	`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []UtilizationMetric
	for rows.Next() {
		var um UtilizationMetric
		if err := rows.Scan(
			&um.ClusterName,
			&um.Namespace,
			&um.PodName,
			&um.AvgCPUUsage,
			&um.AvgCPURequest,
			&um.AvgMemoryUsage,
			&um.AvgMemoryRequest,
			&um.CPUUtilizationPercent,
			&um.MemoryUtilizationPercent,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, um)
	}

	return results, rows.Err()
}

// CostTrends returns daily or weekly cost trends
func (s *CostService) CostTrends(ctx context.Context, tenantID int64, startTime, endTime time.Time, interval string) ([]CostTrend, error) {
	var timeBucket string
	switch interval {
	case "daily", "day":
		timeBucket = "1 day"
	case "weekly", "week":
		timeBucket = "1 week"
	case "hourly", "hour":
		timeBucket = "1 hour"
	default:
		timeBucket = "1 day"
	}

	query := fmt.Sprintf(`
		SELECT 
			time_bucket('%s', time) as bucket_time,
			SUM(cpu_request_millicores) as total_cpu_request,
			SUM(memory_request_bytes) as total_memory_request,
			AVG(cpu_millicores) as avg_cpu_usage,
			AVG(memory_bytes) as avg_memory_usage,
			COUNT(DISTINCT pod_name) as pod_count
		FROM pod_metrics
		WHERE tenant_id = $1 
			AND time >= $2 
			AND time <= $3
			AND pod_name != '__aggregate__'
		GROUP BY bucket_time
		ORDER BY bucket_time ASC
	`, timeBucket)

	rows, err := s.pool.Query(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []CostTrend
	for rows.Next() {
		var ct CostTrend
		if err := rows.Scan(
			&ct.Time,
			&ct.TotalCPURequest,
			&ct.TotalMemoryRequest,
			&ct.AvgCPUUsage,
			&ct.AvgMemoryUsage,
			&ct.PodCount,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		// Calculate estimated cost for this time bucket
		ct.EstimatedCostUSD = s.calculateTimeBucketCost(ctx, tenantID, ct.Time, timeBucket)
		results = append(results, ct)
	}

	return results, rows.Err()
}

// Helper function to calculate namespace cost based on node costs
func (s *CostService) calculateNamespaceCost(ctx context.Context, tenantID int64, namespace string, startTime, endTime time.Time) float64 {
	// Get average node hourly cost for this namespace's cluster
	query := `
		SELECT AVG(hourly_cost_usd) 
		FROM node_metrics nm
		INNER JOIN pod_metrics pm ON pm.cluster_name = nm.cluster_name AND pm.tenant_id = nm.tenant_id
		WHERE pm.tenant_id = $1 
			AND pm.namespace = $2
			AND pm.time >= $3 
			AND pm.time <= $4
			AND pm.pod_name != '__aggregate__'
		LIMIT 1
	`
	var avgHourlyCost float64
	s.pool.QueryRow(ctx, query, tenantID, namespace, startTime, endTime).Scan(&avgHourlyCost)

	// Calculate cost: (total CPU requests / node CPU capacity) * hourly cost * hours
	duration := endTime.Sub(startTime).Hours()
	if duration <= 0 {
		duration = 1
	}

	// Simplified cost calculation: assume 1 CPU millicore = proportional share of node cost
	// This is a simplified model - in production, you'd use more sophisticated allocation
	return avgHourlyCost * duration * 0.1 // Simplified multiplier
}

func (s *CostService) calculateClusterCost(ctx context.Context, tenantID int64, cluster string, startTime, endTime time.Time) float64 {
	query := `
		SELECT AVG(hourly_cost_usd) 
		FROM node_metrics
		WHERE tenant_id = $1 
			AND cluster_name = $2
			AND time >= $3 
			AND time <= $4
	`
	var avgHourlyCost float64
	s.pool.QueryRow(ctx, query, tenantID, cluster, startTime, endTime).Scan(&avgHourlyCost)

	duration := endTime.Sub(startTime).Hours()
	if duration <= 0 {
		duration = 1
	}

	return avgHourlyCost * duration
}

func (s *CostService) calculateTimeBucketCost(ctx context.Context, tenantID int64, bucketTime time.Time, timeBucket string) float64 {
	// Simplified cost calculation for time bucket
	query := `
		SELECT AVG(hourly_cost_usd) 
		FROM node_metrics
		WHERE tenant_id = $1 
			AND time >= $2 - interval '1 hour'
			AND time <= $2 + interval '1 hour'
	`
	var avgHourlyCost float64
	s.pool.QueryRow(ctx, query, tenantID, bucketTime).Scan(&avgHourlyCost)

	var hours float64
	switch timeBucket {
	case "1 hour":
		hours = 1
	case "1 day":
		hours = 24
	case "1 week":
		hours = 168
	default:
		hours = 24
	}

	return avgHourlyCost * hours
}

// Types for cost queries
type NamespaceCost struct {
	Namespace          string  `json:"namespace"`
	TotalCPURequest    int64   `json:"total_cpu_request_millicores"`
	TotalMemoryRequest int64   `json:"total_memory_request_bytes"`
	AvgCPUUsage        float64 `json:"avg_cpu_usage_millicores"`
	AvgMemoryUsage     float64 `json:"avg_memory_usage_bytes"`
	PodCount           int     `json:"pod_count"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
}

type ClusterCost struct {
	ClusterName        string  `json:"cluster_name"`
	TotalCPURequest    int64   `json:"total_cpu_request_millicores"`
	TotalMemoryRequest int64   `json:"total_memory_request_bytes"`
	AvgCPUUsage        float64 `json:"avg_cpu_usage_millicores"`
	AvgMemoryUsage     float64 `json:"avg_memory_usage_bytes"`
	PodCount           int     `json:"pod_count"`
	NamespaceCount     int     `json:"namespace_count"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
}

type UtilizationMetric struct {
	ClusterName              string  `json:"cluster_name"`
	Namespace                string  `json:"namespace"`
	PodName                  string  `json:"pod_name"`
	AvgCPUUsage              float64 `json:"avg_cpu_usage_millicores"`
	AvgCPURequest            float64 `json:"avg_cpu_request_millicores"`
	AvgMemoryUsage           float64 `json:"avg_memory_usage_bytes"`
	AvgMemoryRequest         float64 `json:"avg_memory_request_bytes"`
	CPUUtilizationPercent    float64 `json:"cpu_utilization_percent"`
	MemoryUtilizationPercent float64 `json:"memory_utilization_percent"`
}

type CostTrend struct {
	Time               time.Time `json:"time"`
	TotalCPURequest    int64     `json:"total_cpu_request_millicores"`
	TotalMemoryRequest int64     `json:"total_memory_request_bytes"`
	AvgCPUUsage        float64   `json:"avg_cpu_usage_millicores"`
	AvgMemoryUsage     float64   `json:"avg_memory_usage_bytes"`
	PodCount           int       `json:"pod_count"`
	EstimatedCostUSD   float64   `json:"estimated_cost_usd"`
}
