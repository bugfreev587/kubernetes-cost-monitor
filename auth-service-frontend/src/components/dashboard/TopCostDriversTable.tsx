import type { UtilizationMetric } from '../../types/cost'

interface TopCostDriversTableProps {
  data: UtilizationMetric[]
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(value)
}

function getUtilizationColor(utilization: number): string {
  if (utilization > 100) return '#dc2626' // red - over-committed
  if (utilization >= 70) return '#d97706' // orange - cautious
  return '#059669' // green - safe
}

export default function TopCostDriversTable({ data }: TopCostDriversTableProps) {
  if (data.length === 0) {
    return (
      <div className="table-section">
        <h3>Top Cost Drivers</h3>
        <div className="table-empty">No utilization data available</div>
      </div>
    )
  }

  // Sort by estimated cost descending and take top 10
  const topDrivers = [...data]
    .sort((a, b) => b.estimated_cost - a.estimated_cost)
    .slice(0, 10)

  return (
    <div className="table-section">
      <h3>Top Cost Drivers</h3>
      <div className="table-container">
        <table className="cost-table">
          <thead>
            <tr>
              <th>Pod</th>
              <th>Namespace</th>
              <th>CPU Util</th>
              <th>Memory Util</th>
              <th>Est. Cost/hr</th>
            </tr>
          </thead>
          <tbody>
            {topDrivers.map((metric, index) => (
              <tr key={`${metric.namespace}-${metric.pod_name}-${index}`}>
                <td className="pod-name" title={metric.pod_name}>
                  {metric.pod_name.length > 30
                    ? `${metric.pod_name.slice(0, 27)}...`
                    : metric.pod_name}
                </td>
                <td>{metric.namespace}</td>
                <td>
                  <span
                    className="utilization-badge"
                    style={{ color: getUtilizationColor(metric.cpu_utilization) }}
                  >
                    {metric.cpu_utilization.toFixed(1)}%
                  </span>
                </td>
                <td>
                  <span
                    className="utilization-badge"
                    style={{ color: getUtilizationColor(metric.memory_utilization) }}
                  >
                    {metric.memory_utilization.toFixed(1)}%
                  </span>
                </td>
                <td className="cost-cell">{formatCurrency(metric.estimated_cost)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
