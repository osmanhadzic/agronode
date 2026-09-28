import { memo } from 'react'

interface DataModeSelectorProps {
  mode: 'live' | 'history'
  onChange: (mode: 'live' | 'history') => void
}

export const DataModeSelector = memo(function DataModeSelector({ mode, onChange }: DataModeSelectorProps) {
  return (
    <div className="data-mode-selector">
      <button
        type="button"
        className={`mode-btn ${mode === 'live' ? 'active' : ''}`}
        onClick={() => onChange('live')}
      >
        Live Data
      </button>
      <button
        type="button"
        className={`mode-btn ${mode === 'history' ? 'active' : ''}`}
        onClick={() => onChange('history')}
      >
        History
      </button>
    </div>
  )
})
