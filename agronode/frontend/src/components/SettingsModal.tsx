import { memo, useEffect } from 'react'

import { useI18n } from '../i18n'
import type { OrganizationSummary } from '../types/organization'
import { LanguageSwitcher } from './LanguageSwitcher'

type SettingsModalProps = {
  onClose: () => void
  organizations?: OrganizationSummary[]
  selectedOrganizationScope: number | null
  onOrganizationChange: (organizationId: number | null) => void
}

export const SettingsModal = memo(function SettingsModal({
  onClose,
  organizations,
  selectedOrganizationScope,
  onOrganizationChange,
}: SettingsModalProps) {
  const { t } = useI18n()

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <div role="dialog" aria-modal="true" className="settings-backdrop" onClick={onClose}>
      <div className="settings-modal" onClick={(event) => event.stopPropagation()}>
        <div className="settings-modal-header">
          <h3>{t('settings.title')}</h3>
          <button type="button" onClick={onClose}>{t('common.close')}</button>
        </div>
        <div className="settings-row">
          <LanguageSwitcher />
        </div>
        {organizations && (
          <div className="settings-row">
            <label className="language-switcher">
              <span className="language-switcher-label">{t('dash.organization')}</span>
              <select
                value={selectedOrganizationScope ?? ''}
                onChange={(event) => {
                  const value = event.target.value
                  onOrganizationChange(value === '' ? null : Number(value))
                }}
              >
                <option value="">{t('dash.allOrganizations')}</option>
                {organizations.map((organization) => (
                  <option key={organization.id} value={organization.id}>
                    {organization.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}
      </div>
    </div>
  )
})
