import {
  CartesianGrid,
  Legend,
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
}

type ChartPoint = {
  time: string
  temperature: number
  humidity: number
}

export function TelemetryLineChart({ data }: TelemetryLineChartProps) {
  const points: ChartPoint[] = [...data]
    .reverse()
    .map((reading) => ({
      time: new Date(reading.createdAt).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      }),
      temperature: reading.temperature,
      humidity: reading.humidity,
    }))

  return (
    <div className="chart-panel">
      <h2 className="chart-title">Live Telemetry</h2>
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={points}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="time" />
          <YAxis />
          <Tooltip />
          <Legend />
          <Line
            type="monotone"
            dataKey="temperature"
            stroke="#2563eb"
            strokeWidth={2}
            dot={false}
            name="Temperature (°C)"
          />
          <Line
            type="monotone"
            dataKey="humidity"
            stroke="#16a34a"
            strokeWidth={2}
            dot={false}
            name="Humidity (%)"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
