import { fetchCategories, fetchTags } from '~/composables/useBlogApi'

type WriteResourcesStatus = 'idle' | 'loading' | 'ready' | 'error'

let pendingLoad: Promise<boolean> | null = null

export function useWriteResources() {
  const categories = useState<Array<{ id: string, name: string }>>('write.resources.categories', () => [])
  const tags = useState<Array<{ id: string, name: string }>>('write.resources.tags', () => [])
  const status = useState<WriteResourcesStatus>('write.resources.status', () => 'idle')
  const errorMessage = useState<string>('write.resources.error', () => '')

  async function ensureLoaded() {
    if (status.value === 'ready' && categories.value.length && tags.value.length) {
      return true
    }

    if (pendingLoad) {
      return pendingLoad
    }

    status.value = 'loading'
    errorMessage.value = ''

    pendingLoad = (async () => {
      try {
        const [categoriesData, tagsData] = await Promise.all([
          fetchCategories(1, 200),
          fetchTags(1, 200)
        ])

        categories.value = categoriesData.categories || []
        tags.value = tagsData.tags || []
        status.value = 'ready'
        return true
      } catch (error: unknown) {
        status.value = 'error'
        const typed = error as { message?: unknown, data?: { message?: unknown } }
        errorMessage.value = String(typed?.data?.message || typed?.message || '写作资源加载失败')
        return false
      } finally {
        pendingLoad = null
      }
    })()

    return pendingLoad
  }

  return {
    categories,
    tags,
    status,
    errorMessage,
    isReady: computed(() => status.value === 'ready'),
    isLoading: computed(() => status.value === 'loading'),
    tagSuggestions: computed(() => tags.value.map(item => item.name)),
    ensureLoaded
  }
}
