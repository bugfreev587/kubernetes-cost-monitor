import { useEffect, useState, useRef } from 'react'
import { useUser } from '@clerk/clerk-react'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

interface UserSyncState {
  isSynced: boolean
  isSyncing: boolean
  error: string | null
  tenantId: number | null
  userId: number | null
  pricingPlan: string | null
}

export function useUserSync(): UserSyncState {
  const { isSignedIn, isLoaded, user } = useUser()
  const [state, setState] = useState<UserSyncState>({
    isSynced: false,
    isSyncing: false,
    error: null,
    tenantId: null,
    userId: null,
    pricingPlan: null,
  })
  const syncAttempted = useRef(false)

  useEffect(() => {
    // Reset state when user signs out
    if (isLoaded && !isSignedIn) {
      setState({
        isSynced: false,
        isSyncing: false,
        error: null,
        tenantId: null,
        userId: null,
        pricingPlan: null,
      })
      syncAttempted.current = false
      return
    }

    // Don't sync if not loaded, not signed in, or already synced/syncing
    if (!isLoaded || !isSignedIn || !user || syncAttempted.current) {
      return
    }

    const syncUser = async () => {
      syncAttempted.current = true
      setState(prev => ({ ...prev, isSyncing: true, error: null }))

      try {
        const primaryEmail = user.primaryEmailAddress?.emailAddress
        if (!primaryEmail) {
          throw new Error('No email address found')
        }

        const response = await fetch(`${API_SERVER_URL}/v1/auth/sync`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            clerk_user_id: user.id,
            email: primaryEmail,
            first_name: user.firstName || '',
            last_name: user.lastName || '',
          }),
        })

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}))
          throw new Error(errorData.error || `HTTP ${response.status}`)
        }

        const data = await response.json()

        setState({
          isSynced: true,
          isSyncing: false,
          error: null,
          tenantId: data.tenant_id,
          userId: data.user_id,
          pricingPlan: data.pricing_plan,
        })

        // Store tenant_id in localStorage for other hooks to use
        localStorage.setItem('tenant_id', String(data.tenant_id))
        localStorage.setItem('user_id', String(data.user_id))
        localStorage.setItem('pricing_plan', data.pricing_plan || '')

        if (data.is_new_user) {
          console.log('New user created:', data.email)
        }
      } catch (err) {
        console.error('Failed to sync user:', err)
        setState(prev => ({
          ...prev,
          isSyncing: false,
          error: err instanceof Error ? err.message : 'Failed to sync user',
        }))
      }
    }

    syncUser()
  }, [isLoaded, isSignedIn, user])

  return state
}
