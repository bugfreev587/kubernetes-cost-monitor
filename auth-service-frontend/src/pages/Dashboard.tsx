import { useUser } from '@clerk/clerk-react'
import { useNavigate } from 'react-router-dom'
import Navbar from '../components/Navbar'
import { useCostData } from '../hooks/useCostData'
import {
  TimeRangeSelector,
  SummaryCards,
  CostByNamespaceChart,
  CostTrendChart,
  TopCostDriversTable,
  RecommendationsList,
} from '../components/dashboard'
import '../App.css'
import './Dashboard.css'

export default function Dashboard() {
  const { isLoaded, user } = useUser()
  const navigate = useNavigate()
  const {
    topline,
    namespaceAllocations,
    trends,
    utilization,
    recommendations,
    loading,
    error,
    window,
    setWindow,
    refresh,
    applyRecommendation,
    dismissRecommendation,
  } = useCostData()

  if (!isLoaded) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="dashboard-container">
            <div className="dashboard-loading">
              <div className="spinner"></div>
              <p>Loading...</p>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Show loading state
  if (loading) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="dashboard-container">
            <div className="dashboard-header">
              <h1>Dashboard</h1>
            </div>
            <div className="dashboard-loading">
              <div className="spinner"></div>
              <p>Loading cost data...</p>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Show error state
  if (error) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="dashboard-container">
            <div className="dashboard-header">
              <h1>Dashboard</h1>
            </div>
            <div className="dashboard-error">
              <h3>Unable to load cost data</h3>
              <p>{error}</p>
              <button className="btn btn-primary" onClick={refresh} style={{ marginTop: '1rem' }}>
                Try Again
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Check if we have any data at all
  const hasData = topline || namespaceAllocations.length > 0 || trends.length > 0 || utilization.length > 0

  // Empty state when no data
  if (!hasData) {
    return (
      <div className="page-container">
        <Navbar />
        <div className="page-content">
          <div className="dashboard-container">
            <div className="dashboard-header">
              <h1>Dashboard</h1>
            </div>
            <div className="dashboard-empty">
              <h2>Welcome{user?.firstName ? `, ${user.firstName}` : ''}!</h2>
              <p>
                No cost data available yet. Deploy the cost-agent to your Kubernetes cluster to start
                monitoring costs.
              </p>
              <button className="btn btn-primary" onClick={() => navigate('/management')}>
                Set Up Cost Agent
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      <Navbar />
      <div className="page-content">
        <div className="dashboard-container">
          {/* Header */}
          <div className="dashboard-header">
            <h1>Dashboard</h1>
            <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
              <button className="refresh-btn" onClick={refresh} title="Refresh data">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M1 4v6h6M23 20v-6h-6" />
                  <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15" />
                </svg>
                Refresh
              </button>
              <TimeRangeSelector value={window} onChange={setWindow} />
            </div>
          </div>

          {/* Summary Cards */}
          <SummaryCards topline={topline} window={window} />

          {/* Charts Row */}
          <div className="dashboard-charts">
            <CostByNamespaceChart data={namespaceAllocations} />
            <CostTrendChart data={trends} />
          </div>

          {/* Tables Row */}
          <div className="dashboard-tables">
            <TopCostDriversTable data={utilization} />
            <RecommendationsList
              data={recommendations}
              onApply={applyRecommendation}
              onDismiss={dismissRecommendation}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
