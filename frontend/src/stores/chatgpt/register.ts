/**
 * ChatGPT Registration Machine Pinia Store
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  fetchRegisterConfig,
  updateRegisterConfig,
  startRegister,
  stopRegister,
  resetRegister,
  createRegisterEventsToken,
  createRegisterEventSource,
  type RegisterConfig,
  type RegisterMailConfig,
  type RegisterMailProvider,
  type RegisterMailProviderType,
  type RegisterMode,
  type RegisterUpdatePayload,
} from '@/api/chatgpt'

const defaultProvider = (): RegisterMailProvider => ({
  type: 'mailtm',
  enable: true,
  api_base: 'https://api.mail.tm',
  api_key: '',
  domain: [],
  subdomain: [],
  wildcard: false,
  random_subdomain: false,
})

const defaultMailConfig = (): RegisterMailConfig => ({
  request_timeout: 30,
  wait_timeout: 120,
  wait_interval: 3,
  providers: [defaultProvider()],
})

function cloneMailConfig(mail?: RegisterMailConfig): RegisterMailConfig {
  const fallback = defaultMailConfig()
  return {
    request_timeout: mail?.request_timeout || fallback.request_timeout,
    wait_timeout: mail?.wait_timeout || fallback.wait_timeout,
    wait_interval: mail?.wait_interval || fallback.wait_interval,
    providers: (mail?.providers?.length ? mail.providers : fallback.providers).map((p) => ({
      ...defaultProvider(),
      ...p,
      type: (p.type || 'mailtm') as RegisterMailProviderType,
      enable: p.enable ?? true,
      domain: Array.isArray(p.domain) ? [...p.domain] : [],
      subdomain: Array.isArray(p.subdomain) ? [...p.subdomain] : [],
    })),
  }
}

export const useChatGPTRegisterStore = defineStore('chatgptRegister', () => {
  const config = ref<RegisterConfig | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)
  const sseConnected = ref(false)
  const sseFallback = ref(false)

  const formMode = ref<RegisterMode>('total')
  const formTotal = ref(100)
  const formThreads = ref(3)
  const formProxy = ref('')
  const formTargetQuota = ref(100)
  const formTargetAvailable = ref(10)
  const formCheckInterval = ref(5)
  const formMail = ref<RegisterMailConfig>(defaultMailConfig())

  let eventSource: EventSource | null = null
  let fallbackTimer: number | null = null

  const isRunning = computed(() => config.value?.enabled ?? false)
  const stats = computed(() => config.value?.stats ?? null)
  const recentLogs = computed(() => [...(config.value?.logs ?? [])].reverse().slice(0, 200))

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

    if (config.value.mail?.providers?.length) {
      formMail.value = cloneMailConfig(config.value.mail)
    } else {
      formMail.value = cloneMailConfig({
        request_timeout: 30,
        wait_timeout: 120,
        wait_interval: 3,
        providers: [{
          type: config.value.mail_provider || 'mailtm',
          enable: true,
          api_base: config.value.mail_api_base || 'https://api.mail.tm',
          api_key: config.value.mail_api_key || '',
        }],
      })
    }
  }

  function addProvider(type: RegisterMailProviderType | string = 'mailtm'): void {
    formMail.value.providers.push({ ...defaultProvider(), type })
  }

  function removeProvider(index: number): void {
    formMail.value.providers.splice(index, 1)
    if (formMail.value.providers.length === 0) addProvider()
  }

  function buildMailPayload(): RegisterMailConfig {
    return {
      request_timeout: Number(formMail.value.request_timeout) || 30,
      wait_timeout: Number(formMail.value.wait_timeout) || 120,
      wait_interval: Number(formMail.value.wait_interval) || 3,
      providers: formMail.value.providers.map((p) => ({
        ...p,
        enable: p.enable ?? true,
        domain: Array.isArray(p.domain) ? p.domain.filter(Boolean) : [],
        subdomain: Array.isArray(p.subdomain) ? p.subdomain.filter(Boolean) : [],
      })),
    }
  }

  async function save(): Promise<void> {
    saving.value = true
    error.value = null
    try {
      const mail = buildMailPayload()
      const firstProvider = mail.providers[0]
      const payload: RegisterUpdatePayload = {
        mode: formMode.value,
        total: formTotal.value,
        threads: formThreads.value,
        proxy: formProxy.value,
        target_quota: formTargetQuota.value,
        target_available: formTargetAvailable.value,
        check_interval: formCheckInterval.value,
        mail,
        // Legacy compatibility for older backend.
        mail_provider: firstProvider?.type,
        mail_api_base: firstProvider?.api_base,
        mail_api_key: firstProvider?.api_key,
      }
      const data = await updateRegisterConfig(payload)
      config.value = data.register
      syncFormFromConfig()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '保存注册机配置失败'
    } finally {
      saving.value = false
    }
  }

  async function toggle(): Promise<void> {
    error.value = null
    try {
      await save()
      const data = config.value?.enabled ? await stopRegister() : await startRegister()
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

  function startPollingFallback(): void {
    sseFallback.value = true
    if (fallbackTimer) return
    fallbackTimer = window.setInterval(() => {
      fetchRegisterConfig()
        .then((data) => { config.value = data.register })
        .catch(() => {})
    }, 3000)
  }

  async function startSSE(): Promise<void> {
    stopSSE()
    try {
      const { token } = await createRegisterEventsToken()
      if (!token) throw new Error('empty event token')
      eventSource = createRegisterEventSource(token)
      eventSource.onopen = () => {
        sseConnected.value = true
        sseFallback.value = false
      }
      eventSource.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as { register?: RegisterConfig }
          config.value = payload.register || (payload as unknown as RegisterConfig)
        } catch {
          // Ignore malformed event frames.
        }
      }
      eventSource.onerror = () => {
        sseConnected.value = false
        if (eventSource) {
          eventSource.close()
          eventSource = null
        }
        startPollingFallback()
      }
    } catch {
      startPollingFallback()
    }
  }

  function stopSSE(): void {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    if (fallbackTimer) {
      window.clearInterval(fallbackTimer)
      fallbackTimer = null
    }
    sseConnected.value = false
    sseFallback.value = false
  }

  return {
    config,
    loading,
    saving,
    error,
    sseConnected,
    sseFallback,
    formMode,
    formTotal,
    formThreads,
    formProxy,
    formTargetQuota,
    formTargetAvailable,
    formCheckInterval,
    formMail,
    isRunning,
    stats,
    recentLogs,
    progress,
    load,
    save,
    toggle,
    reset,
    addProvider,
    removeProvider,
    startSSE,
    stopSSE,
  }
})
