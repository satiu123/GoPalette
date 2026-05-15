<script setup lang="ts">
import type { EditorCustomHandlers } from '@nuxt/ui'
import type { Editor } from '@tiptap/core'
import type { Selection } from '@tiptap/pm/state'
import type { EditorView } from '@tiptap/pm/view'
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CellSelection } from '@tiptap/pm/tables'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { ImageUpload } from '~/components/editor/ImageUploadExtension'
import { createPost, deletePost, fetchPostBySlug, POST_STATUS_DRAFT, POST_STATUS_PRIVATE, POST_STATUS_PUBLISHED, updatePost } from '~/composables/useBlogApi'

const route = useRoute()
const runtimeConfig = useRuntimeConfig()
const toast = useToast()
const { csrf, headerName } = useCsrf()
const { categories, tagSuggestions } = useWriteResources()

const room = computed(() => route.query.room as string | undefined)

const user = useState('user', () => ({
  name: getRandomName(),
  color: getRandomColor()
}))

const appConfig = useAppConfig()

const editorRef = useTemplateRef('editorRef')

type EditorToolbarContext = { editor: Editor, view: EditorView, state: { selection: Selection } }
type EditorToolbarViewContext = { editor: Editor, view: EditorView }

const codeLanguageOptions = [
  { label: '自动识别', value: '__auto__' },
  { label: 'Plain Text', value: 'text' },
  { label: 'JavaScript', value: 'js' },
  { label: 'TypeScript', value: 'ts' },
  { label: 'Vue', value: 'vue' },
  { label: 'Go', value: 'go' },
  { label: 'Python', value: 'py' },
  { label: 'Bash', value: 'bash' },
  { label: 'JSON', value: 'json' },
  { label: 'YAML', value: 'yaml' },
  { label: 'HTML', value: 'html' },
  { label: 'CSS', value: 'css' },
  { label: 'SQL', value: 'sql' }
]

const codeLanguageToolbarItems = [[{
  slot: 'codeLanguage' as const
}]]

const { extension: Completion, handlers: aiHandlers, isLoading: aiLoading, aiReview } = useEditorCompletion(editorRef)

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

if (collaborationEnabled) {
  appConfig.ui.colors.primary = user.value.color
}

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

const selectedTags = computed(() => parseTags(postMeta.tagsText))

const isSaving = ref(false)
const deletingPost = ref(false)
const generatingSummary = ref(false)
const lastSavedAt = ref('')
const slugTouched = ref(false)

const canSubmit = computed(() => {
  return Boolean(postMeta.title.trim() && content.value.trim())
})

const canPublish = computed(() => publishChecks.value.every(item => item.ready))

const categoryOptions = computed(() =>
  categories.value.map(category => ({
    id: category.id,
    label: category.name
  }))
)

const selectedCategoryName = computed(() => {
  if (!postMeta.categoryId) return '未选择分类'
  return categories.value.find(category => category.id === postMeta.categoryId)?.name || `分类 ${postMeta.categoryId}`
})

const selectedTagNames = computed<string[]>({
  get: () => selectedTags.value,
  set: (value) => {
    postMeta.tagsText = Array.from(new Set(value.map(tag => tag.trim()).filter(Boolean))).join(', ')
  }
})

