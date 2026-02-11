import type { TimeWindow } from '../../types/cost'

interface TimeRangeSelectorProps {
  value: TimeWindow
  onChange: (window: TimeWindow) => void
}

export default function TimeRangeSelector({ value, onChange }: TimeRangeSelectorProps) {
  return (
    <div className="time-range-selector">
      <button
        className={`time-range-btn ${value === '7d' ? 'active' : ''}`}
        onClick={() => onChange('7d')}
      >
        7 Days
      </button>
      <button
        className={`time-range-btn ${value === '30d' ? 'active' : ''}`}
        onClick={() => onChange('30d')}
      >
        30 Days
      </button>
    </div>
  )
}
