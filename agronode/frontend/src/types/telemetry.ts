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

export interface SensorTrigger {
  deviceId: string
  sensor: string
  min?: number
  max?: number
}

export interface TriggerListItem {
  sensor: string
  min?: number
  max?: number
}

export interface TriggerListResponse {
  deviceId: string
  triggers: TriggerListItem[]
}
