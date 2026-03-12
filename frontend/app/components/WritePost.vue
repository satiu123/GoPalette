<script setup lang="ts">
import { ArrowLeft, Send, Bold, Italic, List, ListOrdered, Image as ImageIcon, Quote, Code } from 'lucide-vue-next'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import type { ApiResponse, Article } from '~/composables/useBlogData'

const router = useRouter()
const route  = useRoute()
const { isLoggedIn, authFetch } = useAuth()

// 未登录跳到登录页
onMounted(() => {
  if (!isLoggedIn.value) router.push('/login')
})

// 编辑模式（?edit=:id）vs 新建模式 
const editId = computed(() =>
  route.query.edit ? Number(route.query.edit) : null
)
const isEdit = computed(() => editId.value !== null)

const title   = ref('')
const summary = ref('')
const status  = ref<'draft' | 'published'>('draft')
const selectedCategoryId = ref<number | null>(null)
const selectedTagIds     = ref<number[]>([])
const submitting = ref(false)
const error      = ref('')
const loading    = ref(false)

// Tiptap 富文本编辑器
const editor = useEditor({
  content: '',
  extensions: [
    StarterKit,
    Image.configure({ inline: false, allowBase64: false }),
  ],
})

// 图片上传
const fileInput = ref<HTMLInputElement | null>(null)
const imageUploading = ref(false)

async function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files?.length) return
  const file = input.files[0]!
  if (file.size > 5 * 1024 * 1024) {
    error.value = '图片不能超过 5MB'
    return
  }
  imageUploading.value = true
  try {
    const formData = new FormData()
    formData.append('image', file)
    const res = await authFetch<ApiResponse<{ url: string }>>('/upload', {
      method: 'POST',
      body: formData,
    })
    if (res.code === 200) {
      editor.value?.chain().focus().setImage({ src: res.data.url }).run()
    }
  } catch {
    error.value = '图片上传失败，请重试'
  } finally {
    imageUploading.value = false
    input.value = ''
  }
}

onBeforeUnmount(() => editor.value?.destroy())

// 拉取真实分类 & 标签
const { data: catData }  = await useCategories()
const { data: tagData }  = await useTags()
const categories = computed(() => catData.value?.data ?? [])
const tags       = computed(() => tagData.value?.data  ?? [])

// 编辑模式：回填原文章数据
onMounted(async () => {
  if (!editId.value) return
  loading.value = true
  try {
    const res = await authFetch<ApiResponse<Article>>(`/articles/${editId.value}`)
    if (res.code === 200) {
      const a = res.data
      title.value   = a.title
      summary.value = a.summary ?? ''
      status.value  = (a.status as 'draft' | 'published') ?? 'draft'
      selectedCategoryId.value = a.category_id ? Number(a.category_id) : null
      selectedTagIds.value     = (a.tags ?? []).map(t => Number(t.id))
      await nextTick()
      editor.value?.commands.setContent(a.content)
    } else {
      error.value = '加载文章失败'
    }
  } catch {
    error.value = '加载文章失败'
  } finally {
    loading.value = false
  }
})

function toggleTag(id: number) {
  const idx = selectedTagIds.value.indexOf(id)
  if (idx === -1) selectedTagIds.value.push(id)
  else selectedTagIds.value.splice(idx, 1)
}

async function publish(draft = false) {
  const htmlContent = editor.value?.getHTML() ?? ''
  const textContent = editor.value?.getText().trim() ?? ''
  if (!title.value.trim() || !textContent) {
    error.value = '标题和正文不能为空'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const body = {
      title:       title.value.trim(),
      summary:     summary.value.trim() || undefined,
      content:     htmlContent,
      category_id: selectedCategoryId.value ?? 0,
      tag_ids:     selectedTagIds.value,
      status:      draft ? 'draft' : 'published'
    }

    let res: ApiResponse<Article>
    if (isEdit.value) {
      // 编辑模式：PUT /api/articles/:id
      res = await authFetch<ApiResponse<Article>>(`/articles/${editId.value}`, {
        method: 'PUT',
        body
      })
    } else {
      // 新建模式：POST /api/articles
      res = await authFetch<ApiResponse<Article>>('/articles', {
        method: 'POST',
        body
      })
    }

    if (res.code === 200) {
      router.push(`/post/${res.data.id}`)
    } else {
      error.value = res.msg
    }
  } catch (e: unknown) {
    error.value = (e as Error)?.message ?? (isEdit.value ? '更新失败，请重试' : '发布失败，请重试')
  } finally {
    submitting.value = false
  }
}

