import { joinURL } from 'ufo'

function trimTrailingSlash(input: string) {
  return input.replace(/\/+$/, '')
}

export function useSiteSeo() {
  const route = useRoute()
  const config = useRuntimeConfig()
  const siteUrl = computed(() => trimTrailingSlash(config.public.siteUrl || 'http://127.0.0.1:3000'))
  const siteName = computed(() => config.public.siteName || 'GoPalette Blog')

  function buildUrl(path: string) {
    const normalizedBase = siteUrl.value
    const normalizedPath = path.startsWith('/') ? path : `/${path}`
    return joinURL(normalizedBase, normalizedPath)
  }

  function currentUrl() {
    return buildUrl(route.fullPath || '/')
  }

  return {
    siteUrl,
    siteName,
    buildUrl,
    currentUrl
  }
}
