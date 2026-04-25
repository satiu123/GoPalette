type WritePreloadStatus = 'idle' | 'loading' | 'ready' | 'error'

export function useWritePreload() {
  const { ensureLoaded } = useWriteResources()
  const status = useState<WritePreloadStatus>('write.preload.status', () => 'idle')
  const errorMessage = useState<string>('write.preload.error', () => '')

  async function preload() {
    if (!import.meta.client) return false
    if (status.value === 'loading' || status.value === 'ready') {
      return status.value === 'ready'
    }

    status.value = 'loading'
    errorMessage.value = ''

    try {
      await Promise.all([
        preloadRouteComponents('/write'),
        import('~/components/write/WriteEditorWorkspace.client.vue'),
        ensureLoaded()
      ])

      status.value = 'ready'
      return true
    } catch (error: unknown) {
      status.value = 'error'
      const typed = error as { message?: unknown, data?: { message?: unknown } }
      errorMessage.value = String(typed?.data?.message || typed?.message || '预加载失败')
      return false
    }
  }

  return {
    status,
    errorMessage,
    isPreloading: computed(() => status.value === 'loading'),
    isReady: computed(() => status.value === 'ready'),
    preload
  }
}
