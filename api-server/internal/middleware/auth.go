package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
)

func NewAPIKeyMiddleware(svc *services.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Accept both Authorization: ApiKey <keyid:secret> or X-Api-Key
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-Api-Key")
		}
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "ApiKey"))
		token = strings.TrimSpace(token)
		ak, err := svc.ValidateKey(c.Request.Context(), token)
		if err != nil {
			// Log the actual error for debugging (remove sensitive info in production)
			tokenPrefix := token
			if len(token) > 20 {
				tokenPrefix = token[:20] + "..."
			}
			log.Printf("API key validation failed: %v (token prefix: %s)", err, tokenPrefix)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key", "details": err.Error()})
			c.Abort()
			return
		}
		// put API key metadata into context for handlers
		c.Set("api_key", ak)
		c.Next()
	}
}
