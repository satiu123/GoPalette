<script setup lang="ts">
import {
  POST_STATUS_ARCHIVED,
  POST_STATUS_DRAFT,
  POST_STATUS_OFFLINE,
  POST_STATUS_PRIVATE,
  POST_STATUS_PUBLISHED,
  deletePost,
  updatePost
} from '~/composables/useBlogApi'
import {
  fetchUserDashboard,
  formatDashboardCount,
  getErrorMessage,
  toPostStatusColor,
  toPostStatusText,
  toPostVisibilityColor,
  toPostVisibilityText
} from '~/composables/useProfileDashboard'
import type { UserDashboardPost, UserDashboardResponse } from '~/composables/useProfileDashboard'

const toast = useToast()
const router = useRouter()
const route = useRoute()

const { session, isLoggedIn, initAuth, fetchProfile } = useAuth()

const loading = ref(true)
const dashboard = ref<UserDashboardResponse | null>(null)
const dashboardError = ref('')
const updatingPostId = ref('')
const deletingPostId = ref('')
const keyword = ref('')
const statusFilter = ref(String(route.query.status || 'all'))
const sortBy = ref('updated')
const page = ref(1)
const pageSize = 8
const selectedIds = ref<string[]>([])

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
  { label: '私密', value: 'private' },
  { label: '归档', value: 'archived' },
  { label: '下线', value: 'offline' }
]

const sortOptions = [
  { label: '最近编辑', value: 'updated' },
  { label: '发布时间', value: 'created' },
  { label: '阅读最多', value: 'views' },
  { label: '评论最多', value: 'comments' }
]

const statusValueMap: Record<string, number> = {
  published: POST_STATUS_PUBLISHED,
  draft: POST_STATUS_DRAFT,
  private: POST_STATUS_PRIVATE,
  archived: POST_STATUS_ARCHIVED,
  offline: POST_STATUS_OFFLINE
}

const allPosts = computed(() => dashboard.value?.authorPosts || [])

const filteredPosts = computed(() => {
  const query = keyword.value.trim().toLowerCase()

  return allPosts.value
    .filter((post) => {
      const matchesStatus = statusFilter.value === 'all' || post.status === statusValueMap[statusFilter.value]
      const searchable = `${post.title} ${post.summary} ${post.category} ${post.tags.join(' ')}`.toLowerCase()
      return matchesStatus && (!query || searchable.includes(query))
    })
    .sort((a, b) => {
      if (sortBy.value === 'views') return b.views - a.views
      if (sortBy.value === 'comments') return b.comments - a.comments
      if (sortBy.value === 'created') return b.createdAt.localeCompare(a.createdAt)
      return b.updatedAt.localeCompare(a.updatedAt)
    })
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredPosts.value.length / pageSize)))

const visiblePosts = computed(() => {
  const start = (page.value - 1) * pageSize
  return filteredPosts.value.slice(start, start + pageSize)
})

const allVisibleSelected = computed(() => visiblePosts.value.length > 0 && visiblePosts.value.every(post => selectedIds.value.includes(post.id)))

watch([keyword, statusFilter, sortBy], () => {
  page.value = 1
  selectedIds.value = []
})

watch(page, () => {
  selectedIds.value = selectedIds.value.filter(id => visiblePosts.value.some(post => post.id === id))
})

async function loadDashboard(userId: string) {
  dashboardError.value = ''
  try {
    dashboard.value = await fetchUserDashboard(userId)
  } catch (error: unknown) {
    dashboard.value = null
    dashboardError.value = getErrorMessage(error, '暂无法加载文章列表')
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
    const targetId = String(profile?.id || session.value.userId || '')
    if (targetId) {
      await loadDashboard(targetId)
    }
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '获取文章失败',
      description: getErrorMessage(error, '请稍后重试')
    })
  } finally {
    loading.value = false
  }
})

function askConfirm(message: string) {
  if (!import.meta.client) return true
  return window.confirm(message)
}

function toggleVisiblePosts(checked: boolean) {
  const ids = visiblePosts.value.map(post => post.id)
  if (checked) {
    selectedIds.value = Array.from(new Set([...selectedIds.value, ...ids]))
    return
  }

  selectedIds.value = selectedIds.value.filter(id => !ids.includes(id))
}

function togglePostSelected(id: string, checked: boolean) {
  if (checked) {
    selectedIds.value = Array.from(new Set([...selectedIds.value, id]))
    return
  }

  selectedIds.value = selectedIds.value.filter(item => item !== id)
}

