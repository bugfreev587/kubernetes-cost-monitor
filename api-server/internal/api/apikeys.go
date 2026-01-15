package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
		kid, secret, err := s.apiKeySvc.CreateKey(c.Request.Context(), req.TenantID, req.Scopes, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return secret only once
		c.JSON(http.StatusCreated, gin.H{"key_id": kid, "secret": secret})
	}
}
