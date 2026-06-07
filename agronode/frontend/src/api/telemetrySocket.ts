import type { DeviceStatusEvent, TelemetryReading } from '../types/telemetry'

function resolveWebSocketUrl(path: string): string {
  const explicitApiBase = import.meta.env.VITE_API_BASE_URL

  if (explicitApiBase) {
    const normalized = explicitApiBase.replace(/^http/, 'ws')
    return `${normalized}${path}`
  }

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.hostname}:8080${path}`
}

function createSocket<T>(
  path: string,
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

    socket = new WebSocket(resolveWebSocketUrl(path))

    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as T
        onMessage(message)
      } catch {
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
  onMessage: (reading: TelemetryReading) => void,
  onError?: () => void,
): () => void {
  return createSocket('/ws/telemetry', onMessage, onError)
}

export function createDeviceStatusSocket(
  onMessage: (event: DeviceStatusEvent) => void,
  onError?: () => void,
): () => void {
  return createSocket('/ws/devices', onMessage, onError)
}
