<script setup lang="ts">
import { isVisiblePostStatus, type BlogPostItem } from '~/composables/useBlogApi'

interface AuthorPageInfo {
  id: string
  username: string
  email: string
  avatarURL: string
  bio: string
  location: string
  createdAt: string
}

interface AuthorPageStats {
  posts: number
  published: number
  drafts: number
  archived: number
  views: number
  likes: number
  comments: number
}

interface AuthorPageResponse {
  userInfo: AuthorPageInfo | null
  postStats: AuthorPageStats
  topPosts: BlogPostItem[]
  authorPosts: BlogPostItem[]
}

type DataRecord = Record<string, unknown>

const route = useRoute()
const authorId = computed(() => String(route.params.id || ''))
const { buildUrl } = useSiteSeo()
const { categoryPath } = useBlogRoutes()

function toRecord(value: unknown): DataRecord | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
  return value as DataRecord
}

function toRecordArray(value: unknown) {
  if (!Array.isArray(value)) return [] as DataRecord[]
  return value.map(item => toRecord(item)).filter((item): item is DataRecord => Boolean(item))
}

function pick(raw: DataRecord | undefined, keys: string[]) {
  if (!raw) return undefined
  for (const key of keys) {
    const value = raw[key]
    if (value !== undefined && value !== null && value !== '') {
      return value
    }
  }
  return undefined
}

function toText(value: unknown, fallback = '') {
  if (value === undefined || value === null) return fallback
  const text = String(value).trim()
  return text || fallback
}

function toNumber(value: unknown, fallback = 0) {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : fallback
}

function formatDate(input: unknown) {
  if (!input) return '未知时间'

  let date: Date | null = null

  if (typeof input === 'string') {
    const parsed = new Date(input)
    if (!Number.isNaN(parsed.getTime())) {
      date = parsed
    }
  } else if (typeof input === 'object' && input) {
    const seconds = toNumber((input as Record<string, unknown>).seconds, NaN)
    if (Number.isFinite(seconds)) {
      date = new Date(seconds * 1000)
    }
  }

  if (!date || Number.isNaN(date.getTime())) return '未知时间'

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(date)
}

function formatCount(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value)
}

function toAvatarFallback(name: string) {
  const text = name.trim()
  if (!text) return '作'
  return text.slice(0, 1).toUpperCase()
}

function normalizePost(input: DataRecord): BlogPostItem {
  const author = toRecord(input.author)
  const category = toRecord(input.category)
  const createdAt = toText(pick(input, ['createdAt', 'created_at']))

  return {
    id: toText(pick(input, ['id'])),
    title: toText(pick(input, ['title']), '未命名文章'),
    summary: toText(pick(input, ['summary']), '暂无摘要'),
    slug: toText(pick(input, ['slug'])),
    status: toNumber(pick(input, ['status'])),
    tags: Array.isArray(input.tags) ? input.tags.map(tag => String(tag)) : [],
    category: toText(pick(category, ['name']), '未分类'),
    categoryId: toText(pick(category, ['id'])),
    author: toText(pick(author, ['name']), '匿名作者'),
    authorId: toText(pick(author, ['id'])),
    publishedAt: formatDate(createdAt),
    readingMinutes: Math.max(1, Math.ceil(toText(pick(input, ['summary'])).length / 300)),
    cover: `/covers/${encodeURIComponent(toText(pick(input, ['slug', 'id']), 'gopalette'))}.svg`,
    createdAt
  }
}

function normalizeAuthorPage(raw: DataRecord): AuthorPageResponse {
  const userInfoRaw = toRecord(pick(raw, ['userInfo', 'user_info'])) || {}
  const statsRaw = toRecord(pick(raw, ['postStats', 'post_stats'])) || {}
  const topPostsRaw = toRecordArray(pick(raw, ['topPosts', 'top_posts']))
  const authorPostsRaw = toRecordArray(pick(raw, ['authorPosts', 'author_posts']))

  const userInfo: AuthorPageInfo | null = Object.keys(userInfoRaw).length
    ? {
        id: toText(pick(userInfoRaw, ['id'])),
        username: toText(pick(userInfoRaw, ['username']), '未命名作者'),
        email: toText(pick(userInfoRaw, ['email'])),
        avatarURL: toText(pick(userInfoRaw, ['avatarURL', 'avatarUrl', 'avatar_u_r_l', 'avatar_url'])),
        bio: toText(pick(userInfoRaw, ['bio']), '这位作者还没有留下简介。'),
        location: toText(pick(userInfoRaw, ['location'])),
        createdAt: formatDate(pick(userInfoRaw, ['createdAt', 'created_at']))
      }
    : null

  return {
    userInfo,
    postStats: {
      posts: toNumber(pick(statsRaw, ['posts'])),
      published: toNumber(pick(statsRaw, ['published'])),
      drafts: toNumber(pick(statsRaw, ['drafts'])),
      archived: toNumber(pick(statsRaw, ['archived'])),
      views: toNumber(pick(statsRaw, ['views'])),
      likes: toNumber(pick(statsRaw, ['likes'])),
      comments: toNumber(pick(statsRaw, ['comments']))
    },
    topPosts: topPostsRaw.map(normalizePost).filter(post => isVisiblePostStatus(post.status)),
    authorPosts: authorPostsRaw.map(normalizePost).filter(post => isVisiblePostStatus(post.status))
  }
}

const { data, error } = await useAsyncData(
  () => `author-page-${authorId.value}`,
  async () => {
    if (!authorId.value) {
      throw createError({ statusCode: 400, statusMessage: '作者 ID 缺失' })
    }

    const response = await $fetch<DataRecord>(`/api/user/full-profile/${authorId.value}`)
    return normalizeAuthorPage(response || {})
  },
  {
    watch: [authorId]
  }
)

