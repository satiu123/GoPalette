<script setup lang="ts">
import { ArrowRight } from 'lucide-vue-next'
import type { Article } from '~/composables/useBlogData'
import { articleImageUrl, userAvatarUrl, formatDate } from '~/composables/useBlogData'

const props = defineProps<{
  post: Article
}>()

const router = useRouter()

function goToPost() {
  router.push(`/post/${props.post.id}`)
}
</script>

<template>
  <section class="py-12 sm:py-20 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
      <div
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :enter="{ opacity: 1, y: 0, transition: { duration: 600, ease: [0.22, 1, 0.36, 1] } }"
        class="flex flex-col gap-6"
      >
        <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container w-fit text-sm font-medium tracking-wide">
          <span class="w-2 h-2 rounded-full bg-m3-sys-light-tertiary animate-pulse"></span>
          Featured Article
        </div>

        <h1
          @click="goToPost"
          class="text-5xl sm:text-6xl lg:text-7xl font-black tracking-tighter leading-[1.1] text-m3-sys-light-on-surface cursor-pointer hover:opacity-80 transition-opacity"
        >
          {{ post.title }}
        </h1>

        <p class="text-xl text-m3-sys-light-on-surface-variant leading-relaxed max-w-xl">
          {{ post.summary || post.content.replace(/<[^>]*>/g, '').substring(0, 160) + '…' }}
        </p>

        <div class="flex items-center gap-4 mt-4">
          <img
            :src="userAvatarUrl(post.author?.username ?? 'user')"
            :alt="post.author?.username"
            class="w-12 h-12 rounded-full object-cover border-2 border-m3-sys-light-surface"
            referrerpolicy="no-referrer"
          />
          <div>
            <p class="font-semibold text-m3-sys-light-on-surface">{{ post.author?.username }}</p>
            <p class="text-sm text-m3-sys-light-on-surface-variant">{{ formatDate(post.created_at) }} · {{ post.read_count }} views</p>
          </div>
        </div>

        <button
          @click="goToPost"
          class="mt-8 flex items-center gap-3 px-8 py-4 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full w-fit font-semibold text-lg hover:bg-m3-sys-light-on-primary-container hover:scale-105 transition-all duration-300 shadow-lg shadow-m3-sys-light-primary/20"
        >
          Read Article
          <ArrowRight class="w-5 h-5" />
        </button>
      </div>

      <div
        v-motion
        :initial="{ opacity: 0, scale: 0.95 }"
        :enter="{ opacity: 1, scale: 1, transition: { duration: 800, ease: [0.22, 1, 0.36, 1], delay: 200 } }"
        class="relative cursor-pointer"
        @click="goToPost"
      >
        <div class="absolute inset-0 bg-m3-sys-light-secondary-container rounded-[3rem] transform rotate-3 scale-105 -z-10"></div>
        <img
          :src="articleImageUrl(post)"
          :alt="post.title"
          class="w-full h-[500px] object-cover rounded-[2.5rem] shadow-2xl transition-transform duration-500 hover:scale-[1.02]"
          referrerpolicy="no-referrer"
        />
        <div class="absolute bottom-6 right-6 bg-m3-sys-light-surface/90 backdrop-blur-md px-6 py-3 rounded-full shadow-lg">
          <span class="font-medium text-m3-sys-light-primary tracking-wide uppercase text-sm">
            {{ post.category?.name ?? 'Article' }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

