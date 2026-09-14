export interface DeviceMeta {
  fw?: string
  ip?: string
  rssi?: number
  uptime?: number
}

export interface TelemetryReading {
  deviceId: string
  sensorId?: string
  temperature?: number | null
  humidity?: number | null
  sensors?: Record<string, number>
  meta?: DeviceMeta
  createdAt: string
}

export interface DeviceSummary {
  id: number
  deviceId: string
  deviceType?: 'publisher' | 'receiver' | 'unknown'
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

export interface SensorTrigger {
  deviceId: string
  sensorId: string
  min?: number
  max?: number
  targetDeviceId?: string
}

export interface TriggerListItem {
  sensorId: string
  min?: number
  max?: number
  targetDeviceId?: string
}

export interface TriggerListResponse {
  deviceId: string
  triggers: TriggerListItem[]
}

export interface DeviceSensor {
  id: number
  deviceId: string
  sensorId: string
  createdAt: string
  updatedAt: string
}
