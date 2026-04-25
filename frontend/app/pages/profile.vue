<script setup lang="ts">
import { POST_STATUS_ARCHIVED, POST_STATUS_DRAFT, POST_STATUS_PUBLISHED, deletePost, updatePost } from '~/composables/useBlogApi'

interface UserDashboardInfo {
  id: string
  username: string
  email: string
  avatarURL: string
  bio: string
  location: string
  createdAt: string
}

interface UserDashboardStats {
  posts: number
  published: number
  drafts: number
  archived: number
  views: number
  likes: number
  comments: number
}

interface UserDashboardPost {
  id: string
  title: string
  slug: string
  summary: string
  status: number
  views: number
  likes: number
  comments: number
  createdAt: string
}

interface UserDashboardComment {
  id: string
  postId: string
  content: string
  createdAt: string
  authorName: string
}

interface UserDashboardResponse {
  userInfo: UserDashboardInfo | null
  postStats: UserDashboardStats
  topPosts: UserDashboardPost[]
  authorPosts: UserDashboardPost[]
  recentComments: UserDashboardComment[]
}

type DataRecord = Record<string, unknown>

const toast = useToast()
const router = useRouter()

const { session, user, isLoggedIn, initAuth, authFetch, fetchProfile, updateProfile, logout } = useAuth()

const loading = ref(true)
const saving = ref(false)
const updatingPostId = ref('')
const deletingPostId = ref('')
const dashboard = ref<UserDashboardResponse | null>(null)
const dashboardError = ref('')

const form = reactive({
  username: '',
  email: '',
  avatarURL: ''
})

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

function toPostStatus(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) return value

  const text = String(value || '').trim().toUpperCase()
  if (!text) return POST_STATUS_DRAFT
  if (text === 'PUBLISHED') return POST_STATUS_PUBLISHED
  if (text === 'ARCHIVED') return POST_STATUS_ARCHIVED
  if (text === 'DRAFT') return POST_STATUS_DRAFT

  const parsed = Number(text)
  return Number.isFinite(parsed) ? parsed : POST_STATUS_DRAFT
}

function formatDate(input: unknown, withTime = false) {
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

  return new Intl.DateTimeFormat('zh-CN', withTime
    ? {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      }
    : {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      }).format(date)
}

function formatCount(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value)
}

function getErrorMessage(error: unknown, fallback: string) {
  if (!error || typeof error !== 'object') return fallback
  const typed = error as { message?: unknown, data?: { message?: unknown } }
  return toText(typed.data?.message ?? typed.message, fallback)
}

function toPostStatusText(value: number) {
  if (value === 1) return '已发布'
  if (value === 2) return '已归档'
  return '草稿'
}

function toPostStatusColor(value: number) {
  if (value === 1) return 'success'
  if (value === 2) return 'warning'
  return 'neutral'
}

function toAvatarFallback(name: string) {
  const text = name.trim()
  if (!text) return '我'
  return text.slice(0, 1).toUpperCase()
}

function normalizeDashboard(raw: DataRecord): UserDashboardResponse {
  const userInfoRaw = toRecord(pick(raw, ['userInfo', 'user_info'])) || {}
  const statsRaw = toRecord(pick(raw, ['postStats', 'post_stats'])) || {}
  const topPostsRaw = toRecordArray(pick(raw, ['topPosts', 'top_posts']))
  const commentsRaw = toRecordArray(pick(raw, ['recentComments', 'recent_comments']))

  const normalizedUser: UserDashboardInfo | null = Object.keys(userInfoRaw).length
    ? {
        id: toText(pick(userInfoRaw, ['id'])),
        username: toText(pick(userInfoRaw, ['username']), '未命名用户'),
        email: toText(pick(userInfoRaw, ['email'])),
        avatarURL: toText(pick(userInfoRaw, ['avatarURL', 'avatarUrl', 'avatar_u_r_l', 'avatar_url'])),
        bio: toText(pick(userInfoRaw, ['bio'])),
        location: toText(pick(userInfoRaw, ['location'])),
        createdAt: formatDate(pick(userInfoRaw, ['createdAt', 'created_at']))
      }
    : null

  return {
    userInfo: normalizedUser,
    postStats: {
      posts: toNumber(pick(statsRaw, ['posts'])),
      published: toNumber(pick(statsRaw, ['published'])),
      drafts: toNumber(pick(statsRaw, ['drafts'])),
      archived: toNumber(pick(statsRaw, ['archived'])),
      views: toNumber(pick(statsRaw, ['views'])),
      likes: toNumber(pick(statsRaw, ['likes'])),
      comments: toNumber(pick(statsRaw, ['comments']))
    },
    topPosts: topPostsRaw.map(item => ({
      id: toText(pick(item, ['id'])),
      title: toText(pick(item, ['title']), '未命名文章'),
      slug: toText(pick(item, ['slug'])),
      summary: toText(pick(item, ['summary']), '暂无摘要'),
      status: toPostStatus(pick(item, ['status'])),
      views: toNumber(pick(item, ['viewCount', 'view_count'])),
      likes: toNumber(pick(item, ['likeCount', 'like_count'])),
      comments: toNumber(pick(item, ['commentCount', 'comment_count'])),
      createdAt: formatDate(pick(item, ['createdAt', 'created_at']))
    })),
    authorPosts: toRecordArray(pick(raw, ['authorPosts', 'author_posts'])).map(item => ({
      id: toText(pick(item, ['id'])),
      title: toText(pick(item, ['title']), '未命名文章'),
      slug: toText(pick(item, ['slug'])),
      summary: toText(pick(item, ['summary']), '暂无摘要'),
      status: toPostStatus(pick(item, ['status'])),
      views: toNumber(pick(item, ['viewCount', 'view_count'])),
      likes: toNumber(pick(item, ['likeCount', 'like_count'])),
      comments: toNumber(pick(item, ['commentCount', 'comment_count'])),
      createdAt: formatDate(pick(item, ['createdAt', 'created_at']))
    })),
    recentComments: commentsRaw.map((item) => {
      const author = toRecord(item.author) || {}
      return {
        id: toText(pick(item, ['id'])),
        postId: toText(pick(item, ['postId', 'post_id'])),
        content: toText(pick(item, ['content']), '暂无评论内容'),
        createdAt: formatDate(pick(item, ['createdAt', 'created_at']), true),
        authorName: toText(pick(author, ['name']), '匿名用户')
      }
    })
  }
}

