<template>
  <Teleport to="body">
    <Transition name="lightbox">
      <div
        v-if="visible"
        class="fixed inset-0 z-[9998] flex items-center justify-center bg-black/80 backdrop-blur-sm"
        @click.self="close"
      >
        <!-- Close -->
        <button
          class="absolute right-4 top-4 z-10 rounded-full bg-white/10 p-2 text-white hover:bg-white/20 transition-colors"
          @click="close"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>

        <!-- Download -->
        <button
          v-if="currentImage"
          class="absolute right-14 top-4 z-10 rounded-full bg-white/10 p-2 text-white hover:bg-white/20 transition-colors"
          :title="t('image.download')"
          @click.stop="downloadCurrent"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
        </button>

        <!-- Prev -->
        <button
          v-if="images.length > 1"
          class="absolute left-4 top-1/2 -translate-y-1/2 z-10 rounded-full bg-white/10 p-2.5 text-white hover:bg-white/20 transition-colors disabled:opacity-30"
          :disabled="currentIndex === 0"
          @click.stop="prev"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" /></svg>
        </button>

        <!-- Next -->
        <button
          v-if="images.length > 1"
          class="absolute right-4 top-1/2 -translate-y-1/2 z-10 rounded-full bg-white/10 p-2.5 text-white hover:bg-white/20 transition-colors disabled:opacity-30"
          :disabled="currentIndex === images.length - 1"
          @click.stop="next"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
        </button>

        <!-- Image -->
        <div class="flex max-h-[90vh] max-w-[90vw] items-center justify-center p-8">
          <img
            v-if="currentImage"
            :src="getImageSrc(currentImage)"
            :alt="currentImage.revised_prompt || 'Generated image'"
            class="max-h-full max-w-full rounded-lg object-contain shadow-2xl"
          />
        </div>

        <!-- Counter & prompt -->
        <div class="absolute bottom-6 left-1/2 -translate-x-1/2 text-center max-w-2xl">
          <p v-if="images.length > 1" class="mb-2 text-sm text-white/60 tabular-nums">
            {{ currentIndex + 1 }} / {{ images.length }}
          </p>
          <p
            v-if="currentImage?.revised_prompt"
            class="rounded-lg bg-black/50 px-4 py-2 text-sm text-white/90 backdrop-blur-sm"
          >{{ currentImage.revised_prompt }}</p>
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
  const link = document.createElement('a'); link.href = src; link.download = `image-${Date.now()}.png`
  document.body.appendChild(link); link.click(); document.body.removeChild(link)
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
  transition: opacity 0.25s ease;
}
.lightbox-enter-from,
.lightbox-leave-to {
  opacity: 0;
}
</style>
