import { useState, useEffect, useCallback } from 'react'
import type {
  ToplineSummary,
  NamespaceAllocation,
  CostTrend,
  UtilizationMetric,
  Recommendation,
  TimeWindow,
} from '../types/cost'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

// Set to true to use mock data for UI testing
const USE_MOCK_DATA = false

// Mock data for UI testing
const generateMockData = (window: TimeWindow) => {
  const days = window === '7d' ? 7 : 30
  const multiplier = window === '7d' ? 1 : 4.3

  const mockTopline: ToplineSummary = {
    total_cost: 1247.83 * multiplier,
    cpu_cost: 723.45 * multiplier,
    memory_cost: 524.38 * multiplier,
    efficiency: 68.5,
    window: window,
  }

  const mockNamespaceAllocations: NamespaceAllocation[] = [
    { namespace: 'production', cpu_cost: 312.50, memory_cost: 187.30, total_cost: 499.80, cpu_hours: 2500, memory_gb_hours: 1500 },
    { namespace: 'staging', cpu_cost: 156.25, memory_cost: 93.65, total_cost: 249.90, cpu_hours: 1250, memory_gb_hours: 750 },
    { namespace: 'development', cpu_cost: 104.17, memory_cost: 62.43, total_cost: 166.60, cpu_hours: 833, memory_gb_hours: 500 },
    { namespace: 'monitoring', cpu_cost: 78.13, memory_cost: 46.82, total_cost: 124.95, cpu_hours: 625, memory_gb_hours: 375 },
    { namespace: 'logging', cpu_cost: 52.08, memory_cost: 31.22, total_cost: 83.30, cpu_hours: 417, memory_gb_hours: 250 },
    { namespace: 'kube-system', cpu_cost: 39.06, memory_cost: 23.41, total_cost: 62.47, cpu_hours: 312, memory_gb_hours: 187 },
    { namespace: 'ingress-nginx', cpu_cost: 26.04, memory_cost: 15.61, total_cost: 41.65, cpu_hours: 208, memory_gb_hours: 125 },
    { namespace: 'cert-manager', cpu_cost: 13.02, memory_cost: 7.80, total_cost: 20.82, cpu_hours: 104, memory_gb_hours: 62 },
  ]

  // Generate trend data for the past N days
  const mockTrends: CostTrend[] = []
  const today = new Date()
  for (let i = days - 1; i >= 0; i--) {
    const date = new Date(today)
    date.setDate(date.getDate() - i)
    const baseValue = 150 + Math.random() * 50
    mockTrends.push({
      date: date.toISOString().split('T')[0],
      cost: baseValue + Math.random() * 30,
      cpu_cost: baseValue * 0.58 + Math.random() * 15,
      memory_cost: baseValue * 0.42 + Math.random() * 10,
    })
  }

  const mockUtilization: UtilizationMetric[] = [
    { pod_name: 'api-server-7d8f9c6b5-x2k4m', namespace: 'production', cluster: 'prod-us-east', cpu_utilization: 78.5, memory_utilization: 65.2, cpu_requested: 1000, cpu_used: 785, memory_requested: 2048, memory_used: 1335, estimated_cost: 45.67 },
    { pod_name: 'worker-processor-5c4d3b2a1-n7p8q', namespace: 'production', cluster: 'prod-us-east', cpu_utilization: 45.2, memory_utilization: 82.1, cpu_requested: 2000, cpu_used: 904, memory_requested: 4096, memory_used: 3363, estimated_cost: 38.92 },
    { pod_name: 'postgres-primary-0', namespace: 'production', cluster: 'prod-us-east', cpu_utilization: 62.8, memory_utilization: 71.5, cpu_requested: 4000, cpu_used: 2512, memory_requested: 8192, memory_used: 5857, estimated_cost: 35.18 },
    { pod_name: 'redis-master-0', namespace: 'production', cluster: 'prod-us-east', cpu_utilization: 23.4, memory_utilization: 89.3, cpu_requested: 1000, cpu_used: 234, memory_requested: 2048, memory_used: 1829, estimated_cost: 28.45 },
    { pod_name: 'nginx-ingress-controller-8f7e6d5c-w3x4y', namespace: 'ingress-nginx', cluster: 'prod-us-east', cpu_utilization: 34.7, memory_utilization: 41.2, cpu_requested: 500, cpu_used: 173, memory_requested: 512, memory_used: 211, estimated_cost: 22.30 },
    { pod_name: 'prometheus-server-0', namespace: 'monitoring', cluster: 'prod-us-east', cpu_utilization: 56.3, memory_utilization: 68.9, cpu_requested: 2000, cpu_used: 1126, memory_requested: 4096, memory_used: 2823, estimated_cost: 19.85 },
    { pod_name: 'grafana-7c6b5a4d3-m2n3o', namespace: 'monitoring', cluster: 'prod-us-east', cpu_utilization: 12.1, memory_utilization: 35.6, cpu_requested: 500, cpu_used: 60, memory_requested: 1024, memory_used: 365, estimated_cost: 15.42 },
    { pod_name: 'elasticsearch-data-0', namespace: 'logging', cluster: 'prod-us-east', cpu_utilization: 71.2, memory_utilization: 85.4, cpu_requested: 2000, cpu_used: 1424, memory_requested: 8192, memory_used: 6996, estimated_cost: 14.78 },
    { pod_name: 'fluentd-aggregator-6d5e4f3c-p9q0r', namespace: 'logging', cluster: 'prod-us-east', cpu_utilization: 48.9, memory_utilization: 52.3, cpu_requested: 1000, cpu_used: 489, memory_requested: 2048, memory_used: 1071, estimated_cost: 12.33 },
    { pod_name: 'staging-api-4b3a2c1d-j5k6l', namespace: 'staging', cluster: 'prod-us-east', cpu_utilization: 18.5, memory_utilization: 24.7, cpu_requested: 1000, cpu_used: 185, memory_requested: 2048, memory_used: 506, estimated_cost: 10.95 },
  ]

  const mockRecommendations: Recommendation[] = [
    { id: 1, pod_name: 'worker-processor-5c4d3b2a1-n7p8q', namespace: 'production', cluster: 'prod-us-east', reason: 'CPU over-provisioned by 55%', current_cpu: 2000, current_memory: 4096, recommended_cpu: 1000, recommended_memory: 4096, estimated_savings: 8.45, status: 'open', created_at: '2024-01-15T10:30:00Z' },
    { id: 2, pod_name: 'redis-master-0', namespace: 'production', cluster: 'prod-us-east', reason: 'CPU over-provisioned by 77%', current_cpu: 1000, current_memory: 2048, recommended_cpu: 250, recommended_memory: 2048, estimated_savings: 6.23, status: 'open', created_at: '2024-01-15T10:30:00Z' },
    { id: 3, pod_name: 'grafana-7c6b5a4d3-m2n3o', namespace: 'monitoring', cluster: 'prod-us-east', reason: 'Resources over-provisioned', current_cpu: 500, current_memory: 1024, recommended_cpu: 100, recommended_memory: 512, estimated_savings: 4.87, status: 'open', created_at: '2024-01-14T08:15:00Z' },
    { id: 4, pod_name: 'staging-api-4b3a2c1d-j5k6l', namespace: 'staging', cluster: 'prod-us-east', reason: 'Memory over-provisioned by 75%', current_cpu: 1000, current_memory: 2048, recommended_cpu: 500, recommended_memory: 512, estimated_savings: 3.92, status: 'open', created_at: '2024-01-14T08:15:00Z' },
    { id: 5, pod_name: 'nginx-ingress-controller-8f7e6d5c-w3x4y', namespace: 'ingress-nginx', cluster: 'prod-us-east', reason: 'CPU over-provisioned by 65%', current_cpu: 500, current_memory: 512, recommended_cpu: 200, recommended_memory: 512, estimated_savings: 2.15, status: 'open', created_at: '2024-01-13T14:45:00Z' },
  ]

  return {
    topline: mockTopline,
    namespaceAllocations: mockNamespaceAllocations,
    trends: mockTrends,
    utilization: mockUtilization,
    recommendations: mockRecommendations,
  }
}

