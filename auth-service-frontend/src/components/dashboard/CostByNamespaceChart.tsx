import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts'
import type { NamespaceAllocation } from '../../types/cost'

interface CostByNamespaceChartProps {
  data: NamespaceAllocation[]
}

const COLORS = [
  '#646cff',
  '#4ade80',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#06b6d4',
  '#ec4899',
  '#84cc16',
]

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

export default function CostByNamespaceChart({ data }: CostByNamespaceChartProps) {
  if (data.length === 0) {
    return (
      <div className="chart-section">
        <h3>Cost by Namespace</h3>
        <div className="chart-empty">No namespace data available</div>
      </div>
    )
  }

  // Sort by total cost descending and take top 8
  const chartData = [...data]
    .sort((a, b) => b.total_cost - a.total_cost)
    .slice(0, 8)
    .map((item, index) => ({
      ...item,
      color: COLORS[index % COLORS.length],
    }))

  return (
    <div className="chart-section">
      <h3>Cost by Namespace</h3>
      <div className="chart-container">
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 60 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis
              dataKey="namespace"
              tick={{ fill: '#6b7280', fontSize: 12 }}
              angle={-45}
              textAnchor="end"
              height={60}
              interval={0}
            />
            <YAxis
              tick={{ fill: '#6b7280', fontSize: 12 }}
              tickFormatter={(value) => `$${value.toFixed(0)}`}
            />
            <Tooltip
              formatter={(value) => formatCurrency(Number(value))}
              labelStyle={{ color: '#374151', fontWeight: 600 }}
              contentStyle={{
                background: 'white',
                border: '1px solid #e5e7eb',
                borderRadius: 8,
                boxShadow: '0 4px 6px -1px rgba(0,0,0,0.1)',
              }}
            />
            <Bar dataKey="total_cost" name="Total Cost" radius={[4, 4, 0, 0]}>
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
