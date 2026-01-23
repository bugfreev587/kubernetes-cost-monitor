package middleware

import (
	"net/http"
	"strings"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RBAC error codes
const (
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeForbidden       = "forbidden"
	ErrCodeUserSuspended   = "user_suspended"
	ErrCodeInsufficientRole = "insufficient_role"
)

// RBACMiddleware provides role-based access control
type RBACMiddleware struct {
	db *gorm.DB
}

// NewRBACMiddleware creates a new RBAC middleware instance
func NewRBACMiddleware(db *gorm.DB) *RBACMiddleware {
	return &RBACMiddleware{db: db}
}

// RequireUser authenticates the user via Clerk user ID header
// This is used for frontend requests where the user is authenticated via Clerk
func (m *RBACMiddleware) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from header (set by frontend after Clerk authentication)
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   ErrCodeUnauthorized,
				"message": "Authentication required. Please sign in.",
			})
			c.Abort()
			return
		}

		// Look up user in database
		var user models.User
		if err := m.db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   ErrCodeUnauthorized,
				"message": "User not found. Please sign in again.",
			})
			c.Abort()
			return
		}

		// Check if user is suspended
		if !user.IsActive() {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   ErrCodeUserSuspended,
				"message": "Your account has been suspended. Please contact your organization administrator.",
			})
			c.Abort()
			return
		}

		// Store user in context for downstream handlers
		c.Set("user", &user)
		c.Set("tenant_id", user.TenantID)
		c.Next()
	}
}

// RequireRole returns a middleware that checks if user has at least the required role
func (m *RBACMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure user is authenticated
		userI, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   ErrCodeUnauthorized,
				"message": "Authentication required.",
			})
			c.Abort()
			return
		}

		user := userI.(*models.User)

		// Check role permission
		if !user.HasPermission(requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   ErrCodeInsufficientRole,
				"message": "You don't have permission to perform this action. Required role: " + requiredRole,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOwner is a convenience middleware for owner-only endpoints
func (m *RBACMiddleware) RequireOwner() gin.HandlerFunc {
	return m.RequireRole(models.RoleOwner)
}

// RequireAdmin is a convenience middleware for admin+ endpoints
func (m *RBACMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin)
}

// RequireEditor is a convenience middleware for editor+ endpoints
func (m *RBACMiddleware) RequireEditor() gin.HandlerFunc {
	return m.RequireRole(models.RoleEditor)
}

// RequireViewer is a convenience middleware for viewer+ endpoints (authenticated users)
func (m *RBACMiddleware) RequireViewer() gin.HandlerFunc {
	return m.RequireRole(models.RoleViewer)
}

// RequireTenantAccess checks if the user belongs to the specified tenant
// This should be used after RequireUser middleware
func (m *RBACMiddleware) RequireTenantAccess(tenantIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userI, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   ErrCodeUnauthorized,
				"message": "Authentication required.",
			})
			c.Abort()
			return
		}

		user := userI.(*models.User)

		// Get tenant ID from URL parameter
		tenantIDStr := c.Param(tenantIDParam)
		if tenantIDStr == "" {
			// Try to get from context (might be set by another middleware)
			if tid, ok := c.Get("tenant_id"); ok {
				if user.TenantID != tid.(uint) {
					c.JSON(http.StatusForbidden, gin.H{
						"error":   ErrCodeForbidden,
						"message": "You don't have access to this tenant.",
					})
					c.Abort()
					return
				}
			}
			c.Next()
			return
		}

		// Parse tenant ID from parameter
		var tenantID uint
		if _, err := parseUint(tenantIDStr, &tenantID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_tenant_id",
				"message": "Invalid tenant ID format.",
			})
			c.Abort()
			return
		}

		// Check if user belongs to this tenant
		if user.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   ErrCodeForbidden,
				"message": "You don't have access to this tenant.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAPIKeyOrUser allows either API key or user authentication
// This is useful for endpoints that can be called by either the agent or the frontend
func (m *RBACMiddleware) RequireAPIKeyOrUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if API key is already set (from API key middleware)
		if _, exists := c.Get("api_key"); exists {
			c.Next()
			return
		}

		// Check for user authentication
		userID := c.GetHeader("X-User-ID")
		if userID != "" {
			var user models.User
			if err := m.db.Where("id = ?", userID).First(&user).Error; err == nil {
				if user.IsActive() {
					c.Set("user", &user)
					c.Set("tenant_id", user.TenantID)
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   ErrCodeUnauthorized,
			"message": "Authentication required. Provide API key or user credentials.",
		})
		c.Abort()
	}
}

// Helper function to parse uint from string
func parseUint(s string, result *uint) (bool, error) {
	s = strings.TrimSpace(s)
	var val uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		val = val*10 + uint64(c-'0')
	}
	*result = uint(val)
	return true, nil
}

// GetUserFromContext retrieves the authenticated user from the request context
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	userI, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	user, ok := userI.(*models.User)
	return user, ok
}

// GetTenantIDFromContext retrieves the tenant ID from the request context
func GetTenantIDFromContext(c *gin.Context) (uint, bool) {
	// First try to get from user
	if user, ok := GetUserFromContext(c); ok {
		return user.TenantID, true
	}

	// Then try from API key
	if akI, exists := c.Get("api_key"); exists {
		ak := akI.(*models.APIKey)
		return ak.TenantID, true
	}

	// Finally try from context directly
	if tid, exists := c.Get("tenant_id"); exists {
		return tid.(uint), true
	}

	return 0, false
}
