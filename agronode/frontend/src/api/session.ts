export type UserRole = 'admin' | 'organization'

export interface SessionInfo {
  token: string
  email: string
  organizationId: number
  role: UserRole
  expiresAt: number
}

const SESSION_STORAGE_KEY = 'agronode.session'
const ORGANIZATION_SCOPE_STORAGE_KEY = 'agronode.organization-scope'
const SESSION_CHANGE_EVENT = 'agronode-session-changed'

function normalizeRole(role: unknown): UserRole {
  return role === 'admin' ? 'admin' : 'organization'
}

function normalizeSession(rawSession: unknown): SessionInfo | null {
  if (!rawSession || typeof rawSession !== 'object') {
    return null
  }

  const candidate = rawSession as Partial<SessionInfo>
  if (
    typeof candidate.token !== 'string' ||
    typeof candidate.email !== 'string' ||
    typeof candidate.organizationId !== 'number' ||
    typeof candidate.expiresAt !== 'number'
  ) {
    return null
  }

  return {
    token: candidate.token,
    email: candidate.email,
    organizationId: candidate.organizationId,
    expiresAt: candidate.expiresAt,
    role: normalizeRole(candidate.role),
  }
}

export function loadSession(): SessionInfo | null {
  const raw = window.localStorage.getItem(SESSION_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    return normalizeSession(JSON.parse(raw))
  } catch {
    return null
  }
}

export function saveSession(session: SessionInfo): void {
  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session))

  if (session.role === 'organization') {
    saveOrganizationScope(session.organizationId)
  } else {
    window.localStorage.removeItem(ORGANIZATION_SCOPE_STORAGE_KEY)
  }

  window.dispatchEvent(new Event(SESSION_CHANGE_EVENT))
}

export function clearSession(): void {
  window.localStorage.removeItem(SESSION_STORAGE_KEY)
  window.localStorage.removeItem(ORGANIZATION_SCOPE_STORAGE_KEY)
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

export function loadOrganizationScope(): number | null {
  const raw = window.localStorage.getItem(ORGANIZATION_SCOPE_STORAGE_KEY)
  if (!raw) {
    return null
  }

  const parsed = Number(raw)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return null
  }

  return parsed
}

export function saveOrganizationScope(organizationId: number | null): void {
  if (organizationId === null) {
    window.localStorage.removeItem(ORGANIZATION_SCOPE_STORAGE_KEY)
  } else {
    window.localStorage.setItem(ORGANIZATION_SCOPE_STORAGE_KEY, String(organizationId))
  }

  window.dispatchEvent(new Event(SESSION_CHANGE_EVENT))
}
