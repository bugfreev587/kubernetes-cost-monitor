package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
)

// getPricingService returns a pricing service instance
func (s *Server) getPricingService() *services.PricingService {
	return services.NewPricingService(s.postgresDB.GetPostgresDB())
}

// GET /v1/pricing/configs
// List all pricing configurations for the tenant
func (s *Server) listPricingConfigs(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	pricingSvc := s.getPricingService()
	configs, err := pricingSvc.ListConfigs(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
	})
}

// POST /v1/pricing/configs
// Create a new pricing configuration
func (s *Server) createPricingConfig(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	var req struct {
		Name      string               `json:"name" binding:"required"`
		Provider  models.CloudProvider `json:"provider" binding:"required"`
		Region    string               `json:"region"`
		IsDefault bool                 `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := &models.PricingConfig{
		TenantID:  tenantID,
		Name:      req.Name,
		Provider:  req.Provider,
		Region:    req.Region,
		IsDefault: req.IsDefault,
	}

	pricingSvc := s.getPricingService()
	if err := pricingSvc.CreateConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"config": config,
	})
}

// GET /v1/pricing/configs/:id
// Get a pricing configuration with its rates
func (s *Server) getPricingConfig(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	configID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	pricingSvc := s.getPricingService()
	config, err := pricingSvc.GetConfig(c.Request.Context(), uint(configID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// Verify tenant ownership
	if config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// PUT /v1/pricing/configs/:id
// Update a pricing configuration
func (s *Server) updatePricingConfig(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	configID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	pricingSvc := s.getPricingService()

	// Get existing config
	config, err := pricingSvc.GetConfig(c.Request.Context(), uint(configID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// Verify tenant ownership
	if config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		Name      string               `json:"name"`
		Provider  models.CloudProvider `json:"provider"`
		Region    string               `json:"region"`
		IsDefault *bool                `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Provider != "" {
		config.Provider = req.Provider
	}
	if req.Region != "" {
		config.Region = req.Region
	}
	if req.IsDefault != nil {
		config.IsDefault = *req.IsDefault
	}

	if err := pricingSvc.UpdateConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// DELETE /v1/pricing/configs/:id
// Delete a pricing configuration
func (s *Server) deletePricingConfig(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	configID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	pricingSvc := s.getPricingService()

	// Get config to verify ownership
	config, err := pricingSvc.GetConfig(c.Request.Context(), uint(configID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := pricingSvc.DeleteConfig(c.Request.Context(), uint(configID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config deleted"})
}

// POST /v1/pricing/configs/:id/rates
// Add a pricing rate to a configuration
func (s *Server) addPricingRate(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	configID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	pricingSvc := s.getPricingService()

	// Verify config ownership
	config, err := pricingSvc.GetConfig(c.Request.Context(), uint(configID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	if config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		ResourceType   models.ResourceType `json:"resource_type" binding:"required"`
		PricingTier    models.PricingTier  `json:"pricing_tier"`
		InstanceFamily string              `json:"instance_family"`
		Unit           string              `json:"unit" binding:"required"`
		CostPerUnit    float64             `json:"cost_per_unit" binding:"required"`
		EffectiveFrom  *time.Time          `json:"effective_from"`
		EffectiveTo    *time.Time          `json:"effective_to"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rate := &models.PricingRate{
		ConfigID:       uint(configID),
		ResourceType:   req.ResourceType,
		PricingTier:    req.PricingTier,
		InstanceFamily: req.InstanceFamily,
		Unit:           req.Unit,
		CostPerUnit:    req.CostPerUnit,
		EffectiveTo:    req.EffectiveTo,
	}

	if req.EffectiveFrom != nil {
		rate.EffectiveFrom = *req.EffectiveFrom
	} else {
		rate.EffectiveFrom = time.Now()
	}

	if rate.PricingTier == "" {
		rate.PricingTier = models.TierOnDemand
	}

	if err := pricingSvc.AddRate(c.Request.Context(), rate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"rate": rate,
	})
}

// PUT /v1/pricing/rates/:id
// Update a pricing rate
func (s *Server) updatePricingRate(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	rateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rate ID"})
		return
	}

	pricingSvc := s.getPricingService()

	// Get rate and verify ownership
	var rate models.PricingRate
	if err := s.postgresDB.GetPostgresDB().First(&rate, rateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate not found"})
		return
	}

	config, err := pricingSvc.GetConfig(c.Request.Context(), rate.ConfigID)
	if err != nil || config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		CostPerUnit   *float64   `json:"cost_per_unit"`
		EffectiveFrom *time.Time `json:"effective_from"`
		EffectiveTo   *time.Time `json:"effective_to"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.CostPerUnit != nil {
		rate.CostPerUnit = *req.CostPerUnit
	}
	if req.EffectiveFrom != nil {
		rate.EffectiveFrom = *req.EffectiveFrom
	}
	if req.EffectiveTo != nil {
		rate.EffectiveTo = req.EffectiveTo
	}

	if err := pricingSvc.UpdateRate(c.Request.Context(), &rate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rate": rate,
	})
}

// DELETE /v1/pricing/rates/:id
// Delete a pricing rate
func (s *Server) deletePricingRate(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	rateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rate ID"})
		return
	}

	pricingSvc := s.getPricingService()

	// Get rate and verify ownership
	var rate models.PricingRate
	if err := s.postgresDB.GetPostgresDB().First(&rate, rateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate not found"})
		return
	}

	config, err := pricingSvc.GetConfig(c.Request.Context(), rate.ConfigID)
	if err != nil || config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := pricingSvc.DeleteRate(c.Request.Context(), uint(rateID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rate deleted"})
}

// PUT /v1/clusters/:name/pricing
// Assign a pricing config to a cluster
func (s *Server) setClusterPricing(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	clusterName := c.Param("name")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster name required"})
		return
	}

	var req struct {
		ConfigID uint `json:"config_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pricingSvc := s.getPricingService()

	// Verify config ownership
	config, err := pricingSvc.GetConfig(c.Request.Context(), req.ConfigID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	if config.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	cp := &models.ClusterPricing{
		ClusterName: clusterName,
		TenantID:    tenantID,
		ConfigID:    req.ConfigID,
	}

	if err := pricingSvc.SetClusterPricing(c.Request.Context(), cp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_pricing": cp,
	})
}

// GET /v1/clusters/:name/pricing
// Get pricing config for a cluster
func (s *Server) getClusterPricing(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	clusterName := c.Param("name")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster name required"})
		return
	}

	pricingSvc := s.getPricingService()
	cp, err := pricingSvc.GetClusterPricing(c.Request.Context(), tenantID, clusterName)
	if err != nil {
		// Return effective rates even if no explicit mapping
		rates, _ := pricingSvc.GetEffectiveRates(c.Request.Context(), tenantID, clusterName, time.Now())
		c.JSON(http.StatusOK, gin.H{
			"cluster_name":     clusterName,
			"config":           nil,
			"effective_rates": rates,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_pricing": cp,
	})
}

// GET /v1/pricing/presets
// Get default pricing presets for all providers
func (s *Server) getPricingPresets(c *gin.Context) {
	provider := c.Query("provider")

	pricingSvc := s.getPricingService()

	if provider != "" {
		rates := pricingSvc.GetProviderPresets(models.CloudProvider(provider))
		c.JSON(http.StatusOK, gin.H{
			"provider": provider,
			"rates":    rates,
		})
		return
	}

	// Return all providers
	presets := make(map[string]map[string]float64)
	for _, p := range pricingSvc.ListProviders() {
		presets[string(p)] = pricingSvc.GetProviderPresets(p)
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": pricingSvc.ListProviders(),
		"presets":   presets,
	})
}

// POST /v1/pricing/import/:provider
// Import default pricing for a provider
func (s *Server) importProviderPricing(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	provider := models.CloudProvider(c.Param("provider"))
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider required"})
		return
	}

	var req struct {
		Name      string `json:"name" binding:"required"`
		Region    string `json:"region"`
		IsDefault bool   `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pricingSvc := s.getPricingService()

	// Create config
	config := &models.PricingConfig{
		TenantID:  tenantID,
		Name:      req.Name,
		Provider:  provider,
		Region:    req.Region,
		IsDefault: req.IsDefault,
	}

	if err := pricingSvc.CreateConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add default rates from presets
	presets := pricingSvc.GetProviderPresets(provider)
	var rates []models.PricingRate

	// Add CPU rates
	if cpuOnDemand, ok := presets["cpu_on_demand"]; ok {
		rate := models.PricingRate{
			ConfigID:      config.ID,
			ResourceType:  models.ResourceCPU,
			PricingTier:   models.TierOnDemand,
			Unit:          "core-hour",
			CostPerUnit:   cpuOnDemand,
			EffectiveFrom: time.Now(),
		}
		pricingSvc.AddRate(c.Request.Context(), &rate)
		rates = append(rates, rate)
	}

	if cpuSpot, ok := presets["cpu_spot"]; ok {
		rate := models.PricingRate{
			ConfigID:      config.ID,
			ResourceType:  models.ResourceCPU,
			PricingTier:   models.TierSpot,
			Unit:          "core-hour",
			CostPerUnit:   cpuSpot,
			EffectiveFrom: time.Now(),
		}
		pricingSvc.AddRate(c.Request.Context(), &rate)
		rates = append(rates, rate)
	}

	// Add Memory rates
	if memOnDemand, ok := presets["memory_on_demand"]; ok {
		rate := models.PricingRate{
			ConfigID:      config.ID,
			ResourceType:  models.ResourceMemory,
			PricingTier:   models.TierOnDemand,
			Unit:          "gb-hour",
			CostPerUnit:   memOnDemand,
			EffectiveFrom: time.Now(),
		}
		pricingSvc.AddRate(c.Request.Context(), &rate)
		rates = append(rates, rate)
	}

	if memSpot, ok := presets["memory_spot"]; ok {
		rate := models.PricingRate{
			ConfigID:      config.ID,
			ResourceType:  models.ResourceMemory,
			PricingTier:   models.TierSpot,
			Unit:          "gb-hour",
			CostPerUnit:   memSpot,
			EffectiveFrom: time.Now(),
		}
		pricingSvc.AddRate(c.Request.Context(), &rate)
		rates = append(rates, rate)
	}

	config.Rates = rates

	c.JSON(http.StatusCreated, gin.H{
		"config": config,
		"message": "imported default pricing for " + string(provider),
	})
}
