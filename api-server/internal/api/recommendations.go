package api

import (
	"net/http"
	"strconv"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GET /recommendations
func (s *Server) getRecommendations(c *gin.Context) {
	akI, exists := c.Get("api_key")
	var tenantID uint = 0
	if exists {
		ak := akI.(*models.APIKey)
		tenantID = ak.TenantID
	}

	var recs []models.Recommendation
	query := s.postgresDB.GetPostgresDB()
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&recs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recs)
}

// POST /v1/recommendations/generate - Generate right-sizing recommendations
func (s *Server) generateRecommendations(c *gin.Context) {
	akI, exists := c.Get("api_key")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no api key context"})
		return
	}
	ak := akI.(*models.APIKey)
	tenantID := int64(ak.TenantID)

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
