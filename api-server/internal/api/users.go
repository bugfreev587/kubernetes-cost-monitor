package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/gin-gonic/gin"
)

// UserResponse represents a user in API responses
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

// InviteUserRequest represents the request to invite a user
type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name"`
	Role  string `json:"role"` // Optional, defaults to viewer
}

// UpdateUserRoleRequest represents the request to update a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// listUsersHandler returns all users in the caller's tenant
func (s *Server) listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		var users []models.User
		if err := db.Where("tenant_id = ?", user.TenantID).Order("created_at ASC").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
			return
		}

		response := ListUsersResponse{
			Users: make([]UserResponse, len(users)),
			Total: len(users),
		}

		for i, u := range users {
			response.Users[i] = UserResponse{
				ID:        u.ID,
				Email:     u.Email,
				Name:      u.Name,
				Role:      u.Role,
				Status:    u.Status,
				CreatedAt: u.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// getUserHandler returns a specific user by ID
func (s *Server) getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var user models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		})
	}
}

// inviteUserHandler creates an invitation for a new user
// This sends an invitation email via Clerk and creates a placeholder user in the database
func (s *Server) inviteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req InviteUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate role
		role := models.RoleViewer
		if req.Role != "" {
			if req.Role != models.RoleViewer && req.Role != models.RoleEditor {
				// Admins can only invite viewers and editors
				// Owner role requires promote-admin endpoint
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_role",
					"message": "Can only invite users as 'viewer' or 'editor'",
				})
				return
			}
			role = req.Role
		}

		db := s.postgresDB.GetPostgresDB()

		// Check if user already exists
		var existingUser models.User
		if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "user_exists",
				"message": "A user with this email already exists",
			})
			return
		}

		// Send invitation via Clerk (this sends the email)
		var clerkInvitationID string
		emailSent := false
		if s.clerkSvc != nil && s.clerkSvc.IsConfigured() {
			invResp, err := s.clerkSvc.CreateInvitation(c.Request.Context(), req.Email, currentUser.TenantID, role, currentUser.Name)
			if err != nil {
				log.Printf("Failed to send Clerk invitation (will continue without email): %v", err)
				// Continue without email - user can still be added manually
			} else {
				clerkInvitationID = invResp.ID
				emailSent = true
				log.Printf("Clerk invitation sent: id=%s, email=%s", invResp.ID, req.Email)
			}
		} else {
			log.Printf("Clerk not configured - invitation email will not be sent for %s", req.Email)
		}

		// Create a pending user in the database (will be activated when they sign up)
		invitedUser := models.User{
			ID:        fmt.Sprintf("pending_%s_%d", req.Email, time.Now().UnixNano()),
			TenantID:  currentUser.TenantID,
			Email:     req.Email,
			Name:      req.Name,
			Role:      role,
			Status:    models.StatusPending, // Special status for invited users
			CreatedAt: time.Now(),
		}

		if err := db.Create(&invitedUser).Error; err != nil {
			log.Printf("Failed to create invited user: %v", err)
			// If we sent a Clerk invitation, try to revoke it
			if clerkInvitationID != "" && s.clerkSvc != nil {
				_ = s.clerkSvc.RevokeInvitation(c.Request.Context(), clerkInvitationID)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to invite user"})
			return
		}

		log.Printf("User invited: email=%s, role=%s, by=%s, email_sent=%v", req.Email, role, currentUser.Email, emailSent)

		responseMsg := "User invited successfully"
		if emailSent {
			responseMsg = "User invited successfully. Invitation email sent."
		} else {
			responseMsg = "User invited successfully. Note: Invitation email could not be sent - please share the signup link manually."
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    responseMsg,
			"email_sent": emailSent,
			"user": UserResponse{
				ID:        invitedUser.ID,
				Email:     invitedUser.Email,
				Name:      invitedUser.Name,
				Role:      invitedUser.Role,
				Status:    invitedUser.Status,
				CreatedAt: invitedUser.CreatedAt,
			},
		})
	}
}

// suspendUserHandler suspends a user
func (s *Server) suspendUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Cannot suspend yourself
		if targetUser.ID == currentUser.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "cannot_suspend_self",
				"message": "You cannot suspend yourself",
			})
			return
		}

		// Cannot suspend owner
		if targetUser.Role == models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "cannot_suspend_owner",
				"message": "Cannot suspend the tenant owner",
			})
			return
		}

		// Admins cannot suspend other admins (only owner can)
		if targetUser.Role == models.RoleAdmin && currentUser.Role != models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permission",
				"message": "Only the owner can suspend admins",
			})
			return
		}

		targetUser.Status = models.StatusSuspended
		if err := db.Save(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to suspend user"})
			return
		}

		log.Printf("User suspended: %s by %s", targetUser.Email, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "User suspended successfully",
			"user": UserResponse{
				ID:        targetUser.ID,
				Email:     targetUser.Email,
				Name:      targetUser.Name,
				Role:      targetUser.Role,
				Status:    targetUser.Status,
				CreatedAt: targetUser.CreatedAt,
			},
		})
	}
}

