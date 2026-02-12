package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"gorm.io/gorm"
)

// PricingService handles cloud pricing configuration and rate lookups
type PricingService struct {
	db    *gorm.DB
	cache *PricingCache
}

// PricingCache provides in-memory caching for pricing lookups
type PricingCache struct {
	mu      sync.RWMutex
	entries map[string]*pricingCacheEntry
	ttl     time.Duration
}

type pricingCacheEntry struct {
	pricing   *models.EffectivePricing
	expiresAt time.Time
}

// NewPricingService creates a new pricing service
func NewPricingService(db *gorm.DB) *PricingService {
	return &PricingService{
		db: db,
		cache: &PricingCache{
			entries: make(map[string]*pricingCacheEntry),
			ttl:     1 * time.Hour,
		},
	}
}

// GetEffectiveRates returns the effective pricing rates for a cluster
func (s *PricingService) GetEffectiveRates(ctx context.Context, tenantID uint, clusterName string, asOf time.Time) (*models.EffectivePricing, error) {
	// 1. Check cache first
	cacheKey := fmt.Sprintf("%d:%s:%s", tenantID, clusterName, asOf.Format("2006-01-02"))
	if cached := s.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	// 2. Find pricing config for cluster
	var configID uint
	var clusterPricing models.ClusterPricing
	err := s.db.Where("tenant_id = ? AND cluster_name = ?", tenantID, clusterName).
		First(&clusterPricing).Error

	if err == gorm.ErrRecordNotFound {
		// Use tenant's default config
		var defaultConfig models.PricingConfig
		err = s.db.Where("tenant_id = ? AND is_default = true", tenantID).
			First(&defaultConfig).Error
		if err != nil {
			// Fall back to system defaults
			pricing := s.getSystemDefaults(models.ProviderCustom)
			s.cache.Set(cacheKey, pricing)
			return pricing, nil
		}
		configID = defaultConfig.ID
	} else if err != nil {
		return nil, fmt.Errorf("failed to lookup cluster pricing: %w", err)
	} else {
		configID = clusterPricing.ConfigID
	}

	// 3. Load pricing config with rates
	var config models.PricingConfig
	err = s.db.Preload("Rates", "effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)", asOf, asOf).
		First(&config, configID).Error
	if err != nil {
		pricing := s.getSystemDefaults(models.ProviderCustom)
		s.cache.Set(cacheKey, pricing)
		return pricing, nil
	}

	// 4. Build effective pricing from rates
	pricing := s.buildEffectivePricing(&config)

	// 5. Load node-level overrides
	var nodeOverrides []models.NodePricing
	s.db.Where("tenant_id = ? AND cluster_name = ?", tenantID, clusterName).Find(&nodeOverrides)
	s.applyNodeOverrides(pricing, nodeOverrides)

	// 6. Cache and return
	s.cache.Set(cacheKey, pricing)
	return pricing, nil
}

// getSystemDefaults returns default pricing for a provider
func (s *PricingService) getSystemDefaults(provider models.CloudProvider) *models.EffectivePricing {
	return &models.EffectivePricing{
		CPUPerCoreHour:  models.GetDefaultCPURate(provider, models.TierOnDemand),
		MemoryPerGBHour: models.GetDefaultMemoryRate(provider, models.TierOnDemand),
		Provider:        provider,
		GPUPerHour:      make(map[string]float64),
		InstancePricing: make(map[string]*models.InstancePrice),
	}
}

