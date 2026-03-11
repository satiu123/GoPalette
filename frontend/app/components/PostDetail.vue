<script setup lang="ts">
import { ArrowLeft, MessageSquare, Heart, Share2, Send } from 'lucide-vue-next'
import type { BlogPost } from '~/composables/useBlogData'

defineProps<{
  post: BlogPost
}>()

const router = useRouter()
const isLiked = ref(false)
const commentText = ref('')

function goBack() {
  router.push('/')
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
        {{ post.category }}
      </div>
      <h1 class="text-4xl sm:text-5xl md:text-6xl font-black tracking-tighter leading-[1.1] text-m3-sys-light-on-surface mb-8">
        {{ post.title }}
      </h1>

      <div class="flex flex-wrap items-center justify-between gap-6 py-6 border-y border-m3-sys-light-surface-variant">
        <div class="flex items-center gap-4">
          <img
            :src="post.author.avatar"
            :alt="post.author.name"
            class="w-14 h-14 rounded-full object-cover"
            referrerpolicy="no-referrer"
          />
          <div>
            <p class="font-bold text-lg text-m3-sys-light-on-surface">{{ post.author.name }}</p>
            <p class="text-m3-sys-light-on-surface-variant">{{ post.date }} · {{ post.readTime }}</p>
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

    <div class="relative mb-16">
      <div class="absolute inset-0 bg-m3-sys-light-primary-container rounded-[3rem] transform -rotate-2 scale-105 -z-10"></div>
      <img
        :src="post.imageUrl"
        :alt="post.title"
        class="w-full h-[400px] md:h-[500px] object-cover rounded-[2.5rem] shadow-xl"
        referrerpolicy="no-referrer"
      />
    </div>

    <div class="prose prose-lg max-w-none mb-20 text-m3-sys-light-on-surface leading-relaxed">
      <p
        v-for="(paragraph, idx) in post.content.split('\n\n')"
        :key="idx"
        class="mb-6 text-xl opacity-90"
      >
        {{ paragraph }}
      </p>
    </div>

    <!-- Comments Section -->
    <section class="bg-m3-sys-light-surface-variant/50 rounded-[3rem] p-8 sm:p-12">
      <div class="flex items-center gap-3 mb-10">
        <MessageSquare class="w-8 h-8 text-m3-sys-light-primary" />
        <h2 class="text-3xl font-bold">Comments ({{ post.comments.length }})</h2>
      </div>

      <!-- Add Comment -->
      <div class="flex gap-4 mb-12">
        <img
          src="https://picsum.photos/seed/currentuser/100/100"
          alt="You"
          class="w-12 h-12 rounded-full object-cover shrink-0"
          referrerpolicy="no-referrer"
        />
        <div class="flex-grow relative">
          <textarea
            v-model="commentText"
            placeholder="Add to the discussion..."
            class="w-full bg-m3-sys-light-surface text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant rounded-[1.5rem] p-5 pr-16 resize-none focus:outline-none focus:ring-4 focus:ring-m3-sys-light-primary/20 transition-all shadow-sm min-h-[120px]"
          />
          <button class="absolute bottom-4 right-4 p-3 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full hover:bg-m3-sys-light-on-primary-container transition-colors shadow-md">
            <Send class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Comment List -->
      <div class="space-y-8">
        <div v-for="comment in post.comments" :key="comment.id" class="flex gap-4">
          <img
            :src="comment.author.avatar"
            :alt="comment.author.name"
            class="w-12 h-12 rounded-full object-cover shrink-0"
            referrerpolicy="no-referrer"
          />
          <div class="bg-m3-sys-light-surface p-6 rounded-[2rem] rounded-tl-none shadow-sm flex-grow">
            <div class="flex items-center justify-between mb-2">
              <span class="font-bold text-m3-sys-light-on-surface">{{ comment.author.name }}</span>
              <span class="text-sm text-m3-sys-light-on-surface-variant">{{ comment.date }}</span>
            </div>
            <p class="text-m3-sys-light-on-surface opacity-90 leading-relaxed">{{ comment.content }}</p>
          </div>
        </div>
        <p v-if="post.comments.length === 0" class="text-center text-m3-sys-light-on-surface-variant italic py-8">
          No comments yet. Be the first to share your thoughts!
        </p>
      </div>
    </section>
  </article>
</template>
