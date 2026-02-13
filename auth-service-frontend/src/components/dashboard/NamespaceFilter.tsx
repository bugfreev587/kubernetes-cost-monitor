import { useState, useRef, useEffect } from 'react'

const SYSTEM_NAMESPACES = ['kube-system', 'kube-public', 'kube-node-lease']

interface NamespaceFilterProps {
  namespaces: string[]
  selected: string[]
  onChange: (selected: string[]) => void
}

export default function NamespaceFilter({ namespaces, selected, onChange }: NamespaceFilterProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const isSystem = (ns: string) => SYSTEM_NAMESPACES.includes(ns)

  const sorted = [...namespaces].sort((a, b) => {
    const aSystem = isSystem(a)
    const bSystem = isSystem(b)
    if (aSystem !== bSystem) return aSystem ? 1 : -1
    return a.localeCompare(b)
  })

  const allSelected = selected.length === 0 || selected.length === namespaces.length

  const toggle = (ns: string) => {
    if (allSelected) {
      // Switching from all to single selection
      onChange([ns])
    } else if (selected.includes(ns)) {
      const next = selected.filter(s => s !== ns)
      onChange(next.length === 0 ? [] : next)
    } else {
      const next = [...selected, ns]
      onChange(next.length === namespaces.length ? [] : next)
    }
  }

  const label = allSelected
    ? 'All Namespaces'
    : selected.length === 1
      ? selected[0]
      : `${selected.length} Namespaces`

  return (
    <div className="namespace-filter" ref={ref}>
      <button className="namespace-filter-trigger" onClick={() => setOpen(!open)}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
        </svg>
        {label}
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      {open && (
        <div className="namespace-filter-dropdown">
          <div className="namespace-filter-actions">
            <button onClick={() => onChange([])}>Select All</button>
            <button onClick={() => onChange(sorted.length > 0 ? [sorted[0]] : [])}>Clear</button>
          </div>
          {sorted.map(ns => {
            const checked = allSelected || selected.includes(ns)
            return (
              <label key={ns} className="namespace-filter-item">
                <input type="checkbox" checked={checked} onChange={() => toggle(ns)} />
                <span className={`namespace-dot ${isSystem(ns) ? 'system' : 'app'}`} />
                <span className="namespace-filter-name">{ns}</span>
              </label>
            )
          })}
        </div>
      )}
    </div>
  )
}