// buildEffectivePricing converts pricing config to effective pricing
func (s *PricingService) buildEffectivePricing(config *models.PricingConfig) *models.EffectivePricing {
	pricing := &models.EffectivePricing{
		Provider:        config.Provider,
		Region:          config.Region,
		GPUPerHour:      make(map[string]float64),
		InstancePricing: make(map[string]*models.InstancePrice),
	}

	// Process rates - prioritize instance-specific rates over generic
	for _, rate := range config.Rates {
		if rate.InstanceFamily != "" {
			// Instance-specific rate
			if _, ok := pricing.InstancePricing[rate.InstanceFamily]; !ok {
				pricing.InstancePricing[rate.InstanceFamily] = &models.InstancePrice{
					InstanceType: rate.InstanceFamily,
				}
			}
			switch rate.ResourceType {
			case models.ResourceCPU:
				pricing.InstancePricing[rate.InstanceFamily].CPUPerCoreHour = rate.CostPerUnit
			case models.ResourceMemory:
				pricing.InstancePricing[rate.InstanceFamily].MemoryPerGBHour = rate.CostPerUnit
			}
		} else {
			// Generic rate (default for this config)
			switch rate.ResourceType {
			case models.ResourceCPU:
				if pricing.CPUPerCoreHour == 0 || rate.PricingTier == models.TierOnDemand {
					pricing.CPUPerCoreHour = rate.CostPerUnit
				}
			case models.ResourceMemory:
				if pricing.MemoryPerGBHour == 0 || rate.PricingTier == models.TierOnDemand {
					pricing.MemoryPerGBHour = rate.CostPerUnit
				}
			case models.ResourceGPU:
				// GPU rates keyed by instance family or generic
				key := "default"
				if rate.InstanceFamily != "" {
					key = rate.InstanceFamily
				}
				pricing.GPUPerHour[key] = rate.CostPerUnit
			case models.ResourceStorage:
				pricing.StoragePerGBMonth = rate.CostPerUnit
			}
		}
	}

	// Apply defaults if no rates were found
	if pricing.CPUPerCoreHour == 0 {
		pricing.CPUPerCoreHour = models.GetDefaultCPURate(config.Provider, models.TierOnDemand)
	}
	if pricing.MemoryPerGBHour == 0 {
		pricing.MemoryPerGBHour = models.GetDefaultMemoryRate(config.Provider, models.TierOnDemand)
	}

	return pricing
}

// applyNodeOverrides applies node-specific pricing
func (s *PricingService) applyNodeOverrides(pricing *models.EffectivePricing, nodes []models.NodePricing) {
	for _, node := range nodes {
		if node.HourlyCostOverride != nil && *node.HourlyCostOverride > 0 {
			pricing.InstancePricing[node.NodeName] = &models.InstancePrice{
				InstanceType: node.InstanceType,
				HourlyCost:   *node.HourlyCostOverride,
			}
		} else if node.InstanceType != "" {
			// Look up instance type pricing
			if instancePrice, ok := pricing.InstancePricing[node.InstanceType]; ok {
				pricing.InstancePricing[node.NodeName] = instancePrice
			}
		}
	}
}

// CreateConfig creates a new pricing configuration
func (s *PricingService) CreateConfig(ctx context.Context, config *models.PricingConfig) error {
	// If setting as default, unset other defaults first
	if config.IsDefault {
		s.db.Model(&models.PricingConfig{}).
			Where("tenant_id = ? AND is_default = true", config.TenantID).
			Update("is_default", false)
	}

	if err := s.db.Create(config).Error; err != nil {
		return err
	}

	// Invalidate cache for this tenant
	s.cache.InvalidateTenant(config.TenantID)
	return nil
}

// UpdateConfig updates an existing pricing configuration
func (s *PricingService) UpdateConfig(ctx context.Context, config *models.PricingConfig) error {
	// If setting as default, unset other defaults first
	if config.IsDefault {
		s.db.Model(&models.PricingConfig{}).
			Where("tenant_id = ? AND is_default = true AND id != ?", config.TenantID, config.ID).
			Update("is_default", false)
	}

	config.UpdatedAt = time.Now()
	if err := s.db.Save(config).Error; err != nil {
		return err
	}

	s.cache.InvalidateTenant(config.TenantID)
	return nil
}

// DeleteConfig deletes a pricing configuration
func (s *PricingService) DeleteConfig(ctx context.Context, configID uint) error {
	var config models.PricingConfig
	if err := s.db.First(&config, configID).Error; err != nil {
		return err
	}

	if err := s.db.Delete(&config).Error; err != nil {
		return err
	}

	s.cache.InvalidateTenant(config.TenantID)
	return nil
}

