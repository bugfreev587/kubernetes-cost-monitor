package models

import (
	"time"
)

// CloudProvider represents supported cloud platforms
type CloudProvider string

const (
	ProviderAWS    CloudProvider = "aws"
	ProviderGCP    CloudProvider = "gcp"
	ProviderAzure  CloudProvider = "azure"
	ProviderOCI    CloudProvider = "oci"
	ProviderCustom CloudProvider = "custom"
)

// PricingTier represents different pricing tiers
type PricingTier string

const (
	TierOnDemand    PricingTier = "on_demand"
	TierSpot        PricingTier = "spot"
	TierPreemptible PricingTier = "preemptible" // GCP term for spot
	TierReserved1Yr PricingTier = "reserved_1yr"
	TierReserved3Yr PricingTier = "reserved_3yr"
)

// ResourceType represents types of resources that can be priced
type ResourceType string

const (
	ResourceCPU     ResourceType = "cpu"
	ResourceMemory  ResourceType = "memory"
	ResourceGPU     ResourceType = "gpu"
	ResourceStorage ResourceType = "storage"
	ResourceNetwork ResourceType = "network"
)

// PricingConfig represents a pricing configuration for a cloud provider/region
type PricingConfig struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	TenantID  uint          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name      string        `gorm:"column:name;size:100;not null" json:"name"`
	Provider  CloudProvider `gorm:"column:provider;size:20;not null" json:"provider"`
	Region    string        `gorm:"column:region;size:50" json:"region,omitempty"`
	IsDefault bool          `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedAt time.Time     `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relations
	Rates []PricingRate `gorm:"foreignKey:ConfigID" json:"rates,omitempty"`
}

func (PricingConfig) TableName() string {
	return "pricing_configs"
}

// PricingRate represents a specific resource pricing rate
type PricingRate struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	ConfigID       uint         `gorm:"column:config_id;not null" json:"config_id"`
	ResourceType   ResourceType `gorm:"column:resource_type;size:20;not null" json:"resource_type"`
	PricingTier    PricingTier  `gorm:"column:pricing_tier;size:20;default:on_demand" json:"pricing_tier"`
	InstanceFamily string       `gorm:"column:instance_family;size:50" json:"instance_family,omitempty"`
	Unit           string       `gorm:"column:unit;size:20;not null" json:"unit"`
	CostPerUnit    float64      `gorm:"column:cost_per_unit;type:decimal(12,8);not null" json:"cost_per_unit"`
	EffectiveFrom  time.Time    `gorm:"column:effective_from;type:date" json:"effective_from"`
	EffectiveTo    *time.Time   `gorm:"column:effective_to;type:date" json:"effective_to,omitempty"`
	CreatedAt      time.Time    `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PricingRate) TableName() string {
	return "pricing_rates"
}

// ClusterPricing maps a cluster to a pricing configuration
type ClusterPricing struct {
	ClusterName string    `gorm:"column:cluster_name;primaryKey;size:255" json:"cluster_name"`
	TenantID    uint      `gorm:"column:tenant_id;primaryKey" json:"tenant_id"`
	ConfigID    uint      `gorm:"column:config_id;not null" json:"config_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relations
	Config *PricingConfig `gorm:"foreignKey:ConfigID" json:"config,omitempty"`
}

func (ClusterPricing) TableName() string {
	return "cluster_pricing"
}