// unsuspendUserHandler unsuspends a user
func (s *Server) unsuspendUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		targetUser.Status = models.StatusActive
		if err := db.Save(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unsuspend user"})
			return
		}

		log.Printf("User unsuspended: %s by %s", targetUser.Email, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "User unsuspended successfully",
			"user": UserResponse{
				ID:        targetUser.ID,
				Email:     targetUser.Email,
				Name:      targetUser.Name,
				Role:      targetUser.Role,
				Status:    targetUser.Status,
				CreatedAt: targetUser.CreatedAt,
			},
		})
	}
}

// updateUserRoleHandler updates a user's role (admin can set viewer/editor)
func (s *Server) updateUserRoleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		var req UpdateUserRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Admins can only set viewer or editor roles
		// Admin/owner roles require owner-only endpoints
		if req.Role != models.RoleViewer && req.Role != models.RoleEditor {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_role",
				"message": "Can only set role to 'viewer' or 'editor'. Use owner endpoints for admin promotion.",
			})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Cannot change owner's role
		if targetUser.Role == models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "cannot_modify_owner",
				"message": "Cannot change the owner's role",
			})
			return
		}

		// Cannot change admin's role (only owner can demote admins)
		if targetUser.Role == models.RoleAdmin && currentUser.Role != models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permission",
				"message": "Only the owner can change admin roles",
			})
			return
		}

		oldRole := targetUser.Role
		targetUser.Role = req.Role
		if err := db.Save(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
			return
		}

		log.Printf("User role changed: %s from %s to %s by %s", targetUser.Email, oldRole, req.Role, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "Role updated successfully",
			"user": UserResponse{
				ID:        targetUser.ID,
				Email:     targetUser.Email,
				Name:      targetUser.Name,
				Role:      targetUser.Role,
				Status:    targetUser.Status,
				CreatedAt: targetUser.CreatedAt,
			},
		})
	}
}

// removeUserHandler removes a user from the tenant
func (s *Server) removeUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Cannot remove yourself
		if targetUser.ID == currentUser.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "cannot_remove_self",
				"message": "You cannot remove yourself",
			})
			return
		}

		// Cannot remove owner
		if targetUser.Role == models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "cannot_remove_owner",
				"message": "Cannot remove the tenant owner. Transfer ownership first.",
			})
			return
		}

		// Admins cannot remove other admins (only owner can)
		if targetUser.Role == models.RoleAdmin && currentUser.Role != models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permission",
				"message": "Only the owner can remove admins",
			})
			return
		}

		if err := db.Delete(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user"})
			return
		}

		log.Printf("User removed: %s by %s", targetUser.Email, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{"message": "User removed successfully"})
	}
}

// promoteToAdminHandler promotes a user to admin (owner only)
func (s *Server) promoteToAdminHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Cannot promote yourself (owner is already max)
		if targetUser.ID == currentUser.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "cannot_promote_self",
				"message": "You cannot promote yourself",
			})
			return
		}

		// Cannot promote owner (already max role)
		if targetUser.Role == models.RoleOwner {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "already_owner",
				"message": "User is already the owner",
			})
			return
		}

		// Already admin
		if targetUser.Role == models.RoleAdmin {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "already_admin",
				"message": "User is already an admin",
			})
			return
		}

		oldRole := targetUser.Role
		targetUser.Role = models.RoleAdmin
		if err := db.Save(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to promote user"})
			return
		}

		log.Printf("User promoted to admin: %s from %s by %s", targetUser.Email, oldRole, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "User promoted to admin successfully",
			"user": UserResponse{
				ID:        targetUser.ID,
				Email:     targetUser.Email,
				Name:      targetUser.Name,
				Role:      targetUser.Role,
				Status:    targetUser.Status,
				CreatedAt: targetUser.CreatedAt,
			},
		})
	}
}

