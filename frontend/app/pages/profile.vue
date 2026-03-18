<script setup lang="ts">
import { PenSquare, Trash2, BookOpen, FileText, Clock, Settings, Camera } from 'lucide-vue-next'
import type { Article, ArticleListData, ApiResponse, ApiUser } from '~/composables/useBlogData'
import { formatDate, articleImageUrl, userAvatarUrl } from '~/composables/useBlogData'

definePageMeta({ middleware: 'auth', ssr: false })

const router = useRouter()
const { user, authFetch } = useAuth()
const { askConfirm } = useConfirmDialog()

// 个人信息（从后端刷新）
const profile = ref<ApiUser | null>(null)
const profilePending = ref(true)

// 编辑个人信息
const editMode = ref(false)
const editUsername = ref('')
const editOldPassword = ref('')
const editNewPassword = ref('')
const editError = ref('')
const editSuccess = ref('')
const editSubmitting = ref(false)

// 头像上传
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarUploading = ref(false)
const avatarError = ref('')

const profileAvatarSrc = computed(() => userAvatarUrl(profile.value ?? user.value))

function openAvatarUpload() {
  avatarInput.value?.click()
}

async function onAvatarSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  avatarError.value = ''
  const allowTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
  if (!allowTypes.includes(file.type)) {
    avatarError.value = '仅支持 JPEG/PNG/WebP/GIF 格式'
    input.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    avatarError.value = '图片大小不能超过 5MB'
    input.value = ''
    return
  }

  avatarUploading.value = true
  try {
    const formData = new FormData()
    formData.append('image', file)
    const uploadRes = await authFetch<ApiResponse<{ url: string }>>('/upload', {
      method: 'POST',
      body: formData
    })
    if (uploadRes.code !== 200) {
      avatarError.value = uploadRes.msg || '头像上传失败'
      return
    }

    const saveRes = await authFetch<ApiResponse<unknown>>('/users/me', {
      method: 'PUT',
      body: { avatar_url: uploadRes.data.url }
    })
    if (saveRes.code !== 200) {
      avatarError.value = saveRes.msg || '头像保存失败'
      return
    }

    await loadProfile()
  } catch {
    avatarError.value = '头像上传失败，请重试'
  } finally {
    avatarUploading.value = false
    input.value = ''
  }
}

function openEdit() {
  editUsername.value = profile.value?.username ?? user.value?.username ?? ''
  editOldPassword.value = ''
  editNewPassword.value = ''
  editError.value = ''
  editSuccess.value = ''
  editMode.value = true
}

async function submitEdit() {
  editError.value = ''
  editSuccess.value = ''
  if (!editUsername.value.trim()) {
    editError.value = '用户名不能为空'
    return
  }
  editSubmitting.value = true
  try {
    const body: Record<string, string> = { username: editUsername.value.trim() }
    if (editNewPassword.value) {
      body.old_password = editOldPassword.value
      body.new_password = editNewPassword.value
    }
    await authFetch('/users/me', { method: 'PUT', body })
    editSuccess.value = '更新成功'
    editMode.value = false
    await loadProfile()
  } catch (e: unknown) {
    editError.value = (e as { data?: { msg?: string } })?.data?.msg ?? '更新失败，请重试'
  } finally {
    editSubmitting.value = false
  }
}

// 我的文章
const page = ref(1)
const PAGE_SIZE = 12
const myArticles = ref<Article[]>([])
const selectedArticleIds = ref<number[]>([])
const totalArticles = ref(0)
const articlesPending = ref(true)
const deleteError = ref('')
const bulkDeleting = ref(false)

async function loadProfile() {
  profilePending.value = true
  try {
    const res = await authFetch<ApiResponse<ApiUser>>('/users/me')
    if (res.code === 200) {
      profile.value = res.data
      user.value = res.data
    }
  } finally {
    profilePending.value = false
  }
}

