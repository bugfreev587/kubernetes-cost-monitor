package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Logger middleware
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		log.Printf("[%s] %s %d %v",
			c.Request.Method,
			path,
			statusCode,
			duration,
		)
	}
}

// CORS middleware
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-User-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// For non-OPTIONS requests, only proceed if origin is allowed
		if !allowed && origin != "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Auth middleware for API keys
func AuthMiddleware(apiKeyHeader string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(apiKeyHeader)

		if apiKey == "" {
			// Try Authorization header
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			c.Abort()
			return
		}

		// Validate API key (simplified - use database in production)
		tenantID, valid := validateAPIKey(apiKey)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		// Set tenant ID in context
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// Simple API key validation (replace with database lookup)
func validateAPIKey(apiKey string) (string, bool) {
	// In production, query database for API key
	validKeys := map[string]string{
		"dev-api-key-12345": "tenant-1",
		"test-api-key":      "tenant-2",
	}

	tenantID, ok := validKeys[apiKey]
	return tenantID, ok
}

// Rate limiting middleware
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(requestsPerMinute)/60, requestsPerMinute)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
