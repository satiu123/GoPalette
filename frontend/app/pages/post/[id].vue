<script setup lang="ts">
import { articleImageUrl } from '~/composables/useBlogData'

const route = useRoute()
const config = useRuntimeConfig()

const { data, error } = await useArticle(route.params.id as string)

if (error.value || !data.value?.data) {
  await navigateTo('/')
}

const post = computed(() => data.value?.data ?? null)

const siteName = 'GoPalette'
const siteUrl = computed(() => String(config.public.siteUrl || 'http://localhost:3000').replace(/\/$/, ''))
const canonicalUrl = computed(() => {
  const slugOrID = route.params.id ? String(route.params.id) : ''
  return `${siteUrl.value}/post/${encodeURIComponent(slugOrID)}`
})
const plainDescription = computed(() => {
  const summary = post.value?.summary?.trim()
  if (summary) return summary.slice(0, 160)
  const content = (post.value?.content ?? '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
  return content.slice(0, 160)
})
const shareImage = computed(() => post.value ? articleImageUrl(post.value) : `${siteUrl.value}/og-default.jpg`)

useSeoMeta({
  title: () => post.value ? `${post.value.title} | ${siteName}` : siteName,
  description: () => plainDescription.value,
  ogTitle: () => post.value?.title ?? siteName,
  ogDescription: () => plainDescription.value,
  ogType: 'article',
  ogUrl: () => canonicalUrl.value,
  ogImage: () => shareImage.value,
  twitterCard: 'summary_large_image',
  twitterTitle: () => post.value?.title ?? siteName,
  twitterDescription: () => plainDescription.value,
  twitterImage: () => shareImage.value
})

useHead({
  link: [{ rel: 'canonical', href: canonicalUrl.value }],
  meta: post.value
    ? [
        { property: 'article:published_time', content: post.value.created_at },
        { property: 'article:modified_time', content: post.value.updated_at }
      ]
    : []
})
</script>

<template>
  <div v-if="post">
    <PostDetail :post="post" />
  </div>
</template>
