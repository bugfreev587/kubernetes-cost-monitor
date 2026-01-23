import { useUser } from '@clerk/clerk-react'
import { useUserSync } from '../hooks/useUserSync'
import Navbar from '../components/Navbar'
import '../App.css'

// Helper to display role with proper formatting
function formatRole(role: string | null): string {
  if (!role) return 'Loading...'
  return role.charAt(0).toUpperCase() + role.slice(1)
}

export default function ProfilePage() {
  const { user, isLoaded } = useUser()
  const { userId, role, tenantId, pricingPlan, isSynced } = useUserSync()

  if (!isLoaded) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="page-card">
            <p>Loading...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="dashboard-card">
          <h1>Profile</h1>
          <div className="user-info">
            <h2>Welcome, {user?.firstName || user?.emailAddresses[0]?.emailAddress || 'User'}!</h2>
            {user?.imageUrl && (
              <img 
                src={user.imageUrl} 
                alt="Profile" 
                className="profile-image"
              />
            )}
            <div className="user-details">
              <p><strong>Email:</strong> {user?.emailAddresses[0]?.emailAddress}</p>
              {user?.firstName && (
                <p><strong>First Name:</strong> {user.firstName}</p>
              )}
              {user?.lastName && (
                <p><strong>Last Name:</strong> {user.lastName}</p>
              )}
              <p><strong>User ID:</strong> {isSynced ? userId : 'Loading...'}</p>
              <p><strong>Role:</strong> {isSynced ? formatRole(role) : 'Loading...'}</p>
              <p><strong>Tenant ID:</strong> {isSynced ? tenantId : 'Loading...'}</p>
              <p><strong>Pricing Plan:</strong> {isSynced ? (pricingPlan || 'Starter') : 'Loading...'}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
