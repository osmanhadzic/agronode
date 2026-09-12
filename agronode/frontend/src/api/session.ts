export interface SessionInfo {
  token: string
  email: string
  organizationId: number
  expiresAt: number
}

const SESSION_STORAGE_KEY = 'agronode.session'
const SESSION_CHANGE_EVENT = 'agronode-session-changed'

export function loadSession(): SessionInfo | null {
  const raw = window.localStorage.getItem(SESSION_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as SessionInfo
  } catch {
    return null
  }
}

export function saveSession(session: SessionInfo): void {
  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session))
  window.dispatchEvent(new Event(SESSION_CHANGE_EVENT))
}

export function clearSession(): void {
  window.localStorage.removeItem(SESSION_STORAGE_KEY)
  window.dispatchEvent(new Event(SESSION_CHANGE_EVENT))
}

export function getSessionToken(): string {
  return loadSession()?.token ?? ''
}

export function onSessionChange(listener: () => void): () => void {
  const storageHandler = () => listener()
  window.addEventListener(SESSION_CHANGE_EVENT, storageHandler)
  return () => window.removeEventListener(SESSION_CHANGE_EVENT, storageHandler)
}
