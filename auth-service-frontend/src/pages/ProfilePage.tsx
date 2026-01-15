import { useUser } from '@clerk/clerk-react'
import Navbar from '../components/Navbar'
import '../App.css'

export default function ProfilePage() {
  const { user, isLoaded } = useUser()

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
              <p><strong>User ID:</strong> {user?.id}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
