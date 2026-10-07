import { getSessionToken } from './session'

export type AdminChatMessage = {
  id: string
  sender: string
  text: string
  sentAt: string
}

export type AdminChatConnection = {
  sendMessage: (text: string) => boolean
  disconnect: () => void
}

function resolveAdminChatWebSocketUrl(): string {
  const explicitApiBase = import.meta.env.VITE_API_BASE_URL
  const baseUrl = explicitApiBase
    ? explicitApiBase.replace(/^http/, 'ws')
    : `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.hostname}:8080`

  const url = new URL('/ws/admin/chat', baseUrl)
  const token = getSessionToken()

  if (token) {
    url.searchParams.set('token', token)
  }

  return url.toString()
}

export function createAdminChatSocket(
  onMessage: (message: AdminChatMessage) => void,
  onStatusChange?: (status: 'connecting' | 'connected' | 'disconnected') => void,
): AdminChatConnection {
  let socket: WebSocket | null = null
  let reconnectTimer: number | null = null
  let shouldReconnect = true

  const connect = () => {
    onStatusChange?.('connecting')
    socket = new WebSocket(resolveAdminChatWebSocketUrl())

    socket.onopen = () => {
      onStatusChange?.('connected')
    }

    socket.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data) as AdminChatMessage
        onMessage(parsed)
      } catch {
        // ignore malformed messages
      }
    }

    socket.onclose = () => {
      onStatusChange?.('disconnected')

      if (!shouldReconnect) {
        return
      }

      reconnectTimer = window.setTimeout(connect, 3000)
    }

    socket.onerror = () => {
      onStatusChange?.('disconnected')
    }
  }

  connect()

  return {
    sendMessage: (text: string) => {
      if (!socket || socket.readyState !== WebSocket.OPEN) {
        return false
      }

      socket.send(JSON.stringify({ text }))
      return true
    },
    disconnect: () => {
      shouldReconnect = false

      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer)
      }

      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.close()
      }
    },
  }
}
