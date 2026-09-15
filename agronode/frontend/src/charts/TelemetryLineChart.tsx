import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import type { TelemetryReading } from '../types/telemetry'

type TelemetryLineChartProps = {
  data: TelemetryReading[]
  selectedSensors: string[]
}

type ChartPoint = {
  time: string
  value: number
}

const lineColors = ['#2563eb', '#16a34a', '#a855f7', '#f59e0b', '#ef4444', '#0ea5e9']

function getReadingSensorValues(reading: TelemetryReading): Record<string, number> {
  const readingSensors = reading.sensors ?? {}
  if (Object.keys(readingSensors).length > 0) {
    return readingSensors
  }

  const fallback: Record<string, number> = {}

  if (!reading.sensorId) {
    if (typeof reading.temperature === 'number') {
      fallback.temperature = reading.temperature
    }
    if (typeof reading.humidity === 'number') {
      fallback.humidity = reading.humidity
    }

    return fallback
  }

  if (
    (reading.sensorId === 'temperature' || reading.sensorId === 'dht11-temp') &&
    typeof reading.temperature === 'number'
  ) {
    fallback[reading.sensorId] = reading.temperature
  }

  if (
    (reading.sensorId === 'humidity' ||
      reading.sensorId === 'humidity_dht11' ||
      reading.sensorId === 'dht11-humidity') &&
    typeof reading.humidity === 'number'
  ) {
    fallback[reading.sensorId] = reading.humidity
  }

  return fallback
}

function toSensorLabel(sensorKey: string): string {
  if (sensorKey === 'co2') {
    return 'CO₂'
  }

  if (sensorKey === 'dht11-temp') {
    return 'DHT11 Temp'
  }

  if (sensorKey === 'dht11-humidity' || sensorKey === 'humidity_dht11') {
    return 'DHT11 Humidity'
  }

  if (sensorKey === 'signal_strength') {
    return 'Signal Strength'
  }

  return sensorKey
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function TelemetryLineChart({ data, selectedSensors }: TelemetryLineChartProps) {
  const pointsBySensor = selectedSensors.reduce<Record<string, ChartPoint[]>>(
    (result, sensorKey) => {
      result[sensorKey] = [...data]
        .reverse()
        .flatMap((reading) => {
          const sensorValues = getReadingSensorValues(reading)
          const value = sensorValues[sensorKey]

          if (value === undefined) {
            return []
          }

          return [
            {
              time: new Date(reading.createdAt).toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
              }),
              value,
            },
          ]
        })

      return result
    },
    {},
  )

  return (
    <div className="chart-panel">
      <h2 className="chart-title">Live Telemetry</h2>
      {selectedSensors.length === 0 ? (
        <p className="dashboard-message">No measurement selected.</p>
      ) : (
        <div className="chart-grid">
          {selectedSensors.map((sensorKey, index) => (
            <section key={sensorKey} className="chart-card">
              <h3 className="chart-card-title">{toSensorLabel(sensorKey)}</h3>
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={pointsBySensor[sensorKey] ?? []}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip />
                  <Line
                    type="monotone"
                    dataKey="value"
                    stroke={lineColors[index % lineColors.length]}
                    strokeWidth={2}
                    dot={false}
                    name={toSensorLabel(sensorKey)}
                  />
                </LineChart>
              </ResponsiveContainer>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
