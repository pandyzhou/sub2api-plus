<template>
  <Teleport to="body">
    <Transition name="lightbox">
      <div
        v-if="visible"
        class="fixed inset-0 z-[9998] flex items-center justify-center bg-[#0a0a0b]/95 backdrop-blur-xl"
        @click.self="close"
      >
        <!-- Top bar -->
        <div class="absolute inset-x-0 top-0 z-10 flex items-center justify-between px-5 py-4">
          <span v-if="images.length > 1" class="text-[13px] font-medium text-white/40 tabular-nums">
            {{ currentIndex + 1 }} <span class="text-white/20">/</span> {{ images.length }}
          </span>
          <span v-else />
          <div class="flex items-center gap-2">
            <button
              v-if="currentImage"
              class="flex h-9 w-9 items-center justify-center rounded-lg bg-white/[0.06] text-white/50 transition-all hover:bg-white/[0.1] hover:text-white/80"
              :title="t('image.download')"
              @click.stop="downloadCurrent"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
            </button>
            <button
              class="flex h-9 w-9 items-center justify-center rounded-lg bg-white/[0.06] text-white/50 transition-all hover:bg-white/[0.1] hover:text-white/80"
              @click="close"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          </div>
        </div>

        <!-- Previous -->
        <button
          v-if="images.length > 1"
          class="absolute left-4 top-1/2 z-10 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-xl bg-white/[0.06] text-white/50 transition-all hover:bg-white/[0.1] hover:text-white/80 disabled:opacity-20"
          :disabled="currentIndex === 0"
          @click.stop="prev"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" /></svg>
        </button>

        <!-- Next -->
        <button
          v-if="images.length > 1"
          class="absolute right-4 top-1/2 z-10 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-xl bg-white/[0.06] text-white/50 transition-all hover:bg-white/[0.1] hover:text-white/80 disabled:opacity-20"
          :disabled="currentIndex === images.length - 1"
          @click.stop="next"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
        </button>

        <!-- Image -->
        <div class="flex max-h-[85vh] max-w-[90vw] items-center justify-center p-12">
          <img
            v-if="currentImage"
            :src="getImageSrc(currentImage)"
            :alt="currentImage.revised_prompt || 'Generated image'"
            class="max-h-full max-w-full rounded-xl object-contain shadow-2xl shadow-black/50"
          />
        </div>

        <!-- Bottom: revised prompt -->
        <div v-if="currentImage?.revised_prompt" class="absolute inset-x-0 bottom-0 z-10 flex justify-center px-6 pb-6">
          <div class="max-w-2xl rounded-xl border border-white/[0.06] bg-[#0e0e10]/80 px-5 py-3 backdrop-blur-xl">
            <p class="text-[13px] leading-relaxed text-white/70">{{ currentImage.revised_prompt }}</p>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ImageResult } from '@/api/image'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  images: ImageResult[]
  initialIndex?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const currentIndex = ref(0)

const currentImage = computed(() => props.images[currentIndex.value] || null)

watch(() => props.visible, (val) => {
  if (val) {
    currentIndex.value = props.initialIndex ?? 0
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
})

watch(() => props.initialIndex, (val) => {
  if (val !== undefined) currentIndex.value = val
})

function getImageSrc(img: ImageResult): string {
  if (img.b64_json) return `data:image/png;base64,${img.b64_json}`
  return img.url || ''
}

function close() { emit('close') }
function prev() { if (currentIndex.value > 0) currentIndex.value-- }
function next() { if (currentIndex.value < props.images.length - 1) currentIndex.value++ }

function downloadCurrent() {
  if (!currentImage.value) return
  const src = getImageSrc(currentImage.value)
  const link = document.createElement('a')
  link.href = src
  link.download = `image-${Date.now()}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function handleKeydown(e: KeyboardEvent) {
  if (!props.visible) return
  if (e.key === 'Escape') close()
  if (e.key === 'ArrowLeft') prev()
  if (e.key === 'ArrowRight') next()
}

onMounted(() => { document.addEventListener('keydown', handleKeydown) })
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = ''
})
</script>

<style scoped>
.lightbox-enter-active,
.lightbox-leave-active {
  transition: opacity 0.3s ease;
}
.lightbox-enter-from,
.lightbox-leave-to {
  opacity: 0;
}
</style>
