import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@clerk/clerk-react'
import Navbar from '../components/Navbar'
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
    name: 'Basic',
    price: 0,
    period: 'month',
    description: 'Perfect for getting started with Kubernetes cost monitoring',
    features: [
      'Up to 3 clusters',
      'Basic cost tracking',
      'Daily cost reports',
      'Email support',
      'Community access',
    ],
    ctaText: 'Get Started Free',
    ctaLink: '/sign-up',
  },
  {
    name: 'Standard',
    price: 49,
    period: 'month',
    description: 'Ideal for growing teams with multiple clusters',
    features: [
      'Up to 10 clusters',
      'Advanced cost analytics',
      'Real-time cost monitoring',
      'Custom cost alerts',
      'Priority email support',
      'Cost optimization recommendations',
      'Historical data (30 days)',
    ],
    ctaText: 'Start Free Trial',
    ctaLink: '/sign-up',
    popular: true,
  },
  {
    name: 'Professional',
    price: 199,
    period: 'month',
    description: 'Enterprise-grade solution for large-scale deployments',
    features: [
      'Unlimited clusters',
      'Enterprise cost analytics',
      'Real-time monitoring & alerts',
      'Advanced cost optimization',
      '24/7 priority support',
      'Custom integrations',
      'Historical data (1 year)',
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
  const [isSelecting, setIsSelecting] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handlePlanSelection = async (planName: string) => {
    setIsSelecting(planName)
    setError(null)

    try {
      // TODO: Get tenant_id from user metadata once tenant assignment is implemented
      // For now, using default tenant_id of 1
      const tenantId = 1

      // Update pricing plan in the database
      const response = await fetch(`${API_SERVER_URL}/v1/admin/tenants/${tenantId}/pricing-plan`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          pricing_plan: planName,
        }),
      })

      if (!response.ok) {
        let errorMessage = 'Failed to update pricing plan'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          errorMessage = `HTTP ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMessage)
      }

      // Store the selected plan in localStorage
      localStorage.setItem('selected_pricing_plan', planName)

      // If user is signed in, navigate to dashboard
      // If not signed in, navigate to sign-up (they'll be redirected to dashboard after sign-up)
      if (isSignedIn) {
        navigate('/dashboard')
      } else {
        navigate('/sign-up')
      }
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
              All plans include a 14-day free trial. No credit card required.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

