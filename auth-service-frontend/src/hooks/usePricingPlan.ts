import { useState, useEffect } from 'react'
import { useAuth } from '@clerk/clerk-react'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

interface PricingPlanStatus {
  hasPlan: boolean
  planName: string | null
  isLoading: boolean
  error: string | null
}

export function usePricingPlan(): PricingPlanStatus {
  const { isSignedIn, isLoaded } = useAuth()
  const [hasPlan, setHasPlan] = useState(false)
  const [planName, setPlanName] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isLoaded) {
      setIsLoading(true)
      return
    }

    if (!isSignedIn) {
      setHasPlan(false)
      setPlanName(null)
      setIsLoading(false)
      setError(null)
      return
    }

    // Get tenant_id from localStorage (set by useUserSync hook)
    const storedTenantId = localStorage.getItem('tenant_id')
    const tenantId = storedTenantId ? parseInt(storedTenantId, 10) : null

    if (!tenantId) {
      // User hasn't been synced yet, wait for sync
      setIsLoading(false)
      setHasPlan(false)
      return
    }

    const fetchPricingPlan = async () => {
      setIsLoading(true)
      setError(null)

      // Get user_id from localStorage for authentication
      const userId = localStorage.getItem('user_id')
      if (!userId) {
        // User not synced yet, can't make authenticated request
        setIsLoading(false)
        setHasPlan(false)
        return
      }

      try {
        const response = await fetch(`${API_SERVER_URL}/v1/admin/tenants/${tenantId}/pricing-plan`, {
          headers: {
            'X-User-ID': userId,
          },
        })

        if (!response.ok) {
          if (response.status === 404) {
            setHasPlan(false)
            setPlanName(null)
            setIsLoading(false)
            return
          }
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }

        const data = await response.json()
        const plan = data.pricing_plan

        if (plan && plan.trim() !== '') {
          setHasPlan(true)
          setPlanName(plan)
        } else {
          setHasPlan(false)
          setPlanName(null)
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch pricing plan')
        setHasPlan(false)
        setPlanName(null)
      } finally {
        setIsLoading(false)
      }
    }

    fetchPricingPlan()
  }, [isSignedIn, isLoaded])

  return { hasPlan, planName, isLoading, error }
}

