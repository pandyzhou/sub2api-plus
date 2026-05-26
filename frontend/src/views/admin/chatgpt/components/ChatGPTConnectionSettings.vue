<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
    <div class="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
      <h2 class="mb-4 text-xl font-semibold text-gray-900 dark:text-white">
        {{ t('chatgpt.connection.title') }}
      </h2>
      <p class="mb-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('chatgpt.connection.description') }}
      </p>

      <div class="space-y-4">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('chatgpt.connection.baseURL') }}
          </label>
          <input
            v-model="baseURL"
            type="text"
            placeholder="http://127.0.0.1:20002"
            class="input w-full"
          />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('chatgpt.connection.authKey') }}
          </label>
          <input
            v-model="authKey"
            type="password"
            placeholder="Admin API Key"
            class="input w-full"
          />
        </div>

        <div v-if="testResult !== null" class="rounded-lg p-3 text-sm" :class="testResult.ok ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'">
          {{ testResult.ok ? `${t('chatgpt.connection.connected')} v${testResult.version}` : testResult.error }}
        </div>
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <button
          v-if="!forceSetup"
          @click="emit('cancel')"
          class="btn btn-secondary"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          @click="handleTest"
          :disabled="testing"
          class="btn btn-secondary"
        >
          {{ testing ? t('chatgpt.connection.testing') : t('chatgpt.connection.test') }}
        </button>
        <button
          @click="handleSave"
          :disabled="!baseURL || !authKey"
          class="btn btn-primary"
        >
          {{ t('chatgpt.connection.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  saveConnectionConfig,
  testConnection as testChatGPTConnection,
  applyStoredConnection,
} from '@/api/chatgpt'

const { t } = useI18n()

const { show, forceSetup } = defineProps<{
  show: boolean
  forceSetup?: boolean
}>()

const emit = defineEmits<{
  (e: 'connected'): void
  (e: 'cancel'): void
}>()

const baseURL = ref('http://127.0.0.1:20002')
const authKey = ref('')
const testing = ref(false)
const testResult = ref<{ ok: boolean; version?: string; error?: string } | null>(null)

// Load previously stored config on mount
const stored = applyStoredConnection()
if (stored) {
  baseURL.value = import.meta.env.VITE_CHATGPT_API_URL || 'http://127.0.0.1:20002'
  try {
    const raw = localStorage.getItem('chatgpt2api_connection')
    if (raw) {
      const parsed = JSON.parse(raw)
      baseURL.value = parsed.baseURL || baseURL.value
      authKey.value = parsed.authKey || ''
    }
  } catch {
    // ignore
  }
}

async function handleTest(): Promise<void> {
  testing.value = true
  testResult.value = null

  // Temporarily apply config for test
  const tempConfig = { baseURL: baseURL.value, authKey: authKey.value }
  saveConnectionConfig(tempConfig)
  testResult.value = await testChatGPTConnection()
  testing.value = false
}

function handleSave(): void {
  saveConnectionConfig({ baseURL: baseURL.value, authKey: authKey.value })
  emit('connected')
}
</script>
