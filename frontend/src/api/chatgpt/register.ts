/**
 * chatgpt2api Registration Machine API
 *
 * Corresponds to chatgpt2api's /api/register endpoints.
 * Controls the automated ChatGPT account registration process.
 */

import { chatgptClient } from './client'

// ==================== Types ====================

export type RegisterMode = 'total' | 'quota' | 'available'

export type RegisterMailProvider = Record<string, unknown>

export type RegisterConfig = {
  enabled: boolean
  mail: {
    request_timeout: number
    wait_timeout: number
    wait_interval: number
    providers: RegisterMailProvider[]
  }
  proxy: string
  total: number
  threads: number
  mode: RegisterMode
  target_quota: number
  target_available: number
  check_interval: number
  stats: {
    job_id?: string
    success: number
    fail: number
    done: number
    running: number
    threads: number
    elapsed_seconds?: number
    avg_seconds?: number
    success_rate?: number
    current_quota?: number
    current_available?: number
    started_at?: string
    updated_at?: string
    finished_at?: string
  }
  logs?: Array<{
    time: string
    text: string
    level: string
  }>
}

export type RegisterUpdatePayload = Partial<{
  mail: RegisterConfig['mail']
  proxy: string
  total: number
  threads: number
  mode: RegisterMode
  target_quota: number
  target_available: number
  check_interval: number
}>

// ==================== API Methods ====================

/**
 * Fetch current registration machine config and status
 */
export async function fetchRegisterConfig(): Promise<{ register: RegisterConfig }> {
  const { data } = await chatgptClient.get<{ register: RegisterConfig }>('/api/register')
  return data
}

/**
 * Update registration machine configuration
 */
export async function updateRegisterConfig(
  updates: RegisterUpdatePayload,
): Promise<{ register: RegisterConfig }> {
  const { data } = await chatgptClient.post<{ register: RegisterConfig }>(
    '/api/register',
    updates,
  )
  return data
}

/**
 * Start the registration machine
 */
export async function startRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await chatgptClient.post<{ register: RegisterConfig }>(
    '/api/register/start',
  )
  return data
}

/**
 * Stop the registration machine
 */
export async function stopRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await chatgptClient.post<{ register: RegisterConfig }>(
    '/api/register/stop',
  )
  return data
}

/**
 * Reset registration statistics
 */
export async function resetRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await chatgptClient.post<{ register: RegisterConfig }>(
    '/api/register/reset',
  )
  return data
}

/**
 * Create an EventSource for real-time registration machine events
 * @param token - Auth token for the SSE stream
 * @param baseURL - chatgpt2api base URL
 * @returns EventSource instance
 */
export function createRegisterEventSource(token: string, baseURL: string): EventSource {
  const url = `${baseURL.replace(/\/$/, '')}/api/register/events?token=${encodeURIComponent(token)}`
  return new EventSource(url)
}
