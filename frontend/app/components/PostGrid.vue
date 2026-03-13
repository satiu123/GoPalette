<script setup lang="ts">
import type { Article } from '~/composables/useBlogData'
import { articleImageUrl, userAvatarUrl, formatDate } from '~/composables/useBlogData'

defineProps<{
  posts: Article[]
  total?: number
  params?: { page: number; page_size: number }
}>()

const emit = defineEmits<{
  'update:params': [value: { page: number; page_size: number; category_id?: number; tag_id?: number }]
}>()

const router = useRouter()

const cardStyles = [
  'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant',
  'bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container',
  'bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container',
  'bg-m3-sys-light-primary-container text-m3-sys-light-on-primary-container'
]

function getCardStyle(index: number): string {
  return cardStyles[index % cardStyles.length] ?? cardStyles[0] ?? ''
}

function goToPost(post: Article) {
  router.push(`/post/${post.id}`)
}
</script>

<template>
  <section class="py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
    <div class="flex items-end justify-between mb-12">
      <h2 class="text-4xl sm:text-5xl font-bold tracking-tight text-m3-sys-light-on-surface">
        Latest Stories
      </h2>
      <span v-if="total" class="hidden sm:inline-flex text-m3-sys-light-on-surface-variant text-sm">
        {{ total }} articles
      </span>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
      <article
        v-for="(post, index) in posts"
        :key="post.id"
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :visible-once="{ opacity: 1, y: 0, transition: { duration: 500, delay: index * 100 } }"
        @click="goToPost(post)"
        class="flex flex-col rounded-[2rem] overflow-hidden shadow-sm hover:shadow-xl hover:-translate-y-2 hover:scale-[1.02] transition-all duration-300 cursor-pointer"
        :class="getCardStyle(index)"
      >
        <div class="relative h-64 overflow-hidden">
          <img
            :src="articleImageUrl(post)"
            :alt="post.title"
            class="w-full h-full object-cover transition-transform duration-700 hover:scale-110"
            referrerpolicy="no-referrer"
          />
          <div class="absolute top-4 left-4 bg-m3-sys-light-surface/90 backdrop-blur-md px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-wider text-m3-sys-light-on-surface">
            {{ post.category?.name ?? 'Article' }}
          </div>
        </div>

        <div class="p-8 flex flex-col flex-grow">
          <div class="flex items-center gap-2 text-sm font-medium opacity-80 mb-4">
            <span>{{ formatDate(post.created_at) }}</span>
            <span>·</span>
            <span>{{ post.read_count }} views</span>
          </div>

          <h3 class="text-2xl font-bold leading-tight mb-4 line-clamp-2">
            {{ post.title }}
          </h3>

          <p class="opacity-90 leading-relaxed mb-8 line-clamp-3 flex-grow">
            {{ post.summary || post.content.replace(/<[^>]*>/g, '').substring(0, 120) }}
          </p>

          <div class="flex items-center gap-3 mt-auto pt-6 border-t border-current/10">
            <img
              :src="userAvatarUrl(post.author)"
              :alt="post.author?.username"
              class="w-10 h-10 rounded-full object-cover"
              referrerpolicy="no-referrer"
            />
            <span class="font-semibold">{{ post.author?.username }}</span>
          </div>
        </div>
      </article>
    </div>

    <div v-if="posts.length === 0" class="text-center py-20 text-m3-sys-light-on-surface-variant">
      No articles found.
    </div>
  </section>
</template>


