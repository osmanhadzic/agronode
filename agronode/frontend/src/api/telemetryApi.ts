import axios from 'axios'

import { httpClient } from './httpClient'

export type DateFilterPeriod = 'hour' | 'day' | 'week' | 'month' | 'year' | 'custom' | ''

export interface DateFilterOptions {
  period?: DateFilterPeriod
  startDate?: string
  endDate?: string
}

export type StreamControlAction = 'pause' | 'resume'

export interface StreamControlResponse {
  deviceId: string
  sensorId: string
  action: StreamControlAction
  streaming: boolean
}
import type {
  DeviceSensor,
  SensorTrigger,
  TelemetryReading,
  TriggerListResponse,
  DeviceSummary
} from '../types/telemetry'

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

export async function fetchTelemetryByDeviceIdAndSensorId(
  deviceId: string,
  sensorId: string,
): Promise<TelemetryReading[]> {
  const encodedSensor = encodeURIComponent(sensorId)
  const { data } = await httpClient.get<TelemetryReading[]>(
    `/api/data/${deviceId}/${encodedSensor}`,
  )
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

export async function fetchSensorsByDeviceId(
  deviceId: string,
): Promise<DeviceSensor[]> {
  const { data } = await httpClient.get<DeviceSensor[]>(`/api/devices/${deviceId}/sensors`)
  return data
}

export async function fetchSensorTriggerByDeviceId(
  deviceId: string,
  sensorId: string,
): Promise<SensorTrigger | null> {
  const encodedSensor = encodeURIComponent(sensorId)

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
  sensorId: string,
  payload: { min?: number; max?: number; targetDeviceId?: string; fuzzyConfig?: any },
): Promise<SensorTrigger> {
  const encodedSensor = encodeURIComponent(sensorId)

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
  sensorId: string,
): Promise<void> {
  const encodedSensor = encodeURIComponent(sensorId)
  await httpClient.delete(`/api/triggers/${deviceId}/${encodedSensor}`)
}

export async function sendDeviceStreamControl(
  deviceId: string,
  action: StreamControlAction,
  sensorId?: string,
): Promise<StreamControlResponse> {
  const { data } = await httpClient.post<StreamControlResponse>(
    `/api/devices/${deviceId}/stream-control`,
    { action, sensorId },
  )

  return data
}

// Fuzzy trigger types for UI evaluate endpoint
export interface MembershipFunction {
  name: string
  sensor?: string
  type: 'triangle' | 'trapezoid'
  parameters: number[]
}

export interface Condition {
  sensor: string
  membership: string
}

export interface Action {
  type: string
  value: number
}

export interface Rule {
  name: string
  conditions: Condition[]
  operator?: 'AND' | 'OR'
  action: Action
}

export interface FuzzyTrigger {
  name?: string
  membershipFunctions: MembershipFunction[]
  rules: Rule[]
}

export interface RuleResult {
  name: string
  strength: number
  action: Action
  value: number
}

export interface EvaluationResult {
  input: Record<string, number>
  memberships: Record<string, Record<string, number>>
  rules: RuleResult[]
}

export async function evaluateFuzzyTrigger(
  deviceId: string,
  sensorId: string,
  payload: { inputs?: Record<string, number>; trigger: FuzzyTrigger },
): Promise<EvaluationResult> {
  const encodedSensor = encodeURIComponent(sensorId)
  const { data } = await httpClient.post<EvaluationResult>(
    `/api/triggers/${deviceId}/${encodedSensor}/evaluate`,
    payload,
  )
  return data
}

export async function evaluateSavedTrigger(
  deviceId: string,
  sensorId: string,
  inputs?: Record<string, number>,
): Promise<EvaluationResult> {
  const encodedSensor = encodeURIComponent(sensorId)
  const { data } = await httpClient.post<EvaluationResult>(
    `/api/triggers/${deviceId}/${encodedSensor}/evaluate-saved`,
    { inputs: inputs ?? {} },
  )
  return data
}
