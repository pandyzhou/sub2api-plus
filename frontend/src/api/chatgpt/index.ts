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
  fetchAccountPoolConfig,
  updateAccountPoolConfig,
  exportAccounts,
} from './accounts'

export type {
  ChatGPTAccount,
  ChatGPTAccountPoolConfig,
  ChatGPTAccountExportFormat,
  ChatGPTAccountExportResponse,
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
  createRegisterEventsToken,
  createRegisterEventSource,
} from './register'

export type {
  RegisterConfig,
  RegisterMailConfig,
  RegisterMailProvider,
  RegisterMailProviderType,
  RegisterMode,
  RegisterUpdatePayload,
} from './register'
