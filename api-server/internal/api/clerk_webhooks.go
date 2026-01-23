package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
)

// ClerkWebhookHandler handles webhooks from Clerk
type ClerkWebhookHandler struct {
	db             *gorm.DB
	grafanaService *services.GrafanaService
}

// ClerkWebhookEvent represents a webhook event from Clerk
type ClerkWebhookEvent struct {
	Type   string                 `json:"type"`
	Object string                 `json:"object"`
	Data   map[string]interface{} `json:"data"`
}

// ClerkUser represents user data from Clerk webhook
type ClerkUser struct {
	ID             string                 `json:"id"`
	Email          string                 `json:"email_addresses"`
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	PublicMetadata map[string]interface{} `json:"public_metadata"`
}

func NewClerkWebhookHandler(db *gorm.DB, grafanaService *services.GrafanaService) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{
		db:             db,
		grafanaService: grafanaService,
	}
}

// HandleWebhook processes Clerk webhook events
func (h *ClerkWebhookHandler) HandleWebhook(c *gin.Context) {
	var event ClerkWebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	log.Printf("Received Clerk webhook: type=%s", event.Type)

	switch event.Type {
	case "user.created":
		h.handleUserCreated(c, event.Data)
	case "user.updated":
		h.handleUserUpdated(c, event.Data)
	case "user.deleted":
		h.handleUserDeleted(c, event.Data)
	default:
		log.Printf("Unhandled webhook type: %s", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook processed"})
}

func (h *ClerkWebhookHandler) handleUserCreated(c *gin.Context, data map[string]interface{}) {
	ctx := c.Request.Context()

	// Parse Clerk user ID
	clerkUserID, ok := data["id"].(string)
	if !ok || clerkUserID == "" {
		log.Printf("No user ID in user.created webhook")
		return
	}

	// Parse email
	emailAddresses, ok := data["email_addresses"].([]interface{})
	if !ok || len(emailAddresses) == 0 {
		log.Printf("No email addresses in user.created webhook")
		return
	}

	emailData, ok := emailAddresses[0].(map[string]interface{})
	if !ok {
		log.Printf("Invalid email address format")
		return
	}

	email, ok := emailData["email_address"].(string)
	if !ok || email == "" {
		log.Printf("Email address not found")
		return
	}

	// Parse name
	firstName, _ := data["first_name"].(string)
	lastName, _ := data["last_name"].(string)
	name := strings.TrimSpace(fmt.Sprintf("%s %s", firstName, lastName))
	if name == "" {
		// Fallback to email if no name provided
		name = email
	}

	// Parse public metadata
	publicMetadata, _ := data["public_metadata"].(map[string]interface{})
	tenantID := h.extractTenantID(publicMetadata)

	// If no tenant_id in metadata, create a new tenant for this user
	if tenantID == 0 {
		tenant, err := h.createDefaultTenant(ctx, name, email)
		if err != nil {
			log.Printf("Failed to create default tenant: %v", err)
			return
		}
		tenantID = tenant.ID

		// TODO: Update Clerk user metadata with tenant_id
		// This requires calling Clerk API to set public_metadata
		log.Printf("Created new tenant %d for user %s", tenantID, email)
	}

	// Determine user role: first user of tenant is owner, others are viewers
	role := models.RoleViewer
	var existingUserCount int64
	h.db.Model(&models.User{}).Where("tenant_id = ?", tenantID).Count(&existingUserCount)
	if existingUserCount == 0 {
		role = models.RoleOwner // First user becomes owner
	}

	// Create user in database with Clerk ID as primary key
	user := models.User{
		ID:        clerkUserID, // Use Clerk ID as primary key
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
		Role:      role,
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&user).Error; err != nil {
		log.Printf("Failed to create user in database: %v", err)
		return
	}

	// Sync Grafana organization for tenant
	if h.grafanaService != nil {
		if err := h.syncGrafanaOrg(ctx, tenantID); err != nil {
			log.Printf("Failed to sync Grafana org: %v", err)
		}
	}

	log.Printf("User created: id=%s, email=%s, tenant_id=%d", clerkUserID, email, tenantID)
}

func (h *ClerkWebhookHandler) handleUserUpdated(c *gin.Context, data map[string]interface{}) {
	ctx := c.Request.Context()

	// Parse Clerk user ID
	clerkUserID, ok := data["id"].(string)
	if !ok || clerkUserID == "" {
		log.Printf("No user ID in user.updated webhook")
		return
	}

	// Find user by Clerk ID (primary key)
	var user models.User
	if err := h.db.Where("id = ?", clerkUserID).First(&user).Error; err != nil {
		log.Printf("User not found for update: clerk_id=%s", clerkUserID)
		return
	}

	// Parse email and update if changed
	emailAddresses, ok := data["email_addresses"].([]interface{})
	if ok && len(emailAddresses) > 0 {
		if emailData, ok := emailAddresses[0].(map[string]interface{}); ok {
			if email, ok := emailData["email_address"].(string); ok && email != "" && email != user.Email {
				user.Email = email
			}
		}
	}

	// Update name if changed
	firstName, _ := data["first_name"].(string)
	lastName, _ := data["last_name"].(string)
	newName := fmt.Sprintf("%s %s", firstName, lastName)

	if newName != user.Name {
		user.Name = newName
	}

	// Check if tenant_id changed in metadata
	publicMetadata, _ := data["public_metadata"].(map[string]interface{})
	newTenantID := h.extractTenantID(publicMetadata)

	if newTenantID != 0 && newTenantID != user.TenantID {
		// Tenant changed - update user and sync Grafana
		user.TenantID = newTenantID

		if h.grafanaService != nil {
			if err := h.syncGrafanaOrg(ctx, newTenantID); err != nil {
				log.Printf("Failed to sync Grafana org: %v", err)
			}
		}

		log.Printf("User tenant updated: id=%s, new_tenant_id=%d", clerkUserID, newTenantID)
	}

	h.db.Save(&user)
}

func (h *ClerkWebhookHandler) handleUserDeleted(c *gin.Context, data map[string]interface{}) {
	// Parse Clerk user ID
	clerkUserID, ok := data["id"].(string)
	if !ok || clerkUserID == "" {
		return
	}

	// Delete user by Clerk ID (primary key)
	result := h.db.Where("id = ?", clerkUserID).Delete(&models.User{})
	if result.Error != nil {
		log.Printf("Failed to delete user: clerk_id=%s, error=%v", clerkUserID, result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("User deleted: clerk_id=%s", clerkUserID)
	} else {
		log.Printf("User not found for deletion: clerk_id=%s", clerkUserID)
	}
}

// Helper functions

func (h *ClerkWebhookHandler) extractTenantID(metadata map[string]interface{}) uint {
	if metadata == nil {
		return 0
	}

	// Try to get tenant_id from metadata
	tenantIDRaw, ok := metadata["tenant_id"]
	if !ok {
		return 0
	}

	// Handle different numeric types
	switch v := tenantIDRaw.(type) {
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case string:
		// Parse string to uint if needed
		var id uint
		fmt.Sscanf(v, "%d", &id)
		return id
	default:
		return 0
	}
}

func (h *ClerkWebhookHandler) createDefaultTenant(ctx context.Context, userName, userEmail string) (*models.Tenant, error) {
	// Create a default tenant name from user info
	tenantName := userName
	if tenantName == "" {
		tenantName = userEmail
	}

	tenant := models.Tenant{
		Name:        tenantName,
		PricingPlan: "Starter", // Default plan (free forever with limits)
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(&tenant).Error; err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (h *ClerkWebhookHandler) syncGrafanaOrg(ctx context.Context, tenantID uint) error {
	// Get tenant name
	var tenant models.Tenant
	if err := h.db.First(&tenant, tenantID).Error; err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	// Create or get Grafana organization
	orgID, err := h.grafanaService.SyncTenantOrganization(ctx, tenant.ID, tenant.Name)
	if err != nil {
		return fmt.Errorf("failed to sync Grafana org: %w", err)
	}

	log.Printf("Synced Grafana org: tenant_id=%d, grafana_org_id=%d", tenantID, orgID)
	return nil
}

// UpdateUserMetadata is a helper endpoint to manually set user metadata
// This would be called from your frontend after user signup
func (h *ClerkWebhookHandler) UpdateUserMetadata(c *gin.Context) {
	var req struct {
		UserID   string   `json:"user_id"` // Clerk user ID (preferred)
		Email    string   `json:"email"`   // Fallback lookup by email
		TenantID uint     `json:"tenant_id" binding:"required"`
		Roles    []string `json:"roles"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserID == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id or email required"})
		return
	}

	// Find user by Clerk ID or email
	var user models.User
	var err error
	if req.UserID != "" {
		err = h.db.Where("id = ?", req.UserID).First(&user).Error
	} else {
		err = h.db.Where("email = ?", req.Email).First(&user).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Update tenant_id
	user.TenantID = req.TenantID
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// Sync Grafana org
	if h.grafanaService != nil {
		if err := h.syncGrafanaOrg(c.Request.Context(), req.TenantID); err != nil {
			log.Printf("Failed to sync Grafana org: %v", err)
		}
	}

	// Return metadata that should be set in Clerk
	metadata := map[string]interface{}{
		"tenant_id": req.TenantID,
		"role":      user.Role,
		"roles":     req.Roles,
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "user updated",
		"metadata": metadata,
		"role":     user.Role,
		"note":     "Update this metadata in Clerk via their API or dashboard",
	})
}

// RegisterClerkWebhookRoutes registers webhook routes
func RegisterClerkWebhookRoutes(router *gin.Engine, handler *ClerkWebhookHandler) {
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/clerk", handler.HandleWebhook)
	}

	// Admin endpoint for manual metadata updates
	admin := router.Group("/v1/admin")
	{
		admin.POST("/users/metadata", handler.UpdateUserMetadata)
	}
}
