export interface DeviceMeta {
  fw?: string
  ip?: string
  rssi?: number
  uptime?: number
}

export interface TelemetryReading {
  deviceId: string
  temperature: number
  humidity: number
  sensors?: Record<string, number>
  meta?: DeviceMeta
  createdAt: string
}
