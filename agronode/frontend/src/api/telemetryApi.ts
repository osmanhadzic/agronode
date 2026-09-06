import axios from 'axios'

import { httpClient } from './httpClient'
import type {
  SensorTrigger,
  TelemetryReading,
  TriggerListResponse,
} from '../types/telemetry'

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

export async function fetchSensorTriggerByDeviceId(
  deviceId: string,
  sensor: string,
): Promise<SensorTrigger | null> {
  const encodedSensor = encodeURIComponent(sensor)

  try {
    const { data } = await httpClient.get<SensorTrigger>(
      `/api/triggers/${deviceId}/${encodedSensor}`,
    )
    return data
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null
    }

    throw error
  }
}

export async function saveSensorTriggerByDeviceId(
  deviceId: string,
  sensor: string,
  payload: { min?: number; max?: number },
): Promise<SensorTrigger> {
  const encodedSensor = encodeURIComponent(sensor)

  const { data } = await httpClient.put<SensorTrigger>(
    `/api/triggers/${deviceId}/${encodedSensor}`,
    payload,
  )
  return data
}

export async function fetchTriggersByDeviceId(
  deviceId: string,
): Promise<TriggerListResponse> {
  const { data } = await httpClient.get<TriggerListResponse>(`/api/triggers/${deviceId}`)
  return data
}

export async function deleteSensorTriggerByDeviceId(
  deviceId: string,
  sensor: string,
): Promise<void> {
  const encodedSensor = encodeURIComponent(sensor)
  await httpClient.delete(`/api/triggers/${deviceId}/${encodedSensor}`)
}
