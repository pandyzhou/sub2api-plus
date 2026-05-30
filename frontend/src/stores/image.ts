/**
 * Image Generator Store
 * Manages image generation sessions, settings, and generation state
 */

import { defineStore } from 'pinia'
import { ref, reactive, readonly } from 'vue'
import * as imageApi from '@/api/image'
import type { ImageSession, ImageRecord, ImageResult, ImageGenerateParams } from '@/api/image'

export interface ImageSettings {
  model: string
  size: string
  quality: string
  n: number
}

function normalizeSession(session: ImageSession): ImageSession {
  return {
    ...session,
    records: session.records || [],
    images: session.images || []
  }
}

function sortSessions(items: ImageSession[]): ImageSession[] {
  return [...items].sort((a, b) => String(b.updated_at || '').localeCompare(String(a.updated_at || '')))
}

function imageResultToStoredImage(image: ImageResult): string | null {
  if (image.b64_json) return image.b64_json
  if (image.url) return image.url
  return null
}

function storedImagesToResults(images: readonly string[] | undefined): ImageResult[] {
  return (images || []).flatMap((image): ImageResult[] => {
    const value = String(image || '').trim()
    if (!value) return []
    if (value.startsWith('http://') || value.startsWith('https://') || value.startsWith('data:')) {
      return [{ url: value }]
    }
    return [{ b64_json: value }]
  })
}

function normalizeImageRequestModel(model: string): string | undefined {
  const trimmed = String(model || '').trim()
  if (!trimmed || trimmed.toLowerCase() === 'auto') return undefined
  return trimmed
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

  // ==================== Internal helpers ====================

  function upsertSession(session: ImageSession) {
    const normalized = normalizeSession(session)
    const index = sessions.value.findIndex((item) => item.id === normalized.id)
    if (index >= 0) {
      sessions.value = sortSessions([
        ...sessions.value.slice(0, index),
        normalized,
        ...sessions.value.slice(index + 1)
      ])
    } else {
      sessions.value = sortSessions([normalized, ...sessions.value])
    }
    return normalized
  }

  function syncCurrentSessionRecord(prompt: string, images: ImageResult[]) {
    if (!currentSession.value) return

    const now = new Date().toISOString()
    const storedImages = images.flatMap((image) => {
      const stored = imageResultToStoredImage(image)
      return stored ? [stored] : []
    })
    const record: ImageRecord = {
      id: `img_rec_${Date.now()}_${Math.random().toString(16).slice(2)}`,
      prompt,
      model: settings.model,
      images: storedImages,
      params: {
        model: settings.model,
        size: settings.size,
        quality: settings.quality,
        n: settings.n
      },
      created_at: now
    }

    const nextSession: ImageSession = {
      ...currentSession.value,
      records: [...(currentSession.value.records || []), record],
      images: [...images, ...(currentSession.value.images || [])],
      updated_at: now
    }
    currentSession.value = nextSession
    upsertSession(nextSession)
  }

  // ==================== Actions ====================

  async function loadSessions() {
    loading.value = true
    try {
      sessions.value = sortSessions((await imageApi.fetchSessions()).map(normalizeSession))
    } catch (error) {
      console.error('Failed to load sessions:', error)
    } finally {
      loading.value = false
    }
  }

  async function loadSession(id: string) {
    try {
      const session = normalizeSession(await imageApi.fetchSession(id))
      currentSession.value = session
      upsertSession(session)
      const recordImages = session.records?.flatMap((record) => storedImagesToResults(record.images)) || []
      results.value = recordImages.length > 0 ? recordImages : [...(session.images || [])]
      return session
    } catch (error) {
      console.error('Failed to load session:', error)
      return null
    }
  }

  async function createAndSelectSession(title: string): Promise<ImageSession | null> {
    try {
      const session = normalizeSession(await imageApi.createSession(title))
      upsertSession(session)
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
        model: normalizeImageRequestModel(settings.model),
        n: settings.n,
        size: settings.size,
        quality: settings.quality,
        session_id: currentSession.value?.id,
        title: currentSession.value ? undefined : prompt.slice(0, 50)
      }
      const response = await imageApi.generateImage(params)
      const images: ImageResult[] = Array.isArray(response) ? response : (response?.data || response?.images || [])
      results.value = [...images, ...results.value]
      syncCurrentSessionRecord(prompt, images)
      return images
    } finally {
      generating.value = false
    }
  }

  async function edit(formData: FormData): Promise<ImageResult[]> {
    generating.value = true
    try {
      if (currentSession.value?.id && !formData.has('session_id')) {
        formData.append('session_id', currentSession.value.id)
      }
      const normalizedModel = normalizeImageRequestModel(String(formData.get('model') || settings.model))
      if (normalizedModel) {
        formData.set('model', normalizedModel)
      } else {
        formData.delete('model')
      }
      const prompt = String(formData.get('prompt') || '')
      const response = await imageApi.editImage(formData)
      const images: ImageResult[] = Array.isArray(response) ? response : (response?.data || response?.images || [])
      results.value = [...images, ...results.value]
      if (prompt) syncCurrentSessionRecord(prompt, images)
      return images
    } finally {
      generating.value = false
    }
  }

  function selectSession(session: ImageSession) {
    const normalized = normalizeSession(session)
    currentSession.value = normalized
    const recordImages = normalized.records?.flatMap((record) => storedImagesToResults(record.images)) || []
    results.value = recordImages.length > 0 ? recordImages : [...(normalized.images || [])]
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
