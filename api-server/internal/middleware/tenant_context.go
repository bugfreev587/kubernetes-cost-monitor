package middleware

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TenantContextMiddleware sets the PostgreSQL session variable for tenant isolation
// This must be called AFTER authentication middleware that sets c.Get("tenant_id")
func TenantContextMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant_id from context (set by auth middleware)
		tenantIDInterface, exists := c.Get("tenant_id")
		if !exists {
			// No tenant context - might be a public endpoint
			c.Next()
			return
		}

		// Convert to uint
		var tenantID uint
		switch v := tenantIDInterface.(type) {
		case uint:
			tenantID = v
		case int:
			tenantID = uint(v)
		case float64:
			tenantID = uint(v)
		default:
			log.Printf("Invalid tenant_id type in context: %T", tenantIDInterface)
			c.Next()
			return
		}

		// Set PostgreSQL session variable for RLS
		if err := SetTenantContext(db, tenantID); err != nil {
			log.Printf("Failed to set tenant context: %v", err)
			// Don't fail the request - this is a safety feature
			// but we log it for monitoring
		}

		// Continue to next handler
		c.Next()

		// Optional: Clear context after request completes
		// This is less critical since connections are pooled
		// but good practice for long-running transactions
		if err := ClearTenantContext(db); err != nil {
			log.Printf("Failed to clear tenant context: %v", err)
		}
	}
}

// SetTenantContext sets the app.current_tenant_id session variable
func SetTenantContext(db *gorm.DB, tenantID uint) error {
	sql := "SELECT set_config('app.current_tenant_id', $1, false)"
	result := db.Exec(sql, fmt.Sprintf("%d", tenantID))
	if result.Error != nil {
		return fmt.Errorf("failed to set tenant context: %w", result.Error)
	}
	return nil
}

// ClearTenantContext clears the tenant context
func ClearTenantContext(db *gorm.DB) error {
	sql := "SELECT set_config('app.current_tenant_id', '', false)"
	result := db.Exec(sql)
	if result.Error != nil {
		return fmt.Errorf("failed to clear tenant context: %w", result.Error)
	}
	return nil
}

// EnableAdminMode enables bypass of RLS policies for admin operations
func EnableAdminMode(db *gorm.DB) error {
	sql := "SELECT set_config('app.bypass_rls', 'true', false)"
	result := db.Exec(sql)
	if result.Error != nil {
		return fmt.Errorf("failed to enable admin mode: %w", result.Error)
	}
	return nil
}

// DisableAdminMode disables RLS bypass
func DisableAdminMode(db *gorm.DB) error {
	sql := "SELECT set_config('app.bypass_rls', 'false', false)"
	result := db.Exec(sql)
	if result.Error != nil {
		return fmt.Errorf("failed to disable admin mode: %w", result.Error)
	}
	return nil
}

// WithTenantContext executes a function with tenant context set
func WithTenantContext(db *gorm.DB, tenantID uint, fn func(*gorm.DB) error) error {
	// Set tenant context
	if err := SetTenantContext(db, tenantID); err != nil {
		return err
	}

	// Execute function
	err := fn(db)

	// Clear context
	if clearErr := ClearTenantContext(db); clearErr != nil {
		log.Printf("Failed to clear tenant context: %v", clearErr)
	}

	return err
}

// WithAdminMode executes a function with admin mode enabled (bypasses RLS)
func WithAdminMode(db *gorm.DB, fn func(*gorm.DB) error) error {
	// Enable admin mode
	if err := EnableAdminMode(db); err != nil {
		return err
	}

	// Execute function
	err := fn(db)

	// Disable admin mode
	if disableErr := DisableAdminMode(db); disableErr != nil {
		log.Printf("Failed to disable admin mode: %v", disableErr)
	}

	return err
}
