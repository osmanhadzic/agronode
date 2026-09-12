import { httpClient } from './httpClient'
import type { SessionInfo } from './session'

export interface LoginRequest {
  email: string
  password: string
}

export async function login(request: LoginRequest): Promise<SessionInfo> {
  const { data } = await httpClient.post<SessionInfo>('/api/auth/login', request)
  return data
}
