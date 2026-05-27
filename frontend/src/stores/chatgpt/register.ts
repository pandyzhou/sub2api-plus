/**
 * ChatGPT Registration Machine Pinia Store
 *
 * Manages the automated ChatGPT account registration machine state:
 * - configuration, start/stop/reset control
 * - real-time SSE log streaming
 * - statistics display
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  fetchRegisterConfig,
  updateRegisterConfig,
  startRegister,
  stopRegister,
  resetRegister,
  type RegisterConfig,
  type RegisterMode,
  type RegisterUpdatePayload,
} from '@/api/chatgpt'

export const useChatGPTRegisterStore = defineStore('chatgptRegister', () => {
  // ==================== State ====================

  const config = ref<RegisterConfig | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)

  // Form state (before saving)
  const formMode = ref<RegisterMode>('total')
  const formTotal = ref(100)
  const formThreads = ref(3)
  const formProxy = ref('')
  const formTargetQuota = ref(100)
  const formTargetAvailable = ref(10)
  const formCheckInterval = ref(5)

  // SSE connection
  let eventSource: EventSource | null = null

  // ==================== Computed ====================

  const isRunning = computed(() => config.value?.enabled ?? false)

  const stats = computed(() => config.value?.stats ?? null)

  const recentLogs = computed(() => {
    const logs = config.value?.logs ?? []
    return [...logs].reverse().slice(0, 200)
  })

  const progress = computed(() => {
    if (!config.value) return null
    const s = config.value.stats
    const total = config.value.mode === 'total'
      ? config.value.total
      : config.value.mode === 'quota'
        ? config.value.target_quota
        : config.value.target_available
    const done = s.done ?? 0
    return total > 0 ? Math.min(100, Math.round((done / total) * 100)) : 0
  })

  // ==================== Actions ====================

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await fetchRegisterConfig()
      config.value = data.register
      syncFormFromConfig()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '加载注册机配置失败'
    } finally {
      loading.value = false
    }
  }

  function syncFormFromConfig(): void {
    if (!config.value) return
    formMode.value = config.value.mode
    formTotal.value = config.value.total
    formThreads.value = config.value.threads
    formProxy.value = config.value.proxy || ''
    formTargetQuota.value = config.value.target_quota
    formTargetAvailable.value = config.value.target_available
    formCheckInterval.value = config.value.check_interval
  }

  async function save(): Promise<void> {
    saving.value = true
    error.value = null
    try {
      const payload: RegisterUpdatePayload = {
        mode: formMode.value,
        total: formTotal.value,
        threads: formThreads.value,
        proxy: formProxy.value,
        target_quota: formTargetQuota.value,
        target_available: formTargetAvailable.value,
        check_interval: formCheckInterval.value,
      }
      const data = await updateRegisterConfig(payload)
      config.value = data.register
    } catch (err) {
      error.value = err instanceof Error ? err.message : '保存注册机配置失败'
    } finally {
      saving.value = false
    }
  }

  async function toggle(): Promise<void> {
    error.value = null
    try {
      // Save config before toggling
      await save()
      const data = config.value?.enabled
        ? await stopRegister()
        : await startRegister()
      config.value = data.register
    } catch (err) {
      error.value = err instanceof Error ? err.message : '操作注册机失败'
    }
  }

  async function reset(): Promise<void> {
    error.value = null
    try {
      const data = await resetRegister()
      config.value = data.register
      syncFormFromConfig()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '重置注册机统计失败'
    }
  }

  function startSSE(): void {
    // Browser EventSource cannot attach the JWT Authorization header used by sub2api admin APIs.
    // Keep this as a no-op; explicit load/save/start/stop actions refresh the register state.
    stopSSE()
  }

  function stopSSE(): void {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  }

  return {
    // state
    config,
    loading,
    saving,
    error,
    formMode,
    formTotal,
    formThreads,
    formProxy,
    formTargetQuota,
    formTargetAvailable,
    formCheckInterval,
    // computed
    isRunning,
    stats,
    recentLogs,
    progress,
    // actions
    load,
    save,
    toggle,
    reset,
    startSSE,
    stopSSE,
  }
})
