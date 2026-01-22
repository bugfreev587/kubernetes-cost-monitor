package api

import (
	"context"
	"fmt"
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
	clerkWebhookHandler *ClerkWebhookHandler
	router              *gin.Engine
}

func NewServer(cfg *config.Config, postgresDB app_interfaces.PostgresService, timescaleDB app_interfaces.TimescaleService, redisClient app_interfaces.RedisService, apiKeySvc *services.APIKeyService, planSvc *services.PlanService) *Server {
	if cfg.Environment == "production" || cfg.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize Clerk webhook handler (grafanaService is nil for now)
	clerkWebhookHandler := NewClerkWebhookHandler(postgresDB.GetPostgresDB(), nil)

	server := &Server{
		serverConfig:        &cfg.Server,
		postgresDB:          postgresDB,
		timescaleDB:         timescaleDB,
		redisClient:         redisClient,
		apiKeySvc:           apiKeySvc,
		planSvc:             planSvc,
		clerkWebhookHandler: clerkWebhookHandler,
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

	// --- health ---
	s.router.GET("/v1/health", s.healthCheckHandler())

	// --- public pricing plans ---
	s.router.GET("/v1/plans", s.listPricingPlansHandler())

	// --- admin ---
	s.router.POST("/v1/admin/api_keys", s.makeCreateAPIKeyHandler())
	s.router.GET("/v1/admin/tenants/:tenant_id/pricing-plan", s.getTenantPricingPlanHandler())
	s.router.PATCH("/v1/admin/tenants/:tenant_id/pricing-plan", s.updateTenantPricingPlanHandler())
	s.router.GET("/v1/admin/tenants/:tenant_id/usage", s.getTenantUsageHandler())

	// --- agent ingest (protected with API Key Middleware) ---
	authMiddleware := middleware.NewAPIKeyMiddleware(s.apiKeySvc)
	s.router.POST("/v1/ingest", authMiddleware, s.makeIngestHandler())

	// --- costs (protected with API Key Middleware) ---
	costAuthMiddleware := middleware.NewAPIKeyMiddleware(s.apiKeySvc)
	s.router.GET("/v1/costs/namespaces", costAuthMiddleware, s.getCostsByNamespace)
	s.router.GET("/v1/costs/clusters", costAuthMiddleware, s.getCostsByCluster)
	s.router.GET("/v1/costs/utilization", costAuthMiddleware, s.getUtilizationVsRequests)
	s.router.GET("/v1/costs/trends", costAuthMiddleware, s.getCostTrends)

	// --- allocation (OpenCost-compatible API, protected with API Key Middleware) ---
	allocAuthMiddleware := middleware.NewAPIKeyMiddleware(s.apiKeySvc)
	s.router.GET("/v1/allocation", allocAuthMiddleware, s.getAllocation)
	s.router.GET("/v1/allocation/compute", allocAuthMiddleware, s.getAllocationCompute)
	s.router.GET("/v1/allocation/summary", allocAuthMiddleware, s.getAllocationSummary)
	s.router.GET("/v1/allocation/summary/topline", allocAuthMiddleware, s.getAllocationTopline)

	// --- recommendations (protected with API Key Middleware) ---
	recAuthMiddleware := middleware.NewAPIKeyMiddleware(s.apiKeySvc)
	s.router.GET("/v1/recommendations", recAuthMiddleware, s.getRecommendations)
	s.router.POST("/v1/recommendations/generate", recAuthMiddleware, s.generateRecommendations)
	s.router.POST("/v1/recommendations/:id/apply", recAuthMiddleware, s.applyRecommendation)
	s.router.POST("/v1/recommendations/:id/dismiss", recAuthMiddleware, s.dismissRecommendation)

	// --- Clerk webhooks (for user signup/update/delete) ---
	s.router.POST("/webhooks/clerk", s.clerkWebhookHandler.HandleWebhook)

	// --- admin user metadata endpoint ---
	s.router.POST("/v1/admin/users/metadata", s.clerkWebhookHandler.UpdateUserMetadata)

	// --- auth endpoints (user sync from frontend) ---
	s.router.POST("/v1/auth/sync", s.syncUserHandler())
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
