import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useImageStore } from '@/stores/image'
import { generateImage, editImage } from '@/api/image'

vi.mock('@/api/image', () => ({
  fetchSessions: vi.fn(async () => []),
  fetchSession: vi.fn(),
  createSession: vi.fn(async (title: string) => ({
    id: 'session-1',
    title,
    created_at: '2026-05-29T00:00:00Z',
    updated_at: '2026-05-29T00:00:00Z',
    records: [],
    images: []
  })),
  deleteSession: vi.fn(),
  clearSessions: vi.fn(),
  generateImage: vi.fn(async () => ({ data: [] })),
  editImage: vi.fn(async () => ({ data: [] }))
}))

describe('useImageStore image requests', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('omits auto-valued options for text image generation so backend can apply image defaults', async () => {
    const store = useImageStore()
    store.settings.model = 'auto'
    store.settings.size = 'auto'
    store.settings.quality = 'auto'

    await store.createAndSelectSession('测试图片')
    await store.generate('一只猫娘在写代码')

    expect(generateImage).toHaveBeenCalledTimes(1)
    const submitted = vi.mocked(generateImage).mock.calls[0][0]
    expect(submitted).not.toHaveProperty('model')
    expect(submitted).not.toHaveProperty('size')
    expect(submitted).not.toHaveProperty('quality')
  })

  it('omits auto-valued options for image edit multipart requests', async () => {
    const store = useImageStore()
    store.settings.model = 'auto'
    store.settings.size = 'auto'
    store.settings.quality = 'auto'

    await store.createAndSelectSession('编辑图片')
    const formData = new FormData()
    formData.append('prompt', '改成赛博朋克风格')
    formData.append('model', store.settings.model)
    formData.append('size', store.settings.size)
    formData.append('quality', store.settings.quality)

    await store.edit(formData)

    expect(editImage).toHaveBeenCalledTimes(1)
    const submitted = vi.mocked(editImage).mock.calls[0][0]
    expect(submitted.get('model')).toBeNull()
    expect(submitted.get('size')).toBeNull()
    expect(submitted.get('quality')).toBeNull()
  })
})
