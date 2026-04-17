<script setup lang="ts">
import {
  POST_STATUS_ARCHIVED,
  POST_STATUS_DRAFT,
  POST_STATUS_PUBLISHED,
  createCategory,
  createTag,
  deleteCategory,
  deleteComment,
  deletePost,
  deleteTag,
  fetchCategories,
  fetchComments,
  fetchPosts,
  fetchTags,
  searchPosts,
  updateCategory,
  updatePost,
  updateTag
} from '~/composables/useBlogApi'
import type { BlogPostItem, CategoryItem, CommentInfo, TagItem } from '~/composables/useBlogApi'

const toast = useToast()
const {
  session,
  user,
  isLoggedIn,
  isAdmin,
  initAuth,
  fetchProfile
} = useAuth()

const checkingAuth = ref(true)

const postKeyword = ref('')
const postPage = ref(1)
const pageSize = 10
const postLoading = ref(false)
const postTotal = ref(0)
const postRows = ref<BlogPostItem[]>([])
const deletingPostId = ref('')
const updatingPostId = ref('')

const commentPostId = ref('')
const commentLoading = ref(false)
const commentRows = ref<CommentInfo[]>([])
const commentTotal = ref(0)
const deletingCommentId = ref('')

const categories = ref<CategoryItem[]>([])
const tags = ref<TagItem[]>([])
const categoryLoading = ref(false)
const tagLoading = ref(false)
const savingCategory = ref(false)
const savingTag = ref(false)
const updatingCategoryId = ref('')
const updatingTagId = ref('')
const deletingCategoryId = ref('')
const deletingTagId = ref('')

const newCategory = reactive({
  name: '',
  slug: '',
  description: ''
})

const newTag = reactive({
  name: '',
  slug: ''
})

const categoryRename = ref<Record<string, string>>({})
const tagRename = ref<Record<string, string>>({})

function getErrorMessage(error: unknown, fallback: string) {
  if (!error || typeof error !== 'object') return fallback
  const typed = error as { message?: unknown, data?: { message?: unknown } }
  const value = typed.data?.message ?? typed.message
  return typeof value === 'string' && value.trim() ? value : fallback
}

function askConfirm(message: string) {
  if (!import.meta.client) return true
  return window.confirm(message)
}

function postStatusText(status: number) {
  if (status === POST_STATUS_PUBLISHED) return '已发布'
  if (status === POST_STATUS_ARCHIVED) return '已归档'
  return '草稿'
}

function postStatusColor(status: number) {
  if (status === POST_STATUS_PUBLISHED) return 'success'
  if (status === POST_STATUS_ARCHIVED) return 'warning'
  return 'neutral'
}

function toCover(seed: string) {
  return `https://picsum.photos/seed/${encodeURIComponent(seed || 'gopalette')}/1200/640`
}

function renderHighlightedText(input: string) {
  const source = String(input || '')
  const marked = source
    .replace(/<em>/gi, '[[EM_OPEN]]')
    .replace(/<\/em>/gi, '[[EM_CLOSE]]')

  const escaped = marked
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

  return escaped
    .replace(/\[\[EM_OPEN\]\]/g, '<mark class="rounded bg-primary/20 px-0.5 text-highlighted">')
    .replace(/\[\[EM_CLOSE\]\]/g, '</mark>')
}