// 退出编辑：回到上一个历史页面，若无历史则返回文章详情（编辑模式）或首页
function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else if (isEdit.value) {
    router.push(`/post/${editId.value}`)
  } else {
    router.push('/')
  }
}
</script>

<template>
  <div
    v-motion
    :initial="{ opacity: 0, y: 20 }"
    :enter="{ opacity: 1, y: 0, transition: { duration: 500, ease: [0.22, 1, 0.36, 1] } }"
    class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"
  >
    <!-- 加载中骨架 -->
    <div v-if="loading" class="space-y-8 animate-pulse">
      <div class="h-8 w-48 bg-m3-sys-light-surface-variant rounded-full" />
      <div class="h-20 bg-m3-sys-light-surface-variant rounded-2xl" />
      <div class="h-10 bg-m3-sys-light-surface-variant rounded-2xl" />
      <div class="h-64 bg-m3-sys-light-surface-variant rounded-2xl" />
    </div>

    <template v-else>
      <div class="flex items-center justify-between mb-12">
        <button
          @click="goBack"
          class="flex items-center gap-2 px-4 py-2 rounded-full hover:bg-m3-sys-light-surface-variant transition-colors text-m3-sys-light-on-surface-variant font-medium"
        >
          <ArrowLeft class="w-5 h-5" />
          {{ isEdit ? 'Cancel Edit' : 'Back' }}
        </button>

        <div class="flex gap-3">
          <button
            @click="publish(true)"
            :disabled="submitting"
            class="px-5 py-2.5 rounded-full font-medium bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-all disabled:opacity-50"
          >
            Save Draft
          </button>
          <button
            @click="publish(false)"
            :disabled="submitting"
            class="flex items-center gap-2 px-6 py-3 bg-m3-sys-light-primary text-m3-sys-light-on-primary rounded-full font-bold hover:bg-m3-sys-light-on-primary-container transition-all shadow-lg shadow-m3-sys-light-primary/20 disabled:opacity-50"
          >
            {{ isEdit ? 'Update' : 'Publish' }}
            <Send class="w-4 h-4" />
          </button>
        </div>
      </div>

      <p v-if="error" class="mb-6 text-red-500 text-sm">{{ error }}</p>

      <div class="space-y-8">
        <!-- 标题 -->
        <textarea
          v-model="title"
          placeholder="Article Title…"
          class="w-full bg-transparent text-5xl sm:text-6xl md:text-7xl font-black tracking-tighter leading-[1.1] text-m3-sys-light-on-surface placeholder:text-m3-sys-light-on-surface-variant/40 resize-none focus:outline-none"
          rows="2"
        />

        <!-- 摘要 -->
        <input
          v-model="summary"
          placeholder="Short summary (optional)…"
          class="w-full bg-transparent text-lg text-m3-sys-light-on-surface-variant placeholder:text-m3-sys-light-on-surface-variant/40 focus:outline-none border-b border-m3-sys-light-surface-variant pb-2"
        />

        <!-- 分类 & 标签 -->
        <div class="flex flex-wrap items-start gap-6 py-6 border-y border-m3-sys-light-surface-variant">
          <!-- 分类 -->
          <div>
            <p class="text-xs font-bold uppercase tracking-wider text-m3-sys-light-on-surface-variant mb-2">Category</p>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="cat in categories"
                :key="cat.id"
                @click="selectedCategoryId = selectedCategoryId === cat.id ? null : cat.id"
                class="px-4 py-1.5 rounded-full text-sm font-bold uppercase tracking-wider whitespace-nowrap transition-colors"
                :class="selectedCategoryId === cat.id
                  ? 'bg-m3-sys-light-secondary-container text-m3-sys-light-on-secondary-container'
                  : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container/50'"
              >
                {{ cat.name }}
              </button>
            </div>
          </div>

          <!-- 标签 -->
          <div v-if="tags.length">
            <p class="text-xs font-bold uppercase tracking-wider text-m3-sys-light-on-surface-variant mb-2">Tags</p>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="tag in tags"
                :key="tag.id"
                @click="toggleTag(tag.id)"
                class="px-3 py-1 rounded-full text-sm font-medium whitespace-nowrap transition-colors"
                :class="selectedTagIds.includes(tag.id)
                  ? 'bg-m3-sys-light-tertiary-container text-m3-sys-light-on-tertiary-container'
                  : 'bg-m3-sys-light-surface-variant text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-tertiary-container/50'"
              >
                #{{ tag.name }}
              </button>
            </div>
          </div>
        </div>

        <!-- 正文 / 富文本编辑器 -->
        <div class="border border-m3-sys-light-outline-variant rounded-2xl overflow-hidden">
          <!-- 工具栏 -->
          <div class="flex flex-wrap gap-1 p-2 bg-m3-sys-light-surface-variant border-b border-m3-sys-light-outline-variant">
            <button type="button" @click="editor?.chain().focus().toggleBold().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('bold') }">
              <Bold class="w-4 h-4" />
            </button>
            <button type="button" @click="editor?.chain().focus().toggleItalic().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('italic') }">
              <Italic class="w-4 h-4" />
            </button>
            <span class="w-px h-6 bg-m3-sys-light-outline-variant mx-1 self-center" />
            <button type="button" @click="editor?.chain().focus().toggleHeading({ level: 2 }).run()"
              class="flex items-center px-2.5 py-2 rounded-lg text-xs font-bold text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('heading', { level: 2 }) }">H2</button>
            <button type="button" @click="editor?.chain().focus().toggleHeading({ level: 3 }).run()"
              class="flex items-center px-2.5 py-2 rounded-lg text-xs font-bold text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('heading', { level: 3 }) }">H3</button>
            <span class="w-px h-6 bg-m3-sys-light-outline-variant mx-1 self-center" />
            <button type="button" @click="editor?.chain().focus().toggleBulletList().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('bulletList') }">
              <List class="w-4 h-4" />
            </button>
            <button type="button" @click="editor?.chain().focus().toggleOrderedList().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('orderedList') }">
              <ListOrdered class="w-4 h-4" />
            </button>
            <span class="w-px h-6 bg-m3-sys-light-outline-variant mx-1 self-center" />
            <button type="button" @click="editor?.chain().focus().toggleBlockquote().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('blockquote') }">
              <Quote class="w-4 h-4" />
            </button>
            <button type="button" @click="editor?.chain().focus().toggleCode().run()"
              class="flex items-center p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors"
              :class="{ 'bg-m3-sys-light-secondary-container': editor?.isActive('code') }">
              <Code class="w-4 h-4" />
            </button>
            <span class="w-px h-6 bg-m3-sys-light-outline-variant mx-1 self-center" />
            <button type="button" @click="fileInput?.click()" :disabled="imageUploading"
              class="flex items-center gap-1.5 p-2 rounded-lg text-m3-sys-light-on-surface-variant hover:bg-m3-sys-light-secondary-container transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              <ImageIcon class="w-4 h-4" />
              <span class="text-xs">{{ imageUploading ? '上传中…' : 'Image' }}</span>
            </button>
          </div>

          <!-- 编辑器内容区 -->
          <EditorContent
            :editor="editor"
            class="prose prose-lg max-w-none p-6 min-h-[400px] text-m3-sys-light-on-surface focus:outline-none [&_.ProseMirror]:outline-none [&_.ProseMirror]:min-h-[360px]"
          />
        </div>

        <!-- 隐藏的图片上传 input -->
        <input ref="fileInput" type="file" accept="image/jpeg,image/png,image/webp,image/gif" class="hidden" @change="handleImageUpload" />
      </div>
    </template>
  </div>
</template>