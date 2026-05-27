<template>
  <AppLayout>
    <div class="register-shell space-y-6 p-4 sm:p-6">
      <section class="register-hero overflow-hidden rounded-[2rem] border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-950">
        <div class="relative p-6 sm:p-8">
          <div class="relative z-10 flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
            <div class="max-w-3xl">
              <div class="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-200 bg-cyan-50 px-3 py-1 text-xs font-black uppercase tracking-[0.28em] text-cyan-700 dark:border-cyan-400/30 dark:bg-cyan-400/10 dark:text-cyan-200">
                <span class="h-1.5 w-8 rounded-full bg-gradient-to-r from-cyan-500 to-emerald-400" />
                Native Orchestrator
              </div>
              <h1 class="hero-title text-3xl font-black tracking-tight text-slate-950 dark:text-white sm:text-4xl">
                {{ t('chatgpt.register.title') }}
              </h1>
              <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
                {{ t('chatgpt.register.subtitle') }}
              </p>
            </div>

            <div class="flex flex-wrap items-center gap-3">
              <button @click="store.save()" :disabled="store.saving" class="action-button action-button-secondary">
                {{ store.saving ? t('common.saving') : t('chatgpt.register.saveConfig') }}
              </button>
              <button @click="store.toggle()" :class="store.isRunning ? 'action-button-stop' : 'action-button-start'" class="action-button">
                {{ store.isRunning ? t('chatgpt.register.stop') : t('chatgpt.register.start') }}
              </button>
              <button @click="store.reset()" class="action-button action-button-secondary">
                {{ t('chatgpt.register.resetStats') }}
              </button>
            </div>
          </div>

          <div class="relative z-10 mt-8 grid gap-3 sm:grid-cols-3">
            <div class="hero-chip">
              <span class="hero-chip-label">Mode</span>
              <span class="hero-chip-value">{{ store.formMode }}</span>
            </div>
            <div class="hero-chip">
              <span class="hero-chip-label">Threads</span>
              <span class="hero-chip-value">{{ store.formThreads }}</span>
            </div>
            <div class="hero-chip">
              <span class="hero-chip-label">Runtime</span>
              <span class="hero-chip-value" :class="store.isRunning ? 'text-emerald-600 dark:text-emerald-300' : 'text-slate-500'">
                {{ store.isRunning ? 'RUNNING' : 'IDLE' }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <div v-if="store.error" class="rounded-2xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700 shadow-sm dark:border-red-900/40 dark:bg-red-950/40 dark:text-red-200">
        {{ store.error }}
      </div>

      <section v-if="store.stats" class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <div v-for="metric in metrics" :key="metric.label" class="telemetry-card rounded-3xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900">
          <div class="flex items-start justify-between gap-3">
            <div class="text-[0.65rem] font-black uppercase tracking-[0.22em] text-slate-400">{{ metric.label }}</div>
            <div class="h-2.5 w-2.5 rounded-full" :class="metric.dot" />
          </div>
          <div class="mt-4 text-3xl font-black tabular-nums" :class="metric.color">{{ metric.value }}</div>
        </div>
      </section>

      <section class="rounded-[1.75rem] border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-950">
        <div class="mb-2 flex items-center justify-between text-xs font-black uppercase tracking-[0.2em] text-slate-400">
          <span>{{ t('chatgpt.register.progress') }}</span>
          <span>{{ store.progress ?? 0 }}%</span>
        </div>
        <div class="progress-rail h-4 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-900">
          <div class="progress-fill h-full rounded-full transition-all duration-500" :style="{ width: (store.progress ?? 0) + '%' }" />
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[minmax(0,0.92fr)_minmax(0,1.08fr)]">
        <section class="rounded-[1.75rem] border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-950">
          <div class="border-b border-slate-200 px-6 py-5 dark:border-slate-800">
            <div class="text-lg font-black text-slate-950 dark:text-white">{{ t('chatgpt.register.configTitle') }}</div>
            <div class="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-400">Configure the target strategy before launching the native account factory.</div>
          </div>

          <div class="space-y-5 p-6">
            <div>
              <label class="form-label">{{ t('chatgpt.register.fieldMode') }}</label>
              <div class="grid grid-cols-3 gap-2 rounded-2xl bg-slate-100 p-1 dark:bg-slate-900">
                <button
                  v-for="mode in modeOptions"
                  :key="mode.value"
                  type="button"
                  class="mode-button"
                  :class="store.formMode === mode.value ? 'mode-button-active' : ''"
                  @click="store.formMode = mode.value"
                >
                  {{ mode.label }}
                </button>
              </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label class="form-label">{{ t('chatgpt.register.fieldTotal') }}</label>
                <input v-model.number="store.formTotal" type="number" min="1" class="native-input w-full" />
              </div>
              <div>
                <label class="form-label">{{ t('chatgpt.register.fieldThreads') }}</label>
                <input v-model.number="store.formThreads" type="number" min="1" max="50" class="native-input w-full" />
              </div>
            </div>

            <div>
              <label class="form-label">{{ t('chatgpt.register.fieldProxy') }}</label>
              <input v-model="store.formProxy" type="text" placeholder="http://user:pass@host:port" class="native-input w-full font-mono text-sm" />
              <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-slate-400">Strongly recommended for OpenAI signup flows. Leave empty only for trusted direct networks.</p>
            </div>

            <div class="grid gap-4 sm:grid-cols-3">
              <div>
                <label class="form-label">{{ t('chatgpt.register.fieldTargetQuota') }}</label>
                <input v-model.number="store.formTargetQuota" type="number" min="1" class="native-input w-full" />
              </div>
              <div>
                <label class="form-label">{{ t('chatgpt.register.fieldTargetAvailable') }}</label>
                <input v-model.number="store.formTargetAvailable" type="number" min="1" class="native-input w-full" />
              </div>
              <div>
                <label class="form-label">{{ t('chatgpt.register.fieldCheckInterval') }}</label>
                <input v-model.number="store.formCheckInterval" type="number" min="1" class="native-input w-full" />
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-[1.75rem] border border-slate-200 bg-slate-950 shadow-sm dark:border-slate-800">
          <div class="flex items-center justify-between border-b border-white/10 px-6 py-5">
            <div>
              <div class="text-lg font-black text-white">{{ t('chatgpt.register.logsTitle') }}</div>
              <div class="mt-1 text-xs text-slate-400">Live task journal from the embedded registrar</div>
            </div>
            <div class="rounded-full border border-white/10 px-3 py-1 text-xs font-black uppercase tracking-[0.22em] text-slate-400">
              {{ store.recentLogs.length }} lines
            </div>
          </div>

          <div class="terminal-window h-[33rem] overflow-y-auto p-4 font-mono text-xs">
            <div v-if="store.recentLogs.length === 0" class="empty-terminal flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-700 text-slate-500">
              {{ t('chatgpt.register.noLogs') }}
            </div>
            <div
              v-for="(log, idx) in store.recentLogs"
              :key="idx"
              class="terminal-line grid gap-3 rounded-xl px-3 py-2 sm:grid-cols-[8.5rem_minmax(0,1fr)]"
              :class="{
                'terminal-error': log.level === 'error',
                'terminal-warning': log.level === 'warning',
                'terminal-info': log.level === 'info',
              }"
            >
              <span class="text-slate-500">{{ formatLogTime(log.time) }}</span>
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
  { label: t('chatgpt.register.statSuccess'), value: store.stats?.success ?? 0, color: 'text-emerald-600 dark:text-emerald-300', dot: 'bg-emerald-500' },
  { label: t('chatgpt.register.statFail'), value: store.stats?.fail ?? 0, color: 'text-red-600 dark:text-red-300', dot: 'bg-red-500' },
  { label: t('chatgpt.register.statDone'), value: store.stats?.done ?? 0, color: 'text-cyan-600 dark:text-cyan-300', dot: 'bg-cyan-500' },
  { label: t('chatgpt.register.statRunning'), value: store.stats?.running ?? 0, color: 'text-slate-800 dark:text-white', dot: store.isRunning ? 'bg-emerald-500' : 'bg-slate-500' },
  { label: t('chatgpt.register.statThreads'), value: store.stats?.threads ?? 0, color: 'text-slate-800 dark:text-white', dot: 'bg-violet-500' },
  { label: t('chatgpt.register.statStatus'), value: store.isRunning ? 'ON' : 'OFF', color: store.isRunning ? 'text-emerald-600 dark:text-emerald-300' : 'text-slate-500', dot: store.isRunning ? 'bg-emerald-500' : 'bg-slate-500' },
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
</script>

