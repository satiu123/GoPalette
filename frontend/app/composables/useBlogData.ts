// 数据类型

export interface ApiUser {
  id: number
  username: string
  role: string
  avatar_url?: string
  created_at?: string
}

export interface ApiCategory {
  id: number
  name: string
}

export interface ApiTag {
  id: number
  name: string
}

export interface Article {
  id: number
  title: string
  summary: string
  content: string
  author_id: number
  category_id: number
  status: string
  read_count: number
  author: ApiUser
  category: ApiCategory
  tags: ApiTag[]
  created_at: string
  updated_at: string
}

export interface Comment {
  id: number
  article_id: number
  user_id: number | null
  content: string
  parent_id: number
  created_at: string
  user?: ApiUser | null
  article?: Article | null
}

export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface ArticleListData {
  total: number
  articles: Article[]
}

export interface CommentListData {
  total: number
  comments: Comment[]
}

// 工具函数

/** 生成固定占位图（用文章 id 做 seed，保证同一文章图片不变） */
export function articleImageUrl(article: Article): string {
  return `https://picsum.photos/seed/article${article.id}/800/600`
}

/** 生成用户头像 URL：优先使用后端头像，缺失时回退占位图 */
export function userAvatarUrl(user: ApiUser | string | null | undefined): string {
  if (typeof user === 'object' && user?.avatar_url) {
    return user.avatar_url
  }
  const username = typeof user === 'string' ? user : (user?.username ?? 'user')
  return `https://picsum.photos/seed/${encodeURIComponent(username)}/100/100`
}

/** 格式化日期 */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric'
  })
}

/** 根据字数估算阅读时间 */
export function readingTime(content: string): string {
  // 去除 HTML 标签后按中英文均速估算
  const text = content.replace(/<[^>]*>/g, '')
  const minutes = Math.max(1, Math.ceil(text.length / 500))
  return `${minutes} min read`
}

// API Composables 

/** 文章分页列表（支持按分类/标签过滤） */
export function useArticleList(params?: Ref<{
  page?: number
  page_size?: number
  category_id?: number
  tag_id?: number
}>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ArticleListData>>('/articles', {
    baseURL: config.public.apiBase,
    query: params
  })
}

/** 单篇文章详情（自动累加阅读数） */
export function useArticle(id: string | number) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<Article>>(`/articles/${id}`, {
    baseURL: config.public.apiBase
  })
}

/** 所有分类列表 */
export function useCategories() {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ApiCategory[]>>('/categories', {
    baseURL: config.public.apiBase
  })
}

/** 所有标签列表 */
export function useTags() {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ApiTag[]>>('/tags', {
    baseURL: config.public.apiBase
  })
}

/** 文章评论列表 */
export function useComments(articleId: string | number) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<Comment[]>>(`/articles/${articleId}/comments`, {
    baseURL: config.public.apiBase
  })
}

/** 全文搜索 */
export function useSearch(params: Ref<{ q: string; page?: number; page_size?: number }>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ArticleListData>>('/search', {
    baseURL: config.public.apiBase,
    query: params,
    immediate: false,
    watch: false
  })
}

/** 当前用户自己的文章列表（含草稿），需要 Authorization 头 */
export function useMyArticles(token: Ref<string | null>, params?: Ref<{ page?: number; page_size?: number }>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ArticleListData>>('/users/me/articles', {
    baseURL: config.public.apiBase,
    query: params,
    headers: computed(() => token.value ? { Authorization: token.value } : undefined)
  })
}

/** 当前用户发表的评论，需要 Authorization 头 */
export function useMyComments(token: Ref<string | null>, params?: Ref<{ page?: number; page_size?: number }>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<CommentListData>>('/users/me/comments', {
    baseURL: config.public.apiBase,
    query: params,
    headers: computed(() => token.value ? { Authorization: token.value } : undefined)
  })
}

/** 当前用户文章收到的评论，需要 Authorization 头 */
export function useReceivedComments(token: Ref<string | null>, params?: Ref<{ page?: number; page_size?: number }>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<CommentListData>>('/users/me/comments/received', {
    baseURL: config.public.apiBase,
    query: params,
    headers: computed(() => token.value ? { Authorization: token.value } : undefined)
  })
}

/** 管理员文章列表（所有状态），需要 admin 权限 */
export function useAdminArticles(token: Ref<string | null>, params?: Ref<{ page?: number; page_size?: number; author_id?: number }>) {
  const config = useRuntimeConfig()
  return useFetch<ApiResponse<ArticleListData>>('/admin/articles', {
    baseURL: config.public.apiBase,
    query: params,
    headers: computed(() => token.value ? { Authorization: token.value } : undefined)
  })
}

