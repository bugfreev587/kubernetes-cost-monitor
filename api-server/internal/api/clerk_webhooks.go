package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	name := fmt.Sprintf("%s %s", firstName, lastName)

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

	// Create user in database
	user := models.User{
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
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

	log.Printf("User created: email=%s, tenant_id=%d", email, tenantID)
}

func (h *ClerkWebhookHandler) handleUserUpdated(c *gin.Context, data map[string]interface{}) {
	ctx := c.Request.Context()

	// Parse email
	emailAddresses, ok := data["email_addresses"].([]interface{})
	if !ok || len(emailAddresses) == 0 {
		return
	}

	emailData, ok := emailAddresses[0].(map[string]interface{})
	if !ok {
		return
	}

	email, ok := emailData["email_address"].(string)
	if !ok || email == "" {
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		log.Printf("User not found for update: %s", email)
		return
	}

	// Update name if changed
	firstName, _ := data["first_name"].(string)
	lastName, _ := data["last_name"].(string)
	newName := fmt.Sprintf("%s %s", firstName, lastName)

	if newName != user.Name {
		user.Name = newName
		h.db.Save(&user)
	}

	// Check if tenant_id changed in metadata
	publicMetadata, _ := data["public_metadata"].(map[string]interface{})
	newTenantID := h.extractTenantID(publicMetadata)

	if newTenantID != 0 && newTenantID != user.TenantID {
		// Tenant changed - update user and sync Grafana
		user.TenantID = newTenantID
		h.db.Save(&user)

		if h.grafanaService != nil {
			if err := h.syncGrafanaOrg(ctx, newTenantID); err != nil {
				log.Printf("Failed to sync Grafana org: %v", err)
			}
		}

		log.Printf("User tenant updated: email=%s, new_tenant_id=%d", email, newTenantID)
	}
}

func (h *ClerkWebhookHandler) handleUserDeleted(c *gin.Context, data map[string]interface{}) {
	// Parse user ID
	userID, ok := data["id"].(string)
	if !ok || userID == "" {
		return
	}

	// For now, we'll soft-delete or mark as inactive
	// In production, you might want to keep the user for audit purposes
	log.Printf("User deleted webhook received: clerk_user_id=%s", userID)

	// Note: You'd need to store clerk_user_id in your User model to delete by it
	// For now, this is a placeholder
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
		PricingPlan: "Basic", // Default plan
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
		Email    string   `json:"email" binding:"required"`
		TenantID uint     `json:"tenant_id" binding:"required"`
		Roles    []string `json:"roles"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
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
		"roles":     req.Roles,
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "user updated",
		"metadata": metadata,
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
