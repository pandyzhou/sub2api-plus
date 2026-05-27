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
                    :aria-pressed="store.formMode === mode.value"
                    :class="store.formMode === mode.value
                      ? 'bg-primary-500 text-white shadow-sm shadow-primary-500/25 dark:bg-primary-500 dark:text-white'
                      : 'text-gray-600 hover:bg-white/70 dark:text-gray-400 dark:hover:bg-dark-800/70'"
                    @click="store.formMode = mode.value"
                  >
                    {{ mode.label }}
                  </button>
                </div>
                <p class="input-hint">{{ activeModeHint }}</p>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('chatgpt.register.fieldThreads') }}</label>
                  <input v-model.number="store.formThreads" type="number" min="1" max="50" class="input" />
                </div>
                <div v-if="store.formMode === 'total'">
                  <label class="input-label">{{ t('chatgpt.register.fieldTotal') }}</label>
                  <input v-model.number="store.formTotal" type="number" min="1" class="input" />
                  <p class="input-hint">总量模式会按注册总数推进进度。</p>
                </div>
                <div v-else-if="store.formMode === 'quota'">
                  <label class="input-label">{{ t('chatgpt.register.fieldTargetQuota') }}</label>
                  <input v-model.number="store.formTargetQuota" type="number" min="1" class="input" />
                  <p class="input-hint">额度模式会以目标额度作为达成条件。</p>
                </div>
                <div v-else>
                  <label class="input-label">{{ t('chatgpt.register.fieldTargetAvailable') }}</label>
                  <input v-model.number="store.formTargetAvailable" type="number" min="1" class="input" />
                  <p class="input-hint">可用模式会以目标可用账号数作为达成条件。</p>
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('chatgpt.register.fieldProxy') }}</label>
                <input v-model="store.formProxy" type="text" placeholder="http://user:pass@host:port" class="input font-mono text-sm" />
                <p class="input-hint">建议为 OpenAI 注册流程配置代理；可信网络内可留空。</p>
              </div>

              <div v-if="store.formMode !== 'total'">
                <label class="input-label">{{ t('chatgpt.register.fieldCheckInterval') }}</label>
                <input v-model.number="store.formCheckInterval" type="number" min="1" class="input" />
                <p class="input-hint">非总量模式会定期检查当前额度或可用数量。</p>
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header flex items-start justify-between gap-4">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">邮件接收配置</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  配置注册机接码超时和多个邮件 provider，按顺序选择启用的 provider 创建邮箱。
                </p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="store.addProvider()">添加 provider</button>
            </div>

            <div class="card-body space-y-5">
              <div class="grid gap-4 sm:grid-cols-3">
                <div>
                  <label class="input-label">请求超时（秒）</label>
                  <input v-model.number="store.formMail.request_timeout" type="number" min="1" class="input" />
                </div>
                <div>
                  <label class="input-label">等待验证码超时（秒）</label>
                  <input v-model.number="store.formMail.wait_timeout" type="number" min="1" class="input" />
                </div>
                <div>
                  <label class="input-label">轮询间隔（秒）</label>
                  <input v-model.number="store.formMail.wait_interval" type="number" min="1" class="input" />
                </div>
              </div>

              <div class="space-y-4">
                <div
                  v-for="(provider, index) in store.formMail.providers"
                  :key="index"
                  class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/60"
                >
                  <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div class="flex flex-wrap items-center gap-3">
                      <span class="text-sm font-semibold text-gray-900 dark:text-white">Provider #{{ index + 1 }}</span>
                      <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                        <input v-model="provider.enable" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800" />
                        启用
                      </label>
                    </div>
                    <button type="button" class="btn btn-danger btn-sm" @click="store.removeProvider(index)">删除</button>
                  </div>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <div>
                      <label class="input-label">类型</label>
                      <select v-model="provider.type" class="input">
                        <option v-for="item in providerTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
                      </select>
                    </div>
                    <div v-if="showField(provider.type, 'api_base')">
                      <label class="input-label">API Base</label>
                      <input v-model="provider.api_base" type="url" placeholder="https://api.mail.tm" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'api_key')">
                      <label class="input-label">API Key</label>
                      <input v-model="provider.api_key" type="password" autocomplete="new-password" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'admin_email')">
                      <label class="input-label">Admin Email</label>
                      <input v-model="provider.admin_email" type="email" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'admin_password')">
                      <label class="input-label">Admin Password</label>
                      <input v-model="provider.admin_password" type="password" autocomplete="new-password" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'ddg_token')">
                      <label class="input-label">DDG Token</label>
                      <input v-model="provider.ddg_token" type="password" autocomplete="new-password" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'cf_inbox_jwt')">
                      <label class="input-label">CF 收件箱 JWT</label>
                      <input v-model="provider.cf_inbox_jwt" type="password" autocomplete="new-password" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'cf_api_base')">
                      <label class="input-label">CF API Base</label>
                      <input v-model="provider.cf_api_base" type="url" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'cf_api_key')">
                      <label class="input-label">CF API Key</label>
                      <input v-model="provider.cf_api_key" type="password" autocomplete="new-password" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'cf_auth_mode')">
                      <label class="input-label">CF 认证模式</label>
                      <select v-model="provider.cf_auth_mode" class="input">
                        <option value="">默认</option>
                        <option value="jwt">jwt</option>
                        <option value="apikey">apikey</option>
                        <option value="x-api-key">x-api-key</option>
                        <option value="query-key">query-key</option>
                        <option value="none">none</option>
                      </select>
                    </div>
                    <div v-if="showField(provider.type, 'cf_messages_path')">
                      <label class="input-label">CF 消息路径</label>
                      <input v-model="provider.cf_messages_path" type="text" placeholder="/api/mails" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'email_prefix')">
                      <label class="input-label">邮箱前缀</label>
                      <input v-model="provider.email_prefix" type="text" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'default_domain')">
                      <label class="input-label">默认域名</label>
                      <input v-model="provider.default_domain" type="text" class="input font-mono text-sm" />
                    </div>
                    <div v-if="showField(provider.type, 'domain')" class="sm:col-span-2">
                      <label class="input-label">域名（每行一个）</label>
                      <textarea :value="listToText(provider.domain)" rows="3" class="input font-mono text-xs" @input="updateList(provider, 'domain', $event)" />
                    </div>
                    <div v-if="showField(provider.type, 'subdomain')" class="sm:col-span-2">
                      <label class="input-label">子域名（每行一个）</label>
                      <textarea :value="listToText(provider.subdomain)" rows="3" class="input font-mono text-xs" @input="updateList(provider, 'subdomain', $event)" />
                    </div>
                    <div class="flex flex-wrap gap-4 sm:col-span-2">
                      <label v-if="showField(provider.type, 'wildcard')" class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                        <input v-model="provider.wildcard" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800" />
                        通配符模式
                      </label>
                      <label v-if="showField(provider.type, 'random_subdomain')" class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                        <input v-model="provider.random_subdomain" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800" />
                        随机子域名
                      </label>
                    </div>
                  </div>
                </div>
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
import type { RegisterMailProvider, RegisterMode } from '@/api/chatgpt'

