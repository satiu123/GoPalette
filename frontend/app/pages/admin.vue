<script setup lang="ts">
import {
  POST_STATUS_ARCHIVED,
  POST_STATUS_DRAFT,
  POST_STATUS_OFFLINE,
  POST_STATUS_PUBLISHED,
  POST_STATUS_PRIVATE,
  USER_ROLE_ADMIN,
  USER_ROLE_USER,
  USER_STATUS_ACTIVE,
  USER_STATUS_INACTIVE,
  createCategory,
  createAdminUser,
  createTag,
  deleteAdminUser,
  deleteCategory,
  deleteComment,
  deletePost,
  deleteTag,
  fetchCategories,
  fetchAdminUsers,
  fetchCommentQueue,
  fetchComments,
  fetchPosts,
  fetchTags,
  searchPosts,
  updateCategory,
  updateAdminUser,
  updatePost,
  updateTag,
  reviewComment,
  COMMENT_STATUS_DELETED,
  COMMENT_STATUS_NORMAL
} from '~/composables/useBlogApi'
import type { AdminUserItem, BlogPostItem, CategoryItem, CommentInfo, TagItem } from '~/composables/useBlogApi'

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
const selectedPostIds = ref<string[]>([])
const deletingPostId = ref('')
const updatingPostId = ref('')

const commentPostId = ref('')
const commentPage = ref(1)
const commentPageSize = 50
const commentLoading = ref(false)
const commentRows = ref<CommentInfo[]>([])
const commentTotal = ref(0)
const selectedCommentIds = ref<string[]>([])
const deletingCommentId = ref('')
const reviewingCommentId = ref('')

const userPage = ref(1)
const userPageSize = 20
const userLoading = ref(false)
const userRows = ref<AdminUserItem[]>([])
const userTotal = ref(0)
const selectedUserIds = ref<string[]>([])
const savingUser = ref(false)
const updatingUserId = ref('')
const deletingUserId = ref('')
const newUser = reactive({
  username: '',
  email: '',
  password: '',
  role: USER_ROLE_USER as 0 | 1
})

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
const selectedCategoryIds = ref<string[]>([])
const selectedTagIds = ref<string[]>([])

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
const taxonomyKeyword = ref('')

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

function keepExistingSelection(source: string[], rows: Array<{ id: string }>) {
  const ids = new Set(rows.map(item => item.id))
  return source.filter(id => ids.has(id))
}

function toggleSelected(target: Ref<string[]>, id: string, checked: boolean) {
  if (!id) return

  if (checked) {
    if (!target.value.includes(id)) {
      target.value = [...target.value, id]
    }
    return
  }

  target.value = target.value.filter(item => item !== id)
}

function toggleVisibleSelection(target: Ref<string[]>, rows: Array<{ id: string }>, checked: boolean) {
  const ids = rows.map(item => item.id).filter(Boolean)
  if (checked) {
    target.value = Array.from(new Set([...target.value, ...ids]))
    return
  }

  const visibleIds = new Set(ids)
  target.value = target.value.filter(id => !visibleIds.has(id))
}

function togglePostSelected(id: string, checked: boolean) {
  toggleSelected(selectedPostIds, id, checked)
}

function toggleVisiblePosts(checked: boolean) {
  toggleVisibleSelection(selectedPostIds, postRows.value, checked)
}

function toggleCommentSelected(id: string, checked: boolean) {
  toggleSelected(selectedCommentIds, id, checked)
}

function toggleVisibleComments(checked: boolean) {
  toggleVisibleSelection(selectedCommentIds, commentRows.value, checked)
}

function toggleUserSelected(id: string, checked: boolean) {
  toggleSelected(selectedUserIds, id, checked)
}

function toggleVisibleUsers(checked: boolean) {
  toggleVisibleSelection(selectedUserIds, userRows.value, checked)
}

function toggleCategorySelected(id: string, checked: boolean) {
  toggleSelected(selectedCategoryIds, id, checked)
}

function toggleVisibleCategories(checked: boolean) {
  toggleVisibleSelection(selectedCategoryIds, filteredCategories.value, checked)
}

function toggleTagSelected(id: string, checked: boolean) {
  toggleSelected(selectedTagIds, id, checked)
}

function toggleVisibleTags(checked: boolean) {
  toggleVisibleSelection(selectedTagIds, filteredTags.value, checked)
}

function postStatusText(status: number) {
  if (status === POST_STATUS_PUBLISHED) return '已发布'
  if (status === POST_STATUS_ARCHIVED) return '已归档'
  if (status === POST_STATUS_PRIVATE) return '私密'
  if (status === POST_STATUS_OFFLINE) return '已下线'
  return '草稿'
}

function postStatusColor(status: number) {
  if (status === POST_STATUS_PUBLISHED) return 'success'
  if (status === POST_STATUS_ARCHIVED) return 'warning'
  if (status === POST_STATUS_PRIVATE) return 'primary'
  if (status === POST_STATUS_OFFLINE) return 'error'
  return 'neutral'
}

function toCommentStatus(status: CommentInfo['status']) {
  if (typeof status === 'number') return status

  const value = String(status || '').trim()
  const numericValue = Number(value)
  if (!Number.isNaN(numericValue)) return numericValue

  const normalized = value.toUpperCase()
  if (normalized.includes('PENDING')) return 2
  if (normalized.includes('DELETED')) return 3
  if (normalized.includes('NORMAL')) return 1
  return 1
}

function commentStatusText(status: CommentInfo['status']) {
  const value = toCommentStatus(status)
  if (value === 2) return '待审'
  if (value === 3) return '已删除'
  return '正常'
}

function commentStatusColor(status: CommentInfo['status']) {
  const value = toCommentStatus(status)
  if (value === 2) return 'warning'
  if (value === 3) return 'error'
  return 'success'
}

function toNumericValue(value: number | string | undefined, fallback: number) {
  if (typeof value === 'number') return value
  const text = String(value || '').trim().toUpperCase()
  if (text === 'ADMIN' || text === 'INACTIVE') return 1
  if (text === 'USER' || text === 'ACTIVE') return 0
  const parsed = Number(text)
  return Number.isFinite(parsed) ? parsed : fallback
}

function userRoleText(role: AdminUserItem['role']) {
  return toNumericValue(role, USER_ROLE_USER) === USER_ROLE_ADMIN ? '管理员' : '用户'
}

