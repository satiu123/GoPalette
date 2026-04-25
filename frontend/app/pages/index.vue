<script setup lang="ts">
import { fetchPosts, fetchTags, isVisiblePostStatus } from '~/composables/useBlogApi'

const { data: postsData } = await useAsyncData('home-posts', () => fetchPosts(1, 60))
const { data: tagsData } = await useAsyncData('home-tags', () => fetchTags(1, 200))

const allPosts = computed(() => postsData.value?.posts || [])
const publishedPosts = computed(() => {
  const visible = allPosts.value.filter(post => isVisiblePostStatus(post.status))
  return visible.length > 0 ? visible : allPosts.value
})
const featuredPosts = computed(() => publishedPosts.value.slice(0, 2))
const latestPosts = computed(() => publishedPosts.value.slice(0, 6))
const tags = computed(() => (tagsData.value?.tags || []).map(tag => tag.name))

const statItems = computed(() => [
  {
    label: '文章总数',
    value: publishedPosts.value.length
  },
  {
    label: '专题标签',
    value: tags.value.length
  },
  {
    label: '平均阅读',
    value: `${Math.max(1, Math.round((publishedPosts.value.reduce((sum, post) => sum + post.readingMinutes, 0)) / Math.max(1, publishedPosts.value.length)))} 分钟`
  }
])

useSeoMeta({
  title: 'GoPalette Blog',
  description: 'GoPalette 博客首页，展示最新文章、专题标签与写作入口。',
  ogTitle: 'GoPalette Blog',
  ogDescription: 'GoPalette 博客首页，展示最新文章、专题标签与写作入口。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton to="/write" icon="i-lucide-pen-line" size="sm" class="sm:hidden" />
      <UButton
        to="/write"
        icon="i-lucide-pen-line"
        label="开始写作"
        size="sm"
        class="hidden sm:inline-flex"
      />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="motion-fade-up rounded-2xl border border-default bg-muted/30 p-6 sm:p-10">
        <div class="grid gap-8 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div class="space-y-5">
            <UBadge
              color="primary"
              variant="subtle"
              label="GoPalette Blog"
            />

            <h1 class="max-w-2xl text-3xl font-semibold tracking-tight text-highlighted sm:text-5xl">
              构建、记录、分享你的工程思考
            </h1>

            <p class="max-w-2xl text-base text-muted sm:text-lg">
              在这里整理你的想法、沉淀技术实践，并把它们发布成清晰可读的文章。
            </p>

            <div class="flex flex-wrap gap-3">
              <UButton
                to="/posts"
                trailing-icon="i-lucide-arrow-right"
                label="浏览文章"
              />
              <UButton
                to="/write"
                color="neutral"
                variant="subtle"
                icon="i-lucide-square-pen"
                label="进入编辑器"
              />
            </div>
          </div>

          <UCard class="motion-card motion-panel" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <p class="text-sm font-medium text-toned">
                站点概览
              </p>
            </template>

            <div class="space-y-4">
              <div
                v-for="item in statItems"
                :key="item.label"
                class="flex items-center justify-between rounded-lg border border-default px-3 py-2.5"
              >
                <span class="text-sm text-toned">{{ item.label }}</span>
                <span class="text-sm font-semibold text-highlighted">{{ item.value }}</span>
              </div>
            </div>
          </UCard>
        </div>
      </section>

      <section class="motion-fade-up motion-delay-1 mt-12">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-highlighted">
            精选文章
          </h2>
          <UButton
            to="/posts"
            size="xs"
            color="neutral"
            variant="ghost"
            trailing-icon="i-lucide-arrow-right"
            label="查看全部"
          />
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <UCard
            v-for="post in featuredPosts"
            :key="post.id"
            :ui="{ body: 'p-0' }"
            class="motion-card motion-panel overflow-hidden"
          >
            <img
              :src="post.cover"
              :alt="post.title"
              class="h-44 w-full object-cover"
            >

            <div class="space-y-4 p-5">
              <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
                <UBadge
                  :label="post.category"
                  color="primary"
                  variant="subtle"
                />
                <span>{{ post.publishedAt }}</span>
                <span>·</span>
                <span>{{ post.readingMinutes }} 分钟</span>
              </div>

              <div class="space-y-2">
                <NuxtLink
                  :to="`/posts/${post.slug}`"
                  class="motion-link block text-lg font-semibold text-highlighted hover:text-primary"
                >
                  {{ post.title }}
                </NuxtLink>
                <p class="line-clamp-2 text-sm text-toned">
                  {{ post.summary }}
                </p>
              </div>

              <div class="flex flex-wrap gap-2">
                <UBadge
                  v-for="tag in post.tags"
                  :key="tag"
                  :label="`#${tag}`"
                  color="neutral"
                  variant="outline"
                />
              </div>
            </div>
          </UCard>
        </div>
      </section>

      <section class="motion-fade-up motion-delay-2 mt-12 grid gap-8 lg:grid-cols-[minmax(0,1fr)_260px]">
        <div>
          <h2 class="mb-5 text-xl font-semibold text-highlighted">
            最新发布
          </h2>

          <div class="space-y-3">
            <NuxtLink
              v-for="post in latestPosts"
              :key="post.id"
              :to="`/posts/${post.slug}`"
              class="motion-card motion-panel block rounded-xl border border-default bg-default p-4 hover:border-primary/40"
            >
              <div class="mb-2 flex items-center gap-2 text-xs text-toned">
                <span>{{ post.publishedAt }}</span>
                <span>·</span>
                <span>{{ post.author }}</span>
              </div>
              <h3 class="text-base font-semibold text-highlighted">
                {{ post.title }}
              </h3>
              <p class="mt-1 line-clamp-2 text-sm text-toned">
                {{ post.summary }}
              </p>
            </NuxtLink>
          </div>
        </div>

        <aside class="space-y-4">
          <h2 class="text-xl font-semibold text-highlighted">
            热门标签
          </h2>

          <div class="flex flex-wrap gap-2">
            <NuxtLink
              v-for="tag in tags"
              :key="tag"
              :to="`/posts?tag=${encodeURIComponent(tag)}`"
              class="inline-flex"
            >
              <UBadge
                :label="`#${tag}`"
                color="neutral"
                variant="outline"
                class="hover:border-primary hover:text-primary"
              />
            </NuxtLink>
          </div>
        </aside>
      </section>
    </main>
  </div>
</template>
