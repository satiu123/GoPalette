// https://nuxt.com/docs/api/configuration/nuxt-config
const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env || {}
const prerenderGatewayBase = env.NUXT_GATEWAY_BASE || 'http://127.0.0.1:8080'
const enablePrerender = env.NUXT_ENABLE_PRERENDER === 'true'

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' ? value as Record<string, unknown> : {}
}

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
    gatewayBase: 'http://127.0.0.1:8080',
    deepseekApiKey: '',
    authCookieSecure: '',
    public: {
      partykitHost: '',
      siteUrl: 'http://127.0.0.1:3000',
      siteName: 'GoPalette Blog'
    }
  },

  sourcemap: {
    client: false,
    server: false
  },

  compatibilityDate: '2025-01-15',

  nitro: {
    sourceMap: false,
    prerender: {
      crawlLinks: enablePrerender,
      failOnError: false,
      routes: enablePrerender ? ['/', '/posts', '/rss.xml', '/sitemap.xml'] : []
    }
  },

  hub: {
    blob: true
  },

  vite: {
    build: {
      sourcemap: false,
      rollupOptions: {
        onLog(level, log, handler) {
          const message = typeof log === 'string' ? log : log.message
          const isSourcemapNoise = level === 'warn'
            && message.includes('Sourcemap is likely to be incorrect')
            && (message.includes('@tailwindcss/vite:generate:build')
              || message.includes('nuxt:module-preload-polyfill'))
          if (isSourcemapNoise) {
            return
          }
          handler(level, log)
        }
      }
    },
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

  hooks: {
    async 'prerender:routes'(ctx) {
      if (!enablePrerender) {
        return
      }

      try {
        const categoryResponse = await fetch(`${prerenderGatewayBase}/v1/categories?page=1&pageSize=200`)
        if (categoryResponse.ok) {
          const categoryPayload = asRecord(await categoryResponse.json())
          const categoryData = asRecord(categoryPayload.data)
          const categoryList = (categoryPayload.categories || categoryPayload.items || categoryData.categories || []) as unknown[]

          for (const item of categoryList) {
            const name = asRecord(item).name
            if (typeof name === 'string' && name.trim()) {
              ctx.routes.add(`/categories/${encodeURIComponent(name.trim())}`)
            }
          }
        }

        const tagResponse = await fetch(`${prerenderGatewayBase}/v1/tags?page=1&pageSize=200`)
        if (tagResponse.ok) {
          const tagPayload = asRecord(await tagResponse.json())
          const tagData = asRecord(tagPayload.data)
          const tagList = (tagPayload.tags || tagPayload.items || tagData.tags || []) as unknown[]

          for (const item of tagList) {
            const name = asRecord(item).name
            if (typeof name === 'string' && name.trim()) {
              ctx.routes.add(`/tags/${encodeURIComponent(name.trim())}`)
            }
          }
        }

        const response = await fetch(`${prerenderGatewayBase}/v1/posts?page=1&pageSize=1000`)
        if (!response.ok) {
          console.warn(`[ssg] skip dynamic post prerender: ${response.status} ${response.statusText}`)
          return
        }

        const payload = asRecord(await response.json())
        const payloadData = asRecord(payload.data)
        const list = (payload.posts || payload.items || payloadData.posts || []) as unknown[]
        const slugs = new Set<string>()

        for (const item of list) {
          const itemRecord = asRecord(item)
          const itemInfo = asRecord(itemRecord.info)
          const itemAuthor = asRecord(itemRecord.author)
          const infoAuthor = asRecord(itemInfo.author)
          const slug = itemRecord.slug || itemInfo.slug
          const authorId = itemAuthor.id || infoAuthor.id
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

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
