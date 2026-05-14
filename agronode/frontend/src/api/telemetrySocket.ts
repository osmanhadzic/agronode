import type { TelemetryReading } from '../types/telemetry'

function resolveWebSocketUrl(): string {
  const explicitApiBase = import.meta.env.VITE_API_BASE_URL

  if (explicitApiBase) {
    const normalized = explicitApiBase.replace(/^http/, 'ws')
    return `${normalized}/ws/telemetry`
  }

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.hostname}:8080/ws/telemetry`
}

export function createTelemetrySocket(
  onMessage: (reading: TelemetryReading) => void,
  onError?: () => void,
): () => void {
  let socket: WebSocket | null = null
  let reconnectTimer: number | null = null
  let isClosed = false

  const connect = () => {
    if (isClosed) {
      return
    }

    socket = new WebSocket(resolveWebSocketUrl())

    socket.onmessage = (event) => {
      try {
        const reading = JSON.parse(event.data) as TelemetryReading
        onMessage(reading)
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
