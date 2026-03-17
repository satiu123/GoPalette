<script setup lang="ts">
import type { Article } from '~/composables/useBlogData'

const route = useRoute()
const router = useRouter()

// 搜索关键词（来自 ?q=... 查询参数或 Header 搜索框）
const searchQuery = computed(() => String(route.query.q ?? '').trim())

const filterParams = ref<{ page: number; page_size: number; category_id?: number; tag_id?: number }>({
  page: 1,
  page_size: 12
})

// 有搜索词时走 /search，否则走 /articles
const searchParams = computed(() => ({
  q: searchQuery.value,
  page: filterParams.value.page,
  page_size: filterParams.value.page_size
}))

const {
  data: listData,
  pending: listPending,
  refresh: refreshList
} = useArticleList(filterParams)

const {
  data: searchData,
  pending: searchPending,
  refresh: refreshSearch
} = useSearch(searchParams)

watch(searchParams, (params) => {
  if (params.q) {
    refreshSearch()
  }
}, { immediate: true, deep: true })

const isSearching = computed(() => !!searchQuery.value)
const pending     = computed(() => isSearching.value ? searchPending.value : listPending.value)

const articles = computed<Article[]>(() => {
  if (isSearching.value) return searchData.value?.data?.articles ?? []
  return listData.value?.data?.articles ?? []
})
const total = computed<number>(() => {
  if (isSearching.value) return searchData.value?.data?.total ?? 0
  return listData.value?.data?.total ?? 0
})

const featuredPost = computed(() => articles.value[0] ?? null)
const regularPosts  = computed(() => articles.value.slice(1))

function clearSearch() {
  router.push('/')
}
</script>

<template>
  <div>
    <!-- 搜索状态提示 -->
    <div v-if="isSearching" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-8">
      <div class="flex items-center gap-3 text-m3-sys-light-on-surface-variant">
        <span>Search results for "<strong class="text-m3-sys-light-on-surface">{{ searchQuery }}</strong>" — {{ total }} found</span>
        <button @click="clearSearch" class="text-m3-sys-light-primary text-sm hover:underline">Clear</button>
      </div>
    </div>

    <div v-if="pending" class="flex justify-center items-center min-h-[50vh] text-m3-sys-light-on-surface-variant">
      Loading…
    </div>
    <template v-else-if="featuredPost">
      <AppHero v-if="!isSearching" :post="featuredPost" />
      <PostGrid :posts="isSearching ? articles : regularPosts" :total="total" :params="filterParams" @update:params="filterParams = $event" />
    </template>
    <div v-else class="flex justify-center items-center min-h-[50vh] text-m3-sys-light-on-surface-variant">
      {{ isSearching ? 'No results found.' : 'No articles yet.' }}
    </div>
  </div>
</template>
