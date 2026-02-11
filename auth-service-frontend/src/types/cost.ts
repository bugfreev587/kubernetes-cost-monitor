// TypeScript interfaces for cost API responses

export interface ToplineSummary {
  total_cost: number
  cpu_cost: number
  memory_cost: number
  efficiency: number
  window: string
}

export interface NamespaceAllocation {
  namespace: string
  cpu_cost: number
  memory_cost: number
  total_cost: number
  cpu_hours: number
  memory_gb_hours: number
}

export interface CostTrend {
  date: string
  cost: number
  cpu_cost: number
  memory_cost: number
}

export interface UtilizationMetric {
  pod_name: string
  namespace: string
  cluster: string
  cpu_utilization: number
  memory_utilization: number
  cpu_requested: number
  cpu_used: number
  memory_requested: number
  memory_used: number
  estimated_cost: number
}

export interface Recommendation {
  id: number
  pod_name: string
  namespace: string
  cluster: string
  reason: string
  current_cpu: number
  current_memory: number
  recommended_cpu: number
  recommended_memory: number
  estimated_savings: number
  status: 'open' | 'applied' | 'dismissed'
  created_at: string
}

// API response wrappers
export interface ToplineSummaryResponse {
  summary: ToplineSummary
}

export interface NamespaceAllocationResponse {
  allocations: NamespaceAllocation[]
  window: string
}

export interface CostTrendsResponse {
  trends: CostTrend[]
  interval: string
}

export interface UtilizationResponse {
  metrics: UtilizationMetric[]
}

export interface RecommendationsResponse {
  recommendations: Recommendation[]
}

// Time window type
export type TimeWindow = '7d' | '30d'
