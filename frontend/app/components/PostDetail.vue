<script setup lang="ts">
import { ArrowLeft, MessageSquare, Heart, Share2, Send } from 'lucide-vue-next'
import type { Article, Comment } from '~/composables/useBlogData'
import { articleImageUrl, userAvatarUrl, formatDate } from '~/composables/useBlogData'
import type { ApiResponse } from '~/composables/useBlogData'

const props = defineProps<{
  post: Article
}>()

const router = useRouter()
const { isLoggedIn, authFetch, user } = useAuth()
const isLiked = ref(false)
const commentText = ref('')
const submitting = ref(false)
const commentError = ref('')

// 获取评论列表
const { data: commentsData, refresh: refreshComments } = await useComments(props.post.id)
const comments = computed<Comment[]>(() => commentsData.value?.data ?? [])

function goBack() {
  router.push('/')
}

async function submitComment() {
  if (!commentText.value.trim()) return
  if (!isLoggedIn.value) {
    router.push('/login')
    return
  }
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
</script>

<template>
  <article
    v-motion
    :initial="{ opacity: 0, y: 20 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500, ease: [0.22, 1, 0.36, 1] } }"
    class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <button
      @click="goBack"
      class="mb-8 flex items-center gap-2 px-4 py-2 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface-variant font-medium"
    >
      <ArrowLeft class="w-5 h-5" />
      Back to Home
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
            :src="userAvatarUrl(post.author?.username ?? 'user')"
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

    <!-- 封面图 -->
    <div class="relative mb-16">
      <div class="absolute inset-0 bg-m3-sys-light-primary-container rounded-[3rem] transform -rotate-2 scale-105 -z-10"></div>
      <img
        :src="articleImageUrl(post)"
        :alt="post.title"
        class="w-full h-[400px] md:h-[500px] object-cover rounded-[2.5rem] shadow-xl"
        referrerpolicy="no-referrer"
      />
    </div>

    <!-- 文章正文（后端返回 HTML，bluemonday 已过滤 XSS） -->
    <div
      class="prose prose-lg max-w-none mb-20 text-m3-sys-light-on-surface leading-relaxed"
      v-html="post.content"
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

    <!-- 评论区 -->
    <section class="bg-m3-sys-light-surface-variant/50 rounded-[3rem] p-8 sm:p-12">
      <div class="flex items-center gap-3 mb-10">
        <MessageSquare class="w-8 h-8 text-m3-sys-light-primary" />
        <h2 class="text-3xl font-bold">Comments ({{ comments.length }})</h2>
      </div>

      <!-- 发表评论 -->
      <div class="flex gap-4 mb-12">
        <img
          :src="userAvatarUrl(user?.username ?? 'you')"
          alt="You"
          class="w-12 h-12 rounded-full object-cover shrink-0"
          referrerpolicy="no-referrer"
        />
        <div class="flex-grow relative">
          <textarea
            v-model="commentText"
            :placeholder="isLoggedIn ? 'Add to the discussion…' : 'Sign in to comment…'"
            :disabled="!isLoggedIn || submitting"
            class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant rounded-[1.5rem] p-5 pr-16 resize-none focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all shadow-sm min-h-[120px] disabled:opacity-60"
          />
          <button
            @click="submitComment"
            :disabled="!isLoggedIn || submitting || !commentText.trim()"
            class="absolute bottom-4 right-4 p-3 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full hover:bg-m3-sys-light-on-primary-container transition-colors shadow-md disabled:opacity-50"
          >
            <Send class="w-5 h-5" />
          </button>
        </div>
      </div>
      <p v-if="commentError" class="text-red-500 text-sm mb-4">{{ commentError }}</p>

      <!-- 未登录提示 -->
      <p v-if="!isLoggedIn" class="text-center text-m3-sys-light-on-surface-variant mb-8">
        <NuxtLink to="/login" class="text-m3-sys-light-primary font-semibold hover:underline">Sign in</NuxtLink>
        to join the discussion.
      </p>

      <!-- 评论列表 -->
      <div class="space-y-8">
        <div v-for="comment in comments" :key="comment.id" class="flex gap-4">
          <img
            :src="userAvatarUrl(comment.user?.username ?? 'user')"
            :alt="comment.user?.username"
            class="w-12 h-12 rounded-full object-cover shrink-0"
            referrerpolicy="no-referrer"
          />
          <div class="bg-m3-sys-light-surface p-6 rounded-[2rem] rounded-tl-none shadow-sm flex-grow">
            <div class="flex items-center justify-between mb-2">
              <span class="font-bold text-m3-sys-light-on-surface">{{ comment.user?.username ?? 'Anonymous' }}</span>
              <span class="text-sm text-m3-sys-light-on-surface-variant">{{ formatDate(comment.created_at) }}</span>
            </div>
            <p class="text-m3-sys-light-on-surface opacity-90 leading-relaxed">{{ comment.content }}</p>
          </div>
        </div>
        <p v-if="comments.length === 0" class="text-center text-m3-sys-light-on-surface-variant italic py-8">
          No comments yet. Be the first to share your thoughts!
        </p>
      </div>
    </section>
  </article>
</template>


