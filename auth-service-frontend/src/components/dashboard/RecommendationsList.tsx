import { useState } from 'react'
import type { Recommendation } from '../../types/cost'

interface RecommendationsListProps {
  data: Recommendation[]
  onApply: (id: number) => Promise<boolean>
  onDismiss: (id: number) => Promise<boolean>
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function formatResource(value: number, type: 'cpu' | 'memory'): string {
  if (type === 'cpu') {
    return `${value}m`
  }
  // Memory in Mi
  return `${value}Mi`
}

export default function RecommendationsList({ data, onApply, onDismiss }: RecommendationsListProps) {
  const [actionInProgress, setActionInProgress] = useState<number | null>(null)

  const handleApply = async (id: number) => {
    setActionInProgress(id)
    await onApply(id)
    setActionInProgress(null)
  }

  const handleDismiss = async (id: number) => {
    setActionInProgress(id)
    await onDismiss(id)
    setActionInProgress(null)
  }

  if (data.length === 0) {
    return (
      <div className="recommendations-section">
        <h3>Recommendations</h3>
        <div className="recommendations-empty">
          <div className="empty-icon">&#10003;</div>
          <p>No open recommendations</p>
          <span>Your resources are well-optimized!</span>
        </div>
      </div>
    )
  }

  return (
    <div className="recommendations-section">
      <h3>Recommendations</h3>
      <div className="recommendations-list">
        {data.map((rec) => (
          <div key={rec.id} className="recommendation-card">
            <div className="recommendation-header">
              <span className="recommendation-pod">{rec.pod_name}</span>
              <span className="recommendation-savings">
                Save {formatCurrency(rec.estimated_savings)}/day
              </span>
            </div>
            <div className="recommendation-details">
              <span className="recommendation-namespace">{rec.namespace}</span>
              <span className="recommendation-reason">{rec.reason}</span>
            </div>
            <div className="recommendation-resources">
              <div className="resource-change">
                <span className="resource-label">CPU:</span>
                <span className="resource-current">{formatResource(rec.current_cpu, 'cpu')}</span>
                <span className="resource-arrow">-&gt;</span>
                <span className="resource-recommended">{formatResource(rec.recommended_cpu, 'cpu')}</span>
              </div>
              <div className="resource-change">
                <span className="resource-label">Memory:</span>
                <span className="resource-current">{formatResource(rec.current_memory, 'memory')}</span>
                <span className="resource-arrow">-&gt;</span>
                <span className="resource-recommended">{formatResource(rec.recommended_memory, 'memory')}</span>
              </div>
            </div>
            <div className="recommendation-actions">
              <button
                className="btn btn-small btn-primary"
                onClick={() => handleApply(rec.id)}
                disabled={actionInProgress === rec.id}
              >
                {actionInProgress === rec.id ? 'Applying...' : 'Apply'}
              </button>
              <button
                className="btn btn-small btn-secondary"
                onClick={() => handleDismiss(rec.id)}
                disabled={actionInProgress === rec.id}
              >
                Dismiss
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
