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
  sensors: Record<string, number>
}

const lineColors = ['#2563eb', '#16a34a', '#a855f7', '#f59e0b', '#ef4444', '#0ea5e9']

function toSensorLabel(sensorKey: string): string {
  if (sensorKey === 'co2') {
    return 'CO₂'
  }

  return sensorKey
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function TelemetryLineChart({ data, selectedSensors }: TelemetryLineChartProps) {
  const points: ChartPoint[] = [...data]
    .reverse()
    .map((reading) => ({
      time: new Date(reading.createdAt).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      }),
      sensors: {
        temperature: reading.temperature,
        humidity: reading.humidity,
        ...(reading.sensors ?? {}),
      },
    }))

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
                <LineChart data={points}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip />
                  <Line
                    type="monotone"
                    dataKey={`sensors.${sensorKey}`}
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
