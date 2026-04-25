<script setup lang="ts">
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { POST_STATUS_PUBLISHED, createComment, deleteComment, fetchComments, fetchPostBySlug, fetchPosts } from '~/composables/useBlogApi'
import type { CommentInfo } from '~/composables/useBlogApi'

const { extension: Emoji } = useEditorEmojis()
const toast = useToast()
const { initAuth, isLoggedIn, session, user, fetchProfile } = useAuth()

const route = useRoute()
const slug = computed(() => String(route.params.slug || ''))

const { data: postData } = await useAsyncData(
  () => `post-${slug.value}`,
  () => fetchPostBySlug(slug.value),
  {
    watch: [slug]
  }
)

if (!postData.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '文章不存在'
  })
}

const post = computed(() => postData.value)

const comments = ref<CommentInfo[]>([])
const commentsTotal = ref(0)
const commentsLoading = ref(false)
const submittingComment = ref(false)
const deletingCommentId = ref('')
const commentText = ref('')

function formatCommentTime(input?: string) {
  if (!input) return '刚刚'

  const date = new Date(input)
  if (Number.isNaN(date.getTime())) return '刚刚'

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function canDeleteComment(item: CommentInfo) {
  if (!isLoggedIn.value) return false
  const currentUserId = String(session.value.userId || user.value?.id || '')
  if (!currentUserId) return false

  return item.userId === currentUserId
}

async function loadComments() {
  if (!post.value?.id) return

  commentsLoading.value = true
  try {
    const response = await fetchComments(post.value.id, 1, 100)
    comments.value = response.comments || []
    commentsTotal.value = response.total || comments.value.length
  } finally {
    commentsLoading.value = false
  }
}

async function submitComment() {
  const content = commentText.value.trim()
  if (!content) {
    toast.add({ title: '评论内容不能为空', color: 'warning' })
    return
  }

  if (!post.value?.id) {
    toast.add({ title: '文章信息不存在，无法评论', color: 'error' })
    return
  }

  if (!isLoggedIn.value) {
    toast.add({ title: '请先登录后再评论', color: 'warning' })
    await navigateTo(`/login?redirect=/posts/${slug.value}`)
    return
  }

  submittingComment.value = true
  try {
    await createComment({
      postId: post.value.id,
      content
    })

    commentText.value = ''
    toast.add({ title: '评论发布成功', color: 'success' })
    await loadComments()
  } catch (error: any) {
    toast.add({
      title: '评论发布失败',
      description: error?.data?.message || error?.message || '请稍后重试',
      color: 'error'
    })
  } finally {
    submittingComment.value = false
  }
}

async function removeComment(id: string) {
  if (!id) return

  deletingCommentId.value = id
  try {
    await deleteComment(id)
    toast.add({ title: '评论已删除', color: 'success' })
    await loadComments()
  } catch (error: any) {
    toast.add({
      title: '删除评论失败',
      description: error?.data?.message || error?.message || '请稍后重试',
      color: 'error'
    })
  } finally {
    deletingCommentId.value = ''
  }
}

onMounted(async () => {
  initAuth()

  if (session.value.userId && !user.value) {
    await fetchProfile(session.value.userId)
  }

  await loadComments()
})

watch(post, async (value) => {
  if (!value?.id) return
  await loadComments()
})

const { data: relatedData } = await useAsyncData(
  () => `related-${slug.value}`,
  () => fetchPosts(1, 40),
  {
    watch: [slug]
  }
)

const relatedPosts = computed(() => {
  const current = post.value
  if (!current) return []

  return (relatedData.value?.posts || [])
    .filter(item => item.status === POST_STATUS_PUBLISHED)
    .filter(item => item.slug !== current.slug)
    .filter(item => item.categoryId === current.categoryId || item.category === current.category)
    .slice(0, 3)
})

const viewerExtensions = [
  CodeBlockShiki.configure({
    defaultTheme: 'material-theme',
    themes: {
      light: 'material-theme-lighter',
      dark: 'material-theme-palenight'
    }
  }),
  Emoji,
  TableKit,
  TaskList,
  TaskItem
]

const seoTitle = computed(() => post.value?.title || '文章详情')
const seoDescription = computed(() => post.value?.summary || '')
const seoImage = computed(() => post.value?.cover || '')

useSeoMeta({
  title: seoTitle,
  description: seoDescription,
  ogTitle: seoTitle,
  ogDescription: seoDescription,
  ogImage: seoImage,
  twitterCard: 'summary_large_image'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton to="/write" icon="i-lucide-square-pen" size="sm" class="sm:hidden" />
      <UButton to="/write" icon="i-lucide-square-pen" label="继续写作" size="sm" class="hidden sm:inline-flex" />
    </AppHeader>

    <main v-if="post" class="mx-auto w-full max-w-5xl px-4 pb-20 pt-10 sm:px-14">
      <article class="rounded-2xl border border-default bg-default p-6 sm:p-10">
        <div class="space-y-5 border-b border-default pb-8">
          <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
            <UBadge :label="post.category" color="primary" variant="subtle" />
            <span>{{ post.publishedAt }}</span>
            <span>·</span>
            <span>{{ post.readingMinutes }} 分钟</span>
            <span>·</span>
            <span>{{ post.author }}</span>
          </div>

          <h1 class="text-3xl font-semibold tracking-tight text-highlighted sm:text-4xl">
            {{ post.title }}
          </h1>

          <p class="text-base text-toned sm:text-lg">
            {{ post.summary }}
          </p>

          <img :src="post.cover" :alt="post.title" class="h-60 w-full rounded-xl object-cover">
        </div>

        <div class="mt-8">
          <p v-if="!post.content?.trim()" class="leading-7 text-toned">
            暂无正文内容。
          </p>

          <UEditor v-else :model-value="post.content" content-type="markdown" :editable="false"
            :extensions="viewerExtensions" class="min-h-0" :ui="{
              base: 'p-4 sm:p-14',
              content: 'max-w-4xl mx-auto'
            }" />
        </div>

        <div class="mt-10 flex flex-wrap gap-2 border-t border-default pt-6">
          <UBadge v-for="tag in post.tags" :key="tag" :label="`#${tag}`" color="neutral" variant="outline" />
        </div>
      </article>

      <section class="mt-10">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-highlighted">
            相关文章
          </h2>
          <UButton to="/posts" size="xs" color="neutral" variant="ghost" trailing-icon="i-lucide-arrow-right"
            label="返回列表" />
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <NuxtLink v-for="item in relatedPosts" :key="item.id" :to="`/posts/${item.slug}`"
            class="rounded-xl border border-default bg-default p-4 transition-all hover:-translate-y-0.5 hover:border-primary/40">
            <p class="text-xs text-toned">
              {{ item.publishedAt }}
            </p>
            <p class="mt-2 line-clamp-2 text-sm font-semibold text-highlighted">
              {{ item.title }}
            </p>
          </NuxtLink>
        </div>
      </section>

      <section class="mt-10 rounded-2xl border border-default bg-default p-6 sm:p-8">
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-xl font-semibold text-highlighted">
            评论（{{ commentsTotal || comments.length }}）
          </h2>
          <UBadge v-if="commentsLoading" label="加载中" color="neutral" variant="soft" />
        </div>

        <div class="mt-5 space-y-3">
          <UTextarea v-model="commentText" :rows="4" :maxlength="500" placeholder="写下你的观点，支持 Markdown 文本。" />

          <div class="flex items-center justify-between text-xs text-toned">
            <p>
              {{ isLoggedIn ? '已登录，可直接发表评论' : '登录后可发表评论' }}
            </p>
            <p>{{ commentText.length }}/500</p>
          </div>

          <div class="flex justify-end">
            <UButton :loading="submittingComment" label="发表评论" icon="i-lucide-send" @click="submitComment" />
          </div>
        </div>

        <div class="mt-8 space-y-4">
          <article v-for="item in comments" :key="item.id" class="rounded-xl border border-default p-4">
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="text-sm font-medium text-highlighted">
                  {{ item.author?.name || `用户 ${item.userId}` }}
                </p>
                <p class="mt-1 text-xs text-toned">
                  {{ formatCommentTime(item.createdAt) }}
                </p>
              </div>

              <UButton v-if="canDeleteComment(item)" :loading="deletingCommentId === item.id" size="xs" color="error"
                variant="ghost" icon="i-lucide-trash-2" @click="removeComment(item.id)" />
            </div>

            <p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-toned">
              {{ item.content }}
            </p>

            <div v-if="item.replies?.length" class="mt-4 space-y-3 border-l border-default pl-4">
              <article v-for="reply in item.replies" :key="reply.id" class="rounded-lg bg-elevated/40 p-3">
                <p class="text-xs font-medium text-highlighted">
                  {{ reply.author?.name || `用户 ${reply.userId}` }}
                </p>
                <p class="mt-1 text-xs text-toned">
                  {{ formatCommentTime(reply.createdAt) }}
                </p>
                <p class="mt-2 whitespace-pre-wrap text-sm text-toned">
                  {{ reply.content }}
                </p>
              </article>
            </div>
          </article>

          <UAlert v-if="!commentsLoading && comments.length === 0" title="暂无评论" description="成为第一个发表评论的人。"
            icon="i-lucide-message-circle" color="neutral" variant="soft" />
        </div>
      </section>
    </main>
  </div>
</template>
