import { memo } from 'react'

import { useI18n } from '../i18n'

type DeviceSelectorProps = {
  devices: string[]
  selectedDeviceId: string
  onChange: (deviceId: string) => void
  deviceTypes?: Record<string, string>
}

export const DeviceSelector = memo(function DeviceSelector({
  devices,
  selectedDeviceId,
  onChange,
  deviceTypes,
}: DeviceSelectorProps) {
  const { t } = useI18n()
  const formatDeviceLabel = (deviceId: string) => {
    const deviceType = deviceTypes?.[deviceId]
    if (!deviceType || deviceType === 'unknown') {
      return deviceId
    }

    return `${deviceId} (${deviceType})`
  }

  return (
    <div className="device-selector">
      <label htmlFor="device-select">{t('device.label')}</label>
      <select
        id="device-select"
        value={selectedDeviceId}
        onChange={(event) => onChange(event.target.value)}
      >
        {devices.map((deviceId) => (
          <option key={deviceId} value={deviceId}>
            {formatDeviceLabel(deviceId)}
          </option>
        ))}
      </select>
    </div>
  )
})
