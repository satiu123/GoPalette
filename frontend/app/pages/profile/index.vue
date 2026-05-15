<script setup lang="ts">
import { POST_STATUS_DRAFT } from '~/composables/useBlogApi'
import {
  fetchUserDashboard,
  formatDashboardCount,
  formatDashboardDate,
  getErrorMessage,
  toAvatarFallback,
  toPostStatusColor,
  toPostStatusText,
  toPostVisibilityColor,
  toPostVisibilityText
} from '~/composables/useProfileDashboard'
import type { UserDashboardResponse } from '~/composables/useProfileDashboard'

const toast = useToast()
const router = useRouter()

const { session, user, isLoggedIn, initAuth, fetchProfile, logout } = useAuth()

const loading = ref(true)
const dashboard = ref<UserDashboardResponse | null>(null)
const dashboardError = ref('')

const emptyStats = {
  posts: 0,
  published: 0,
  drafts: 0,
  archived: 0,
  views: 0,
  likes: 0,
  comments: 0
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
    createdAt: formatDashboardDate(user.value?.createdAt)
  }
})

const stats = computed(() => dashboard.value?.postStats || emptyStats)

const statsItems = computed(() => [
  {
    label: '已发布文章',
    value: formatDashboardCount(stats.value.published),
    icon: 'i-lucide-check-circle-2',
    helper: stats.value.views > 0 ? `累计 ${formatDashboardCount(stats.value.views)} 次阅读` : '发布后会显示阅读表现'
  },
  {
    label: '草稿',
    value: formatDashboardCount(stats.value.drafts),
    icon: 'i-lucide-pencil-line',
    helper: stats.value.drafts > 0 ? '继续打磨未完成内容' : '暂无待完成草稿'
  },
  {
    label: '总点赞',
    value: formatDashboardCount(stats.value.likes),
    icon: 'i-lucide-heart',
    helper: stats.value.likes > 0 ? '读者认可会汇总在这里' : '获得点赞后会更新'
  },
  {
    label: '新评论',
    value: formatDashboardCount(stats.value.comments),
    icon: 'i-lucide-message-circle',
    helper: stats.value.comments > 0 ? '最近互动可在右侧查看' : '暂无评论互动'
  }
])

const recentPosts = computed(() => [...(dashboard.value?.authorPosts || [])].slice(0, 5))

const draftPosts = computed(() => (dashboard.value?.authorPosts || [])
  .filter(post => post.status === POST_STATUS_DRAFT)
  .slice(0, 3))

const sortedActivePosts = computed(() => [...(dashboard.value?.topPosts || [])]
  .sort((a, b) => (b.views + b.likes * 3 + b.comments * 5) - (a.views + a.likes * 3 + a.comments * 5))
  .slice(0, 4))

const postSlugById = computed(() => {
  const entries = [
    ...(dashboard.value?.authorPosts || []),
    ...(dashboard.value?.topPosts || [])
  ]
    .filter(post => post.id && post.slug)
    .map(post => [post.id, post.slug] as const)

  return new Map(entries)
})

function commentTarget(comment: { id: string, postId: string }) {
  const slug = postSlugById.value.get(comment.postId)
  return slug ? `/posts/${slug}#comment-${comment.id}` : '/posts'
}

