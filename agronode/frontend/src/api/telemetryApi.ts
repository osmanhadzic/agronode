import axios from 'axios'

import { httpClient } from './httpClient'
import type { DeviceSummary, TelemetryReading } from '../types/telemetry'

export async function fetchAllTelemetry(): Promise<TelemetryReading[]> {
  const { data } = await httpClient.get<TelemetryReading[]>('/api/data')
  return data
}

export async function fetchTelemetryByDeviceId(
  deviceId: string,
): Promise<TelemetryReading[]> {
  const { data } = await httpClient.get<TelemetryReading[]>(`/api/data/${deviceId}`)
  return data
}

export async function fetchLatestTelemetryByDeviceId(
  deviceId: string,
): Promise<TelemetryReading | null> {
  try {
    const { data } = await httpClient.get<TelemetryReading>(
      `/api/latest/${deviceId}`,
    )
    return data
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null
    }

    throw error
  }
}

export async function fetchDevices(): Promise<DeviceSummary[]> {
  const { data } = await httpClient.get<DeviceSummary[]>('/api/devices')
  return data
}
