<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import { createHighlighter } from 'shiki'
import { ArrowLeft, MessageSquare, Heart, Share2, Send, Reply, Trash2, ChevronUp, ListTree } from 'lucide-vue-next'
import type { Article, Comment } from '~/composables/useBlogData'
import { userAvatarUrl, formatDate } from '~/composables/useBlogData'
import type { ApiResponse } from '~/composables/useBlogData'

const props = defineProps<{
  post: Article
}>()

const router = useRouter()
const { isLoggedIn, authFetch, user } = useAuth()
const { askConfirm } = useConfirmDialog()
const isLiked = ref(false)
const postBodyRef = ref<HTMLElement | null>(null)
const readingProgress = ref(0)
const showBackTop = ref(false)

interface TocItem {
  id: string
  text: string
  level: 2 | 3 | 4
}

const tocItems = ref<TocItem[]>([])
const activeHeadingId = ref('')

const highlighter = await createHighlighter({
  themes: ['github-light'],
  langs: ['text', 'bash', 'javascript', 'typescript', 'json', 'go', 'yaml', 'markdown', 'html', 'css', 'sql']
})

function normalizeLang(lang: string) {
  const lower = lang.toLowerCase()
  if (lower === 'js') return 'javascript'
  if (lower === 'ts') return 'typescript'
  if (lower === 'sh' || lower === 'shell') return 'bash'
  if (lower === 'yml') return 'yaml'
  return lower
}

function decodeHtmlEntities(content: string) {
  return content
    .replace(/&#34;|&quot;/g, '"')
    .replace(/&#39;|&apos;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
}

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: true,
  highlight(code, lang) {
    const normalized = normalizeLang(lang || 'text')
    const language = highlighter.getLoadedLanguages().includes(normalized as never) ? normalized : 'text'
    return highlighter.codeToHtml(code, {
      lang: language,
      theme: 'github-light'
    })
  }
})

function looksLikeHtml(content: string) {
  return /<([a-z][\w0-9-]*)(\s[^>]*)?>/i.test(content)
}

const renderedContent = computed(() => {
  const raw = props.post.content ?? ''
  if (!raw.trim()) return ''
  if (looksLikeHtml(raw)) return raw
  return markdown.render(decodeHtmlEntities(raw))
})

function slugify(text: string) {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}

function updateHeadingState() {
  if (!import.meta.client || !postBodyRef.value) return
  const headings = Array.from(postBodyRef.value.querySelectorAll('h2, h3, h4')) as HTMLElement[]
  if (!headings.length) {
    activeHeadingId.value = ''
    return
  }

  const threshold = window.scrollY + 140
  let current: HTMLElement | null = headings[0] ?? null
  for (const heading of headings) {
    const top = heading.getBoundingClientRect().top + window.scrollY
    if (top <= threshold) current = heading
    else break
  }
  activeHeadingId.value = current?.id ?? ''
}

function updateReadingState() {
  if (!import.meta.client) return
  const doc = document.documentElement
  const scrollTop = window.scrollY || doc.scrollTop
  const max = doc.scrollHeight - window.innerHeight
  readingProgress.value = max > 0 ? Math.min(100, Math.max(0, (scrollTop / max) * 100)) : 0
  showBackTop.value = scrollTop > 500
  updateHeadingState()
}

function buildTOCAndDecorateHeadings() {
  if (!postBodyRef.value) return

  const headings = Array.from(postBodyRef.value.querySelectorAll('h2, h3, h4')) as HTMLElement[]
  const slugCount: Record<string, number> = {}
  tocItems.value = headings.map((heading) => {
    const text = heading.textContent?.trim() || 'Section'
    const base = slugify(text) || 'section'
    const count = slugCount[base] ?? 0
    slugCount[base] = count + 1
    const id = count === 0 ? base : `${base}-${count}`
    heading.id = id

    if (!heading.querySelector('.article-anchor')) {
      const anchor = document.createElement('a')
      anchor.className = 'article-anchor'
      anchor.href = `#${id}`
      anchor.textContent = '#'
      anchor.setAttribute('aria-label', `跳转到 ${text}`)
      heading.appendChild(anchor)
    }

    return {
      id,
      text,
      level: Number(heading.tagName.slice(1)) as 2 | 3 | 4
    }
  })
}