interface CostDataState {
  topline: ToplineSummary | null
  namespaceAllocations: NamespaceAllocation[]
  trends: CostTrend[]
  utilization: UtilizationMetric[]
  recommendations: Recommendation[]
  loading: boolean
  error: string | null
}

interface UseCostDataResult extends CostDataState {
  window: TimeWindow
  setWindow: (window: TimeWindow) => void
  refresh: () => void
  applyRecommendation: (id: number) => Promise<boolean>
  dismissRecommendation: (id: number) => Promise<boolean>
}

export function useCostData(): UseCostDataResult {
  const [window, setWindow] = useState<TimeWindow>('7d')
  const [state, setState] = useState<CostDataState>({
    topline: null,
    namespaceAllocations: [],
    trends: [],
    utilization: [],
    recommendations: [],
    loading: true,
    error: null,
  })

  const userId = localStorage.getItem('user_id')

  const getHeaders = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-User-ID': userId || '',
  }), [userId])

  const fetchData = useCallback(async () => {
    // Use mock data for UI testing
    if (USE_MOCK_DATA) {
      setState(prev => ({ ...prev, loading: true, error: null }))
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 500))
      const mockData = generateMockData(window)
      setState({
        ...mockData,
        loading: false,
        error: null,
      })
      return
    }

    if (!userId) {
      setState(prev => ({ ...prev, loading: false, error: 'User not authenticated' }))
      return
    }

    setState(prev => ({ ...prev, loading: true, error: null }))

    try {
      const [toplineRes, allocationsRes, trendsRes, utilizationRes, recommendationsRes] = await Promise.all([
        fetch(`${API_SERVER_URL}/v1/allocation/summary/topline?window=${window}`, { headers: getHeaders() }),
        fetch(`${API_SERVER_URL}/v1/allocation/summary?window=${window}&aggregate=namespace`, { headers: getHeaders() }),
        fetch(`${API_SERVER_URL}/v1/costs/trends?interval=daily`, { headers: getHeaders() }),
        fetch(`${API_SERVER_URL}/v1/costs/utilization`, { headers: getHeaders() }),
        fetch(`${API_SERVER_URL}/v1/recommendations`, { headers: getHeaders() }),
      ])

      // Parse responses - handle both success and empty data gracefully
      const toplineData = toplineRes.ok ? await toplineRes.json() : null
      const allocationsData = allocationsRes.ok ? await allocationsRes.json() : null
      const trendsData = trendsRes.ok ? await trendsRes.json() : { trends: [] }
      const utilizationData = utilizationRes.ok ? await utilizationRes.json() : { metrics: [] }
      const recommendationsData = recommendationsRes.ok ? await recommendationsRes.json() : { recommendations: [] }

      // Transform API response to frontend format
      // API returns: { code, status, data: { totalCost, totalCPUCost, totalRAMCost, avgEfficiency, window } }
      const topline: ToplineSummary | null = toplineData?.data ? {
        total_cost: toplineData.data.totalCost || 0,
        cpu_cost: toplineData.data.totalCPUCost || 0,
        memory_cost: toplineData.data.totalRAMCost || 0,
        efficiency: toplineData.data.avgEfficiency || 0,
        window: toplineData.data.window || window,
      } : null

      // API returns: { code, status, data: { items: [...], totalCost, ... } }
      // items have: { name, cpuCost, ramCost, totalCost, cpuCoreHours, ramByteHours, totalEfficiency }
      const namespaceAllocations: NamespaceAllocation[] = (allocationsData?.data?.items || []).map((item: {
        name: string
        cpuCost: number
        ramCost: number
        totalCost: number
        cpuCoreHours: number
        ramByteHours: number
      }) => ({
        namespace: item.name,
        cpu_cost: item.cpuCost || 0,
        memory_cost: item.ramCost || 0,
        total_cost: item.totalCost || 0,
        cpu_hours: item.cpuCoreHours || 0,
        memory_gb_hours: (item.ramByteHours || 0) / (1024 * 1024 * 1024), // Convert bytes to GB
      }))

      // Transform trends from API format
      // API returns: { time, estimated_cost_usd, total_cpu_request_millicores, total_memory_request_bytes, ... }
      // Frontend expects: { date, cost, cpu_cost, memory_cost }
      const trends: CostTrend[] = (trendsData?.trends || []).map((t: {
        time?: string
        estimated_cost_usd?: number
        total_cpu_request_millicores?: number
        total_memory_request_bytes?: number
      }) => {
        const totalCost = t.estimated_cost_usd || 0
        // Estimate CPU/memory cost split using resource request proportions with typical cloud pricing weights
        const cpuCores = (t.total_cpu_request_millicores || 0) / 1000
        const ramGB = (t.total_memory_request_bytes || 0) / (1024 * 1024 * 1024)
        const cpuWeighted = cpuCores * 0.031611 // typical CPU $/core-hour
        const ramWeighted = ramGB * 0.004237    // typical RAM $/GB-hour
        const totalWeighted = cpuWeighted + ramWeighted
        const cpuRatio = totalWeighted > 0 ? cpuWeighted / totalWeighted : 0.58
        return {
          date: t.time ? new Date(t.time).toISOString().split('T')[0] : '',
          cost: totalCost,
          cpu_cost: totalCost * cpuRatio,
          memory_cost: totalCost * (1 - cpuRatio),
        }
      })

      // Transform utilization from API format
      // API returns: { cluster_name, cpu_utilization_percent, memory_utilization_percent, avg_cpu_usage_millicores, avg_cpu_request_millicores, avg_memory_usage_bytes, avg_memory_request_bytes }
      // Frontend expects: { cluster, cpu_utilization, memory_utilization, cpu_used, cpu_requested, memory_used, memory_requested, estimated_cost }
      const utilization: UtilizationMetric[] = (utilizationData?.metrics || []).map((m: {
        cluster_name?: string
        namespace?: string
        pod_name?: string
        cpu_utilization_percent?: number
        memory_utilization_percent?: number
        avg_cpu_usage_millicores?: number
        avg_cpu_request_millicores?: number
        avg_memory_usage_bytes?: number
        avg_memory_request_bytes?: number
      }) => {
        const cpuCores = (m.avg_cpu_request_millicores || 0) / 1000
        const ramGB = (m.avg_memory_request_bytes || 0) / (1024 * 1024 * 1024)
        return {
          pod_name: m.pod_name || '',
          namespace: m.namespace || '',
          cluster: m.cluster_name || '',
          cpu_utilization: m.cpu_utilization_percent || 0,
          memory_utilization: m.memory_utilization_percent || 0,
          cpu_requested: m.avg_cpu_request_millicores || 0,
          cpu_used: m.avg_cpu_usage_millicores || 0,
          memory_requested: m.avg_memory_request_bytes || 0,
          memory_used: m.avg_memory_usage_bytes || 0,
          estimated_cost: cpuCores * 0.031611 + ramGB * 0.004237, // $/hour estimate
        }
      })

      // Transform recommendations from API format
      // API returns array of: { ID, PodName, Namespace, ClusterName, ResourceType, CurrentRequest, RecommendedRequest, PotentialSavingsUSD, Status, CreatedAt, Reason }
      // Frontend expects: { id, pod_name, namespace, cluster, reason, current_cpu, current_memory, recommended_cpu, recommended_memory, estimated_savings, status, created_at }
      const rawRecs: Recommendation[] = (Array.isArray(recommendationsData) ? recommendationsData : recommendationsData?.recommendations || []).map((r: {
        ID?: number; id?: number
        PodName?: string; pod_name?: string
        Namespace?: string; namespace?: string
        ClusterName?: string; cluster_name?: string; cluster?: string
        ResourceType?: string; resource_type?: string
        CurrentRequest?: number; current_request?: number
        RecommendedRequest?: number; recommended_request?: number
        PotentialSavingsUSD?: number; potential_savings_usd?: number; estimated_savings?: number
        Status?: string; status?: string
        CreatedAt?: string; created_at?: string
        Reason?: string; reason?: string
        current_cpu?: number; current_memory?: number
        recommended_cpu?: number; recommended_memory?: number
      }) => {
        const resourceType = (r.ResourceType || r.resource_type || 'cpu').toLowerCase()
        const currentReq = r.CurrentRequest || r.current_request || 0
        const recommendedReq = r.RecommendedRequest || r.recommended_request || 0
        return {
          id: r.ID || r.id || 0,
          pod_name: r.PodName || r.pod_name || '',
          namespace: r.Namespace || r.namespace || '',
          cluster: r.ClusterName || r.cluster_name || r.cluster || '',
          reason: r.Reason || r.reason || '',
          current_cpu: resourceType === 'cpu' ? currentReq : (r.current_cpu || 0),
          current_memory: resourceType === 'memory' ? currentReq : (r.current_memory || 0),
          recommended_cpu: resourceType === 'cpu' ? recommendedReq : (r.recommended_cpu || 0),
          recommended_memory: resourceType === 'memory' ? recommendedReq : (r.recommended_memory || 0),
          estimated_savings: r.PotentialSavingsUSD || r.potential_savings_usd || r.estimated_savings || 0,
          status: (r.Status || r.status || 'open') as 'open' | 'applied' | 'dismissed',
          created_at: r.CreatedAt || r.created_at || '',
        }
      })

      setState({
        topline,
        namespaceAllocations,
        trends,
        utilization,
        recommendations: rawRecs.filter(r => r.status === 'open'),
        loading: false,
        error: null,
      })
    } catch (err) {
      console.error('Failed to fetch cost data:', err)
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Failed to fetch cost data',
      }))
    }
  }, [window, userId, getHeaders])

  const applyRecommendation = useCallback(async (id: number): Promise<boolean> => {
    if (USE_MOCK_DATA) {
      // Simulate applying recommendation - remove from list
      await new Promise(resolve => setTimeout(resolve, 300))
      setState(prev => ({
        ...prev,
        recommendations: prev.recommendations.filter(r => r.id !== id),
      }))
      return true
    }

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/recommendations/${id}/apply`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (response.ok) {
        // Refresh recommendations after action
        fetchData()
        return true
      }
      return false
    } catch {
      return false
    }
  }, [getHeaders, fetchData])

  const dismissRecommendation = useCallback(async (id: number): Promise<boolean> => {
    if (USE_MOCK_DATA) {
      // Simulate dismissing recommendation - remove from list
      await new Promise(resolve => setTimeout(resolve, 300))
      setState(prev => ({
        ...prev,
        recommendations: prev.recommendations.filter(r => r.id !== id),
      }))
      return true
    }

    try {
      const response = await fetch(`${API_SERVER_URL}/v1/recommendations/${id}/dismiss`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (response.ok) {
        // Refresh recommendations after action
        fetchData()
        return true
      }
      return false
    } catch {
      return false
    }
  }, [getHeaders, fetchData])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return {
    ...state,
    window,
    setWindow,
    refresh: fetchData,
    applyRecommendation,
    dismissRecommendation,
  }
}
