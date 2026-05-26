/**
 * ChatGPT Account Pool API Module
 *
 * Uses sub2api's native Go backend for account management.
 * No external chatgpt2api dependency required.
 */

// Client & connection management
export {
  testConnection,
  applyStoredConnection,
} from './client'

// Account management APIs
export {
  fetchAccounts,
  createAccounts,
  deleteAccounts,
  refreshAccounts,
  updateAccount,
  exportAccounts,
} from './accounts'

export type {
  ChatGPTAccount,
  ChatGPTAccountListResponse,
  ChatGPTAccountMutationResponse,
} from './client'

// Registration machine APIs
export {
  fetchRegisterConfig,
  updateRegisterConfig,
  startRegister,
  stopRegister,
  resetRegister,
  createRegisterEventSource,
} from './register'

export type {
  RegisterConfig,
  RegisterMode,
  RegisterUpdatePayload,
} from './register'
