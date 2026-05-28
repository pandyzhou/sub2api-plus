import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatGPTRegisterStore } from '@/stores/chatgpt/register'
import { resetRegister, startRegister, stopRegister, updateRegisterConfig } from '@/api/chatgpt'
import type { RegisterConfig } from '@/api/chatgpt'

vi.mock('@/api/chatgpt', () => ({
  fetchRegisterConfig: vi.fn(),
  updateRegisterConfig: vi.fn(),
  startRegister: vi.fn(),
  stopRegister: vi.fn(),
  resetRegister: vi.fn(),
  createRegisterEventsToken: vi.fn(),
  createRegisterEventSource: vi.fn(),
}))

function config(overrides: Partial<RegisterConfig> = {}): RegisterConfig {
  return {
    enabled: false,
    mode: 'total',
    total: 10,
    threads: 3,
    proxy: '',
    target_quota: 100,
    target_available: 10,
    check_interval: 5,
    mail: { request_timeout: 30, wait_timeout: 120, wait_interval: 3, providers: [] },
    stats: { success: 0, fail: 0, done: 0, running: 0, threads: 3 },
    logs: [],
    ...overrides,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useChatGPTRegisterStore P2 UX states', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(updateRegisterConfig).mockResolvedValue({ register: config() })
  })

  it('toggle exposes busy state while auto-saving and starting', async () => {
    const start = deferred<{ register: RegisterConfig }>()
    vi.mocked(startRegister).mockReturnValue(start.promise as ReturnType<typeof startRegister>)
    const store = useChatGPTRegisterStore()
    store.config = config({ enabled: false })

    const task = store.toggle()

    expect(store.toggling).toBe(true)
    expect(store.actionBusy).toBe(true)
    expect(store.formDisabled).toBe(true)

    start.resolve({ register: config({ enabled: true }) })
    await task

    expect(store.toggling).toBe(false)
    expect(store.isRunning).toBe(true)
  })

  it('reset exposes a separate resetting state', async () => {
    const reset = deferred<{ register: RegisterConfig }>()
    vi.mocked(resetRegister).mockReturnValue(reset.promise as ReturnType<typeof resetRegister>)
    const store = useChatGPTRegisterStore()
    store.config = config({ enabled: false })

    const task = store.reset()

    expect(store.resetting).toBe(true)
    expect(store.actionBusy).toBe(true)

    reset.resolve({ register: config() })
    await task

    expect(store.resetting).toBe(false)
  })

  it('stopping uses toggle busy state without auto-saving first', async () => {
    const stop = deferred<{ register: RegisterConfig }>()
    vi.mocked(stopRegister).mockReturnValue(stop.promise as ReturnType<typeof stopRegister>)
    const store = useChatGPTRegisterStore()
    store.config = config({ enabled: true })

    const task = store.toggle()

    expect(store.toggling).toBe(true)
    expect(updateRegisterConfig).not.toHaveBeenCalled()

    stop.resolve({ register: config({ enabled: false }) })
    await task

    expect(store.toggling).toBe(false)
  })
})
