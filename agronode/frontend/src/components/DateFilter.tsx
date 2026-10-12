import { memo, useState } from 'react'
import type { DateFilterPeriod } from '../api/telemetryApi'
import { useI18n } from '../i18n'

interface DateFilterProps {
  onFilterChange: (period: DateFilterPeriod, startDate?: string, endDate?: string) => void
  selectedPeriod: DateFilterPeriod
}

export const DateFilter = memo(function DateFilter({ onFilterChange, selectedPeriod }: DateFilterProps) {
  const { t } = useI18n()
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
    { value: 'hour', label: t('period.hour') },
    { value: 'day', label: t('period.day') },
    { value: 'week', label: t('period.week') },
    { value: 'month', label: t('period.month') },
    { value: 'year', label: t('period.year') },
    { value: 'custom', label: t('period.custom') },
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
            {t('common.reset')}
          </button>
        )}
      </div>

      {showCustomRange && (
        <div className="custom-range">
          <div className="date-inputs">
            <div className="date-input-group">
              <label htmlFor="start-date">{t('common.from')}</label>
              <input
                id="start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <div className="date-input-group">
              <label htmlFor="end-date">{t('common.to')}</label>
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
              {t('common.apply')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
})