if (error.value) {
  throw createError({
    statusCode: Number((error.value as { statusCode?: number })?.statusCode || 500),
    statusMessage: '作者主页加载失败'
  })
}

const author = computed(() => data.value?.userInfo || null)
const topPosts = computed(() => data.value?.topPosts || [])
const authorPosts = computed(() => data.value?.authorPosts || [])
const stats = computed(() => data.value?.postStats || {
  posts: 0,
  published: 0,
  drafts: 0,
  archived: 0,
  views: 0,
  likes: 0,
  comments: 0
})

if (!author.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '作者不存在'
  })
}

const statItems = computed(() => [
  { label: '公开文章', value: formatCount(stats.value.published) },
  { label: '总阅读', value: formatCount(stats.value.views) },
  { label: '总点赞', value: formatCount(stats.value.likes) },
  { label: '总评论', value: formatCount(stats.value.comments) }
])

useSeoMeta({
  title: computed(() => `${author.value?.username || '作者'} - GoPalette`),
  description: computed(() => author.value?.bio || '查看作者主页与文章归档。'),
  ogTitle: computed(() => `${author.value?.username || '作者'} - GoPalette`),
  ogDescription: computed(() => author.value?.bio || '查看作者主页与文章归档。')
})

useHead({
  link: [
    {
      rel: 'canonical',
      href: computed(() => buildUrl(`/authors/${encodeURIComponent(authorId.value)}`))
    }
  ]
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton to="/posts" icon="i-lucide-book-open" size="sm" class="sm:hidden" />
      <UButton to="/posts" icon="i-lucide-book-open" label="浏览文章" size="sm" class="hidden sm:inline-flex" />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="motion-fade-up rounded-2xl border border-default bg-muted/30 p-6 sm:p-8">
        <div class="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
          <div class="flex items-center gap-4">
            <div class="h-20 w-20 overflow-hidden rounded-full border border-default bg-muted">
              <img
                v-if="author?.avatarURL"
                :src="author.avatarURL"
                :alt="author.username"
                class="h-full w-full object-cover"
              >
              <div
                v-else
                class="flex h-full w-full items-center justify-center text-2xl font-semibold text-toned"
              >
                {{ toAvatarFallback(author?.username || '') }}
              </div>
            </div>

            <div class="space-y-2">
              <div>
                <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
                  {{ author?.username }}
                </h1>
                <p class="mt-1 text-sm text-toned">
                  加入时间：{{ author?.createdAt }}
                </p>
              </div>

              <p class="max-w-2xl text-sm text-toned sm:text-base">
                {{ author?.bio }}
              </p>

              <div class="flex flex-wrap gap-2 text-xs text-toned">
                <span v-if="author?.location">{{ author.location }}</span>
                <span v-if="author?.location && author?.email">·</span>
                <span v-if="author?.email">{{ author.email }}</span>
              </div>
            </div>
          </div>

          <UButton to="/posts" color="neutral" variant="soft" trailing-icon="i-lucide-arrow-right" label="查看全部文章" />
        </div>
      </section>

      <section class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <UCard v-for="item in statItems" :key="item.label" :ui="{ body: 'p-4' }">
          <p class="text-xs text-toned">
            {{ item.label }}
          </p>
          <p class="mt-1 text-xl font-semibold text-highlighted">
            {{ item.value }}
          </p>
        </UCard>
      </section>

      <section class="motion-fade-up motion-delay-1 mt-8 grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
        <UCard>
          <template #header>
            <h2 class="text-lg font-semibold text-highlighted">
              精选文章
            </h2>
          </template>

          <div v-if="!topPosts.length" class="text-sm text-toned">
            这位作者还没有公开文章。
          </div>

          <div v-else class="space-y-3">
            <NuxtLink
              v-for="post in topPosts"
              :key="post.id"
              :to="`/posts/${post.slug}`"
              class="motion-card block rounded-xl border border-default bg-default p-4 hover:border-primary/40"
            >
              <p class="line-clamp-2 text-sm font-semibold text-highlighted">
                {{ post.title }}
              </p>
              <p class="mt-1 text-xs text-toned">
                {{ post.publishedAt }} · {{ post.readingMinutes }} 分钟
              </p>
            </NuxtLink>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-highlighted">
                作者文章
              </h2>
              <span class="text-sm text-toned">
                {{ authorPosts.length }} 篇
              </span>
            </div>
          </template>

          <div v-if="!authorPosts.length" class="text-sm text-toned">
            暂无可展示文章。
          </div>

          <div v-else class="space-y-3">
            <article
              v-for="post in authorPosts"
              :key="post.id"
              class="rounded-xl border border-default p-4"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-2">
                  <NuxtLink :to="`/posts/${post.slug}`" class="text-base font-semibold text-highlighted hover:text-primary">
                    {{ post.title }}
                  </NuxtLink>
                  <p class="line-clamp-2 text-sm text-toned">
                    {{ post.summary }}
                  </p>
                  <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
                    <NuxtLink :to="categoryPath(post.category)" class="inline-flex">
                      <UBadge :label="post.category" color="primary" variant="subtle" class="hover:opacity-90" />
                    </NuxtLink>
                    <span>{{ post.publishedAt }}</span>
                    <span>·</span>
                    <span>{{ post.readingMinutes }} 分钟</span>
                  </div>
                </div>

                <div class="flex flex-wrap gap-2">
                  <UBadge
                    v-for="tag in post.tags"
                    :key="`${post.id}-${tag}`"
                    :label="`#${tag}`"
                    color="neutral"
                    variant="outline"
                  />
                </div>
              </div>
            </article>
          </div>
        </UCard>
      </section>
    </main>
  </div>
</template>
