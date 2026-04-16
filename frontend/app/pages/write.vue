<script setup lang="ts">
import type { EditorCustomHandlers } from '@nuxt/ui'
import type { Editor } from '@tiptap/core'
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CellSelection } from '@tiptap/pm/tables'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { ImageUpload } from '~/components/editor/ImageUploadExtension'
import { createPost, deletePost, fetchCategories, fetchPostBySlug, fetchTags, POST_STATUS_DRAFT, POST_STATUS_PUBLISHED, updatePost } from '~/composables/useBlogApi'

const route = useRoute()
const runtimeConfig = useRuntimeConfig()
const toast = useToast()
const { initAuth, isLoggedIn } = useAuth()

const room = computed(() => route.query.room as string | undefined)

const user = useState('user', () => ({
  name: getRandomName(),
  color: getRandomColor()
}))

const appConfig = useAppConfig()

const editorRef = useTemplateRef('editorRef')

const { extension: Completion, handlers: aiHandlers, isLoading: aiLoading } = useEditorCompletion(editorRef)

const {
  enabled: collaborationEnabled,
  ready: collaborationReady,
  extensions: collaborationExtensions,
  connectedUsers
} = useEditorCollaboration({
  room: room.value,
  host: runtimeConfig.public.partykitHost,
  user: {
    name: user.value.name,
    color: COLORS[user.value.color]!
  }
})

// Set primary color for the app
if (collaborationEnabled) {
  appConfig.ui.colors.primary = user.value.color
}

// Custom handlers for editor (merged with AI handlers)
const customHandlers = {
  imageUpload: {
    canExecute: (editor: Editor) => editor.can().insertContent({ type: 'imageUpload' }),
    execute: (editor: Editor) => editor.chain().focus().insertContent({ type: 'imageUpload' }),
    isActive: (editor: Editor) => editor.isActive('imageUpload'),
    isDisabled: undefined
  },
  table: {
    canExecute: (editor: Editor) => editor.can().insertTable({ rows: 3, cols: 3, withHeaderRow: true }),
    execute: (editor: Editor) => editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }),
    isActive: (editor: Editor) => editor.isActive('table'),
    isDisabled: undefined
  },
  ...aiHandlers
} satisfies EditorCustomHandlers

const { items: emojiItems, extension: Emoji } = useEditorEmojis()
const { items: mentionItems } = useEditorMentions(connectedUsers)
const { items: suggestionItems } = useEditorSuggestions(customHandlers)
const { getItems: getDragHandleItems, onNodeChange } = useEditorDragHandle(customHandlers)
const { toolbarItems, bubbleToolbarItems, getImageToolbarItems, getTableToolbarItems } = useEditorToolbar(customHandlers, { aiLoading })

const postMeta = reactive({
  id: '',
  title: '',
  summary: '',
  slug: '',
  tagsText: '',
  categoryId: ''
})

const { data: categoriesData } = await useAsyncData('write-categories', () => fetchCategories(1, 200))
const { data: tagsData } = await useAsyncData('write-tags', () => fetchTags(1, 200))

const categories = computed(() => categoriesData.value?.categories || [])
const tagSuggestions = computed(() => (tagsData.value?.tags || []).map(item => item.name))
const selectedTags = computed(() => parseTags(postMeta.tagsText))

const isSaving = ref(false)
const deletingPost = ref(false)

const canSubmit = computed(() => {
  return Boolean(postMeta.title.trim() && content.value.trim())
})

