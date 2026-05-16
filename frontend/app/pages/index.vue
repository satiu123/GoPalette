<script setup lang="ts">
import { fetchPosts, fetchTags, isVisiblePostStatus } from '~/composables/useBlogApi'

const { buildUrl } = useSiteSeo()
const { categoryPath, tagPath } = useBlogRoutes()

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

useSeoMeta({
  title: 'GoPalette Blog',
  description: 'GoPalette 博客首页，展示最新文章、专题标签与写作入口。',
  ogTitle: 'GoPalette Blog',
  ogDescription: 'GoPalette 博客首页，展示最新文章、专题标签与写作入口。'
})

useHead({
  link: [
    {
      rel: 'canonical',
      href: buildUrl('/')
    }
  ]
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton
        to="/write"
        icon="i-lucide-pen-line"
        size="sm"
        class="sm:hidden"
      />
      <UButton
        to="/write"
        icon="i-lucide-pen-line"
        label="开始写作"
        size="sm"
        class="hidden sm:inline-flex"
      />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="motion-fade-up motion-delay-1">
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
                <NuxtLink
                  :to="categoryPath(post.category)"
                  class="inline-flex"
                >
                  <UBadge
                    :label="post.category"
                    color="primary"
                    variant="subtle"
                    class="hover:opacity-90"
                  />
                </NuxtLink>
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
                <NuxtLink
                  v-for="tag in post.tags"
                  :key="tag"
                  :to="tagPath(tag)"
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
            <article
              v-for="post in latestPosts"
              :key="post.id"
              class="motion-card motion-panel overflow-hidden rounded-xl border border-default bg-default hover:border-primary/40"
            >
              <div class="grid gap-4 p-4 sm:grid-cols-[160px_minmax(0,1fr)]">
                <NuxtLink
                  :to="`/posts/${post.slug}`"
                  class="block overflow-hidden rounded-lg border border-default bg-muted"
                >
                  <img
                    :src="post.cover"
                    :alt="post.title"
                    class="aspect-[16/10] h-full w-full object-cover transition duration-300 hover:scale-105"
                  >
                </NuxtLink>

                <div class="min-w-0">
                  <div class="mb-2 flex flex-wrap items-center gap-2 text-xs text-toned">
                    <span>{{ post.publishedAt }}</span>
                    <span>·</span>
                    <NuxtLink
                      :to="categoryPath(post.category)"
                      class="inline-flex"
                    >
                      <UBadge
                        :label="post.category"
                        color="primary"
                        variant="subtle"
                        class="hover:opacity-90"
                      />
                    </NuxtLink>
                    <span>·</span>
                    <NuxtLink
                      v-if="post.authorId"
                      :to="`/authors/${post.authorId}`"
                      class="hover:text-primary"
                    >
                      {{ post.author }}
                    </NuxtLink>
                    <span v-else>{{ post.author }}</span>
                  </div>
                  <NuxtLink
                    :to="`/posts/${post.slug}`"
                    class="line-clamp-2 text-base font-semibold text-highlighted hover:text-primary"
                  >
                    {{ post.title }}
                  </NuxtLink>
                  <p class="mt-2 line-clamp-2 text-sm text-toned">
                    {{ post.summary }}
                  </p>
                </div>
              </div>
            </article>
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
              :to="tagPath(tag)"
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
