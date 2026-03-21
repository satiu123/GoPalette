<script setup lang="ts">
import type { Ref } from 'vue'
import { LayoutDashboard, FileText, Tag, FolderOpen, Trash2, Plus, X, MessageSquare } from 'lucide-vue-next'
import type { Article, ArticleListData, ApiCategory, ApiTag, ApiResponse, Comment } from '~/composables/useBlogData'
import { formatDate, articlePath } from '~/composables/useBlogData'

definePageMeta({ ssr: false })

const router = useRouter()
const { user, authFetch, isLoggedIn } = useAuth()
const { askConfirm } = useConfirmDialog()

// 权限检查
onMounted(async () => {
  if (!isLoggedIn.value) { router.push('/login'); return }
  await loadData()
  if (user.value?.role !== 'admin') router.push('/')
})

// 选项卡
const tab = ref<'articles' | 'categories' | 'tags' | 'comments'>('articles')

// 文章管理
const articles = ref<Article[]>([])
const selectedArticleIds = ref<number[]>([])
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
      selectedArticleIds.value = []
    }
  } finally {
    articlePending.value = false
  }
}

async function deleteArticle(id: number) {
  const ok = await askConfirm({
    title: '删除文章',
    message: '确认删除这篇文章吗？该操作不可撤销。',
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return
  await authFetch(`/articles/${id}`, { method: 'DELETE' })
  await loadArticles()
}

watch(articlePage, loadArticles)

// 分类管理
const categories = ref<ApiCategory[]>([])
const selectedCategoryIds = ref<number[]>([])
const newCategoryName = ref('')
const catError = ref('')

async function loadCategories() {
  const res = await authFetch<ApiResponse<ApiCategory[]>>('/categories')
  if (res.code === 200) {
    categories.value = res.data ?? []
    selectedCategoryIds.value = []
  }
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
  const ok = await askConfirm({
    title: '删除分类',
    message: '确认删除这个分类吗？该操作不可撤销。',
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return
  await authFetch(`/categories/${id}`, { method: 'DELETE' })
  await loadCategories()
}

// 标签管理
const tags = ref<ApiTag[]>([])
const selectedTagIds = ref<number[]>([])
const newTagName = ref('')
const tagError = ref('')

async function loadTags() {
  const res = await authFetch<ApiResponse<ApiTag[]>>('/tags')
  if (res.code === 200) {
    tags.value = res.data ?? []
    selectedTagIds.value = []
  }
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
  const ok = await askConfirm({
    title: '删除标签',
    message: '确认删除这个标签吗？该操作不可撤销。',
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return
  await authFetch(`/tags/${id}`, { method: 'DELETE' })
  await loadTags()
}

// 评论管理
interface AdminCommentResp {
  comments: Comment[]
  total: number
}

const comments = ref<Comment[]>([])
const selectedCommentIds = ref<number[]>([])
const totalComments = ref(0)
const commentPage = ref(1)
const COMMENT_PAGE_SIZE = 20
const commentPending = ref(false)

async function loadComments() {
  commentPending.value = true
  try {
    const res = await authFetch<ApiResponse<AdminCommentResp>>(
      `/admin/comments?page=${commentPage.value}&page_size=${COMMENT_PAGE_SIZE}`
    )
    if (res.code === 200) {
      comments.value = res.data.comments ?? []
      totalComments.value = res.data.total ?? 0
      selectedCommentIds.value = []
    }
  } finally {
    commentPending.value = false
  }
}

async function deleteCommentAdmin(id: number) {
  const ok = await askConfirm({
    title: '删除评论',
    message: '确认删除这条评论吗？该操作不可撤销。',
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return
  await authFetch(`/comments/${id}`, { method: 'DELETE' })
  await loadComments()
}

watch(commentPage, loadComments)

const bulkDeleting = ref(false)
const bulkError = ref('')

function updateSelection(list: Ref<number[]>, id: number, checked: boolean) {
  if (checked) {
    if (!list.value.includes(id)) list.value.push(id)
    return
  }
  list.value = list.value.filter(item => item !== id)
}

function updateArticleSelection(id: number, checked: boolean) {
  updateSelection(selectedArticleIds, id, checked)
}

function updateCategorySelection(id: number, checked: boolean) {
  updateSelection(selectedCategoryIds, id, checked)
}

function updateTagSelection(id: number, checked: boolean) {
  updateSelection(selectedTagIds, id, checked)
}

function updateCommentSelection(id: number, checked: boolean) {
  updateSelection(selectedCommentIds, id, checked)
}

async function deleteMany(requests: Array<Promise<unknown>>) {
  const results = await Promise.allSettled(requests)
  return results.filter(item => item.status === 'rejected').length
}

const allArticlesSelected = computed(() =>
  articles.value.length > 0 && selectedArticleIds.value.length === articles.value.length
)

function toggleAllArticles(checked: boolean) {
  selectedArticleIds.value = checked ? articles.value.map(item => item.id) : []
}

async function bulkDeleteArticles() {
  if (!selectedArticleIds.value.length) return
  const count = selectedArticleIds.value.length
  const ok = await askConfirm({
    title: '批量删除文章',
    message: `确认删除已选中的 ${count} 篇文章吗？该操作不可撤销。`,
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return

  bulkDeleting.value = true
  bulkError.value = ''
  try {
    const failed = await deleteMany(
      selectedArticleIds.value.map(id => authFetch(`/articles/${id}`, { method: 'DELETE' }))
    )
    await loadArticles()
    if (failed > 0) bulkError.value = `有 ${failed} 篇文章删除失败，请重试。`
  } finally {
    bulkDeleting.value = false
  }
}

const allCategoriesSelected = computed(() =>
  categories.value.length > 0 && selectedCategoryIds.value.length === categories.value.length
)

function toggleAllCategories(checked: boolean) {
  selectedCategoryIds.value = checked ? categories.value.map(item => item.id) : []
}

async function bulkDeleteCategories() {
  if (!selectedCategoryIds.value.length) return
  const count = selectedCategoryIds.value.length
  const ok = await askConfirm({
    title: '批量删除分类',
    message: `确认删除已选中的 ${count} 个分类吗？该操作不可撤销。`,
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return

  bulkDeleting.value = true
  bulkError.value = ''
  try {
    const failed = await deleteMany(
      selectedCategoryIds.value.map(id => authFetch(`/categories/${id}`, { method: 'DELETE' }))
    )
    await loadCategories()
    if (failed > 0) bulkError.value = `有 ${failed} 个分类删除失败，请重试。`
  } finally {
    bulkDeleting.value = false
  }
}

const allTagsSelected = computed(() =>
  tags.value.length > 0 && selectedTagIds.value.length === tags.value.length
)

function toggleAllTags(checked: boolean) {
  selectedTagIds.value = checked ? tags.value.map(item => item.id) : []
}

async function bulkDeleteTags() {
  if (!selectedTagIds.value.length) return
  const count = selectedTagIds.value.length
  const ok = await askConfirm({
    title: '批量删除标签',
    message: `确认删除已选中的 ${count} 个标签吗？该操作不可撤销。`,
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return

  bulkDeleting.value = true
  bulkError.value = ''
  try {
    const failed = await deleteMany(
      selectedTagIds.value.map(id => authFetch(`/tags/${id}`, { method: 'DELETE' }))
    )
    await loadTags()
    if (failed > 0) bulkError.value = `有 ${failed} 个标签删除失败，请重试。`
  } finally {
    bulkDeleting.value = false
  }
}

const allCommentsSelected = computed(() =>
  comments.value.length > 0 && selectedCommentIds.value.length === comments.value.length
)

function toggleAllComments(checked: boolean) {
  selectedCommentIds.value = checked ? comments.value.map(item => item.id) : []
}

async function bulkDeleteComments() {
  if (!selectedCommentIds.value.length) return
  const count = selectedCommentIds.value.length
  const ok = await askConfirm({
    title: '批量删除评论',
    message: `确认删除已选中的 ${count} 条评论吗？该操作不可撤销。`,
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return

  bulkDeleting.value = true
  bulkError.value = ''
  try {
    const failed = await deleteMany(
      selectedCommentIds.value.map(id => authFetch(`/comments/${id}`, { method: 'DELETE' }))
    )
    await loadComments()
    if (failed > 0) bulkError.value = `有 ${failed} 条评论删除失败，请重试。`
  } finally {
    bulkDeleting.value = false
  }
}

// 初始化
async function loadData() {
  await Promise.all([loadArticles(), loadCategories(), loadTags(), loadComments()])
}

const articleTotalPages = computed(() => Math.ceil(totalArticles.value / ARTICLE_PAGE_SIZE))
const commentTotalPages = computed(() => Math.ceil(totalComments.value / COMMENT_PAGE_SIZE))

const stats = computed(() => [
  { label: 'Total Articles', value: totalArticles.value, icon: FileText },
  { label: 'Categories', value: categories.value.length, icon: FolderOpen },
  { label: 'Tags', value: tags.value.length, icon: Tag },
  { label: 'Comments', value: totalComments.value, icon: MessageSquare },
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
        v-for="t in (['articles', 'categories', 'tags', 'comments'] as const)" :key="t"
        @click="tab = t"
        class="px-5 py-2 rounded-xl font-medium text-sm capitalize transition-all"
        :class="tab === t
          ? 'bg-m3-sys-light-surface text-m3-sys-light-on-surface shadow-sm'
          : 'text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-surface'"
      >
        {{ t }}
      </button>
    </div>
    <p v-if="bulkError" class="text-red-500 text-sm mb-4">{{ bulkError }}</p>

    <!-- 文章管理 -->
    <div v-if="tab === 'articles'">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-m3-sys-light-on-surface">All Articles</h2>
        <div class="flex items-center gap-2">
          <button
            v-if="selectedArticleIds.length > 0"
            :disabled="bulkDeleting"
            @click="bulkDeleteArticles"
            class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-error-container text-m3-sys-light-on-error-container rounded-full text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-60"
          >
            <Trash2 class="w-4 h-4" />删除已选 ({{ selectedArticleIds.length }})
          </button>
          <button
            @click="router.push('/write')"
            class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full text-sm font-medium hover:opacity-90 transition-opacity"
          >
            <Plus class="w-4 h-4" />New Article
          </button>
        </div>
      </div>

      <div v-if="articlePending" class="space-y-3">
        <div v-for="i in 6" :key="i" class="h-14 rounded-xl bg-m3-sys-light-surface-variant animate-pulse" />
      </div>

      <div v-else class="rounded-2xl overflow-hidden border border-m3-sys-light-outline-variant">
        <table class="w-full text-sm">
          <thead class="bg-m3-sys-light-surface-variant">
            <tr>
              <th class="w-12 px-3 py-3">
                <input
                  type="checkbox"
                  :checked="allArticlesSelected"
                  @change="toggleAllArticles(($event.target as HTMLInputElement).checked)"
                  class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
                />
              </th>
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
              <td class="px-3 py-3">
                <input
                  type="checkbox"
                  :checked="selectedArticleIds.includes(article.id)"
                  @change="updateArticleSelection(article.id, ($event.target as HTMLInputElement).checked)"
                  class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
                />
              </td>
              <td class="px-5 py-3">
                <NuxtLink :to="articlePath(article)" class="font-medium text-m3-sys-light-on-surface hover:text-m3-sys-light-primary transition-colors line-clamp-1">
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
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-m3-sys-light-on-surface">Manage Categories</h2>
        <button
          v-if="selectedCategoryIds.length > 0"
          :disabled="bulkDeleting"
          @click="bulkDeleteCategories"
          class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-error-container text-m3-sys-light-on-error-container rounded-full text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          <Trash2 class="w-4 h-4" />删除已选 ({{ selectedCategoryIds.length }})
        </button>
      </div>

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
      <label class="inline-flex items-center gap-2 mb-3 text-sm text-m3-sys-light-on-surface-variant">
        <input
          type="checkbox"
          :checked="allCategoriesSelected"
          @change="toggleAllCategories(($event.target as HTMLInputElement).checked)"
          class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
        />
        全选
      </label>

      <!-- 分类列表 -->
      <ul class="space-y-2">
        <li
          v-for="cat in categories" :key="cat.id"
          class="flex items-center justify-between bg-m3-sys-light-surface rounded-xl px-4 py-3"
        >
          <div class="flex items-center gap-2 min-w-0">
            <input
              type="checkbox"
              :checked="selectedCategoryIds.includes(cat.id)"
              @change="updateCategorySelection(cat.id, ($event.target as HTMLInputElement).checked)"
              class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
            />
            <span class="font-medium text-m3-sys-light-on-surface truncate">{{ cat.name }}</span>
          </div>
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
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-m3-sys-light-on-surface">Manage Tags</h2>
        <button
          v-if="selectedTagIds.length > 0"
          :disabled="bulkDeleting"
          @click="bulkDeleteTags"
          class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-error-container text-m3-sys-light-on-error-container rounded-full text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          <Trash2 class="w-4 h-4" />删除已选 ({{ selectedTagIds.length }})
        </button>
      </div>

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
      <label class="inline-flex items-center gap-2 mb-3 text-sm text-m3-sys-light-on-surface-variant">
        <input
          type="checkbox"
          :checked="allTagsSelected"
          @change="toggleAllTags(($event.target as HTMLInputElement).checked)"
          class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
        />
        全选
      </label>

      <!-- 标签云 -->
      <div class="flex flex-wrap gap-2">
        <span
          v-for="tag in tags" :key="tag.id"
          class="flex items-center gap-1.5 bg-m3-sys-light-surface rounded-full px-3 py-1.5 text-sm font-medium text-m3-sys-light-on-surface"
        >
          <input
            type="checkbox"
            :checked="selectedTagIds.includes(tag.id)"
            @change="updateTagSelection(tag.id, ($event.target as HTMLInputElement).checked)"
            class="w-3.5 h-3.5 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
          />
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

    <!-- 评论管理 -->
    <div v-if="tab === 'comments'">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-m3-sys-light-on-surface">All Comments</h2>
        <button
          v-if="selectedCommentIds.length > 0"
          :disabled="bulkDeleting"
          @click="bulkDeleteComments"
          class="flex items-center gap-1.5 px-4 py-2 bg-m3-sys-light-error-container text-m3-sys-light-on-error-container rounded-full text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          <Trash2 class="w-4 h-4" />删除已选 ({{ selectedCommentIds.length }})
        </button>
      </div>

      <div v-if="commentPending" class="space-y-3">
        <div v-for="i in 6" :key="i" class="h-14 rounded-xl bg-m3-sys-light-surface-variant animate-pulse" />
      </div>

      <div v-else class="rounded-2xl overflow-hidden border border-m3-sys-light-outline-variant">
        <table class="w-full text-sm">
          <thead class="bg-m3-sys-light-surface-variant">
            <tr>
              <th class="w-12 px-3 py-3">
                <input
                  type="checkbox"
                  :checked="allCommentsSelected"
                  @change="toggleAllComments(($event.target as HTMLInputElement).checked)"
                  class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
                />
              </th>
              <th class="text-left px-5 py-3 font-semibold text-m3-sys-light-on-surface-variant">评论内容</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden md:table-cell">所属文章</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden sm:table-cell">评论者</th>
              <th class="text-left px-4 py-3 font-semibold text-m3-sys-light-on-surface-variant hidden lg:table-cell">时间</th>
              <th class="px-4 py-3" />
            </tr>
          </thead>
          <tbody class="divide-y divide-m3-sys-light-outline-variant">
            <tr
              v-for="comment in comments" :key="comment.id"
              class="bg-m3-sys-light-surface hover:bg-m3-sys-light-surface-variant/50 transition-colors"
            >
              <td class="px-3 py-3">
                <input
                  type="checkbox"
                  :checked="selectedCommentIds.includes(comment.id)"
                  @change="updateCommentSelection(comment.id, ($event.target as HTMLInputElement).checked)"
                  class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
                />
              </td>
              <td class="px-5 py-3 max-w-xs">
                <p class="line-clamp-2 text-m3-sys-light-on-surface">{{ comment.content }}</p>
              </td>
              <td class="px-4 py-3 hidden md:table-cell text-m3-sys-light-on-surface-variant">
                <NuxtLink v-if="comment.article" :to="articlePath(comment.article)" class="hover:text-m3-sys-light-primary transition-colors line-clamp-1">
                  {{ comment.article.title }}
                </NuxtLink>
                <span v-else>—</span>
              </td>
              <td class="px-4 py-3 hidden sm:table-cell text-m3-sys-light-on-surface-variant">
                {{ comment.user?.username || '匿名用户' }}
              </td>
              <td class="px-4 py-3 text-m3-sys-light-on-surface-variant text-xs hidden lg:table-cell">
                {{ formatDate(comment.created_at) }}
              </td>
              <td class="px-4 py-3 text-right">
                <button
                  @click="deleteCommentAdmin(comment.id)"
                  class="p-1.5 rounded-lg hover:bg-m3-sys-light-error-container text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-error-container transition-colors"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </td>
            </tr>
            <tr v-if="comments.length === 0">
              <td colspan="6" class="px-5 py-10 text-center text-m3-sys-light-on-surface-variant">暂无评论</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 评论分页 -->
      <div v-if="commentTotalPages > 1" class="flex justify-center gap-2 mt-6">
        <button
          v-for="p in commentTotalPages" :key="p"
          @click="commentPage = p"
          class="w-9 h-9 rounded-full text-sm font-medium transition-colors"
          :class="p === commentPage
            ? 'bg-m3-sys-light-primary text-m3-sys-light-on-primary'
            : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container'"
        >{{ p }}</button>
      </div>
    </div>
  </div>
</template>
