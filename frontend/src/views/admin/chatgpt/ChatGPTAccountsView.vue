<template>
  <AppLayout>
    <div class="chatgpt-shell space-y-6 p-4 sm:p-6">
      <section class="hero-panel overflow-hidden rounded-[2rem] border border-slate-200/80 bg-white shadow-sm dark:border-slate-700/70 dark:bg-slate-950">
        <div class="hero-grid relative p-6 sm:p-8">
          <div class="relative z-10 flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div class="max-w-3xl">
              <div class="mb-4 inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-semibold uppercase tracking-[0.28em] text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-200">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_16px_rgba(16,185,129,0.9)]" />
                Native ChatGPT Pool
              </div>
              <h1 class="hero-title text-3xl font-black tracking-tight text-slate-950 dark:text-white sm:text-4xl">
                {{ t('chatgpt.accounts.title') }}
              </h1>
              <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
                {{ t('chatgpt.accounts.subtitle') }}
              </p>
            </div>

            <div class="flex flex-wrap items-center gap-3">
              <button
                @click="store.refreshAll()"
                :disabled="store.loading"
                class="control-button control-button-secondary"
              >
                <span :class="store.loading ? 'animate-spin' : ''">↻</span>
                {{ t('chatgpt.accounts.refreshAll') }}
              </button>
              <button @click="showImportDialog = true" class="control-button control-button-primary">
                <span>+</span>
                {{ t('chatgpt.accounts.import') }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <div v-if="store.error" class="rounded-2xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700 shadow-sm dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-200">
        <div class="flex items-start justify-between gap-4">
          <span>{{ store.error }}</span>
          <button @click="store.error = null" class="text-xs font-semibold uppercase tracking-wider underline-offset-4 hover:underline">Dismiss</button>
        </div>
      </div>

      <section class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <div
          v-for="stat in statCards"
          :key="stat.label"
          class="metric-card group rounded-3xl border border-slate-200 bg-white p-4 shadow-sm transition duration-300 hover:-translate-y-0.5 hover:shadow-md dark:border-slate-800 dark:bg-slate-900"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="text-[0.65rem] font-bold uppercase tracking-[0.22em] text-slate-400">{{ stat.label }}</div>
            <div class="h-2 w-2 rounded-full" :class="stat.dot" />
          </div>
          <div class="mt-4 text-3xl font-black tabular-nums" :class="stat.color">
            {{ stat.value }}
          </div>
        </div>
      </section>

      <section class="rounded-[1.75rem] border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-950">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-center">
          <div class="grid flex-1 gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
            <div class="relative">
              <input
                v-model="store.searchQuery"
                type="text"
                :placeholder="t('chatgpt.accounts.search')"
                class="native-input w-full pl-11"
              />
              <span class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-sm font-black text-slate-400">⌕</span>
            </div>
            <select v-model="store.filterStatus" class="native-input w-full">
              <option value="全部">{{ t('chatgpt.accounts.allStatus') }}</option>
              <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
              <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
              <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
              <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
            </select>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <span v-if="store.selectedCount > 0" class="rounded-full bg-slate-100 px-3 py-2 text-xs font-bold text-slate-600 dark:bg-slate-800 dark:text-slate-300">
              {{ t('chatgpt.accounts.selectedCount', { n: store.selectedCount }) }}
            </span>
            <button v-if="store.selectedCount > 0" @click="store.refreshSelected()" class="control-button control-button-secondary text-xs">
              {{ t('chatgpt.accounts.refreshSelected', { n: store.selectedCount }) }}
            </button>
            <button v-if="store.selectedCount > 0" @click="store.showExportDialog = true" class="control-button control-button-secondary text-xs">
              {{ t('chatgpt.accounts.exportSelected', { n: store.selectedCount }) }}
            </button>
            <button v-if="store.selectedCount > 0" @click="confirmDelete()" class="control-button control-button-danger text-xs">
              {{ t('chatgpt.accounts.deleteSelected', { n: store.selectedCount }) }}
            </button>
          </div>
        </div>
      </section>

      <section class="overflow-hidden rounded-[1.75rem] border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-950">
        <div class="border-b border-slate-200 bg-slate-50/70 px-5 py-4 dark:border-slate-800 dark:bg-slate-900/70">
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="text-sm font-black text-slate-900 dark:text-white">Account Runtime Matrix</div>
              <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ store.filteredAccounts.length }} visible / {{ store.accounts.length }} total</div>
            </div>
            <div class="hidden rounded-full border border-slate-200 px-3 py-1 text-xs font-bold text-slate-500 dark:border-slate-700 dark:text-slate-400 sm:block">
              sub2api native
            </div>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead class="bg-slate-950 text-[0.65rem] uppercase tracking-[0.22em] text-slate-300 dark:bg-black">
              <tr>
                <th class="px-4 py-4">
                  <input
                    type="checkbox"
                    class="native-checkbox"
                    :checked="store.selectedCount > 0 && store.selectedCount === store.filteredAccounts.length"
                    @change="toggleSelectAll"
                  />
                </th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colStatus') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colType') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colEmail') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colQuota') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colSuccess') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colLastUsed') }}</th>
                <th class="px-4 py-4 font-bold">{{ t('chatgpt.accounts.colActions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
              <tr
                v-for="acc in store.filteredAccounts"
                :key="acc.access_token"
                class="table-row transition hover:bg-emerald-50/40 dark:hover:bg-emerald-950/20"
              >
                <td class="px-4 py-4">
                  <input
                    type="checkbox"
                    class="native-checkbox"
                    :checked="store.isSelected(acc.access_token)"
                    @change="store.toggleSelect(acc.access_token)"
                  />
                </td>
                <td class="px-4 py-4">
                  <span :class="statusBadgeClass(acc.status)" class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-black">
                    {{ acc.status }}
                  </span>
                </td>
                <td class="px-4 py-4">
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-bold text-slate-600 dark:bg-slate-800 dark:text-slate-300">{{ acc.type || '-' }}</span>
                </td>
                <td class="px-4 py-4">
                  <div class="font-semibold text-slate-800 dark:text-slate-100">{{ acc.email || '-' }}</div>
                  <div class="mt-1 max-w-[16rem] truncate font-mono text-[0.68rem] text-slate-400">{{ acc.access_token }}</div>
                </td>
                <td class="px-4 py-4 font-black tabular-nums text-slate-700 dark:text-slate-200">
                  {{ acc.image_quota_unknown ? '?' : (acc.quota ?? 0) }}
                </td>
                <td class="px-4 py-4 text-slate-600 dark:text-slate-300">
                  <span class="font-black tabular-nums">{{ acc.success ?? 0 }}</span>
                  <span class="text-slate-400">/{{ (acc.success ?? 0) + (acc.fail ?? 0) }}</span>
                </td>
                <td class="px-4 py-4 text-slate-400 dark:text-slate-500">{{ formatDate(acc.last_used_at) }}</td>
                <td class="px-4 py-4">
                  <button @click="store.openEdit(acc)" class="rounded-full border border-slate-200 px-3 py-1.5 text-xs font-bold text-slate-700 transition hover:border-emerald-300 hover:bg-emerald-50 hover:text-emerald-700 dark:border-slate-700 dark:text-slate-300 dark:hover:border-emerald-500/40 dark:hover:bg-emerald-950/30 dark:hover:text-emerald-200">
                    {{ t('common.edit') }}
                  </button>
                </td>
              </tr>
              <tr v-if="store.filteredAccounts.length === 0 && !store.loading">
                <td colspan="8" class="px-4 py-16 text-center">
                  <div class="mx-auto max-w-sm rounded-3xl border border-dashed border-slate-300 p-8 dark:border-slate-700">
                    <div class="text-sm font-black text-slate-700 dark:text-slate-200">
                      {{ store.error ? t('chatgpt.accounts.errorLoading') : t('chatgpt.accounts.empty') }}
                    </div>
                    <div class="mt-2 text-xs text-slate-400">Import tokens or let the register machine create accounts.</div>
                  </div>
                </td>
              </tr>
              <tr v-if="store.loading">
                <td colspan="8" class="px-4 py-12 text-center text-sm font-semibold text-slate-400">
                  {{ t('common.loading') }}...
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <Teleport to="body">
      <div v-if="showImportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm">
        <div class="w-full max-w-2xl rounded-[2rem] border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-700 dark:bg-slate-950">
          <div class="mb-5">
            <h2 class="text-xl font-black text-slate-950 dark:text-white">{{ t('chatgpt.accounts.importTitle') }}</h2>
            <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Paste access tokens or structured JSON exported from an existing session.</p>
          </div>
          <textarea
            v-model="importText"
            rows="9"
            :placeholder="t('chatgpt.accounts.importPlaceholder')"
            class="native-textarea mb-5 w-full font-mono text-xs"
          />
          <div class="flex justify-end gap-2">
            <button @click="showImportDialog = false" class="control-button control-button-secondary">{{ t('common.cancel') }}</button>
            <button @click="handleImport" :disabled="!importText.trim()" class="control-button control-button-primary">{{ t('chatgpt.accounts.doImport') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="store.editingAccount" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm">
        <div class="w-full max-w-md rounded-[2rem] border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-700 dark:bg-slate-950">
          <h2 class="mb-5 text-xl font-black text-slate-950 dark:text-white">{{ t('chatgpt.accounts.editTitle') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="form-label">{{ t('chatgpt.accounts.colType') }}</label>
              <input v-model="store.editType" type="text" class="native-input w-full" />
            </div>
            <div>
              <label class="form-label">{{ t('chatgpt.accounts.colStatus') }}</label>
              <select v-model="store.editStatus" class="native-input w-full">
                <option value="正常">{{ t('chatgpt.accounts.statusNormal') }}</option>
                <option value="限流">{{ t('chatgpt.accounts.statusLimited') }}</option>
                <option value="异常">{{ t('chatgpt.accounts.statusError') }}</option>
                <option value="禁用">{{ t('chatgpt.accounts.statusDisabled') }}</option>
              </select>
            </div>
            <div>
              <label class="form-label">{{ t('chatgpt.accounts.colQuota') }}</label>
              <input v-model.number="store.editQuota" type="number" class="native-input w-full" />
            </div>
          </div>
          <div class="mt-6 flex justify-end gap-2">
            <button @click="store.closeEdit()" class="control-button control-button-secondary">{{ t('common.cancel') }}</button>
            <button @click="store.saveEdit()" class="control-button control-button-primary">{{ t('common.save') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="store.showExportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm">
        <div class="w-full max-w-sm rounded-[2rem] border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-700 dark:bg-slate-950">
          <h2 class="mb-5 text-xl font-black text-slate-950 dark:text-white">{{ t('chatgpt.accounts.exportTitle') }}</h2>
          <select v-model="store.exportFormat" class="native-input mb-5 w-full">
            <option value="json">JSON</option>
            <option value="zip">ZIP</option>
          </select>
          <div class="flex justify-end gap-2">
            <button @click="store.showExportDialog = false" class="control-button control-button-secondary">{{ t('common.cancel') }}</button>
            <button @click="store.downloadExport()" class="control-button control-button-primary">{{ t('chatgpt.accounts.doExport') }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm">
        <div class="w-full max-w-sm rounded-[2rem] border border-red-200 bg-white p-6 shadow-2xl dark:border-red-900/50 dark:bg-slate-950">
          <h2 class="mb-3 text-xl font-black text-slate-950 dark:text-white">{{ t('chatgpt.accounts.deleteConfirmTitle') }}</h2>
          <p class="mb-6 text-sm leading-6 text-slate-500 dark:text-slate-400">
            {{ t('chatgpt.accounts.deleteConfirmMsg', { n: store.selectedCount }) }}
          </p>
          <div class="flex justify-end gap-2">
            <button @click="showDeleteConfirm = false" class="control-button control-button-secondary">{{ t('common.cancel') }}</button>
            <button @click="handleDelete()" class="control-button control-button-danger">{{ t('common.confirm') }}</button>
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
  { key: 'total', label: t('chatgpt.accounts.statsTotal'), value: store.statusCounts.total ?? 0, color: 'text-slate-950 dark:text-white', dot: 'bg-slate-400' },
  { key: '正常', label: t('chatgpt.accounts.statusNormal'), value: store.statusCounts['正常'] ?? 0, color: 'text-emerald-600 dark:text-emerald-300', dot: 'bg-emerald-500' },
  { key: '限流', label: t('chatgpt.accounts.statusLimited'), value: store.statusCounts['限流'] ?? 0, color: 'text-amber-600 dark:text-amber-300', dot: 'bg-amber-500' },
  { key: '异常', label: t('chatgpt.accounts.statusError'), value: store.statusCounts['异常'] ?? 0, color: 'text-red-600 dark:text-red-300', dot: 'bg-red-500' },
  { key: '禁用', label: t('chatgpt.accounts.statusDisabled'), value: store.statusCounts['禁用'] ?? 0, color: 'text-slate-500 dark:text-slate-400', dot: 'bg-slate-500' },
  { key: 'quota', label: t('chatgpt.accounts.statsQuota'), value: store.totalQuota, color: 'text-cyan-600 dark:text-cyan-300', dot: 'bg-cyan-500' },
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
    case '正常': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-400/10 dark:text-emerald-200'
    case '限流': return 'bg-amber-100 text-amber-800 dark:bg-amber-400/10 dark:text-amber-200'
    case '异常': return 'bg-red-100 text-red-800 dark:bg-red-400/10 dark:text-red-200'
    case '禁用': return 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400'
    default: return 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300'
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
.chatgpt-shell {
  font-family: "Aptos", "Segoe UI Variable", "Helvetica Neue", sans-serif;
}
.hero-panel {
  background:
    radial-gradient(circle at 16% 10%, rgba(16, 185, 129, 0.16), transparent 34%),
    radial-gradient(circle at 84% 0%, rgba(6, 182, 212, 0.16), transparent 30%),
    linear-gradient(135deg, rgba(248, 250, 252, 0.98), rgba(255, 255, 255, 0.95));
}
:global(.dark) .hero-panel {
  background:
    radial-gradient(circle at 12% 0%, rgba(16, 185, 129, 0.22), transparent 34%),
    radial-gradient(circle at 92% 12%, rgba(6, 182, 212, 0.18), transparent 32%),
    linear-gradient(135deg, rgba(2, 6, 23, 1), rgba(15, 23, 42, 0.98));
}
.hero-grid::before {
  content: "";
  position: absolute;
  inset: 0;
  opacity: 0.35;
  background-image: linear-gradient(rgba(15, 23, 42, 0.08) 1px, transparent 1px), linear-gradient(90deg, rgba(15, 23, 42, 0.08) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: linear-gradient(90deg, black, transparent 80%);
}
.hero-title {
  letter-spacing: -0.045em;
}
.metric-card {
  background-image: linear-gradient(180deg, rgba(248, 250, 252, 0.9), rgba(255, 255, 255, 1));
}
:global(.dark) .metric-card {
  background-image: linear-gradient(180deg, rgba(15, 23, 42, 1), rgba(2, 6, 23, 1));
}
.control-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 999px;
  padding: 0.65rem 1rem;
  font-size: 0.875rem;
  font-weight: 800;
  transition: all 180ms ease;
}
.control-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.control-button-primary {
  background: linear-gradient(135deg, #10b981, #0891b2);
  color: white;
  box-shadow: 0 14px 34px rgba(16, 185, 129, 0.28);
}
.control-button-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 18px 42px rgba(16, 185, 129, 0.34);
}
.control-button-secondary {
  border: 1px solid rgba(148, 163, 184, 0.45);
  background: rgba(255, 255, 255, 0.82);
  color: #334155;
}
:global(.dark) .control-button-secondary {
  background: rgba(15, 23, 42, 0.86);
  color: #cbd5e1;
}
.control-button-danger {
  background: linear-gradient(135deg, #ef4444, #b91c1c);
  color: white;
}
.native-input,
.native-textarea {
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.38);
  background: rgba(248, 250, 252, 0.9);
  padding: 0.75rem 0.95rem;
  color: #0f172a;
  outline: none;
  transition: all 180ms ease;
}
.native-input:focus,
.native-textarea:focus {
  border-color: rgba(16, 185, 129, 0.6);
  box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.12);
}
:global(.dark) .native-input,
:global(.dark) .native-textarea {
  border-color: rgba(71, 85, 105, 0.9);
  background: rgba(15, 23, 42, 0.88);
  color: #e2e8f0;
}
.native-checkbox {
  height: 1rem;
  width: 1rem;
  accent-color: #10b981;
}
.form-label {
  margin-bottom: 0.4rem;
  display: block;
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #64748b;
}
.table-row:last-child {
  border-bottom: 0;
}
</style>
