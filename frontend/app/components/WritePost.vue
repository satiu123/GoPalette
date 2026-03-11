<script setup lang="ts">
import { ArrowLeft, Image as ImageIcon, Send } from 'lucide-vue-next'

const router = useRouter()
const title = ref('')
const content = ref('')
const category = ref('Design')

const categories = ['Design', 'Development', 'UX', 'Typography', 'Psychology']

function goBack() {
  router.push('/')
}
</script>

<template>
  <div
    v-motion
    :initial="{ opacity: 0, y: 20 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500, ease: [0.22, 1, 0.36, 1] } }"
    class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <div class="flex items-center justify-between mb-12">
      <button
        @click="goBack"
        class="flex items-center gap-2 px-4 py-2 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface-variant font-medium"
      >
        <ArrowLeft class="w-5 h-5" />
        Back
      </button>

      <button class="flex items-center gap-2 px-6 py-3 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full font-bold hover:bg-m3-sys-light-on-primary-container transition-all shadow-lg shadow-m3-sys-light-primary/20">
        Publish
        <Send class="w-4 h-4" />
      </button>
    </div>

    <div class="space-y-8">
      <!-- Title Input -->
      <textarea
        v-model="title"
        placeholder="Article Title..."
        class="w-full bg-transparent text-5xl sm:text-6xl md:text-7xl font-black tracking-tighter leading-[1.1] text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant/40 resize-none focus:outline-none"
        rows="2"
      />

      <!-- Meta Data -->
      <div class="flex flex-wrap items-center gap-4 py-6 border-y border-m3-sys-light-surface-variant">
        <div class="flex items-center gap-3">
          <img
            src="https://picsum.photos/seed/currentuser/100/100"
            alt="You"
            class="w-10 h-10 rounded-full object-cover"
            referrerpolicy="no-referrer"
          />
          <span class="font-bold text-m3-sys-light-on-surface">You</span>
        </div>

        <div class="h-6 w-px bg-m3-sys-light-surface-variant hidden sm:block"></div>

        <div class="flex items-center gap-2 overflow-x-auto pb-2 sm:pb-0">
          <button
            v-for="cat in categories"
            :key="cat"
            @click="category = cat"
            class="px-4 py-1.5 rounded-full text-sm font-bold uppercase tracking-wider whitespace-nowrap transition-colors"
            :class="category === cat
              ? 'bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container'
              : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container/50'"
          >
            {{ cat }}
          </button>
        </div>
      </div>

      <!-- Cover Image Upload (Mock) -->
      <button class="w-full h-48 sm:h-64 border-2 border-dashed border-m3-sys-light-outline/30 rounded-[2.5rem] flex flex-col items-center justify-center gap-4 text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-surface-variant/50 hover:border-m3-sys-light-primary/50 transition-all group">
        <div class="p-4 bg-m3-sys-light-surface-variant rounded-full group-hover:bg-m3-sys-light-primary-container group-hover:text-m3-sys-light-on-primary-container transition-colors">
          <ImageIcon class="w-8 h-8" />
        </div>
        <span class="font-medium">Add a cover image</span>
      </button>

      <!-- Content Input -->
      <textarea
        v-model="content"
        placeholder="Start writing your story..."
        class="w-full bg-transparent text-xl leading-relaxed text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant/50 resize-none focus:outline-none min-h-[400px]"
      />
    </div>
  </div>
</template>
