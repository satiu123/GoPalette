<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { fetchPosts, fetchTags, isVisiblePostStatus, searchPosts } from '~/composables/useBlogApi'

const route = useRoute()
const router = useRouter()

const selectedTag = ref(typeof route.query.tag === 'string' ? route.query.tag : '')
const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const currentPage = ref(Math.max(1, Number(route.query.page || 1)))
const pageSize = 8
const searchLoading = ref(false)
let searchSeq = 0
const searchState = ref<{ items: Array<{ id: string; title: string; summary: string; slug: string; category: string; tags: string[]; publishedAt: string; readingMinutes: number; cover: string }>; total: number; totalPages: number }>({
  items: [],
  total: 0,
  totalPages: 0
})

const { data: postsData } = await useAsyncData('all-posts-for-list', () => fetchPosts(1, 200))
const { data: tagsData } = await useAsyncData('all-tags-for-list', () => fetchTags(1, 200))

const tags = computed(() => (tagsData.value?.tags || []).map(tag => tag.name))

function toDisplayDate(input?: string) {
  if (!input) return '未知时间'

  const date = new Date(input)
  if (Number.isNaN(date.getTime())) return '未知时间'

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(date)
}

function toCover(seed: string) {
  return `https://picsum.photos/seed/${encodeURIComponent(seed || 'gopalette')}/1200/640`
}

function renderHighlightedText(input: string) {
  const source = String(input || '')
  const marked = source
    .replace(/<em>/gi, '[[EM_OPEN]]')
    .replace(/<\/em>/gi, '[[EM_CLOSE]]')

  const escaped = marked
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

  return escaped
    .replace(/\[\[EM_OPEN\]\]/g, '<mark class="rounded bg-primary/20 px-0.5 text-highlighted">')
    .replace(/\[\[EM_CLOSE\]\]/g, '</mark>')
}

async function runSearch() {
  const query = keyword.value.trim()
  if (!query) {
    searchState.value = { items: [], total: 0, totalPages: 0 }
    return
  }

  const seq = ++searchSeq
  searchLoading.value = true
  try {
    const response = await searchPosts(query, currentPage.value, pageSize)

    if (seq !== searchSeq) return

    searchState.value = {
      items: response.results.map(item => ({
        id: item.id,
        title: item.title,
        summary: item.summary,
        slug: item.slug,
        category: item.categoryName,
        tags: item.tags,
        publishedAt: toDisplayDate(item.createdAt),
        readingMinutes: Math.max(1, Math.ceil(item.summary.length / 300)),
        cover: toCover(item.slug || item.id)
      })),
      total: response.total,
      totalPages: response.totalPages
    }
  } finally {
    if (seq === searchSeq) {
      searchLoading.value = false
    }
  }
}

const triggerSearch = useDebounceFn(() => {
  void runSearch()
}, 250)

watch([keyword, currentPage], () => {
  triggerSearch()
}, { immediate: true })

const filteredPosts = computed(() => {
  if (keyword.value.trim()) {
    const items = searchState.value.items
    return selectedTag.value ? items.filter(item => item.tags.includes(selectedTag.value)) : items
  }

  const allPosts = postsData.value?.posts || []
  const visiblePosts = allPosts.filter(post => isVisiblePostStatus(post.status))
  const posts = visiblePosts.length > 0 ? visiblePosts : allPosts

  return posts.filter((post) => {
    const byTag = selectedTag.value ? post.tags.includes(selectedTag.value) : true
    const byKeyword = keyword.value
      ? [post.title, post.summary, post.category, post.author, ...post.tags].join(' ').toLowerCase().includes(keyword.value.toLowerCase())
      : true

    return byTag && byKeyword
  })
})

const totalPages = computed(() => {
  if (keyword.value.trim()) {
    return Math.max(1, searchState.value.totalPages || 1)
  }

  return Math.max(1, Math.ceil(filteredPosts.value.length / pageSize))
})
const paginatedPosts = computed(() => {
  if (keyword.value.trim()) {
    return filteredPosts.value
  }

  const page = Math.min(currentPage.value, totalPages.value)
  const start = (page - 1) * pageSize

  return filteredPosts.value.slice(start, start + pageSize)
})

function syncQuery() {
  const query = {
    ...route.query,
    tag: selectedTag.value || undefined,
    q: keyword.value || undefined,
    page: currentPage.value > 1 ? String(currentPage.value) : undefined
  }

  router.replace({ query })
}

