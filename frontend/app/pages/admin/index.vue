<script setup lang="ts">
import { LayoutDashboard, FileText, Tag, FolderOpen, Trash2, Plus, X } from 'lucide-vue-next'
import type { Article, ArticleListData, ApiCategory, ApiTag, ApiResponse } from '~/composables/useBlogData'
import { formatDate } from '~/composables/useBlogData'

definePageMeta({ ssr: false })

const router = useRouter()
const { user, authFetch, isLoggedIn } = useAuth()

// 权限检查
onMounted(async () => {
  if (!isLoggedIn.value) { router.push('/login'); return }
  await loadData()
  if (user.value?.role !== 'admin') router.push('/')
})

// 选项卡
const tab = ref<'articles' | 'categories' | 'tags'>('articles')

// 文章管理
const articles = ref<Article[]>([])
const totalArticles = ref(0)
const articlePage = ref(1)
const ARTICLE_PAGE_SIZE = 20
const articlePending = ref(false)

async function loadArticles() {
  articlePending.value = true
  try {
    const res = await authFetch<ApiResponse<ArticleListData>>(
      `/admin/articles?page=${articlePage.value}&page_size=${ARTICLE_PAGE_SIZE}`
    )
    if (res.code === 200) {
      articles.value = res.data.articles ?? []
      totalArticles.value = res.data.total ?? 0
    }
  } finally {
    articlePending.value = false
  }
}

async function deleteArticle(id: number) {
  if (!confirm('确认删除此文章？')) return
  await authFetch(`/articles/${id}`, { method: 'DELETE' })
  await loadArticles()
}

watch(articlePage, loadArticles)

// 分类管理
const categories = ref<ApiCategory[]>([])
const newCategoryName = ref('')
const catError = ref('')

async function loadCategories() {
  const res = await authFetch<ApiResponse<ApiCategory[]>>('/categories')
  if (res.code === 200) categories.value = res.data ?? []
}

async function createCategory() {
  catError.value = ''
  if (!newCategoryName.value.trim()) { catError.value = '分类名不能为空'; return }
  try {
    await authFetch('/categories', { method: 'POST', body: { name: newCategoryName.value.trim() } })
    newCategoryName.value = ''
    await loadCategories()
  } catch {
    catError.value = '创建失败，分类名可能已存在'
  }
}

async function deleteCategory(id: number) {
  if (!confirm('确认删除此分类？')) return
  await authFetch(`/categories/${id}`, { method: 'DELETE' })
  await loadCategories()
}

// 标签管理
const tags = ref<ApiTag[]>([])
const newTagName = ref('')
const tagError = ref('')

async function loadTags() {
  const res = await authFetch<ApiResponse<ApiTag[]>>('/tags')
  if (res.code === 200) tags.value = res.data ?? []
}

async function createTag() {
  tagError.value = ''
  if (!newTagName.value.trim()) { tagError.value = '标签名不能为空'; return }
  try {
    await authFetch('/tags', { method: 'POST', body: { name: newTagName.value.trim() } })
    newTagName.value = ''
    await loadTags()
  } catch {
    tagError.value = '创建失败，标签名可能已存在'
  }
}

async function deleteTag(id: number) {
  if (!confirm('确认删除此标签？')) return
  await authFetch(`/tags/${id}`, { method: 'DELETE' })
  await loadTags()
}

// 初始化
async function loadData() {
  await Promise.all([loadArticles(), loadCategories(), loadTags()])
}

const articleTotalPages = computed(() => Math.ceil(totalArticles.value / ARTICLE_PAGE_SIZE))

const stats = computed(() => [
  { label: 'Total Articles', value: totalArticles.value, icon: FileText },
  { label: 'Categories', value: categories.value.length, icon: FolderOpen },
  { label: 'Tags', value: tags.value.length, icon: Tag },
])
</script>

