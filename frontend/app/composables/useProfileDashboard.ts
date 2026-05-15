import { POST_STATUS_ARCHIVED, POST_STATUS_DRAFT, POST_STATUS_OFFLINE, POST_STATUS_PRIVATE, POST_STATUS_PUBLISHED } from '~/composables/useBlogApi'

export interface UserDashboardInfo {
  id: string
  username: string
  email: string
  avatarURL: string
  bio: string
  location: string
  createdAt: string
}

export interface UserDashboardStats {
  posts: number
  published: number
  drafts: number
  archived: number
  views: number
  likes: number
  comments: number
}

export interface UserDashboardPost {
  id: string
  title: string
  slug: string
  summary: string
  status: number
  views: number
  likes: number
  comments: number
  createdAt: string
  updatedAt: string
  category: string
  tags: string[]
}

export interface UserDashboardComment {
  id: string
  postId: string
  content: string
  createdAt: string
  authorName: string
}

export interface UserDashboardResponse {
  userInfo: UserDashboardInfo | null
  postStats: UserDashboardStats
  topPosts: UserDashboardPost[]
  authorPosts: UserDashboardPost[]
  recentComments: UserDashboardComment[]
}

export type DataRecord = Record<string, unknown>

export function toRecord(value: unknown): DataRecord | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
  return value as DataRecord
}

export function toRecordArray(value: unknown) {
  if (!Array.isArray(value)) return [] as DataRecord[]
  return value.map(item => toRecord(item)).filter((item): item is DataRecord => Boolean(item))
}

export function pick(raw: DataRecord | undefined, keys: string[]) {
  if (!raw) return undefined
  for (const key of keys) {
    const value = raw[key]
    if (value !== undefined && value !== null && value !== '') {
      return value
    }
  }
  return undefined
}

export function toText(value: unknown, fallback = '') {
  if (value === undefined || value === null) return fallback
  const text = String(value).trim()
  return text || fallback
}

export function toNumber(value: unknown, fallback = 0) {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : fallback
}

export function toPostStatus(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) return value

  const text = String(value || '').trim().toUpperCase()
  if (!text) return POST_STATUS_DRAFT
  if (text === 'PUBLISHED') return POST_STATUS_PUBLISHED
  if (text === 'ARCHIVED') return POST_STATUS_ARCHIVED
  if (text === 'PRIVATE') return POST_STATUS_PRIVATE
  if (text === 'OFFLINE') return POST_STATUS_OFFLINE
  if (text === 'DRAFT') return POST_STATUS_DRAFT

  const parsed = Number(text)
  return Number.isFinite(parsed) ? parsed : POST_STATUS_DRAFT
}

export function formatDashboardDate(input: unknown, withTime = false) {
  if (!input) return '未知时间'

  let date: Date | null = null

  if (typeof input === 'string') {
    const parsed = new Date(input)
    if (!Number.isNaN(parsed.getTime())) {
      date = parsed
    }
  } else if (typeof input === 'object' && input) {
    const seconds = toNumber((input as Record<string, unknown>).seconds, NaN)
    if (Number.isFinite(seconds)) {
      date = new Date(seconds * 1000)
    }
  }

  if (!date || Number.isNaN(date.getTime())) return '未知时间'

  return new Intl.DateTimeFormat('zh-CN', withTime
    ? {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      }
    : {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      }).format(date)
}

export function formatDashboardCount(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value)
}

export function getErrorMessage(error: unknown, fallback: string) {
  if (!error || typeof error !== 'object') return fallback
  const typed = error as { message?: unknown, data?: { message?: unknown } }
  return toText(typed.data?.message ?? typed.message, fallback)
}

export function toPostStatusText(value: number) {
  if (value === POST_STATUS_PUBLISHED) return '已发布'
  if (value === POST_STATUS_ARCHIVED) return '已归档'
  if (value === POST_STATUS_PRIVATE) return '已发布'
  if (value === POST_STATUS_OFFLINE) return '已下线'
  return '草稿'
}

export function toPostVisibilityText(value: number) {
  if (value === POST_STATUS_PRIVATE) return '私密'
  if (value === POST_STATUS_OFFLINE) return '不公开'
  return '公开'
}

