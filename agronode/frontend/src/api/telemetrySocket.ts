import type { DeviceStatusEvent, TelemetryReading } from '../types/telemetry'

import { getSessionToken } from './session'

function resolveWebSocketUrl(path: string, organizationId?: number): string {
  const explicitApiBase = import.meta.env.VITE_API_BASE_URL
  const baseUrl = explicitApiBase
    ? explicitApiBase.replace(/^http/, 'ws')
    : `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.hostname}:8080`

  const url = new URL(path, baseUrl)
  const token = getSessionToken()

  if (token) {
    url.searchParams.set('token', token)
  }

  if (organizationId && organizationId > 0) {
    url.searchParams.set('organizationId', String(organizationId))
  }

  return url.toString()
}

function createSocket<T>(
  path: string,
  organizationId: number | undefined,
  onMessage: (message: T) => void,
  onError?: () => void,
): () => void {
  let socket: WebSocket | null = null
  let reconnectTimer: number | null = null
  let isClosed = false

  const connect = () => {
    if (isClosed) {
      return
    }

    socket = new WebSocket(resolveWebSocketUrl(path, organizationId))

    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as T
        onMessage(message)
      } catch {
        // Ignore parsing errors
      }
    }

    socket.onerror = () => {
      onError?.()
    }

    socket.onclose = () => {
      if (isClosed) {
        return
      }

      reconnectTimer = window.setTimeout(connect, 3000)
    }
  }

  connect()

  return () => {
    isClosed = true

    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer)
    }

    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.close()
    }
  }
}

export function createTelemetrySocket(
  organizationId: number | undefined,
  onMessage: (reading: TelemetryReading) => void,
  onError?: () => void,
): () => void {
  return createSocket('/ws/telemetry', organizationId, onMessage, onError)
}

export function createDeviceStatusSocket(
  organizationId: number | undefined,
  onMessage: (event: DeviceStatusEvent) => void,
  onError?: () => void,
): () => void {
  return createSocket('/ws/devices', organizationId, onMessage, onError)
}
