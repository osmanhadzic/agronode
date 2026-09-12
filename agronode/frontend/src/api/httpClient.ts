import axios, { AxiosHeaders } from 'axios'

import { clearSession, getSessionToken } from './session'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export const httpClient = axios.create({
  baseURL: apiBaseUrl,
  timeout: 10_000,
})

httpClient.interceptors.request.use((config) => {
  const headers = AxiosHeaders.from(config.headers ?? {})

  const token = getSessionToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  config.headers = headers

  return config
})

httpClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      clearSession()
    }

    return Promise.reject(error)
  },
)
