import { useState } from 'react'
import type { DateFilterPeriod } from '../api/telemetryApi'

interface DateFilterProps {
  onFilterChange: (period: DateFilterPeriod, startDate?: string, endDate?: string) => void
  selectedPeriod: DateFilterPeriod
}

export function DateFilter({ onFilterChange, selectedPeriod }: DateFilterProps) {
  const [showCustomRange, setShowCustomRange] = useState(false)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  const handlePeriodClick = (period: DateFilterPeriod) => {
    if (period === 'custom') {
      setShowCustomRange(true)
    } else {
      setShowCustomRange(false)
      onFilterChange(period)
    }
  }

  const handleCustomApply = () => {
    if (startDate && endDate) {
      const start = new Date(startDate).toISOString()
      const end = new Date(endDate + 'T23:59:59').toISOString()
      onFilterChange('custom', start, end)
    }
  }

  const handleReset = () => {
    setShowCustomRange(false)
    setStartDate('')
    setEndDate('')
    onFilterChange('')
  }

  const periods: { value: DateFilterPeriod; label: string }[] = [
    { value: 'hour', label: 'Sat' },
    { value: 'day', label: 'Dan' },
    { value: 'week', label: 'Sedmica' },
    { value: 'month', label: 'Mjesec' },
    { value: 'year', label: 'Godina' },
    { value: 'custom', label: 'Custom' },
  ]

  return (
    <div className="date-filter">
      <div className="filter-buttons">
        {periods.map((period) => (
          <button
            key={period.value}
            className={`filter-btn ${selectedPeriod === period.value ? 'active' : ''}`}
            onClick={() => handlePeriodClick(period.value)}
          >
            {period.label}
          </button>
        ))}
        {selectedPeriod && (
          <button className="filter-btn reset" onClick={handleReset}>
            Reset
          </button>
        )}
      </div>

      {showCustomRange && (
        <div className="custom-range">
          <div className="date-inputs">
            <div className="date-input-group">
              <label htmlFor="start-date">Od:</label>
              <input
                id="start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <div className="date-input-group">
              <label htmlFor="end-date">Do:</label>
              <input
                id="end-date"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
              />
            </div>
            <button
              className="apply-btn"
              onClick={handleCustomApply}
              disabled={!startDate || !endDate}
            >
              Primijeni
            </button>
          </div>
        </div>
      )}

      <style>{`
        .date-filter {
          margin: 1rem 0;
          padding: 1rem;
          background-color: #f5f5f5;
          border-radius: 8px;
        }

        .filter-buttons {
          display: flex;
          gap: 0.5rem;
          flex-wrap: wrap;
        }

        .filter-btn {
          padding: 0.5rem 1rem;
          border: 2px solid #ddd;
          background-color: white;
          border-radius: 4px;
          cursor: pointer;
          font-size: 0.9rem;
          transition: all 0.2s;
        }

        .filter-btn:hover {
          background-color: #e9ecef;
        }

        .filter-btn.active {
          background-color: #007bff;
          color: white;
          border-color: #007bff;
        }

        .filter-btn.reset {
          background-color: #dc3545;
          color: white;
          border-color: #dc3545;
        }

        .filter-btn.reset:hover {
          background-color: #c82333;
          border-color: #bd2130;
        }

        .custom-range {
          margin-top: 1rem;
          padding: 1rem;
          background-color: white;
          border-radius: 4px;
        }

        .date-inputs {
          display: flex;
          gap: 1rem;
          align-items: flex-end;
          flex-wrap: wrap;
        }

        .date-input-group {
          display: flex;
          flex-direction: column;
          gap: 0.25rem;
        }

        .date-input-group label {
          font-size: 0.85rem;
          font-weight: 500;
          color: #495057;
        }

        .date-input-group input {
          padding: 0.5rem;
          border: 1px solid #ced4da;
          border-radius: 4px;
          font-size: 0.9rem;
        }

        .apply-btn {
          padding: 0.5rem 1.5rem;
          background-color: #28a745;
          color: white;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 0.9rem;
          transition: background-color 0.2s;
        }

        .apply-btn:hover:not(:disabled) {
          background-color: #218838;
        }

        .apply-btn:disabled {
          background-color: #6c757d;
          cursor: not-allowed;
          opacity: 0.6;
        }
      `}</style>
    </div>
  )
}
