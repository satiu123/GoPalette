<script setup>
const config = useRuntimeConfig()
const siteName = config.public.siteName || 'GoPalette Blog'
const siteUrl = (config.public.siteUrl || 'http://127.0.0.1:3000').replace(/\/+$/, '')

useHead({
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' }
  ],
  link: [
    { rel: 'icon', href: '/favicon.ico' },
    { rel: 'alternate', type: 'application/rss+xml', title: `${siteName} RSS`, href: `${siteUrl}/rss.xml` }
  ],
  htmlAttrs: {
    lang: 'zh-CN'
  }
})

const title = siteName
const description = 'GoPalette 博客前端，包含文章展示与富文本写作工作台。'

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description,
  ogSiteName: siteName,
  ogImage: `${siteUrl}/og-image.png`,
  twitterCard: 'summary_large_image'
})
</script>

<template>
  <UApp>
    <NuxtPage
      :transition="{
        name: 'page',
        mode: 'out-in'
      }"
    />
  </UApp>
</template>
