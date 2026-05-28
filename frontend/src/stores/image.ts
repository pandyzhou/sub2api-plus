/**
 * Image Generator Store
 * Manages image generation sessions, settings, and generation state
 */

import { defineStore } from 'pinia'
import { ref, reactive, readonly } from 'vue'
import * as imageApi from '@/api/image'
import type { ImageSession, ImageResult, ImageGenerateParams } from '@/api/image'

export interface ImageSettings {
  model: string
  size: string
  quality: string
  n: number
}

export const useImageStore = defineStore('image', () => {
  // ==================== State ====================

  const sessions = ref<ImageSession[]>([])
  const currentSession = ref<ImageSession | null>(null)
  const generating = ref(false)
  const editMode = ref(false)
  const loading = ref(false)
  const results = ref<ImageResult[]>([])

  const settings = reactive<ImageSettings>({
    model: 'auto',
    size: 'auto',
    quality: 'auto',
    n: 1
  })

  // ==================== Actions ====================

  async function loadSessions() {
    loading.value = true
    try {
      sessions.value = await imageApi.fetchSessions()
    } catch (error) {
      console.error('Failed to load sessions:', error)
    } finally {
      loading.value = false
    }
  }

  async function loadSession(id: string) {
    try {
      const session = await imageApi.fetchSession(id)
      currentSession.value = session
      results.value = [...(session.images || [])]
    } catch (error) {
      console.error('Failed to load session:', error)
    }
  }

  async function createAndSelectSession(title: string): Promise<ImageSession | null> {
    try {
      const session = await imageApi.createSession(title)
      sessions.value.unshift(session)
      currentSession.value = session
      results.value = []
      return session
    } catch (error) {
      console.error('Failed to create session:', error)
      return null
    }
  }

  async function removeSession(id: string) {
    try {
      await imageApi.deleteSession(id)
      sessions.value = sessions.value.filter(s => s.id !== id)
      if (currentSession.value?.id === id) {
        currentSession.value = sessions.value[0] || null
        results.value = [...(currentSession.value?.images || [])]
      }
      return true
    } catch (error) {
      console.error('Failed to delete session:', error)
      return false
    }
  }

  async function removeAllSessions() {
    try {
      await imageApi.clearSessions()
      sessions.value = []
      currentSession.value = null
      results.value = []
      return true
    } catch (error) {
      console.error('Failed to clear sessions:', error)
      return false
    }
  }

  async function generate(prompt: string): Promise<ImageResult[]> {
    generating.value = true
    try {
      const params: ImageGenerateParams = {
        prompt,
        model: settings.model,
        n: settings.n,
        size: settings.size,
        quality: settings.quality
      }
      const response = await imageApi.generateImage(params)
      const images: ImageResult[] = response?.data || response?.images || []
      results.value = [...images, ...results.value]
      return images
    } finally {
      generating.value = false
    }
  }

  async function edit(formData: FormData): Promise<ImageResult[]> {
    generating.value = true
    try {
      const response = await imageApi.editImage(formData)
      const images: ImageResult[] = response?.data || response?.images || []
      results.value = [...images, ...results.value]
      return images
    } finally {
      generating.value = false
    }
  }

  function selectSession(session: ImageSession) {
    currentSession.value = session
    results.value = [...(session.images || [])]
  }

  function clearResults() {
    results.value = []
  }

  return {
    // State
    sessions: readonly(sessions),
    currentSession,
    generating: readonly(generating),
    editMode,
    loading: readonly(loading),
    results,
    settings,

    // Actions
    loadSessions,
    loadSession,
    createAndSelectSession,
    removeSession,
    removeAllSessions,
    generate,
    edit,
    selectSession,
    clearResults
  }
})