function decorateCodeBlocks() {
  if (!postBodyRef.value) return
  const blocks = Array.from(postBodyRef.value.querySelectorAll('pre.shiki')) as HTMLElement[]

  for (const pre of blocks) {
    if (pre.parentElement?.classList.contains('article-code-wrap')) continue

    const wrapper = document.createElement('div')
    wrapper.className = 'article-code-wrap'
    pre.parentNode?.insertBefore(wrapper, pre)
    wrapper.appendChild(pre)

    const code = pre.querySelector('code')
    const codeText = code?.textContent?.replace(/\n$/, '') ?? ''
    const lines = Math.max(1, codeText.split('\n').length)

    const gutter = document.createElement('div')
    gutter.className = 'article-code-gutter'
    const nums = Array.from({ length: lines }, (_, i) => `<span>${i + 1}</span>`).join('')
    gutter.innerHTML = nums
    wrapper.insertBefore(gutter, pre)

    const copyBtn = document.createElement('button')
    copyBtn.type = 'button'
    copyBtn.className = 'article-copy-btn'
    copyBtn.textContent = '复制'
    wrapper.appendChild(copyBtn)
  }
}

async function onBodyClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  const button = target.closest('.article-copy-btn') as HTMLButtonElement | null
  if (!button) return
  const wrapper = button.closest('.article-code-wrap')
  const code = wrapper?.querySelector('pre code')
  const raw = code?.textContent ?? ''
  if (!raw || !import.meta.client) return

  try {
    await navigator.clipboard.writeText(raw)
    button.textContent = '已复制'
    setTimeout(() => {
      button.textContent = '复制'
    }, 1200)
  } catch {
    button.textContent = '复制失败'
    setTimeout(() => {
      button.textContent = '复制'
    }, 1200)
  }
}

function enhanceArticleBody() {
  if (!postBodyRef.value) return
  buildTOCAndDecorateHeadings()
  decorateCodeBlocks()
  updateHeadingState()
}

