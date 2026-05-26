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
            {{ t('chatgpt.register.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('chatgpt.register.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="store.save()"
            :disabled="store.saving"
            class="btn btn-secondary"
          >
            {{ store.saving ? t('common.saving') : t('chatgpt.register.saveConfig') }}
          </button>
          <button
            @click="store.toggle()"
            :class="store.isRunning ? 'btn-danger' : 'btn-primary'"
            class="btn"
          >
            {{ store.isRunning ? t('chatgpt.register.stop') : t('chatgpt.register.start') }}
          </button>
          <button
            @click="store.reset()"
            class="btn btn-secondary"
          >
            {{ t('chatgpt.register.resetStats') }}
          </button>
        </div>
      </div>

      <!-- Error -->
      <div v-if="store.error" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/30 dark:text-red-300">
        {{ store.error }}
      </div>

      <!-- Stats -->
      <div v-if="store.stats" class="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold text-green-600">{{ store.stats.success }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statSuccess') }}</div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold text-red-600">{{ store.stats.fail }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statFail') }}</div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold text-blue-600">{{ store.stats.done }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statDone') }}</div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold text-gray-700 dark:text-gray-300">{{ store.stats.running }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statRunning') }}</div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold text-gray-700 dark:text-gray-300">{{ store.stats.threads }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statThreads') }}</div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <div class="text-2xl font-bold" :class="store.isRunning ? 'text-green-600' : 'text-gray-400'">
            {{ store.isRunning ? '&#9679;' : '&#9675;' }}
          </div>
          <div class="mt-1 text-xs text-gray-500">{{ t('chatgpt.register.statStatus') }}</div>
        </div>
      </div>

      <!-- Progress bar -->
      <div v-if="store.progress !== null && store.isRunning" class="space-y-1">
        <div class="flex justify-between text-xs text-gray-500">
          <span>{{ t('chatgpt.register.progress') }}</span>
          <span>{{ store.progress }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
          <div
            class="h-full rounded-full bg-blue-500 transition-all"
            :style="{ width: store.progress + '%' }"
          />
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <!-- Config form -->
        <div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
          <h3 class="mb-4 text-lg font-semibold">{{ t('chatgpt.register.configTitle') }}</h3>
          <div class="space-y-4">
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldMode') }}</label>
              <select v-model="store.formMode" class="input w-full">
                <option value="total">{{ t('chatgpt.register.modeTotal') }}</option>
                <option value="quota">{{ t('chatgpt.register.modeQuota') }}</option>
                <option value="available">{{ t('chatgpt.register.modeAvailable') }}</option>
              </select>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldTotal') }}</label>
              <input v-model.number="store.formTotal" type="number" min="1" class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldThreads') }}</label>
              <input v-model.number="store.formThreads" type="number" min="1" max="50" class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldProxy') }}</label>
              <input v-model="store.formProxy" type="text" placeholder="http://..." class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldTargetQuota') }}</label>
              <input v-model.number="store.formTargetQuota" type="number" min="1" class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldTargetAvailable') }}</label>
              <input v-model.number="store.formTargetAvailable" type="number" min="1" class="input w-full" />
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium">{{ t('chatgpt.register.fieldCheckInterval') }}</label>
              <input v-model.number="store.formCheckInterval" type="number" min="1" class="input w-full" />
            </div>
          </div>
        </div>

        <!-- Logs -->
        <div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
          <h3 class="mb-4 text-lg font-semibold">{{ t('chatgpt.register.logsTitle') }}</h3>
          <div class="h-96 overflow-y-auto rounded-lg border border-gray-100 bg-gray-50 p-3 font-mono text-xs dark:border-gray-600 dark:bg-gray-900">
            <div v-if="store.recentLogs.length === 0" class="text-gray-400">
              {{ t('chatgpt.register.noLogs') }}
            </div>
            <div
              v-for="(log, idx) in store.recentLogs"
              :key="idx"
              class="mb-1"
              :class="{
                'text-red-600 dark:text-red-400': log.level === 'error',
                'text-yellow-600 dark:text-yellow-400': log.level === 'warning',
                'text-gray-600 dark:text-gray-400': log.level === 'info',
              }"
            >
              <span class="text-gray-400">{{ formatLogTime(log.time) }}</span>
              {{ log.text }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useChatGPTRegisterStore } from '@/stores/chatgpt'
import ChatGPTConnectionSettings from './components/ChatGPTConnectionSettings.vue'

const { t } = useI18n()
const store = useChatGPTRegisterStore()

const showConnectionDialog = ref(false)

onMounted(() => {
  store.load()
  store.startSSE()
})

onUnmounted(() => {
  store.stopSSE()
})

function onConnected(): void {
  showConnectionDialog.value = false
  store.load()
  store.startSSE()
}

function formatLogTime(timeStr?: string): string {
  if (!timeStr) return ''
  try {
    const d = new Date(timeStr)
    return d.toLocaleTimeString() + ' '
  } catch {
    return timeStr + ' '
  }
}
</script>
