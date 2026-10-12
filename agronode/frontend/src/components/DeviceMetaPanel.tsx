import { memo } from 'react'
import { useI18n, type TranslationKey } from '../i18n'
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

function rssiLabelKey(rssi: number): TranslationKey {
  if (rssi >= -50) return 'meta.rssi.excellent'
  if (rssi >= -60) return 'meta.rssi.good'
  if (rssi >= -70) return 'meta.rssi.fair'
  return 'meta.rssi.weak'
}

export const DeviceMetaPanel = memo(function DeviceMetaPanel({ meta }: DeviceMetaPanelProps) {
  const { t } = useI18n()

  if (!meta) return null

  const items: { label: string; value: string }[] = []

  if (meta.fw) items.push({ label: 'Firmware', value: meta.fw })
  if (meta.ip) items.push({ label: 'IP', value: meta.ip })
  if (meta.rssi !== undefined)
    items.push({ label: t('meta.signal'), value: `${meta.rssi} dBm (${t(rssiLabelKey(meta.rssi))})` })
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
