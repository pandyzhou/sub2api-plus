/**
 * ChatGPT Account Pool API Client
 *
 * Uses sub2api's native Go backend for ChatGPT Web account management.
 * No external chatgpt2api dependency required.
 */

import { apiClient } from '../client'

const BASE = '/admin/chatgpt'

export type ChatGPTAccount = {
  access_token: string
  type: string
  status: string
  name?: string
  email?: string
  user_id?: string
  plan_type?: string
  chatgpt_account_id?: string
  created_at?: string
  // Legacy fields for backward compatibility
  quota?: number
  image_quota_unknown?: boolean
  account_id?: string
  success?: number
  fail?: number
  last_used_at?: string | null
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
 * Update account (status etc.)
 */
export async function updateAccount(
  accessToken: string,
  updates: Record<string, unknown>,
): Promise<void> {
  await apiClient.post(`${BASE}/accounts/update`, {
    access_token: accessToken,
    ...updates,
  })
}

/**
 * Export accounts
 */
export async function exportAccounts(tokens: string[]): Promise<ChatGPTAccountListResponse> {
  const { data } = await apiClient.post(`${BASE}/accounts/export`, {
    access_tokens: tokens,
  })
  return data?.data || data
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
