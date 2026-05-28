import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatGPTAccountsStore } from '@/stores/chatgpt/accounts'
import { fetchAccounts, refreshAccounts } from '@/api/chatgpt'

const appStoreMock = {
  showSuccess: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showInfo: vi.fn(),
}

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

vi.mock('@/api/chatgpt', () => ({
  fetchAccounts: vi.fn(),
  createAccounts: vi.fn(),
  deleteAccounts: vi.fn(),
  refreshAccounts: vi.fn(),
  updateAccount: vi.fn(),
  fetchAccountPoolConfig: vi.fn(),
  updateAccountPoolConfig: vi.fn(),
  exportAccounts: vi.fn(),
}))

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useChatGPTAccountsStore refresh UX', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(fetchAccounts).mockResolvedValue({ items: [] })
  })

  it('refreshAll immediately exposes refreshing state and reports summary', async () => {
    const refresh = deferred<{ refreshed: number; errors: Array<{ token: string; error: string }> }>()
    vi.mocked(refreshAccounts).mockReturnValue(refresh.promise as ReturnType<typeof refreshAccounts>)
    const store = useChatGPTAccountsStore()

    const task = store.refreshAll()

    expect(store.refreshing).toBe(true)
    expect(store.refreshingScope).toBe('all')
    expect(store.refreshingTokens.size).toBe(0)
    expect(store.refreshMessage).toContain('正在刷新全部账号')

    refresh.resolve({ refreshed: 8, errors: [] })
    await task

    expect(refreshAccounts).toHaveBeenCalledWith([])
    expect(store.refreshing).toBe(false)
    expect(store.refreshingScope).toBeNull()
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('刷新成功 8 个账号')
  })

  it('refreshOne only marks the clicked account as refreshing', async () => {
    const refresh = deferred<{ refreshed: number; errors: Array<{ token: string; error: string }> }>()
    vi.mocked(refreshAccounts).mockReturnValue(refresh.promise as ReturnType<typeof refreshAccounts>)
    const store = useChatGPTAccountsStore()

    const task = store.refreshOne('token-1')

    expect(store.refreshing).toBe(true)
    expect(store.refreshingScope).toBe('single')
    expect(store.isTokenRefreshing('token-1')).toBe(true)
    expect(store.isTokenRefreshing('token-2')).toBe(false)

    refresh.resolve({ refreshed: 1, errors: [] })
    await task

    expect(refreshAccounts).toHaveBeenCalledWith(['token-1'])
    expect(store.isTokenRefreshing('token-1')).toBe(false)
  })

  it('reports partial refresh failures with the first error', async () => {
    vi.mocked(refreshAccounts).mockResolvedValue({ refreshed: 6, errors: [{ token: 'bad-token', error: 'HTTP 403' }] })
    const store = useChatGPTAccountsStore()

    await store.refreshAll()

    expect(appStoreMock.showError).toHaveBeenCalledWith('刷新成功 6 个，失败 1 个，首个错误：HTTP 403')
    expect(store.refreshing).toBe(false)
  })
})
