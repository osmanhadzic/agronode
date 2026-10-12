import { useEffect, useMemo, useState, type FormEvent } from 'react'

import { createAdminChatSocket, type AdminChatMessage } from '../api/adminChatSocket'
import { getLocaleTag, useI18n } from '../i18n'

function formatTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleTimeString(getLocaleTag())
}

export function AdminChatPanel() {
  const { t } = useI18n()
  const [messages, setMessages] = useState<AdminChatMessage[]>([])
  const [draft, setDraft] = useState('')
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const [sendError, setSendError] = useState(false)
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
        return t('chat.connected')
      case 'disconnected':
        return t('chat.disconnected')
      default:
        return t('chat.connecting')
    }
  }, [status, t])

  const handleSend = (event: FormEvent) => {
    event.preventDefault()

    const text = draft.trim()
    if (!text) {
      return
    }

    if (!connection || !connection.sendMessage(text)) {
      setSendError(true)
      return
    }

    setDraft('')
    setSendError(false)
  }

  return (
    <section className="admin-chat-panel dashboard-card" aria-label={t('chat.title')}>
      <div className="admin-chat-header">
        <div>
          <p className="dashboard-card-eyebrow">{t('chat.eyebrow')}</p>
          <h2 className="dashboard-card-title">{t('chat.title')}</h2>
        </div>
        <span className={`admin-chat-status admin-chat-status-${status}`}>{statusLabel}</span>
      </div>

      <div className="admin-chat-messages" role="log" aria-live="polite">
        {messages.length === 0 && <p className="dashboard-message">{t('chat.empty')}</p>}

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
          placeholder={t('chat.placeholder')}
          maxLength={1000}
          disabled={status !== 'connected'}
        />
        <button type="submit" disabled={status !== 'connected' || draft.trim().length === 0}>
          {t('common.send')}
        </button>
      </form>

      {sendError && <p className="dashboard-message">{t('chat.notSent')}</p>}
    </section>
  )
}