function selectTag(tag: string) {
  selectedTag.value = tag
  currentPage.value = 1
  syncQuery()
}

function clearTag() {
  selectedTag.value = ''
  currentPage.value = 1
  syncQuery()
}

function onKeywordChange() {
  currentPage.value = 1
  syncQuery()
}

function goPrevPage() {
  if (currentPage.value <= 1) return

  currentPage.value -= 1
  syncQuery()
}

function goNextPage() {
  if (currentPage.value >= totalPages.value) return

  currentPage.value += 1
  syncQuery()
}

watch(totalPages, (value) => {
  if (currentPage.value > value) {
    currentPage.value = value
    syncQuery()
  }
})

watch(() => route.query.tag, (value) => {
  selectedTag.value = typeof value === 'string' ? value : ''
})

watch(() => route.query.q, (value) => {
  keyword.value = typeof value === 'string' ? value : ''
})

watch(() => route.query.page, (value) => {
  const page = Number(value || 1)
  currentPage.value = Number.isFinite(page) && page > 0 ? page : 1
})

const seoTitle = computed(() => `文章归档 - 第 ${currentPage.value} 页`)
const seoDescription = computed(() => `GoPalette 文章归档，当前第 ${currentPage.value} 页，共 ${totalPages.value} 页。`)

useSeoMeta({
  title: seoTitle,
  description: seoDescription
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton to="/write" icon="i-lucide-pen-line" label="写文章" size="sm" />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section
        class="mb-6 flex flex-col gap-4 rounded-2xl border border-default bg-muted/30 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
        <div>
          <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
            文章归档
          </h1>
          <p class="mt-1 text-sm text-toned">
            共 {{ keyword.trim() ? searchState.total : filteredPosts.length }} 篇文章 · 第 {{ currentPage }} / {{ totalPages
            }} 页
          </p>
          <p v-if="keyword.trim()" class="mt-1 text-xs text-toned">
            Meilisearch 即时检索中
          </p>
        </div>

        <UInput v-model="keyword" icon="i-lucide-search" placeholder="搜索标题、标签、分类" class="w-full sm:w-80"
          @update:model-value="onKeywordChange" />
      </section>

      <section class="mb-6 flex flex-wrap gap-2">
        <UButton size="xs" label="全部" :variant="selectedTag ? 'ghost' : 'solid'"
          :color="selectedTag ? 'neutral' : 'primary'" @click="clearTag" />

        <UButton v-for="tag in tags" :key="tag" size="xs" :label="`#${tag}`"
          :variant="selectedTag === tag ? 'solid' : 'ghost'" :color="selectedTag === tag ? 'primary' : 'neutral'"
          @click="selectTag(tag)" />
      </section>

      <section class="grid gap-4 md:grid-cols-2">
        <UCard v-for="post in paginatedPosts" :key="post.id" :ui="{ body: 'p-0' }" class="overflow-hidden">
          <img :src="post.cover" :alt="post.title" class="h-40 w-full object-cover">

          <div class="space-y-3 p-5">
            <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
              <UBadge :label="post.category" color="primary" variant="subtle" />
              <span>{{ post.publishedAt }}</span>
              <span>·</span>
              <span>{{ post.readingMinutes }} 分钟</span>
            </div>

            <NuxtLink :to="`/posts/${post.slug}`"
              class="block text-lg font-semibold text-highlighted transition-colors hover:text-primary">
              <span v-html="renderHighlightedText(post.title)" />
            </NuxtLink>

            <p
              class="line-clamp-2 text-sm text-toned"
              v-html="renderHighlightedText(post.summary)"
            />

            <div class="flex flex-wrap gap-2">
              <UBadge v-for="tag in post.tags" :key="tag" :label="`#${tag}`" color="neutral" variant="outline" />
            </div>
          </div>
        </UCard>
      </section>

      <p v-if="searchLoading" class="mt-4 text-sm text-toned">
        正在检索相关文章...
      </p>

      <section class="mt-8 flex items-center justify-between rounded-xl border border-default bg-muted/20 p-4">
        <UButton icon="i-lucide-chevron-left" label="上一页" color="neutral" variant="ghost" :disabled="currentPage <= 1"
          @click="goPrevPage" />

        <p class="text-sm text-toned">
          第 {{ currentPage }} / {{ totalPages }} 页
        </p>

        <UButton trailing-icon="i-lucide-chevron-right" label="下一页" color="neutral" variant="ghost"
          :disabled="currentPage >= totalPages" @click="goNextPage" />
      </section>
    </main>
  </div>
</template>
