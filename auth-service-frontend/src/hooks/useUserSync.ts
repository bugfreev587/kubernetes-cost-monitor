import { useEffect, useState, useRef } from 'react'
import { useUser } from '@clerk/clerk-react'

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

// User roles in order of permission level
export type UserRole = 'owner' | 'admin' | 'editor' | 'viewer'
export type UserStatus = 'active' | 'suspended' | 'pending'

interface UserSyncState {
  isSynced: boolean
  isSyncing: boolean
  error: string | null
  tenantId: number | null
  userId: string | null  // Clerk user ID (e.g., 'user_xxx')
  role: UserRole | null
  status: UserStatus | null
  pricingPlan: string | null
}

// Helper to check if user has at least a certain role level
export function hasPermission(userRole: UserRole | null, requiredRole: UserRole): boolean {
  if (!userRole) return false
  const levels: Record<UserRole, number> = { owner: 4, admin: 3, editor: 2, viewer: 1 }
  return levels[userRole] >= levels[requiredRole]
}

export function useUserSync(): UserSyncState {
  const { isSignedIn, isLoaded, user } = useUser()
  const [state, setState] = useState<UserSyncState>({
    isSynced: false,
    isSyncing: false,
    error: null,
    tenantId: null,
    userId: null,
    role: null,
    status: null,
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
        role: null,
        status: null,
        pricingPlan: null,
      })
      // Clear localStorage
      localStorage.removeItem('tenant_id')
      localStorage.removeItem('user_id')
      localStorage.removeItem('user_role')
      localStorage.removeItem('user_status')
      localStorage.removeItem('pricing_plan')
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
          // Handle suspended user specifically
          if (errorData.error === 'user_suspended') {
            throw new Error('Your account has been suspended. Please contact your organization administrator.')
          }
          throw new Error(errorData.message || errorData.error || `HTTP ${response.status}`)
        }

        const data = await response.json()

        setState({
          isSynced: true,
          isSyncing: false,
          error: null,
          tenantId: data.tenant_id,
          userId: data.user_id,
          role: data.role as UserRole,
          status: data.status as UserStatus,
          pricingPlan: data.pricing_plan,
        })

        // Store in localStorage for other hooks/components to use
        localStorage.setItem('tenant_id', String(data.tenant_id))
        localStorage.setItem('user_id', String(data.user_id))
        localStorage.setItem('user_role', data.role || '')
        localStorage.setItem('user_status', data.status || '')
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
