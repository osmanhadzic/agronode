import { memo } from 'react'
import type { DeviceMeta } from '../types/telemetry'

type DeviceMetaPanelProps = {
  meta: DeviceMeta | undefined
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

function rssiLabel(rssi: number): string {
  if (rssi >= -50) return 'Excellent'
  if (rssi >= -60) return 'Good'
  if (rssi >= -70) return 'Fair'
  return 'Weak'
}

export const DeviceMetaPanel = memo(function DeviceMetaPanel({ meta }: DeviceMetaPanelProps) {
  if (!meta) return null

  const items: { label: string; value: string }[] = []

  if (meta.fw) items.push({ label: 'Firmware', value: meta.fw })
  if (meta.ip) items.push({ label: 'IP', value: meta.ip })
  if (meta.rssi !== undefined)
    items.push({ label: 'Signal', value: `${meta.rssi} dBm (${rssiLabel(meta.rssi)})` })
  if (meta.uptime !== undefined)
    items.push({ label: 'Uptime', value: formatUptime(meta.uptime) })

  if (items.length === 0) return null

  return (
    <section className="device-meta-panel">
      {items.map(({ label, value }) => (
        <span key={label} className="device-meta-item">
          <span className="device-meta-label">{label}</span>
          <span className="device-meta-value">{value}</span>
        </span>
      ))}
    </section>
  )
})
