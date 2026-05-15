<script setup lang="ts">
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { POST_STATUS_PUBLISHED, createComment, deleteComment, fetchComments, fetchPostBySlug, fetchPostLikeState, fetchPosts, recordPostView, togglePostLike } from '~/composables/useBlogApi'
import type { CommentInfo } from '~/composables/useBlogApi'

const toast = useToast()
const { initAuth, isLoggedIn, session, user, fetchProfile } = useAuth()
const { buildUrl, siteName } = useSiteSeo()
const { categoryPath, tagPath } = useBlogRoutes()
const articleContentRef = ref<HTMLElement | null>(null)
const activeHeadingId = ref('')
let cleanupArticleProgress: (() => void) | undefined

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
const visibleViewCount = ref(post.value?.viewCount || 0)
const visibleLikeCount = ref(post.value?.likeCount || 0)
const likedByCurrentUser = ref(false)
const likeLoading = ref(false)
const submittingComment = ref(false)
const deletingCommentId = ref('')
const commentText = ref('')
const replyText = ref('')
const replyTarget = ref<CommentInfo | null>(null)
const submittingReplyId = ref('')

type HeadingItem = {
  id: string
  text: string
  depth: number
}

function normalizeHeadingText(value: string) {
  return value
    .replace(/[#>*`_~[\]()!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function toAnchorId(value: string, index: number) {
  const normalized = normalizeHeadingText(value)
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')

  return normalized ? `heading-${normalized}` : `heading-${index + 1}`
}

const headingItems = computed(() => {
  const source = post.value?.content || ''
  const seen = new Map<string, number>()
  const headings: HeadingItem[] = []
  let inCodeFence = false

  for (const line of source.split('\n')) {
    const trimmed = line.trim()
    if (/^```/.test(trimmed) || /^~~~/.test(trimmed)) {
      inCodeFence = !inCodeFence
      continue
    }

    if (inCodeFence) continue

    const match = /^(#{2,3})\s+(.+?)\s*#*$/.exec(trimmed)
    if (!match) continue

    const text = normalizeHeadingText(match[2] || '')
    if (!text) continue

    const baseId = toAnchorId(text, seen.size)
    const count = seen.get(baseId) || 0
    seen.set(baseId, count + 1)

    headings.push({
      id: count > 0 ? `${baseId}-${count + 1}` : baseId,
      text,
      depth: match[1]?.length || 2
    })
  }

  return headings
})

async function copyText(text: string, successTitle: string) {
  if (!import.meta.client) return

  try {
    await navigator.clipboard.writeText(text)
    toast.add({
      color: 'success',
      title: successTitle
    })
  } catch {
    toast.add({
      color: 'error',
      title: '复制失败',
      description: '浏览器暂时不允许写入剪贴板'
    })
  }
}

async function copyArticleLink() {
  await copyText(canonicalUrl.value, '链接已复制')
}

async function shareArticle() {
  if (!import.meta.client || !post.value) return

  if (navigator.share) {
    try {
      await navigator.share({
        title: post.value.title,
        text: post.value.summary,
        url: canonicalUrl.value
      })
      return
    } catch (error: unknown) {
      const typed = error as { name?: string }
      if (typed.name === 'AbortError') return
    }
  }

  await copyArticleLink()
}

function enhanceArticleContent() {
  if (!import.meta.client) return

  const root = articleContentRef.value
  if (!root) return

  const headings = Array.from(root.querySelectorAll<HTMLElement>('h2, h3'))
  headings.forEach((heading, headingIndex) => {
    const item = headingItems.value[headingIndex]
    if (!item) return

    heading.id = item.id
    heading.classList.add('scroll-mt-24')
  })

  const blocks = Array.from(root.querySelectorAll<HTMLElement>('pre'))
  for (const block of blocks) {
    if (block.dataset.copyEnhanced === 'true') continue
    block.dataset.copyEnhanced = 'true'
    block.classList.add('article-code-block')

    const frame = document.createElement('div')
    frame.className = 'article-code-frame'
    block.parentNode?.insertBefore(frame, block)
    frame.appendChild(block)

    const button = document.createElement('button')
    button.type = 'button'
    button.textContent = '复制'
    button.className = 'article-code-copy'
    button.addEventListener('click', async () => {
      const code = block.querySelector('code')?.textContent || ''
      await copyText(code.trim(), '代码已复制')
    })

    frame.appendChild(button)
  }
}

function setupArticleProgress() {
  if (!import.meta.client) return

  cleanupArticleProgress?.()

  const root = articleContentRef.value
  if (!root || headingItems.value.length === 0) {
    activeHeadingId.value = ''
    cleanupArticleProgress = undefined
    return
  }

  const headingNodes = headingItems.value
    .map(item => document.getElementById(item.id))
    .filter((item): item is HTMLElement => Boolean(item))

  if (headingNodes.length === 0) {
    cleanupArticleProgress = undefined
    return
  }

  let frame = 0
  const updateActiveHeading = () => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(() => {
      const anchorOffset = 128
      const current = headingNodes
        .filter(heading => heading.getBoundingClientRect().top <= anchorOffset)
        .at(-1)

      activeHeadingId.value = current?.id || headingNodes[0]?.id || ''
    })
  }

  const observer = new IntersectionObserver(updateActiveHeading, {
    rootMargin: '-96px 0px -65% 0px',
    threshold: [0, 1]
  })

  headingNodes.forEach(heading => observer.observe(heading))
  window.addEventListener('scroll', updateActiveHeading, { passive: true })
  window.addEventListener('resize', updateActiveHeading)
  updateActiveHeading()

  cleanupArticleProgress = () => {
    cancelAnimationFrame(frame)
    observer.disconnect()
    window.removeEventListener('scroll', updateActiveHeading)
    window.removeEventListener('resize', updateActiveHeading)
  }
}

async function refreshArticleEnhancements() {
  await nextTick()
  if (!import.meta.client) return

  requestAnimationFrame(() => {
    enhanceArticleContent()
    setupArticleProgress()
  })
}

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

function commentAuthorName(item?: CommentInfo | null) {
  if (!item) return '用户'
  return item.author?.name || `用户 ${item.userId}`
}

function commentInitial(item?: CommentInfo | null) {
  const name = commentAuthorName(item)
  return name.trim().slice(0, 1).toUpperCase() || 'U'
}

function getErrorMessage(error: unknown, fallback: string) {
  if (!error || typeof error !== 'object') return fallback
  const typed = error as { message?: unknown, data?: { message?: unknown } }
  if (typeof typed.data?.message === 'string') return typed.data.message
  if (typeof typed.message === 'string') return typed.message
  return fallback
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

const viewedPostIds = new Set<string>()

async function recordCurrentPostView() {
  if (!import.meta.client) return

  const current = post.value
  if (!current?.id || viewedPostIds.has(current.id)) return

  viewedPostIds.add(current.id)
  visibleViewCount.value = current.viewCount || visibleViewCount.value

  try {
    const result = await recordPostView(current.id)
    if (Number.isFinite(result.viewCount) && result.viewCount !== undefined) {
      visibleViewCount.value = result.viewCount
    }
  } catch {
    // 阅读统计失败不阻断文章浏览。
  }
}

async function loadLikeState() {
  const current = post.value
  visibleLikeCount.value = current?.likeCount || 0
  likedByCurrentUser.value = false

  if (!current?.id || !isLoggedIn.value) return

  try {
    const result = await fetchPostLikeState(current.id)
    likedByCurrentUser.value = result.liked
    visibleLikeCount.value = result.likeCount
  } catch {
    likedByCurrentUser.value = false
  }
}

async function handleToggleLike() {
  const current = post.value
  if (!current?.id) return

  if (!isLoggedIn.value) {
    toast.add({ title: '请先登录后再点赞', color: 'warning' })
    await navigateTo(`/login?redirect=/posts/${slug.value}`)
    return
  }

  if (likeLoading.value) return

  likeLoading.value = true
  try {
    const result = await togglePostLike(current.id)
    likedByCurrentUser.value = result.liked
    visibleLikeCount.value = result.likeCount
    toast.add({
      title: result.liked ? '已点赞' : '已取消点赞',
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: '操作失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    likeLoading.value = false
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
  } catch (error: unknown) {
    toast.add({
      title: '评论发布失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    submittingComment.value = false
  }
}

function startReply(item: CommentInfo) {
  if (!isLoggedIn.value) {
    toast.add({ title: '请先登录后再回复', color: 'warning' })
    navigateTo(`/login?redirect=/posts/${slug.value}`)
    return
  }

  replyTarget.value = item
  replyText.value = ''
}

function cancelReply() {
  replyTarget.value = null
  replyText.value = ''
}

async function submitReply() {
  const target = replyTarget.value
  const content = replyText.value.trim()

  if (!target) return
  if (!content) {
    toast.add({ title: '回复内容不能为空', color: 'warning' })
    return
  }

  if (!post.value?.id) {
    toast.add({ title: '文章信息不存在，无法回复', color: 'error' })
    return
  }

  if (!isLoggedIn.value) {
    toast.add({ title: '请先登录后再回复', color: 'warning' })
    await navigateTo(`/login?redirect=/posts/${slug.value}`)
    return
  }

  submittingReplyId.value = target.id
  try {
    await createComment({
      postId: post.value.id,
      content,
      parentId: target.id
    })

    cancelReply()
    toast.add({ title: '回复发布成功', color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '回复发布失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    submittingReplyId.value = ''
  }
}

async function removeComment(id: string) {
  if (!id) return

  deletingCommentId.value = id
  try {
    await deleteComment(id)
    toast.add({ title: '评论已删除', color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '删除评论失败',
      description: getErrorMessage(error, '请稍后重试'),
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
  await loadLikeState()
  await recordCurrentPostView()
  await refreshArticleEnhancements()
})

watch(post, async (value) => {
  if (!value?.id) return
  visibleViewCount.value = value.viewCount || 0
  visibleLikeCount.value = value.likeCount || 0
  await loadComments()
  await loadLikeState()
  await recordCurrentPostView()
  await refreshArticleEnhancements()
})

onBeforeUnmount(() => {
  cleanupArticleProgress?.()
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

const displayedCommentCount = computed(() =>
  comments.value.reduce((total, item) => total + 1 + (item.replies?.length || 0), 0)
)

const ViewerCodeBlockShiki = CodeBlockShiki.extend({
  markdownTokenName: 'code',
  parseMarkdown: (token, helpers) => {
    const typedToken = token as {
      raw?: string
      codeBlockStyle?: string
      lang?: string
      text?: string
    }
    if (
      typedToken.raw?.startsWith('```') === false
      && typedToken.raw?.startsWith('~~~') === false
      && typedToken.codeBlockStyle !== 'indented'
    ) {
      return []
    }

    return helpers.createNode(
      'codeBlock',
      { language: typedToken.lang || null },
      typedToken.text ? [helpers.createTextNode(typedToken.text)] : []
    )
  },
  renderMarkdown: (node, helpers) => {
    const typedNode = node as { attrs?: { language?: string }, content?: unknown }
    const language = typedNode.attrs?.language || ''
    if (!typedNode.content) return `\`\`\`${language}\n\n\`\`\``

    return [`\`\`\`${language}`, helpers.renderChildren(typedNode.content), '```'].join('\n')
  }
})

const viewerExtensions = [
  ViewerCodeBlockShiki.configure({
    defaultTheme: 'material-theme',
    themes: {
      light: 'material-theme-lighter',
      dark: 'material-theme-palenight'
    }
  }),
  TableKit,
  TaskList,
  TaskItem
]

const seoTitle = computed(() => post.value?.title || '文章详情')
const seoDescription = computed(() => post.value?.summary || '')
const seoImage = computed(() => post.value?.cover || '')
const canonicalUrl = computed(() => buildUrl(`/posts/${encodeURIComponent(slug.value)}`))
const articleJsonLd = computed(() => {
  if (!post.value) return ''

  return JSON.stringify({
    '@context': 'https://schema.org',
    '@type': 'BlogPosting',
    'headline': post.value.title,
    'description': post.value.summary,
    'image': post.value.cover,
    'datePublished': post.value.createdAt || undefined,
    'dateModified': post.value.createdAt || undefined,
    'author': {
      '@type': 'Person',
      'name': post.value.author,
      'url': post.value.authorId ? buildUrl(`/authors/${post.value.authorId}`) : undefined
    },
    'mainEntityOfPage': canonicalUrl.value,
    'publisher': {
      '@type': 'Organization',
      'name': siteName.value
    }
  })
})

useSeoMeta({
  title: seoTitle,
  description: seoDescription,
  ogTitle: seoTitle,
  ogDescription: seoDescription,
  ogImage: seoImage,
  twitterCard: 'summary_large_image'
})

useHead({
  link: [
    {
      rel: 'canonical',
      href: canonicalUrl
    }
  ],
  script: articleJsonLd.value
    ? [
        {
          key: 'article-jsonld',
          type: 'application/ld+json',
          innerHTML: articleJsonLd.value
        }
      ]
    : []
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader>
      <UButton
        to="/write"
        icon="i-lucide-square-pen"
        size="sm"
        class="sm:hidden"
      />
      <UButton
        to="/write"
        icon="i-lucide-square-pen"
        label="继续写作"
        size="sm"
        class="hidden sm:inline-flex"
      />
    </AppHeader>

    <main
      v-if="post"
      class="mx-auto w-full max-w-7xl px-4 pb-20 pt-10 sm:px-8 xl:px-12"
    >
      <div class="grid gap-8 lg:grid-cols-[minmax(0,900px)_minmax(220px,1fr)]">
        <article class="motion-fade-up min-w-0 rounded-2xl border border-default bg-default p-6 sm:p-10">
          <div class="space-y-5 border-b border-default pb-8">
            <div class="flex flex-wrap items-center gap-2 text-xs text-toned">
              <NuxtLink
                :to="categoryPath(post.category)"
                class="inline-flex"
              >
                <UBadge
                  :label="post.category"
                  color="primary"
                  variant="subtle"
                  class="hover:opacity-90"
                />
              </NuxtLink>
              <span>{{ post.publishedAt }}</span>
              <span>·</span>
              <span>{{ post.readingMinutes }} 分钟</span>
              <span>·</span>
              <span>{{ visibleViewCount }} 阅读</span>
              <span>·</span>
              <span>{{ visibleLikeCount }} 点赞</span>
              <span>·</span>
              <NuxtLink
                v-if="post.authorId"
                :to="`/authors/${post.authorId}`"
                class="hover:text-primary"
              >
                {{ post.author }}
              </NuxtLink>
              <span v-else>{{ post.author }}</span>
            </div>

            <h1 class="text-3xl font-semibold tracking-tight text-highlighted sm:text-4xl">
              {{ post.title }}
            </h1>

            <p class="text-base text-toned sm:text-lg">
              {{ post.summary }}
            </p>

            <div class="flex flex-wrap gap-2">
              <UButton
                :icon="likedByCurrentUser ? 'i-lucide-heart' : 'i-lucide-heart'"
                size="sm"
                :color="likedByCurrentUser ? 'primary' : 'neutral'"
                :variant="likedByCurrentUser ? 'solid' : 'soft'"
                :loading="likeLoading"
                :label="likedByCurrentUser ? `已点赞 ${visibleLikeCount}` : `点赞 ${visibleLikeCount}`"
                @click="handleToggleLike"
              />
              <UButton
                icon="i-lucide-link"
                size="sm"
                color="neutral"
                variant="soft"
                label="复制链接"
                @click="copyArticleLink"
              />
              <UButton
                icon="i-lucide-share-2"
                size="sm"
                color="neutral"
                variant="ghost"
                label="分享"
                @click="shareArticle"
              />
            </div>

            <img
              :src="post.cover"
              :alt="post.title"
              class="h-60 w-full rounded-xl object-cover"
            >
          </div>

          <div class="mt-8">
            <p
              v-if="!post.content?.trim()"
              class="leading-7 text-toned"
            >
              暂无正文内容。
            </p>

            <div
              v-else
              ref="articleContentRef"
              class="article-viewer min-w-0"
            >
              <ClientOnly>
                <UEditor
                  :key="post.slug"
                  :model-value="post.content"
                  content-type="markdown"
                  :editable="false"
                  :extensions="viewerExtensions"
                  :starter-kit="{ codeBlock: false }"
                  class="min-h-0"
                  :ui="{
                    base: 'p-0',
                    content: 'max-w-none'
                  }"
                  @create="refreshArticleEnhancements"
                />

                <template #fallback>
                  <div class="space-y-3">
                    <div class="loading-shimmer h-4 w-11/12 rounded" />
                    <div class="loading-shimmer h-4 w-9/12 rounded" />
                    <div class="loading-shimmer h-24 w-full rounded-xl" />
                  </div>
                </template>
              </ClientOnly>
            </div>
          </div>

          <div class="mt-10 flex flex-wrap gap-2 border-t border-default pt-6">
            <NuxtLink
              v-for="tag in post.tags"
              :key="tag"
              :to="tagPath(tag)"
              class="inline-flex"
            >
              <UBadge
                :label="`#${tag}`"
                color="neutral"
                variant="outline"
                class="hover:border-primary hover:text-primary"
              />
            </NuxtLink>
          </div>
        </article>

        <aside
          v-if="headingItems.length"
          class="hidden lg:block"
        >
          <div class="sticky top-24 rounded-2xl border border-default bg-default/70 p-5">
            <p class="text-xs font-medium uppercase tracking-wide text-toned">
              目录
            </p>
            <nav class="mt-4 space-y-1">
              <a
                v-for="item in headingItems"
                :key="item.id"
                :href="`#${item.id}`"
                class="block rounded-lg px-3 py-2 text-sm transition-colors hover:bg-elevated hover:text-highlighted"
                :class="[
                  item.depth === 3 ? 'ml-3 text-xs' : '',
                  activeHeadingId === item.id ? 'bg-primary/10 text-primary' : 'text-toned'
                ]"
              >
                {{ item.text }}
              </a>
            </nav>
          </div>
        </aside>
      </div>

      <section class="motion-fade-up motion-delay-1 mt-10">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-highlighted">
            相关文章
          </h2>
          <UButton
            to="/posts"
            size="xs"
            color="neutral"
            variant="ghost"
            trailing-icon="i-lucide-arrow-right"
            label="返回列表"
          />
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <NuxtLink
            v-for="item in relatedPosts"
            :key="item.id"
            :to="`/posts/${item.slug}`"
            class="motion-card motion-panel rounded-xl border border-default bg-default p-4 hover:border-primary/40"
          >
            <p class="text-xs text-toned">
              {{ item.publishedAt }}
            </p>
            <p class="mt-2 line-clamp-2 text-sm font-semibold text-highlighted">
              {{ item.title }}
            </p>
          </NuxtLink>
        </div>
      </section>

      <section class="motion-fade-up motion-delay-2 mt-10 max-w-[900px] rounded-2xl border border-default bg-default p-6 sm:p-8">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-xl font-semibold text-highlighted">
              评论（{{ displayedCommentCount || commentsTotal || comments.length }}）
            </h2>
            <p class="mt-1 text-sm text-toned">
              写下你的想法，也可以直接回复某一条评论。
            </p>
          </div>
          <UBadge
            v-if="commentsLoading"
            label="加载中"
            color="neutral"
            variant="soft"
          />
        </div>

        <div class="mt-6 rounded-xl border border-default bg-elevated/30 p-4">
          <UTextarea
            v-model="commentText"
            :rows="4"
            :maxlength="500"
            placeholder="写下你的观点，支持 Markdown 文本。"
            class="w-full"
          />

          <div class="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs text-toned">
            <p>
              {{ isLoggedIn ? '已登录，可直接发表评论' : '登录后可发表评论' }}
            </p>

            <div class="flex items-center gap-3">
              <span>{{ commentText.length }}/500</span>
              <UButton
                :loading="submittingComment"
                label="发表评论"
                icon="i-lucide-send"
                @click="submitComment"
              />
            </div>
          </div>
        </div>

        <div class="mt-8 space-y-5">
          <article
            v-for="item in comments"
            :id="`comment-${item.id}`"
            :key="item.id"
            class="rounded-xl border border-default bg-default p-4 sm:p-5"
          >
            <div class="flex gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                {{ commentInitial(item) }}
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-highlighted">
                      {{ commentAuthorName(item) }}
                    </p>
                    <p class="mt-1 text-xs text-toned">
                      {{ formatCommentTime(item.createdAt) }}
                    </p>
                  </div>

                  <div class="flex items-center gap-1">
                    <UButton
                      size="xs"
                      color="neutral"
                      variant="ghost"
                      icon="i-lucide-message-circle"
                      label="回复"
                      @click="startReply(item)"
                    />
                    <UButton
                      v-if="canDeleteComment(item)"
                      :loading="deletingCommentId === item.id"
                      size="xs"
                      color="error"
                      variant="ghost"
                      icon="i-lucide-trash-2"
                      @click="removeComment(item.id)"
                    />
                  </div>
                </div>

                <p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-toned">
                  {{ item.content }}
                </p>

                <div
                  v-if="replyTarget?.id === item.id"
                  class="mt-4 rounded-xl border border-default bg-elevated/40 p-3"
                >
                  <p class="mb-2 text-xs text-toned">
                    回复 {{ commentAuthorName(replyTarget) }}
                  </p>
                  <UTextarea
                    v-model="replyText"
                    :rows="3"
                    :maxlength="500"
                    placeholder="写下回复内容。"
                    class="w-full"
                  />
                  <div class="mt-3 flex items-center justify-between gap-3 text-xs text-toned">
                    <span>{{ replyText.length }}/500</span>
                    <div class="flex gap-2">
                      <UButton
                        size="xs"
                        color="neutral"
                        variant="ghost"
                        label="取消"
                        @click="cancelReply"
                      />
                      <UButton
                        size="xs"
                        :loading="submittingReplyId === item.id"
                        label="发布回复"
                        icon="i-lucide-send"
                        @click="submitReply"
                      />
                    </div>
                  </div>
                </div>

                <div
                  v-if="item.replies?.length"
                  class="mt-5 space-y-3 border-l border-default pl-4"
                >
                  <article
                    v-for="reply in item.replies"
                    :id="`comment-${reply.id}`"
                    :key="reply.id"
                    class="rounded-xl bg-elevated/40 p-4"
                  >
                    <div class="flex gap-3">
                      <div class="flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-toned">
                        {{ commentInitial(reply) }}
                      </div>

                      <div class="min-w-0 flex-1">
                        <div class="flex flex-wrap items-start justify-between gap-3">
                          <div>
                            <p class="text-sm font-semibold text-highlighted">
                              {{ commentAuthorName(reply) }}
                              <span
                                v-if="reply.replyToAuthor?.name"
                                class="font-normal text-toned"
                              >
                                回复 @{{ reply.replyToAuthor.name }}
                              </span>
                            </p>
                            <p class="mt-1 text-xs text-toned">
                              {{ formatCommentTime(reply.createdAt) }}
                            </p>
                          </div>

                          <div class="flex items-center gap-1">
                            <UButton
                              size="xs"
                              color="neutral"
                              variant="ghost"
                              icon="i-lucide-message-circle"
                              label="回复"
                              @click="startReply(reply)"
                            />
                            <UButton
                              v-if="canDeleteComment(reply)"
                              :loading="deletingCommentId === reply.id"
                              size="xs"
                              color="error"
                              variant="ghost"
                              icon="i-lucide-trash-2"
                              @click="removeComment(reply.id)"
                            />
                          </div>
                        </div>

                        <p class="mt-3 whitespace-pre-wrap text-sm leading-6 text-toned">
                          {{ reply.content }}
                        </p>

                        <div
                          v-if="replyTarget?.id === reply.id"
                          class="mt-4 rounded-xl border border-default bg-default p-3"
                        >
                          <p class="mb-2 text-xs text-toned">
                            回复 {{ commentAuthorName(replyTarget) }}
                          </p>
                          <UTextarea
                            v-model="replyText"
                            :rows="3"
                            :maxlength="500"
                            placeholder="写下回复内容。"
                            class="w-full"
                          />
                          <div class="mt-3 flex items-center justify-between gap-3 text-xs text-toned">
                            <span>{{ replyText.length }}/500</span>
                            <div class="flex gap-2">
                              <UButton
                                size="xs"
                                color="neutral"
                                variant="ghost"
                                label="取消"
                                @click="cancelReply"
                              />
                              <UButton
                                size="xs"
                                :loading="submittingReplyId === reply.id"
                                label="发布回复"
                                icon="i-lucide-send"
                                @click="submitReply"
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </article>

          <UAlert
            v-if="!commentsLoading && comments.length === 0"
            title="暂无评论"
            description="成为第一个发表评论的人。"
            icon="i-lucide-message-circle"
            color="neutral"
            variant="soft"
          />
        </div>
      </section>
    </main>
  </div>
</template>