// demoteAdminHandler demotes an admin to editor (owner only)
func (s *Server) demoteAdminHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID := c.Param("user_id")

		db := s.postgresDB.GetPostgresDB()

		var targetUser models.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, currentUser.TenantID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Only admins can be demoted via this endpoint
		if targetUser.Role != models.RoleAdmin {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "not_admin",
				"message": "User is not an admin. Use role update endpoint for viewers/editors.",
			})
			return
		}

		targetUser.Role = models.RoleEditor
		if err := db.Save(&targetUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to demote admin"})
			return
		}

		log.Printf("Admin demoted to editor: %s by %s", targetUser.Email, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "Admin demoted to editor successfully",
			"user": UserResponse{
				ID:        targetUser.ID,
				Email:     targetUser.Email,
				Name:      targetUser.Name,
				Role:      targetUser.Role,
				Status:    targetUser.Status,
				CreatedAt: targetUser.CreatedAt,
			},
		})
	}
}

// TransferOwnershipRequest represents the request to transfer ownership
type TransferOwnershipRequest struct {
	NewOwnerID string `json:"new_owner_id" binding:"required"`
}

// transferOwnershipHandler transfers tenant ownership to another user
func (s *Server) transferOwnershipHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req TransferOwnershipRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.NewOwnerID == currentUser.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "already_owner",
				"message": "You are already the owner",
			})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		// Find the new owner
		var newOwner models.User
		if err := db.Where("id = ? AND tenant_id = ?", req.NewOwnerID, currentUser.TenantID).First(&newOwner).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "target user not found in your tenant"})
			return
		}

		// New owner must be active
		if newOwner.Status != models.StatusActive {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "user_not_active",
				"message": "Cannot transfer ownership to a suspended user",
			})
			return
		}

		// Start transaction
		tx := db.Begin()

		// Demote current owner to admin
		if err := tx.Model(&models.User{}).Where("id = ?", currentUser.ID).Update("role", models.RoleAdmin).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer ownership"})
			return
		}

		// Promote new user to owner
		if err := tx.Model(&models.User{}).Where("id = ?", newOwner.ID).Update("role", models.RoleOwner).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer ownership"})
			return
		}

		tx.Commit()

		log.Printf("Ownership transferred: from %s to %s", currentUser.Email, newOwner.Email)

		c.JSON(http.StatusOK, gin.H{
			"message":   "Ownership transferred successfully",
			"new_owner": newOwner.Email,
		})
	}
}

// deleteTenantHandler deletes a tenant and all associated data (owner only)
func (s *Server) deleteTenantHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tenantIDStr := c.Param("tenant_id")
		var tenantID uint
		if _, err := fmt.Sscanf(tenantIDStr, "%d", &tenantID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
			return
		}

		// Verify ownership
		if currentUser.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "you don't have access to this tenant"})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		// Delete tenant (cascade will delete users, api_keys, recommendations)
		var tenant models.Tenant
		if err := db.First(&tenant, tenantID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}

		if err := db.Delete(&tenant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tenant"})
			return
		}

		log.Printf("Tenant deleted: id=%d, name=%s by %s", tenantID, tenant.Name, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted successfully"})
	}
}

// listAPIKeysHandler returns all API keys for the tenant (masked secrets)
func (s *Server) listAPIKeysHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		db := s.postgresDB.GetPostgresDB()

		var apiKeys []models.APIKey
		if err := db.Where("tenant_id = ?", currentUser.TenantID).Order("created_at DESC").Find(&apiKeys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch API keys"})
			return
		}

		type APIKeyResponse struct {
			ID        uint       `json:"id"`
			KeyID     string     `json:"key_id"`
			Scopes    []string   `json:"scopes"`
			Revoked   bool       `json:"revoked"`
			ExpiresAt *time.Time `json:"expires_at"`
			CreatedAt time.Time  `json:"created_at"`
		}

		response := make([]APIKeyResponse, len(apiKeys))
		for i, key := range apiKeys {
			response[i] = APIKeyResponse{
				ID:        key.ID,
				KeyID:     key.KeyID,
				Scopes:    key.Scopes,
				Revoked:   key.Revoked,
				ExpiresAt: key.ExpiresAt,
				CreatedAt: key.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"api_keys": response,
			"total":    len(response),
		})
	}
}

// revokeAPIKeyHandler revokes an API key
func (s *Server) revokeAPIKeyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := middleware.GetUserFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		keyID := c.Param("key_id")

		db := s.postgresDB.GetPostgresDB()

		var apiKey models.APIKey
		if err := db.Where("key_id = ? AND tenant_id = ?", keyID, currentUser.TenantID).First(&apiKey).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}

		if apiKey.Revoked {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key is already revoked"})
			return
		}

		apiKey.Revoked = true
		if err := db.Save(&apiKey).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke API key"})
			return
		}

		log.Printf("API key revoked: %s by %s", keyID, currentUser.Email)

		c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
	}
}
