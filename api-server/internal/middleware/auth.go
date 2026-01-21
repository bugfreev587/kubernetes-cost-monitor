package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
)

// APIKeyError codes for client handling
const (
	ErrCodeMissingKey   = "api_key_missing"
	ErrCodeInvalidKey   = "api_key_invalid"
	ErrCodeExpiredKey   = "api_key_expired"
	ErrCodeRevokedKey   = "api_key_revoked"
	ErrCodeBadFormat    = "api_key_bad_format"
)

func NewAPIKeyMiddleware(svc *services.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Accept both Authorization: ApiKey <keyid:secret> or X-Api-Key
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-Api-Key")
		}
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   ErrCodeMissingKey,
				"message": "API key is required. Provide via 'Authorization: ApiKey <key>' or 'X-Api-Key' header.",
			})
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

			// Parse error type and return appropriate response
			errStr := err.Error()
			response := gin.H{}

			switch {
			case strings.Contains(errStr, "expired"):
				response["error"] = ErrCodeExpiredKey
				response["message"] = "API key has expired. Please generate a new key from your dashboard."
			case strings.Contains(errStr, "revoked"):
				response["error"] = ErrCodeRevokedKey
				response["message"] = "API key has been revoked. Please generate a new key from your dashboard."
			case strings.Contains(errStr, "bad key format"):
				response["error"] = ErrCodeBadFormat
				response["message"] = "Invalid API key format. Expected format: 'keyid:secret'."
			case strings.Contains(errStr, "not found"):
				response["error"] = ErrCodeInvalidKey
				response["message"] = "API key not found. Please check your key or generate a new one."
			case strings.Contains(errStr, "hash mismatch"):
				response["error"] = ErrCodeInvalidKey
				response["message"] = "Invalid API key secret. Please check your key or generate a new one."
			default:
				response["error"] = ErrCodeInvalidKey
				response["message"] = "Invalid API key. Please check your key or generate a new one."
			}

			c.JSON(http.StatusUnauthorized, response)
			c.Abort()
			return
		}
		// put API key metadata into context for handlers
		c.Set("api_key", ak)
		c.Next()
	}
}
