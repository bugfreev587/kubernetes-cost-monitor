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
	UserID      string `json:"user_id"`
	TenantID    uint   `json:"tenant_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	PricingPlan string `json:"pricing_plan"`
	IsNewUser   bool   `json:"is_new_user"`
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

		// Check if user already exists by email
		var existingUser models.User
		result := db.Where("email = ?", req.Email).First(&existingUser)

		if result.Error == nil {
			// User exists, return their info
			var tenant models.Tenant
			db.First(&tenant, existingUser.TenantID)

			c.JSON(http.StatusOK, SyncUserResponse{
				UserID:      existingUser.ID,
				TenantID:    existingUser.TenantID,
				Email:       existingUser.Email,
				Name:        existingUser.Name,
				Role:        existingUser.Role,
				PricingPlan: tenant.PricingPlan,
				IsNewUser:   false,
			})
			return
		}

		// User doesn't exist, create new tenant and user
		name := fmt.Sprintf("%s %s", req.FirstName, req.LastName)
		if name == " " {
			name = req.Email
		}

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

		// Create user (first user is admin)
		user := models.User{
			TenantID:  tenant.ID,
			Email:     req.Email,
			Name:      name,
			Role:      "admin",
			CreatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed to create user: %v", err)
			// Rollback tenant creation
			db.Delete(&tenant)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create user: %v", err)})
			return
		}

		log.Printf("Created new user: email=%s, tenant_id=%d, user_id=%d", req.Email, tenant.ID, user.ID)

		c.JSON(http.StatusCreated, SyncUserResponse{
			UserID:      user.ID,
			TenantID:    tenant.ID,
			Email:       user.Email,
			Name:        user.Name,
			Role:        user.Role,
			PricingPlan: tenant.PricingPlan,
			IsNewUser:   true,
		})
	}
}