async function loadPosts() {
  postLoading.value = true
  try {
    const query = postKeyword.value.trim()
    if (query) {
      const response = await searchPosts(query, postPage.value, pageSize)
      postTotal.value = response.total
      postRows.value = response.results.map(item => ({
        id: item.id,
        title: item.title,
        summary: item.summary,
        slug: item.slug,
        status: POST_STATUS_PUBLISHED,
        tags: item.tags,
        category: item.categoryName,
        categoryId: '',
        author: '未知作者',
        publishedAt: item.createdAt || '未知时间',
        readingMinutes: Math.max(1, Math.ceil(item.summary.length / 300)),
        cover: toCover(item.slug || item.id)
      }))
      return
    }

    const response = await fetchPosts(postPage.value, pageSize)
    postTotal.value = response.total
    postRows.value = response.posts
  } catch (error: unknown) {
    toast.add({
      title: '加载文章失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    postLoading.value = false
  }
}

async function loadComments() {
  const postId = commentPostId.value.trim()
  if (!postId) {
    commentRows.value = []
    commentTotal.value = 0
    return
  }

  commentLoading.value = true
  try {
    const response = await fetchComments(postId, 1, 100)
    commentRows.value = response.comments
    commentTotal.value = response.total
  } catch (error: unknown) {
    toast.add({
      title: '加载评论失败',
      description: getErrorMessage(error, '请检查文章 ID 是否正确'),
      color: 'error'
    })
  } finally {
    commentLoading.value = false
  }
}

async function removeComment(id: string) {
  if (!id) return
  if (!askConfirm('确认删除这条评论吗？')) return

  deletingCommentId.value = id
  try {
    await deleteComment(id)
    toast.add({ title: '评论已删除', color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '删除评论失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    deletingCommentId.value = ''
  }
}

async function setPostStatus(item: BlogPostItem, status: number) {
  if (!item.id || updatingPostId.value) return
  updatingPostId.value = item.id
  try {
    await updatePost(item.id, {
      title: item.title,
      summary: item.summary,
      slug: item.slug,
      content: item.content || '',
      status,
      categoryId: item.categoryId || undefined,
      tags: item.tags,
      updateMask: 'status'
    })
    toast.add({ title: '文章状态已更新', color: 'success' })
    await loadPosts()
  } catch (error: unknown) {
    toast.add({
      title: '更新文章状态失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    updatingPostId.value = ''
  }
}

async function removePost(id: string) {
  if (!id || deletingPostId.value) return
  if (!askConfirm('确认删除这篇文章吗？此操作不可恢复。')) return

  deletingPostId.value = id
  try {
    await deletePost(id)
    toast.add({ title: '文章已删除', color: 'success' })
    await loadPosts()
  } catch (error: unknown) {
    toast.add({
      title: '删除文章失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    deletingPostId.value = ''
  }
}

async function loadTaxonomy() {
  categoryLoading.value = true
  tagLoading.value = true
  try {
    const [categoryResponse, tagResponse] = await Promise.all([
      fetchCategories(1, 200),
      fetchTags(1, 200)
    ])
    categories.value = categoryResponse.categories
    tags.value = tagResponse.tags
    categoryRename.value = Object.fromEntries(categories.value.map(item => [item.id, item.name]))
    tagRename.value = Object.fromEntries(tags.value.map(item => [item.id, item.name]))
  } finally {
    categoryLoading.value = false
    tagLoading.value = false
  }
}

async function createCategoryEntry() {
  const name = newCategory.name.trim()
  if (!name || savingCategory.value) return

  savingCategory.value = true
  try {
    await createCategory({
      name,
      slug: newCategory.slug.trim() || undefined,
      description: newCategory.description.trim() || undefined
    })
    newCategory.name = ''
    newCategory.slug = ''
    newCategory.description = ''
    toast.add({ title: '分类创建成功', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '创建分类失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    savingCategory.value = false
  }
}

async function renameCategory(id: string) {
  const name = (categoryRename.value[id] || '').trim()
  if (!id || !name || updatingCategoryId.value) return

  updatingCategoryId.value = id
  try {
    await updateCategory(id, { name, updateMask: 'name' })
    toast.add({ title: '分类已更新', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '更新分类失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    updatingCategoryId.value = ''
  }
}

async function removeCategory(id: string) {
  if (!id || deletingCategoryId.value) return
  if (!askConfirm('确认删除该分类吗？')) return

  deletingCategoryId.value = id
  try {
    await deleteCategory(id)
    toast.add({ title: '分类已删除', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '删除分类失败',
      description: getErrorMessage(error, '请先清空该分类下文章后重试'),
      color: 'error'
    })
  } finally {
    deletingCategoryId.value = ''
  }
}

async function createTagEntry() {
  const name = newTag.name.trim()
  if (!name || savingTag.value) return

  savingTag.value = true
  try {
    await createTag({
      name,
      slug: newTag.slug.trim() || undefined
    })
    newTag.name = ''
    newTag.slug = ''
    toast.add({ title: '标签创建成功', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '创建标签失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    savingTag.value = false
  }
}

async function renameTag(id: string) {
  const name = (tagRename.value[id] || '').trim()
  if (!id || !name || updatingTagId.value) return

  updatingTagId.value = id
  try {
    await updateTag(id, { name, updateMask: 'name' })
    toast.add({ title: '标签已更新', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '更新标签失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    updatingTagId.value = ''
  }
}

async function removeTag(id: string) {
  if (!id || deletingTagId.value) return
  if (!askConfirm('确认删除该标签吗？')) return

  deletingTagId.value = id
  try {
    await deleteTag(id)
    toast.add({ title: '标签已删除', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '删除标签失败',
      description: getErrorMessage(error, '请先取消文章与该标签关联后重试'),
      color: 'error'
    })
  } finally {
    deletingTagId.value = ''
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(postTotal.value / pageSize)))

watch(postPage, () => {
  void loadPosts()
})

onMounted(async () => {
  initAuth()

  if (!isLoggedIn.value) {
    await navigateTo('/login?redirect=/admin')
    return
  }

  if (!user.value && session.value.userId) {
    await fetchProfile()
  }

  if (!isAdmin.value) {
    toast.add({
      title: '无权限访问后台',
      description: '当前账号不是管理员',
      color: 'error'
    })
    await navigateTo('/profile')
    return
  }

  checkingAuth.value = false

  await Promise.all([
    loadPosts(),
    loadTaxonomy()
  ])
})

useSeoMeta({
  title: '后台管理',
  description: 'GoPalette 管理后台：文章、评论、分类、标签一体化管理。'
})
</script>

<template>
  <div class="min-h-screen bg-default">
    <AppHeader />

    <main class="mx-auto w-full max-w-7xl px-4 pb-20 pt-10 sm:px-14">
      <div v-if="checkingAuth" class="rounded-2xl border border-default bg-default p-8 text-sm text-toned">
        正在校验管理员身份...
      </div>

      <template v-else>
        <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
              后台管理
            </h1>
            <p class="mt-1 text-sm text-toned">
              文章发布流转、评论审核、分类标签管理
            </p>
          </div>
          <UButton to="/write" icon="i-lucide-pen-line" label="写新文章" />
        </div>

        <section class="rounded-2xl border border-default bg-default p-5 sm:p-6">
          <div class="flex flex-wrap items-end gap-3">
            <UFormField label="文章关键词" class="min-w-56 flex-1">
              <UInput v-model="postKeyword" placeholder="按标题检索" icon="i-lucide-search" />
            </UFormField>

            <UButton :loading="postLoading" label="查询文章" icon="i-lucide-filter" @click="loadPosts" />
          </div>

          <div class="mt-5 space-y-3">
            <article v-for="item in postRows" :key="item.id" class="rounded-xl border border-default p-4">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <p class="text-base font-semibold text-highlighted" v-html="renderHighlightedText(item.title)" />
                    <UBadge size="xs" :label="postStatusText(item.status)" :color="postStatusColor(item.status)" variant="subtle" />
                  </div>
                  <p class="text-xs text-toned">
                    ID: {{ item.id }} · 分类: {{ item.category }} · 标签: {{ item.tags.join(', ') || '-' }}
                  </p>
                </div>

                <div class="flex flex-wrap items-center gap-2">
                  <UButton :to="`/write?slug=${item.slug}`" size="xs" color="primary" variant="soft" icon="i-lucide-square-pen" label="编辑" />
                  <UButton
                    v-if="item.status !== POST_STATUS_PUBLISHED"
                    size="xs"
                    color="success"
                    variant="soft"
                    icon="i-lucide-send"
                    :loading="updatingPostId === item.id"
                    label="发布"
                    @click="setPostStatus(item, POST_STATUS_PUBLISHED)"
                  />
                  <UButton
                    v-if="item.status !== POST_STATUS_DRAFT"
                    size="xs"
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-file-pen-line"
                    :loading="updatingPostId === item.id"
                    label="转草稿"
                    @click="setPostStatus(item, POST_STATUS_DRAFT)"
                  />
                  <UButton
                    v-if="item.status !== POST_STATUS_ARCHIVED"
                    size="xs"
                    color="warning"
                    variant="soft"
                    icon="i-lucide-archive"
                    :loading="updatingPostId === item.id"
                    label="归档"
                    @click="setPostStatus(item, POST_STATUS_ARCHIVED)"
                  />
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-message-square"
                    label="看评论"
                    @click="commentPostId = item.id; loadComments()"
                  />
                  <UButton
                    size="xs"
                    color="error"
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    :loading="deletingPostId === item.id"
                    label="删除"
                    @click="removePost(item.id)"
                  />
                </div>
              </div>
            </article>

            <UAlert
              v-if="!postLoading && postRows.length === 0"
              title="暂无文章"
              description="请更换关键词或稍后重试。"
              icon="i-lucide-file-text"
              color="neutral"
              variant="soft"
            />
          </div>

          <div v-if="postRows.length > 0" class="mt-5 flex items-center justify-end gap-2">
            <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-chevron-left" :disabled="postPage <= 1" @click="postPage = Math.max(1, postPage - 1)" />
            <span class="text-xs text-toned">第 {{ postPage }} / {{ totalPages }} 页</span>
            <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-chevron-right" :disabled="postPage >= totalPages" @click="postPage = Math.min(totalPages, postPage + 1)" />
          </div>
        </section>

        <section class="mt-8 rounded-2xl border border-default bg-default p-5 sm:p-6">
          <div class="flex flex-wrap items-end gap-3">
            <UFormField label="评论审核（按文章 ID）" class="min-w-56 flex-1">
              <UInput v-model="commentPostId" placeholder="输入文章 ID 拉取评论" icon="i-lucide-message-circle" />
            </UFormField>

            <UButton :loading="commentLoading" label="查询评论" icon="i-lucide-list-filter" color="neutral" @click="loadComments" />
          </div>

          <p class="mt-4 text-xs text-toned">
            共 {{ commentTotal || commentRows.length }} 条评论
          </p>

          <div class="mt-4 space-y-3">
            <article v-for="item in commentRows" :key="item.id" class="rounded-xl border border-default p-4">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="text-sm font-medium text-highlighted">
                    {{ item.author?.name || `用户 ${item.userId}` }}
                  </p>
                  <p class="mt-1 text-xs text-toned">
                    评论ID: {{ item.id }}
                  </p>
                </div>

                <UButton
                  :loading="deletingCommentId === item.id"
                  size="xs"
                  color="error"
                  variant="ghost"
                  icon="i-lucide-trash-2"
                  label="删除"
                  @click="removeComment(item.id)"
                />
              </div>

              <p class="mt-3 whitespace-pre-wrap text-sm text-toned">
                {{ item.content }}
              </p>
            </article>

            <UAlert
              v-if="!commentLoading && commentRows.length === 0"
              title="暂无评论数据"
              description="输入文章 ID 后点击查询评论。"
              icon="i-lucide-message-square-off"
              color="neutral"
              variant="soft"
            />
          </div>
        </section>

        <section class="mt-8 grid gap-6 lg:grid-cols-2">
          <UCard>
            <template #header>
              <h2 class="text-lg font-semibold text-highlighted">
                分类管理
              </h2>
            </template>

            <div class="space-y-3">
              <div class="grid gap-3 sm:grid-cols-2">
                <UInput v-model="newCategory.name" placeholder="分类名（必填）" />
                <UInput v-model="newCategory.slug" placeholder="slug（可选）" />
              </div>
              <UTextarea v-model="newCategory.description" :rows="2" placeholder="描述（可选）" />
              <div class="flex justify-end">
                <UButton :loading="savingCategory" label="新增分类" icon="i-lucide-plus" @click="createCategoryEntry" />
              </div>
            </div>

            <div class="mt-4 space-y-2">
              <div v-if="categoryLoading" class="text-sm text-toned">
                分类加载中...
              </div>
              <article v-for="item in categories" :key="item.id" class="rounded-lg border border-default p-3">
                <div class="flex items-center gap-2">
                  <UInput v-model="categoryRename[item.id]" class="flex-1" />
                  <UButton size="xs" color="neutral" variant="soft" :loading="updatingCategoryId === item.id" label="保存" @click="renameCategory(item.id)" />
                  <UButton size="xs" color="error" variant="ghost" :loading="deletingCategoryId === item.id" icon="i-lucide-trash-2" @click="removeCategory(item.id)" />
                </div>
              </article>
            </div>
          </UCard>

          <UCard>
            <template #header>
              <h2 class="text-lg font-semibold text-highlighted">
                标签管理
              </h2>
            </template>

            <div class="space-y-3">
              <div class="grid gap-3 sm:grid-cols-2">
                <UInput v-model="newTag.name" placeholder="标签名（必填）" />
                <UInput v-model="newTag.slug" placeholder="slug（可选）" />
              </div>
              <div class="flex justify-end">
                <UButton :loading="savingTag" label="新增标签" icon="i-lucide-plus" @click="createTagEntry" />
              </div>
            </div>

            <div class="mt-4 space-y-2">
              <div v-if="tagLoading" class="text-sm text-toned">
                标签加载中...
              </div>
              <article v-for="item in tags" :key="item.id" class="rounded-lg border border-default p-3">
                <div class="flex items-center gap-2">
                  <UInput v-model="tagRename[item.id]" class="flex-1" />
                  <UButton size="xs" color="neutral" variant="soft" :loading="updatingTagId === item.id" label="保存" @click="renameTag(item.id)" />
                  <UButton size="xs" color="error" variant="ghost" :loading="deletingTagId === item.id" icon="i-lucide-trash-2" @click="removeTag(item.id)" />
                </div>
              </article>
            </div>
          </UCard>
        </section>
      </template>
    </main>
  </div>
</template>
