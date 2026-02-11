import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@clerk/clerk-react'
import Navbar from '../components/Navbar'
import { useUserSync } from '../hooks/useUserSync'
import '../App.css'

interface PricingPlan {
  name: string
  price: number
  period: string
  description: string
  features: string[]
  ctaText: string
  ctaLink: string
  popular?: boolean
}

const plans: PricingPlan[] = [
  {
    name: 'Starter',
    price: 0,
    period: 'month',
    description: 'Perfect for trying out Kubernetes cost monitoring',
    features: [
      '1 cluster',
      'Up to 5 nodes',
      '7-day data retention',
      'Basic cost tracking',
      'Email support',
    ],
    ctaText: 'Get Started Free',
    ctaLink: '/sign-up',
  },
  {
    name: 'Premium',
    price: 49,
    period: 'month',
    description: 'Ideal for growing teams with multiple clusters',
    features: [
      'Up to 10 clusters',
      'Up to 100 nodes',
      '30-day data retention',
      'Advanced cost analytics',
      'Cost optimization recommendations',
      'Custom alerts',
      'Priority support',
    ],
    ctaText: 'Start Premium',
    ctaLink: '/sign-up',
    popular: true,
  },
  {
    name: 'Business',
    price: 199,
    period: 'month',
    description: 'Enterprise-grade solution for large-scale deployments',
    features: [
      'Unlimited clusters',
      'Unlimited nodes',
      '1 year data retention',
      'Enterprise analytics',
      '24/7 priority support',
      'Custom integrations',
      'Dedicated account manager',
      'SLA guarantee',
    ],
    ctaText: 'Contact Sales',
    ctaLink: '/sign-up',
  },
]

const API_SERVER_URL = import.meta.env.VITE_API_SERVER_URL || 'http://localhost:8080'

export default function PricingPage() {
  const navigate = useNavigate()
  const { isSignedIn } = useAuth()
  const { userId, tenantId, role, pricingPlan, isSynced } = useUserSync()
  const [isSelecting, setIsSelecting] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)

  const isOwner = role === 'owner'

  const handlePlanSelection = async (planName: string) => {
    // Clear previous messages
    setError(null)
    setSuccess(null)
    setInfo(null)

    // If not signed in, navigate to sign-up
    if (!isSignedIn) {
      localStorage.setItem('selected_pricing_plan', planName)
      navigate('/sign-up')
      return
    }

    // Only owners can change the pricing plan
    if (!isOwner) {
      setError('Only the tenant owner can change the pricing plan')
      return
    }

    if (!isSynced || !tenantId || !userId) {
      setError('Please wait for user sync to complete')
      return
    }

    // Check if already on this plan
    if (pricingPlan === planName) {
      setInfo(`You are already on the ${planName} plan`)
      return
    }

    setIsSelecting(planName)

    try {
      // Update pricing plan - owner only endpoint
      const response = await fetch(`${API_SERVER_URL}/v1/owner/tenants/${tenantId}/pricing-plan`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId,
        },
        body: JSON.stringify({
          pricing_plan: planName,
        }),
      })

      if (!response.ok) {
        let errorMessage = 'Failed to update pricing plan'
        try {
          const errorData = await response.json()
          errorMessage = errorData.message || errorData.error || errorMessage
        } catch {
          errorMessage = `HTTP ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMessage)
      }

      // Store the selected plan in localStorage
      localStorage.setItem('selected_pricing_plan', planName)
      localStorage.setItem('pricing_plan', planName)

      // Show success message
      setSuccess(`Successfully upgraded to ${planName} plan!`)
      setIsSelecting(null)

      // Navigate to management page after a short delay
      setTimeout(() => {
        navigate('/management')
      }, 2000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
      console.error('Error updating pricing plan:', err)
      setIsSelecting(null)
    }
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="pricing-container">
          <div className="pricing-header">
            <h1>Simple, Transparent Pricing</h1>
            <p className="pricing-subtitle">
              Choose the plan that's right for your Kubernetes infrastructure
            </p>
          </div>

          <div className="pricing-grid">
            {plans.map((plan) => (
              <div
                key={plan.name}
                className={`pricing-card ${plan.popular ? 'pricing-card-popular' : ''}`}
              >
                {plan.popular && (
                  <div className="pricing-badge">Most Popular</div>
                )}
                <div className="pricing-card-header">
                  <h2 className="pricing-plan-name">{plan.name}</h2>
                  <p className="pricing-description">{plan.description}</p>
                </div>
                <div className="pricing-price">
                  <span className="pricing-currency">$</span>
                  <span className="pricing-amount">{plan.price}</span>
                  <span className="pricing-period">/{plan.period}</span>
                </div>
                <ul className="pricing-features">
                  {plan.features.map((feature, index) => (
                    <li key={index} className="pricing-feature">
                      <span className="pricing-feature-check">✓</span>
                      {feature}
                    </li>
                  ))}
                </ul>
                <button
                  onClick={() => handlePlanSelection(plan.name)}
                  disabled={isSelecting === plan.name}
                  className={`pricing-button ${
                    plan.popular
                      ? 'pricing-button-primary'
                      : 'pricing-button-secondary'
                  }`}
                >
                  {isSelecting === plan.name ? 'Selecting...' : plan.ctaText}
                </button>
              </div>
            ))}
          </div>

          {success && (
            <div style={{
              marginTop: '2rem',
              padding: '1rem',
              background: '#d4edda',
              color: '#155724',
              borderRadius: '8px',
              textAlign: 'center',
              fontWeight: 'bold'
            }}>
              {success}
            </div>
          )}

          {info && (
            <div style={{
              marginTop: '2rem',
              padding: '1rem',
              background: '#cce5ff',
              color: '#004085',
              borderRadius: '8px',
              textAlign: 'center'
            }}>
              {info}
            </div>
          )}

          {error && (
            <div style={{
              marginTop: '2rem',
              padding: '1rem',
              background: '#fee',
              color: '#c82333',
              borderRadius: '8px',
              textAlign: 'center'
            }}>
              {error}
            </div>
          )}

          <div className="pricing-footer">
            <p className="pricing-note">
              Starter plan is free forever. No credit card required.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

