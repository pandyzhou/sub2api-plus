/**
 * ChatGPT Registration Machine API
 *
 * Uses sub2api's native Go backend for register machine management.
 */

import { apiClient } from '../client'

const BASE = '/admin/chatgpt'

// ==================== Types ====================

export type RegisterMode = 'total' | 'quota' | 'available'

export type RegisterMailProviderType =
  | 'mailtm'
  | 'custom'
  | 'cloudflare_temp_email'
  | 'tempmail_lol'
  | 'inbucket'
  | 'moemail'
  | 'cloudmail_gen'
  | 'ddg_mail'
  | 'duckmail'
  | 'gptmail'
  | 'yyds_mail'

export type RegisterMailProvider = {
  type: RegisterMailProviderType | string
  enable: boolean
  provider_ref?: string
  label?: string
  api_base?: string
  api_key?: string
  admin_email?: string
  admin_password?: string
  ddg_token?: string
  cf_inbox_jwt?: string
  cf_api_base?: string
  cf_api_key?: string
  cf_auth_mode?: string
  cf_create_path?: string
  cf_messages_path?: string
  cf_domain?: string[]
  domain?: string[]
  subdomain?: string[]
  default_domain?: string
  wildcard?: boolean
  random_subdomain?: boolean
  email_prefix?: string
  expiry_time?: number
}

export type RegisterMailConfig = {
  request_timeout: number
  wait_timeout: number
  wait_interval: number
  providers: RegisterMailProvider[]
}

export type RegisterUpdatePayload = Partial<{
  proxy: string
  total: number
  threads: number
  mode: RegisterMode
  target_quota: number
  target_available: number
  check_interval: number
  mail: RegisterMailConfig
  // Legacy fields for backward compatibility with older backend responses.
  mail_provider: string
  mail_api_base: string
  mail_api_key: string
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
  mail?: RegisterMailConfig
  mail_provider?: string
  mail_api_base?: string
  mail_api_key?: string
  stats: {
    success: number
    fail: number
    done: number
    running: number
    threads?: number
    elapsed_seconds?: number
    avg_seconds?: number
    success_rate?: number
    current_quota?: number
    current_available?: number
  }
  logs?: Array<{
    time: string
    text: string
    level: string
  }>
}

export type RegisterEventTokenResponse = {
  token: string
  expires_at?: string
  ttl_seconds?: number
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
 * Create a short-lived token for EventSource registration status stream.
 */
export async function createRegisterEventsToken(): Promise<RegisterEventTokenResponse> {
  const { data } = await apiClient.post(`${BASE}/register/events-token`)
  return data?.data || data
}

/**
 * Create an EventSource for real-time registration machine events.
 */
export function createRegisterEventSource(token: string): EventSource {
  const url = `/api/v1${BASE}/register/events?token=${encodeURIComponent(token)}`
  return new EventSource(url)
}
