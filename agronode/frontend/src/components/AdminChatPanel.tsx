import { useEffect, useMemo, useState, type FormEvent } from 'react'

import { createAdminChatSocket, type AdminChatMessage } from '../api/adminChatSocket'

function formatTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleTimeString()
}

export function AdminChatPanel() {
  const [messages, setMessages] = useState<AdminChatMessage[]>([])
  const [draft, setDraft] = useState('')
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const [sendError, setSendError] = useState('')
  const [connection, setConnection] = useState<ReturnType<typeof createAdminChatSocket> | null>(null)

  useEffect(() => {
    const socketConnection = createAdminChatSocket(
      (message) => {
        setMessages((previous) => [...previous.slice(-149), message])
      },
      (nextStatus) => {
        setStatus(nextStatus)
      },
    )

    setConnection(socketConnection)

    return () => {
      socketConnection.disconnect()
    }
  }, [])

  const statusLabel = useMemo(() => {
    switch (status) {
      case 'connected':
        return 'Connected'
      case 'disconnected':
        return 'Disconnected'
      default:
        return 'Connecting...'
    }
  }, [status])

  const handleSend = (event: FormEvent) => {
    event.preventDefault()

    const text = draft.trim()
    if (!text) {
      return
    }

    if (!connection || !connection.sendMessage(text)) {
      setSendError('Message was not sent. Connection is not ready.')
      return
    }

    setDraft('')
    setSendError('')
  }

  return (
    <section className="admin-chat-panel dashboard-card" aria-label="Admin chat">
      <div className="admin-chat-header">
        <div>
          <p className="dashboard-card-eyebrow">Admin</p>
          <h2 className="dashboard-card-title">Chat</h2>
        </div>
        <span className={`admin-chat-status admin-chat-status-${status}`}>{statusLabel}</span>
      </div>

      <div className="admin-chat-messages" role="log" aria-live="polite">
        {messages.length === 0 && <p className="dashboard-message">No messages yet.</p>}

        {messages.map((message) => (
          <article key={message.id} className="admin-chat-message">
            <header>
              <strong>{message.sender}</strong>
              <time>{formatTime(message.sentAt)}</time>
            </header>
            <p>{message.text}</p>
          </article>
        ))}
      </div>

      <form className="admin-chat-form" onSubmit={handleSend}>
        <input
          type="text"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder="Type a message"
          maxLength={1000}
          disabled={status !== 'connected'}
        />
        <button type="submit" disabled={status !== 'connected' || draft.trim().length === 0}>
          Send
        </button>
      </form>

      {sendError && <p className="dashboard-message">{sendError}</p>}
    </section>
  )
}
