package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

// SyncUserRequest represents the request body for user sync
type SyncUserRequest struct {
	ClerkUserID string `json:"clerk_user_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// SyncUserResponse represents the response for user sync
type SyncUserResponse struct {
	UserID      string  `json:"user_id"`
	TenantID    uint    `json:"tenant_id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	PricingPlan string  `json:"pricing_plan"`
	IsNewUser   bool    `json:"is_new_user"`
	APIKey      *string `json:"api_key,omitempty"` // Only returned for new users
}

// syncUserHandler handles POST /v1/auth/sync
// This endpoint is called by the frontend after Clerk authentication
// to ensure the user exists in our database
func (s *Server) syncUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SyncUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		// Check if user already exists by Clerk ID (primary key)
		var existingUser models.User
		result := db.Where("id = ?", req.ClerkUserID).First(&existingUser)

		if result.Error == nil {
			// User exists - check if suspended
			if existingUser.Status == models.StatusSuspended {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "user_suspended",
					"message": "Your account has been suspended. Please contact your organization administrator.",
				})
				return
			}

			// Ensure Clerk user metadata is up to date (for Grafana OAuth integration)
			if s.clerkSvc != nil && s.clerkSvc.IsConfigured() {
				if err := s.clerkSvc.UpdateUserMetadata(c.Request.Context(), req.ClerkUserID, existingUser.TenantID, existingUser.Role); err != nil {
					log.Printf("Warning: Failed to update Clerk user metadata: %v", err)
					// Don't fail the sync, just log the warning
				}
			}

			// Return user info
			var tenant models.Tenant
			db.First(&tenant, existingUser.TenantID)

			c.JSON(http.StatusOK, SyncUserResponse{
				UserID:      existingUser.ID,
				TenantID:    existingUser.TenantID,
				Email:       existingUser.Email,
				Name:        existingUser.Name,
				Role:        existingUser.Role,
				Status:      existingUser.Status,
				PricingPlan: tenant.PricingPlan,
				IsNewUser:   false,
			})
			return
		}

		// User doesn't exist by Clerk ID - check for pending invitation by email
		var pendingUser models.User
		pendingResult := db.Where("email = ? AND status = ?", req.Email, models.StatusPending).First(&pendingUser)

		name := fmt.Sprintf("%s %s", req.FirstName, req.LastName)
		if name == " " {
			name = req.Email
		}

		if pendingResult.Error == nil {
			// Found pending invitation - update the user record with Clerk ID and activate
			oldID := pendingUser.ID
			pendingUser.ID = req.ClerkUserID
			pendingUser.Name = name
			pendingUser.Status = models.StatusActive

			// Delete old pending record and create new one with Clerk ID
			if err := db.Delete(&models.User{}, "id = ?", oldID).Error; err != nil {
				log.Printf("Failed to delete pending user: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate invited user"})
				return
			}

			if err := db.Create(&pendingUser).Error; err != nil {
				log.Printf("Failed to create activated user: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate invited user"})
				return
			}

			log.Printf("Activated invited user: email=%s, tenant_id=%d, user_id=%s, role=%s",
				pendingUser.Email, pendingUser.TenantID, pendingUser.ID, pendingUser.Role)

			// Update Clerk user metadata for Grafana OAuth integration
			if s.clerkSvc != nil && s.clerkSvc.IsConfigured() {
				if err := s.clerkSvc.UpdateUserMetadata(c.Request.Context(), req.ClerkUserID, pendingUser.TenantID, pendingUser.Role); err != nil {
					log.Printf("Warning: Failed to update Clerk user metadata: %v", err)
					// Don't fail the sync, just log the warning
				}
			}

			// Get tenant info
			var tenant models.Tenant
			db.First(&tenant, pendingUser.TenantID)

			c.JSON(http.StatusOK, SyncUserResponse{
				UserID:      pendingUser.ID,
				TenantID:    pendingUser.TenantID,
				Email:       pendingUser.Email,
				Name:        pendingUser.Name,
				Role:        pendingUser.Role,
				Status:      pendingUser.Status,
				PricingPlan: tenant.PricingPlan,
				IsNewUser:   true, // Treat as new user for welcome experience
			})
			return
		}

		// No pending invitation - create new tenant and user
		// Create tenant
		tenant := models.Tenant{
			Name:        name,
			PricingPlan: "Starter",
			CreatedAt:   time.Now(),
		}

		if err := db.Create(&tenant).Error; err != nil {
			log.Printf("Failed to create tenant: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create tenant: %v", err)})
			return
		}

		// Create user with Clerk ID as the primary key (first user is owner)
		user := models.User{
			ID:        req.ClerkUserID, // Use Clerk ID as primary key
			TenantID:  tenant.ID,
			Email:     req.Email,
			Name:      name,
			Role:      models.RoleOwner, // First user of tenant is owner
			Status:    models.StatusActive,
			CreatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed to create user: %v", err)
			// Rollback tenant creation
			db.Delete(&tenant)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create user: %v", err)})
			return
		}

		log.Printf("Created new user (owner): email=%s, tenant_id=%d, user_id=%s", req.Email, tenant.ID, user.ID)

		// Update Clerk user metadata for Grafana OAuth integration
		if s.clerkSvc != nil && s.clerkSvc.IsConfigured() {
			if err := s.clerkSvc.UpdateUserMetadata(c.Request.Context(), req.ClerkUserID, tenant.ID, user.Role); err != nil {
				log.Printf("Warning: Failed to update Clerk user metadata: %v", err)
				// Don't fail the sync, just log the warning
			}
		}

		// Create a default API key for the new tenant
		var apiKeyStr *string
		expiresAt := time.Now().AddDate(1, 0, 0) // Expires in 1 year
		keyID, secret, err := s.apiKeySvc.CreateKey(c.Request.Context(), tenant.ID, []string{"*"}, &expiresAt)
		if err != nil {
			log.Printf("Warning: Failed to create default API key for tenant %d: %v", tenant.ID, err)
			// Don't fail the signup, just log the warning
		} else {
			fullKey := fmt.Sprintf("%s:%s", keyID, secret)
			apiKeyStr = &fullKey
			log.Printf("Created default API key for tenant %d: key_id=%s", tenant.ID, keyID)
		}

		c.JSON(http.StatusCreated, SyncUserResponse{
			UserID:      user.ID,
			TenantID:    tenant.ID,
			Email:       user.Email,
			Name:        user.Name,
			Role:        user.Role,
			Status:      user.Status,
			PricingPlan: tenant.PricingPlan,
			IsNewUser:   true,
			APIKey:      apiKeyStr,
		})
	}
}
