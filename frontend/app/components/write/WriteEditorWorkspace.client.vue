<script setup lang="ts">
import type { EditorCustomHandlers } from '@nuxt/ui'
import type { Editor } from '@tiptap/core'
import type { Selection } from '@tiptap/pm/state'
import { TextSelection } from '@tiptap/pm/state'
import type { EditorView } from '@tiptap/pm/view'
import { TaskList, TaskItem } from '@tiptap/extension-list'
import { TableKit } from '@tiptap/extension-table'
import { CellSelection } from '@tiptap/pm/tables'
import { CodeBlockShiki } from 'tiptap-extension-code-block-shiki'
import { ImageUpload } from '~/components/editor/ImageUploadExtension'
import { createPost, deletePost, fetchManagePostBySlug, POST_STATUS_DRAFT, POST_STATUS_PRIVATE, POST_STATUS_PUBLISHED, updatePost } from '~/composables/useBlogApi'
import { normalizeLooseMarkdownTables } from '~/utils/markdown'

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
const coverUploadRef = useTemplateRef('coverUploadRef')

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
  codeBlock: {
    canExecute: (editor: Editor) => Boolean(editor.state.schema.nodes.codeBlock),
    execute: (editor: Editor) => {
      if (editor.isActive('codeBlock')) {
        return editor.chain().focus().setParagraph()
      }

      return editor.chain().focus().command(({ state, tr, dispatch }) => {
        const codeBlockType = state.schema.nodes.codeBlock
        if (!codeBlockType) return false

        const { from, to, empty } = state.selection
        const selectedText = empty ? '' : state.doc.textBetween(from, to, '\n')
        const code = selectedText.replace(/\n$/, '')
        const language = code.trim() ? inferCodeLanguage(code) : 'text'
        const contentNode = code ? state.schema.text(code) : undefined
        const node = codeBlockType.create({ language }, contentNode)

        tr.replaceRangeWith(from, to, node)
        tr.setSelection(TextSelection.near(tr.doc.resolve(Math.min(from + 1, tr.doc.content.size))))

        dispatch?.(tr.scrollIntoView())
        return true
      })
    },
    isActive: (editor: Editor) => editor.isActive('codeBlock'),
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
  coverUrl: '',
  tagsText: '',
  categoryId: ''
})

const selectedTags = computed(() => parseTags(postMeta.tagsText))

const isSaving = ref(false)
const deletingPost = ref(false)
const uploadingCover = ref(false)
const lastSavedAt = ref('')
const slugTouched = ref(false)
const settingsPanelOpen = ref(false)
const upload = useUpload('/api/upload', {
  formKey: 'file',
  multiple: false,
  headers: { [headerName]: csrf }
})

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

const effectiveCoverUrl = computed(() => {
  const customCover = postMeta.coverUrl.trim()
  if (customCover) return customCover
  return `/covers/${encodeURIComponent((postMeta.slug || toSlug(postMeta.title) || postMeta.id || 'gopalette').trim() || 'gopalette')}.svg`
})

const publishStatusText = computed(() => {
  if (isSaving.value) return '正在保存...'
  if (lastSavedAt.value) return `草稿已保存 ${formatLastSavedAt()}`
  return '尚未保存'
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
  { label: '分类', ready: Boolean(postMeta.categoryId.trim()) }
])

const editorStarterKit = computed(() => ({
  codeBlock: false as const,
  ...(collaborationEnabled ? { undoRedo: false as const } : {})
}))
const fixedToolbarItems = computed(() => [
  ...toolbarItems.value,
  ...bubbleToolbarItems.value
])

const content = ref('')
let removeSettingsPanelListener: (() => void) | undefined

onMounted(() => {
  const media = window.matchMedia('(min-width: 1280px)')
  const syncSettingsPanel = () => {
    settingsPanelOpen.value = media.matches
  }

  syncSettingsPanel()
  media.addEventListener('change', syncSettingsPanel)
  removeSettingsPanelListener = () => media.removeEventListener('change', syncSettingsPanel)
})

