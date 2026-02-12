// Cloud provider types
export type CloudProvider = 'aws' | 'gcp' | 'azure' | 'oci' | 'custom'

// Pricing tier types
export type PricingTier = 'on_demand' | 'spot' | 'preemptible' | 'reserved_1yr' | 'reserved_3yr'

// Resource types
export type ResourceType = 'cpu' | 'memory' | 'gpu' | 'storage' | 'network'

// Pricing rate
export interface PricingRate {
  id: number
  config_id: number
  resource_type: ResourceType
  pricing_tier: PricingTier
  instance_family?: string
  unit: string
  cost_per_unit: number
  effective_from: string
  effective_to?: string
  created_at: string
}

// Pricing configuration
export interface PricingConfig {
  id: number
  tenant_id: number
  name: string
  provider: CloudProvider
  region?: string
  is_default: boolean
  created_at: string
  updated_at: string
  rates?: PricingRate[]
}

// Cluster pricing assignment
export interface ClusterPricing {
  cluster_name: string
  tenant_id: number
  config_id: number
  created_at: string
  config?: PricingConfig
}

// Provider presets response
export interface ProviderPresets {
  providers: CloudProvider[]
  presets: Record<CloudProvider, Record<string, number>>
}

// Form data for creating/editing config
export interface PricingConfigFormData {
  name: string
  provider: CloudProvider
  region: string
  is_default: boolean
}

// Form data for creating/editing rate
export interface PricingRateFormData {
  resource_type: ResourceType
  pricing_tier: PricingTier
  instance_family: string
  unit: string
  cost_per_unit: number
}

// Display helpers
export const providerDisplayNames: Record<CloudProvider, string> = {
  aws: 'Amazon Web Services',
  gcp: 'Google Cloud Platform',
  azure: 'Microsoft Azure',
  oci: 'Oracle Cloud Infrastructure',
  custom: 'Custom',
}

export const tierDisplayNames: Record<PricingTier, string> = {
  on_demand: 'On-Demand',
  spot: 'Spot',
  preemptible: 'Preemptible',
  reserved_1yr: 'Reserved (1 Year)',
  reserved_3yr: 'Reserved (3 Year)',
}

export const resourceDisplayNames: Record<ResourceType, string> = {
  cpu: 'CPU',
  memory: 'Memory',
  gpu: 'GPU',
  storage: 'Storage',
  network: 'Network',
}

export const defaultUnits: Record<ResourceType, string> = {
  cpu: 'core-hour',
  memory: 'gb-hour',
  gpu: 'gpu-hour',
  storage: 'gb-month',
  network: 'gb',
}
