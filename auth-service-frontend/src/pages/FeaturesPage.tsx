import { Link } from 'react-router-dom'
import Navbar from '../components/Navbar'
import '../App.css'
import './FeaturesPage.css'

export default function FeaturesPage() {
  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="features-container">

          {/* Section 1: Hero Header */}
          <div className="features-hero">
            <h1>Kubernetes Cost Visibility, Simplified</h1>
            <p className="features-hero-subtitle">
              k8s-cost gives you real-time cost tracking, resource optimization, and actionable
              insights — with a lightweight agent that deploys in minutes.
            </p>
            <Link to="/sign-up" className="cta-button cta-button-primary features-hero-cta">
              Get Started Free
            </Link>
          </div>

          {/* Section 2: How It Works */}
          <div className="features-section">
            <div className="features-section-header">
              <h2>How It Works</h2>
              <p>Three steps from deploy to cost savings</p>
            </div>
            <div className="features-steps">
              <div className="features-step-card">
                <div className="features-step-number">1</div>
                <h3>Deploy the Agent</h3>
                <p>
                  Install a lightweight agent into your cluster with a single kubectl apply.
                  It collects CPU, memory, and resource request metrics every 10 minutes —
                  with zero application changes.
                </p>
              </div>
              <div className="features-step-card">
                <div className="features-step-number">2</div>
                <h3>Analyze Your Costs</h3>
                <p>
                  Metrics flow into a time-series backend where k8s-cost calculates per-pod,
                  per-namespace, and per-cluster costs using your actual cloud pricing rates.
                </p>
              </div>
              <div className="features-step-card">
                <div className="features-step-number">3</div>
                <h3>Optimize & Save</h3>
                <p>
                  Get right-sizing recommendations powered by P95 usage analysis. See exactly
                  which pods are over-provisioned and how much you can save.
                </p>
              </div>
            </div>
          </div>

          {/* Section 3: Core Feature Grid */}
          <div className="features-section">
            <div className="features-section-header">
              <h2>Core Features</h2>
              <p>Everything you need to understand and reduce Kubernetes costs</p>
            </div>
            <div className="features-grid">

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="2" y1="12" x2="22" y2="12" />
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                </svg>
                <h3>Multi-Cloud Pricing</h3>
                <p>
                  Built-in pricing presets for AWS, GCP, Azure, and OCI. Configure on-demand,
                  spot, and reserved instance rates — or define custom pricing per cluster.
                </p>
              </div>

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="3" width="7" height="7" />
                  <rect x="14" y="3" width="7" height="7" />
                  <rect x="3" y="14" width="7" height="7" />
                  <rect x="14" y="14" width="7" height="7" />
                </svg>
                <h3>Namespace-Level Breakdown</h3>
                <p>
                  See exactly where costs are allocated. Filter by namespace to isolate
                  application costs from system overhead like kube-system.
                </p>
              </div>

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
                  <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
                  <line x1="12" y1="22.08" x2="12" y2="12" />
                </svg>
                <h3>Right-Sizing Recommendations</h3>
                <p>
                  Automatic recommendations based on P95 resource usage with a 20% safety buffer.
                  Each suggestion shows current vs recommended requests and estimated savings.
                </p>
              </div>

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                  <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                </svg>
                <h3>Team Access Control</h3>
                <p>
                  Role-based access with Owner, Admin, Editor, and Viewer roles. Invite team
                  members, manage permissions, and maintain full audit control.
                </p>
              </div>

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
                </svg>
                <h3>Time-Series Analytics</h3>
                <p>
                  Powered by TimescaleDB for efficient storage and querying. Track cost trends
                  hourly, daily, or weekly across any time window.
                </p>
              </div>

              <div className="features-card">
                <svg className="features-card-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
                <h3>One Key Per Cluster</h3>
                <p>
                  Each cluster gets its own API key with automatic validation. Metrics are
                  isolated per-tenant — no cross-cluster data leakage, ever.
                </p>
              </div>

            </div>
          </div>

          {/* Section 4: Why k8s-cost */}
          <div className="features-section">
            <div className="features-section-header">
              <h2>Why k8s-cost</h2>
              <p>Built different from legacy cost tools</p>
            </div>
            <div className="features-advantages">

              <div className="features-advantage-card">
                <h3>Zero CRDs, Zero Sidecars</h3>
                <p>
                  Unlike Kubecost or OpenCost, there's no in-cluster Prometheus, no custom CRDs,
                  no sidecar containers. One lightweight agent binary — that's it.
                </p>
              </div>

              <div className="features-advantage-card">
                <h3>SaaS-Native Multi-Tenancy</h3>
                <p>
                  Built from day one for multi-tenant SaaS. Tenant-isolated data, plan-based
                  limits, and team management are core — not bolted on.
                </p>
              </div>

              <div className="features-advantage-card">
                <h3>Your Pricing, Your Rules</h3>
                <p>
                  Most tools use a generic pricing model. k8s-cost lets you configure exact rates
                  per provider, region, instance family, and pricing tier (spot, reserved, on-demand).
                </p>
              </div>

              <div className="features-advantage-card">
                <h3>Minutes to First Insight</h3>
                <p>
                  No lengthy setup. Create an account, generate an API key, deploy the agent.
                  Cost data starts flowing immediately.
                </p>
              </div>

            </div>
          </div>

          {/* Section 5: Bottom CTA */}
          <div className="features-bottom-cta">
            <h2>Ready to see where your Kubernetes budget goes?</h2>
            <div className="features-bottom-cta-buttons">
              <Link to="/sign-up" className="cta-button cta-button-primary">
                Start Free
              </Link>
              <Link to="/pricing" className="cta-button cta-button-secondary">
                View Pricing
              </Link>
            </div>
          </div>

        </div>
      </div>
    </div>
  )
}
