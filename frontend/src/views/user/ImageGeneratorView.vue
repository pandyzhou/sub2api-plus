<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-4rem)] overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <!-- ==================== Left: Session Sidebar ==================== -->
      <aside
        :class="[
          'flex flex-col border-r border-gray-200 bg-gray-50/80 dark:border-dark-700 dark:bg-dark-800/60',
          'w-64 shrink-0',
          mobileShowSessions ? 'absolute inset-0 z-20 w-full sm:relative sm:w-64' : 'hidden sm:flex'
        ]"
      >
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('image.title') }}</h2>
          <button
            class="sm:hidden rounded-md p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            @click="mobileShowSessions = false"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <!-- New Session -->
        <div class="p-3">
          <button class="btn btn-primary btn-sm w-full" @click="handleNewSession">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
            {{ t('image.newSession') }}
          </button>
        </div>

        <!-- Session List -->
        <div class="flex-1 overflow-y-auto scrollbar-hide px-2">
          <div v-if="imageStore.loading" class="flex justify-center py-8"><LoadingSpinner /></div>
          <div v-else-if="sessions.length === 0" class="py-8 text-center text-xs text-gray-400 dark:text-gray-500">
            {{ t('image.noSessions') }}
          </div>
          <div v-else class="space-y-0.5">
            <div
              v-for="session in sessions"
              :key="session.id"
              :class="[
                'group relative flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 transition-colors',
                currentSessionId === session.id
                  ? 'bg-primary-50 dark:bg-primary-900/20'
                  : 'hover:bg-gray-100 dark:hover:bg-dark-700'
              ]"
              @click="handleSelectSession(session)"
            >
              <div class="min-w-0 flex-1">
                <p
                  :class="[
                    'truncate text-sm font-medium',
                    currentSessionId === session.id
                      ? 'text-primary-700 dark:text-primary-300'
                      : 'text-gray-800 dark:text-gray-200'
                  ]"
                >{{ session.title || t('image.newSession') }}</p>
                <p class="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500">
                  {{ formatTime(session.updated_at || session.created_at) }}
                </p>
              </div>
              <button
                class="shrink-0 rounded p-1 text-gray-300 opacity-0 transition-opacity hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:text-gray-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('image.deleteSession')"
                @click.stop="handleDeleteSession(session.id)"
              >
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Clear -->
        <div class="border-t border-gray-200 p-3 dark:border-dark-700">
          <button
            v-if="sessions.length > 0"
            class="btn btn-ghost btn-sm w-full text-xs text-gray-400 hover:text-red-500"
            @click="handleClearHistory"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
            {{ t('image.clearHistory') }}
          </button>
        </div>
      </aside>

      <!-- ==================== Right: Main Area ==================== -->
      <div class="flex flex-1 flex-col overflow-hidden">
        <!-- Top Bar -->
        <div class="flex items-center gap-2 border-b border-gray-200 bg-white px-4 py-2 dark:border-dark-700 dark:bg-dark-900">
          <!-- Mobile toggle -->
          <button
            class="sm:hidden rounded-md p-1.5 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
            @click="mobileShowSessions = true"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
          </button>

          <!-- Mode tabs -->
          <div class="flex rounded-lg bg-gray-100 p-0.5 dark:bg-dark-800">
            <button
              :class="[
                'rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                !imageStore.editMode
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
              ]"
              @click="imageStore.editMode = false"
            >{{ t('image.textToImage') }}</button>
            <button
              :class="[
                'rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                imageStore.editMode
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
              ]"
              @click="imageStore.editMode = true"
            >{{ t('image.imageEdit') }}</button>
          </div>

          <div class="flex-1" />

          <!-- Settings toggle -->
          <button
            class="btn btn-ghost btn-sm"
            :class="showSettings && 'bg-gray-100 dark:bg-dark-700'"
            @click="showSettings = !showSettings"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
          </button>

          <button
            v-if="results.length > 1"
            class="btn btn-ghost btn-sm"
            @click="downloadAll"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
            {{ t('image.downloadAll') }}
          </button>
        </div>

        <!-- Settings Panel -->
        <Transition name="slide-down">
          <div v-if="showSettings" class="border-b border-gray-100 bg-gray-50/50 px-4 py-3 dark:border-dark-800 dark:bg-dark-800/40">
            <div class="flex flex-wrap items-end gap-4">
              <div v-for="field in settingsFields" :key="field.key" class="flex flex-col gap-1">
                <label class="input-label">{{ field.label }}</label>
                <select
                  :value="getSettingValue(field.key)"
                  @change="setSettingValue(field.key, ($event.target as HTMLSelectElement).value)"
                  class="input !py-1.5 !text-sm"
                >
                  <option v-for="opt in field.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Edit Mode: Upload -->
        <Transition name="slide-down">
          <div v-if="imageStore.editMode" class="border-b border-gray-100 px-4 py-3 dark:border-dark-800">
            <div
              :class="[
                'relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-5 transition-colors',
                isDragging
                  ? 'border-primary-400 bg-primary-50/50 dark:border-primary-600 dark:bg-primary-900/20'
                  : 'border-gray-200 bg-gray-50 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-dark-500'
              ]"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <div v-if="editPreview" class="relative">
                <img :src="editPreview" class="max-h-40 rounded-lg object-contain shadow-md" alt="Reference" />
                <button
                  class="absolute -right-2 -top-2 rounded-full bg-red-500 p-1 text-white shadow hover:bg-red-600"
                  @click="clearEditImage"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              <div v-else class="text-center">
                <svg class="mx-auto h-8 w-8 text-gray-300 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
                </svg>
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('image.dragOrClick') }}</p>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/*"
                  class="absolute inset-0 cursor-pointer opacity-0"
                  @change="handleFileSelect"
                />
              </div>
            </div>
          </div>
        </Transition>

        <!-- ==================== Canvas ==================== -->
        <div class="flex-1 overflow-y-auto">
          <!-- Empty State -->
          <div v-if="results.length === 0 && !imageStore.generating" class="flex h-full flex-col items-center justify-center px-6">
            <svg class="h-16 w-16 text-gray-200 dark:text-dark-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
            </svg>
            <p class="mt-3 text-sm text-gray-400 dark:text-gray-500">{{ t('image.noImages') }}</p>
          </div>

          <!-- Generating skeleton -->
          <div v-if="imageStore.generating" class="p-4">
            <div class="grid gap-4" :class="gridColsClass">
              <div
                v-for="i in imageStore.settings.n"
                :key="`skel-${i}`"
                class="aspect-square animate-pulse rounded-xl bg-gray-100 dark:bg-dark-800"
              >
                <div class="flex h-full items-center justify-center">
                  <LoadingSpinner />
                </div>
              </div>
            </div>
          </div>

          <!-- Image Grid -->
          <div v-if="results.length > 0" class="p-4">
            <div class="grid gap-4" :class="gridColsClass">
              <div
                v-for="(img, index) in results"
                :key="index"
                class="group relative overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition-shadow hover:shadow-md dark:border-dark-600 dark:bg-dark-800 animate-fade-in"
                :style="{ animationDelay: `${index * 60}ms` }"
              >
                <div class="aspect-square cursor-pointer" @click="openLightbox(index)">
                  <img
                    :src="getImageSrc(img)"
                    :alt="img.revised_prompt || 'Generated image'"
                    class="h-full w-full object-cover"
                    loading="lazy"
                  />
                </div>
                <!-- Hover overlay -->
                <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
                  <div class="flex items-end justify-between gap-2">
                    <p v-if="img.revised_prompt" class="line-clamp-2 text-[11px] leading-relaxed text-white/90">{{ img.revised_prompt }}</p>
                    <button
                      class="shrink-0 rounded-lg bg-white/20 p-1.5 text-white backdrop-blur-sm hover:bg-white/30"
                      :title="t('image.download')"
                      @click.stop="downloadImage(img, index)"
                    >
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ==================== Prompt Input ==================== -->
        <div class="border-t border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
          <div class="flex gap-3">
            <textarea
              v-model="prompt"
              :placeholder="t('image.promptPlaceholder')"
              rows="2"
              class="input flex-1 resize-none"
              @keydown.ctrl.enter="handleGenerate"
              @keydown.meta.enter="handleGenerate"
            />
            <button
              :disabled="imageStore.generating || !prompt.trim()"
              :class="[
                'btn self-end px-5 py-2.5',
                imageStore.generating || !prompt.trim()
                  ? 'btn-secondary cursor-not-allowed opacity-50'
                  : 'btn-primary'
              ]"
              @click="handleGenerate"
            >
              <svg v-if="imageStore.generating" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {{ imageStore.generating ? t('image.generating') : (imageStore.editMode ? t('image.edit') : t('image.generate')) }}
            </button>
          </div>
          <p class="mt-1.5 text-[11px] text-gray-400 dark:text-gray-500">Ctrl + Enter {{ t('image.generate') }}</p>
        </div>
      </div>

      <!-- Lightbox -->
      <ImageLightbox
        :visible="lightboxVisible"
        :images="results"
        :initial-index="lightboxIndex"
        @close="lightboxVisible = false"
      />

      <!-- Confirm dialogs -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="showClearConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50"
            @click.self="showClearConfirm = false"
          >
            <div class="card mx-4 w-full max-w-sm p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.clearHistory') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmClear') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button class="btn btn-secondary btn-sm" @click="showClearConfirm = false">{{ t('common.cancel') }}</button>
                <button class="btn btn-danger btn-sm" @click="confirmClear">{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>

      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="showDeleteConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50"
            @click.self="showDeleteConfirm = false"
          >
            <div class="card mx-4 w-full max-w-sm p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.deleteSession') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmDelete') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button class="btn btn-secondary btn-sm" @click="showDeleteConfirm = false">{{ t('common.cancel') }}</button>
                <button class="btn btn-danger btn-sm" @click="confirmDelete">{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useImageStore } from '@/stores/image'
import type { ImageSession, ImageResult } from '@/api/image'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ImageLightbox from '@/components/image/ImageLightbox.vue'

const { t } = useI18n()
const imageStore = useImageStore()

// ==================== State ====================
const prompt = ref('')
const isDragging = ref(false)
const editFile = ref<File | null>(null)
const editPreview = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const mobileShowSessions = ref(false)
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const showClearConfirm = ref(false)
const showDeleteConfirm = ref(false)
const pendingDeleteId = ref<string | null>(null)
const showSettings = ref(false)

// ==================== Settings ====================
const settingsFields = computed(() => [
  {
    key: 'model', label: t('image.model'),
    options: [
      { value: 'auto', label: 'auto' },
      { value: 'gpt-image-1', label: 'gpt-image-1' },
      { value: 'dall-e-3', label: 'dall-e-3' },
      { value: 'dall-e-2', label: 'dall-e-2' },
    ]
  },
  {
    key: 'size', label: t('image.size'),
    options: [
      { value: 'auto', label: 'auto' },
      { value: '1024x1024', label: '1024×1024' },
      { value: '1536x1024', label: '1536×1024' },
      { value: '1024x1536', label: '1024×1536' },
    ]
  },
  {
    key: 'quality', label: t('image.quality'),
    options: [
      { value: 'auto', label: 'auto' },
      { value: 'low', label: 'low' },
      { value: 'medium', label: 'medium' },
      { value: 'high', label: 'high' },
    ]
  },
  {
    key: 'n', label: t('image.count'),
    options: [
      { value: '1', label: '1' },
      { value: '2', label: '2' },
      { value: '3', label: '3' },
      { value: '4', label: '4' },
    ]
  },
])

function getSettingValue(key: string): string {
  if (key === 'model') return imageStore.settings.model
  if (key === 'size') return imageStore.settings.size
  if (key === 'quality') return imageStore.settings.quality
  if (key === 'n') return String(imageStore.settings.n)
  return ''
}

function setSettingValue(key: string, value: string) {
  if (key === 'model') imageStore.settings.model = value
  else if (key === 'size') imageStore.settings.size = value
  else if (key === 'quality') imageStore.settings.quality = value
  else if (key === 'n') imageStore.settings.n = Number(value)
}

// ==================== Computed ====================
const sessions = computed(() => imageStore.sessions)
const results = computed(() => imageStore.results)
const currentSessionId = computed(() => imageStore.currentSession?.id || null)

const gridColsClass = computed(() => {
  const n = imageStore.settings.n
  if (n <= 1) return 'grid-cols-1 max-w-xl mx-auto'
  if (n === 2) return 'grid-cols-1 sm:grid-cols-2 max-w-2xl mx-auto'
  return 'grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'
})

// ==================== Methods ====================
function getImageSrc(img: ImageResult): string {
  if (img.b64_json) return `data:image/png;base64,${img.b64_json}`
  return img.url || ''
}

function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr), now = new Date(), diff = now.getTime() - d.getTime()
    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
    const m = String(d.getMonth() + 1).padStart(2, '0'), day = String(d.getDate()).padStart(2, '0')
    const h = String(d.getHours()).padStart(2, '0'), min = String(d.getMinutes()).padStart(2, '0')
    return d.getFullYear() === now.getFullYear() ? `${m}-${day} ${h}:${min}` : `${d.getFullYear()}-${m}-${day} ${h}:${min}`
  } catch { return dateStr }
}

async function handleNewSession() {
  const title = prompt.value.trim().slice(0, 50) || t('image.newSession')
  await imageStore.createAndSelectSession(title)
  mobileShowSessions.value = false
}

function handleSelectSession(session: ImageSession) {
  imageStore.selectSession(session)
  mobileShowSessions.value = false
}

function handleDeleteSession(id: string) {
  pendingDeleteId.value = id
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  if (pendingDeleteId.value) await imageStore.removeSession(pendingDeleteId.value)
  pendingDeleteId.value = null
  showDeleteConfirm.value = false
}

function handleClearHistory() { showClearConfirm.value = true }
async function confirmClear() { await imageStore.removeAllSessions(); showClearConfirm.value = false }

async function handleGenerate() {
  const trimmed = prompt.value.trim()
  if (!trimmed || imageStore.generating) return
  if (!imageStore.currentSession) await imageStore.createAndSelectSession(trimmed.slice(0, 50))
  try {
    if (imageStore.editMode && editFile.value) {
      const fd = new FormData()
      fd.append('image', editFile.value)
      fd.append('prompt', trimmed)
      fd.append('model', imageStore.settings.model)
      fd.append('n', String(imageStore.settings.n))
      fd.append('size', imageStore.settings.size)
      fd.append('quality', imageStore.settings.quality)
      await imageStore.edit(fd)
    } else {
      await imageStore.generate(trimmed)
    }
  } catch (e) { console.error('Generation failed:', e) }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.[0]) setEditFile(input.files[0])
}
function handleDrop(e: DragEvent) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) setEditFile(file)
}
function setEditFile(file: File) {
  editFile.value = file
  const reader = new FileReader()
  reader.onload = (e) => { editPreview.value = e.target?.result as string }
  reader.readAsDataURL(file)
}
function clearEditImage() {
  editFile.value = null; editPreview.value = null
  if (fileInput.value) fileInput.value.value = ''
}

function downloadImage(img: ImageResult, index: number) {
  const src = getImageSrc(img)
  const link = document.createElement('a'); link.href = src; link.download = `image-${Date.now()}-${index}.png`
  document.body.appendChild(link); link.click(); document.body.removeChild(link)
}
function downloadAll() { results.value.forEach((img, i) => setTimeout(() => downloadImage(img, i), i * 300)) }
function openLightbox(index: number) { lightboxIndex.value = index; lightboxVisible.value = true }

// ==================== Lifecycle ====================
onMounted(() => { imageStore.loadSessions() })
</script>

<style scoped>
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}
.slide-down-enter-from,
.slide-down-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
.slide-down-enter-to,
.slide-down-leave-from {
  max-height: 200px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@keyframes fade-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.animate-fade-in {
  animation: fade-in 0.35s ease-out both;
}
</style>
