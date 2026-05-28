<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-4rem)] overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
      <!-- Left Sidebar: Session History -->
      <aside
        :class="[
          'flex flex-col border-r border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800',
          'w-70 shrink-0',
          mobileShowSessions ? 'absolute inset-0 z-20 w-full sm:relative sm:w-70' : 'hidden sm:flex'
        ]"
      >
        <!-- Sidebar Header -->
        <div class="flex items-center justify-between border-b border-gray-200 p-4 dark:border-dark-700">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('image.title') }}</h2>
          <button
            class="sm:hidden rounded-md p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            @click="mobileShowSessions = false"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- New Session Button -->
        <div class="p-3">
          <button
            class="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-3 py-2.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
            @click="handleNewSession"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            {{ t('image.newSession') }}
          </button>
        </div>

        <!-- Session List -->
        <div class="flex-1 overflow-y-auto scrollbar-hide px-2">
          <div v-if="imageStore.loading" class="flex justify-center py-8">
            <LoadingSpinner />
          </div>
          <div v-else-if="sessions.length === 0" class="py-8 text-center text-sm text-gray-400 dark:text-gray-500">
            {{ t('image.noSessions') }}
          </div>
          <div v-else class="space-y-1">
            <div
              v-for="session in sessions"
              :key="session.id"
              :class="[
                'group relative cursor-pointer rounded-lg px-3 py-2.5 transition-colors',
                currentSessionId === session.id
                  ? 'bg-blue-50 dark:bg-blue-900/30'
                  : 'hover:bg-gray-100 dark:hover:bg-dark-700'
              ]"
              @click="handleSelectSession(session)"
            >
              <div class="flex items-start gap-2.5">
                <div class="min-w-0 flex-1">
                  <p
                    :class="[
                      'truncate text-sm font-medium',
                      currentSessionId === session.id
                        ? 'text-blue-700 dark:text-blue-300'
                        : 'text-gray-900 dark:text-white'
                    ]"
                  >
                    {{ session.title || t('image.newSession') }}
                  </p>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ formatTime(session.updated_at || session.created_at) }}
                  </p>
                </div>
                <button
                  class="shrink-0 rounded p-1 text-gray-400 opacity-0 transition-opacity hover:bg-red-100 hover:text-red-500 group-hover:opacity-100 dark:hover:bg-red-900/30 dark:hover:text-red-400"
                  :title="t('image.deleteSession')"
                  @click.stop="handleDeleteSession(session.id)"
                >
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Clear History -->
        <div class="border-t border-gray-200 p-3 dark:border-dark-700">
          <button
            v-if="sessions.length > 0"
            class="flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2 text-xs text-gray-500 hover:bg-gray-200 hover:text-red-500 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-red-400 transition-colors"
            @click="handleClearHistory"
          >
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
            </svg>
            {{ t('image.clearHistory') }}
          </button>
        </div>
      </aside>

      <!-- Right Main Area -->
      <div class="flex flex-1 flex-col overflow-hidden">
        <!-- Top Bar: Mode Tabs + Mobile Toggle -->
        <div class="flex items-center border-b border-gray-200 bg-white px-4 dark:border-dark-700 dark:bg-dark-900">
          <!-- Mobile session toggle -->
          <button
            class="mr-3 sm:hidden rounded-md p-1.5 text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
            @click="mobileShowSessions = true"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
            </svg>
          </button>

          <!-- Mode Tabs -->
          <div class="flex gap-1 py-2">
            <button
              :class="[
                'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
                !imageStore.editMode
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'
              ]"
              @click="imageStore.editMode = false"
            >
              {{ t('image.textToImage') }}
            </button>
            <button
              :class="[
                'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
                imageStore.editMode
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'
              ]"
              @click="imageStore.editMode = true"
            >
              {{ t('image.imageEdit') }}
            </button>
          </div>

          <!-- Spacer -->
          <div class="flex-1" />

          <!-- Download All -->
          <button
            v-if="results.length > 1"
            class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700 transition-colors"
            @click="downloadAll"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
            </svg>
            {{ t('image.downloadAll') }}
          </button>
        </div>

        <!-- Content Area -->
        <div class="flex flex-1 flex-col overflow-y-auto">
          <!-- Settings Panel -->
          <div class="border-b border-gray-100 bg-gray-50/50 px-4 py-3 dark:border-dark-800 dark:bg-dark-800/50">
            <div class="flex flex-wrap items-end gap-4">
              <!-- Model -->
              <div class="flex flex-col gap-1">
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('image.model') }}</label>
                <select
                  v-model="imageStore.settings.model"
                  class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-dark-600 dark:bg-dark-700 dark:text-white"
                >
                  <option value="auto">auto</option>
                  <option value="gpt-image-1">gpt-image-1</option>
                  <option value="dall-e-3">dall-e-3</option>
                  <option value="dall-e-2">dall-e-2</option>
                </select>
              </div>

              <!-- Size -->
              <div class="flex flex-col gap-1">
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('image.size') }}</label>
                <select
                  v-model="imageStore.settings.size"
                  class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-dark-600 dark:bg-dark-700 dark:text-white"
                >
                  <option value="auto">auto</option>
                  <option value="1024x1024">1024x1024</option>
                  <option value="1536x1024">1536x1024</option>
                  <option value="1024x1536">1024x1536</option>
                </select>
              </div>

              <!-- Quality -->
              <div class="flex flex-col gap-1">
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('image.quality') }}</label>
                <select
                  v-model="imageStore.settings.quality"
                  class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-dark-600 dark:bg-dark-700 dark:text-white"
                >
                  <option value="auto">auto</option>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                </select>
              </div>

              <!-- Count -->
              <div class="flex flex-col gap-1">
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('image.count') }}</label>
                <select
                  v-model.number="imageStore.settings.n"
                  class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-dark-600 dark:bg-dark-700 dark:text-white"
                >
                  <option :value="1">1</option>
                  <option :value="2">2</option>
                  <option :value="3">3</option>
                  <option :value="4">4</option>
                </select>
              </div>
            </div>
          </div>

          <!-- Edit Mode: Image Upload -->
          <div v-if="imageStore.editMode" class="border-b border-gray-100 px-4 py-4 dark:border-dark-800">
            <div
              :class="[
                'relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-6 transition-colors',
                isDragging
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-300 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'
              ]"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <!-- Preview -->
              <div v-if="editPreview" class="relative">
                <img
                  :src="editPreview"
                  class="max-h-48 rounded-lg object-contain shadow-md"
                  alt="Reference image"
                />
                <button
                  class="absolute -right-2 -top-2 rounded-full bg-red-500 p-1 text-white shadow hover:bg-red-600"
                  @click="clearEditImage"
                >
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <!-- Upload prompt -->
              <div v-else class="text-center">
                <svg class="mx-auto h-10 w-10 text-gray-400 dark:text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
                </svg>
                <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.dragOrClick') }}</p>
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

          <!-- Prompt Input -->
          <div class="border-b border-gray-100 px-4 py-4 dark:border-dark-800">
            <div class="flex gap-3">
              <textarea
                v-model="prompt"
                :placeholder="t('image.promptPlaceholder')"
                rows="3"
                class="flex-1 resize-none rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-900 placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-dark-600 dark:bg-dark-700 dark:text-white dark:placeholder:text-gray-500"
                @keydown.ctrl.enter="handleGenerate"
                @keydown.meta.enter="handleGenerate"
              />
              <button
                :disabled="imageStore.generating || !prompt.trim()"
                :class="[
                  'flex items-center justify-center self-end rounded-xl px-5 py-3 text-sm font-medium text-white transition-colors',
                  imageStore.generating || !prompt.trim()
                    ? 'cursor-not-allowed bg-gray-400 dark:bg-gray-600'
                    : 'bg-blue-600 hover:bg-blue-700'
                ]"
                @click="handleGenerate"
              >
                <svg v-if="imageStore.generating" class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                {{ imageStore.generating ? t('image.generating') : (imageStore.editMode ? t('image.edit') : t('image.generate')) }}
              </button>
            </div>
            <p class="mt-2 text-xs text-gray-400 dark:text-gray-500">Ctrl + Enter {{ t('image.generate') }}</p>
          </div>

          <!-- Results Gallery -->
          <div class="flex-1 p-4">
            <!-- Empty State -->
            <div v-if="results.length === 0 && !imageStore.generating" class="flex flex-col items-center justify-center py-20">
              <svg class="h-16 w-16 text-gray-300 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
              </svg>
              <p class="mt-4 text-sm text-gray-400 dark:text-gray-500">{{ t('image.noImages') }}</p>
            </div>

            <!-- Loading State -->
            <div v-if="imageStore.generating" class="mb-6">
              <div class="grid gap-4" :class="gridColsClass">
                <div
                  v-for="i in imageStore.settings.n"
                  :key="`loading-${i}`"
                  class="aspect-square animate-pulse rounded-xl bg-gray-200 dark:bg-dark-700"
                >
                  <div class="flex h-full items-center justify-center">
                    <svg class="h-8 w-8 animate-spin text-gray-400 dark:text-gray-500" viewBox="0 0 24 24" fill="none">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                  </div>
                </div>
              </div>
            </div>

            <!-- Image Grid -->
            <div v-if="results.length > 0" class="grid gap-4" :class="gridColsClass">
              <div
                v-for="(img, index) in results"
                :key="index"
                class="group relative overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition-shadow hover:shadow-md dark:border-dark-600 dark:bg-dark-800"
              >
                <div class="aspect-square cursor-pointer" @click="openLightbox(index)">
                  <img
                    :src="getImageSrc(img)"
                    :alt="img.revised_prompt || 'Generated image'"
                    class="h-full w-full object-cover"
                    loading="lazy"
                  />
                </div>
                <!-- Overlay actions -->
                <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
                  <div class="flex items-end justify-between">
                    <p v-if="img.revised_prompt" class="line-clamp-2 text-xs text-white/90">
                      {{ img.revised_prompt }}
                    </p>
                    <button
                      class="shrink-0 rounded-lg bg-white/20 p-1.5 text-white backdrop-blur-sm hover:bg-white/30"
                      :title="t('image.download')"
                      @click.stop="downloadImage(img, index)"
                    >
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Lightbox -->
      <ImageLightbox
        :visible="lightboxVisible"
        :images="results"
        :initial-index="lightboxIndex"
        @close="lightboxVisible = false"
      />

      <!-- Confirm Dialog for Clear -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="showClearConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50"
            @click.self="showClearConfirm = false"
          >
            <div class="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-dark-800">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.clearHistory') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmClear') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button
                  class="rounded-lg px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
                  @click="showClearConfirm = false"
                >
                  {{ t('common.cancel') }}
                </button>
                <button
                  class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
                  @click="confirmClear"
                >
                  {{ t('common.confirm') }}
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>

      <!-- Confirm Dialog for Delete -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="showDeleteConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50"
            @click.self="showDeleteConfirm = false"
          >
            <div class="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-dark-800">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.deleteSession') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmDelete') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button
                  class="rounded-lg px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
                  @click="showDeleteConfirm = false"
                >
                  {{ t('common.cancel') }}
                </button>
                <button
                  class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
                  @click="confirmDelete"
                >
                  {{ t('common.confirm') }}
                </button>
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
    const d = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - d.getTime()

    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`

    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hours = String(d.getHours()).padStart(2, '0')
    const mins = String(d.getMinutes()).padStart(2, '0')

    if (year === now.getFullYear()) return `${month}-${day} ${hours}:${mins}`
    return `${year}-${month}-${day} ${hours}:${mins}`
  } catch {
    return dateStr
  }
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
  if (pendingDeleteId.value) {
    const success = await imageStore.removeSession(pendingDeleteId.value)
    if (success) {
      // Toast feedback could be added here
    }
  }
  pendingDeleteId.value = null
  showDeleteConfirm.value = false
}

function handleClearHistory() {
  showClearConfirm.value = true
}

async function confirmClear() {
  await imageStore.removeAllSessions()
  showClearConfirm.value = false
}

async function handleGenerate() {
  const trimmed = prompt.value.trim()
  if (!trimmed || imageStore.generating) return

  // Auto-create session if none selected
  if (!imageStore.currentSession) {
    const title = trimmed.slice(0, 50)
    await imageStore.createAndSelectSession(title)
  }

  try {
    if (imageStore.editMode && editFile.value) {
      const formData = new FormData()
      formData.append('image', editFile.value)
      formData.append('prompt', trimmed)
      formData.append('model', imageStore.settings.model)
      formData.append('n', String(imageStore.settings.n))
      formData.append('size', imageStore.settings.size)
      formData.append('quality', imageStore.settings.quality)
      await imageStore.edit(formData)
    } else {
      await imageStore.generate(trimmed)
    }
  } catch (error: any) {
    console.error('Generation failed:', error)
    // Could add toast notification here
  }
}

// Edit mode: file handling
function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.[0]) {
    setEditFile(input.files[0])
  }
}

function handleDrop(e: DragEvent) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) {
    setEditFile(file)
  }
}

function setEditFile(file: File) {
  editFile.value = file
  const reader = new FileReader()
  reader.onload = (e) => {
    editPreview.value = e.target?.result as string
  }
  reader.readAsDataURL(file)
}

function clearEditImage() {
  editFile.value = null
  editPreview.value = null
  if (fileInput.value) fileInput.value.value = ''
}

// Download
function downloadImage(img: ImageResult, index: number) {
  const src = getImageSrc(img)
  const link = document.createElement('a')
  link.href = src
  link.download = `image-${Date.now()}-${index}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function downloadAll() {
  results.value.forEach((img, i) => {
    setTimeout(() => downloadImage(img, i), i * 300)
  })
}

// Lightbox
function openLightbox(index: number) {
  lightboxIndex.value = index
  lightboxVisible.value = true
}

// ==================== Lifecycle ====================

onMounted(() => {
  imageStore.loadSessions()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
