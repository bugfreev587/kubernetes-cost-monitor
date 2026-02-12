package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GET /v1/recommendations
func (s *Server) getRecommendations(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	var recs []models.Recommendation
	if err := s.postgresDB.GetPostgresDB().Where("tenant_id = ?", tenantID).Find(&recs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transform to frontend-expected format
	type recResponse struct {
		ID                int     `json:"id"`
		PodName           string  `json:"pod_name"`
		Namespace         string  `json:"namespace"`
		Cluster           string  `json:"cluster"`
		Reason            string  `json:"reason"`
		CurrentCPU        int64   `json:"current_cpu"`
		CurrentMemory     int64   `json:"current_memory"`
		RecommendedCPU    int64   `json:"recommended_cpu"`
		RecommendedMemory int64   `json:"recommended_memory"`
		EstimatedSavings  float64 `json:"estimated_savings"`
		Status            string  `json:"status"`
		CreatedAt         string  `json:"created_at"`
	}

	var result []recResponse
	for _, r := range recs {
		resp := recResponse{
			ID:               int(r.ID),
			PodName:          r.PodName,
			Namespace:        r.Namespace,
			Cluster:          r.ClusterName,
			Reason:           r.Reason,
			EstimatedSavings: r.PotentialSavingsUSD,
			Status:           r.Status,
			CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if strings.EqualFold(r.ResourceType, "cpu") {
			resp.CurrentCPU = r.CurrentRequest
			resp.RecommendedCPU = r.RecommendedRequest
		} else if strings.EqualFold(r.ResourceType, "memory") {
			resp.CurrentMemory = r.CurrentRequest
			resp.RecommendedMemory = r.RecommendedRequest
		}
		result = append(result, resp)
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": result})
}

// POST /v1/recommendations/generate - Generate right-sizing recommendations
func (s *Server) generateRecommendations(c *gin.Context) {
	tid, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}
	tenantID := int64(tid)

	lookbackHours := 24 // Default
	if hoursStr := c.Query("lookback_hours"); hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 {
			lookbackHours = hours
		}
	}

	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	recSvc := services.NewRecommendationService(s.postgresDB.GetPostgresDB(), pool)
	
	if err := recSvc.GenerateRightSizingRecommendations(c.Request.Context(), tenantID, lookbackHours); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recommendations generated", "lookback_hours": lookbackHours})
}

// POST /recommendations/:id/dismiss
func (s *Server) dismissRecommendation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recommendation ID"})
		return
	}

	err = s.postgresDB.DismissRecommendation(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dismiss recommendation: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "dismissed"})
}

// POST /recommendations/:id/apply
func (s *Server) applyRecommendation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recommendation ID"})
		return
	}

	err = s.postgresDB.ApplyRecommendation(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply recommendation: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "applied"})
}
