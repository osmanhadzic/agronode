import { memo } from 'react'

interface DataModeSelectorProps {
  mode: 'live' | 'history'
  onChange: (mode: 'live' | 'history') => void
}

export const DataModeSelector = memo(function DataModeSelector({ mode, onChange }: DataModeSelectorProps) {
  return (
    <div className="data-mode-selector">
      <button
        className={`mode-btn ${mode === 'live' ? 'active' : ''}`}
        onClick={() => onChange('live')}
      >
        📡 Live Data
      </button>
      <button
        className={`mode-btn ${mode === 'history' ? 'active' : ''}`}
        onClick={() => onChange('history')}
      >
        📊 History
      </button>

      <style>{`
        .data-mode-selector {
          display: flex;
          gap: 0.5rem;
          margin: 1rem 0;
          padding: 0.5rem;
          background-color: #f8f9fa;
          border-radius: 8px;
          justify-content: center;
        }

        .mode-btn {
          padding: 0.75rem 2rem;
          border: 2px solid #dee2e6;
          background-color: white;
          border-radius: 6px;
          cursor: pointer;
          font-size: 1rem;
          font-weight: 500;
          transition: all 0.2s;
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }

        .mode-btn:hover {
          background-color: #e9ecef;
          transform: translateY(-1px);
        }

        .mode-btn.active {
          background-color: #007bff;
          color: white;
          border-color: #007bff;
          box-shadow: 0 2px 4px rgba(0, 123, 255, 0.3);
        }
      `}</style>
    </div>
  )
})
