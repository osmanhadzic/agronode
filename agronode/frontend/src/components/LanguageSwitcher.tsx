import { memo } from 'react'

import { LANGUAGES, useI18n, type Language } from '../i18n'

export const LanguageSwitcher = memo(function LanguageSwitcher() {
  const { t, language, setLanguage } = useI18n()

  return (
    <label className="language-switcher">
      <span className="language-switcher-label">{t('common.language')}</span>
      <select
        value={language}
        onChange={(event) => setLanguage(event.target.value as Language)}
        aria-label={t('common.language')}
      >
        {LANGUAGES.map((item) => (
          <option key={item.code} value={item.code}>
            {item.label}
          </option>
        ))}
      </select>
    </label>
  )
})
