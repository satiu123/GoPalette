<script setup lang="ts">
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { POST_STATUS_PUBLISHED, fetchPostBySlug, fetchPosts } from '~/composables/useBlogApi'

const { extension: Emoji } = useEditorEmojis()

const route = useRoute()
const slug = computed(() => String(route.params.slug || ''))

const { data: postData } = await useAsyncData(
  () => `post-${slug.value}`,
  () => fetchPostBySlug(slug.value),
  {
    watch: [slug]
  }
)

if (!postData.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '文章不存在'
  })
}

const post = computed(() => postData.value)

const { data: relatedData } = await useAsyncData(
  () => `related-${slug.value}`,
  () => fetchPosts(1, 40),
  {
    watch: [slug]
  }
)

const relatedPosts = computed(() => {
  const current = post.value
  if (!current) return []

  return (relatedData.value?.posts || [])
    .filter(item => item.status === POST_STATUS_PUBLISHED)
    .filter(item => item.slug !== current.slug)
    .filter(item => item.categoryId === current.categoryId || item.category === current.category)
    .slice(0, 3)
})

const viewerExtensions = [
  CodeBlockShiki.configure({
    defaultTheme: 'material-theme',
    themes: {
      light: 'material-theme-lighter',
      dark: 'material-theme-palenight'
    }
  }),
  Emoji,
  TableKit,
  TaskList,
  TaskItem
]

const seoTitle = computed(() => post.value?.title || '文章详情')
const seoDescription = computed(() => post.value?.summary || '')
const seoImage = computed(() => post.value?.cover || '')

useSeoMeta({
  title: seoTitle,
  description: seoDescription,
  ogTitle: seoTitle,
  ogDescription: seoDescription,
  ogImage: seoImage,
  twitterCard: 'summary_large_image'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton to="/write" icon="i-lucide-square-pen" label="继续写作" size="sm" />
    </AppHeader>

    <main v-if="post" class="mx-auto w-full max-w-5xl px-4 pb-20 pt-10 sm:px-14">
      <article class="rounded-2xl border border-default bg-default p-6 sm:p-10">
        <div class="space-y-5 border-b border-default pb-8">
          <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
            <UBadge :label="post.category" color="primary" variant="subtle" />
            <span>{{ post.publishedAt }}</span>
            <span>·</span>
            <span>{{ post.readingMinutes }} 分钟</span>
            <span>·</span>
            <span>{{ post.author }}</span>
          </div>

          <h1 class="text-3xl font-semibold tracking-tight text-highlighted sm:text-4xl">
            {{ post.title }}
          </h1>

          <p class="text-base text-toned sm:text-lg">
            {{ post.summary }}
          </p>

          <img :src="post.cover" :alt="post.title" class="h-60 w-full rounded-xl object-cover">
        </div>

        <div class="mt-8">
          <p v-if="!post.content?.trim()" class="leading-7 text-toned">
            暂无正文内容。
          </p>

          <UEditor v-else :model-value="post.content" content-type="markdown" :editable="false"
            :extensions="viewerExtensions" class="min-h-0" :ui="{
              base: 'p-4 sm:p-14',
              content: 'max-w-4xl mx-auto'
            }" />
        </div>

        <div class="mt-10 flex flex-wrap gap-2 border-t border-default pt-6">
          <UBadge v-for="tag in post.tags" :key="tag" :label="`#${tag}`" color="neutral" variant="outline" />
        </div>
      </article>

      <section class="mt-10">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-highlighted">
            相关文章
          </h2>
          <UButton to="/posts" size="xs" color="neutral" variant="ghost" trailing-icon="i-lucide-arrow-right"
            label="返回列表" />
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <NuxtLink v-for="item in relatedPosts" :key="item.id" :to="`/posts/${item.slug}`"
            class="rounded-xl border border-default bg-default p-4 transition-all hover:-translate-y-0.5 hover:border-primary/40">
            <p class="text-xs text-toned">
              {{ item.publishedAt }}
            </p>
            <p class="mt-2 line-clamp-2 text-sm font-semibold text-highlighted">
              {{ item.title }}
            </p>
          </NuxtLink>
        </div>
      </section>
    </main>
  </div>
</template>
