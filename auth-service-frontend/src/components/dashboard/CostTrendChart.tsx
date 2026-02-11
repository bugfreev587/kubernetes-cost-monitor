import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import type { CostTrend } from '../../types/cost'

interface CostTrendChartProps {
  data: CostTrend[]
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

export default function CostTrendChart({ data }: CostTrendChartProps) {
  if (data.length === 0) {
    return (
      <div className="chart-section">
        <h3>Cost Trend</h3>
        <div className="chart-empty">No trend data available</div>
      </div>
    )
  }

  const chartData = data.map((item) => ({
    ...item,
    formattedDate: formatDate(item.date),
  }))

  return (
    <div className="chart-section">
      <h3>Cost Trend</h3>
      <div className="chart-container">
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 20 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis
              dataKey="formattedDate"
              tick={{ fill: '#6b7280', fontSize: 12 }}
            />
            <YAxis
              tick={{ fill: '#6b7280', fontSize: 12 }}
              tickFormatter={(value) => `$${value.toFixed(0)}`}
            />
            <Tooltip
              formatter={(value) => formatCurrency(Number(value))}
              labelFormatter={(label) => `Date: ${label}`}
              contentStyle={{
                background: 'white',
                border: '1px solid #e5e7eb',
                borderRadius: 8,
                boxShadow: '0 4px 6px -1px rgba(0,0,0,0.1)',
              }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="cost"
              name="Total Cost"
              stroke="#646cff"
              strokeWidth={2}
              dot={{ fill: '#646cff', strokeWidth: 2, r: 4 }}
              activeDot={{ r: 6 }}
            />
            <Line
              type="monotone"
              dataKey="cpu_cost"
              name="CPU Cost"
              stroke="#4ade80"
              strokeWidth={2}
              dot={{ fill: '#4ade80', strokeWidth: 2, r: 3 }}
            />
            <Line
              type="monotone"
              dataKey="memory_cost"
              name="Memory Cost"
              stroke="#f59e0b"
              strokeWidth={2}
              dot={{ fill: '#f59e0b', strokeWidth: 2, r: 3 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
