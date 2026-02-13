import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@clerk/clerk-react'
import HomePage from './pages/HomePage'
import SignInPage from './pages/SignInPage'
import SignUpPage from './pages/SignUpPage'
import DashboardRoute from './components/DashboardRoute'
import ProfilePage from './pages/ProfilePage'
import FeaturesPage from './pages/FeaturesPage'
import PricingPage from './pages/PricingPage'
import ManagementPage from './pages/ManagementPage'
import PricingConfigPage from './pages/PricingConfigPage'
import APIKeyModal from './components/APIKeyModal'
import AutoSignOutWarning from './components/AutoSignOutWarning'
import { useUserSync } from './hooks/useUserSync'
import { useAutoSignOut } from './hooks/useAutoSignOut'
import './App.css'

// Component that syncs user with backend on sign in
function UserSyncProvider({ children }: { children: React.ReactNode }) {
  const { isSyncing, error, isNewUser, apiKey } = useUserSync()
  const [showAPIKeyModal, setShowAPIKeyModal] = useState(false)
  const [displayedAPIKey, setDisplayedAPIKey] = useState<string | null>(null)

  // Show modal when new user with API key is detected
  useEffect(() => {
    if (isNewUser && apiKey) {
      setDisplayedAPIKey(apiKey)
      setShowAPIKeyModal(true)
    }
  }, [isNewUser, apiKey])

  const handleCloseModal = () => {
    setShowAPIKeyModal(false)
    setDisplayedAPIKey(null)
  }

  // Optionally show syncing state or error
  if (error) {
    console.warn('User sync error:', error)
  }

  if (isSyncing) {
    // You can show a loading state here if needed
    // For now, we just render children to avoid blocking
  }

  return (
    <>
      {children}
      {showAPIKeyModal && displayedAPIKey && (
        <APIKeyModal apiKey={displayedAPIKey} onClose={handleCloseModal} />
      )}
    </>
  )
}

// Component that handles auto sign-out for signed-in users
function AutoSignOutProvider({ children }: { children: React.ReactNode }) {
  const { isSignedIn } = useAuth()
  const { showWarning, timeRemaining, handleContinue, handleSignOut } = useAutoSignOut()

  return (
    <>
      {children}
      {isSignedIn && showWarning && (
        <AutoSignOutWarning
          timeRemaining={timeRemaining}
          onContinue={handleContinue}
          onSignOut={handleSignOut}
        />
      )}
    </>
  )
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

function PublicRoute({ children, fallbackRedirect = "/dashboard" }: { children: React.ReactNode, fallbackRedirect?: string }) {
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
    const selectedPlan = localStorage.getItem('selected_pricing_plan')
    if (selectedPlan) {
      return <Navigate to="/dashboard" replace />
    }
    return <Navigate to={fallbackRedirect} replace />
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
    return <Navigate to="/dashboard" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <UserSyncProvider>
      <AutoSignOutProvider>
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
              <PublicRoute fallbackRedirect="/pricing">
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
          <Route
            path="/management"
            element={
              <ProtectedRoute>
                <ManagementPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/pricing-config"
            element={
              <ProtectedRoute>
                <PricingConfigPage />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AutoSignOutProvider>
    </UserSyncProvider>
  )
}

export default App
