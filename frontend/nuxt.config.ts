// https://nuxt.com/docs/api/configuration/nuxt-config
const gatewayBase = import.meta.env.NUXT_GATEWAY_BASE || 'http://127.0.0.1:8080'

export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxthub/core',
    'nuxt-csurf'
  ],

  devtools: {
    enabled: true
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
    public: {
      partykitHost: ''
    }
  },

  nitro: {
    prerender: {
      crawlLinks: true,
      routes: ['/', '/posts']
    }
  },

  hooks: {
    async 'prerender:routes'(ctx) {
      try {
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
          if (typeof slug === 'string' && slug.trim()) {
            slugs.add(slug.trim())
          }
        }

        for (const slug of slugs) {
          ctx.routes.add(`/posts/${encodeURIComponent(slug)}`)
        }
      } catch (error) {
        console.warn('[ssg] skip dynamic post prerender, failed to fetch slugs from gateway:', error)
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
        'y-partykit/provider'
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