export function toPostStatusColor(value: number) {
  if (value === POST_STATUS_PUBLISHED || value === POST_STATUS_PRIVATE) return 'success'
  if (value === POST_STATUS_ARCHIVED) return 'warning'
  if (value === POST_STATUS_OFFLINE) return 'error'
  return 'neutral'
}

export function toPostVisibilityColor(value: number) {
  if (value === POST_STATUS_PRIVATE) return 'primary'
  if (value === POST_STATUS_OFFLINE) return 'warning'
  return 'neutral'
}

export function toAvatarFallback(name: string) {
  const text = name.trim()
  if (!text) return '我'
  return text.slice(0, 1).toUpperCase()
}

function normalizePost(item: DataRecord): UserDashboardPost {
  const category = toRecord(pick(item, ['category'])) || {}

  return {
    id: toText(pick(item, ['id'])),
    title: toText(pick(item, ['title']), '未命名文章'),
    slug: toText(pick(item, ['slug'])),
    summary: toText(pick(item, ['summary']), '暂无摘要'),
    status: toPostStatus(pick(item, ['status'])),
    views: toNumber(pick(item, ['viewCount', 'view_count', 'views'])),
    likes: toNumber(pick(item, ['likeCount', 'like_count', 'likes'])),
    comments: toNumber(pick(item, ['commentCount', 'comment_count', 'comments'])),
    createdAt: formatDashboardDate(pick(item, ['createdAt', 'created_at'])),
    updatedAt: formatDashboardDate(pick(item, ['updatedAt', 'updated_at']), true),
    category: toText(pick(category, ['name']), '未分类'),
    tags: Array.isArray(item.tags) ? item.tags.map(tag => String(tag)) : []
  }
}

export function normalizeDashboard(raw: DataRecord): UserDashboardResponse {
  const userInfoRaw = toRecord(pick(raw, ['userInfo', 'user_info'])) || {}
  const statsRaw = toRecord(pick(raw, ['postStats', 'post_stats'])) || {}
  const topPostsRaw = toRecordArray(pick(raw, ['topPosts', 'top_posts']))
  const commentsRaw = toRecordArray(pick(raw, ['recentComments', 'recent_comments']))

  const normalizedUser: UserDashboardInfo | null = Object.keys(userInfoRaw).length
    ? {
        id: toText(pick(userInfoRaw, ['id'])),
        username: toText(pick(userInfoRaw, ['username']), '未命名用户'),
        email: toText(pick(userInfoRaw, ['email'])),
        avatarURL: toText(pick(userInfoRaw, ['avatarURL', 'avatarUrl', 'avatar_u_r_l', 'avatar_url'])),
        bio: toText(pick(userInfoRaw, ['bio'])),
        location: toText(pick(userInfoRaw, ['location'])),
        createdAt: formatDashboardDate(pick(userInfoRaw, ['createdAt', 'created_at']))
      }
    : null

  return {
    userInfo: normalizedUser,
    postStats: {
      posts: toNumber(pick(statsRaw, ['posts'])),
      published: toNumber(pick(statsRaw, ['published'])),
      drafts: toNumber(pick(statsRaw, ['drafts'])),
      archived: toNumber(pick(statsRaw, ['archived'])),
      views: toNumber(pick(statsRaw, ['views'])),
      likes: toNumber(pick(statsRaw, ['likes'])),
      comments: toNumber(pick(statsRaw, ['comments']))
    },
    topPosts: topPostsRaw.map(normalizePost),
    authorPosts: toRecordArray(pick(raw, ['authorPosts', 'author_posts'])).map(normalizePost),
    recentComments: commentsRaw.map((item) => {
      const author = toRecord(item.author) || {}
      return {
        id: toText(pick(item, ['id'])),
        postId: toText(pick(item, ['postId', 'post_id'])),
        content: toText(pick(item, ['content']), '暂无评论内容'),
        createdAt: formatDashboardDate(pick(item, ['createdAt', 'created_at']), true),
        authorName: toText(pick(author, ['name', 'username']), '匿名用户')
      }
    })
  }
}

export async function fetchUserDashboard(userId: string) {
  const { authFetch } = useAuth()
  const response = await authFetch<DataRecord>(`/api/user/full-profile/${userId}`, {
    method: 'GET'
  })

  return normalizeDashboard(response || {})
}