function goTop() {
  if (!import.meta.client) return
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function scrollToHeading(id: string) {
  if (!import.meta.client) return
  const el = document.getElementById(id)
  if (!el) return
  const top = el.getBoundingClientRect().top + window.scrollY - 90
  window.scrollTo({ top, behavior: 'smooth' })
}

watch(renderedContent, async () => {
  await nextTick()
  enhanceArticleBody()
})

onMounted(async () => {
  await nextTick()
  enhanceArticleBody()
  if (import.meta.client) {
    window.addEventListener('scroll', updateReadingState, { passive: true })
    postBodyRef.value?.addEventListener('click', onBodyClick)
    updateReadingState()
  }
})

onBeforeUnmount(() => {
  if (import.meta.client) {
    window.removeEventListener('scroll', updateReadingState)
    postBodyRef.value?.removeEventListener('click', onBodyClick)
  }
})

// 获取评论列表
const { data: commentsData, refresh: refreshComments } = await useComments(props.post.id)

// 将平铺评论按 parent_id 分组：顶级 + replies map
const topLevelComments = computed<Comment[]>(() =>
  (commentsData.value?.data ?? []).filter(c => !c.parent_id || c.parent_id === 0)
)
const repliesMap = computed<Record<number, Comment[]>>(() => {
  const map: Record<number, Comment[]> = {}
  for (const c of commentsData.value?.data ?? []) {
    if (c.parent_id && c.parent_id !== 0) {
      if (!map[c.parent_id]) map[c.parent_id] = []
      ;(map[c.parent_id] as Comment[]).push(c)
    }
  }
  return map
})

// 发顶级评论
const commentText = ref('')
const submitting  = ref(false)
const commentError = ref('')

async function submitComment() {
  if (!commentText.value.trim()) return
  submitting.value = true
  commentError.value = ''
  try {
    await authFetch<ApiResponse<Comment>>(`/articles/${props.post.id}/comments`, {
      method: 'POST',
      body: { content: commentText.value.trim(), parent_id: 0 }
    })
    commentText.value = ''
    await refreshComments()
  } catch (err: unknown) {
    commentError.value = (err as Error)?.message ?? '评论失败，请重试'
  } finally {
    submitting.value = false
  }
}

// 回复评论（replyingTo：当前展开回复框的评论 id，null = 无）
const replyingTo   = ref<number | null>(null)
const replyTexts   = reactive<Record<number, string>>({})
const replySubmitting = reactive<Record<number, boolean>>({})

function toggleReply(commentId: number) {
  replyingTo.value = replyingTo.value === commentId ? null : commentId
  if (!replyTexts[commentId]) replyTexts[commentId] = ''
}

async function submitReply(parentComment: Comment) {
  const text = (replyTexts[parentComment.id] ?? '').trim()
  if (!text) return

  replySubmitting[parentComment.id] = true
  try {
    await authFetch<ApiResponse<Comment>>(`/articles/${props.post.id}/comments`, {
      method: 'POST',
      body: { content: text, parent_id: parentComment.id }
    })
    replyTexts[parentComment.id] = ''
    replyingTo.value = null
    await refreshComments()
  } catch {
    // 静默失败，可根据需要添加提示
  } finally {
    replySubmitting[parentComment.id] = false
  }
}

// 删除评论
const deletingIds = reactive<Set<number>>(new Set())

async function deleteComment(commentId: number) {
  const ok = await askConfirm({
    title: '删除评论',
    message: '确认删除这条评论吗？该操作不可撤销。',
    confirmText: '删除',
    tone: 'danger'
  })
  if (!ok) return
  deletingIds.add(commentId)
  try {
    await authFetch(`/comments/${commentId}`, { method: 'DELETE' })
    await refreshComments()
  } catch {
    // 静默失败
  } finally {
    deletingIds.delete(commentId)
  }
}

// 判断当前用户是否可以删除该评论
function canDelete(comment: Comment) {
  return isLoggedIn.value && (
    (comment.user_id !== null && user.value?.id === comment.user_id) ||
    user.value?.role === 'admin'
  )
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/')
}
</script>

<template>
  <article
    v-motion
    :initial="{ opacity: 0, y: 20 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500, ease: [0.22, 1, 0.36, 1] } }"
    class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <div class="fixed top-0 left-0 right-0 h-1 bg-m3-sys-light-surface-variant/70 z-50">
      <div class="h-full bg-m3-sys-light-primary transition-[width] duration-150" :style="{ width: `${readingProgress}%` }" />
    </div>

    <button
      @click="goBack"
      class="mb-8 flex items-center gap-2 px-4 py-2 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface-variant font-medium"
    >
      <ArrowLeft class="w-5 h-5" />
      Back
    </button>

    <div class="mb-12">
      <div class="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container text-sm font-bold uppercase tracking-wider mb-6">
        {{ post.category?.name ?? 'Article' }}
      </div>
      <h1 class="text-4xl sm:text-5xl md:text-6xl font-black tracking-tighter leading-[1.1] text-m3-sys-light-on-surface mb-8">
        {{ post.title }}
      </h1>

      <div class="flex flex-wrap items-center justify-between gap-6 py-6 border-y border-m3-sys-light-surface-variant">
        <div class="flex items-center gap-4">
          <img
            :src="userAvatarUrl(post.author)"
            :alt="post.author?.username"
            class="w-14 h-14 rounded-full object-cover"
            referrerpolicy="no-referrer"
          />
          <div>
            <p class="font-bold text-lg text-m3-sys-light-on-surface">{{ post.author?.username }}</p>
            <p class="text-m3-sys-light-on-surface-variant">{{ formatDate(post.created_at) }} · {{ post.read_count }} views</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="isLiked = !isLiked"
            class="p-3 rounded-full transition-colors"
            :class="isLiked ? 'bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container' : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container'"
          >
            <Heart class="w-6 h-6" :class="isLiked ? 'fill-current' : ''" />
          </button>
          <button class="p-3 rounded-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors">
            <Share2 class="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="tocItems.length" class="lg:hidden mb-6 p-4 rounded-2xl border border-m3-sys-light-outline-variant bg-m3-sys-light-surface">
      <div class="flex items-center gap-2 text-sm font-semibold text-m3-sys-light-on-surface mb-3">
        <ListTree class="w-4 h-4" /> 目录
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="item in tocItems"
          :key="item.id"
          @click="scrollToHeading(item.id)"
          class="px-3 py-1.5 rounded-full text-xs transition-colors"
          :class="activeHeadingId === item.id
            ? 'bg-m3-sys-light-primary text-m3-sys-light-on-primary'
            : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container'"
        >
          {{ item.text }}
        </button>
      </div>
    </div>

    <div class="lg:grid lg:grid-cols-[minmax(0,1fr)_240px] gap-8 items-start">
      <div>
        <!-- 文章正文 -->
        <div
          ref="postBodyRef"
          class="article-rich-content max-w-none mb-20 text-m3-sys-light-on-surface leading-relaxed"
          v-html="renderedContent"
        />

        <!-- Tags -->
        <div v-if="post.tags?.length" class="flex flex-wrap gap-2 mb-12">
          <span
            v-for="tag in post.tags"
            :key="tag.id"
            class="px-3 py-1 rounded-full bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant text-sm"
          >
            #{{ tag.name }}
          </span>
        </div>
      </div>

      <aside v-if="tocItems.length" class="hidden lg:block sticky top-24">
        <div class="rounded-2xl border border-m3-sys-light-outline-variant bg-m3-sys-light-surface p-4">
          <div class="flex items-center gap-2 text-sm font-semibold text-m3-sys-light-on-surface mb-3">
            <ListTree class="w-4 h-4" /> 目录
          </div>
          <ul class="space-y-1.5">
            <li v-for="item in tocItems" :key="item.id">
              <button
                @click="scrollToHeading(item.id)"
                class="w-full text-left text-sm rounded-lg px-2 py-1.5 transition-colors"
                :class="[
                  item.level === 3 ? 'pl-4' : item.level === 4 ? 'pl-6' : '',
                  activeHeadingId === item.id
                    ? 'bg-m3-sys-light-primary-container text-m3-sys-light-on-primary-container'
                    : 'text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-surface-variant'
                ]"
              >
                {{ item.text }}
              </button>
            </li>
          </ul>
        </div>
      </aside>
    </div>

    <button
      v-if="showBackTop"
      @click="goTop"
      class="fixed right-5 bottom-6 z-40 w-11 h-11 rounded-full bg-m3-sys-light-primary text-m3-sys-light-on-primary shadow-lg hover:opacity-90 transition-opacity flex items-center justify-center"
      aria-label="回到顶部"
    >
      <ChevronUp class="w-5 h-5" />
    </button>

    <section class="bg-m3-sys-light-surface-variant/50 rounded-[3rem] p-8 sm:p-12">
      <div class="flex items-center gap-3 mb-10">
        <MessageSquare class="w-8 h-8 text-m3-sys-light-primary" />
        <h2 class="text-3xl font-bold">Comments ({{ (commentsData?.data ?? []).length }})</h2>
      </div>

      <!-- 发表顶级评论 -->
      <div class="flex gap-4 mb-12">
        <img
          :src="userAvatarUrl(user)"
          alt="You"
          class="w-12 h-12 rounded-full object-cover shrink-0"
          referrerpolicy="no-referrer"
        />
        <div class="flex-grow relative">
          <textarea
            v-model="commentText"
            placeholder="Add to the discussion…"
            :disabled="submitting"
            class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant rounded-[1.5rem] p-5 pr-16 resize-none focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all shadow-sm min-h-[100px] disabled:opacity-60"
          />
          <button
            @click="submitComment"
            :disabled="submitting || !commentText.trim()"
            class="absolute bottom-4 right-4 p-3 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full hover:opacity-90 transition-opacity shadow-md disabled:opacity-50"
          >
            <Send class="w-5 h-5" />
          </button>
        </div>
      </div>
      <p v-if="commentError" class="text-red-500 text-sm -mt-8 mb-6">{{ commentError }}</p>

      <!-- 未登录提示 -->
      <p v-if="!isLoggedIn" class="text-center text-m3-sys-light-on-surface-variant mb-8 text-sm">
        <NuxtLink to="/login" class="text-m3-sys-light-primary font-semibold hover:underline">Sign in</NuxtLink>
        to link your comment to your account, or post anonymously.
      </p>

      <!-- 评论列表（顶级 + 嵌套回复） -->
      <div class="space-y-6">
        <div v-if="topLevelComments.length === 0" class="text-center text-m3-sys-light-on-surface-variant italic py-8">
          No comments yet. Be the first to share your thoughts!
        </div>

        <div v-for="comment in topLevelComments" :key="comment.id">
          <!-- 顶级评论 -->
          <div class="flex gap-4">
            <img
              :src="userAvatarUrl(comment.user)"
              :alt="comment.user?.username"
              class="w-11 h-11 rounded-full object-cover shrink-0 mt-1"
              referrerpolicy="no-referrer"
            />
            <div class="flex-1">
              <div class="bg-m3-sys-light-surface p-5 rounded-[1.5rem] rounded-tl-none shadow-sm">
                <div class="flex items-center justify-between mb-1.5">
                  <span class="font-bold text-m3-sys-light-on-surface">{{ comment.user?.username || '匿名用户' }}</span>
                  <span class="text-xs text-m3-sys-light-on-surface-variant">{{ formatDate(comment.created_at) }}</span>
                </div>
                <p class="text-m3-sys-light-on-surface opacity-90 leading-relaxed">{{ comment.content }}</p>
              </div>

              <!-- 回复 & 删除操作栏 -->
              <div class="flex items-center gap-3 mt-1.5 ml-1">
                <button
                  v-if="isLoggedIn"
                  @click="toggleReply(comment.id)"
                  class="flex items-center gap-1 text-xs font-medium text-m3-sys-light-on-surface-variant hover:text-m3-sys-light-primary transition-colors"
                >
                  <Reply class="w-3.5 h-3.5" />
                  {{ replyingTo === comment.id ? 'Cancel' : 'Reply' }}
                </button>
                <button
                  v-if="canDelete(comment)"
                  :disabled="deletingIds.has(comment.id)"
                  @click="deleteComment(comment.id)"
                  class="flex items-center gap-1 text-xs font-medium text-m3-sys-light-on-surface-variant hover:text-red-500 transition-colors disabled:opacity-50"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                  Delete
                </button>
              </div>

              <!-- 回复输入框 -->
              <div v-if="replyingTo === comment.id" class="mt-3 flex gap-3">
                <img
                  :src="userAvatarUrl(user)"
                  alt="You"
                  class="w-8 h-8 rounded-full object-cover shrink-0 mt-1"
                  referrerpolicy="no-referrer"
                />
                <div class="flex-1 relative">
                  <textarea
                    v-model="replyTexts[comment.id]"
                    :placeholder="`Reply to ${comment.user?.username ?? 'user'}…`"
                    :disabled="replySubmitting[comment.id]"
                    rows="2"
                    class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant rounded-2xl p-4 pr-14 resize-none focus:outline-none focus:ring-2 focus:ring-m3-sys-light-primary/30 transition-all text-sm disabled:opacity-60"
                  />
                  <button
                    @click="submitReply(comment)"
                    :disabled="replySubmitting[comment.id] || !(replyTexts[comment.id] ?? '').trim()"
                    class="absolute bottom-3 right-3 p-2 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full hover:opacity-90 transition-opacity disabled:opacity-50"
                  >
                    <Send class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <!-- 子回复列表 -->
              <div v-if="repliesMap[comment.id]?.length" class="mt-4 space-y-3 pl-4 border-l-2 border-m3-sys-light-primary/20">
                <div v-for="reply in repliesMap[comment.id]" :key="reply.id" class="flex gap-3">
                  <img
                    :src="userAvatarUrl(reply.user)"
                    :alt="reply.user?.username"
                    class="w-8 h-8 rounded-full object-cover shrink-0 mt-1"
                    referrerpolicy="no-referrer"
                  />
                  <div class="flex-1">
                    <div class="bg-m3-sys-light-surface-variant/70 p-4 rounded-xl rounded-tl-none">
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-semibold text-sm text-m3-sys-light-on-surface">{{ reply.user?.username || '匿名用户' }}</span>
                        <span class="text-xs text-m3-sys-light-on-surface-variant">{{ formatDate(reply.created_at) }}</span>
                      </div>
                      <p class="text-sm text-m3-sys-light-on-surface opacity-90 leading-relaxed">{{ reply.content }}</p>
                    </div>
                    <!-- 删除回复 -->
                    <div v-if="canDelete(reply)" class="mt-1 ml-1">
                      <button
                        :disabled="deletingIds.has(reply.id)"
                        @click="deleteComment(reply.id)"
                        class="flex items-center gap-1 text-xs font-medium text-m3-sys-light-on-surface-variant hover:text-red-500 transition-colors disabled:opacity-50"
                      >
                        <Trash2 class="w-3 h-3" />
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </article>
</template>

