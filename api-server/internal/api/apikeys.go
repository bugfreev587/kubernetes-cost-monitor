package api

import (
	"log"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

type CreateKeyReq struct {
	ClusterName string     `json:"cluster_name" binding:"required"` // Each API key is for one cluster
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (s *Server) makeCreateAPIKeyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateKeyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current user to verify tenant access
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tenantID := currentUser.TenantID
		db := s.postgresDB.GetPostgresDB()

		// Get tenant's pricing plan limits
		planLimits, err := s.planSvc.GetTenantPlanLimits(c.Request.Context(), tenantID)
		if err != nil {
			log.Printf("Warning: Failed to get plan limits for tenant %d: %v", tenantID, err)
			// Default to Starter plan limits if we can't get the plan
			planLimits = &models.PricingPlan{
				Name:         "Starter",
				ClusterLimit: 1,
			}
		}

		// Check active API key count (each key = 1 cluster)
		var activeKeyCount int64
		if err := db.Model(&models.APIKey{}).Where("tenant_id = ? AND revoked = ?", tenantID, false).Count(&activeKeyCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check API key count"})
			return
		}

		// Check against plan's cluster limit (-1 = unlimited)
		if planLimits.ClusterLimit != -1 && int(activeKeyCount) >= planLimits.ClusterLimit {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "cluster_limit_reached",
				"message":       "You have reached the maximum number of clusters for your plan. Please upgrade your plan or revoke an existing API key.",
				"cluster_limit": planLimits.ClusterLimit,
				"active_keys":   activeKeyCount,
				"plan":          planLimits.Name,
			})
			return
		}

		// Check if cluster name is already in use by another active key
		var existingKey models.APIKey
		if err := db.Where("tenant_id = ? AND cluster_name = ? AND revoked = ?", tenantID, req.ClusterName, false).First(&existingKey).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":        "cluster_name_exists",
				"message":      "An API key for this cluster already exists. Revoke the existing key first or use a different cluster name.",
				"cluster_name": req.ClusterName,
				"existing_key": existingKey.KeyID,
			})
			return
		}

		kid, secret, err := s.apiKeySvc.CreateKey(c.Request.Context(), tenantID, req.ClusterName, req.Scopes, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return secret only once
		c.JSON(http.StatusCreated, gin.H{
			"key_id":        kid,
			"secret":        secret,
			"cluster_name":  req.ClusterName,
			"active_keys":   activeKeyCount + 1,
			"cluster_limit": planLimits.ClusterLimit,
			"plan":          planLimits.Name,
		})
	}
}
