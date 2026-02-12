package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // New import for interfaces
	"github.com/bugfreev587/k8s-cost-api-server/internal/config"
	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services" // Still needed for apiKeySvc

	// New import for HealthCheckResponse
	"github.com/gin-gonic/gin"
)

type Server struct {
	serverConfig        *config.ServerCfg
	postgresDB          app_interfaces.PostgresService
	timescaleDB         app_interfaces.TimescaleService
	redisClient         app_interfaces.RedisService
	apiKeySvc           *services.APIKeyService
	planSvc             *services.PlanService
	clerkSvc            *services.ClerkService
	grafanaSvc          *services.GrafanaService
	clerkWebhookHandler *ClerkWebhookHandler
	rbac                *middleware.RBACMiddleware
	router              *gin.Engine
}

func NewServer(cfg *config.Config, postgresDB app_interfaces.PostgresService, timescaleDB app_interfaces.TimescaleService, redisClient app_interfaces.RedisService, apiKeySvc *services.APIKeyService, planSvc *services.PlanService) *Server {
	if cfg.Environment == "production" || cfg.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize Grafana service for multi-tenant org management
	var grafanaSvc *services.GrafanaService
	if cfg.Grafana.URL != "" {
		grafanaSvc = services.NewGrafanaService(cfg.Grafana.URL, cfg.Grafana.APIToken, cfg.Grafana.Username, cfg.Grafana.Password)
		log.Printf("Grafana service initialized: %s", cfg.Grafana.URL)
	} else {
		log.Printf("WARNING: Grafana service not configured - GRAFANA_URL environment variable not set")
	}

	// Initialize Clerk webhook handler with Grafana service
	clerkWebhookHandler := NewClerkWebhookHandler(postgresDB.GetPostgresDB(), grafanaSvc)

	// Initialize Clerk service for invitation emails
	clerkSvc := services.NewClerkService(cfg.Clerk.SecretKey, cfg.Clerk.FrontendURL)
	if clerkSvc.IsConfigured() {
		log.Printf("Clerk service initialized successfully (secret key configured)")
	} else {
		log.Printf("WARNING: Clerk service not configured - CLERK_SECRET_KEY environment variable not set")
	}

	// Initialize RBAC middleware
	rbacMiddleware := middleware.NewRBACMiddleware(postgresDB.GetPostgresDB())

	server := &Server{
		serverConfig:        &cfg.Server,
		postgresDB:          postgresDB,
		timescaleDB:         timescaleDB,
		redisClient:         redisClient,
		apiKeySvc:           apiKeySvc,
		planSvc:             planSvc,
		clerkSvc:            clerkSvc,
		grafanaSvc:          grafanaSvc,
		clerkWebhookHandler: clerkWebhookHandler,
		rbac:                rbacMiddleware,
		router:              router,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(LoggerMiddleware())

	// CORS middleware
	s.router.Use(CORSMiddleware(s.serverConfig.CORSOrigins))

	// Rate limiting
	s.router.Use(RateLimitMiddleware(s.serverConfig.RateLimitPerMinute))
}

func (s *Server) setupRoutes() {
	// API key middleware for agent authentication
	apiKeyAuth := middleware.NewAPIKeyMiddleware(s.apiKeySvc)

	// ===========================================
	// PUBLIC ROUTES (no authentication required)
	// ===========================================

	// Health check
	s.router.GET("/v1/health", s.healthCheckHandler())

	// Public pricing plans (for pricing page)
	s.router.GET("/v1/plans", s.listPricingPlansHandler())

	// Auth endpoints (user sync from frontend after Clerk auth)
	s.router.POST("/v1/auth/sync", s.syncUserHandler())

	// Clerk webhooks (verified by Clerk signature)
	s.router.POST("/webhooks/clerk", s.clerkWebhookHandler.HandleWebhook)

	// ===========================================
	// AGENT ROUTES (API key authentication only)
	// ===========================================

	// Metrics ingestion - API key required (agent sends metrics)
	s.router.POST("/v1/ingest", apiKeyAuth, s.makeIngestHandler())

	// ===========================================
	// VIEWER+ ROUTES (authenticated user, any role)
	// ===========================================

	// Dashboard routes group - require user auth + viewer role
	dashboard := s.router.Group("/v1")
	dashboard.Use(s.rbac.RequireUser(), s.rbac.RequireViewer())
	{
		// Cost data - read only (viewer can access)
		dashboard.GET("/costs/namespaces", s.getCostsByNamespace)
		dashboard.GET("/costs/clusters", s.getCostsByCluster)
		dashboard.GET("/costs/utilization", s.getUtilizationVsRequests)
		dashboard.GET("/costs/trends", s.getCostTrends)

		// Allocation data - read only
		dashboard.GET("/allocation", s.getAllocation)
		dashboard.GET("/allocation/compute", s.getAllocationCompute)
		dashboard.GET("/allocation/summary", s.getAllocationSummary)
		dashboard.GET("/allocation/summary/topline", s.getAllocationTopline)

		// Recommendations - read only
		dashboard.GET("/recommendations", s.getRecommendations)

		// User management - view team members
		dashboard.GET("/users", s.listUsersHandler())
		dashboard.GET("/users/:user_id", s.getUserHandler())

		// Pricing - read only
		dashboard.GET("/pricing/configs", s.listPricingConfigs)
		dashboard.GET("/pricing/configs/:id", s.getPricingConfig)
		dashboard.GET("/pricing/presets", s.getPricingPresets)
		dashboard.GET("/clusters/:name/pricing", s.getClusterPricing)
	}

	// ===========================================
	// EDITOR+ ROUTES (editor, admin, owner)
	// ===========================================

	editor := s.router.Group("/v1")
	editor.Use(s.rbac.RequireUser(), s.rbac.RequireEditor())
	{
		// Recommendations - can generate, apply, dismiss
		editor.POST("/recommendations/generate", s.generateRecommendations)
		editor.POST("/recommendations/:id/apply", s.applyRecommendation)
		editor.POST("/recommendations/:id/dismiss", s.dismissRecommendation)
	}

	// ===========================================
	// ADMIN+ ROUTES (admin, owner)
	// ===========================================

	admin := s.router.Group("/v1/admin")
	admin.Use(s.rbac.RequireUser(), s.rbac.RequireAdmin())
	{
		// API key management
		admin.POST("/api_keys", s.makeCreateAPIKeyHandler())
		admin.GET("/api_keys", s.listAPIKeysHandler())
		admin.DELETE("/api_keys/:key_id", s.revokeAPIKeyHandler())
		admin.DELETE("/api_keys/:key_id/permanent", s.deleteAPIKeyHandler())

		// Tenant management (view)
		admin.GET("/tenants/:tenant_id/pricing-plan", s.rbac.RequireTenantAccess("tenant_id"), s.getTenantPricingPlanHandler())
		admin.GET("/tenants/:tenant_id/usage", s.rbac.RequireTenantAccess("tenant_id"), s.getTenantUsageHandler())

		// User management - invite, suspend, promote to editor
		admin.POST("/users/invite", s.inviteUserHandler())
		admin.PATCH("/users/:user_id/suspend", s.suspendUserHandler())
		admin.PATCH("/users/:user_id/unsuspend", s.unsuspendUserHandler())
		admin.PATCH("/users/:user_id/role", s.updateUserRoleHandler())
		admin.DELETE("/users/:user_id", s.removeUserHandler())

		// Pricing configuration management
		admin.POST("/pricing/configs", s.createPricingConfig)
		admin.PUT("/pricing/configs/:id", s.updatePricingConfig)
		admin.DELETE("/pricing/configs/:id", s.deletePricingConfig)
		admin.POST("/pricing/configs/:id/rates", s.addPricingRate)
		admin.PUT("/pricing/rates/:id", s.updatePricingRate)
		admin.DELETE("/pricing/rates/:id", s.deletePricingRate)
		admin.PUT("/clusters/:name/pricing", s.setClusterPricing)
		admin.POST("/pricing/import/:provider", s.importProviderPricing)
	}

	// ===========================================
	// OWNER ONLY ROUTES
	// ===========================================

	owner := s.router.Group("/v1/owner")
	owner.Use(s.rbac.RequireUser(), s.rbac.RequireOwner())
	{
		// Pricing plan changes (billing)
		owner.PATCH("/tenants/:tenant_id/pricing-plan", s.rbac.RequireTenantAccess("tenant_id"), s.updateTenantPricingPlanHandler())

		// Admin management - only owner can promote to admin or remove admins
		owner.POST("/users/:user_id/promote-admin", s.promoteToAdminHandler())
		owner.DELETE("/users/:user_id/demote-admin", s.demoteAdminHandler())

		// Transfer ownership
		owner.POST("/transfer-ownership", s.transferOwnershipHandler())

		// Delete tenant (danger zone)
		owner.DELETE("/tenants/:tenant_id", s.rbac.RequireTenantAccess("tenant_id"), s.deleteTenantHandler())
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%s", s.serverConfig.Host, s.serverConfig.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}