// GetConfig retrieves a pricing configuration with its rates
func (s *PricingService) GetConfig(ctx context.Context, configID uint) (*models.PricingConfig, error) {
	var config models.PricingConfig
	err := s.db.Preload("Rates").First(&config, configID).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListConfigs lists all pricing configurations for a tenant
func (s *PricingService) ListConfigs(ctx context.Context, tenantID uint) ([]models.PricingConfig, error) {
	var configs []models.PricingConfig
	err := s.db.Where("tenant_id = ?", tenantID).
		Preload("Rates").
		Order("is_default DESC, name ASC").
		Find(&configs).Error
	return configs, err
}

// AddRate adds a pricing rate to a configuration
func (s *PricingService) AddRate(ctx context.Context, rate *models.PricingRate) error {
	if rate.EffectiveFrom.IsZero() {
		rate.EffectiveFrom = time.Now()
	}

	if err := s.db.Create(rate).Error; err != nil {
		return err
	}

	// Get tenant ID for cache invalidation
	var config models.PricingConfig
	if err := s.db.First(&config, rate.ConfigID).Error; err == nil {
		s.cache.InvalidateTenant(config.TenantID)
	}
	return nil
}

// UpdateRate updates an existing pricing rate
func (s *PricingService) UpdateRate(ctx context.Context, rate *models.PricingRate) error {
	if err := s.db.Save(rate).Error; err != nil {
		return err
	}

	var config models.PricingConfig
	if err := s.db.First(&config, rate.ConfigID).Error; err == nil {
		s.cache.InvalidateTenant(config.TenantID)
	}
	return nil
}

// DeleteRate deletes a pricing rate
func (s *PricingService) DeleteRate(ctx context.Context, rateID uint) error {
	var rate models.PricingRate
	if err := s.db.First(&rate, rateID).Error; err != nil {
		return err
	}

	if err := s.db.Delete(&rate).Error; err != nil {
		return err
	}

	var config models.PricingConfig
	if err := s.db.First(&config, rate.ConfigID).Error; err == nil {
		s.cache.InvalidateTenant(config.TenantID)
	}
	return nil
}

// SetClusterPricing assigns a pricing config to a cluster
func (s *PricingService) SetClusterPricing(ctx context.Context, clusterPricing *models.ClusterPricing) error {
	// Upsert
	err := s.db.Save(clusterPricing).Error
	if err != nil {
		return err
	}

	s.cache.InvalidateTenant(clusterPricing.TenantID)
	return nil
}

// GetClusterPricing retrieves pricing config for a cluster
func (s *PricingService) GetClusterPricing(ctx context.Context, tenantID uint, clusterName string) (*models.ClusterPricing, error) {
	var cp models.ClusterPricing
	err := s.db.Preload("Config").
		Where("tenant_id = ? AND cluster_name = ?", tenantID, clusterName).
		First(&cp).Error
	if err != nil {
		return nil, err
	}
	return &cp, nil
}

// ListClusterPricing lists all cluster pricing assignments for a tenant
func (s *PricingService) ListClusterPricing(ctx context.Context, tenantID uint) ([]models.ClusterPricing, error) {
	var cps []models.ClusterPricing
	err := s.db.Preload("Config").
		Where("tenant_id = ?", tenantID).
		Find(&cps).Error
	return cps, err
}

// DeleteClusterPricing removes pricing config assignment for a cluster
func (s *PricingService) DeleteClusterPricing(ctx context.Context, tenantID uint, clusterName string) error {
	result := s.db.Where("tenant_id = ? AND cluster_name = ?", tenantID, clusterName).
		Delete(&models.ClusterPricing{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.cache.InvalidateTenant(tenantID)
	return nil
}

// SetNodePricing sets pricing override for a node
func (s *PricingService) SetNodePricing(ctx context.Context, nodePricing *models.NodePricing) error {
	nodePricing.UpdatedAt = time.Now()

	// Check if exists
	var existing models.NodePricing
	err := s.db.Where("tenant_id = ? AND cluster_name = ? AND node_name = ?",
		nodePricing.TenantID, nodePricing.ClusterName, nodePricing.NodeName).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		if err := s.db.Create(nodePricing).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update existing
		nodePricing.ID = existing.ID
		nodePricing.CreatedAt = existing.CreatedAt
		if err := s.db.Save(nodePricing).Error; err != nil {
			return err
		}
	}

	s.cache.InvalidateTenant(nodePricing.TenantID)
	return nil
}

// GetProviderPresets returns default pricing presets for a provider
func (s *PricingService) GetProviderPresets(provider models.CloudProvider) map[string]float64 {
	if rates, ok := models.DefaultPricingRates[provider]; ok {
		return rates
	}
	return models.DefaultPricingRates[models.ProviderCustom]
}

// ListProviders returns available cloud providers
func (s *PricingService) ListProviders() []models.CloudProvider {
	return []models.CloudProvider{
		models.ProviderAWS,
		models.ProviderGCP,
		models.ProviderAzure,
		models.ProviderOCI,
		models.ProviderCustom,
	}
}

// PricingCache methods

// Get retrieves a cached pricing entry
func (c *PricingCache) Get(key string) *models.EffectivePricing {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.pricing
}

// Set stores a pricing entry in cache
func (c *PricingCache) Set(key string, pricing *models.EffectivePricing) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &pricingCacheEntry{
		pricing:   pricing,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// InvalidateTenant removes all cached entries for a tenant
func (c *PricingCache) InvalidateTenant(tenantID uint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := fmt.Sprintf("%d:", tenantID)
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
}

// Clear removes all cached entries
func (c *PricingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*pricingCacheEntry)
}
