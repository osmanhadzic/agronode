import { memo } from 'react'

import { useI18n } from '../i18n'

interface DataModeSelectorProps {
  mode: 'live' | 'history'
  onChange: (mode: 'live' | 'history') => void
}

export const DataModeSelector = memo(function DataModeSelector({ mode, onChange }: DataModeSelectorProps) {
  const { t } = useI18n()

  return (
    <div className="data-mode-selector">
      <button
        type="button"
        className={`mode-btn ${mode === 'live' ? 'active' : ''}`}
        onClick={() => onChange('live')}
      >
        {t('mode.live')}
      </button>
      <button
        type="button"
        className={`mode-btn ${mode === 'history' ? 'active' : ''}`}
        onClick={() => onChange('history')}
      >
        {t('mode.history')}
      </button>
    </div>
  )
})