const displayUser = computed(() => {
  const current = dashboard.value?.userInfo
  if (current) return current

  return {
    id: user.value?.id || '',
    username: user.value?.username || '未命名用户',
    email: user.value?.email || '',
    avatarURL: user.value?.avatarURL || '',
    bio: '',
    location: '',
    createdAt: formatDate(user.value?.createdAt)
  }
})

const statsItems = computed(() => {
  const stats = dashboard.value?.postStats || {
    posts: 0,
    published: 0,
    drafts: 0,
    archived: 0,
    views: 0,
    likes: 0,
    comments: 0
  }

  return [
    { label: '文章总数', value: formatCount(stats.posts), icon: 'i-lucide-file-text' },
    { label: '已发布', value: formatCount(stats.published), icon: 'i-lucide-check-circle-2' },
    { label: '草稿', value: formatCount(stats.drafts), icon: 'i-lucide-pencil-line' },
    { label: '已归档', value: formatCount(stats.archived), icon: 'i-lucide-archive' },
    { label: '总阅读', value: formatCount(stats.views), icon: 'i-lucide-eye' },
    { label: '总点赞', value: formatCount(stats.likes), icon: 'i-lucide-heart' },
    { label: '总评论', value: formatCount(stats.comments), icon: 'i-lucide-message-circle' }
  ]
})

const canManagePosts = computed(() => {
  if (!isLoggedIn.value) return false
  return String(session.value.userId || '') === String(displayUser.value.id || '')
})

function fillFormFromUser() {
  form.username = user.value?.username || displayUser.value.username || ''
  form.email = user.value?.email || displayUser.value.email || ''
  form.avatarURL = user.value?.avatarURL || displayUser.value.avatarURL || ''
}

async function loadDashboard(userId: string) {
  dashboardError.value = ''
  try {
    const response = await authFetch<DataRecord>(`/api/user/full-profile/${userId}`, {
      method: 'GET'
    })
    dashboard.value = normalizeDashboard(response || {})
  } catch (error: unknown) {
    dashboard.value = null
    dashboardError.value = getErrorMessage(error, '暂无法加载聚合信息')
  }
}

onMounted(async () => {
  initAuth()

  if (!isLoggedIn.value) {
    await router.replace('/login')
    return
  }

  try {
    const profile = await fetchProfile()
    if (profile) {
      form.username = profile.username || ''
      form.email = profile.email || ''
      form.avatarURL = profile.avatarURL || ''
    }

    const targetId = String(profile?.id || session.value.userId || '')
    if (targetId) {
      await loadDashboard(targetId)
      fillFormFromUser()
    }
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '获取个人信息失败',
      description: getErrorMessage(error, '请稍后重试')
    })
  } finally {
    loading.value = false
  }
})

async function onSave() {
  if (saving.value) return

  saving.value = true
  try {
    await updateProfile({
      username: form.username.trim(),
      email: form.email.trim(),
      avatarURL: form.avatarURL.trim()
    })

    const targetId = session.value.userId
    if (targetId) {
      await loadDashboard(targetId)
    }

    toast.add({
      color: 'success',
      title: '个人信息已更新'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '更新失败',
      description: getErrorMessage(error, '请稍后再试')
    })
  } finally {
    saving.value = false
  }
}

