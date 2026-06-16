import axios from 'axios'

import { httpClient } from './httpClient'
import type { TelemetryReading } from '../types/telemetry'

export type DateFilterPeriod = 'hour' | 'day' | 'week' | 'month' | 'year' | 'custom' | ''

export interface DateFilterOptions {
  period?: DateFilterPeriod
  startDate?: string
  endDate?: string
}

export async function fetchAllTelemetry(): Promise<TelemetryReading[]> {
  const { data } = await httpClient.get<TelemetryReading[]>('/api/data')
  return data
}

export async function fetchTelemetryByDeviceId(
  deviceId: string,
  dateFilter?: DateFilterOptions,
): Promise<TelemetryReading[]> {
  const params = new URLSearchParams()
  
  if (dateFilter?.period) {
    params.append('period', dateFilter.period)
  }
  
  if (dateFilter?.startDate) {
    params.append('startDate', dateFilter.startDate)
  }
  
  if (dateFilter?.endDate) {
    params.append('endDate', dateFilter.endDate)
  }
  
  const queryString = params.toString()
  const url = `/api/data/${deviceId}${queryString ? '?' + queryString : ''}`
  
  const { data } = await httpClient.get<TelemetryReading[]>(url)
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
