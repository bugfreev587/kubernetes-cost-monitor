import type { ToplineSummary, TimeWindow } from '../../types/cost'

interface SummaryCardsProps {
  topline: ToplineSummary | null
  window: TimeWindow
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function getEfficiencyColor(efficiency: number): string {
  if (efficiency >= 80) return '#059669' // green
  if (efficiency >= 60) return '#d97706' // amber
  return '#dc2626' // red
}

function getMonthlyProjection(cost: number, window: TimeWindow): number {
  const days = window === '7d' ? 7 : 30
  return (cost / days) * 30
}

export default function SummaryCards({ topline, window }: SummaryCardsProps) {
  if (!topline) {
    return (
      <div className="summary-cards">
        <div className="summary-card">
          <div className="summary-card-label">Total Cost</div>
          <div className="summary-card-value">--</div>
        </div>
        <div className="summary-card">
          <div className="summary-card-label">CPU Cost</div>
          <div className="summary-card-value">--</div>
        </div>
        <div className="summary-card">
          <div className="summary-card-label">Memory Cost</div>
          <div className="summary-card-value">--</div>
        </div>
        <div className="summary-card">
          <div className="summary-card-label">Efficiency</div>
          <div className="summary-card-value">--</div>
        </div>
      </div>
    )
  }

  const monthlyProjection = getMonthlyProjection(topline.total_cost, window)

  return (
    <div className="summary-cards">
      <div className="summary-card">
        <div className="summary-card-label">Total Cost</div>
        <div className="summary-card-value">{formatCurrency(topline.total_cost)}</div>
        <div className="summary-card-subtext">
          ~{formatCurrency(monthlyProjection)}/mo projected
        </div>
      </div>
      <div className="summary-card">
        <div className="summary-card-label">CPU Cost</div>
        <div className="summary-card-value">{formatCurrency(topline.cpu_cost)}</div>
        <div className="summary-card-subtext">
          {((topline.cpu_cost / topline.total_cost) * 100).toFixed(1)}% of total
        </div>
      </div>
      <div className="summary-card">
        <div className="summary-card-label">Memory Cost</div>
        <div className="summary-card-value">{formatCurrency(topline.memory_cost)}</div>
        <div className="summary-card-subtext">
          {((topline.memory_cost / topline.total_cost) * 100).toFixed(1)}% of total
        </div>
      </div>
      <div className="summary-card">
        <div className="summary-card-label">Efficiency</div>
        <div
          className="summary-card-value"
          style={{ color: getEfficiencyColor(topline.efficiency) }}
        >
          {topline.efficiency.toFixed(1)}%
        </div>
        <div className="summary-card-subtext">
          resource utilization
        </div>
      </div>
    </div>
  )
}