function askConfirm(message: string) {
  if (!import.meta.client) return true
  return window.confirm(message)
}

async function changePostStatus(item: UserDashboardPost, status: number) {
  if (!canManagePosts.value || !item.id || updatingPostId.value) return

  updatingPostId.value = item.id
  try {
    await updatePost(item.id, {
      title: item.title,
      summary: item.summary,
      slug: item.slug,
      content: '',
      status,
      updateMask: 'status'
    })

    if (session.value.userId) {
      await loadDashboard(session.value.userId)
    }

    toast.add({
      color: 'success',
      title: '文章状态已更新'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '更新状态失败',
      description: getErrorMessage(error, '请稍后再试')
    })
  } finally {
    updatingPostId.value = ''
  }
}

async function removePostItem(item: UserDashboardPost) {
  if (!canManagePosts.value || !item.id || deletingPostId.value) return
  if (!askConfirm('确认删除这篇文章吗？此操作不可恢复。')) return

  deletingPostId.value = item.id
  try {
    await deletePost(item.id)
    if (session.value.userId) {
      await loadDashboard(session.value.userId)
    }
    toast.add({
      color: 'success',
      title: '文章已删除'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '删除失败',
      description: getErrorMessage(error, '请稍后再试')
    })
  } finally {
    deletingPostId.value = ''
  }
}

async function onLogout() {
  await logout()
  await router.push('/login')
}

