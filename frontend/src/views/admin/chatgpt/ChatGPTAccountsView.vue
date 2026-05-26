<template>
  <AppLayout>
    <ChatGPTConnectionSettings
      :show="showConnectionDialog"
      :force-setup="true"
      @connected="onConnected"
      @cancel="showConnectionDialog = false"
    />

    <div class="space-y-6 p-4 sm:p-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ t('chatgpt.accounts.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('chatgpt.accounts.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="store.refreshAll()"
            :disabled="store.loading"
            class="btn btn-secondary"
          >
            <span :class="store.loading ? 'animate-spin' : ''">&#x21bb;</span>
            {{ t('chatgpt.accounts.refreshAll') }}
          </button>
          <button @click="showImportDialog = true" class="btn btn-primary">
            + {{ t('chatgpt.accounts.import') }}
          </button>
        </div>
      </div>

      <!-- Error banner -->
      <div v-if="store.error" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/30 dark:text-red-300">
        {{ store.error }}
        <button @click="store.error = null" class="ml-2 underline">Dismiss</button>
      </div>

      <!-- Stats cards -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <div v-for="stat in statCards" :key="stat.label" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold" :class="stat.color">{{ store.statusCounts[stat.key] ?? 0 }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
        </div>
      </div>

      <!-- Filters -->
      <div class="flex flex-wrap items-center gap-3">
        <input
          v-model="store.searchQuery"
          type="text"
          :placeholder="t('chatgpt.accounts.search')"
          class="input w-full sm:w-64"
        />
        <select v-model="store.filterStatus" class="input w-full sm:w-36">
          <option value="全部">{{ t('chatgpt.accounts.allStatus') }}</option>
          <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
          <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
          <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
          <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
        </select>

        <div class="flex-1" />

        <!-- Batch actions -->
        <button
          v-if="store.selectedCount > 0"
          @click="store.refreshSelected()"
          class="btn btn-secondary text-sm"
        >
          {{ t('chatgpt.accounts.refreshSelected', { n: store.selectedCount }) }}
        </button>
        <button
          v-if="store.selectedCount > 0"
          @click="store.showExportDialog = true"
          class="btn btn-secondary text-sm"
        >
          {{ t('chatgpt.accounts.exportSelected', { n: store.selectedCount }) }}
        </button>
        <button
          v-if="store.selectedCount > 0"
          @click="confirmDelete()"
          class="btn btn-danger text-sm"
        >
          {{ t('chatgpt.accounts.deleteSelected', { n: store.selectedCount }) }}
        </button>
        <span v-if="store.selectedCount > 0" class="text-sm text-gray-500">
          {{ t('chatgpt.accounts.selectedCount', { n: store.selectedCount }) }}
        </span>
      </div>

      <!-- Table -->
      <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
        <table class="w-full text-left text-sm">
          <thead class="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
            <tr>
              <th class="px-3 py-3">
                <input
                  type="checkbox"
                  :checked="store.selectedCount > 0 && store.selectedCount === store.filteredAccounts.length"
                  @change="toggleSelectAll"
                />
              </th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colStatus') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colType') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colEmail') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colQuota') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colSuccess') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colLastUsed') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('chatgpt.accounts.colActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="acc in store.filteredAccounts"
              :key="acc.access_token"
              class="border-b border-gray-100 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800/50"
            >
              <td class="px-3 py-2">
                <input
                  type="checkbox"
                  :checked="store.isSelected(acc.access_token)"
                  @change="store.toggleSelect(acc.access_token)"
                />
              </td>
              <td class="px-3 py-2">
                <span :class="statusBadgeClass(acc.status)" class="inline-block rounded-full px-2 py-0.5 text-xs font-medium">
                  {{ acc.status }}
                </span>
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ acc.type || '-' }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ acc.email || '-' }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                {{ acc.image_quota_unknown ? '?' : (acc.quota ?? 0) }}
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                {{ acc.success }}/{{ acc.success + acc.fail }}
              </td>
              <td class="px-3 py-2 text-gray-400 dark:text-gray-500">{{ formatDate(acc.last_used_at) }}</td>
              <td class="px-3 py-2">
                <button @click="store.openEdit(acc)" class="text-blue-600 hover:underline dark:text-blue-400">
                  {{ t('common.edit') }}
                </button>
              </td>
            </tr>
            <tr v-if="store.filteredAccounts.length === 0 && !store.loading">
              <td colspan="8" class="px-3 py-8 text-center text-gray-400 dark:text-gray-500">
                {{ store.error ? t('chatgpt.accounts.errorLoading') : t('chatgpt.accounts.empty') }}
              </td>
            </tr>
            <tr v-if="store.loading">
              <td colspan="8" class="px-3 py-8 text-center text-gray-400 dark:text-gray-500">
                {{ t('common.loading') }}...
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Import dialog -->
    <Teleport to="body">
      <div v-if="showImportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
          <h2 class="mb-4 text-lg font-semibold">{{ t('chatgpt.accounts.importTitle') }}</h2>
          <textarea
            v-model="importText"
            rows="8"
            :placeholder="t('chatgpt.accounts.importPlaceholder')"
            class="input mb-4 w-full font-mono text-xs"
          />
          <div class="flex justify-end gap-2">
            <button @click="showImportDialog = false" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="handleImport" :disabled="!importText.trim()" class="btn btn-primary">{{ t('chatgpt.accounts.doImport') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit dialog -->
    <Teleport to="body">
      <div v-if="store.editingAccount" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
          <h2 class="mb-4 text-lg font-semibold">{{ t('chatgpt.accounts.editTitle') }}</h2>
          <div class="space-y-3">
            <div>
              <label class="mb-1 block text-xs font-medium">{{ t('chatgpt.accounts.colType') }}</label>
              <input v-model="store.editType" type="text" class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium">{{ t('chatgpt.accounts.colStatus') }}</label>
              <select v-model="store.editStatus" class="input w-full">
                <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
                <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
                <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
                <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
              </select>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium">{{ t('chatgpt.accounts.colQuota') }}</label>
              <input v-model.number="store.editQuota" type="number" class="input w-full" />
            </div>
          </div>
          <div class="mt-4 flex justify-end gap-2">
            <button @click="store.closeEdit()" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button @click="store.saveEdit()" class="btn btn-primary">{{ t('common.save') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Export dialog -->
    <Teleport to="body">
      <div v-if="store.showExportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
          <h2 class="mb-4 text-lg font-semibold">{{ t('chatgpt.accounts.exportTitle') }}</h2>
          <select v-model="store.exportFormat" class="input mb-4 w-full">
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

    <!-- Delete confirm dialog -->
    <Teleport to="body">
      <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
          <h2 class="mb-4 text-lg font-semibold">{{ t('chatgpt.accounts.deleteConfirmTitle') }}</h2>
          <p class="mb-4 text-sm text-gray-500">
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
import { applyStoredConnection } from '@/api/chatgpt'
import ChatGPTConnectionSettings from './components/ChatGPTConnectionSettings.vue'

const { t } = useI18n()
const store = useChatGPTAccountsStore()

const showConnectionDialog = ref(false)
const showImportDialog = ref(false)
const showDeleteConfirm = ref(false)
const importText = ref('')

const statCards = computed(() => [
  { key: 'total', label: t('chatgpt.accounts.statsTotal'), color: 'text-gray-900 dark:text-white' },
  { key: '正常', label: t('chatgpt.accounts.statusNormal'), color: 'text-green-600' },
  { key: '限流', label: t('chatgpt.accounts.statusLimited'), color: 'text-yellow-600' },
  { key: '异常', label: t('chatgpt.accounts.statusError'), color: 'text-red-600' },
  { key: '禁用', label: t('chatgpt.accounts.statusDisabled'), color: 'text-gray-400' },
  { key: 'total', label: `${t('chatgpt.accounts.statsQuota')}: ${store.totalQuota}`, color: 'text-blue-600' },
])

onMounted(() => {
  if (!applyStoredConnection()) {
    showConnectionDialog.value = true
    return
  }
  store.load()
})

function onConnected(): void {
  showConnectionDialog.value = false
  store.load()
}

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

  // Try to parse as JSON first (CPA or session format)
  try {
    const parsed = JSON.parse(text)
    if (Array.isArray(parsed)) {
      await store.importAccounts([], parsed)
    } else {
      // Single object
      await store.importAccounts([], [parsed])
    }
  } catch {
    // Treat as plain token text (one per line)
    const tokens = text.split('\n').map((l) => l.trim()).filter(Boolean)
    await store.importAccounts(tokens, [])
  }

  importText.value = ''
  showImportDialog.value = false
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case '正常': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
    case '限流': return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
    case '异常': return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case '禁用': return 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400'
    default: return 'bg-gray-100 text-gray-500'
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
