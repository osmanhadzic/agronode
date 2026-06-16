import { memo } from 'react'

type DeviceSelectorProps = {
  devices: string[]
  selectedDeviceId: string
  onChange: (deviceId: string) => void
}

export const DeviceSelector = memo(function DeviceSelector({
  devices,
  selectedDeviceId,
  onChange,
}: DeviceSelectorProps) {
  return (
    <div className="device-selector">
      <label htmlFor="device-select">Device</label>
      <select
        id="device-select"
        value={selectedDeviceId}
        onChange={(event) => onChange(event.target.value)}
      >
        {devices.map((deviceId) => (
          <option key={deviceId} value={deviceId}>
            {deviceId}
          </option>
        ))}
      </select>
    </div>
  )
})
