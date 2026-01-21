package api

import (
	"fmt"
	"net/http"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

type UpdatePricingPlanReq struct {
	PricingPlan string `json:"pricing_plan" binding:"required"`
}

func (s *Server) updateTenantPricingPlanHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
			return
		}

		var req UpdatePricingPlanReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate pricing plan value
		validPlans := map[string]bool{
			"Starter":  true,
			"Premium":  true,
			"Business": true,
		}
		if !validPlans[req.PricingPlan] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing_plan. Must be one of: Starter, Premium, Business"})
			return
		}

		// Parse tenant_id as uint
		var tenantIDUint uint
		if _, err := fmt.Sscanf(tenantID, "%d", &tenantIDUint); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
			return
		}

		// Update tenant pricing plan using GORM directly
		db := s.postgresDB.GetPostgresDB()
		result := db.Model(&models.Tenant{}).
			Where("id = ?", tenantIDUint).
			Update("pricing_plan", req.PricingPlan)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"tenant_id":    tenantIDUint,
			"pricing_plan": req.PricingPlan,
			"message":      "pricing plan updated successfully",
		})
	}
}

func (s *Server) getTenantPricingPlanHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
			return
		}

		// Parse tenant_id as uint
		var tenantIDUint uint
		if _, err := fmt.Sscanf(tenantID, "%d", &tenantIDUint); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
			return
		}

		// Get tenant pricing plan using GORM directly
		var tenant models.Tenant
		db := s.postgresDB.GetPostgresDB()
		if err := db.First(&tenant, tenantIDUint).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"tenant_id":    tenant.ID,
			"pricing_plan": tenant.PricingPlan,
		})
	}
}

// listPricingPlansHandler returns all available pricing plans
func (s *Server) listPricingPlansHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var plans []models.PricingPlan
		db := s.postgresDB.GetPostgresDB()
		if err := db.Order("price_cents ASC").Find(&plans).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch plans"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"plans": plans,
		})
	}
}

// getTenantUsageHandler returns the current usage vs limits for a tenant
func (s *Server) getTenantUsageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
			return
		}

		// Parse tenant_id as uint
		var tenantIDUint uint
		if _, err := fmt.Sscanf(tenantID, "%d", &tenantIDUint); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
			return
		}

		// Get tenant usage from plan service
		usage, err := s.planSvc.GetTenantUsage(c.Request.Context(), tenantIDUint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, usage)
	}
}
