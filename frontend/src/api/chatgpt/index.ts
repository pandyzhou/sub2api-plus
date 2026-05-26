/**
 * chatgpt2api API Module
 *
 * Provides HTTP clients and API methods for communicating with
 * the chatgpt2api Python backend (ChatGPT account pool management
 * and automated account registration machine).
 */

// Client & connection management
export {
  chatgptClient,
  getChatGPTBaseURL,
  setChatGPTBaseURL,
  getChatGPTAuthKey,
  setChatGPTAuthKey,
  loadConnectionConfig,
  saveConnectionConfig,
  clearConnectionConfig,
  applyStoredConnection,
  testConnection,
} from './client'

export type { ChatGPTConnectionConfig } from './client'

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
  ChatGPTAccountType,
  ChatGPTAccountStatus,
  ChatGPTAccountListResponse,
  ChatGPTAccountMutationResponse,
  ChatGPTAccountImportPayload,
} from './accounts'

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
  RegisterMailProvider,
} from './register'
