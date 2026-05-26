/**
 * ChatGPT Accounts Pinia Store
 *
 * Manages ChatGPT access token account list state:
 * - fetching, filtering, sorting
 * - import, delete, refresh, export operations
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  fetchAccounts,
  createAccounts,
  deleteAccounts,
  refreshAccounts,
  updateAccount,
  exportAccounts,
  type ChatGPTAccount,
  type ChatGPTAccountMutationResponse,
} from '@/api/chatgpt'

export const useChatGPTAccountsStore = defineStore('chatgptAccounts', () => {
  // ==================== State ====================

  const accounts = ref<ChatGPTAccount[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const selectedIds = ref<Set<string>>(new Set())

  // Filters
  const filterStatus = ref<string | '全部'>('全部')
  const filterType = ref<string | '全部'>('全部')
  const searchQuery = ref('')

  // Edit modal state
  const editingAccount = ref<ChatGPTAccount | null>(null)
  const editType = ref<string>('')
  const editStatus = ref<string>('正常')
  const editQuota = ref(0)

  // Export dialog state
  const showExportDialog = ref(false)
  const exportFormat = ref<'json' | 'zip'>('json')

  // ==================== Computed ====================

  const statusCounts = computed(() => {
    const counts: Record<string, number> = { total: 0, '正常': 0, '限流': 0, '异常': 0, '禁用': 0 }
    for (const acc of accounts.value) {
      counts.total++
      counts[acc.status] = (counts[acc.status] || 0) + 1
    }
    return counts
  })

  const totalQuota = computed(() => {
    return accounts.value
      .filter((a) => a.status === '正常' && !a.image_quota_unknown)
      .reduce((sum, a) => sum + (a.quota || 0), 0)
  })

  const filteredAccounts = computed(() => {
    let result = accounts.value

    if (filterStatus.value !== '全部') {
      result = result.filter((a) => a.status === filterStatus.value)
    }
    if (filterType.value !== '全部') {
      result = result.filter((a) => a.type === filterType.value)
    }
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      result = result.filter(
        (a) =>
          (a.email && a.email.toLowerCase().includes(q)) ||
          (a.access_token && a.access_token.toLowerCase().includes(q)) ||
          (a.account_id && a.account_id.toLowerCase().includes(q)),
      )
    }

    return result
  })

  const isSelected = (token: string) => selectedIds.value.has(token)

  const selectedCount = computed(() => selectedIds.value.size)

  const selectedAccounts = computed(() => {
    return accounts.value.filter((a) => selectedIds.value.has(a.access_token))
  })

  // ==================== Actions ====================

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

  function toggleSelect(token: string): void {
    const next = new Set(selectedIds.value)
    if (next.has(token)) {
      next.delete(token)
    } else {
      next.add(token)
    }
    selectedIds.value = next
  }

  function selectAll(): void {
    const allTokens = filteredAccounts.value.map((a) => a.access_token)
    selectedIds.value = new Set(allTokens)
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
    })
    editingAccount.value = null
    await load()
  }

  function openEdit(account: ChatGPTAccount): void {
    editingAccount.value = account
    editType.value = account.type
    editStatus.value = account.status
    editQuota.value = account.quota
  }

  function closeEdit(): void {
    editingAccount.value = null
  }

  async function downloadExport(): Promise<void> {
    const tokens = Array.from(selectedIds.value)
    const blob = await exportAccounts(exportFormat.value, tokens)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chatgpt-accounts.${exportFormat.value}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    showExportDialog.value = false
  }

  return {
    // state
    accounts,
    loading,
    error,
    selectedIds,
    filterStatus,
    filterType,
    searchQuery,
    editingAccount,
    editType,
    editStatus,
    editQuota,
    showExportDialog,
    exportFormat,
    // computed
    statusCounts,
    totalQuota,
    filteredAccounts,
    selectedCount,
    selectedAccounts,
    // actions
    load,
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
