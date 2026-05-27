<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('chatgpt.accounts.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('chatgpt.accounts.subtitle') }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button @click="store.refreshAll()" :disabled="store.loading" class="btn btn-secondary">
            <span :class="store.loading ? 'animate-spin' : ''">↻</span>
            {{ t('chatgpt.accounts.refreshAll') }}
          </button>
          <button @click="showImportDialog = true" class="btn btn-primary">
            + {{ t('chatgpt.accounts.import') }}
          </button>
        </div>
      </div>

      <div v-if="store.error" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
        <div class="flex items-start justify-between gap-4">
          <span>{{ store.error }}</span>
          <button @click="store.error = null" class="text-xs font-medium underline-offset-4 hover:underline">
            Dismiss
          </button>
        </div>
      </div>

      <section class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
        <div v-for="stat in statCards" :key="stat.label" class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg" :class="stat.iconBg">
              <span class="h-2.5 w-2.5 rounded-full" :class="stat.dot" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ stat.label }}
              </p>
              <p class="mt-1 text-xl font-bold tabular-nums" :class="stat.color">
                {{ stat.value }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-4">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-center">
          <div class="grid flex-1 gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
            <input
              v-model="store.searchQuery"
              type="text"
              :placeholder="t('chatgpt.accounts.search')"
              class="input"
            />
            <select v-model="store.filterStatus" class="input">
              <option value="全部">{{ t('chatgpt.accounts.allStatus') }}</option>
              <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
              <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
              <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
              <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
            </select>
          </div>

          <div v-if="store.selectedCount > 0" class="flex flex-wrap items-center gap-2">
            <span class="rounded-full bg-gray-100 px-3 py-2 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              {{ t('chatgpt.accounts.selectedCount', { n: store.selectedCount }) }}
            </span>
            <button @click="store.refreshSelected()" class="btn btn-secondary btn-sm">
              {{ t('chatgpt.accounts.refreshSelected', { n: store.selectedCount }) }}
            </button>
            <button @click="store.showExportDialog = true" class="btn btn-secondary btn-sm">
              {{ t('chatgpt.accounts.exportSelected', { n: store.selectedCount }) }}
            </button>
            <button @click="confirmDelete()" class="btn btn-danger btn-sm">
              {{ t('chatgpt.accounts.deleteSelected', { n: store.selectedCount }) }}
            </button>
          </div>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="flex items-center justify-between gap-4">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('chatgpt.accounts.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ store.filteredAccounts.length }} visible / {{ store.accounts.length }} total
              </p>
            </div>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full min-w-max divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-5 py-3 text-left">
                  <input
                    type="checkbox"
                    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-900"
                    :checked="store.selectedCount > 0 && store.selectedCount === store.filteredAccounts.length"
                    @change="toggleSelectAll"
                  />
                </th>
                <th class="table-th">{{ t('chatgpt.accounts.colStatus') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colType') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colEmail') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colQuota') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colSuccess') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colLastUsed') }}</th>
                <th class="table-th">{{ t('chatgpt.accounts.colActions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-900">
              <tr
                v-for="acc in store.filteredAccounts"
                :key="acc.access_token"
                class="transition hover:bg-gray-50 dark:hover:bg-dark-800"
              >
                <td class="px-5 py-4">
                  <input
                    type="checkbox"
                    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-900"
                    :checked="store.isSelected(acc.access_token)"
                    @change="store.toggleSelect(acc.access_token)"
                  />
                </td>
                <td class="table-td">
                  <span :class="statusBadgeClass(acc.status)" class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium">
                    {{ acc.status }}
                  </span>
                </td>
                <td class="table-td">
                  <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                    {{ acc.type || '-' }}
                  </span>
                </td>
                <td class="table-td">
                  <div class="font-medium text-gray-900 dark:text-white">{{ acc.email || '-' }}</div>
                  <div class="mt-1 max-w-[16rem] truncate font-mono text-xs text-gray-400 dark:text-dark-400">{{ acc.access_token }}</div>
                </td>
                <td class="table-td font-semibold tabular-nums">
                  {{ acc.image_quota_unknown ? '?' : (acc.quota ?? 0) }}
                </td>
                <td class="table-td">
                  <span class="font-semibold tabular-nums">{{ acc.success ?? 0 }}</span>
                  <span class="text-gray-400 dark:text-dark-400">/{{ (acc.success ?? 0) + (acc.fail ?? 0) }}</span>
                </td>
                <td class="table-td text-gray-500 dark:text-gray-400">{{ formatDate(acc.last_used_at) }}</td>
                <td class="table-td">
                  <button @click="store.openEdit(acc)" class="btn btn-secondary btn-sm">
                    {{ t('common.edit') }}
                  </button>
                </td>
              </tr>
              <tr v-if="store.filteredAccounts.length === 0 && !store.loading">
                <td colspan="8" class="px-5 py-12 text-center">
                  <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ store.error ? t('chatgpt.accounts.errorLoading') : t('chatgpt.accounts.empty') }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    可导入 Token，或通过注册机创建账号。
                  </div>
                </td>
              </tr>
              <tr v-if="store.loading">
                <td colspan="8" class="px-5 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('common.loading') }}...
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <Teleport to="body">
      <div v-if="showImportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
        <div class="w-full max-w-2xl rounded-2xl border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-5">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('chatgpt.accounts.importTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">粘贴 access token，或粘贴从现有会话导出的结构化 JSON。</p>
          </div>
          <textarea
            v-model="importText"
            rows="9"
            :placeholder="t('chatgpt.accounts.importPlaceholder')"
            class="input mb-5 font-mono text-xs"
          />
          <div class="flex justify-end gap-2">
            <button @click="showImportDialog = false" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="handleImport" :disabled="!importText.trim()" class="btn btn-primary">{{ t('chatgpt.accounts.doImport') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="store.editingAccount" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
        <div class="w-full max-w-md rounded-2xl border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="mb-5 text-xl font-semibold text-gray-900 dark:text-white">{{ t('chatgpt.accounts.editTitle') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="input-label">{{ t('chatgpt.accounts.colType') }}</label>
              <input v-model="store.editType" type="text" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('chatgpt.accounts.colStatus') }}</label>
              <select v-model="store.editStatus" class="input">
                <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
                <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
                <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
                <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('chatgpt.accounts.colQuota') }}</label>
              <input v-model.number="store.editQuota" type="number" class="input" />
            </div>
          </div>
          <div class="mt-6 flex justify-end gap-2">
            <button @click="store.closeEdit()" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="store.saveEdit()" class="btn btn-primary">{{ t('common.save') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="store.showExportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
        <div class="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="mb-5 text-xl font-semibold text-gray-900 dark:text-white">{{ t('chatgpt.accounts.exportTitle') }}</h2>
          <select v-model="store.exportFormat" class="input mb-5">
            <option value="json">JSON</option>
            <option value="zip">ZIP</option>
          </select>
          <div class="flex justify-end gap-2">
            <button @click="store.showExportDialog = false" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="store.downloadExport()" class="btn btn-primary">{{ t('chatgpt.accounts.doExport') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
        <div class="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <h2 class="mb-3 text-xl font-semibold text-gray-900 dark:text-white">{{ t('chatgpt.accounts.deleteConfirmTitle') }}</h2>
          <p class="mb-6 text-sm leading-6 text-gray-500 dark:text-gray-400">
            {{ t('chatgpt.accounts.deleteConfirmMsg', { n: store.selectedCount }) }}
          </p>
          <div class="flex justify-end gap-2">
            <button @click="showDeleteConfirm = false" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="handleDelete()" class="btn btn-danger">{{ t('common.confirm') }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useChatGPTAccountsStore } from '@/stores/chatgpt'

const { t } = useI18n()
const store = useChatGPTAccountsStore()

const showImportDialog = ref(false)
const showDeleteConfirm = ref(false)
const importText = ref('')

const statCards = computed(() => [
  { key: 'total', label: t('chatgpt.accounts.statsTotal'), value: store.statusCounts.total ?? 0, color: 'text-gray-900 dark:text-white', dot: 'bg-gray-500', iconBg: 'bg-gray-100 dark:bg-dark-700' },
  { key: '正常', label: t('chatgpt.accounts.statusNormal'), value: store.statusCounts['正常'] ?? 0, color: 'text-emerald-600 dark:text-emerald-400', dot: 'bg-emerald-500', iconBg: 'bg-emerald-100 dark:bg-emerald-900/30' },
  { key: '限流', label: t('chatgpt.accounts.statusLimited'), value: store.statusCounts['限流'] ?? 0, color: 'text-amber-600 dark:text-amber-400', dot: 'bg-amber-500', iconBg: 'bg-amber-100 dark:bg-amber-900/30' },
  { key: '异常', label: t('chatgpt.accounts.statusError'), value: store.statusCounts['异常'] ?? 0, color: 'text-red-600 dark:text-red-400', dot: 'bg-red-500', iconBg: 'bg-red-100 dark:bg-red-900/30' },
  { key: '禁用', label: t('chatgpt.accounts.statusDisabled'), value: store.statusCounts['禁用'] ?? 0, color: 'text-gray-500 dark:text-gray-400', dot: 'bg-gray-400', iconBg: 'bg-gray-100 dark:bg-dark-700' },
  { key: 'quota', label: t('chatgpt.accounts.statsQuota'), value: store.totalQuota, color: 'text-cyan-600 dark:text-cyan-400', dot: 'bg-cyan-500', iconBg: 'bg-cyan-100 dark:bg-cyan-900/30' },
])

onMounted(() => {
  store.load()
})

function toggleSelectAll(): void {
  if (store.selectedCount > 0 && store.selectedCount === store.filteredAccounts.length) {
    store.clearSelection()
  } else {
    store.selectAll()
  }
}

function confirmDelete(): void {
  showDeleteConfirm.value = true
}

async function handleDelete(): Promise<void> {
  showDeleteConfirm.value = false
  await store.removeSelected()
}

async function handleImport(): Promise<void> {
  const text = importText.value.trim()
  if (!text) return

  try {
    const parsed = JSON.parse(text)
    if (Array.isArray(parsed)) {
      await store.importAccounts([], parsed)
    } else {
      await store.importAccounts([], [parsed])
    }
  } catch {
    const tokens = text.split('\n').map((l) => l.trim()).filter(Boolean)
    await store.importAccounts(tokens, [])
  }

  importText.value = ''
  showImportDialog.value = false
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case '正常': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case '限流': return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case '异常': return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case '禁用': return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  }
}

function formatDate(dateStr?: string | null): string {
  if (!dateStr) return '-'
  try {
    return new Date(dateStr).toLocaleDateString()
  } catch {
    return dateStr
  }
}
</script>

<style scoped>
.table-th {
  @apply px-5 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400;
}
.table-td {
  @apply whitespace-nowrap px-5 py-4 text-sm text-gray-700 dark:text-gray-300;
}
</style>