// Default content - only used when Y.js document is empty
const content = ref(`# Nuxt Editor Template :sparkles:

A Notion-like WYSIWYG editor with AI-powered completions and real-time collaboration in [Vue](https://vuejs.org/) & [Nuxt](https://nuxt.com/).

> Add [\`?room=my-room\`](/write?room=my-room) to the URL and share the link to collaborate with others.

---

## Rich Text Editing

Full formatting support with **bold**, *italic*, <u>underline</u>, ~~strikethrough~~, and \`inline code\`.

![Image Placeholder](/placeholder.jpeg)

### Code Blocks

Code blocks are supported with syntax highlighting using [Shiki](https://shiki.dev/).

\`\`\`vue
<template>
  <UEditor v-slot="{ editor }" v-model="value" content-type="markdown">
    <UEditorToolbar :editor="editor" :items="items" />
  </UEditor>
</template>
\`\`\`

### Lists

1. Numbered lists for sequential items
2. With automatic numbering

- Bullet lists work too
  - With nested items
  - At multiple levels

- [ ] Task lists for todos
- [x] Mark items as complete

### Tables

Insert and edit tables with row/column controls and cell selection.

| Feature | Description | Status |
| ------- | ----------- | ------ |
| Tables | Full table support | ✅ |
| Markdown | Content serialization | ✅ |

---

## Features

### Bubble & Fixed Toolbars

Select text to see the bubble toolbar with formatting options. The fixed toolbar at the top provides quick access to common actions.

### Drag Handle

Use the drag handle on the left side of any block to reorder, duplicate, delete, or convert between block types.

### Slash Commands

Type \`/\` anywhere to access quick insertion commands for headings, lists, code blocks, tables, images, and more.

### Image Upload

Custom image upload node powered by [\`UFileUpload\`](https://ui.nuxt.com/docs/components/file-upload) component and [NuxtHub](https://hub.nuxt.com/docs/blob) with [Vercel Blob](https://vercel.com/docs/vercel-blob) support.

<div data-type="image-upload"></div>

### Mentions & Emojis

Mention collaborators with \`@\` and add emojis with \`:\` syntax :rocket:

### AI-powered Features

Inline completions and text transformations powered by [AI SDK](https://ai-sdk.dev/).

- **Autocompletion**: Suggestions appear as you type
- **Selection actions**: Fix, extend, simplify, or translate selected text

> *Pro tip: Press \`⌘J\` to manually trigger AI completion.*

### Real-time Collaboration

Collaborative editing powered by [PartyKit](https://partykit.io/). Add [\`?room=my-room\`](/write?room=my-room) to the URL and share the link to collaborate with others in real-time. See collaborators' cursors and selections as they type.

---

Visit the [Nuxt UI documentation](https://ui.nuxt.com/docs/components/editor) to learn more about the Editor component.
`)

// Set initial content for collaborative documents (only if empty)
function onCreate({ editor }: { editor: Editor }) {
  if (!collaborationEnabled) return

  const storageKey = `editor-initialized-${room.value}`

  // Skip if already initialized this session (handles HMR)
  if (sessionStorage.getItem(storageKey)) return

  // Wait for Y.js to sync existing content from server before checking if empty
  setTimeout(() => {
    const text = editor.state.doc.textContent.trim()
    if (!text) {
      editor.commands.setContent(content.value, { contentType: 'markdown' })
    }
    sessionStorage.setItem(storageKey, 'true')
  }, 500)
}

function onUpdate(value: string) {
  content.value = value
}

function toSlug(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\u4e00-\u9fa5\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}

