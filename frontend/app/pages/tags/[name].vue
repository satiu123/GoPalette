<script setup lang="ts">
import { fetchPosts, fetchTags, isVisiblePostStatus } from '~/composables/useBlogApi'

const route = useRoute()
const { buildUrl } = useSiteSeo()
const { tagPath } = useBlogRoutes()

const tagName = computed(() => decodeURIComponent(String(route.params.name || '')).trim())
const currentPage = ref(1)
const pageSize = 8

const { data: tagsData } = await useAsyncData('tag-page-tags', () => fetchTags(1, 200))
const { data: postsData } = await useAsyncData('tag-page-posts', () => fetchPosts(1, 300))

const tags = computed(() => tagsData.value?.tags || [])
const currentTag = computed(() => tags.value.find(item => item.name === tagName.value) || null)

const visiblePosts = computed(() => {
  const allPosts = postsData.value?.posts || []
  return allPosts
    .filter(post => isVisiblePostStatus(post.status))
    .filter(post => post.tags.includes(tagName.value))
})

const totalPages = computed(() => Math.max(1, Math.ceil(visiblePosts.value.length / pageSize)))
const paginatedPosts = computed(() => {
  const page = Math.min(currentPage.value, totalPages.value)
  const start = (page - 1) * pageSize
  return visiblePosts.value.slice(start, start + pageSize)
})

const siblingTags = computed(() => tags.value
  .filter(item => item.name !== tagName.value)
  .slice(0, 16))

watch(() => route.query.page, (value) => {
  const page = Number(value || 1)
  currentPage.value = Number.isFinite(page) && page > 0 ? page : 1
}, { immediate: true })

watch(totalPages, (value) => {
  if (currentPage.value > value) {
    currentPage.value = value
  }
})

if (!currentTag.value && tagName.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '标签不存在'
  })
}

async function goPage(page: number) {
  await navigateTo({
    query: page > 1 ? { page: String(page) } : {}
  })
}

const canonicalUrl = computed(() => {
  const suffix = currentPage.value > 1 ? `?page=${currentPage.value}` : ''
  return buildUrl(`${tagPath(tagName.value)}${suffix}`)
})

useSeoMeta({
  title: computed(() => `#${tagName.value || '标签'} - GoPalette`),
  description: computed(() => `标签 #${tagName.value || '标签'} 相关文章归档，共 ${visiblePosts.value.length} 篇。`)
})

useHead({
  link: [
    {
      rel: 'canonical',
      href: canonicalUrl
    }
  ]
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton
        to="/posts"
        icon="i-lucide-book-open"
        size="sm"
        class="sm:hidden"
      />
      <UButton
        to="/posts"
        icon="i-lucide-book-open"
        label="文章归档"
        size="sm"
        class="hidden sm:inline-flex"
      />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="motion-fade-up rounded-2xl border border-default bg-muted/30 p-6 sm:p-8">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div class="space-y-3">
            <UBadge
              color="primary"
              variant="subtle"
              :label="`标签：#${tagName}`"
            />
            <div>
              <h1 class="text-2xl font-semibold text-highlighted sm:text-4xl">
                #{{ tagName }}
              </h1>
              <p class="mt-2 text-sm text-toned sm:text-base">
                共 {{ visiblePosts.length }} 篇公开文章，当前第 {{ currentPage }} / {{ totalPages }} 页。
              </p>
            </div>
          </div>

          <UButton
            to="/posts"
            color="neutral"
            variant="soft"
            trailing-icon="i-lucide-arrow-right"
            label="查看全部文章"
          />
        </div>
      </section>

      <section
        v-if="siblingTags.length"
        class="motion-fade-up motion-delay-1 mt-6 flex flex-wrap gap-2"
      >
        <NuxtLink
          v-for="item in siblingTags"
          :key="item.id"
          :to="tagPath(item.name)"
          class="inline-flex"
        >
          <UBadge
            :label="`#${item.name}`"
            color="neutral"
            variant="outline"
            class="hover:border-primary hover:text-primary"
          />
        </NuxtLink>
      </section>

      <section
        v-if="paginatedPosts.length"
        class="motion-fade-up motion-delay-2 mt-8 grid gap-4 md:grid-cols-2"
      >
        <UCard
          v-for="post in paginatedPosts"
          :key="post.id"
          :ui="{ body: 'p-0' }"
          class="motion-card motion-panel overflow-hidden"
        >
          <img
            :src="post.cover"
            :alt="post.title"
            class="h-40 w-full object-cover"
          >

          <div class="space-y-3 p-5">
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

            <NuxtLink
              :to="`/posts/${post.slug}`"
              class="block text-lg font-semibold text-highlighted hover:text-primary"
            >
              {{ post.title }}
            </NuxtLink>

            <p class="line-clamp-2 text-sm text-toned">
              {{ post.summary }}
            </p>

            <div class="flex flex-wrap gap-2">
              <NuxtLink
                v-for="tag in post.tags"
                :key="`${post.id}-${tag}`"
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
      </section>

      <section
        v-else
        class="motion-fade-up motion-delay-2 mt-8"
      >
        <UCard>
          <div class="py-10 text-center">
            <p class="text-base font-medium text-highlighted">
              这个标签下还没有公开文章
            </p>
            <p class="mt-2 text-sm text-toned">
              可以先去文章归档看看其他内容。
            </p>
          </div>
        </UCard>
      </section>

      <section
        v-if="totalPages > 1"
        class="mt-8 flex items-center justify-between"
      >
        <UButton
          icon="i-lucide-chevron-left"
          label="上一页"
          color="neutral"
          variant="ghost"
          :disabled="currentPage <= 1"
          @click="goPage(currentPage - 1)"
        />

        <p class="text-sm text-toned">
          第 {{ currentPage }} / {{ totalPages }} 页
        </p>

        <UButton
          trailing-icon="i-lucide-chevron-right"
          label="下一页"
          color="neutral"
          variant="ghost"
          :disabled="currentPage >= totalPages"
          @click="goPage(currentPage + 1)"
        />
      </section>
    </main>
  </div>
</template>
