import { memo, useState } from 'react'
import type { DateFilterPeriod } from '../api/telemetryApi'

interface DateFilterProps {
  onFilterChange: (period: DateFilterPeriod, startDate?: string, endDate?: string) => void
  selectedPeriod: DateFilterPeriod
}

export const DateFilter = memo(function DateFilter({ onFilterChange, selectedPeriod }: DateFilterProps) {
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
            type="button"
            className={`filter-btn ${selectedPeriod === period.value ? 'active' : ''}`}
            onClick={() => handlePeriodClick(period.value)}
          >
            {period.label}
          </button>
        ))}
        {selectedPeriod && (
          <button type="button" className="filter-btn reset" onClick={handleReset}>
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
              type="button"
              onClick={handleCustomApply}
              disabled={!startDate || !endDate}
            >
              Primijeni
            </button>
          </div>
        </div>
      )}
    </div>
  )
})
