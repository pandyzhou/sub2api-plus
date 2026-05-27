/**
 * ChatGPT Account Pool API Client
 *
 * Uses sub2api's native Go backend for ChatGPT Web account management.
 * No external chatgpt2api dependency required.
 */

import { apiClient } from '../client'

const BASE = '/admin/chatgpt'

export type ChatGPTLimitProgress = Record<string, unknown>

export type ChatGPTAccount = {
  access_token: string
  refresh_token?: string
  id_token?: string
  password?: string
  export_type?: string
  type: string
  account_type?: string
  status: string
  name?: string
  email?: string
  user_id?: string
  plan_type?: string
  chatgpt_account_id?: string
  account_id?: string
  created_at?: string
  updated_at?: string
  last_used_at?: string | null
  quota?: number
  image_quota_unknown?: boolean
  limits_progress?: ChatGPTLimitProgress[] | ChatGPTLimitProgress | unknown[]
  default_model_slug?: string | null
  restore_at?: string | null
  success?: number
  fail?: number
  invalid_count?: number
  last_invalid_at?: string | null
  last_refresh_error?: string | null
  last_refresh_error_at?: string | null
  last_token_refresh_at?: string | null
  last_token_refresh_error?: string | null
  last_token_refresh_error_at?: string | null
  expired?: string | boolean | null
  last_refresh?: string | null
}

export type ChatGPTAccountPoolConfig = {
  refresh_account_interval_minute: number
  auto_remove_invalid_accounts: boolean
  auto_remove_rate_limited_accounts: boolean
  image_account_concurrency: number
}

export type ChatGPTAccountListResponse = {
  items: ChatGPTAccount[]
}

export type ChatGPTAccountMutationResponse = {
  items?: ChatGPTAccount[]
  added?: number
  skipped?: number
  removed?: number
  refreshed?: number
  errors?: Array<{ token: string; error: string }>
}

export type ChatGPTAccountExportFormat = 'json' | 'zip'

export type ChatGPTAccountExportResponse = {
  blob: Blob
  filename: string
}

/**
 * Fetch all ChatGPT Web accounts from sub2api native API
 */
export async function fetchAccounts(): Promise<ChatGPTAccountListResponse> {
  const { data } = await apiClient.get(`${BASE}/accounts`)
  return data?.data || data
}

/**
 * Import accounts (tokens or structured payloads)
 */
export async function createAccounts(
  tokens: string[],
  accounts: Record<string, unknown>[] = [],
): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await apiClient.post(`${BASE}/accounts`, {
    tokens,
    ...(accounts.length > 0 ? { accounts } : {}),
  })
  return data?.data || data
}

/**
 * Delete accounts by access token
 */
export async function deleteAccounts(tokens: string[]): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await apiClient.delete(`${BASE}/accounts`, {
    data: { tokens },
  })
  return data?.data || data
}

/**
 * Refresh accounts (fetch latest user info)
 */
export async function refreshAccounts(tokens: string[]): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await apiClient.post(`${BASE}/accounts/refresh`, {
    access_tokens: tokens,
  })
  return data?.data || data
}

/**
 * Update account (status/type/quota/image_quota_unknown)
 */
export async function updateAccount(
  accessToken: string,
  updates: Record<string, unknown>,
): Promise<ChatGPTAccountMutationResponse | void> {
  const { data } = await apiClient.post(`${BASE}/accounts/update`, {
    access_token: accessToken,
    ...updates,
  })
  return data?.data || data
}

/**
 * Fetch account pool configuration.
 */
export async function fetchAccountPoolConfig(): Promise<ChatGPTAccountPoolConfig> {
  const { data } = await apiClient.get(`${BASE}/account-pool/config`)
  return data?.data || data
}

/**
 * Update account pool configuration.
 */
export async function updateAccountPoolConfig(
  updates: Partial<ChatGPTAccountPoolConfig>,
): Promise<ChatGPTAccountPoolConfig> {
  const { data } = await apiClient.post(`${BASE}/account-pool/config`, updates)
  return data?.data || data
}

function filenameFromDisposition(disposition?: string): string | null {
  if (!disposition) return null
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8?.[1]) return decodeURIComponent(utf8[1].replace(/"/g, ''))
  const plain = disposition.match(/filename="?([^";]+)"?/i)
  return plain?.[1] || null
}

/**
 * Export accounts as backend-generated JSON or ZIP blob.
 */
export async function exportAccounts(
  tokens: string[],
  format: ChatGPTAccountExportFormat = 'json',
): Promise<ChatGPTAccountExportResponse> {
  const response = await apiClient.post(
    `${BASE}/accounts/export`,
    { access_tokens: tokens, format },
    { responseType: 'blob' },
  )
  const fallback = `chatgpt-accounts.${format}`
  return {
    blob: response.data,
    filename: filenameFromDisposition(response.headers?.['content-disposition']) || fallback,
  }
}

/**
 * Test connection — always returns OK since we use sub2api's own API
 */
export async function testConnection(): Promise<{ ok: boolean; version?: string; error?: string }> {
  try {
    await apiClient.get(`${BASE}/accounts`)
    return { ok: true, version: 'native' }
  } catch (e: unknown) {
    return { ok: false, error: e instanceof Error ? e.message : '连接失败' }
  }
}

/**
 * Apply stored connection — no-op since we use sub2api's own auth
 */
export function applyStoredConnection(): void {
  // No external connection to apply — sub2api uses its own JWT auth
}