async function loadDashboard(userId: string) {
  dashboardError.value = ''
  try {
    dashboard.value = await fetchUserDashboard(userId)
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
    const targetId = String(profile?.id || session.value.userId || '')
    if (targetId) {
      await loadDashboard(targetId)
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

async function onLogout() {
  await logout()
  await router.push('/login')
}

useSeoMeta({
  title: '个人中心 - GoPalette',
  description: '查看个人写作概览、最近文章和互动数据。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton
        color="neutral"
        variant="subtle"
        icon="i-lucide-log-out"
        class="sm:hidden"
        @click="onLogout"
      />
      <UButton
        color="neutral"
        variant="subtle"
        icon="i-lucide-log-out"
        label="退出登录"
        class="hidden sm:inline-flex"
        @click="onLogout"
      />
    </AppHeader>

    <main class="mx-auto w-full max-w-6xl px-4 pb-20 pt-10 sm:px-14">
      <section class="rounded-lg border border-default bg-muted/30 p-5 sm:p-6">
        <div
          v-if="loading"
          class="space-y-3"
        >
          <USkeleton class="h-8 w-44" />
          <USkeleton class="h-4 w-72" />
        </div>

        <div
          v-else
          class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between"
        >
          <div class="flex min-w-0 items-center gap-4">
            <UAvatar
              :src="displayUser.avatarURL"
              :alt="displayUser.username"
              :text="toAvatarFallback(displayUser.username)"
              size="3xl"
            />
            <div class="min-w-0">
              <h1 class="truncate text-2xl font-semibold text-highlighted sm:text-3xl">
                {{ displayUser.username }}
              </h1>
              <p class="mt-1 text-sm text-toned">
                {{ displayUser.email || '暂无邮箱' }} · 注册时间 {{ displayUser.createdAt }}
              </p>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <UButton
              to="/write"
              icon="i-lucide-square-pen"
              label="写新文章"
            />
            <UButton
              to="/profile/posts"
              color="neutral"
              variant="soft"
              icon="i-lucide-files"
              label="管理文章"
            />
            <UButton
              to="/profile/settings"
              color="neutral"
              variant="soft"
              icon="i-lucide-settings"
              label="资料设置"
            />
          </div>
        </div>
      </section>

      <UAlert
        v-if="dashboardError"
        class="mt-5"
        color="error"
        variant="soft"
        icon="i-lucide-circle-alert"
        title="个人中心暂不可用"
        :description="dashboardError"
      />

      <section class="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <article
          v-for="item in statsItems"
          :key="item.label"
          class="rounded-lg border border-default bg-default p-4"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs text-toned">
                {{ item.label }}
              </p>
              <p class="mt-1 text-2xl font-semibold text-highlighted">
                {{ item.value }}
              </p>
            </div>
            <UIcon
              :name="item.icon"
              class="size-5 shrink-0 text-primary"
            />
          </div>
          <p class="mt-3 truncate text-xs text-muted">
            {{ item.helper }}
          </p>
        </article>
      </section>

      <section class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div class="space-y-6">
          <section class="rounded-lg border border-default bg-default">
            <div class="flex items-center justify-between gap-3 border-b border-default p-4 sm:p-5">
              <div>
                <h2 class="text-base font-semibold text-highlighted">
                  最近文章
                </h2>
                <p class="mt-1 text-xs text-toned">
                  最近创建或编辑的内容
                </p>
              </div>
              <UButton
                to="/profile/posts"
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-arrow-right"
                label="全部"
              />
            </div>

            <div
              v-if="!recentPosts.length"
              class="p-5"
            >
              <UAlert
                title="暂无文章"
                description="写下第一篇文章后，最近内容会显示在这里。"
                icon="i-lucide-file-text"
                color="neutral"
                variant="soft"
              />
            </div>

            <div
              v-else
              class="divide-y divide-default"
            >
              <NuxtLink
                v-for="post in recentPosts"
                :key="post.id"
                :to="post.slug ? `/write?slug=${post.slug}` : '/profile/posts'"
                class="block p-4 transition-colors hover:bg-muted/40 sm:p-5"
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <p class="line-clamp-1 text-base font-semibold text-highlighted">
                        {{ post.title }}
                      </p>
                      <UBadge
                        size="xs"
                        :label="toPostStatusText(post.status)"
                        :color="toPostStatusColor(post.status)"
                        variant="subtle"
                      />
                      <UBadge
                        size="xs"
                        :label="toPostVisibilityText(post.status)"
                        :color="toPostVisibilityColor(post.status)"
                        variant="subtle"
                      />
                    </div>
                    <p class="mt-2 line-clamp-2 text-sm text-toned">
                      {{ post.summary }}
                    </p>
                    <p class="mt-3 text-xs text-muted">
                      发布 {{ post.createdAt }} · 最后编辑 {{ post.updatedAt }}
                    </p>
                  </div>
                  <div class="flex shrink-0 items-center gap-3 text-xs text-toned">
                    <span class="inline-flex items-center gap-1"><UIcon
                      name="i-lucide-eye"
                      class="size-3.5"
                    />{{ formatDashboardCount(post.views) }}</span>
                    <span class="inline-flex items-center gap-1"><UIcon
                      name="i-lucide-message-circle"
                      class="size-3.5"
                    />{{ formatDashboardCount(post.comments) }}</span>
                  </div>
                </div>
              </NuxtLink>
            </div>
          </section>

          <section class="rounded-lg border border-default bg-default p-4 sm:p-5">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-highlighted">
                  最近编辑草稿
                </h2>
                <p class="mt-1 text-xs text-toned">
                  可以继续完善后发布
                </p>
              </div>
              <UButton
                to="/profile/posts?status=draft"
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-list-filter"
                label="筛选"
              />
            </div>

            <div
              v-if="!draftPosts.length"
              class="mt-4 rounded-lg border border-dashed border-default p-4 text-sm text-toned"
            >
              暂无草稿，新的写作计划会在这里等待你继续推进。
            </div>

            <div
              v-else
              class="mt-4 grid gap-3 sm:grid-cols-3"
            >
              <NuxtLink
                v-for="post in draftPosts"
                :key="post.id"
                :to="post.slug ? `/write?slug=${post.slug}` : '/write'"
                class="rounded-lg border border-default p-3 transition-colors hover:border-primary/50"
              >
                <p class="line-clamp-1 text-sm font-medium text-highlighted">
                  {{ post.title }}
                </p>
                <p class="mt-2 line-clamp-2 text-xs text-toned">
                  {{ post.summary }}
                </p>
              </NuxtLink>
            </div>
          </section>
        </div>

        <aside class="space-y-6">
          <section class="rounded-lg border border-default bg-default p-4 sm:p-5">
            <h2 class="text-base font-semibold text-highlighted">
              个人资料摘要
            </h2>
            <dl class="mt-4 space-y-3 text-sm">
              <div class="flex justify-between gap-3">
                <dt class="text-toned">
                  显示名称
                </dt>
                <dd class="truncate text-highlighted">
                  {{ displayUser.username }}
                </dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-toned">
                  邮箱
                </dt>
                <dd class="truncate text-highlighted">
                  {{ displayUser.email || '-' }}
                </dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-toned">
                  文章总数
                </dt>
                <dd class="text-highlighted">
                  {{ formatDashboardCount(stats.posts) }}
                </dd>
              </div>
            </dl>
            <UButton
              to="/profile/settings"
              block
              class="mt-4"
              color="neutral"
              variant="soft"
              icon="i-lucide-user-pen"
              label="编辑资料"
            />
          </section>

          <section class="rounded-lg border border-default bg-default p-4 sm:p-5">
            <h2 class="text-base font-semibold text-highlighted">
              热门文章
            </h2>

            <div
              v-if="!sortedActivePosts.length"
              class="mt-4 rounded-lg border border-dashed border-default p-4 text-sm text-toned"
            >
              发布更多文章后，这里会按阅读、点赞和评论表现展示内容。
            </div>

            <div
              v-else
              class="mt-4 space-y-3"
            >
              <NuxtLink
                v-for="post in sortedActivePosts"
                :key="post.id"
                :to="post.slug ? `/posts/${post.slug}` : '/posts'"
                class="block rounded-lg border border-default p-3 transition-colors hover:border-primary/50"
              >
                <p class="line-clamp-1 text-sm font-medium text-highlighted">
                  {{ post.title }}
                </p>
                <p class="mt-2 text-xs text-toned">
                  {{ formatDashboardCount(post.views) }} 阅读 · {{ formatDashboardCount(post.likes) }} 点赞 · {{ formatDashboardCount(post.comments) }} 评论
                </p>
              </NuxtLink>
            </div>
          </section>

          <section class="rounded-lg border border-default bg-default p-4 sm:p-5">
            <h2 class="text-base font-semibold text-highlighted">
              最近评论
            </h2>

            <div
              v-if="!dashboard?.recentComments?.length"
              class="mt-4 rounded-lg border border-dashed border-default p-4"
            >
              <p class="text-sm font-medium text-highlighted">
                暂无评论互动
              </p>
              <p class="mt-1 text-xs text-toned">
                当读者评论你的文章后，会显示在这里。
              </p>
            </div>

            <div
              v-else
              class="mt-4 space-y-3"
            >
              <NuxtLink
                v-for="comment in dashboard.recentComments"
                :key="comment.id"
                :to="commentTarget(comment)"
                class="block rounded-lg border border-default px-3 py-2 transition-colors hover:border-primary/50 hover:bg-muted/40"
              >
                <p class="line-clamp-2 text-sm text-highlighted">
                  {{ comment.content }}
                </p>
                <p class="mt-1 text-xs text-toned">
                  {{ comment.createdAt }} · {{ comment.authorName }}
                </p>
              </NuxtLink>
            </div>
          </section>
        </aside>
      </section>
    </main>
  </div>
</template>