async function changePostStatus(item: UserDashboardPost, status: number) {
  if (!item.id || updatingPostId.value) return

  const needsConfirm = status === POST_STATUS_ARCHIVED || status === POST_STATUS_OFFLINE
  if (needsConfirm && !askConfirm(`确认将《${item.title}》${status === POST_STATUS_ARCHIVED ? '归档' : '下线'}吗？`)) return

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

async function changeSelectedStatus(status: number) {
  if (!selectedIds.value.length || updatingPostId.value) return

  const selectedPosts = allPosts.value.filter(post => selectedIds.value.includes(post.id))
  if (!askConfirm(`确认批量更新 ${selectedPosts.length} 篇文章的状态吗？`)) return

  updatingPostId.value = '__batch__'
  try {
    for (const post of selectedPosts) {
      await updatePost(post.id, {
        title: post.title,
        summary: post.summary,
        slug: post.slug,
        content: '',
        status,
        updateMask: 'status'
      })
    }

    selectedIds.value = []
    if (session.value.userId) {
      await loadDashboard(session.value.userId)
    }

    toast.add({
      color: 'success',
      title: '批量状态已更新'
    })
  } catch (error: unknown) {
    toast.add({
      color: 'error',
      title: '批量更新失败',
      description: getErrorMessage(error, '请稍后再试')
    })
  } finally {
    updatingPostId.value = ''
  }
}

async function removePostItem(item: UserDashboardPost) {
  if (!item.id || deletingPostId.value) return
  if (!askConfirm(`确认删除《${item.title}》吗？此操作不可恢复。`)) return

  deletingPostId.value = item.id
  try {
    await deletePost(item.id)
    if (session.value.userId) {
      await loadDashboard(session.value.userId)
    }
    selectedIds.value = selectedIds.value.filter(id => id !== item.id)
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

function getPostMenuItems(item: UserDashboardPost) {
  return [
    {
      label: '发布',
      icon: 'i-lucide-send',
      disabled: item.status === POST_STATUS_PUBLISHED,
      onSelect: () => changePostStatus(item, POST_STATUS_PUBLISHED)
    },
    {
      label: '转为草稿',
      icon: 'i-lucide-file-pen-line',
      disabled: item.status === POST_STATUS_DRAFT,
      onSelect: () => changePostStatus(item, POST_STATUS_DRAFT)
    },
    {
      label: '设为私密',
      icon: 'i-lucide-lock',
      disabled: item.status === POST_STATUS_PRIVATE,
      onSelect: () => changePostStatus(item, POST_STATUS_PRIVATE)
    },
    {
      label: '归档',
      icon: 'i-lucide-archive',
      disabled: item.status === POST_STATUS_ARCHIVED,
      onSelect: () => changePostStatus(item, POST_STATUS_ARCHIVED)
    },
    {
      label: '下线',
      icon: 'i-lucide-cloud-off',
      disabled: item.status === POST_STATUS_OFFLINE,
      onSelect: () => changePostStatus(item, POST_STATUS_OFFLINE)
    },
    {
      label: '删除',
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => removePostItem(item)
    }
  ]
}

useSeoMeta({
  title: '文章管理 - GoPalette',
  description: '搜索、筛选、分页和管理个人文章。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader />

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="flex items-center gap-1 text-sm text-toned">
            <NuxtLink
              to="/profile"
              class="transition-colors hover:text-primary"
            >
              个人中心
            </NuxtLink>
            <span>/</span>
            <span>内容管理</span>
          </p>
          <h1 class="mt-1 text-2xl font-semibold text-highlighted">
            我的文章管理
          </h1>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            to="/profile"
            color="neutral"
            variant="soft"
            icon="i-lucide-arrow-left"
            label="返回概览"
          />
          <UButton
            to="/write"
            icon="i-lucide-plus"
            label="新建文章"
          />
        </div>
      </div>

      <section class="rounded-lg border border-default bg-default">
        <div class="border-b border-default p-4 sm:p-5">
          <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_180px_auto] lg:items-end">
            <UFormField label="搜索文章">
              <UInput
                v-model="keyword"
                icon="i-lucide-search"
                placeholder="搜索标题、摘要、分类或标签"
              />
            </UFormField>
            <UFormField label="状态">
              <USelect
                v-model="statusFilter"
                :items="statusOptions"
              />
            </UFormField>
            <UFormField label="排序">
              <USelect
                v-model="sortBy"
                :items="sortOptions"
              />
            </UFormField>
            <UButton
              color="neutral"
              variant="soft"
              icon="i-lucide-refresh-cw"
              label="刷新"
              :loading="loading"
              @click="session.userId && loadDashboard(session.userId)"
            />
          </div>
        </div>

        <div
          v-if="visiblePosts.length > 0"
          class="flex flex-wrap items-center justify-between gap-3 border-b border-default bg-muted/30 px-4 py-3 sm:px-5"
        >
          <label class="inline-flex items-center gap-2 text-xs font-medium text-toned">
            <input
              type="checkbox"
              class="size-4 rounded border-default accent-[var(--ui-primary)]"
              :checked="allVisibleSelected"
              @change="toggleVisiblePosts(($event.target as HTMLInputElement).checked)"
            >
            当前页全选
            <span
              v-if="selectedIds.length > 0"
              class="text-muted"
            >
              已选 {{ selectedIds.length }}
            </span>
          </label>

          <div class="flex flex-wrap items-center gap-2">
            <UButton
              size="xs"
              color="success"
              variant="soft"
              icon="i-lucide-send"
              label="发布"
              :disabled="selectedIds.length === 0"
              :loading="updatingPostId === '__batch__'"
              @click="changeSelectedStatus(POST_STATUS_PUBLISHED)"
            />
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              icon="i-lucide-file-pen-line"
              label="转草稿"
              :disabled="selectedIds.length === 0"
              :loading="updatingPostId === '__batch__'"
              @click="changeSelectedStatus(POST_STATUS_DRAFT)"
            />
            <UButton
              size="xs"
              color="warning"
              variant="soft"
              icon="i-lucide-archive"
              label="归档"
              :disabled="selectedIds.length === 0"
              :loading="updatingPostId === '__batch__'"
              @click="changeSelectedStatus(POST_STATUS_ARCHIVED)"
            />
          </div>
        </div>

        <div
          v-if="dashboardError"
          class="p-4 sm:p-5"
        >
          <UAlert
            color="error"
            variant="soft"
            icon="i-lucide-circle-alert"
            title="暂时无法拉取文章列表"
            :description="dashboardError"
          />
        </div>

        <div
          v-else-if="loading"
          class="space-y-3 p-4 sm:p-5"
        >
          <USkeleton
            v-for="index in 5"
            :key="index"
            class="h-24 w-full"
          />
        </div>

        <div
          v-else-if="!filteredPosts.length"
          class="p-4 sm:p-5"
        >
          <UAlert
            title="暂无匹配文章"
            description="可以调整搜索或筛选条件，也可以新建一篇文章。"
            icon="i-lucide-file-text"
            color="neutral"
            variant="soft"
          />
        </div>

        <div
          v-else
          class="divide-y divide-default"
        >
          <article
            v-for="item in visiblePosts"
            :key="item.id"
            class="p-4 transition-colors hover:bg-muted/40 sm:p-5"
          >
            <div class="grid gap-3 xl:grid-cols-[auto_minmax(0,1fr)_auto] xl:items-center">
              <input
                type="checkbox"
                class="mt-1 size-4 rounded border-default accent-[var(--ui-primary)] xl:mt-0"
                :checked="selectedIds.includes(item.id)"
                @change="togglePostSelected(item.id, ($event.target as HTMLInputElement).checked)"
              >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="line-clamp-1 text-base font-semibold text-highlighted">
                    {{ item.title }}
                  </p>
                  <UBadge
                    size="xs"
                    :label="toPostStatusText(item.status)"
                    :color="toPostStatusColor(item.status)"
                    variant="subtle"
                  />
                  <UBadge
                    size="xs"
                    :label="toPostVisibilityText(item.status)"
                    :color="toPostVisibilityColor(item.status)"
                    variant="subtle"
                  />
                </div>
                <p class="mt-2 line-clamp-2 text-sm text-toned">
                  {{ item.summary }}
                </p>
                <div class="mt-3 grid gap-2 text-xs text-muted sm:grid-cols-2 xl:grid-cols-4">
                  <span class="inline-flex items-center gap-1"><UIcon
                    name="i-lucide-folder"
                    class="size-3.5"
                  />{{ item.category || '未分类' }}</span>
                  <span class="inline-flex items-center gap-1"><UIcon
                    name="i-lucide-calendar"
                    class="size-3.5"
                  />发布 {{ item.createdAt }}</span>
                  <span class="inline-flex items-center gap-1"><UIcon
                    name="i-lucide-clock-3"
                    class="size-3.5"
                  />编辑 {{ item.updatedAt }}</span>
                  <span class="inline-flex items-center gap-1"><UIcon
                    name="i-lucide-eye"
                    class="size-3.5"
                  />{{ formatDashboardCount(item.views) }} 阅读 · {{ formatDashboardCount(item.comments) }} 评论</span>
                </div>
              </div>

              <div class="flex items-center gap-2 xl:justify-end">
                <UButton
                  :to="`/write?slug=${item.slug}`"
                  size="xs"
                  color="primary"
                  variant="soft"
                  icon="i-lucide-square-pen"
                  label="编辑"
                />
                <UDropdownMenu
                  :items="getPostMenuItems(item)"
                  :content="{ align: 'end' }"
                  :modal="false"
                >
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-ellipsis"
                    label="更多"
                    :loading="updatingPostId === item.id || deletingPostId === item.id"
                  />
                </UDropdownMenu>
              </div>
            </div>
          </article>
        </div>

        <div
          v-if="filteredPosts.length > 0"
          class="flex flex-wrap items-center justify-between gap-3 border-t border-default p-4 sm:p-5"
        >
          <span class="text-xs text-toned">共 {{ filteredPosts.length }} 条，当前第 {{ page }} / {{ totalPages }} 页</span>
          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              icon="i-lucide-chevron-left"
              :disabled="page <= 1"
              @click="page = Math.max(1, page - 1)"
            />
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              icon="i-lucide-chevron-right"
              :disabled="page >= totalPages"
              @click="page = Math.min(totalPages, page + 1)"
            />
          </div>
        </div>
      </section>
    </main>
  </div>
</template>
