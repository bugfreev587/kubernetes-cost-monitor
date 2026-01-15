import { Navigate } from 'react-router-dom'
import { useAuth } from '@clerk/clerk-react'
import Dashboard from '../pages/Dashboard'
import { usePricingPlan } from '../hooks/usePricingPlan'

export default function DashboardRoute() {
  const { isLoaded, isSignedIn } = useAuth()
  const { hasPlan, isLoading } = usePricingPlan()

  if (!isLoaded || isLoading) {
    return (
      <div className="page-container">
        <div className="page-content">
          <div className="page-card">
            <p>Loading...</p>
          </div>
        </div>
      </div>
    )
  }

  // If user is not signed in, redirect to sign-in page
  if (!isSignedIn) {
    return <Navigate to="/sign-in" replace />
  }

  // If user doesn't have a pricing plan, redirect to pricing page
  if (!hasPlan) {
    return <Navigate to="/pricing" replace />
  }

  // User is signed in and has a pricing plan, show Dashboard
  return <Dashboard />
}