function userStatusText(status: AdminUserItem['status']) {
  return toNumericValue(status, USER_STATUS_ACTIVE) === USER_STATUS_INACTIVE ? '停用' : '活跃'
}

function userStatusColor(status: AdminUserItem['status']) {
  return toNumericValue(status, USER_STATUS_ACTIVE) === USER_STATUS_INACTIVE ? 'warning' : 'success'
}

function toCover(seed: string) {
  return `/covers/${encodeURIComponent(seed || 'gopalette')}.svg`
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
        authorId: '',
        publishedAt: item.createdAt || '未知时间',
        readingMinutes: Math.max(1, Math.ceil(item.summary.length / 300)),
        cover: toCover(item.slug || item.id)
      }))
      selectedPostIds.value = keepExistingSelection(selectedPostIds.value, postRows.value)
      return
    }

    const response = await fetchPosts(postPage.value, pageSize)
    postTotal.value = response.total
    postRows.value = response.posts
    selectedPostIds.value = keepExistingSelection(selectedPostIds.value, postRows.value)
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

  commentLoading.value = true
  try {
    const response = postId
      ? await fetchComments(postId, commentPage.value, commentPageSize)
      : await fetchCommentQueue(commentPage.value, commentPageSize)
    commentRows.value = response.comments
    commentTotal.value = response.total
    selectedCommentIds.value = keepExistingSelection(selectedCommentIds.value, commentRows.value)
  } catch (error: unknown) {
    toast.add({
      title: '加载评论失败',
      description: getErrorMessage(error, postId ? '请检查文章 ID 是否正确' : '请确认当前账号有管理员权限'),
      color: 'error'
    })
  } finally {
    commentLoading.value = false
  }
}

function reloadCommentsFromFirstPage() {
  if (commentPage.value === 1) {
    void loadComments()
    return
  }
  commentPage.value = 1
}

function openPostComments(postId: string) {
  commentPostId.value = postId
  reloadCommentsFromFirstPage()
}

