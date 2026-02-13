import { SignUp } from '@clerk/clerk-react'
import { Link } from 'react-router-dom'
import '../App.css'

export default function SignUpPage() {
  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Create Account</h1>
        <p className="auth-subtitle">Sign up to get started</p>
        <SignUp
          routing="virtual"
          signInUrl="/sign-in"
          forceRedirectUrl="/pricing"
        />
        <p className="auth-switch">
          Already have an account? <Link to="/sign-in">Sign in</Link>
        </p>
      </div>
    </div>
  )
}

