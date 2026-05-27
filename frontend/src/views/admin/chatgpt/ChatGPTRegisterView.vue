<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('chatgpt.register.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('chatgpt.register.subtitle') }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button @click="store.save()" :disabled="store.saving" class="btn btn-secondary">
            {{ store.saving ? t('common.saving') : t('chatgpt.register.saveConfig') }}
          </button>
          <button @click="store.toggle()" :class="store.isRunning ? 'btn-danger' : 'btn-primary'" class="btn">
            {{ store.isRunning ? t('chatgpt.register.stop') : t('chatgpt.register.start') }}
          </button>
          <button @click="store.reset()" class="btn btn-secondary">
            {{ t('chatgpt.register.resetStats') }}
          </button>
        </div>
      </div>

      <div v-if="store.error" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
        {{ store.error }}
      </div>

      <section v-if="store.stats" class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
        <div v-for="metric in metrics" :key="metric.label" class="card p-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg" :class="metric.iconBg">
              <span class="h-2.5 w-2.5 rounded-full" :class="metric.dot" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ metric.label }}
              </p>
              <p class="mt-1 text-xl font-bold tabular-nums" :class="metric.color">
                {{ metric.value }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-5">
        <div class="mb-2 flex items-center justify-between text-xs font-medium text-gray-500 dark:text-gray-400">
          <span>{{ t('chatgpt.register.progress') }}</span>
          <span>{{ store.progress ?? 0 }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
          <div
            class="h-full rounded-full bg-primary-500 transition-all duration-500"
            :style="{ width: (store.progress ?? 0) + '%' }"
          />
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.9fr)]">
        <div class="space-y-6">
          <section class="card">
            <div class="card-header">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('chatgpt.register.configTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                配置注册目标、并发和网络代理。
              </p>
            </div>

            <div class="card-body space-y-5">
              <div>
                <label class="input-label">{{ t('chatgpt.register.fieldMode') }}</label>
                <div class="grid grid-cols-3 gap-2 rounded-xl border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-900">
                  <button
                    v-for="mode in modeOptions"
                    :key="mode.value"
                    type="button"
                    class="rounded-lg px-3 py-2 text-sm font-medium transition-colors"
                    :class="store.formMode === mode.value
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-600 hover:bg-white/70 dark:text-gray-400 dark:hover:bg-dark-800/70'"
                    @click="store.formMode = mode.value"
                  >
                    {{ mode.label }}
                  </button>
                </div>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldTotal') }}</label>
                  <input v-model.number="store.formTotal" type="number" min="1" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldThreads') }}</label>
                  <input v-model.number="store.formThreads" type="number" min="1" max="50" class="input" />
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('chatgpt.register.fieldProxy') }}</label>
                <input v-model="store.formProxy" type="text" placeholder="http://user:pass@host:port" class="input font-mono text-sm" />
                <p class="input-hint">建议为 OpenAI 注册流程配置代理；可信网络内可留空。</p>
              </div>

              <div class="grid gap-4 sm:grid-cols-3">
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldTargetQuota') }}</label>
                  <input v-model.number="store.formTargetQuota" type="number" min="1" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldTargetAvailable') }}</label>
                  <input v-model.number="store.formTargetAvailable" type="number" min="1" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldCheckInterval') }}</label>
                  <input v-model.number="store.formCheckInterval" type="number" min="1" class="input" />
                </div>
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">邮件接收配置</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                仅用于 ChatGPT 注册机接收 OpenAI 验证码，不影响 Sub2API 账号注册邮件。
              </p>
            </div>

            <div class="card-body space-y-4">
              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">邮件服务商</label>
                  <select v-model="store.formMailProvider" class="input">
                    <option value="mailtm">mail.tm</option>
                    <option value="custom">自定义 mail.tm 兼容接口</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">API Base</label>
                  <input v-model="store.formMailAPIBase" type="url" placeholder="https://api.mail.tm" class="input font-mono text-sm" />
                </div>
              </div>
              <div>
                <label class="input-label">API Key</label>
                <input
                  v-model="store.formMailAPIKey"
                  type="password"
                  autocomplete="new-password"
                  placeholder="可选，自建接码服务使用 Bearer Token"
                  class="input font-mono text-sm"
                />
              </div>
            </div>
          </section>
        </div>

        <section class="card flex min-h-[30rem] flex-col overflow-hidden">
          <div class="card-header flex items-center justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('chatgpt.register.logsTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">注册任务运行日志</p>
            </div>
            <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              {{ store.recentLogs.length }} lines
            </span>
          </div>

          <div class="flex-1 overflow-y-auto bg-gray-50 p-4 font-mono text-xs dark:bg-dark-900/70">
            <div v-if="store.recentLogs.length === 0" class="flex h-full items-center justify-center rounded-xl border border-dashed border-gray-300 text-gray-500 dark:border-dark-700 dark:text-gray-400">
              {{ t('chatgpt.register.noLogs') }}
            </div>
            <div
              v-for="(log, idx) in store.recentLogs"
              :key="idx"
              class="grid gap-3 rounded-lg px-3 py-2 transition hover:bg-white dark:hover:bg-dark-800 sm:grid-cols-[8.5rem_minmax(0,1fr)]"
              :class="logLineClass(log.level)"
            >
              <span class="text-gray-400 dark:text-dark-400">{{ formatLogTime(log.time) }}</span>
              <span class="break-words">{{ log.text }}</span>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useChatGPTRegisterStore } from '@/stores/chatgpt'
