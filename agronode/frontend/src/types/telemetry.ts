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

export interface DeviceSummary {
  id: number
  deviceId: string
  status: string
  firmwareVersion?: string
  metadata?: {
    battery?: number
    signalStrength?: number
    hardware?: Record<string, string>
  }
  lastSeen?: string
  createdAt: string
  updatedAt: string
}

export interface DeviceStatusEvent {
  deviceId: string
  oldStatus: string
  newStatus: string
  eventType: 'device.online' | 'device.offline' | string
  timestamp: string
}
