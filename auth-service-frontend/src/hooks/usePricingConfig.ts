import { useState, useEffect, useCallback } from 'react'
import type {
  PricingConfig,
  PricingRate,
  ClusterPricing,
  ProviderPresets,
  PricingConfigFormData,
  PricingRateFormData,
  CloudProvider,
} from '../types/pricing'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

interface UsePricingConfigResult {
  // Data
  configs: PricingConfig[]
  presets: ProviderPresets | null
  clusterPricings: ClusterPricing[]

  // State
  loading: boolean
  error: string | null

  // Actions
  refresh: () => Promise<void>
  createConfig: (data: PricingConfigFormData) => Promise<PricingConfig>
  updateConfig: (id: number, data: Partial<PricingConfigFormData>) => Promise<PricingConfig>
  deleteConfig: (id: number) => Promise<void>
  addRate: (configId: number, data: PricingRateFormData) => Promise<PricingRate>
  updateRate: (rateId: number, data: Partial<PricingRateFormData>) => Promise<PricingRate>
  deleteRate: (rateId: number) => Promise<void>
  importProviderDefaults: (provider: CloudProvider, name: string, region?: string) => Promise<PricingConfig>
  setClusterPricing: (clusterName: string, configId: number) => Promise<ClusterPricing>
  getClusterPricing: (clusterName: string) => Promise<ClusterPricing | null>
}

export function usePricingConfig(): UsePricingConfigResult {
  const [configs, setConfigs] = useState<PricingConfig[]>([])
  const [presets, setPresets] = useState<ProviderPresets | null>(null)
  const [clusterPricings] = useState<ClusterPricing[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const getHeaders = useCallback(() => {
    const userId = localStorage.getItem('user_id')
    return {
      'Content-Type': 'application/json',
      'X-User-ID': userId || '',
    }
  }, [])

  const handleError = (err: unknown): string => {
    if (err instanceof Error) return err.message
    return 'An unexpected error occurred'
  }

  // Fetch all pricing configs
  const fetchConfigs = useCallback(async () => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/pricing/configs`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to fetch pricing configs')
      }
      const data = await response.json()
      setConfigs(data.configs || [])
    } catch (err) {
      throw err
    }
  }, [getHeaders])

  // Fetch provider presets
  const fetchPresets = useCallback(async () => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/pricing/presets`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to fetch presets')
      }
      const data = await response.json()
      setPresets(data)
    } catch (err) {
      throw err
    }
  }, [getHeaders])

  // Refresh all data
  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await Promise.all([fetchConfigs(), fetchPresets()])
    } catch (err) {
      setError(handleError(err))
    } finally {
      setLoading(false)
    }
  }, [fetchConfigs, fetchPresets])

  // Initial load
  useEffect(() => {
    const userId = localStorage.getItem('user_id')
    if (userId) {
      refresh()
    }
  }, [refresh])

  // Create a new pricing config
  const createConfig = useCallback(async (data: PricingConfigFormData): Promise<PricingConfig> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/configs`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify(data),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to create config')
    }
    const result = await response.json()
    await fetchConfigs()
    return result.config
  }, [getHeaders, fetchConfigs])

  // Update a pricing config
  const updateConfig = useCallback(async (id: number, data: Partial<PricingConfigFormData>): Promise<PricingConfig> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/configs/${id}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify(data),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to update config')
    }
    const result = await response.json()
    await fetchConfigs()
    return result.config
  }, [getHeaders, fetchConfigs])

  // Delete a pricing config
  const deleteConfig = useCallback(async (id: number): Promise<void> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/configs/${id}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to delete config')
    }
    await fetchConfigs()
  }, [getHeaders, fetchConfigs])

  // Add a rate to a config
  const addRate = useCallback(async (configId: number, data: PricingRateFormData): Promise<PricingRate> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/configs/${configId}/rates`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify(data),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to add rate')
    }
    const result = await response.json()
    await fetchConfigs()
    return result.rate
  }, [getHeaders, fetchConfigs])

  // Update a rate
  const updateRate = useCallback(async (rateId: number, data: Partial<PricingRateFormData>): Promise<PricingRate> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/rates/${rateId}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify(data),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to update rate')
    }
    const result = await response.json()
    await fetchConfigs()
    return result.rate
  }, [getHeaders, fetchConfigs])

  // Delete a rate
  const deleteRate = useCallback(async (rateId: number): Promise<void> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/rates/${rateId}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to delete rate')
    }
    await fetchConfigs()
  }, [getHeaders, fetchConfigs])

  // Import provider defaults
  const importProviderDefaults = useCallback(async (
    provider: CloudProvider,
    name: string,
    region?: string
  ): Promise<PricingConfig> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/pricing/import/${provider}`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ name, region }),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to import provider defaults')
    }
    const result = await response.json()
    await fetchConfigs()
    return result.config
  }, [getHeaders, fetchConfigs])

  // Set cluster pricing
  const setClusterPricing = useCallback(async (clusterName: string, configId: number): Promise<ClusterPricing> => {
    const response = await fetch(`${API_SERVER_URL}/v1/admin/clusters/${encodeURIComponent(clusterName)}/pricing`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify({ config_id: configId }),
    })
    if (!response.ok) {
      const respData = await response.json()
      throw new Error(respData.error || 'Failed to set cluster pricing')
    }
    const result = await response.json()
    return result.cluster_pricing
  }, [getHeaders])

  // Get cluster pricing
  const getClusterPricing = useCallback(async (clusterName: string): Promise<ClusterPricing | null> => {
    try {
      const response = await fetch(`${API_SERVER_URL}/v1/clusters/${encodeURIComponent(clusterName)}/pricing`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        if (response.status === 404) return null
        const respData = await response.json()
        throw new Error(respData.error || 'Failed to get cluster pricing')
      }
      const result = await response.json()
      return result.cluster_pricing
    } catch {
      return null
    }
  }, [getHeaders])

  return {
    configs,
    presets,
    clusterPricings,
    loading,
    error,
    refresh,
    createConfig,
    updateConfig,
    deleteConfig,
    addRate,
    updateRate,
    deleteRate,
    importProviderDefaults,
    setClusterPricing,
    getClusterPricing,
  }
}