function plainTextSummary(markdown: string) {
  const text = markdown
    .replace(/[#>*`_~\[\]()!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  return text.slice(0, 140)
}

function parseTags(tagsText: string) {
  return Array.from(new Set(tagsText
    .split(/[，,\s]+/)
    .map(tag => tag.trim())
    .filter(Boolean)))
}

function addTag(tag: string) {
  const merged = Array.from(new Set([...selectedTags.value, tag]))
  postMeta.tagsText = merged.join(', ')
}

function removeTag(tag: string) {
  const merged = selectedTags.value.filter(item => item !== tag)
  postMeta.tagsText = merged.join(', ')
}

function normalizeSlug() {
  postMeta.slug = toSlug(postMeta.slug || postMeta.title)
}

async function savePost(status: number) {
  if (!canSubmit.value || isSaving.value) return

  isSaving.value = true

  try {
    const payload = {
      title: postMeta.title.trim(),
      summary: (postMeta.summary || plainTextSummary(content.value)).trim(),
      slug: (postMeta.slug || toSlug(postMeta.title)).trim(),
      content: content.value,
      status,
      categoryId: postMeta.categoryId.trim() || undefined,
      tags: parseTags(postMeta.tagsText)
    }

    const saved = postMeta.id
      ? await updatePost(postMeta.id, payload)
      : await createPost(payload)

    if (!saved) {
      throw new Error('保存失败，后端没有返回文章对象')
    }

    postMeta.id = saved.id
    postMeta.slug = saved.slug
    postMeta.summary = saved.summary
    postMeta.tagsText = saved.tags.join(', ')

    toast.add({
      title: status === POST_STATUS_PUBLISHED ? '发布成功' : '草稿已保存',
      description: `文章标识：${saved.id}`,
      color: 'success'
    })
  } catch (error: any) {
    toast.add({
      title: '保存失败',
      description: error?.message || '请检查网关与后端服务状态',
      color: 'error'
    })
  } finally {
    isSaving.value = false
  }
}

function saveDraft() {
  return savePost(POST_STATUS_DRAFT)
}

function publishNow() {
  return savePost(POST_STATUS_PUBLISHED)
}

async function removeCurrentPost() {
  if (!postMeta.id || deletingPost.value) return
  if (import.meta.client && !window.confirm('确认删除当前文章吗？此操作不可恢复。')) return

  deletingPost.value = true
  try {
    await deletePost(postMeta.id)
    toast.add({
      title: '文章已删除',
      color: 'success'
    })
    await navigateTo('/admin')
  } catch (error: unknown) {
    const typed = error as { message?: string, data?: { message?: string } }
    toast.add({
      title: '删除失败',
      description: typed?.data?.message || typed?.message || '请稍后重试',
      color: 'error'
    })
  } finally {
    deletingPost.value = false
  }
}

watch(() => postMeta.title, (value) => {
  if (!postMeta.slug.trim()) {
    postMeta.slug = toSlug(value)
  }
})

const editingSlug = computed(() => typeof route.query.slug === 'string' ? route.query.slug : '')

if (editingSlug.value) {
  const existingPost = await fetchPostBySlug(editingSlug.value)

  if (existingPost) {
    postMeta.id = existingPost.id
    postMeta.title = existingPost.title
    postMeta.summary = existingPost.summary
    postMeta.slug = existingPost.slug
    postMeta.tagsText = existingPost.tags.join(', ')
    postMeta.categoryId = existingPost.categoryId
    content.value = existingPost.content || content.value
  }
}

onMounted(async () => {
  initAuth()
  if (!isLoggedIn.value) {
    await navigateTo('/login?redirect=/write')
  }
})

useSeoMeta({
  title: '写作工作台',
  description: 'GoPalette 富文本写作工作台，支持草稿保存与文章发布。'
})

const extensions = computed(() => [
  CodeBlockShiki.configure({
    defaultTheme: 'material-theme',
    themes: {
      light: 'material-theme-lighter',
      dark: 'material-theme-palenight'
    }
  }),
  Completion,
  Emoji,
  ImageUpload,
  TableKit,
  TaskList,
  TaskItem,
  ...collaborationExtensions.value
])
</script>

<template>
  <UEditor v-if="collaborationReady" ref="editorRef" v-slot="{ editor, handlers }"
    :model-value="collaborationEnabled ? undefined : content" content-type="markdown" :extensions="extensions"
    :starter-kit="collaborationEnabled ? { undoRedo: false } : undefined" :handlers="customHandlers" autofocus
    placeholder="Write, type '/' for commands..." class="min-h-screen" :ui="{
      base: 'p-4 sm:p-14',
      content: 'max-w-4xl mx-auto'
    }" @update:model-value="onUpdate" @create="onCreate">
    <AppHeader>
      <div class="flex items-center gap-2">
        <UButton icon="i-lucide-save" color="neutral" variant="soft" size="sm" :disabled="!canSubmit || isSaving"
          :loading="isSaving" label="草稿" @click="saveDraft" />

        <UButton icon="i-lucide-send" size="sm" :disabled="!canSubmit || isSaving" :loading="isSaving" label="发布"
          @click="publishNow" />

        <UButton
          v-if="postMeta.id"
          icon="i-lucide-trash-2"
          size="sm"
          color="error"
          variant="ghost"
          :loading="deletingPost"
          label="删除"
          @click="removeCurrentPost"
        />
      </div>

      <EditorCollaborationUsers :users="connectedUsers" />

      <UEditorToolbar :editor="editor" :items="toolbarItems" />
    </AppHeader>

    <section class="mx-auto mb-4 w-full max-w-4xl">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <p class="text-sm font-medium text-highlighted">
              文章信息
            </p>
            <p class="text-xs text-toned">
              发布前请确认标题、slug、分类、标签与摘要
            </p>
          </div>
        </template>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="标题" name="title" required>
            <UInput v-model="postMeta.title" placeholder="请输入文章标题" />
          </UFormField>

          <UFormField label="Slug" name="slug" required>
            <UInput v-model="postMeta.slug" placeholder="article-slug" @blur="normalizeSlug" />
          </UFormField>

          <UFormField label="分类" name="categoryId">
            <UInput v-model="postMeta.categoryId" placeholder="请输入分类 ID" />
            <div class="mt-2 flex flex-wrap gap-2">
              <UButton v-for="category in categories" :key="category.id" size="xs"
                :label="`${category.name} (${category.id})`"
                :variant="postMeta.categoryId === category.id ? 'solid' : 'ghost'"
                :color="postMeta.categoryId === category.id ? 'primary' : 'neutral'"
                @click="postMeta.categoryId = category.id" />
            </div>
          </UFormField>

          <UFormField label="标签" name="tagsText">
            <UInput v-model="postMeta.tagsText" placeholder="多个标签用逗号分隔" />
            <div class="mt-2 flex flex-wrap gap-2">
              <UBadge v-for="tag in selectedTags" :key="`selected-${tag}`" color="primary" variant="subtle"
                class="cursor-pointer" :label="`#${tag} ×`" @click="removeTag(tag)" />
            </div>
            <div class="mt-2 flex flex-wrap gap-2">
              <UButton v-for="tag in tagSuggestions" :key="`suggest-${tag}`" size="xs" color="neutral" variant="ghost"
                :label="`#${tag}`" @click="addTag(tag)" />
            </div>
          </UFormField>

          <UFormField label="摘要" name="summary" class="md:col-span-2">
            <UTextarea v-model="postMeta.summary" :rows="3" autoresize placeholder="不填写时将自动从正文提取前 140 字" />
          </UFormField>
        </div>
      </UCard>
    </section>

    <UEditorToolbar :editor="editor" :items="bubbleToolbarItems" layout="bubble" :should-show="({ editor, view, state }: any) => {
      if (editor.isActive('imageUpload') || editor.isActive('image') || state.selection instanceof CellSelection) {
        return false
      }
      const { selection } = state
      return view.hasFocus() && !selection.empty
    }">
      <template #link>
        <EditorLinkPopover :editor="editor" />
      </template>
    </UEditorToolbar>

    <UEditorToolbar :editor="editor" :items="getImageToolbarItems(editor)" layout="bubble" :should-show="({ editor, view }: any) => {
      return editor.isActive('image') && view.hasFocus()
    }" />

    <UEditorToolbar :editor="editor" :items="getTableToolbarItems(editor)" layout="bubble" :should-show="({ editor, view }: any) => {
      return editor.state.selection instanceof CellSelection && view.hasFocus()
    }" />

    <UEditorEmojiMenu :editor="editor" :items="emojiItems" />

    <UEditorMentionMenu :editor="editor" :items="mentionItems" />

    <UEditorSuggestionMenu :editor="editor" :items="suggestionItems" />

    <UEditorDragHandle v-slot="{ ui, onClick }" :editor="editor" @node-change="onNodeChange">
      <UButton icon="i-lucide-plus" color="neutral" variant="ghost" size="sm" :class="ui.handle()" @click="(e: MouseEvent) => {
        e.stopPropagation()
        const node = onClick()

        handlers.suggestion?.execute(editor, { pos: node?.pos }).run()
      }" />

      <UDropdownMenu v-slot="{ open }" :modal="false" :items="getDragHandleItems(editor)" :content="{ side: 'left' }"
        :ui="{ content: 'w-48', label: 'text-xs' }"
        @update:open="editor.chain().setMeta('lockDragHandle', $event).run()">
        <UButton color="neutral" variant="ghost" active-variant="soft" size="sm" icon="i-lucide-grip-vertical"
          :active="open" :class="ui.handle()" />
      </UDropdownMenu>
    </UEditorDragHandle>
  </UEditor>
</template>