useSeoMeta({
  title: '个人主页 - GoPalette',
  description: '管理你的个人资料，查看写作与互动数据。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton color="neutral" variant="subtle" icon="i-lucide-log-out" class="sm:hidden" @click="onLogout" />
      <UButton color="neutral" variant="subtle" icon="i-lucide-log-out" label="退出登录" class="hidden sm:inline-flex" @click="onLogout" />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="rounded-2xl border border-default bg-muted/30 p-6 sm:p-8">
        <div v-if="loading" class="space-y-3">
          <div class="loading-shimmer h-8 w-44 rounded" />
          <div class="loading-shimmer h-4 w-72 rounded" />
        </div>

        <div v-else class="flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-4">
            <div class="h-16 w-16 overflow-hidden rounded-full border border-default bg-muted">
              <img
                v-if="displayUser.avatarURL"
                :src="displayUser.avatarURL"
                :alt="displayUser.username"
                class="h-full w-full object-cover"
              >
              <div
                v-else
                class="flex h-full w-full items-center justify-center text-lg font-semibold text-toned"
              >
                {{ toAvatarFallback(displayUser.username) }}
              </div>
            </div>

            <div>
              <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
                {{ displayUser.username }}
              </h1>
              <p class="mt-1 text-sm text-toned">
                {{ displayUser.email || '暂无邮箱' }} · ID {{ displayUser.id || '-' }}
              </p>
              <p class="mt-1 text-xs text-toned">
                注册时间：{{ displayUser.createdAt }}
              </p>
            </div>
          </div>

          <UButton to="/write" icon="i-lucide-square-pen" label="写新文章" />
        </div>
      </section>

      <section class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <UCard
          v-for="item in statsItems"
          :key="item.label"
          :ui="{ body: 'p-4' }"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-xs text-toned">
                {{ item.label }}
              </p>
              <p class="mt-1 text-xl font-semibold text-highlighted">
                {{ item.value }}
              </p>
            </div>
            <UIcon :name="item.icon" class="text-xl text-primary" />
          </div>
        </UCard>
      </section>

      <section class="mt-8 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <UCard>
          <template #header>
            <div class="space-y-1">
              <h2 class="text-lg font-semibold text-highlighted">
                编辑个人资料
              </h2>
              <p class="text-sm text-toned">
                这些信息会用于站点作者展示与互动场景。
              </p>
            </div>
          </template>

          <form class="space-y-4" @submit.prevent="onSave">
            <UFormField label="用户名" name="username">
              <UInput v-model="form.username" required />
            </UFormField>

            <UFormField label="邮箱" name="email">
              <UInput v-model="form.email" type="email" required />
            </UFormField>

            <UFormField label="头像 URL" name="avatarURL">
              <UInput v-model="form.avatarURL" placeholder="https://example.com/avatar.png" />
            </UFormField>

            <UButton type="submit" :loading="saving" label="保存修改" />
          </form>
        </UCard>

        <div class="space-y-6">
          <UCard>
            <template #header>
              <h2 class="text-lg font-semibold text-highlighted">
                热门文章
              </h2>
            </template>

            <div v-if="dashboardError" class="text-sm text-error">
              {{ dashboardError }}
            </div>

            <div v-else-if="!dashboard?.topPosts?.length" class="text-sm text-toned">
              暂无热门文章，先去发布第一篇内容吧。
            </div>

            <div v-else class="space-y-3">
              <NuxtLink
                v-for="post in dashboard.topPosts"
                :key="post.id"
                :to="post.slug ? `/posts/${post.slug}` : '/posts'"
                class="motion-card motion-panel block rounded-xl border border-default bg-default p-4 hover:border-primary/40"
              >
                <div class="flex items-start justify-between gap-2">
                  <p class="line-clamp-1 text-sm font-semibold text-highlighted">
                    {{ post.title }}
                  </p>
                  <UBadge
                    size="xs"
                    :label="toPostStatusText(post.status)"
                    :color="toPostStatusColor(post.status)"
                    variant="subtle"
                  />
                </div>
                <p class="mt-1 line-clamp-2 text-xs text-toned">
                  {{ post.summary }}
                </p>
                <div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-toned">
                  <span>{{ post.createdAt }}</span>
                  <span>· 👁 {{ formatCount(post.views) }}</span>
                  <span>· ❤️ {{ formatCount(post.likes) }}</span>
                  <span>· 💬 {{ formatCount(post.comments) }}</span>
                </div>
              </NuxtLink>
            </div>
          </UCard>

          <UCard>
            <template #header>
              <h2 class="text-lg font-semibold text-highlighted">
                最近评论
              </h2>
            </template>

            <div v-if="dashboardError" class="text-sm text-toned">
              评论流暂不可用
            </div>

            <div v-else-if="!dashboard?.recentComments?.length" class="text-sm text-toned">
              你最近还没有评论互动。
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="comment in dashboard.recentComments"
                :key="comment.id"
                class="rounded-xl border border-default px-4 py-3"
              >
                <p class="line-clamp-2 text-sm text-highlighted">
                  {{ comment.content }}
                </p>
                <p class="mt-1 text-xs text-toned">
                  {{ comment.createdAt }} · {{ comment.authorName }}
                </p>
              </div>
            </div>
          </UCard>
        </div>
      </section>

      <section class="mt-8">
        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-highlighted">
                我的文章管理
              </h2>
              <UButton to="/write" size="xs" icon="i-lucide-plus" label="新建文章" />
            </div>
          </template>

          <div v-if="dashboardError" class="text-sm text-toned">
            暂时无法拉取文章列表
          </div>

          <div v-else-if="!dashboard?.authorPosts?.length" class="text-sm text-toned">
            你还没有文章，点击右上角开始写作。
          </div>

          <div v-else class="space-y-3">
            <article
              v-for="item in dashboard.authorPosts"
              :key="item.id"
              class="rounded-xl border border-default p-4"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <p class="text-base font-semibold text-highlighted">
                      {{ item.title }}
                    </p>
                    <UBadge
                      size="xs"
                      :label="toPostStatusText(item.status)"
                      :color="toPostStatusColor(item.status)"
                      variant="subtle"
                    />
                  </div>
                  <p class="text-xs text-toned">
                    {{ item.createdAt }} · 👁 {{ formatCount(item.views) }} · ❤️ {{ formatCount(item.likes) }} · 💬 {{ formatCount(item.comments) }}
                  </p>
                </div>

                <div class="flex flex-wrap items-center gap-2">
                  <UButton
                    :to="`/write?slug=${item.slug}`"
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-square-pen"
                    label="编辑"
                  />
                  <UButton
                    v-if="canManagePosts && item.status !== POST_STATUS_PUBLISHED"
                    size="xs"
                    color="success"
                    variant="soft"
                    :loading="updatingPostId === item.id"
                    label="发布"
                    @click="changePostStatus(item, POST_STATUS_PUBLISHED)"
                  />
                  <UButton
                    v-if="canManagePosts && item.status !== POST_STATUS_DRAFT"
                    size="xs"
                    color="neutral"
                    variant="soft"
                    :loading="updatingPostId === item.id"
                    label="转草稿"
                    @click="changePostStatus(item, POST_STATUS_DRAFT)"
                  />
                  <UButton
                    v-if="canManagePosts && item.status !== POST_STATUS_ARCHIVED"
                    size="xs"
                    color="warning"
                    variant="soft"
                    :loading="updatingPostId === item.id"
                    label="归档"
                    @click="changePostStatus(item, POST_STATUS_ARCHIVED)"
                  />
                  <UButton
                    v-if="canManagePosts"
                    size="xs"
                    color="error"
                    variant="ghost"
                    :loading="deletingPostId === item.id"
                    icon="i-lucide-trash-2"
                    label="删除"
                    @click="removePostItem(item)"
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
