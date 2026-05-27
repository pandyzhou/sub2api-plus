/**
 * ChatGPT Accounts Pinia Store
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  fetchAccounts,
  createAccounts,
  deleteAccounts,
  refreshAccounts,
  updateAccount,
  fetchAccountPoolConfig,
  updateAccountPoolConfig,
  exportAccounts,
  type ChatGPTAccount,
  type ChatGPTAccountPoolConfig,
} from '@/api/chatgpt'

const defaultPoolConfig = (): ChatGPTAccountPoolConfig => ({
  refresh_account_interval_minute: 5,
  auto_remove_invalid_accounts: false,
  auto_remove_rate_limited_accounts: false,
  image_account_concurrency: 3,
})

export const useChatGPTAccountsStore = defineStore('chatgptAccounts', () => {
  const accounts = ref<ChatGPTAccount[]>([])
  const loading = ref(false)
  const configLoading = ref(false)
  const configSaving = ref(false)
  const error = ref<string | null>(null)
  const selectedIds = ref<Set<string>>(new Set())

  const poolConfig = ref<ChatGPTAccountPoolConfig>(defaultPoolConfig())

  const filterStatus = ref<string | '全部'>('全部')
  const filterType = ref<string | '全部'>('全部')
  const searchQuery = ref('')

  const editingAccount = ref<ChatGPTAccount | null>(null)
  const editType = ref<string>('')
  const editStatus = ref<string>('正常')
  const editQuota = ref(0)
  const editImageQuotaUnknown = ref(false)

  const showExportDialog = ref(false)
  const exportFormat = ref<'json' | 'zip'>('json')

  const statusCounts = computed(() => {
    const counts: Record<string, number> = { total: 0, '正常': 0, '限流': 0, '异常': 0, '禁用': 0 }
    for (const acc of accounts.value) {
      counts.total++
      counts[acc.status] = (counts[acc.status] || 0) + 1
    }
    return counts
  })

  const totalQuota = computed(() => accounts.value
    .filter((a) => a.status === '正常' && !a.image_quota_unknown)
    .reduce((sum, a) => sum + (Number(a.quota) || 0), 0))

  const typeOptions = computed(() => Array.from(new Set(accounts.value.map((a) => a.type).filter(Boolean))))

  const filteredAccounts = computed(() => {
    let result = accounts.value
    if (filterStatus.value !== '全部') result = result.filter((a) => a.status === filterStatus.value)
    if (filterType.value !== '全部') result = result.filter((a) => a.type === filterType.value)
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      result = result.filter((a) =>
        (a.email && a.email.toLowerCase().includes(q)) ||
        (a.access_token && a.access_token.toLowerCase().includes(q)) ||
        (a.account_id && a.account_id.toLowerCase().includes(q)) ||
        (a.user_id && a.user_id.toLowerCase().includes(q)) ||
        (a.last_refresh_error && a.last_refresh_error.toLowerCase().includes(q)) ||
        (a.last_token_refresh_error && a.last_token_refresh_error.toLowerCase().includes(q)),
      )
    }
    return result
  })

  const isSelected = (token: string) => selectedIds.value.has(token)
  const selectedCount = computed(() => selectedIds.value.size)
  const selectedAccounts = computed(() => accounts.value.filter((a) => selectedIds.value.has(a.access_token)))

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await fetchAccounts()
      accounts.value = data.items || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : '加载账号列表失败'
    } finally {
      loading.value = false
    }
  }

  async function loadPoolConfig(): Promise<void> {
    configLoading.value = true
    try {
      poolConfig.value = { ...defaultPoolConfig(), ...(await fetchAccountPoolConfig()) }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '加载账号池配置失败'
    } finally {
      configLoading.value = false
    }
  }

  function updatePoolConfig(updates: Partial<ChatGPTAccountPoolConfig>): void {
    poolConfig.value = { ...poolConfig.value, ...updates }
  }

  async function savePoolConfig(): Promise<void> {
    configSaving.value = true
    try {
      poolConfig.value = await updateAccountPoolConfig(poolConfig.value)
    } catch (err) {
      error.value = err instanceof Error ? err.message : '保存账号池配置失败'
    } finally {
      configSaving.value = false
    }
  }

  function toggleSelect(token: string): void {
    const next = new Set(selectedIds.value)
    if (next.has(token)) next.delete(token)
    else next.add(token)
    selectedIds.value = next
  }

  function selectAll(): void {
    selectedIds.value = new Set(filteredAccounts.value.map((a) => a.access_token))
  }

  function clearSelection(): void {
    selectedIds.value = new Set()
  }

  async function importAccounts(tokens: string[], payloads: Record<string, unknown>[] = []): Promise<void> {
    await createAccounts(tokens, payloads)
    await load()
  }

  async function removeSelected(): Promise<void> {
    const tokens = Array.from(selectedIds.value)
    if (!tokens.length) return
    await deleteAccounts(tokens)
    clearSelection()
    await load()
  }

  async function refreshSelected(): Promise<void> {
    const tokens = Array.from(selectedIds.value)
    if (!tokens.length) return
    await refreshAccounts(tokens)
    await load()
  }

  async function refreshAll(): Promise<void> {
    await refreshAccounts([])
    await load()
  }

  async function saveEdit(): Promise<void> {
    if (!editingAccount.value) return
    await updateAccount(editingAccount.value.access_token, {
      type: editType.value || undefined,
      status: editStatus.value || undefined,
      quota: editQuota.value,
      image_quota_unknown: editImageQuotaUnknown.value,
    })
    editingAccount.value = null
    await load()
  }

  function openEdit(account: ChatGPTAccount): void {
    editingAccount.value = account
    editType.value = account.type
    editStatus.value = account.status
    editQuota.value = Number(account.quota) || 0
    editImageQuotaUnknown.value = Boolean(account.image_quota_unknown)
  }

  function closeEdit(): void {
    editingAccount.value = null
  }

  async function downloadExport(): Promise<void> {
    const tokens = Array.from(selectedIds.value)
    const { blob, filename } = await exportAccounts(tokens, exportFormat.value)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    showExportDialog.value = false
  }

  return {
    accounts,
    loading,
    configLoading,
    configSaving,
    error,
    selectedIds,
    poolConfig,
    filterStatus,
    filterType,
    searchQuery,
    editingAccount,
    editType,
    editStatus,
    editQuota,
    editImageQuotaUnknown,
    showExportDialog,
    exportFormat,
    statusCounts,
    totalQuota,
    typeOptions,
    filteredAccounts,
    selectedCount,
    selectedAccounts,
    load,
    loadPoolConfig,
    savePoolConfig,
    updatePoolConfig,
    toggleSelect,
    selectAll,
    clearSelection,
    importAccounts,
    removeSelected,
    refreshSelected,
    refreshAll,
    saveEdit,
    openEdit,
    closeEdit,
    downloadExport,
    isSelected,
  }
})
