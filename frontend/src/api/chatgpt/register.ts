/**
 * ChatGPT Registration Machine API
 *
 * Uses sub2api's native Go backend for register machine management.
 */

import { apiClient } from '../client'

const BASE = '/api/v1/admin/chatgpt'

// ==================== Types ====================

export type RegisterMode = 'total' | 'quota' | 'available'

export type RegisterUpdatePayload = Partial<{
  proxy: string
  total: number
  threads: number
  mode: RegisterMode
  target_quota: number
  target_available: number
  check_interval: number
}>

export type RegisterConfig = {
  enabled: boolean
  mode: RegisterMode
  total: number
  threads: number
  proxy: string
  target_quota: number
  target_available: number
  check_interval: number
  stats: {
    success: number
    fail: number
    done: number
    running: number
    threads?: number
  }
  logs?: Array<{
    time: string
    text: string
    level: string
  }>
}

// ==================== API Methods ====================

/**
 * Fetch current registration machine config and status
 */
export async function fetchRegisterConfig(): Promise<{ register: RegisterConfig }> {
  const { data } = await apiClient.get(`${BASE}/register`)
  return data?.data || data
}

/**
 * Update registration machine configuration
 */
export async function updateRegisterConfig(
  updates: RegisterUpdatePayload,
): Promise<{ register: RegisterConfig }> {
  const { data } = await apiClient.post(`${BASE}/register`, updates)
  return data?.data || data
}

/**
 * Start the registration machine
 */
export async function startRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await apiClient.post(`${BASE}/register/start`)
  return data?.data || data
}

/**
 * Stop the registration machine
 */
export async function stopRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await apiClient.post(`${BASE}/register/stop`)
  return data?.data || data
}

/**
 * Reset registration statistics
 */
export async function resetRegister(): Promise<{ register: RegisterConfig }> {
  const { data } = await apiClient.post(`${BASE}/register/reset`)
  return data?.data || data
}

/**
 * Create an EventSource for real-time registration machine events
 * @returns EventSource instance
 */
export function createRegisterEventSource(): EventSource {
  // Use sub2api's own SSE endpoint for register events
  const url = `/api/v1/admin/chatgpt/register/events`
  return new EventSource(url)
}