async function loadArticles() {
  articlesPending.value = true
  try {
    const res = await authFetch<ApiResponse<ArticleListData>>(
      `/users/me/articles?page=${page.value}&page_size=${PAGE_SIZE}`
    )
    if (res.code === 200) {
      myArticles.value = res.data.articles ?? []
      totalArticles.value = res.data.total ?? 0
      selectedArticleIds.value = []
    }
  } finally {
    articlesPending.value = false
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
  deleteError.value = ''
  try {
    await authFetch(`/articles/${id}`, { method: 'DELETE' })
    await loadArticles()
  } catch {
    deleteError.value = '删除失败，请重试'
  }
}

watch(page, loadArticles)

const allArticlesSelected = computed(() =>
  myArticles.value.length > 0 && selectedArticleIds.value.length === myArticles.value.length
)

function toggleAllArticles(checked: boolean) {
  selectedArticleIds.value = checked ? myArticles.value.map(item => item.id) : []
}

function updateSelection(id: number, checked: boolean) {
  if (checked) {
    if (!selectedArticleIds.value.includes(id)) selectedArticleIds.value.push(id)
    return
  }
  selectedArticleIds.value = selectedArticleIds.value.filter(item => item !== id)
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
  deleteError.value = ''
  try {
    const results = await Promise.allSettled(
      selectedArticleIds.value.map(id => authFetch(`/articles/${id}`, { method: 'DELETE' }))
    )
    const failed = results.filter(item => item.status === 'rejected').length
    await loadArticles()
    if (failed > 0) deleteError.value = `有 ${failed} 篇文章删除失败，请重试。`
  } finally {
    bulkDeleting.value = false
  }
}

onMounted(async () => {
  await loadProfile()
  await loadArticles()
})

const totalPages = computed(() => Math.ceil(totalArticles.value / PAGE_SIZE))

const totalReads = computed(() =>
  myArticles.value.reduce((sum, a) => sum + (a.read_count ?? 0), 0)
)
</script>

<template>
  <div
    v-motion
    :initial="{ opacity: 0, y: 16 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500 } }"
    class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <!-- 个人信息卡片 -->
    <div class="bg-m3-sys-light-surface-variant rounded-[2rem] p-8 mb-10 flex flex-col sm:flex-row items-center sm:items-start gap-6">
      <div class="relative shrink-0">
        <img
          :src="profileAvatarSrc"
          :alt="profile?.username ?? user?.username"
          class="w-20 h-20 rounded-full object-cover border-2 border-m3-sys-light-surface"
          referrerpolicy="no-referrer"
        />
        <button
          @click="openAvatarUpload"
          :disabled="avatarUploading"
          class="absolute -right-1 -bottom-1 w-8 h-8 rounded-full bg-m3-sys-light-primary text-m3-sys-light-on-primary flex items-center justify-center hover:opacity-90 transition-opacity disabled:opacity-60"
          title="上传头像"
        >
          <Camera class="w-4 h-4" />
        </button>
        <input
          ref="avatarInput"
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif"
          class="hidden"
          @change="onAvatarSelected"
        />
      </div>
      <div class="flex-1 text-center sm:text-left">
        <div v-if="profilePending" class="h-6 w-32 bg-m3-sys-light-outline-variant rounded animate-pulse mb-2" />
        <template v-else>
          <h1 class="text-3xl font-black tracking-tight text-m3-sys-light-on-surface">
            {{ profile?.username ?? user?.username }}
          </h1>
          <span
            class="inline-block mt-1 px-3 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider"
            :class="profile?.role === 'admin'
              ? 'bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container'
              : 'bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container'"
          >
            {{ profile?.role === 'admin' ? 'Admin' : 'Member' }}
          </span>
          <p v-if="profile?.created_at" class="mt-2 text-sm text-m3-sys-light-on-surface-variant flex items-center justify-center sm:justify-start gap-1">
            <Clock class="w-3.5 h-3.5" />
            Joined {{ formatDate(profile.created_at) }}
          </p>
          <p v-if="avatarUploading" class="mt-2 text-sm text-m3-sys-light-on-surface-variant">头像上传中...</p>
          <p v-if="avatarError" class="mt-2 text-sm text-red-500">{{ avatarError }}</p>
        </template>
      </div>
      <!-- 统计 -->
      <div class="flex gap-8">
        <div class="text-center">
          <p class="text-3xl font-black text-m3-sys-light-primary">{{ totalArticles }}</p>
          <p class="text-xs text-m3-sys-light-on-surface-variant mt-0.5">Articles</p>
        </div>
        <div class="text-center">
          <p class="text-3xl font-black text-m3-sys-light-primary">{{ totalReads }}</p>
          <p class="text-xs text-m3-sys-light-on-surface-variant mt-0.5">Total Reads</p>
        </div>
      </div>

      <!-- 编辑按钮 -->
      <button
        @click="openEdit"
        class="shrink-0 p-2.5 rounded-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
        title="编辑个人信息"
      >
        <Settings class="w-5 h-5" />
      </button>
    </div>

    <!-- 编辑个人信息表单 -->
    <div v-if="editMode" class="bg-m3-sys-light-surface rounded-[2rem] p-8 mb-8 border border-m3-sys-light-outline-variant">
      <h3 class="text-lg font-bold text-m3-sys-light-on-surface mb-6">编辑个人信息</h3>
      <div class="space-y-4 max-w-sm">
        <div>
          <label class="block text-sm font-medium text-m3-sys-light-on-surface-variant mb-1">用户名</label>
          <input
            v-model="editUsername"
            type="text"
            class="w-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30"
          />
        </div>
        <div class="pt-2 border-t border-m3-sys-light-outline-variant">
          <p class="text-xs text-m3-sys-light-on-surface-variant mb-3">留空则不修改密码</p>
          <label class="block text-sm font-medium text-m3-sys-light-on-surface-variant mb-1">当前密码</label>
          <input
            v-model="editOldPassword"
            type="password"
            placeholder="修改密码时必填"
            class="w-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30 mb-3"
          />
          <label class="block text-sm font-medium text-m3-sys-light-on-surface-variant mb-1">新密码</label>
          <input
            v-model="editNewPassword"
            type="password"
            placeholder="至少 6 位"
            class="w-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30"
          />
        </div>
        <p v-if="editError" class="text-red-500 text-sm">{{ editError }}</p>
        <p v-if="editSuccess" class="text-green-600 text-sm">{{ editSuccess }}</p>
        <div class="flex gap-3 pt-2">
          <button
            @click="submitEdit"
            :disabled="editSubmitting"
            class="px-5 py-2.5 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {{ editSubmitting ? '保存中…' : '保存' }}
          </button>
          <button
            @click="editMode = false"
            class="px-5 py-2.5 bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant rounded-full font-medium hover:bg-m3-sys-light-secondary-container transition-colors"
          >
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 写文章入口 -->
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-bold text-m3-sys-light-on-surface">My Articles</h2>
      <div class="flex items-center gap-2">
        <button
          v-if="selectedArticleIds.length > 0"
          :disabled="bulkDeleting"
          @click="bulkDeleteArticles"
          class="flex items-center gap-2 px-5 py-2.5 bg-m3-sys-light-error-container text-m3-sys-light-on-error-container rounded-full font-medium hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          <Trash2 class="w-4 h-4" />
          <span>删除已选 ({{ selectedArticleIds.length }})</span>
        </button>
        <button
          @click="router.push('/write')"
          class="flex items-center gap-2 px-5 py-2.5 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full font-medium hover:opacity-90 transition-opacity"
        >
          <PenSquare class="w-4 h-4" />
          <span>Write New</span>
        </button>
      </div>
    </div>

    <p v-if="deleteError" class="text-red-500 text-sm mb-4">{{ deleteError }}</p>
    <label v-if="myArticles.length > 0" class="inline-flex items-center gap-2 mb-4 text-sm text-m3-sys-light-on-surface-variant">
      <input
        type="checkbox"
        :checked="allArticlesSelected"
        @change="toggleAllArticles(($event.target as HTMLInputElement).checked)"
        class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary"
      />
      全选
    </label>

    <!-- 文章列表 -->
    <div v-if="articlesPending" class="space-y-4">
      <div v-for="i in 4" :key="i" class="h-24 rounded-2xl bg-m3-sys-light-surface-variant animate-pulse" />
    </div>

    <div v-else-if="myArticles.length === 0" class="text-center py-20 text-m3-sys-light-on-surface-variant">
      No articles yet. <button @click="router.push('/write')" class="text-m3-sys-light-primary hover:underline">Write your first one!</button>
    </div>

    <ul v-else class="space-y-4">
      <li
        v-for="article in myArticles"
        :key="article.id"
        class="flex items-center gap-4 bg-m3-sys-light-surface rounded-2xl px-6 py-4 shadow-sm hover:shadow-md transition-shadow group"
      >
        <input
          type="checkbox"
          :checked="selectedArticleIds.includes(article.id)"
          @change="updateSelection(article.id, ($event.target as HTMLInputElement).checked)"
          class="w-4 h-4 rounded border-m3-sys-light-outline accent-m3-sys-light-primary shrink-0"
        />
        <!-- 缩略图 -->
        <img :src="articleImageUrl(article)" alt="" class="w-16 h-16 rounded-xl object-cover shrink-0 hidden sm:block" />

        <!-- 文章信息 -->
        <div class="flex-1 min-w-0">
          <NuxtLink :to="`/post/${article.id}`" class="font-semibold text-m3-sys-light-on-surface group-hover:text-m3-sys-light-primary transition-colors line-clamp-1">
            {{ article.title }}
          </NuxtLink>
          <p class="text-sm text-m3-sys-light-on-surface-variant mt-0.5 flex items-center gap-3">
            <span class="flex items-center gap-1">
              <BookOpen class="w-3.5 h-3.5" />{{ article.read_count ?? 0 }} reads
            </span>
            <span>·</span>
            <span>{{ formatDate(article.created_at) }}</span>
          </p>
        </div>

        <!-- 状态徽章 -->
        <span
          class="shrink-0 px-2.5 py-1 rounded-full text-xs font-semibold"
          :class="article.status === 'published'
            ? 'bg-green-100 text-green-700'
            : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant'"
        >
          {{ article.status === 'published' ? 'Published' : 'Draft' }}
        </span>

        <!-- 操作按钮 -->
        <div class="flex items-center gap-2 shrink-0">
          <NuxtLink
            :to="`/write?edit=${article.id}`"
            class="p-2 rounded-full hover:bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-primary transition-colors"
            title="Edit"
          >
            <PenSquare class="w-4 h-4" />
          </NuxtLink>
          <button
            @click="deleteArticle(article.id)"
            class="p-2 rounded-full hover:bg-m3-sys-light-error-container text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-on-error-container transition-colors"
            title="Delete"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </li>
    </ul>

    <!-- 分页 -->
    <div v-if="totalPages > 1" class="flex justify-center gap-2 mt-8">
      <button
        v-for="p in totalPages" :key="p"
        @click="page = p"
        class="w-10 h-10 rounded-full font-medium transition-colors"
        :class="p === page
          ? 'bg-m3-sys-light-primary text-m3-sys-light-on-primary'
          : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container'"
      >
        {{ p }}
      </button>
    </div>

    <!-- Admin 入口 -->
    <div v-if="profile?.role === 'admin'" class="mt-10 text-center">
      <NuxtLink
        to="/admin"
        class="inline-flex items-center gap-2 px-6 py-3 bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container rounded-full font-medium hover:opacity-90 transition-opacity"
      >
        <FileText class="w-4 h-4" />
        Go to Admin Dashboard
      </NuxtLink>
    </div>
  </div>
</template>