onBeforeUnmount(() => {
  removeSettingsPanelListener?.()
})

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

function isFixedEditorToolbarViewport() {
  return true
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
    .replace(/[#>*`_~[\]()!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  return text.slice(0, 140)
}

function normalizeHeadingText(value: string) {
  return value
    .replace(/[#>*`_~[\]()!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

const outlineItems = computed(() => {
  const headings: { text: string, depth: number }[] = []
  let inCodeFence = false

  for (const line of content.value.split('\n')) {
    const trimmed = line.trim()
    if (/^```/.test(trimmed) || /^~~~/.test(trimmed)) {
      inCodeFence = !inCodeFence
      continue
    }

    if (inCodeFence) continue

    const match = /^(#{1,3})\s+(.+?)\s*#*$/.exec(trimmed)
    if (!match) continue

    const text = normalizeHeadingText(match[2] || '')
    if (text) {
      headings.push({
        text,
        depth: match[1]?.length || 1
      })
    }
  }

  return headings.slice(0, 12)
})

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

async function generateSummaryFromContent(markdownContent: string) {
  const source = markdownContent.trim()
  if (!source) return ''
  try {
    const response = await $fetch<{ summary?: string }>('/api/blog/summary', {
      method: 'POST',
      headers: withCsrfHeaders(),
      body: {
        title: postMeta.title.trim(),
        content: source
      }
    })
    return response.summary?.trim() || ''
  } catch {
    return ''
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

async function onCoverChange() {
  const target = coverUploadRef.value?.inputRef
  if (!target) return

  uploadingCover.value = true
  try {
    const result = await upload(target)
    postMeta.coverUrl = String(result.url || `/images/${result.pathname}` || '')
    toast.add({
      title: '封面图已上传',
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: '封面图上传失败',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    uploadingCover.value = false
  }
}

function clearCover() {
  postMeta.coverUrl = ''
}

function goBack() {
  if (import.meta.client && window.history.length > 1) {
    window.history.back()
  } else {
    navigateTo('/admin')
  }
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
    const markdownContent = normalizeCodeFenceLanguages(normalizeLooseMarkdownTables(content.value))
    const generatedSummary = await generateSummaryFromContent(markdownContent)
    const payload = {
      title: postMeta.title.trim(),
      summary: (generatedSummary || plainTextSummary(markdownContent)).trim(),
      slug: postMeta.slug.trim(),
      content: markdownContent,
      coverUrl: postMeta.coverUrl.trim() || undefined,
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
    postMeta.coverUrl = saved.coverUrl || ''
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
  const existingPost = await fetchManagePostBySlug(editingSlug.value)

  if (existingPost) {
    postMeta.id = existingPost.id
    postMeta.title = existingPost.title
    postMeta.summary = existingPost.summary
    postMeta.slug = existingPost.slug
    postMeta.coverUrl = existingPost.coverUrl || ''
    slugTouched.value = true
    postMeta.tagsText = existingPost.tags.join(', ')
    postMeta.categoryId = existingPost.categoryId
    content.value = normalizeLooseMarkdownTables(existingPost.content || content.value)
  }
}

const MarkdownCodeBlockShiki = CodeBlockShiki.extend({
  markdownTokenName: 'code',
  parseMarkdown: (token, helpers) => {
    const typedToken = token as {
      lang?: string
      text?: string
    }

    const text = (typedToken.text || '').replace(/\n+$/, '')
    return helpers.createNode(
      'codeBlock',
      { language: typedToken.lang || null },
      text ? [helpers.createTextNode(text)] : []
    )
  },
  renderMarkdown: (node, helpers) => {
    const typedNode = node as { attrs?: { language?: string }, content?: unknown }
    const language = typedNode.attrs?.language || ''
    const content = typedNode.content
      ? helpers.renderChildren(typedNode.content).replace(/\n+$/, '')
      : ''

    return content
      ? [`\`\`\`${language}`, content, '```'].join('\n')
      : `\`\`\`${language}\n\`\`\``
  }
})

const extensions = computed(() => [
  MarkdownCodeBlockShiki.configure({
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
    :starter-kit="editorStarterKit"
    :handlers="customHandlers"
    autofocus
    placeholder="Write, type '/' for commands..."
    class="min-h-screen"
    :ui="{
      base: 'px-0 pb-16 pt-4 xl:pt-10',
      content: 'mx-auto w-full max-w-[920px] px-4 pb-32'
    }"
    @update:model-value="onUpdate"
    @create="onCreate"
  >
    <AppHeader compact>
      <div class="flex min-w-0 items-center gap-2">
        <UButton
          icon="i-lucide-arrow-left"
          color="neutral"
          variant="ghost"
          size="sm"
          label="返回"
          @click="goBack"
        />
        <UBadge
          color="neutral"
          variant="soft"
          size="sm"
          :label="publishStatusText"
        />
      </div>

      <div class="flex items-center gap-2">
        <UButton
          icon="i-lucide-save"
          color="neutral"
          variant="soft"
          size="sm"
          :disabled="!canSubmit || isSaving"
          :loading="isSaving"
          label="草稿"
          @click="saveDraft"
        />
        <UButton
          icon="i-lucide-send"
          size="sm"
          :disabled="!canSubmit || !canPublish || isSaving"
          :loading="isSaving"
          label="发布"
          @click="publishNow"
        />
      </div>
    </AppHeader>

    <div class="sticky top-[64px] z-40 border-b border-default bg-default/95 shadow-sm backdrop-blur">
      <div class="mx-auto flex h-12 max-w-[920px] items-center justify-start overflow-x-auto px-2 sm:px-4 xl:justify-center">
        <UEditorToolbar
          :editor="editor"
          :items="fixedToolbarItems"
          class="min-w-max"
        >
          <template #link>
            <EditorLinkPopover :editor="editor" />
          </template>
        </UEditorToolbar>
      </div>
    </div>

    <aside class="fixed left-6 top-32 z-30 hidden w-64 xl:block">
      <div class="min-h-72 rounded-2xl border border-default bg-default/85 p-4 shadow-sm backdrop-blur">
        <div class="flex items-center justify-between">
          <p class="text-sm font-medium text-highlighted">
            目录
          </p>
          <UIcon
            name="i-lucide-list-tree"
            class="size-4 text-muted"
          />
        </div>

        <nav
          v-if="outlineItems.length"
          class="mt-4 space-y-1"
        >
          <p
            v-for="(item, index) in outlineItems"
            :key="`${item.text}-${index}`"
            class="truncate rounded-md px-2 py-1.5 text-sm text-toned"
            :class="item.depth > 1 ? 'ml-3' : ''"
          >
            {{ item.text }}
          </p>
        </nav>

        <p
          v-else
          class="mt-28 text-center text-xs leading-5 text-muted"
        >
          对正文应用标题样式后生成目录
        </p>
      </div>
    </aside>

    <section class="mx-auto w-full max-w-[920px] px-3 pb-2 pt-6 sm:px-4 xl:pt-10">
      <UInput
        v-model="postMeta.title"
        placeholder="请输入标题（建议30字以内）"
        variant="none"
        size="xl"
        class="write-title-input"
        :ui="{ base: 'px-0 text-3xl font-semibold leading-tight text-highlighted placeholder:text-muted sm:text-4xl' }"
        @blur="handleTitleBlur"
      />
      <p class="mt-4 text-sm text-toned">
        {{ wordCount }} 字 · 约 {{ estimatedReadingMinutes }} 分钟
      </p>
    </section>

    <aside class="mx-auto mb-2 w-full max-w-[1180px] px-3 sm:px-4 xl:fixed xl:right-6 xl:top-32 xl:z-30 xl:mb-0 xl:w-80 xl:px-0">
      <details
        class="rounded-xl border border-default bg-default shadow-sm xl:contents"
        :open="settingsPanelOpen"
        @toggle="settingsPanelOpen = ($event.target as HTMLDetailsElement).open"
      >
        <summary class="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2.5 text-sm font-medium text-highlighted xl:hidden [&::-webkit-details-marker]:hidden">
          <span class="inline-flex items-center gap-2">
            <UIcon
              name="i-lucide-sliders-horizontal"
              class="size-4 text-toned"
            />
            文章设置
          </span>
          <span class="inline-flex items-center gap-2 text-xs font-normal text-toned">
            分类、标签、封面
            <UIcon
              name="i-lucide-chevron-down"
              class="size-4"
            />
          </span>
        </summary>

        <div class="space-y-3 border-t border-default p-2.5 xl:max-h-[calc(100vh-6rem)] xl:overflow-y-auto xl:rounded-2xl xl:border xl:bg-default xl:p-3 xl:shadow-sm">
        <section class="rounded-xl border border-default bg-elevated/40 p-3">
          <p class="text-sm font-medium text-highlighted">
            分类
          </p>
          <USelectMenu
            v-model="postMeta.categoryId"
            class="mt-2"
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
        </section>

        <section class="rounded-xl border border-default bg-elevated/40 p-3">
          <p class="text-sm font-medium text-highlighted">
            标签
          </p>
          <USelectMenu
            v-model="selectedTagNames"
            class="mt-2"
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
        </section>

        <section class="rounded-xl border border-default bg-elevated/40 p-3">
          <p class="text-sm font-medium text-highlighted">
            封面图
          </p>
          <img
            :src="effectiveCoverUrl"
            alt="封面图"
            class="mt-2 h-32 w-full rounded-lg border border-default object-cover"
          >
          <UFileUpload
            ref="coverUploadRef"
            class="mt-2"
            accept="image/*"
            :preview="false"
            label="上传封面图"
            :description="uploadingCover ? '上传中...' : '支持常见图片格式，未上传时自动使用占位图'"
            @update:model-value="onCoverChange"
          />
          <div class="mt-2 flex justify-end">
            <UButton
              size="xs"
              color="neutral"
              variant="ghost"
              label="恢复占位图"
              :disabled="!postMeta.coverUrl.trim()"
              @click="clearCover"
            />
          </div>
        </section>

        <section class="rounded-xl border border-default bg-elevated/40 p-3">
          <p class="text-sm font-medium text-highlighted">
            SEO 设置
          </p>
          <UFormField
            label="Slug"
            name="slug"
            required
            class="mt-2"
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
          <p class="mt-2 text-xs text-toned">
            文章路径：/posts/{{ (postMeta.slug || toSlug(postMeta.title) || 'your-slug').trim() || 'your-slug' }}
          </p>
        </section>

        <section class="rounded-xl border border-default bg-elevated/40 p-3">
          <p class="text-sm font-medium text-highlighted">
            高级设置
          </p>
          <div class="mt-2 flex flex-wrap gap-2">
            <UButton
              icon="i-lucide-lock"
              color="primary"
              variant="soft"
              size="sm"
              :disabled="!canSubmit || isSaving"
              :loading="isSaving"
              label="保存为私密"
              @click="savePrivate"
            />
            <UButton
              v-if="postMeta.id"
              icon="i-lucide-trash-2"
              size="sm"
              color="error"
              variant="ghost"
              :loading="deletingPost"
              label="删除文章"
              @click="removeCurrentPost"
            />
          </div>
        </section>
        </div>
      </details>
    </aside>

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

    <UEditorToolbar
      :editor="editor"
      :items="bubbleToolbarItems"
      layout="bubble"
      :should-show="({ editor, view, state }: EditorToolbarContext) => {
        if (!isDesktopEditorViewport()) {
          return false
        }
        if (isFixedEditorToolbarViewport()) {
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