import type { RegisterMode } from '@/api/chatgpt'

const { t } = useI18n()
const store = useChatGPTRegisterStore()

const modeOptions = computed<Array<{ value: RegisterMode; label: string }>>(() => [
  { value: 'total', label: t('chatgpt.register.modeTotal') },
  { value: 'quota', label: t('chatgpt.register.modeQuota') },
  { value: 'available', label: t('chatgpt.register.modeAvailable') },
])

const metrics = computed(() => [
  { label: t('chatgpt.register.statSuccess'), value: store.stats?.success ?? 0, color: 'text-emerald-600 dark:text-emerald-400', dot: 'bg-emerald-500', iconBg: 'bg-emerald-100 dark:bg-emerald-900/30' },
  { label: t('chatgpt.register.statFail'), value: store.stats?.fail ?? 0, color: 'text-red-600 dark:text-red-400', dot: 'bg-red-500', iconBg: 'bg-red-100 dark:bg-red-900/30' },
  { label: t('chatgpt.register.statDone'), value: store.stats?.done ?? 0, color: 'text-cyan-600 dark:text-cyan-400', dot: 'bg-cyan-500', iconBg: 'bg-cyan-100 dark:bg-cyan-900/30' },
  { label: t('chatgpt.register.statRunning'), value: store.stats?.running ?? 0, color: 'text-gray-900 dark:text-white', dot: store.isRunning ? 'bg-emerald-500' : 'bg-gray-400', iconBg: 'bg-gray-100 dark:bg-dark-700' },
  { label: t('chatgpt.register.statThreads'), value: store.stats?.threads ?? 0, color: 'text-violet-600 dark:text-violet-400', dot: 'bg-violet-500', iconBg: 'bg-violet-100 dark:bg-violet-900/30' },
  { label: t('chatgpt.register.statStatus'), value: store.isRunning ? 'ON' : 'OFF', color: store.isRunning ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-500 dark:text-gray-400', dot: store.isRunning ? 'bg-emerald-500' : 'bg-gray-400', iconBg: store.isRunning ? 'bg-emerald-100 dark:bg-emerald-900/30' : 'bg-gray-100 dark:bg-dark-700' },
])

onMounted(() => {
  store.load()
  store.startSSE()
})

onUnmounted(() => {
  store.stopSSE()
})

function formatLogTime(timeStr?: string): string {
  if (!timeStr) return ''
  try {
    const d = new Date(timeStr)
    return d.toLocaleTimeString()
  } catch {
    return timeStr
  }
}

function logLineClass(level?: string): string {
  switch (level) {
    case 'error':
      return 'text-red-700 dark:text-red-300'
    case 'warning':
      return 'text-amber-700 dark:text-amber-300'
    default:
      return 'text-gray-700 dark:text-gray-300'
  }
}
</script>
