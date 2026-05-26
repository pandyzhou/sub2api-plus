/**
 * chatgpt2api Account Management API
 *
 * Corresponds to chatgpt2api's /api/accounts endpoints.
 * Manages ChatGPT access tokens used for image generation.
 */

import { chatgptClient } from './client'

// ==================== Types ====================

export type ChatGPTAccountType = string

export type ChatGPTAccountStatus = '正常' | '限流' | '异常' | '禁用'

export type ChatGPTAccount = {
  access_token: string
  type: ChatGPTAccountType
  export_type?: string | null
  status: ChatGPTAccountStatus
  quota: number
  image_quota_unknown?: boolean
  email?: string | null
  expired?: string | null
  id_token?: string | null
  account_id?: string | null
  last_refresh?: string | null
  refresh_token?: string | null
  user_id?: string | null
  limits_progress?: Array<{
    feature_name?: string
    remaining?: number
    reset_after?: string
  }>
  default_model_slug?: string | null
  restore_at?: string | null
  success: number
  fail: number
  last_used_at?: string | null
}

export type ChatGPTAccountListResponse = {
  items: ChatGPTAccount[]
}

export type ChatGPTAccountMutationResponse = {
  items: ChatGPTAccount[]
  added?: number
  skipped?: number
  removed?: number
  refreshed?: number
  errors?: Array<{ access_token: string; error: string }>
}

export type ChatGPTAccountImportPayload = {
  access_token: string
  accessToken?: string
  type?: string
  export_type?: string
  email?: string
  expired?: string
  id_token?: string
  account_id?: string
  last_refresh?: string
  refresh_token?: string
  [key: string]: unknown
}

// ==================== API Methods ====================

/**
 * Fetch all ChatGPT accounts
 */
export async function fetchAccounts(): Promise<ChatGPTAccountListResponse> {
  const { data } = await chatgptClient.get<ChatGPTAccountListResponse>('/api/accounts')
  return data
}

/**
 * Import accounts (tokens + optional structured payloads)
 */
export async function createAccounts(
  tokens: string[],
  accounts: ChatGPTAccountImportPayload[] = [],
): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await chatgptClient.post<ChatGPTAccountMutationResponse>('/api/accounts', {
    tokens,
    ...(accounts.length > 0 ? { accounts } : {}),
  })
  return data
}

/**
 * Delete accounts by access token
 */
export async function deleteAccounts(tokens: string[]): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await chatgptClient.delete<ChatGPTAccountMutationResponse>('/api/accounts', {
    data: { tokens },
  })
  return data
}

/**
 * Refresh accounts (fetch remote quota/status info)
 */
export async function refreshAccounts(
  accessTokens: string[],
): Promise<ChatGPTAccountMutationResponse> {
  const { data } = await chatgptClient.post<ChatGPTAccountMutationResponse>(
    '/api/accounts/refresh',
    { access_tokens: accessTokens },
  )
  return data
}

/**
 * Update a single account's type/status/quota
 */
export async function updateAccount(
  accessToken: string,
  updates: {
    type?: ChatGPTAccountType
    status?: ChatGPTAccountStatus
    quota?: number
  },
): Promise<{ item: ChatGPTAccount; items: ChatGPTAccount[] }> {
  const { data } = await chatgptClient.post<{ item: ChatGPTAccount; items: ChatGPTAccount[] }>(
    '/api/accounts/update',
    {
      access_token: accessToken,
      ...updates,
    },
  )
  return data
}

/**
 * Export accounts (JSON or ZIP)
 */
export async function exportAccounts(
  format: 'json' | 'zip' = 'json',
  accessTokens: string[] = [],
): Promise<Blob> {
  const response = await chatgptClient.post<Blob>(
    '/api/accounts/export',
    { format, access_tokens: accessTokens },
    { responseType: 'blob' },
  )
  return response.data
}
