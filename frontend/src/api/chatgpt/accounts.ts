/**
 * ChatGPT Account Pool Management API
 *
 * Re-exports from the native client.
 */

export {
  fetchAccounts,
  createAccounts,
  deleteAccounts,
  refreshAccounts,
  updateAccount,
  exportAccounts,
} from './client'

export type {
  ChatGPTAccount,
  ChatGPTAccountListResponse,
  ChatGPTAccountMutationResponse,
} from './client'

// Legacy type aliases for backward compatibility
export type ChatGPTAccountType = string
export type ChatGPTAccountStatus = string
export type ChatGPTAccountImportPayload = Record<string, unknown>
