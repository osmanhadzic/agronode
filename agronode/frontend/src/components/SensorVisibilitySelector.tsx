import { memo } from 'react'

type SensorVisibilitySelectorProps = {
  sensors: string[]
  selectedSensors: string[]
  onToggleSensor: (sensorKey: string) => void
}

function toSensorLabel(sensorKey: string): string {
  if (sensorKey === 'co2') {
    return 'CO₂'
  }

  return sensorKey
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export const SensorVisibilitySelector = memo(function SensorVisibilitySelector({
  sensors,
  selectedSensors,
  onToggleSensor,
}: SensorVisibilitySelectorProps) {
  if (sensors.length === 0) {
    return null
  }

  return (
    <section className="sensor-visibility-panel">
      <h2 className="sensor-visibility-title">Choose data to display</h2>
      <div className="sensor-visibility-list">
        {sensors.map((sensorKey) => {
          const checked = selectedSensors.includes(sensorKey)

          return (
            <label key={sensorKey} className="sensor-visibility-item">
              <input
                type="checkbox"
                checked={checked}
                onChange={() => onToggleSensor(sensorKey)}
              />
              <span>{toSensorLabel(sensorKey)}</span>
            </label>
          )
        })}
      </div>
    </section>
  )
})
