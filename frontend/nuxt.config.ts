// https://nuxt.com/docs/api/configuration/nuxt-config
const gatewayBase = import.meta.env.NUXT_GATEWAY_BASE || 'http://127.0.0.1:8080'
const publicSiteUrl = import.meta.env.NUXT_PUBLIC_SITE_URL || 'http://127.0.0.1:3000'
const publicSiteName = import.meta.env.NUXT_PUBLIC_SITE_NAME || 'GoPalette Blog'
const deepseekApiKey = import.meta.env.DEEPSEEK_API_KEY || ''
const enablePrerender = import.meta.env.NUXT_ENABLE_PRERENDER === 'true'
const authCookieSecure = import.meta.env.NUXT_AUTH_COOKIE_SECURE || ''

export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxthub/core',
    'nuxt-csurf'
  ],

  devtools: {
    enabled: import.meta.dev
  },

  css: ['~/assets/css/main.css'],

  ui: {
    fonts: false,
    experimental: {
      componentDetection: true
    }
  },

  runtimeConfig: {
    gatewayBase,
    deepseekApiKey,
    authCookieSecure,
    public: {
      partykitHost: '',
      siteUrl: publicSiteUrl,
      siteName: publicSiteName
    }
  },

  nitro: {
    prerender: {
      crawlLinks: enablePrerender,
      failOnError: false,
      routes: enablePrerender ? ['/', '/posts', '/rss.xml', '/sitemap.xml'] : []
    }
  },

  hooks: {
    async 'prerender:routes'(ctx) {
      if (!enablePrerender) {
        return
      }

      try {
        const categoryResponse = await fetch(`${gatewayBase}/v1/categories?page=1&pageSize=200`)
        if (categoryResponse.ok) {
          const categoryPayload = await categoryResponse.json() as any
          const categoryList = categoryPayload?.categories || categoryPayload?.items || categoryPayload?.data?.categories || []

          for (const item of categoryList as Array<any>) {
            const name = item?.name
            if (typeof name === 'string' && name.trim()) {
              ctx.routes.add(`/categories/${encodeURIComponent(name.trim())}`)
            }
          }
        }

        const tagResponse = await fetch(`${gatewayBase}/v1/tags?page=1&pageSize=200`)
        if (tagResponse.ok) {
          const tagPayload = await tagResponse.json() as any
          const tagList = tagPayload?.tags || tagPayload?.items || tagPayload?.data?.tags || []

          for (const item of tagList as Array<any>) {
            const name = item?.name
            if (typeof name === 'string' && name.trim()) {
              ctx.routes.add(`/tags/${encodeURIComponent(name.trim())}`)
            }
          }
        }

        const response = await fetch(`${gatewayBase}/v1/posts?page=1&pageSize=1000`)
        if (!response.ok) {
          console.warn(`[ssg] skip dynamic post prerender: ${response.status} ${response.statusText}`)
          return
        }

        const payload = await response.json() as any
        const list = payload?.posts || payload?.items || payload?.data?.posts || []
        const slugs = new Set<string>()

        for (const item of list as Array<any>) {
          const slug = item?.slug || item?.info?.slug
          const authorId = item?.author?.id || item?.info?.author?.id
          if (typeof slug === 'string' && slug.trim()) {
            slugs.add(slug.trim())
          }
          if (authorId !== undefined && authorId !== null && String(authorId).trim()) {
            ctx.routes.add(`/authors/${encodeURIComponent(String(authorId).trim())}`)
          }
        }

        for (const slug of slugs) {
          ctx.routes.add(`/posts/${encodeURIComponent(slug)}`)
        }
      } catch (error) {
        console.warn('[ssg] skip dynamic prerender discovery, failed to fetch data from gateway:', error)
      }
    }
  },

  compatibilityDate: '2025-01-15',

  hub: {
    blob: true
  },

  vite: {
    optimizeDeps: {
      include: [
        '@nuxt/ui > prosemirror-state',
        'yjs',
        'y-partykit/provider',
        '@vue/devtools-core',
        '@vue/devtools-kit',
        '@tiptap/extension-list',
        '@tiptap/extension-table',
        '@tiptap/pm/tables',
        'tiptap-extension-code-block-shiki',
        '@tiptap/extension-emoji',
        '@ai-sdk/vue',
        '@tiptap/extension-collaboration',
        '@tiptap/extension-collaboration-caret',
        '@vueuse/core',
        '@tiptap/core',
        '@tiptap/vue-3',
        '@tiptap/pm/view',
        '@tiptap/pm/state'
      ]
    }
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