// NodePricing provides node-level pricing overrides
type NodePricing struct {
	ID                 uint        `gorm:"primaryKey" json:"id"`
	NodeName           string      `gorm:"column:node_name;size:255;not null" json:"node_name"`
	ClusterName        string      `gorm:"column:cluster_name;size:255;not null" json:"cluster_name"`
	TenantID           uint        `gorm:"column:tenant_id;not null" json:"tenant_id"`
	InstanceType       string      `gorm:"column:instance_type;size:50" json:"instance_type,omitempty"`
	PricingTier        PricingTier `gorm:"column:pricing_tier;size:20;default:on_demand" json:"pricing_tier"`
	HourlyCostOverride *float64    `gorm:"column:hourly_cost_override;type:decimal(10,6)" json:"hourly_cost_override,omitempty"`
	CreatedAt          time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time   `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (NodePricing) TableName() string {
	return "node_pricing"
}

// EffectivePricing holds resolved pricing rates for cost calculation
type EffectivePricing struct {
	CPUPerCoreHour    float64            `json:"cpu_per_core_hour"`
	MemoryPerGBHour   float64            `json:"memory_per_gb_hour"`
	GPUPerHour        map[string]float64 `json:"gpu_per_hour,omitempty"`
	StoragePerGBMonth float64            `json:"storage_per_gb_month,omitempty"`

	// Provider info
	Provider CloudProvider `json:"provider"`
	Region   string        `json:"region,omitempty"`

	// Instance-specific pricing overrides
	InstancePricing map[string]*InstancePrice `json:"instance_pricing,omitempty"`
}

// InstancePrice holds pricing for a specific instance type
type InstancePrice struct {
	InstanceType    string  `json:"instance_type"`
	CPUPerCoreHour  float64 `json:"cpu_per_core_hour"`
	MemoryPerGBHour float64 `json:"memory_per_gb_hour"`
	HourlyCost      float64 `json:"hourly_cost,omitempty"` // Total hourly cost if known
}

// DefaultPricingRates contains default pricing by cloud provider
var DefaultPricingRates = map[CloudProvider]map[string]float64{
	ProviderAWS: {
		"cpu_on_demand":      0.0425,  // $/core-hour (m5 family average)
		"cpu_spot":           0.0128,  // ~70% discount
		"cpu_reserved_1yr":   0.0270,  // ~36% discount
		"cpu_reserved_3yr":   0.0180,  // ~58% discount
		"memory_on_demand":   0.0053,  // $/GB-hour
		"memory_spot":        0.0016,
		"memory_reserved":    0.0034,
		"gpu_p4d":            32.77,   // $/GPU-hour (P4d instances)
		"gpu_g4dn":           0.526,   // $/GPU-hour (G4dn instances)
		"gpu_p3":             3.06,    // $/GPU-hour (P3 instances)
	},
	ProviderGCP: {
		"cpu_on_demand":      0.0350,  // $/core-hour (n1-standard)
		"cpu_spot":           0.0105,  // Preemptible ~70% off
		"cpu_committed_1yr":  0.0220,  // Committed use
		"cpu_committed_3yr":  0.0150,
		"memory_on_demand":   0.0047,  // $/GB-hour
		"memory_spot":        0.0014,
		"gpu_t4":             0.35,    // $/GPU-hour
		"gpu_v100":           2.48,
		"gpu_a100":           2.93,
	},
	ProviderAzure: {
		"cpu_on_demand":      0.0420,
		"cpu_spot":           0.0126,
		"cpu_reserved_1yr":   0.0265,
		"cpu_reserved_3yr":   0.0175,
		"memory_on_demand":   0.0052,
		"memory_spot":        0.0016,
		"gpu_nc6":            0.90,
		"gpu_nc24":           3.60,
	},
	ProviderOCI: {
		"cpu_on_demand":      0.0250,  // OCI is generally cheaper
		"cpu_preemptible":    0.0063,  // 75% discount
		"memory_on_demand":   0.0015,
		"memory_preemptible": 0.0004,
		"gpu_a10":            2.00,
		"gpu_a100":           4.00,
	},
	ProviderCustom: {
		"cpu_on_demand":    0.031611, // Default fallback
		"memory_on_demand": 0.004237,
	},
}

// GetDefaultCPURate returns the default CPU rate for a provider and tier
func GetDefaultCPURate(provider CloudProvider, tier PricingTier) float64 {
	rates, ok := DefaultPricingRates[provider]
	if !ok {
		rates = DefaultPricingRates[ProviderCustom]
	}

	key := "cpu_" + string(tier)
	if rate, ok := rates[key]; ok {
		return rate
	}
	// Fallback to on_demand
	if rate, ok := rates["cpu_on_demand"]; ok {
		return rate
	}
	return 0.031611 // Ultimate fallback
}

// GetDefaultMemoryRate returns the default memory rate for a provider and tier
func GetDefaultMemoryRate(provider CloudProvider, tier PricingTier) float64 {
	rates, ok := DefaultPricingRates[provider]
	if !ok {
		rates = DefaultPricingRates[ProviderCustom]
	}

	key := "memory_" + string(tier)
	if rate, ok := rates[key]; ok {
		return rate
	}
	// Fallback to on_demand
	if rate, ok := rates["memory_on_demand"]; ok {
		return rate
	}
	return 0.004237 // Ultimate fallback
}
