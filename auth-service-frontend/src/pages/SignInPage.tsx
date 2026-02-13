import { SignIn } from '@clerk/clerk-react'
import { Link } from 'react-router-dom'
import '../App.css'

export default function SignInPage() {
  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Welcome Back</h1>
        <p className="auth-subtitle">Sign in to your account</p>
        <SignIn
          routing="virtual"
          signUpUrl="/sign-up"
          forceRedirectUrl="/dashboard"
        />
        <p className="auth-switch">
          Don't have an account? <Link to="/sign-up">Sign up</Link>
        </p>
      </div>
    </div>
  )
}