const { t } = useI18n()
const store = useChatGPTRegisterStore()

const modeOptions = computed<Array<{ value: RegisterMode; label: string }>>(() => [
  { value: 'total', label: t('chatgpt.register.modeTotal') },
  { value: 'quota', label: t('chatgpt.register.modeQuota') },
  { value: 'available', label: t('chatgpt.register.modeAvailable') },
])

const activeModeHint = computed(() => {
  switch (store.formMode) {
    case 'quota':
      return '当前为额度模式：只配置目标额度和检查间隔。'
    case 'available':
      return '当前为可用模式：只配置目标可用数和检查间隔。'
    default:
      return '当前为总量模式：只配置注册总数。'
  }
})

const providerTypeOptions = [
  { value: 'mailtm', label: 'mail.tm' },
  { value: 'custom', label: '自定义 mail.tm 兼容接口' },
  { value: 'cloudflare_temp_email', label: 'Cloudflare 临时邮箱' },
  { value: 'tempmail_lol', label: 'TempMail.lol' },
  { value: 'inbucket', label: 'Inbucket' },
  { value: 'moemail', label: 'MoEmail' },
  { value: 'cloudmail_gen', label: 'CloudMail Gen' },
  { value: 'ddg_mail', label: 'DDG 邮箱 + CF 中转' },
  { value: 'duckmail', label: 'DuckMail' },
  { value: 'gptmail', label: 'GPTMail' },
  { value: 'yyds_mail', label: 'YYDS Mail' },
]

const fieldsByProvider: Record<string, string[]> = {
  mailtm: ['api_base', 'api_key'],
  custom: ['api_base', 'api_key', 'domain'],
  cloudflare_temp_email: ['api_base', 'admin_password', 'domain'],
  tempmail_lol: ['api_base', 'api_key', 'domain'],
  inbucket: ['api_base', 'domain', 'random_subdomain'],
  moemail: ['api_base', 'api_key', 'domain'],
  cloudmail_gen: ['api_base', 'api_key', 'admin_email', 'admin_password', 'domain', 'subdomain', 'email_prefix'],
  ddg_mail: ['ddg_token', 'cf_inbox_jwt', 'cf_api_base', 'cf_api_key', 'cf_auth_mode', 'admin_password', 'cf_messages_path'],
  duckmail: ['api_key', 'default_domain'],
  gptmail: ['api_key', 'default_domain'],
  yyds_mail: ['api_base', 'api_key', 'domain', 'subdomain', 'wildcard'],
}

function showField(type: string, field: string): boolean {
  return (fieldsByProvider[type] || fieldsByProvider.custom).includes(field)
}

function listToText(value?: string[]): string {
  return Array.isArray(value) ? value.join('\n') : ''
}

function updateList(provider: RegisterMailProvider, field: 'domain' | 'subdomain', event: Event): void {
  provider[field] = (event.target as HTMLTextAreaElement).value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

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
