/**
 * Image Generation API
 * API calls for image generation, editing, and session management
 */

import { apiClient } from './client'

const IMAGE_REQUEST_TIMEOUT_MS = 600000

export interface ImageGenerateParams {
  prompt: string
  model?: string
  n?: number
  size?: string
  quality?: string
  response_format?: string
  session_id?: string
  title?: string
}

export interface ImageRecord {
  id: string
  prompt: string
  model?: string
  images?: readonly string[]
  params?: unknown
  created_at: string
}

export interface ImageSession {
  id: string
  title: string
  created_at: string
  updated_at: string
  records?: readonly ImageRecord[]
  images?: readonly ImageResult[]
}

export interface ImageResult {
  b64_json?: string
  url?: string
  revised_prompt?: string
}

/**
 * Generate images from text prompt
 */
export async function generateImage(params: ImageGenerateParams): Promise<any> {
  const { data } = await apiClient.post('/user/image/generate', params, { timeout: IMAGE_REQUEST_TIMEOUT_MS })
  return data?.data || data
}

/**
 * Edit an existing image with mask/prompt
 */
export async function editImage(formData: FormData): Promise<any> {
  const { data } = await apiClient.post('/user/image/edit', formData, {
    timeout: IMAGE_REQUEST_TIMEOUT_MS,
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return data?.data || data
}

/**
 * Get all image sessions
 */
export async function fetchSessions(): Promise<ImageSession[]> {
  const { data } = await apiClient.get('/user/image/sessions')
  return data?.sessions || data || []
}

/**
 * Get a single session by ID
 */
export async function fetchSession(id: string): Promise<ImageSession> {
  const { data } = await apiClient.get(`/user/image/sessions/${id}`)
  return data
}

/**
 * Create a new image session
 */
export async function createSession(title: string): Promise<ImageSession> {
  const { data } = await apiClient.post('/user/image/sessions', { title })
  return data
}

/**
 * Delete a single session
 */
export async function deleteSession(id: string): Promise<void> {
  await apiClient.delete(`/user/image/sessions/${id}`)
}

/**
 * Clear all sessions
 */
export async function clearSessions(): Promise<void> {
  await apiClient.delete('/user/image/sessions')
}
