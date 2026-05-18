export interface TelemetryReading {
  deviceId: string
  temperature: number
  humidity: number
  sensors?: Record<string, number>
  createdAt: string
}