<template>
  <div
    v-motion
    :initial="{ opacity: 0, y: 16 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500 } }"
    class="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <!-- 页头 -->
    <div class="flex items-center gap-4 mb-10">
      <div class="w-12 h-12 rounded-2xl bg-m3-sys-light-tertiary-container flex items-center justify-center">
        <LayoutDashboard class="w-6 h-6 text-m3-sys-light-on-tertiary-container" />
      </div>
      <div>
        <h1 class="text-3xl font-black tracking-tight text-m3-sys-light-on-surface">Admin Dashboard</h1>
        <p class="text-m3-sys-light-on-surface-variant text-sm">Welcome back, {{ user?.username }}</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-3 gap-4 mb-10">
      <div
        v-for="stat in stats" :key="stat.label"
        class="bg-m3-sys-light-surface-variant rounded-2xl p-6 flex items-center gap-4"
      >
        <div class="w-10 h-10 rounded-xl bg-m3-sys-light-primary-container flex items-center justify-center">
          <component :is="stat.icon" class="w-5 h-5 text-m3-sys-light-on-primary-container" />
        </div>
        <div>
          <p class="text-2xl font-black text-m3-sys-light-on-surface">{{ stat.value }}</p>
          <p class="text-xs text-m3-sys-light-on-surface-variant">{{ stat.label }}</p>
        </div>
      </div>
    </div>

    <!-- 选项卡 -->
    <div class="flex gap-1 bg-m3-sys-light-surface-variant rounded-2xl p-1 mb-8 w-fit">
      <button
        v-for="t in (['articles', 'categories', 'tags'] as const)" :key="t"
        @click="tab = t"
        class="px-5 py-2 rounded-xl font-medium text-sm capitalize transition-all"
        :class="tab === t
          ? 'bg-m3-sys-light-surface text-m3-sys-light-on-surface shadow-sm'
          : 'text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-surface'"
      >
        {{ t }}
      </button>
    </div>

    <!-- 文章管理 -->
    <div v-if="tab === 'articles'">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-m3-sys-light-on-surface">All Articles</h2>
        <button
          @click="router.push('/write')"
          class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full text-sm font-medium hover:opacity-90 transition-opacity"
        >
          <Plus class="w-4 h-4" />New Article
        </button>
      </div>

      <div v-if="articlePending" class="space-y-3">
        <div v-for="i in 6" :key="i" class="h-14 rounded-xl bg-m3-sys-light-surface-variant animate-pulse" />
      </div>

      <div v-else class="rounded-2xl overflow-hidden border border-m3-sys-light-outline-variant">
        <table class="w-full text-sm">
          <thead class="bg-m3-sys-light-surface-variant">
            <tr>
              <th class="text-left px-5 py-3 font-semibold text-m3-sys-light-on-surface-variant">Title</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden sm:table-cell">Author</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden md:table-cell">Status</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden lg:table-cell">Date</th>
              <th class="px-4 py-3" />
            </tr>
          </thead>
          <tbody class="divide-y divide-m3-sys-light-outline-variant">
            <tr
              v-for="article in articles" :key="article.id"
              class="bg-m3-sys-light-surface hover:bg-m3-sys-light-surface-variant/50 transition-colors"
            >
              <td class="px-5 py-3">
                <NuxtLink :to="`/post/${article.id}`" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors line-clamp-1">
                  {{ article.title }}
                </NuxtLink>
              </td>
              <td class="px-4 py-3 text-m3-sys-light-on-surface-variant hidden sm:table-cell">
                {{ article.author?.username ?? '—' }}
              </td>
              <td class="px-4 py-3 hidden md:table-cell">
                <span
                  class="px-2.5 py-0.5 rounded-full text-xs font-semibold"
                  :class="article.status === 'published' ? 'bg-green-100 text-green-700' : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant'"
                >
                  {{ article.status === 'published' ? 'Published' : 'Draft' }}
                </span>
              </td>
              <td class="px-4 py-3 text-m3-sys-light-on-surface-variant text-xs hidden lg:table-cell">
                {{ formatDate(article.created_at) }}
              </td>
              <td class="px-4 py-3 text-right">
                <button
                  @click="deleteArticle(article.id)"
                  class="p-1.5 rounded-lg hover:bg-m3-sys-light-error-container text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-error-container transition-colors"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 文章分页 -->
      <div v-if="articleTotalPages > 1" class="flex justify-center gap-2 mt-6">
        <button
          v-for="p in articleTotalPages" :key="p"
          @click="articlePage = p"
          class="w-9 h-9 rounded-full text-sm font-medium transition-colors"
          :class="p === articlePage
            ? 'bg-m3-sys-light-primary text-m3-sys-light-on-primary'
            : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container'"
        >{{ p }}</button>
      </div>
    </div>

    <!-- 分类管理 -->
    <div v-if="tab === 'categories'" class="max-w-md">
      <h2 class="font-bold text-m3-sys-light-on-surface mb-4">Manage Categories</h2>

      <!-- 新建表单 -->
      <form @submit.prevent="createCategory" class="flex gap-2 mb-4">
        <input
          v-model="newCategoryName"
          placeholder="New category name…"
          class="flex-1 bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30"
        />
        <button type="submit" class="px-4 py-2.5 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-xl font-medium hover:opacity-90 transition-opacity flex items-center gap-1">
          <Plus class="w-4 h-4" />Add
        </button>
      </form>
      <p v-if="catError" class="text-red-500 text-sm mb-3">{{ catError }}</p>

      <!-- 分类列表 -->
      <ul class="space-y-2">
        <li
          v-for="cat in categories" :key="cat.id"
          class="flex items-center justify-between bg-m3-sys-light-surface rounded-xl px-4 py-3"
        >
          <span class="font-medium text-m3-sys-light-on-surface">{{ cat.name }}</span>
          <button
            @click="deleteCategory(cat.id)"
            class="p-1.5 rounded-lg hover:bg-m3-sys-light-error-container text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-error-container transition-colors"
          >
            <X class="w-4 h-4" />
          </button>
        </li>
        <li v-if="categories.length === 0" class="text-m3-sys-light-on-surface-variant text-sm py-4 text-center">
          No categories yet.
        </li>
      </ul>
    </div>

    <!-- 标签管理 -->
    <div v-if="tab === 'tags'" class="max-w-2xl">
      <h2 class="font-bold text-m3-sys-light-on-surface mb-4">Manage Tags</h2>

      <!-- 新建表单 -->
      <form @submit.prevent="createTag" class="flex gap-2 mb-4">
        <input
          v-model="newTagName"
          placeholder="New tag name…"
          class="flex-1 bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30"
        />
        <button type="submit" class="px-4 py-2.5 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-xl font-medium hover:opacity-90 transition-opacity flex items-center gap-1">
          <Plus class="w-4 h-4" />Add
        </button>
      </form>
      <p v-if="tagError" class="text-red-500 text-sm mb-3">{{ tagError }}</p>

      <!-- 标签云 -->
      <div class="flex flex-wrap gap-2">
        <span
          v-for="tag in tags" :key="tag.id"
          class="flex items-center gap-1.5 bg-m3-sys-light-surface rounded-full px-3 py-1.5 text-sm font-medium text-m3-sys-light-on-surface"
        >
          # {{ tag.name }}
          <button
            @click="deleteTag(tag.id)"
            class="text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-error transition-colors"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </span>
        <span v-if="tags.length === 0" class="text-m3-sys-light-on-surface-variant text-sm py-2">
          No tags yet.
        </span>
      </div>
    </div>
  </div>
</template>