function showGlobalCommentQueue() {
  commentPostId.value = ''
  reloadCommentsFromFirstPage()
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

async function removeSelectedComments() {
  const ids = [...selectedCommentIds.value]
  if (ids.length === 0 || deletingCommentId.value) return
  if (!askConfirm(`确认删除选中的 ${ids.length} 条评论吗？`)) return

  deletingCommentId.value = '__batch__'
  try {
    await Promise.all(ids.map(id => deleteComment(id)))
    selectedCommentIds.value = []
    toast.add({ title: '选中评论已删除', color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '批量删除评论失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    deletingCommentId.value = ''
  }
}

async function setCommentReviewStatus(id: string, status: number) {
  if (!id || reviewingCommentId.value) return

  reviewingCommentId.value = id
  try {
    await reviewComment(id, status)
    toast.add({ title: status === COMMENT_STATUS_NORMAL ? '评论已通过' : '评论已删除', color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '审核评论失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    reviewingCommentId.value = ''
  }
}

async function reviewSelectedComments(status: number) {
  const ids = [...selectedCommentIds.value]
  if (ids.length === 0 || reviewingCommentId.value) return
  const action = status === COMMENT_STATUS_NORMAL ? '通过' : '删除'
  if (!askConfirm(`确认${action}选中的 ${ids.length} 条评论吗？`)) return

  reviewingCommentId.value = '__batch__'
  try {
    await Promise.all(ids.map(id => reviewComment(id, status)))
    selectedCommentIds.value = []
    toast.add({ title: `选中评论已${action}`, color: 'success' })
    await loadComments()
  } catch (error: unknown) {
    toast.add({
      title: '批量审核评论失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    reviewingCommentId.value = ''
  }
}

async function loadUsers() {
  userLoading.value = true
  try {
    const response = await fetchAdminUsers(userPage.value, userPageSize)
    userRows.value = response.users
    userTotal.value = response.total
    selectedUserIds.value = keepExistingSelection(selectedUserIds.value, userRows.value)
  } catch (error: unknown) {
    toast.add({
      title: '加载用户失败',
      description: getErrorMessage(error, '请确认当前账号有管理员权限'),
      color: 'error'
    })
  } finally {
    userLoading.value = false
  }
}

async function createUserEntry() {
  if (!newUser.username.trim() || !newUser.email.trim() || !newUser.password.trim() || savingUser.value) return

  savingUser.value = true
  try {
    await createAdminUser({
      username: newUser.username.trim(),
      email: newUser.email.trim(),
      password: newUser.password,
      role: newUser.role
    })
    newUser.username = ''
    newUser.email = ''
    newUser.password = ''
    newUser.role = USER_ROLE_USER
    toast.add({ title: '用户创建成功', color: 'success' })
    await loadUsers()
  } catch (error: unknown) {
    toast.add({
      title: '创建用户失败',
      description: getErrorMessage(error, '请检查邮箱是否已存在'),
      color: 'error'
    })
  } finally {
    savingUser.value = false
  }
}

async function setUserStatus(item: AdminUserItem, status: number) {
  if (!item.id || updatingUserId.value) return

  updatingUserId.value = item.id
  try {
    await updateAdminUser(item.id, {
      username: item.username,
      email: item.email,
      role: toNumericValue(item.role, USER_ROLE_USER),
      status,
      bio: item.bio || '',
      location: item.location || '',
      avatarURL: item.avatarURL || '',
      updateMask: 'status'
    })
    toast.add({ title: '用户状态已更新', color: 'success' })
    await loadUsers()
  } catch (error: unknown) {
    toast.add({
      title: '更新用户失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    updatingUserId.value = ''
  }
}

async function setUserRole(item: AdminUserItem, role: number) {
  if (!item.id || updatingUserId.value) return

  updatingUserId.value = item.id
  try {
    await updateAdminUser(item.id, {
      username: item.username,
      email: item.email,
      role,
      status: toNumericValue(item.status, USER_STATUS_ACTIVE),
      bio: item.bio || '',
      location: item.location || '',
      avatarURL: item.avatarURL || '',
      updateMask: 'role'
    })
    toast.add({ title: '用户角色已更新', color: 'success' })
    await loadUsers()
  } catch (error: unknown) {
    toast.add({
      title: '更新角色失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    updatingUserId.value = ''
  }
}

async function removeUser(id: string) {
  if (!id || deletingUserId.value) return
  if (!askConfirm('确认删除该用户吗？此操作不可恢复。')) return

  deletingUserId.value = id
  try {
    await deleteAdminUser(id)
    toast.add({ title: '用户已删除', color: 'success' })
    await loadUsers()
  } catch (error: unknown) {
    toast.add({
      title: '删除用户失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    deletingUserId.value = ''
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

async function setSelectedPostStatus(status: number) {
  const ids = new Set(selectedPostIds.value)
  const items = postRows.value.filter(item => ids.has(item.id))
  if (items.length === 0 || updatingPostId.value) return

  updatingPostId.value = '__batch__'
  try {
    await Promise.all(items.map(item => updatePost(item.id, {
      title: item.title,
      summary: item.summary,
      slug: item.slug,
      content: item.content || '',
      status,
      categoryId: item.categoryId || undefined,
      tags: item.tags,
      updateMask: 'status'
    })))
    selectedPostIds.value = []
    toast.add({ title: '选中文章状态已更新', color: 'success' })
    await loadPosts()
  } catch (error: unknown) {
    toast.add({
      title: '批量更新文章失败',
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

async function removeSelectedPosts() {
  const ids = [...selectedPostIds.value]
  if (ids.length === 0 || deletingPostId.value) return
  if (!askConfirm(`确认删除选中的 ${ids.length} 篇文章吗？此操作不可恢复。`)) return

  deletingPostId.value = '__batch__'
  try {
    await Promise.all(ids.map(id => deletePost(id)))
    selectedPostIds.value = []
    toast.add({ title: '选中文章已删除', color: 'success' })
    await loadPosts()
  } catch (error: unknown) {
    toast.add({
      title: '批量删除文章失败',
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
    selectedCategoryIds.value = keepExistingSelection(selectedCategoryIds.value, categories.value)
    selectedTagIds.value = keepExistingSelection(selectedTagIds.value, tags.value)
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

async function removeSelectedCategories() {
  const ids = [...selectedCategoryIds.value]
  if (ids.length === 0 || deletingCategoryId.value) return
  if (!askConfirm(`确认删除选中的 ${ids.length} 个分类吗？`)) return

  deletingCategoryId.value = '__batch__'
  try {
    await Promise.all(ids.map(id => deleteCategory(id)))
    selectedCategoryIds.value = []
    toast.add({ title: '选中分类已删除', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '批量删除分类失败',
      description: getErrorMessage(error, '请先清空分类下文章后重试'),
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

async function removeSelectedTags() {
  const ids = [...selectedTagIds.value]
  if (ids.length === 0 || deletingTagId.value) return
  if (!askConfirm(`确认删除选中的 ${ids.length} 个标签吗？`)) return

  deletingTagId.value = '__batch__'
  try {
    await Promise.all(ids.map(id => deleteTag(id)))
    selectedTagIds.value = []
    toast.add({ title: '选中标签已删除', color: 'success' })
    await loadTaxonomy()
  } catch (error: unknown) {
    toast.add({
      title: '批量删除标签失败',
      description: getErrorMessage(error, '请先取消文章与标签关联后重试'),
      color: 'error'
    })
  } finally {
    deletingTagId.value = ''
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(postTotal.value / pageSize)))
const commentTotalPages = computed(() => Math.max(1, Math.ceil(commentTotal.value / commentPageSize)))
const userTotalPages = computed(() => Math.max(1, Math.ceil(userTotal.value / userPageSize)))
const visiblePostTotal = computed(() => postTotal.value || postRows.value.length)
const publishedPostCount = computed(() => postRows.value.filter(item => item.status === POST_STATUS_PUBLISHED).length)
const draftPostCount = computed(() => postRows.value.filter(item => item.status === POST_STATUS_DRAFT).length)
const archivedPostCount = computed(() => postRows.value.filter(item => item.status === POST_STATUS_ARCHIVED).length)
const privatePostCount = computed(() => postRows.value.filter(item => item.status === POST_STATUS_PRIVATE).length)
const offlinePostCount = computed(() => postRows.value.filter(item => item.status === POST_STATUS_OFFLINE).length)
const taxonomyTotal = computed(() => categories.value.length + tags.value.length)
const taxonomyQuery = computed(() => taxonomyKeyword.value.trim().toLowerCase())
const filteredCategories = computed(() => {
  const query = taxonomyQuery.value
  if (!query) return categories.value

  return categories.value.filter(item => item.name.toLowerCase().includes(query) || item.id.toLowerCase().includes(query))
})
const filteredTags = computed(() => {
  const query = taxonomyQuery.value
  if (!query) return tags.value

  return tags.value.filter(item => item.name.toLowerCase().includes(query) || item.id.toLowerCase().includes(query))
})
const allVisiblePostsSelected = computed(() => postRows.value.length > 0 && postRows.value.every(item => selectedPostIds.value.includes(item.id)))
const allVisibleCommentsSelected = computed(() => commentRows.value.length > 0 && commentRows.value.every(item => selectedCommentIds.value.includes(item.id)))
const allVisibleUsersSelected = computed(() => userRows.value.length > 0 && userRows.value.every(item => selectedUserIds.value.includes(item.id)))
const allVisibleCategoriesSelected = computed(() => filteredCategories.value.length > 0 && filteredCategories.value.every(item => selectedCategoryIds.value.includes(item.id)))
const allVisibleTagsSelected = computed(() => filteredTags.value.length > 0 && filteredTags.value.every(item => selectedTagIds.value.includes(item.id)))
const activeCommentPostTitle = computed(() => {
  const postId = commentPostId.value.trim()
  if (!postId) return ''

  return postRows.value.find(item => item.id === postId)?.title || ''
})

const adminStats = computed(() => [
  {
    label: '文章总量',
    value: visiblePostTotal.value,
    meta: postKeyword.value.trim() ? '当前检索结果' : '当前分页范围',
    icon: 'i-lucide-files',
    color: 'text-primary'
  },
  {
    label: '已发布',
    value: publishedPostCount.value,
    meta: `草稿 ${draftPostCount.value} / 私密 ${privatePostCount.value} / 下线 ${offlinePostCount.value} / 归档 ${archivedPostCount.value}`,
    icon: 'i-lucide-send',
    color: 'text-success'
  },
  {
    label: '评论队列',
    value: commentTotal.value || commentRows.value.length,
    meta: activeCommentPostTitle.value || (commentPostId.value.trim() ? '已选择文章' : '全站队列'),
    icon: 'i-lucide-message-square-text',
    color: 'text-warning'
  },
  {
    label: '内容结构',
    value: taxonomyTotal.value,
    meta: `${categories.value.length} 个分类 / ${tags.value.length} 个标签`,
    icon: 'i-lucide-tags',
    color: 'text-info'
  },
  {
    label: '用户',
    value: userTotal.value || userRows.value.length,
    meta: `${userRows.value.filter(item => toNumericValue(item.role, USER_ROLE_USER) === USER_ROLE_ADMIN).length} 个管理员`,
    icon: 'i-lucide-users',
    color: 'text-primary'
  }
])

watch(postPage, () => {
  void loadPosts()
})

watch(commentPage, () => {
  void loadComments()
})

watch(userPage, () => {
  void loadUsers()
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
    loadComments(),
    loadUsers(),
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

    <main class="mx-auto w-full max-w-7xl px-4 pb-20 pt-8 sm:px-14">
      <div
        v-if="checkingAuth"
        class="rounded-lg border border-default bg-default p-6"
      >
        <div class="flex items-center gap-3">
          <USkeleton class="size-10 rounded-md" />
          <div class="flex-1 space-y-2">
            <USkeleton class="h-4 w-40" />
            <USkeleton class="h-3 w-64 max-w-full" />
          </div>
        </div>
      </div>

      <template v-else>
        <div class="mb-5 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
              后台管理
            </h1>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <UButton
              color="neutral"
              variant="soft"
              icon="i-lucide-refresh-cw"
              label="刷新数据"
              :loading="postLoading || commentLoading || userLoading || categoryLoading || tagLoading"
              @click="loadPosts(); loadComments(); loadUsers(); loadTaxonomy()"
            />
            <UButton
              to="/write"
              icon="i-lucide-pen-line"
              label="写新文章"
            />
          </div>
        </div>

        <section class="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          <article
            v-for="item in adminStats"
            :key="item.label"
            class="rounded-lg border border-default bg-default p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-toned">
                  {{ item.label }}
                </p>
                <p class="mt-2 text-2xl font-semibold text-highlighted">
                  {{ item.value }}
                </p>
              </div>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
                <UIcon
                  :name="item.icon"
                  class="size-4"
                  :class="item.color"
                />
              </div>
            </div>
            <p class="mt-3 truncate text-xs text-muted">
              {{ item.meta }}
            </p>
          </article>
        </section>

        <section class="rounded-lg border border-default bg-default">
          <div class="flex flex-wrap items-end justify-between gap-3 border-b border-default p-4 sm:p-5">
            <div>
              <h2 class="text-base font-semibold text-highlighted">
                文章管理
              </h2>
              <p class="mt-1 text-xs text-toned">
                快速检索、切换状态和进入编辑
              </p>
            </div>

            <div class="flex w-full flex-wrap items-end gap-2 lg:w-auto">
              <UFormField
                label="文章关键词"
                class="min-w-56 flex-1 lg:w-72 lg:flex-none"
              >
                <UInput
                  v-model="postKeyword"
                  placeholder="按标题检索"
                  icon="i-lucide-search"
                  @keyup.enter="loadPosts"
                />
              </UFormField>

              <UButton
                :loading="postLoading"
                label="查询"
                icon="i-lucide-filter"
                @click="loadPosts"
              />
            </div>
          </div>

          <div
            v-if="postRows.length > 0"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-default bg-muted/30 px-4 py-3 sm:px-5"
          >
            <label class="inline-flex items-center gap-2 text-xs font-medium text-toned">
              <input
                type="checkbox"
                class="size-4 rounded border-default accent-[var(--ui-primary)]"
                :checked="allVisiblePostsSelected"
                @change="toggleVisiblePosts(($event.target as HTMLInputElement).checked)"
              >
              当前页全选
              <span
                v-if="selectedPostIds.length > 0"
                class="text-muted"
              >
                已选 {{ selectedPostIds.length }}
              </span>
            </label>

            <div class="flex flex-wrap items-center gap-2">
              <UButton
                size="xs"
                color="success"
                variant="soft"
                icon="i-lucide-send"
                label="批量发布"
                :disabled="selectedPostIds.length === 0"
                :loading="updatingPostId === '__batch__'"
                @click="setSelectedPostStatus(POST_STATUS_PUBLISHED)"
              />
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-file-pen-line"
                label="转草稿"
                :disabled="selectedPostIds.length === 0"
                :loading="updatingPostId === '__batch__'"
                @click="setSelectedPostStatus(POST_STATUS_DRAFT)"
              />
              <UButton
                size="xs"
                color="warning"
                variant="soft"
                icon="i-lucide-archive"
                label="批量归档"
                :disabled="selectedPostIds.length === 0"
                :loading="updatingPostId === '__batch__'"
                @click="setSelectedPostStatus(POST_STATUS_ARCHIVED)"
              />
              <UButton
                size="xs"
                color="primary"
                variant="soft"
                icon="i-lucide-lock"
                label="批量私密"
                :disabled="selectedPostIds.length === 0"
                :loading="updatingPostId === '__batch__'"
                @click="setSelectedPostStatus(POST_STATUS_PRIVATE)"
              />
              <UButton
                size="xs"
                color="error"
                variant="soft"
                icon="i-lucide-cloud-off"
                label="批量下线"
                :disabled="selectedPostIds.length === 0"
                :loading="updatingPostId === '__batch__'"
                @click="setSelectedPostStatus(POST_STATUS_OFFLINE)"
              />
              <UButton
                size="xs"
                color="error"
                variant="ghost"
                icon="i-lucide-trash-2"
                label="批量删除"
                :disabled="selectedPostIds.length === 0"
                :loading="deletingPostId === '__batch__'"
                @click="removeSelectedPosts"
              />
            </div>
          </div>

          <div class="divide-y divide-default">
            <div
              v-if="postLoading && postRows.length === 0"
              class="space-y-3 p-4 sm:p-5"
            >
              <div
                v-for="index in 4"
                :key="index"
                class="rounded-lg border border-default p-4"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="min-w-0 flex-1 space-y-3">
                    <USkeleton class="h-5 w-2/3" />
                    <USkeleton class="h-3 w-full" />
                    <USkeleton class="h-3 w-1/2" />
                  </div>
                  <USkeleton class="h-8 w-28" />
                </div>
              </div>
            </div>

            <article
              v-for="item in postRows"
              :key="item.id"
              class="p-4 transition-colors hover:bg-muted/40 sm:p-5"
            >
              <div class="grid gap-3 xl:grid-cols-[auto_minmax(0,1fr)_auto] xl:items-center">
                <input
                  type="checkbox"
                  class="mt-1 size-4 rounded border-default accent-[var(--ui-primary)] xl:mt-0"
                  :checked="selectedPostIds.includes(item.id)"
                  @change="togglePostSelected(item.id, ($event.target as HTMLInputElement).checked)"
                >
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <!-- eslint-disable vue/no-v-html -->
                    <p
                      class="min-w-0 text-base font-semibold text-highlighted"
                      v-html="renderHighlightedText(item.title)"
                    />
                    <!-- eslint-enable vue/no-v-html -->
                    <UBadge
                      size="xs"
                      :label="postStatusText(item.status)"
                      :color="postStatusColor(item.status)"
                      variant="subtle"
                    />
                  </div>
                  <p class="mt-2 line-clamp-2 text-sm text-toned">
                    {{ item.summary }}
                  </p>
                  <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
                    <span class="inline-flex items-center gap-1">
                      <UIcon
                        name="i-lucide-hash"
                        class="size-3.5"
                      />
                      {{ item.id }}
                    </span>
                    <span class="inline-flex items-center gap-1">
                      <UIcon
                        name="i-lucide-folder"
                        class="size-3.5"
                      />
                      {{ item.category || '未分类' }}
                    </span>
                    <span class="inline-flex items-center gap-1">
                      <UIcon
                        name="i-lucide-clock-3"
                        class="size-3.5"
                      />
                      {{ item.publishedAt }}
                    </span>
                    <span class="inline-flex items-center gap-1">
                      <UIcon
                        name="i-lucide-tags"
                        class="size-3.5"
                      />
                      {{ item.tags.join(', ') || '-' }}
                    </span>
                  </div>
                </div>

                <div class="flex flex-wrap items-center gap-2 xl:justify-end">
                  <UButton
                    :to="`/write?slug=${item.slug}`"
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-square-pen"
                    label="编辑"
                  />
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
                    v-if="item.status !== POST_STATUS_PRIVATE"
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-lock"
                    :loading="updatingPostId === item.id"
                    label="私密"
                    @click="setPostStatus(item, POST_STATUS_PRIVATE)"
                  />
                  <UButton
                    v-if="item.status !== POST_STATUS_OFFLINE"
                    size="xs"
                    color="error"
                    variant="soft"
                    icon="i-lucide-cloud-off"
                    :loading="updatingPostId === item.id"
                    label="下线"
                    @click="setPostStatus(item, POST_STATUS_OFFLINE)"
                  />
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    icon="i-lucide-message-square"
                    label="看评论"
                    @click="openPostComments(item.id)"
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
              class="m-4 sm:m-5"
              title="暂无文章"
              description="请更换关键词或稍后重试。"
              icon="i-lucide-file-text"
              color="neutral"
              variant="soft"
            />
          </div>

          <div
            v-if="postRows.length > 0"
            class="flex items-center justify-between gap-3 border-t border-default p-4 sm:p-5"
          >
            <span class="text-xs text-toned">共 {{ visiblePostTotal }} 条</span>
            <div class="flex items-center gap-2">
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-chevron-left"
                :disabled="postPage <= 1"
                @click="postPage = Math.max(1, postPage - 1)"
              />
              <span class="text-xs text-toned">第 {{ postPage }} / {{ totalPages }} 页</span>
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-lucide-chevron-right"
                :disabled="postPage >= totalPages"
                @click="postPage = Math.min(totalPages, postPage + 1)"
              />
            </div>
          </div>
        </section>

        <section class="mt-6 rounded-lg border border-default bg-default">
          <div class="flex flex-wrap items-end justify-between gap-3 border-b border-default p-4 sm:p-5">
            <div>
              <h2 class="text-base font-semibold text-highlighted">
                全站评论队列
              </h2>
              <p class="mt-1 text-xs text-toned">
                {{ activeCommentPostTitle || (commentPostId.trim() ? '按文章过滤评论' : '默认显示全站最新评论') }}
              </p>
            </div>

            <div class="flex w-full flex-wrap items-end gap-2 lg:w-auto">
              <UFormField
                label="文章 ID（可选）"
                class="min-w-56 flex-1 lg:w-80 lg:flex-none"
              >
                <UInput
                  v-model="commentPostId"
                  placeholder="留空查看全站队列"
                  icon="i-lucide-message-circle"
                  @keyup.enter="reloadCommentsFromFirstPage"
                />
              </UFormField>

              <UButton
                :loading="commentLoading"
                label="查询"
                icon="i-lucide-list-filter"
                color="neutral"
                @click="reloadCommentsFromFirstPage"
              />
              <UButton
                v-if="commentPostId.trim()"
                label="全站队列"
                icon="i-lucide-list"
                color="neutral"
                variant="soft"
                @click="showGlobalCommentQueue"
              />
            </div>
          </div>

          <div class="space-y-3 p-4 sm:p-5">
            <div class="flex flex-wrap items-center justify-between gap-3 text-xs text-toned">
              <label class="inline-flex items-center gap-2 font-medium">
                <input
                  type="checkbox"
                  class="size-4 rounded border-default accent-[var(--ui-primary)]"
                  :checked="allVisibleCommentsSelected"
                  :disabled="commentRows.length === 0"
                  @change="toggleVisibleComments(($event.target as HTMLInputElement).checked)"
                >
                {{ commentPostId.trim() ? '当前文章' : '全站队列' }} · 共 {{ commentTotal || commentRows.length }} 条评论
                <span
                  v-if="selectedCommentIds.length > 0"
                  class="text-muted"
                >
                  已选 {{ selectedCommentIds.length }}
                </span>
              </label>

              <div class="flex flex-wrap items-center gap-2">
                <span v-if="commentPostId.trim()">文章 ID: {{ commentPostId.trim() }}</span>
                <UButton
                  size="xs"
                  color="success"
                  variant="soft"
                  icon="i-lucide-check"
                  label="批量通过"
                  :disabled="selectedCommentIds.length === 0"
                  :loading="reviewingCommentId === '__batch__'"
                  @click="reviewSelectedComments(COMMENT_STATUS_NORMAL)"
                />
                <UButton
                  size="xs"
                  color="warning"
                  variant="soft"
                  icon="i-lucide-ban"
                  label="批量驳回"
                  :disabled="selectedCommentIds.length === 0"
                  :loading="reviewingCommentId === '__batch__'"
                  @click="reviewSelectedComments(COMMENT_STATUS_DELETED)"
                />
                <UButton
                  size="xs"
                  color="error"
                  variant="ghost"
                  icon="i-lucide-trash-2"
                  label="批量删除"
                  :disabled="selectedCommentIds.length === 0"
                  :loading="deletingCommentId === '__batch__'"
                  @click="removeSelectedComments"
                />
              </div>
            </div>

            <div
              v-if="commentLoading && commentRows.length === 0"
              class="space-y-3"
            >
              <div
                v-for="index in 2"
                :key="index"
                class="rounded-lg border border-default p-4"
              >
                <USkeleton class="h-4 w-40" />
                <USkeleton class="mt-3 h-3 w-full" />
                <USkeleton class="mt-2 h-3 w-3/4" />
              </div>
            </div>

            <article
              v-for="item in commentRows"
              :key="item.id"
              class="rounded-lg border border-default p-4"
            >
              <div class="flex items-start gap-3">
                <input
                  type="checkbox"
                  class="mt-0.5 size-4 rounded border-default accent-[var(--ui-primary)]"
                  :checked="selectedCommentIds.includes(item.id)"
                  @change="toggleCommentSelected(item.id, ($event.target as HTMLInputElement).checked)"
                >
                <div class="min-w-0 flex-1">
                  <div class="flex items-start justify-between gap-3">
                    <div>
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-medium text-highlighted">
                          {{ item.author?.name || `用户 ${item.userId}` }}
                        </p>
                        <UBadge
                          size="xs"
                          variant="soft"
                          :color="commentStatusColor(item.status)"
                        >
                          {{ commentStatusText(item.status) }}
                        </UBadge>
                      </div>
                      <p class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-toned">
                        <span>文章ID: {{ item.postId }}</span>
                        <span>评论ID: {{ item.id }}</span>
                        <span v-if="item.parentId">父评论: {{ item.parentId }}</span>
                        <span v-if="item.createdAt">{{ item.createdAt }}</span>
                      </p>
                    </div>

                    <div class="flex flex-wrap items-center justify-end gap-2">
                      <UButton
                        v-if="toCommentStatus(item.status) === 2"
                        :loading="reviewingCommentId === item.id"
                        size="xs"
                        color="success"
                        variant="soft"
                        icon="i-lucide-check"
                        label="通过"
                        @click="setCommentReviewStatus(item.id, COMMENT_STATUS_NORMAL)"
                      />
                      <UButton
                        v-if="toCommentStatus(item.status) !== 3"
                        :loading="reviewingCommentId === item.id"
                        size="xs"
                        color="warning"
                        variant="soft"
                        icon="i-lucide-ban"
                        label="驳回"
                        @click="setCommentReviewStatus(item.id, COMMENT_STATUS_DELETED)"
                      />
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
                  </div>

                  <p class="mt-3 whitespace-pre-wrap text-sm text-toned">
                    {{ item.content }}
                  </p>
                </div>
              </div>
            </article>

            <UAlert
              v-if="!commentLoading && commentRows.length === 0"
              :title="commentPostId.trim() ? '该文章暂无评论' : '暂无全站评论'"
              :description="commentPostId.trim() ? '可以清空文章 ID 查看全站队列。' : '新的评论会出现在这里。'"
              icon="i-lucide-message-square-off"
              color="neutral"
              variant="soft"
            />

            <div
              v-if="commentRows.length > 0"
              class="flex items-center justify-between gap-3 border-t border-default pt-3"
            >
              <span class="text-xs text-toned">每页 {{ commentPageSize }} 条</span>
              <div class="flex items-center gap-2">
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-chevron-left"
                  :disabled="commentPage <= 1"
                  @click="commentPage = Math.max(1, commentPage - 1)"
                />
                <span class="text-xs text-toned">第 {{ commentPage }} / {{ commentTotalPages }} 页</span>
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-chevron-right"
                  :disabled="commentPage >= commentTotalPages"
                  @click="commentPage = Math.min(commentTotalPages, commentPage + 1)"
                />
              </div>
            </div>
          </div>
        </section>

        <section class="mt-6 rounded-lg border border-default bg-default">
          <div class="flex flex-wrap items-end justify-between gap-3 border-b border-default p-4 sm:p-5">
            <div>
              <h2 class="text-base font-semibold text-highlighted">
                用户管理
              </h2>
              <p class="mt-1 text-xs text-toned">
                查看账号、切换状态和维护角色
              </p>
            </div>

            <UButton
              :loading="userLoading"
              label="刷新用户"
              icon="i-lucide-refresh-cw"
              color="neutral"
              variant="soft"
              @click="loadUsers"
            />
          </div>

          <div class="grid gap-4 p-4 lg:grid-cols-[20rem_minmax(0,1fr)] sm:p-5">
            <div class="rounded-lg border border-default p-4">
              <h3 class="text-sm font-semibold text-highlighted">
                新增用户
              </h3>
              <div class="mt-4 space-y-3">
                <UInput
                  v-model="newUser.username"
                  placeholder="用户名"
                />
                <UInput
                  v-model="newUser.email"
                  placeholder="邮箱"
                  type="email"
                />
                <UInput
                  v-model="newUser.password"
                  placeholder="初始密码"
                  type="password"
                />
                <USelect
                  v-model="newUser.role"
                  :items="[
                    { label: '普通用户', value: USER_ROLE_USER },
                    { label: '管理员', value: USER_ROLE_ADMIN }
                  ]"
                />
                <UButton
                  class="w-full justify-center"
                  :loading="savingUser"
                  label="创建用户"
                  icon="i-lucide-user-plus"
                  @click="createUserEntry"
                />
              </div>
            </div>

            <div class="min-w-0">
              <div
                v-if="userRows.length > 0"
                class="mb-3 flex flex-wrap items-center justify-between gap-3 text-xs text-toned"
              >
                <label class="inline-flex items-center gap-2 font-medium">
                  <input
                    type="checkbox"
                    class="size-4 rounded border-default accent-[var(--ui-primary)]"
                    :checked="allVisibleUsersSelected"
                    @change="toggleVisibleUsers(($event.target as HTMLInputElement).checked)"
                  >
                  当前页全选
                  <span
                    v-if="selectedUserIds.length > 0"
                    class="text-muted"
                  >
                    已选 {{ selectedUserIds.length }}
                  </span>
                </label>
                <span>共 {{ userTotal || userRows.length }} 个用户</span>
              </div>

              <div
                v-if="userLoading && userRows.length === 0"
                class="space-y-3"
              >
                <USkeleton
                  v-for="index in 3"
                  :key="index"
                  class="h-20 w-full rounded-lg"
                />
              </div>

              <div class="space-y-3">
                <article
                  v-for="item in userRows"
                  :key="item.id"
                  class="rounded-lg border border-default p-4"
                >
                  <div class="grid gap-3 xl:grid-cols-[auto_minmax(0,1fr)_auto] xl:items-center">
                    <input
                      type="checkbox"
                      class="mt-1 size-4 rounded border-default accent-[var(--ui-primary)] xl:mt-0"
                      :checked="selectedUserIds.includes(item.id)"
                      @change="toggleUserSelected(item.id, ($event.target as HTMLInputElement).checked)"
                    >
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-semibold text-highlighted">
                          {{ item.username || `用户 ${item.id}` }}
                        </p>
                        <UBadge
                          size="xs"
                          variant="soft"
                          :color="toNumericValue(item.role, USER_ROLE_USER) === USER_ROLE_ADMIN ? 'primary' : 'neutral'"
                        >
                          {{ userRoleText(item.role) }}
                        </UBadge>
                        <UBadge
                          size="xs"
                          variant="soft"
                          :color="userStatusColor(item.status)"
                        >
                          {{ userStatusText(item.status) }}
                        </UBadge>
                      </div>
                      <p class="mt-2 text-xs text-toned">
                        {{ item.email || '-' }} · ID {{ item.id }}<span v-if="item.location"> · {{ item.location }}</span>
                      </p>
                      <p
                        v-if="item.bio"
                        class="mt-2 line-clamp-1 text-xs text-muted"
                      >
                        {{ item.bio }}
                      </p>
                    </div>

                    <div class="flex flex-wrap items-center gap-2 xl:justify-end">
                      <UButton
                        v-if="toNumericValue(item.role, USER_ROLE_USER) !== USER_ROLE_ADMIN"
                        size="xs"
                        color="primary"
                        variant="soft"
                        icon="i-lucide-shield"
                        label="设为管理员"
                        :loading="updatingUserId === item.id"
                        @click="setUserRole(item, USER_ROLE_ADMIN)"
                      />
                      <UButton
                        v-if="toNumericValue(item.role, USER_ROLE_USER) !== USER_ROLE_USER"
                        size="xs"
                        color="neutral"
                        variant="soft"
                        icon="i-lucide-user"
                        label="设为用户"
                        :loading="updatingUserId === item.id"
                        @click="setUserRole(item, USER_ROLE_USER)"
                      />
                      <UButton
                        v-if="toNumericValue(item.status, USER_STATUS_ACTIVE) !== USER_STATUS_INACTIVE"
                        size="xs"
                        color="warning"
                        variant="soft"
                        icon="i-lucide-user-x"
                        label="停用"
                        :loading="updatingUserId === item.id"
                        @click="setUserStatus(item, USER_STATUS_INACTIVE)"
                      />
                      <UButton
                        v-if="toNumericValue(item.status, USER_STATUS_ACTIVE) !== USER_STATUS_ACTIVE"
                        size="xs"
                        color="success"
                        variant="soft"
                        icon="i-lucide-user-check"
                        label="启用"
                        :loading="updatingUserId === item.id"
                        @click="setUserStatus(item, USER_STATUS_ACTIVE)"
                      />
                      <UButton
                        size="xs"
                        color="error"
                        variant="ghost"
                        icon="i-lucide-trash-2"
                        label="删除"
                        :loading="deletingUserId === item.id"
                        @click="removeUser(item.id)"
                      />
                    </div>
                  </div>
                </article>
              </div>

              <UAlert
                v-if="!userLoading && userRows.length === 0"
                title="暂无用户"
                description="创建用户后会显示在这里。"
                icon="i-lucide-users"
                color="neutral"
                variant="soft"
              />

              <div
                v-if="userRows.length > 0"
                class="mt-4 flex items-center justify-between gap-3 border-t border-default pt-3"
              >
                <span class="text-xs text-toned">每页 {{ userPageSize }} 个</span>
                <div class="flex items-center gap-2">
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-chevron-left"
                    :disabled="userPage <= 1"
                    @click="userPage = Math.max(1, userPage - 1)"
                  />
                  <span class="text-xs text-toned">第 {{ userPage }} / {{ userTotalPages }} 页</span>
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="soft"
                    icon="i-lucide-chevron-right"
                    :disabled="userPage >= userTotalPages"
                    @click="userPage = Math.min(userTotalPages, userPage + 1)"
                  />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="mt-6">
          <div class="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-highlighted">
                内容结构
              </h2>
              <p class="mt-1 text-xs text-toned">
                {{ taxonomyTotal }} 个条目，支持快速过滤和就地维护
              </p>
            </div>

            <UInput
              v-model="taxonomyKeyword"
              class="w-full sm:w-72"
              icon="i-lucide-search"
              placeholder="搜索分类或标签"
            />
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <UCard :ui="{ root: 'rounded-lg', body: 'p-4 sm:p-5', header: 'p-4 sm:p-5' }">
              <template #header>
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h2 class="text-base font-semibold text-highlighted">
                      分类管理
                    </h2>
                    <p class="mt-1 text-xs text-toned">
                      显示 {{ filteredCategories.length }} / {{ categories.length }} 个分类
                    </p>
                  </div>
                  <UIcon
                    name="i-lucide-folder-tree"
                    class="size-5 text-primary"
                  />
                </div>
              </template>

              <div class="space-y-3">
                <div class="grid gap-3 sm:grid-cols-2">
                  <UInput
                    v-model="newCategory.name"
                    placeholder="分类名（必填）"
                  />
                  <UInput
                    v-model="newCategory.slug"
                    placeholder="slug（可选）"
                  />
                </div>
                <UTextarea
                  v-model="newCategory.description"
                  :rows="2"
                  placeholder="描述（可选）"
                />
                <div class="flex justify-end">
                  <UButton
                    :loading="savingCategory"
                    label="新增分类"
                    icon="i-lucide-plus"
                    @click="createCategoryEntry"
                  />
                </div>
              </div>

              <div class="mt-4 max-h-[28rem] space-y-2 overflow-y-auto pr-1">
                <div
                  v-if="filteredCategories.length > 0"
                  class="sticky top-0 z-10 flex items-center justify-between gap-3 border border-default bg-default/95 px-3 py-2 text-xs text-toned"
                >
                  <label class="inline-flex items-center gap-2 font-medium">
                    <input
                      type="checkbox"
                      class="size-4 rounded border-default accent-[var(--ui-primary)]"
                      :checked="allVisibleCategoriesSelected"
                      @change="toggleVisibleCategories(($event.target as HTMLInputElement).checked)"
                    >
                    全选当前结果
                    <span
                      v-if="selectedCategoryIds.length > 0"
                      class="text-muted"
                    >
                      已选 {{ selectedCategoryIds.length }}
                    </span>
                  </label>
                  <UButton
                    size="xs"
                    color="error"
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    label="批量删除"
                    :disabled="selectedCategoryIds.length === 0"
                    :loading="deletingCategoryId === '__batch__'"
                    @click="removeSelectedCategories"
                  />
                </div>

                <div
                  v-if="categoryLoading"
                  class="space-y-2"
                >
                  <USkeleton
                    v-for="index in 3"
                    :key="index"
                    class="h-10 w-full"
                  />
                </div>
                <article
                  v-for="item in filteredCategories"
                  :key="item.id"
                  class="rounded-md border border-default bg-muted/20 p-2"
                >
                  <div class="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      class="size-4 shrink-0 rounded border-default accent-[var(--ui-primary)]"
                      :checked="selectedCategoryIds.includes(item.id)"
                      @change="toggleCategorySelected(item.id, ($event.target as HTMLInputElement).checked)"
                    >
                    <UInput
                      v-model="categoryRename[item.id]"
                      class="min-w-0 flex-1"
                      size="sm"
                    />
                    <UButton
                      size="xs"
                      color="neutral"
                      variant="soft"
                      icon="i-lucide-save"
                      :loading="updatingCategoryId === item.id"
                      @click="renameCategory(item.id)"
                    />
                    <UButton
                      size="xs"
                      color="error"
                      variant="ghost"
                      icon="i-lucide-trash-2"
                      :loading="deletingCategoryId === item.id"
                      @click="removeCategory(item.id)"
                    />
                  </div>
                </article>
                <UAlert
                  v-if="!categoryLoading && filteredCategories.length === 0"
                  :title="categories.length === 0 ? '暂无分类' : '没有匹配的分类'"
                  :description="categories.length === 0 ? '创建分类后可用于组织文章。' : '换个关键词再试。'"
                  icon="i-lucide-folder-open"
                  color="neutral"
                  variant="soft"
                />
              </div>
            </UCard>

            <UCard :ui="{ root: 'rounded-lg', body: 'p-4 sm:p-5', header: 'p-4 sm:p-5' }">
              <template #header>
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h2 class="text-base font-semibold text-highlighted">
                      标签管理
                    </h2>
                    <p class="mt-1 text-xs text-toned">
                      显示 {{ filteredTags.length }} / {{ tags.length }} 个标签
                    </p>
                  </div>
                  <UIcon
                    name="i-lucide-tags"
                    class="size-5 text-primary"
                  />
                </div>
              </template>

              <div class="space-y-3">
                <div class="grid gap-3 sm:grid-cols-2">
                  <UInput
                    v-model="newTag.name"
                    placeholder="标签名（必填）"
                  />
                  <UInput
                    v-model="newTag.slug"
                    placeholder="slug（可选）"
                  />
                </div>
                <div class="flex justify-end">
                  <UButton
                    :loading="savingTag"
                    label="新增标签"
                    icon="i-lucide-plus"
                    @click="createTagEntry"
                  />
                </div>
              </div>

              <div class="mt-4 max-h-[28rem] space-y-2 overflow-y-auto pr-1">
                <div
                  v-if="filteredTags.length > 0"
                  class="sticky top-0 z-10 flex items-center justify-between gap-3 border border-default bg-default/95 px-3 py-2 text-xs text-toned"
                >
                  <label class="inline-flex items-center gap-2 font-medium">
                    <input
                      type="checkbox"
                      class="size-4 rounded border-default accent-[var(--ui-primary)]"
                      :checked="allVisibleTagsSelected"
                      @change="toggleVisibleTags(($event.target as HTMLInputElement).checked)"
                    >
                    全选当前结果
                    <span
                      v-if="selectedTagIds.length > 0"
                      class="text-muted"
                    >
                      已选 {{ selectedTagIds.length }}
                    </span>
                  </label>
                  <UButton
                    size="xs"
                    color="error"
                    variant="ghost"
                    icon="i-lucide-trash-2"
                    label="批量删除"
                    :disabled="selectedTagIds.length === 0"
                    :loading="deletingTagId === '__batch__'"
                    @click="removeSelectedTags"
                  />
                </div>

                <div
                  v-if="tagLoading"
                  class="space-y-2"
                >
                  <USkeleton
                    v-for="index in 3"
                    :key="index"
                    class="h-10 w-full"
                  />
                </div>
                <article
                  v-for="item in filteredTags"
                  :key="item.id"
                  class="rounded-md border border-default bg-muted/20 p-2"
                >
                  <div class="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      class="size-4 shrink-0 rounded border-default accent-[var(--ui-primary)]"
                      :checked="selectedTagIds.includes(item.id)"
                      @change="toggleTagSelected(item.id, ($event.target as HTMLInputElement).checked)"
                    >
                    <UInput
                      v-model="tagRename[item.id]"
                      class="min-w-0 flex-1"
                      size="sm"
                    />
                    <UButton
                      size="xs"
                      color="neutral"
                      variant="soft"
                      icon="i-lucide-save"
                      :loading="updatingTagId === item.id"
                      @click="renameTag(item.id)"
                    />
                    <UButton
                      size="xs"
                      color="error"
                      variant="ghost"
                      icon="i-lucide-trash-2"
                      :loading="deletingTagId === item.id"
                      @click="removeTag(item.id)"
                    />
                  </div>
                </article>
                <UAlert
                  v-if="!tagLoading && filteredTags.length === 0"
                  :title="tags.length === 0 ? '暂无标签' : '没有匹配的标签'"
                  :description="tags.length === 0 ? '创建标签后可用于标记文章主题。' : '换个关键词再试。'"
                  icon="i-lucide-tag"
                  color="neutral"
                  variant="soft"
                />
              </div>
            </UCard>
          </div>
        </section>
      </template>
    </main>
  </div>
</template>