<style scoped>
.register-shell {
  font-family: "Aptos", "Segoe UI Variable", "Helvetica Neue", sans-serif;
}
.register-hero {
  background:
    radial-gradient(circle at 10% 4%, rgba(6, 182, 212, 0.18), transparent 34%),
    radial-gradient(circle at 92% 14%, rgba(16, 185, 129, 0.16), transparent 32%),
    linear-gradient(135deg, #ffffff, #f8fafc);
}
:global(.dark) .register-hero {
  background:
    radial-gradient(circle at 8% 0%, rgba(6, 182, 212, 0.22), transparent 34%),
    radial-gradient(circle at 92% 10%, rgba(16, 185, 129, 0.2), transparent 32%),
    linear-gradient(135deg, #020617, #0f172a);
}
.register-hero::before {
  content: "";
  position: absolute;
  inset: 0;
  opacity: 0.32;
  background-image: repeating-linear-gradient(120deg, rgba(15, 23, 42, 0.11) 0, rgba(15, 23, 42, 0.11) 1px, transparent 1px, transparent 18px);
  mask-image: linear-gradient(90deg, black, transparent 72%);
}
.hero-title {
  letter-spacing: -0.045em;
}
.hero-chip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(255, 255, 255, 0.7);
  padding: 0.9rem 1rem;
  backdrop-filter: blur(16px);
}
:global(.dark) .hero-chip {
  background: rgba(15, 23, 42, 0.72);
}
.hero-chip-label {
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #64748b;
}
.hero-chip-value {
  font-size: 0.9rem;
  font-weight: 950;
  text-transform: uppercase;
  color: #0f172a;
}
:global(.dark) .hero-chip-value {
  color: #f8fafc;
}
.telemetry-card {
  background-image: linear-gradient(180deg, rgba(248, 250, 252, 0.9), #fff);
}
:global(.dark) .telemetry-card {
  background-image: linear-gradient(180deg, rgba(15, 23, 42, 1), rgba(2, 6, 23, 1));
}
.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0.72rem 1.05rem;
  font-size: 0.875rem;
  font-weight: 900;
  transition: all 180ms ease;
}
.action-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.action-button-secondary {
  border: 1px solid rgba(148, 163, 184, 0.45);
  background: rgba(255, 255, 255, 0.82);
  color: #334155;
}
:global(.dark) .action-button-secondary {
  background: rgba(15, 23, 42, 0.86);
  color: #cbd5e1;
}
.action-button-start {
  background: linear-gradient(135deg, #06b6d4, #10b981);
  color: white;
  box-shadow: 0 16px 38px rgba(6, 182, 212, 0.28);
}
.action-button-stop {
  background: linear-gradient(135deg, #ef4444, #b91c1c);
  color: white;
  box-shadow: 0 16px 38px rgba(239, 68, 68, 0.24);
}
.action-button:hover:not(:disabled) {
  transform: translateY(-1px);
}
.progress-rail {
  box-shadow: inset 0 1px 2px rgba(15, 23, 42, 0.12);
}
.progress-fill {
  background: linear-gradient(90deg, #06b6d4, #10b981, #a7f3d0);
  box-shadow: 0 0 30px rgba(16, 185, 129, 0.42);
}
.form-label {
  margin-bottom: 0.45rem;
  display: block;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #64748b;
}
.native-input {
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.38);
  background: rgba(248, 250, 252, 0.9);
  padding: 0.78rem 0.95rem;
  color: #0f172a;
  outline: none;
  transition: all 180ms ease;
}
.native-input:focus {
  border-color: rgba(6, 182, 212, 0.62);
  box-shadow: 0 0 0 4px rgba(6, 182, 212, 0.12);
}
:global(.dark) .native-input {
  border-color: rgba(71, 85, 105, 0.9);
  background: rgba(15, 23, 42, 0.88);
  color: #e2e8f0;
}
.mode-button {
  border-radius: 0.9rem;
  padding: 0.72rem 0.65rem;
  font-size: 0.78rem;
  font-weight: 900;
  color: #64748b;
  transition: all 180ms ease;
}
.mode-button-active {
  background: #ffffff;
  color: #0891b2;
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
}
:global(.dark) .mode-button-active {
  background: #0f172a;
  color: #67e8f9;
}
.terminal-window {
  background:
    linear-gradient(rgba(15, 23, 42, 0.78), rgba(2, 6, 23, 0.96)),
    repeating-linear-gradient(0deg, rgba(255, 255, 255, 0.04) 0, rgba(255, 255, 255, 0.04) 1px, transparent 1px, transparent 24px);
}
.terminal-line {
  color: #cbd5e1;
}
.terminal-line:hover {
  background: rgba(255, 255, 255, 0.045);
}
.terminal-error {
  color: #fca5a5;
}
.terminal-warning {
  color: #fcd34d;
}
.terminal-info {
  color: #cbd5e1;
}
.empty-terminal {
  background: rgba(15, 23, 42, 0.55);
}
</style>
