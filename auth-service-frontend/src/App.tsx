import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@clerk/clerk-react'
import HomePage from './pages/HomePage'
import SignInPage from './pages/SignInPage'
import SignUpPage from './pages/SignUpPage'
import DashboardRoute from './components/DashboardRoute'
import ProfilePage from './pages/ProfilePage'
import FeaturesPage from './pages/FeaturesPage'
import PricingPage from './pages/PricingPage'
import { useUserSync } from './hooks/useUserSync'
import './App.css'

// Component that syncs user with backend on sign in
function UserSyncProvider({ children }: { children: React.ReactNode }) {
  const { isSyncing, error } = useUserSync()

  // Optionally show syncing state or error
  if (error) {
    console.warn('User sync error:', error)
  }

  if (isSyncing) {
    // You can show a loading state here if needed
    // For now, we just render children to avoid blocking
  }

  return <>{children}</>
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isLoaded, isSignedIn } = useAuth()

  if (!isLoaded) {
    return (
      <div className="dashboard-container">
        <div className="dashboard-card">
          <p>Loading...</p>
        </div>
      </div>
    )
  }

  if (!isSignedIn) {
    return <Navigate to="/sign-in" replace />
  }

  return <>{children}</>
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isLoaded, isSignedIn } = useAuth()

  if (!isLoaded) {
    return (
      <div className="auth-container">
        <div className="auth-card">
          <p>Loading...</p>
        </div>
      </div>
    )
  }

  if (isSignedIn) {
    // If user has selected a pricing plan, redirect to dashboard
    // Otherwise, redirect to features page
    const selectedPlan = localStorage.getItem('selected_pricing_plan')
    if (selectedPlan) {
      return <Navigate to="/dashboard" replace />
    }
    return <Navigate to="/features" replace />
  }

  return <>{children}</>
}

function PublicHomeRoute({ children }: { children: React.ReactNode }) {
  const { isLoaded, isSignedIn } = useAuth()

  if (!isLoaded) {
    return (
      <div className="homepage">
        <div className="homepage-content">
          <p>Loading...</p>
        </div>
      </div>
    )
  }

  if (isSignedIn) {
    return <Navigate to="/features" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <UserSyncProvider>
      <BrowserRouter>
        <Routes>
        <Route
          path="/"
          element={
            <PublicHomeRoute>
              <HomePage />
            </PublicHomeRoute>
          }
        />
        <Route
          path="/features"
          element={<FeaturesPage />}
        />
        <Route
          path="/pricing"
          element={<PricingPage />}
        />
        <Route
          path="/sign-in"
          element={
            <PublicRoute>
              <SignInPage />
            </PublicRoute>
          }
        />
        <Route
          path="/sign-up"
          element={
            <PublicRoute>
              <SignUpPage />
            </PublicRoute>
          }
        />
        <Route
          path="/dashboard"
          element={<DashboardRoute />}
        />
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <ProfilePage />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </UserSyncProvider>
  )
}

export default App
