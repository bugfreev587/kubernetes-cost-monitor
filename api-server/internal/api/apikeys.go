package api

import (
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

const MaxActiveAPIKeys = 3

type CreateKeyReq struct {
	TenantID  uint       `json:"tenant_id" binding:"required"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
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

		// Use the user's tenant ID for security (ignore tenant_id from request)
		tenantID := currentUser.TenantID

		// Check active API key count
		db := s.postgresDB.GetPostgresDB()
		var activeKeyCount int64
		if err := db.Model(&models.APIKey{}).Where("tenant_id = ? AND revoked = ?", tenantID, false).Count(&activeKeyCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check API key count"})
			return
		}

		if activeKeyCount >= MaxActiveAPIKeys {
			c.JSON(http.StatusConflict, gin.H{
				"error":           "max_keys_reached",
				"message":         "Maximum number of active API keys reached. Please revoke an existing key before creating a new one.",
				"max_keys":        MaxActiveAPIKeys,
				"active_keys":     activeKeyCount,
			})
			return
		}

		kid, secret, err := s.apiKeySvc.CreateKey(c.Request.Context(), tenantID, req.Scopes, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return secret only once
		c.JSON(http.StatusCreated, gin.H{
			"key_id":      kid,
			"secret":      secret,
			"active_keys": activeKeyCount + 1,
			"max_keys":    MaxActiveAPIKeys,
		})
	}
}
