import Navbar from '../components/Navbar'
import { Link } from 'react-router-dom'
import '../App.css'

export default function HomePage() {
  return (
    <div className="homepage">
      <Navbar />
      <div className="homepage-content">
        <div className="homepage-hero">
          <h1 className="homepage-title">Welcome to k8s-cost</h1>
          <p className="homepage-subtitle">
            Manage and optimize your Kubernetes costs with ease
          </p>
          <div className="homepage-cta">
            <Link to="/sign-up" className="cta-button cta-button-primary">
              Get Started
            </Link>
            <Link to="/sign-in" className="cta-button cta-button-secondary">
              Sign In
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}

