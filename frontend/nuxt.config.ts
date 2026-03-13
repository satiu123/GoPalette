import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ['@vueuse/motion/nuxt'],
  css: ['~/assets/css/main.css'],
  vite: {
    // @ts-expect-error Nuxt/Vite 类型冲突（运行时可用）
    plugins: [tailwindcss()]
  },
  runtimeConfig: {
    public: {
      apiBase: '/api'
    }
  },
  routeRules: {
    '/api/**': { proxy: 'http://localhost:8080/api/**' },
    '/static/**': { proxy: 'http://localhost:8080/static/**' }
  }
})
