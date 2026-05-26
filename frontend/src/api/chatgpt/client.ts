/**
 * chatgpt2api HTTP Client
 *
 * Separate axios instance for chatgpt2api's Python backend.
 * Uses a different base URL and auth mechanism than sub2api's own API.
 */

import axios, { type AxiosInstance, type AxiosError } from 'axios'

// Default base URL — can be overridden via the connection store
let _baseURL = 'http://127.0.0.1:20002'
let _authKey = ''

export function getChatGPTBaseURL(): string {
  return _baseURL
}

export function setChatGPTBaseURL(url: string): void {
  _baseURL = url.replace(/\/$/, '')
  chatgptClient.defaults.baseURL = _baseURL
}

export function getChatGPTAuthKey(): string {
  return _authKey
}

export function setChatGPTAuthKey(key: string): void {
  _authKey = key
}

// ==================== Axios Instance ====================

export const chatgptClient: AxiosInstance = axios.create({
  baseURL: _baseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: inject auth key
chatgptClient.interceptors.request.use((config) => {
  if (_authKey && config.headers) {
    config.headers.Authorization = `Bearer ${_authKey}`
  }
  return config
})

// Response interceptor: unwrap chatgpt2api error format
chatgptClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ detail?: { error?: string } | string; error?: string; message?: string }>) => {
    if (error.response) {
      const payload = error.response.data
      let message = '请求失败'

      if (payload) {
        if (typeof payload.detail === 'object' && payload.detail !== null) {
          message = (payload.detail as { error?: string }).error || message
        } else if (typeof payload.detail === 'string') {
          message = payload.detail
        } else if (typeof payload.error === 'string') {
          message = payload.error
        } else if (typeof payload.message === 'string') {
          message = payload.message
        }
      }

      if (error.response.status === 401) {
        message = 'chatgpt2api 认证失败，请检查连接配置中的 Admin Key'
      }

      return Promise.reject(new Error(message))
    }

    return Promise.reject(new Error(error.message || '网络错误'))
  },
)

// ==================== Connection Helpers ====================

const CONNECTION_STORAGE_KEY = 'chatgpt2api_connection'

export interface ChatGPTConnectionConfig {
  baseURL: string
  authKey: string
}

export function loadConnectionConfig(): ChatGPTConnectionConfig | null {
  try {
    const raw = localStorage.getItem(CONNECTION_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.baseURL === 'string' && typeof parsed.authKey === 'string') {
      return { baseURL: parsed.baseURL, authKey: parsed.authKey }
    }
  } catch {
    // ignore
  }
  return null
}

export function saveConnectionConfig(config: ChatGPTConnectionConfig): void {
  localStorage.setItem(CONNECTION_STORAGE_KEY, JSON.stringify(config))
  setChatGPTBaseURL(config.baseURL)
  setChatGPTAuthKey(config.authKey)
}

export function clearConnectionConfig(): void {
  localStorage.removeItem(CONNECTION_STORAGE_KEY)
  setChatGPTBaseURL('http://127.0.0.1:20002')
  setChatGPTAuthKey('')
}

export function applyStoredConnection(): boolean {
  const config = loadConnectionConfig()
  if (config) {
    setChatGPTBaseURL(config.baseURL)
    setChatGPTAuthKey(config.authKey)
    return true
  }
  return false
}

export async function testConnection(): Promise<{ ok: boolean; version?: string; error?: string }> {
  try {
    const { data } = await chatgptClient.get<{ version: string }>('/version')
    return { ok: true, version: data.version }
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : '连接失败' }
  }
}
