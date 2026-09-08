import { memo } from 'react'

type SensorCardProps = {
  label: string
  value: number | null
  unit: string
}

export const SensorCard = memo(function SensorCard({ label, value, unit }: SensorCardProps) {
  const displayValue = value === null ? '--' : `${value.toFixed(1)} ${unit}`

  return (
    <article className="sensor-card">
      <p className="sensor-label">{label}</p>
      <p className="sensor-value">{displayValue}</p>
    </article>
  )
})
