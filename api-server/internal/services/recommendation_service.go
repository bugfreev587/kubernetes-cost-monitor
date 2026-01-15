package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type RecommendationService struct {
	postgresDB *gorm.DB
	timescaleDB *pgxpool.Pool
}

func NewRecommendationService(postgresDB *gorm.DB, timescaleDB *pgxpool.Pool) *RecommendationService {
	return &RecommendationService{
		postgresDB:  postgresDB,
		timescaleDB: timescaleDB,
	}
}

// GenerateRightSizingRecommendations analyzes pod metrics and generates right-sizing recommendations
func (s *RecommendationService) GenerateRightSizingRecommendations(ctx context.Context, tenantID int64, lookbackHours int) error {
	if lookbackHours <= 0 {
		lookbackHours = 24 // Default: 24 hours
	}

	startTime := time.Now().Add(-time.Duration(lookbackHours) * time.Hour)
	endTime := time.Now()

	// Query pods with low utilization vs requests
	query := `
		SELECT 
			cluster_name,
			namespace,
			pod_name,
			AVG(cpu_millicores) as avg_cpu_usage,
			AVG(cpu_request_millicores) as avg_cpu_request,
			AVG(memory_bytes) as avg_memory_usage,
			AVG(memory_request_bytes) as avg_memory_request,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY cpu_millicores) as p95_cpu_usage,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY memory_bytes) as p95_memory_usage
		FROM pod_metrics
		WHERE tenant_id = $1 
			AND time >= $2 
			AND time <= $3
			AND pod_name != '__aggregate__'
			AND cpu_request_millicores > 0
			AND memory_request_bytes > 0
		GROUP BY cluster_name, namespace, pod_name
		HAVING 
			(AVG(cpu_millicores)::numeric / NULLIF(AVG(cpu_request_millicores), 0)::numeric) < 0.5
			OR (AVG(memory_bytes)::numeric / NULLIF(AVG(memory_request_bytes), 0)::numeric) < 0.5
		ORDER BY (AVG(cpu_millicores)::numeric / NULLIF(AVG(cpu_request_millicores), 0)::numeric) ASC
		LIMIT 50
	`

	rows, err := s.timescaleDB.Query(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			clusterName      string
			namespace        string
			podName          string
			avgCPUUsage      float64
			avgCPURequest    float64
			avgMemoryUsage   float64
			avgMemoryRequest float64
			p95CPUUsage      float64
			p95MemoryUsage   float64
		)

		if err := rows.Scan(
			&clusterName,
			&namespace,
			&podName,
			&avgCPUUsage,
			&avgCPURequest,
			&avgMemoryUsage,
			&avgMemoryRequest,
			&p95CPUUsage,
			&p95MemoryUsage,
		); err != nil {
			continue
		}

		// Calculate recommended requests (use P95 + 20% buffer)
		recommendedCPU := int64(p95CPUUsage * 1.2)
		recommendedMemory := int64(p95MemoryUsage * 1.2)

		// Ensure recommendations are at least 10% of current requests (don't recommend too small)
		if recommendedCPU < int64(avgCPURequest*0.1) {
			recommendedCPU = int64(avgCPURequest * 0.1)
		}
		if recommendedMemory < int64(avgMemoryRequest*0.1) {
			recommendedMemory = int64(avgMemoryRequest * 0.1)
		}

		// Calculate potential savings (simplified: based on resource reduction)
		// In production, this would use actual node costs
		cpuReduction := avgCPURequest - float64(recommendedCPU)
		memoryReduction := avgMemoryRequest - float64(recommendedMemory)
		
		// Simplified cost calculation: assume $0.10 per CPU core-hour and $0.01 per GB-hour
		potentialSavings := (cpuReduction/1000.0)*0.10*float64(lookbackHours) + (memoryReduction/1e9)*0.01*float64(lookbackHours)

		// Calculate confidence based on consistency of usage
		utilizationRatio := avgCPUUsage / avgCPURequest
		if utilizationRatio > 0.8 {
			utilizationRatio = avgMemoryUsage / avgMemoryRequest
		}
		confidence := 1.0 - utilizationRatio // Lower utilization = higher confidence in recommendation

		// Generate recommendation reason
		reason := fmt.Sprintf(
			"Pod uses %.1f%% of requested CPU and %.1f%% of requested memory. Recommended: %d mCPU, %d MB (P95 + 20%% buffer)",
			(avgCPUUsage/avgCPURequest)*100,
			(avgMemoryUsage/avgMemoryRequest)*100,
			recommendedCPU,
			recommendedMemory/1e6,
		)

		// Check if recommendation already exists
		var existingRec models.Recommendation
		err := s.postgresDB.Where("tenant_id = ? AND cluster_name = ? AND namespace = ? AND pod_name = ? AND status = ?",
			tenantID, clusterName, namespace, podName, "open").First(&existingRec).Error

		if err == gorm.ErrRecordNotFound {
			// Create new recommendation
			rec := models.Recommendation{
				TenantID:            uint(tenantID),
				ClusterName:         clusterName,
				Namespace:           namespace,
				PodName:             podName,
				ResourceType:        "requests",
				CurrentRequest:       int64(avgCPURequest),
				RecommendedRequest:   recommendedCPU,
				PotentialSavingsUSD: potentialSavings,
				Confidence:          confidence,
				Reason:              reason,
				Status:              "open",
			}
			s.postgresDB.Create(&rec)
		} else if err == nil {
			// Update existing recommendation
			existingRec.RecommendedRequest = recommendedCPU
			existingRec.PotentialSavingsUSD = potentialSavings
			existingRec.Confidence = confidence
			existingRec.Reason = reason
			s.postgresDB.Save(&existingRec)
		}
	}

	return rows.Err()
}