const wordCount = computed(() => {
  return content.value
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*`_~[\]()!-]/g, ' ')
    .replace(/\s+/g, '')
    .length
})

const estimatedReadingMinutes = computed(() => Math.max(1, Math.ceil(wordCount.value / 450)))

const publishChecks = computed(() => [
  { label: '标题', ready: Boolean(postMeta.title.trim()) },
  { label: '正文', ready: Boolean(content.value.trim()) },
  { label: 'Slug', ready: Boolean((postMeta.slug || toSlug(postMeta.title)).trim()) },
  { label: '分类', ready: Boolean(postMeta.categoryId.trim()) },
  { label: '摘要', ready: Boolean((postMeta.summary || plainTextSummary(content.value)).trim()) }
])

const readyCheckCount = computed(() => publishChecks.value.filter(item => item.ready).length)
const publishReadiness = computed(() => `${readyCheckCount.value}/${publishChecks.value.length}`)

const content = ref('')

function onCreate({ editor }: { editor: Editor }) {
  if (!collaborationEnabled) return

  const storageKey = `editor-initialized-${room.value}`

  if (sessionStorage.getItem(storageKey)) return

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

function isDesktopEditorViewport() {
  return globalThis.window?.matchMedia('(min-width: 768px)').matches ?? true
}

const mobileToolbarItems = computed(() =>
  bubbleToolbarItems.value.map(group =>
    group.map((item) => {
      if ('label' in item && (item.label === 'Improve' || item.label === 'Turn into')) {
        return {
          ...item,
          label: undefined
        }
      }

      return item
    })
  )
)

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
    .replace(/[#>*`_~[\]()!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  return text.slice(0, 140)
}

function inferCodeLanguage(source: string) {
  const code = source.trim()
  if (!code) return 'text'

  if (code.startsWith('{') || code.startsWith('[')) {
    try {
      JSON.parse(code)
      return 'json'
    } catch {
      // Continue with heuristic checks below.
    }
  }

  if (/<template[\s>][\s\S]*<\/template>|<script\s+setup|defineProps\(/i.test(code)) return 'vue'
  if (/^\s*package\s+\w+/m.test(code) || /\bfunc\s+\w+\s*\(/.test(code) || /\bfmt\.\w+\(/.test(code) || /:=/.test(code)) return 'go'
  if (/^\s*def\s+\w+\s*\(|^\s*class\s+\w+.*:|^\s*from\s+\w+\s+import\s+|^\s*import\s+\w+|print\(/m.test(code)) return 'py'
  if (/^\s*(select|insert\s+into|update|delete\s+from|create\s+table)\b/i.test(code)) return 'sql'
  if (/^#!\/.*\b(sh|bash)\b|^\s*(cd|curl|docker|git|go|npm|pnpm|bun|yarn|export)\b/m.test(code)) return 'bash'
  if (/<\/?[a-z][\s\S]*>/i.test(code)) return 'html'
  if (/^\s*[.#]?[\w-]+\s*\{[\s\S]*\}|^\s*(@media|:root)\b|^\s*[\w-]+\s*:\s*[^;]+;/m.test(code)) return 'css'
  if (/\binterface\s+\w+|^\s*type\s+\w+\s*=|:\s*(string|number|boolean|unknown|Record<)|\bimport\s+type\b/m.test(code)) return 'ts'
  if (/\b(function|const|let|var)\s+|=>|console\.log|^\s*import\s+.+\s+from\s+/m.test(code)) return 'js'
  if (/^[\w.-]+:\s+.+$/m.test(code)) return 'yaml'

  return 'text'
}

function getActiveCodeBlockText(editor: Editor) {
  const { $from } = editor.state.selection
  for (let depth = $from.depth; depth > 0; depth -= 1) {
    const node = $from.node(depth)
    if (node.type.name === 'codeBlock') {
      return node.textContent
    }
  }
  return ''
}

function getActiveCodeLanguage(editor: Editor) {
  return String(editor.getAttributes('codeBlock').language || '__auto__')
}

function setActiveCodeLanguage(editor: Editor, value: string) {
  const language = value === '__auto__'
    ? inferCodeLanguage(getActiveCodeBlockText(editor))
    : value

  editor.chain().focus().updateAttributes('codeBlock', { language }).run()
}

function getCodeLanguageLabel(editor: Editor) {
  const language = getActiveCodeLanguage(editor)
  return codeLanguageOptions.find(item => item.value === language)?.label || language
}

function getCodeLanguageMenuItems(editor: Editor) {
  return [codeLanguageOptions.map(item => ({
    label: item.label,
    icon: item.value === getActiveCodeLanguage(editor) ? 'i-lucide-check' : 'i-lucide-code',
    onSelect: () => setActiveCodeLanguage(editor, item.value)
  }))]
}

function normalizeCodeFenceLanguages(markdown: string) {
  return markdown.replace(/(^|\n)(```|~~~)([^\n`]*)\n([\s\S]*?)\n\2/g, (match, prefix: string, fence: string, rawInfo: string, code: string) => {
    const info = rawInfo.trim()
    if (info) return match

    const language = inferCodeLanguage(code)
    return `${prefix}${fence}${language}\n${code}\n${fence}`
  })
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

function createTagFromInput(tag: string) {
  const normalized = tag.trim().replace(/^#/, '')
  if (!normalized) return

  addTag(normalized)
}

function removeTag(tag: string) {
  const merged = selectedTags.value.filter(item => item !== tag)
  postMeta.tagsText = merged.join(', ')
}

function getErrorMessage(error: unknown) {
  if (error && typeof error === 'object') {
    const data = 'data' in error ? (error as { data?: { message?: unknown } }).data : undefined
    if (typeof data?.message === 'string') return data.message
    if ('message' in error && typeof (error as { message?: unknown }).message === 'string') return (error as { message: string }).message
  }

  return '请稍后重试'
}

function withCsrfHeaders(headers?: Record<string, string>) {
  const token = unref(csrf)
  const name = unref(headerName)

  if (!token || !name) {
    return headers
  }

  return {
    ...(headers || {}),
    [name]: token
  }
}

async function fillSummaryFromContent() {
  if (generatingSummary.value) return

  const source = content.value.trim()
  if (!source) {
    toast.add({
      title: '无法生成摘要',
      description: '请先填写正文内容',
      color: 'warning'
    })
    return
  }

  generatingSummary.value = true

  try {
    const response = await $fetch<{ summary?: string }>('/api/blog/summary', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: {
        title: postMeta.title.trim(),
        content: source
      }
    })
    const summary = response.summary?.trim()

    if (!summary) {
      throw new Error('未生成有效摘要')
    }

    postMeta.summary = summary
    toast.add({
      title: '摘要已生成',
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: '摘要生成失败',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    generatingSummary.value = false
  }
}

function formatLastSavedAt() {
  if (!lastSavedAt.value) return ''

  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(lastSavedAt.value))
}

function normalizeSlug() {
  postMeta.slug = toSlug(postMeta.slug)
}

function generateSlugFromTitle() {
  postMeta.slug = toSlug(postMeta.title)
  slugTouched.value = false
}

function markSlugTouched() {
  slugTouched.value = true
}

function fillSlugIfNeeded() {
  if (postMeta.slug.trim() || slugTouched.value) return
  generateSlugFromTitle()
}

function handleTitleBlur() {
  fillSlugIfNeeded()
}

async function savePost(status: number) {
  if (!canSubmit.value || isSaving.value) return

  fillSlugIfNeeded()

  if (status === POST_STATUS_PUBLISHED && !canPublish.value) {
    const missing = publishChecks.value
      .filter(item => !item.ready)
      .map(item => item.label)
      .join('、')

    toast.add({
      title: '发布信息不完整',
      description: `请先补全：${missing}`,
      color: 'warning'
    })
    return
  }

  isSaving.value = true

  try {
    const markdownContent = normalizeCodeFenceLanguages(content.value)
    const payload = {
      title: postMeta.title.trim(),
      summary: (postMeta.summary || plainTextSummary(markdownContent)).trim(),
      slug: postMeta.slug.trim(),
      content: markdownContent,
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
    postMeta.categoryId = saved.categoryId || postMeta.categoryId
    lastSavedAt.value = new Date().toISOString()

    toast.add({
      title: status === POST_STATUS_PUBLISHED ? '发布成功' : status === POST_STATUS_PRIVATE ? '已保存为私密' : '草稿已保存',
      description: `文章标识：${saved.id}`,
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: '保存失败',
      description: getErrorMessage(error),
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

function savePrivate() {
  return savePost(POST_STATUS_PRIVATE)
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

const editingSlug = computed(() => typeof route.query.slug === 'string' ? route.query.slug : '')

if (editingSlug.value) {
  const existingPost = await fetchPostBySlug(editingSlug.value)

  if (existingPost) {
    postMeta.id = existingPost.id
    postMeta.title = existingPost.title
    postMeta.summary = existingPost.summary
    postMeta.slug = existingPost.slug
    slugTouched.value = true
    postMeta.tagsText = existingPost.tags.join(', ')
    postMeta.categoryId = existingPost.categoryId
    content.value = existingPost.content || content.value
  }
}

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
  <UEditor
    v-if="collaborationReady"
    ref="editorRef"
    v-slot="{ editor, handlers }"
    :model-value="collaborationEnabled ? undefined : content"
    content-type="markdown"
    :extensions="extensions"
    :starter-kit="collaborationEnabled ? { undoRedo: false } : undefined"
    :handlers="customHandlers"
    autofocus
    placeholder="Write, type '/' for commands..."
    class="min-h-screen"
    :ui="{
      base: 'p-4 pt-20 sm:p-14',
      content: 'max-w-4xl mx-auto'
    }"
    @update:model-value="onUpdate"
    @create="onCreate"
  >
    <AppHeader compact>
      <div class="flex min-w-0 items-center gap-1 sm:gap-2">
        <UEditorToolbar
          :editor="editor"
          :items="toolbarItems"
        />

        <USeparator
          orientation="vertical"
          class="h-7 max-[380px]:hidden"
        />

        <UButton
          icon="i-lucide-save"
          color="neutral"
          variant="soft"
          size="sm"
          :disabled="!canSubmit || isSaving"
          :loading="isSaving"
          label="草稿"
          aria-label="保存草稿"
          :ui="{ label: 'max-sm:sr-only' }"
          @click="saveDraft"
        />

        <UButton
          icon="i-lucide-send"
          size="sm"
          :disabled="!canSubmit || !canPublish || isSaving"
          :loading="isSaving"
          label="发布"
          aria-label="发布"
          :ui="{ label: 'max-sm:sr-only' }"
          @click="publishNow"
        />

        <UButton
          icon="i-lucide-lock"
          color="primary"
          variant="soft"
          size="sm"
          :disabled="!canSubmit || isSaving"
          :loading="isSaving"
          label="私密"
          aria-label="保存为私密"
          :ui="{ label: 'max-sm:sr-only' }"
          @click="savePrivate"
        />

        <UButton
          v-if="postMeta.id"
          icon="i-lucide-trash-2"
          size="sm"
          color="error"
          variant="ghost"
          :loading="deletingPost"
          label="删除"
          aria-label="删除"
          :ui="{ label: 'max-sm:sr-only' }"
          @click="removeCurrentPost"
        />
      </div>

      <div class="hidden items-center gap-2 sm:flex">
        <EditorCollaborationUsers :users="connectedUsers" />
      </div>

      <template #search>
        <div class="mx-auto max-w-xl overflow-x-auto">
          <UEditorToolbar
            :editor="editor"
            :items="mobileToolbarItems"
            layout="fixed"
            class="w-full min-w-max justify-between"
            :ui="{
              base: 'w-full justify-between',
              group: 'flex-1 justify-center',
              separator: 'shrink-0'
            }"
          >
            <template #link>
              <EditorLinkPopover :editor="editor" />
            </template>
          </UEditorToolbar>
        </div>
      </template>
    </AppHeader>

    <div
      v-if="aiReview.isOpen.value"
      class="fixed bottom-6 right-6 z-50 w-[min(520px,calc(100vw-2rem))] rounded-xl border border-default bg-default/95 p-4 shadow-xl backdrop-blur"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <UIcon
            name="i-lucide-sparkles"
            class="text-primary"
          />
          <p class="text-sm font-medium text-highlighted">
            AI candidate
          </p>
          <UBadge
            v-if="aiLoading"
            label="Streaming"
            color="primary"
            variant="soft"
          />
        </div>

        <div class="flex items-center gap-1">
          <UButton
            v-if="aiLoading"
            size="xs"
            color="error"
            variant="soft"
            icon="i-lucide-square"
            label="Stop"
            @click="aiReview.stop"
          />
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-refresh-cw"
            label="Reroll"
            :disabled="aiLoading"
            @click="aiReview.reroll"
          />
          <UButton
            size="xs"
            icon="i-lucide-check"
            label="Accept"
            :disabled="!aiReview.previewText.value.trim()"
            @click="aiReview.accept"
          />
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            @click="aiReview.discard"
          />
        </div>
      </div>

      <div
        class="mt-3 max-h-56 overflow-auto rounded-lg border border-default bg-elevated/50 p-3 text-sm leading-6 whitespace-pre-wrap text-toned"
      >
        {{ aiReview.previewText.value || 'Waiting for AI output...' }}
      </div>

      <div
        v-if="aiReview.candidates.value.length > 0"
        class="mt-3 flex flex-wrap gap-2"
      >
        <UButton
          v-for="(candidate, index) in aiReview.candidates.value"
          :key="candidate.id"
          size="xs"
          :color="index === aiReview.activeCandidateIndex.value ? 'primary' : 'neutral'"
          :variant="index === aiReview.activeCandidateIndex.value ? 'soft' : 'ghost'"
          :label="`${index + 1}${candidate.stopped ? ' partial' : ''}`"
          @click="aiReview.select(index)"
        />
      </div>
    </div>

    <section class="mx-auto mb-6 mt-16 w-full max-w-5xl md:mt-0">
      <UCard>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-sm font-medium text-highlighted">
                文章信息
              </p>
              <p class="mt-1 text-xs text-toned">
                发布前确认标题、slug、分类、标签与摘要，减少返工。
              </p>
            </div>

            <p class="text-xs text-toned">
              {{ wordCount }} 字 · 约 {{ estimatedReadingMinutes }} 分钟
              <span v-if="lastSavedAt"> · 已保存 {{ formatLastSavedAt() }}</span>
            </p>
          </div>
        </template>

        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_260px]">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField
              label="标题"
              name="title"
              required
            >
              <UInput
                v-model="postMeta.title"
                placeholder="请输入文章标题"
                @blur="handleTitleBlur"
              />
            </UFormField>

            <UFormField
              label="Slug"
              name="slug"
              required
            >
              <div class="flex gap-2">
                <UInput
                  v-model="postMeta.slug"
                  placeholder="article-slug"
                  class="flex-1"
                  @input="markSlugTouched"
                  @blur="normalizeSlug"
                />
                <UButton
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-refresh-cw"
                  label="生成"
                  @click="generateSlugFromTitle"
                />
              </div>
            </UFormField>

            <UFormField
              label="分类"
              name="categoryId"
              required
            >
              <USelectMenu
                v-model="postMeta.categoryId"
                :items="categoryOptions"
                value-key="id"
                label-key="label"
                placeholder="搜索并选择分类"
                icon="i-lucide-folder"
                :search-input="{ placeholder: '搜索分类名称' }"
                clear
              >
                <template #item-label="{ item }">
                  <span>{{ item.label }}</span>
                </template>
              </USelectMenu>
              <p class="mt-2 text-xs text-toned">
                当前：{{ selectedCategoryName }}
              </p>
            </UFormField>

            <UFormField
              label="标签"
              name="tagsText"
            >
              <USelectMenu
                v-model="selectedTagNames"
                :items="tagSuggestions"
                multiple
                create-item
                placeholder="搜索或创建标签"
                icon="i-lucide-tags"
                :search-input="{ placeholder: '输入标签名，回车创建' }"
                @create="createTagFromInput"
              >
                <template #create-item-label="{ item }">
                  创建标签 #{{ item }}
                </template>
              </USelectMenu>
              <div class="mt-2 flex flex-wrap gap-2">
                <UBadge
                  v-for="tag in selectedTags"
                  :key="`selected-${tag}`"
                  color="primary"
                  variant="subtle"
                  class="cursor-pointer"
                  :label="`#${tag} ×`"
                  @click="removeTag(tag)"
                />
              </div>
            </UFormField>

            <UFormField
              label="摘要"
              name="summary"
              class="md:col-span-2"
            >
              <div class="space-y-2">
                <UTextarea
                  v-model="postMeta.summary"
                  :rows="3"
                  autoresize
                  placeholder="不填写时将自动从正文提取前 140 字"
                />
                <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-toned">
                  <span>{{ (postMeta.summary || plainTextSummary(content)).length }}/140</span>
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-wand-sparkles"
                    label="从正文生成摘要"
                    :loading="generatingSummary"
                    :disabled="generatingSummary || !content.trim()"
                    @click="fillSummaryFromContent"
                  />
                </div>
              </div>
            </UFormField>
          </div>

          <aside class="rounded-2xl border border-default bg-elevated/40 p-4">
            <div class="flex items-center justify-between gap-3">
              <p class="text-sm font-medium text-highlighted">
                发布检查
              </p>
              <UBadge
                :label="publishReadiness"
                color="neutral"
                variant="soft"
              />
            </div>

            <div class="mt-4 space-y-3">
              <div
                v-for="item in publishChecks"
                :key="item.label"
                class="flex items-center gap-2 text-sm"
                :class="item.ready ? 'text-highlighted' : 'text-toned'"
              >
                <UIcon
                  :name="item.ready ? 'i-lucide-check-circle-2' : 'i-lucide-circle'"
                  :class="item.ready ? 'text-primary' : 'text-muted'"
                />
                <span>{{ item.label }}</span>
              </div>
            </div>

            <UAlert
              class="mt-4"
              color="neutral"
              variant="soft"
              icon="i-lucide-lightbulb"
              title="小提示"
              description="发布前先保存草稿，确认详情页预览无误后再公开。"
            />
          </aside>
        </div>
      </UCard>
    </section>

    <UEditorToolbar
      :editor="editor"
      :items="bubbleToolbarItems"
      layout="bubble"
      :should-show="({ editor, view, state }: EditorToolbarContext) => {
        if (!isDesktopEditorViewport()) {
          return false
        }
        if (editor.isActive('imageUpload') || editor.isActive('image') || state.selection instanceof CellSelection) {
          return false
        }
        const { selection } = state
        return view.hasFocus() && !selection.empty
      }"
    >
      <template #link>
        <EditorLinkPopover :editor="editor" />
      </template>
    </UEditorToolbar>

    <UEditorToolbar
      :editor="editor"
      :items="getImageToolbarItems(editor)"
      layout="bubble"
      :should-show="({ editor, view }: EditorToolbarViewContext) => {
        if (!isDesktopEditorViewport()) {
          return false
        }
        return editor.isActive('image') && view.hasFocus()
      }"
    />

    <UEditorToolbar
      :editor="editor"
      :items="getTableToolbarItems(editor)"
      layout="bubble"
      :should-show="({ editor, view }: EditorToolbarViewContext) => {
        if (!isDesktopEditorViewport()) {
          return false
        }
        return editor.state.selection instanceof CellSelection && view.hasFocus()
      }"
    />

    <UEditorToolbar
      :editor="editor"
      :items="codeLanguageToolbarItems"
      layout="bubble"
      :should-show="({ editor, view }: EditorToolbarViewContext) => {
        if (!isDesktopEditorViewport()) {
          return false
        }
        return editor.isActive('codeBlock') && view.hasFocus()
      }"
    >
      <template #codeLanguage>
        <div class="flex items-center gap-2 px-1">
          <UDropdownMenu
            :items="getCodeLanguageMenuItems(editor)"
            :content="{ align: 'start', side: 'bottom' }"
          >
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              icon="i-lucide-square-code"
              trailing-icon="i-lucide-chevron-down"
              :label="getCodeLanguageLabel(editor)"
            />
          </UDropdownMenu>
        </div>
      </template>
    </UEditorToolbar>

    <UEditorEmojiMenu
      :editor="editor"
      :items="emojiItems"
    />

    <UEditorMentionMenu
      :editor="editor"
      :items="mentionItems"
    />

    <UEditorSuggestionMenu
      :editor="editor"
      :items="suggestionItems"
    />

    <UEditorDragHandle
      v-slot="{ ui, onClick }"
      :editor="editor"
      @node-change="onNodeChange"
    >
      <UButton
        icon="i-lucide-plus"
        color="neutral"
        variant="ghost"
        size="sm"
        :class="ui.handle()"
        @click="(e: MouseEvent) => {
          e.stopPropagation()
          const node = onClick()

          handlers.suggestion?.execute(editor, { pos: node?.pos }).run()
        }"
      />

      <UDropdownMenu
        v-slot="{ open }"
        :modal="false"
        :items="getDragHandleItems(editor)"
        :content="{ side: 'left' }"
        :ui="{ content: 'w-48', label: 'text-xs' }"
        @update:open="editor.chain().setMeta('lockDragHandle', $event).run()"
      >
        <UButton
          color="neutral"
          variant="ghost"
          active-variant="soft"
          size="sm"
          icon="i-lucide-grip-vertical"
          :active="open"
          :class="ui.handle()"
        />
      </UDropdownMenu>
    </UEditorDragHandle>
  </UEditor>
</template>
