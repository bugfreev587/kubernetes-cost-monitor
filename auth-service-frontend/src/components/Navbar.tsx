import { useState, useRef, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useUser, useAuth, SignOutButton } from '@clerk/clerk-react'
import { useUserSync, hasPermission } from '../hooks/useUserSync'
import '../App.css'

export default function Navbar() {
  const { user, isLoaded: userLoaded } = useUser()
  const { isLoaded: authLoaded, isSignedIn } = useAuth()
  const { role, isSynced } = useUserSync()
  const [showUserMenu, setShowUserMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Check if user has admin or owner role
  const canAccessManagement = isSynced && hasPermission(role, 'admin')

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setShowUserMenu(false)
      }
    }

    if (showUserMenu) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showUserMenu])

  const isLoaded = userLoaded && authLoaded

  return (
    <nav className="navbar">
      <div className="navbar-container">
        {/* Logo */}
        <Link to="/" className="navbar-logo">
          k8s-cost
        </Link>

        {/* Center Navigation */}
        <div className="navbar-center">
          <Link to="/features" className="navbar-link">
            Features
          </Link>
          <Link to="/pricing" className="navbar-link">
            Pricing
          </Link>
        </div>

        {/* Right Navigation */}
        <div className="navbar-right">
          {!isLoaded ? (
            <div className="navbar-loading">Loading...</div>
          ) : isSignedIn ? (
            <>
            <Link to="/dashboard" className="navbar-link">
              Dashboard
            </Link>
            <div className="user-menu-container" ref={menuRef}>
              <button
                className="user-menu-button"
                onClick={() => setShowUserMenu(!showUserMenu)}
                aria-label="User menu"
              >
                {user?.imageUrl ? (
                  <img
                    src={user.imageUrl}
                    alt="Profile"
                    className="user-avatar"
                  />
                ) : (
                  <div className="user-avatar-placeholder">
                    {user?.firstName?.[0] || user?.emailAddresses[0]?.emailAddress?.[0] || 'U'}
                  </div>
                )}
              </button>
              {showUserMenu && (
                <div className="user-menu-dropdown">
                  <div className="user-menu-header">
                    <p className="user-menu-name">
                      {user?.firstName && user?.lastName
                        ? `${user.firstName} ${user.lastName}`
                        : user?.firstName || user?.emailAddresses[0]?.emailAddress || 'User'}
                    </p>
                    <p className="user-menu-email">
                      {user?.emailAddresses[0]?.emailAddress}
                    </p>
                  </div>
                  <div className="user-menu-divider"></div>
                  <Link
                    to="/profile"
                    className="user-menu-item"
                    onClick={() => setShowUserMenu(false)}
                  >
                    Profile
                  </Link>
                  {canAccessManagement && (
                    <>
                      <Link
                        to="/management"
                        className="user-menu-item"
                        onClick={() => setShowUserMenu(false)}
                      >
                        Management
                      </Link>
                      <Link
                        to="/pricing-config"
                        className="user-menu-item"
                        onClick={() => setShowUserMenu(false)}
                      >
                        Pricing Config
                      </Link>
                    </>
                  )}
                  <div className="user-menu-divider"></div>
                  <SignOutButton>
                    <button className="user-menu-item user-menu-signout">
                      Sign Out
                    </button>
                  </SignOutButton>
                </div>
              )}
            </div>
            </>
          ) : (
            <>
              <Link to="/sign-in" className="navbar-button navbar-button-secondary">
                Sign In
              </Link>
              <Link to="/sign-up" className="navbar-button navbar-button-primary">
                Sign Up
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}
