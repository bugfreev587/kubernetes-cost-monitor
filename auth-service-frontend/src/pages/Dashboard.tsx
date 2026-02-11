import { useUser } from '@clerk/clerk-react'
import { useNavigate } from 'react-router-dom'
import Navbar from '../components/Navbar'
import '../App.css'

export default function Dashboard() {
  const { isLoaded, user } = useUser()
  const navigate = useNavigate()

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
          <h1>Dashboard</h1>
          <div className="dashboard-welcome">
            <h2>Welcome{user?.firstName ? `, ${user.firstName}` : ''}!</h2>
            <p>You're signed in to Kubernetes Cost Monitor.</p>
            <div className="dashboard-actions">
              <button
                className="btn btn-primary"
                onClick={() => navigate('/management')}
              >
                Go to Management
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
